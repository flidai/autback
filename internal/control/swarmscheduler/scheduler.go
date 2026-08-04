package swarmscheduler

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/flidai/autback/internal/control"
	"github.com/flidai/autback/internal/protocol"
	jobsecrets "github.com/flidai/autback/internal/secrets"
)

type Config struct {
	Client             Client
	CASAddress         string
	CASInstance        string
	JobsRoot           string
	EntrypointHostPath string
	CacheRoot          string
	HostUID            string
	HostGID            string
}

type Client interface {
	Check(context.Context) error
	ValidateImage(context.Context, string) error
	Create(context.Context, Spec) (string, error)
	Status(context.Context, string) (protocol.Job, error)
	Logs(context.Context, string, bool, io.Writer) error
	Cancel(context.Context, string) error
	ListResults(context.Context) ([]JobResult, error)
	Remove(context.Context, string) error
}

type Spec struct {
	ID                 string
	Image              string
	CASAddress         string
	CASInstance        string
	RootDigest         string
	JobsRoot           string
	Command            []string
	WorkingDirectory   string
	Environment        map[string]string
	EntrypointHostPath string
	Timeout            time.Duration
	CacheRoot          string
	ProjectID          string
	Caches             []CacheMount
	Secrets            []SecretMount
	HasSecrets         bool
	HostUID            string
	HostGID            string
}

type CacheMount struct {
	Name   string
	Target string
}

type SecretMount struct {
	Source string
	Target string
}

type JobResult struct {
	ID  string
	Job protocol.Job
	Err error
}

type Scheduler struct {
	config Config
}

func New(config Config) *Scheduler { return &Scheduler{config: config} }

func (s *Scheduler) Check(ctx context.Context) error { return s.config.Client.Check(ctx) }

func (s *Scheduler) ValidateImage(ctx context.Context, image string) error {
	return s.config.Client.ValidateImage(ctx, image)
}

func (s *Scheduler) Create(ctx context.Context, job control.Job) error {
	if err := prepareCacheDirectories(s.config.CacheRoot, job.ProjectID, job.Caches); err != nil {
		return err
	}
	_, err := s.config.Client.Create(ctx, specForJob(s.config, job))
	return err
}

func specForJob(config Config, job control.Job) Spec {
	caches := make([]CacheMount, 0, len(job.Caches))
	for _, cache := range job.Caches {
		caches = append(caches, CacheMount{Name: cache.Name, Target: cache.Target})
	}
	secretMounts := make([]SecretMount, 0, len(job.Secrets))
	for index, secret := range job.Secrets {
		if secret.File != "" {
			secretMounts = append(secretMounts, SecretMount{
				Source: filepath.Join(config.JobsRoot, job.ID, "secrets", jobsecrets.ValueFile(index, secret.Name)),
				Target: secret.File,
			})
		}
	}
	return Spec{
		ID:    job.ID,
		Image: job.Image, CASAddress: config.CASAddress, CASInstance: config.CASInstance,
		RootDigest: job.RootDigest, JobsRoot: config.JobsRoot, Command: job.Command,
		WorkingDirectory: job.WorkingDirectory, Environment: job.Environment,
		EntrypointHostPath: config.EntrypointHostPath,
		Timeout:            job.Timeout,
		CacheRoot:          config.CacheRoot, ProjectID: job.ProjectID, Caches: caches,
		HostUID: config.HostUID, HostGID: config.HostGID, Secrets: secretMounts, HasSecrets: len(job.Secrets) > 0,
	}
}

func prepareCacheDirectories(root, projectID string, caches []control.CacheMount) error {
	if len(caches) == 0 {
		return nil
	}
	if root == "" || !safeComponent(projectID) {
		return errors.New("cache root and safe project ID are required")
	}
	for _, cache := range caches {
		if !safeComponent(cache.Name) {
			return errors.New("cache name is not a safe path component")
		}
		directory := filepath.Join(root, projectID, cache.Name)
		if err := os.MkdirAll(directory, 0o700); err != nil {
			return err
		}
		if err := os.Chmod(directory, 0o700); err != nil {
			return err
		}
		now := time.Now()
		if err := os.Chtimes(directory, now, now); err != nil {
			return err
		}
	}
	return nil
}

func safeComponent(value string) bool {
	return value != "" && value != "." && value != ".." && filepath.Base(value) == value
}

func (s *Scheduler) Status(ctx context.Context, id string) (protocol.Job, error) {
	return s.config.Client.Status(ctx, id)
}

func (s *Scheduler) Logs(ctx context.Context, id string, follow bool, output io.Writer) error {
	if !follow {
		file, err := os.Open(filepath.Join(s.config.JobsRoot, id, "job.log"))
		if err == nil {
			defer file.Close()
			_, err = io.Copy(output, file)
			return err
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return s.config.Client.Logs(ctx, id, follow, output)
}

func (s *Scheduler) Cancel(ctx context.Context, id string) error {
	return s.config.Client.Cancel(ctx, id)
}

func (s *Scheduler) ManagedJobs(ctx context.Context) ([]control.RuntimeJob, error) {
	results, err := s.config.Client.ListResults(ctx)
	jobs := make([]control.RuntimeJob, 0, len(results))
	for _, result := range results {
		jobs = append(jobs, control.RuntimeJob{ID: result.ID, Job: result.Job, Err: result.Err})
	}
	return jobs, err
}

func (s *Scheduler) Remove(ctx context.Context, id string) error {
	return s.config.Client.Remove(ctx, id)
}

var _ control.Scheduler = (*Scheduler)(nil)
