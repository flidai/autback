package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flidai/autback/internal/control"
)

const (
	RuntimeDirectory = "/run/autback/secrets"
	manifestName     = "manifest.json"
	maximumValueSize = 1 << 20
)

var ErrRevoked = errors.New("secret reference is unavailable or revoked")

var errInvalidValue = errors.New("secret value is empty or exceeds 1 MiB")

type Resolver interface {
	Resolve(context.Context, string, string) ([]byte, error)
}

type JobStore interface {
	Job(context.Context, string) (control.Job, error)
}

type AccessRecorder interface {
	RecordSecretAccess(context.Context, string, string, string) error
}

type Config struct {
	JobsRoot string
	Store    JobStore
	Resolver Resolver
	Access   AccessRecorder
}

type Manager struct{ config Config }

func NewManager(config Config) *Manager { return &Manager{config: config} }

type ResolutionError struct {
	Name string
	Err  error
}

func (e *ResolutionError) Error() string {
	if errors.Is(e.Err, ErrRevoked) {
		return fmt.Sprintf("resolve job secret %q: reference is unavailable or revoked", e.Name)
	}
	if errors.Is(e.Err, errInvalidValue) {
		return fmt.Sprintf("resolve job secret %q: value is invalid", e.Name)
	}
	return fmt.Sprintf("resolve job secret %q: external provider failed", e.Name)
}
func (e *ResolutionError) Unwrap() error { return e.Err }
func (e *ResolutionError) Permanent() bool {
	return errors.Is(e.Err, ErrRevoked) || errors.Is(e.Err, errInvalidValue)
}

type manifest struct {
	Bindings []manifestBinding `json:"bindings"`
}

type manifestBinding struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Environment string `json:"environment,omitempty"`
	File        string `json:"file,omitempty"`
}

func (m *Manager) Prepare(ctx context.Context, operation control.Operation) error {
	if operation.Kind != control.OperationJob {
		return nil
	}
	if m == nil || m.config.Store == nil || m.config.Resolver == nil || m.config.JobsRoot == "" {
		return errors.New("job secret manager is not configured")
	}
	job, err := m.config.Store.Job(ctx, operation.ID)
	if err != nil || len(job.Secrets) == 0 {
		return err
	}
	jobDirectory := filepath.Join(m.config.JobsRoot, job.ID)
	if err := os.MkdirAll(jobDirectory, 0o700); err != nil {
		return fmt.Errorf("prepare job secret directory: %w", err)
	}
	temporary, err := os.MkdirTemp(jobDirectory, ".secrets-")
	if err != nil {
		return fmt.Errorf("prepare job secret directory: %w", err)
	}
	defer os.RemoveAll(temporary)
	if err := os.Chmod(temporary, 0o700); err != nil {
		return err
	}
	data := manifest{Bindings: make([]manifestBinding, 0, len(job.Secrets))}
	for index, binding := range job.Secrets {
		value, err := m.config.Resolver.Resolve(ctx, job.ProjectID, binding.Name)
		if err != nil {
			return &ResolutionError{Name: binding.Name, Err: err}
		}
		if len(value) == 0 || len(value) > maximumValueSize {
			clear(value)
			return &ResolutionError{Name: binding.Name, Err: errInvalidValue}
		}
		filename := ValueFile(index, binding.Name)
		writeErr := os.WriteFile(filepath.Join(temporary, filename), value, 0o600)
		clear(value)
		if writeErr != nil {
			return fmt.Errorf("materialize job secret %q: %w", binding.Name, writeErr)
		}
		data.Bindings = append(data.Bindings, manifestBinding{Name: binding.Name, Source: filename, Environment: binding.Environment, File: binding.File})
		if m.config.Access != nil {
			if err := m.config.Access.RecordSecretAccess(ctx, job.ProjectID, job.ID, binding.Name); err != nil {
				return fmt.Errorf("record access to job secret %q: %w", binding.Name, err)
			}
		}
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(temporary, manifestName), payload, 0o600); err != nil {
		return fmt.Errorf("write job secret manifest: %w", err)
	}
	destination := filepath.Join(jobDirectory, "secrets")
	if err := os.RemoveAll(destination); err != nil {
		return err
	}
	if err := os.Rename(temporary, destination); err != nil {
		return fmt.Errorf("publish job secret material: %w", err)
	}
	return nil
}

func (m *Manager) Cleanup(_ context.Context, operation control.Operation) error {
	if operation.Kind != control.OperationJob || m == nil || m.config.JobsRoot == "" {
		return nil
	}
	return os.RemoveAll(filepath.Join(m.config.JobsRoot, operation.ID, "secrets"))
}

func ValueFile(index int, name string) string {
	return fmt.Sprintf("%03d-%s", index, name)
}

type RuntimeValues struct {
	Values      []string
	Environment []string
}

func LoadRuntime(directory string) (RuntimeValues, error) {
	payload, err := os.ReadFile(filepath.Join(directory, manifestName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RuntimeValues{}, nil
		}
		return RuntimeValues{}, errors.New("read job secret manifest")
	}
	var data manifest
	if err := json.Unmarshal(payload, &data); err != nil {
		return RuntimeValues{}, errors.New("decode job secret manifest")
	}
	result := RuntimeValues{Values: make([]string, 0, len(data.Bindings))}
	for _, binding := range data.Bindings {
		if filepath.Base(binding.Source) != binding.Source || strings.Contains(binding.Source, "..") {
			return RuntimeValues{}, errors.New("job secret manifest contains an invalid source")
		}
		value, err := os.ReadFile(filepath.Join(directory, binding.Source))
		if err != nil {
			return RuntimeValues{}, fmt.Errorf("read job secret %q", binding.Name)
		}
		result.Values = append(result.Values, string(value))
		if binding.Environment != "" {
			result.Environment = append(result.Environment, binding.Environment+"="+string(value))
		}
	}
	return result, nil
}
