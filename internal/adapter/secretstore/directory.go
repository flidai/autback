package secretstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/flidai/autback/internal/secrets"
)

type Directory struct{ Root string }

func (d Directory) Resolve(_ context.Context, projectID, name string) ([]byte, error) {
	if !safe(projectID) || !safe(name) || d.Root == "" {
		return nil, secrets.ErrRevoked
	}
	root, err := os.OpenRoot(d.Root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, secrets.ErrRevoked
	}
	if err != nil {
		return nil, fmt.Errorf("open external secret store: %w", err)
	}
	defer root.Close()
	file, err := root.Open(filepath.Join(projectID, name))
	if errors.Is(err, os.ErrNotExist) {
		return nil, secrets.ErrRevoked
	}
	if err != nil {
		return nil, fmt.Errorf("read external secret reference: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return nil, secrets.ErrRevoked
	}
	value, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil {
		return nil, fmt.Errorf("read external secret reference: %w", err)
	}
	return value, nil
}

func safe(value string) bool {
	return value != "" && value != "." && value != ".." && filepath.Base(value) == value
}

var _ secrets.Resolver = Directory{}
