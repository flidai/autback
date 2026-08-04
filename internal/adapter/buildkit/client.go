package buildkit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/flidai/autback/internal/capacity"
	buildkitclient "github.com/moby/buildkit/client"
)

type Config struct {
	Address          string
	OperationTimeout time.Duration
}

type engine interface {
	Info(context.Context) (*buildkitclient.Info, error)
	Prune(context.Context, chan buildkitclient.UsageInfo, ...buildkitclient.PruneOption) error
	Close() error
}

type Client struct {
	engine           engine
	operationTimeout time.Duration
}

func New(ctx context.Context, config Config) (*Client, error) {
	if config.Address == "" {
		return nil, errors.New("BuildKit address is required")
	}
	native, err := buildkitclient.New(ctx, normalizeAddress(config.Address))
	if err != nil {
		return nil, fmt.Errorf("create BuildKit client: %w", err)
	}
	timeout := config.OperationTimeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	return &Client{engine: native, operationTimeout: timeout}, nil
}

func newClient(engine engine) *Client {
	return &Client{engine: engine, operationTimeout: 5 * time.Minute}
}

func (c *Client) Check(ctx context.Context) error {
	info, err := c.engine.Info(ctx)
	if err != nil {
		return fmt.Errorf("inspect BuildKit: %w", err)
	}
	if info == nil || info.BuildkitVersion.Version == "" {
		return errors.New("inspect BuildKit: daemon returned no version")
	}
	return nil
}

func (c *Client) Prune(ctx context.Context, maxUsedBytes int64) error {
	if maxUsedBytes <= 0 {
		return errors.New("BuildKit maximum cache usage must be positive")
	}
	pruneCtx, cancel := context.WithTimeout(ctx, c.operationTimeout)
	defer cancel()
	err := c.engine.Prune(pruneCtx, nil,
		buildkitclient.PruneAll,
		buildkitclient.WithKeepOpt(0, 0, maxUsedBytes, 0),
	)
	if err != nil {
		return fmt.Errorf("prune BuildKit cache: %w", err)
	}
	return nil
}

func (c *Client) Close() error {
	return c.engine.Close()
}

func normalizeAddress(address string) string {
	if strings.Contains(address, "://") {
		return address
	}
	return "tcp://" + address
}

var _ capacity.BuildCache = (*Client)(nil)
