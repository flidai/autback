package secretstore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	jobsecrets "github.com/flidai/autback/internal/secrets"
)

func TestDirectoryResolvesProjectScopedRegularFile(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "project-1"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "project-1", "registry-token"), []byte("sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	value, err := (Directory{Root: root}).Resolve(context.Background(), "project-1", "registry-token")
	if err != nil || string(value) != "sentinel" {
		t.Fatalf("Resolve() = %q, %v", value, err)
	}
}

func TestDirectoryMapsRevocationAndUnsafeReferencesToUnavailable(t *testing.T) {
	root := t.TempDir()
	for _, test := range []struct{ project, name string }{
		{"project", "missing"}, {"../project", "name"}, {"project", "../name"},
	} {
		_, err := (Directory{Root: root}).Resolve(context.Background(), test.project, test.name)
		if !errors.Is(err, jobsecrets.ErrRevoked) {
			t.Fatalf("Resolve(%q, %q) error = %v", test.project, test.name, err)
		}
	}
}

func TestDirectoryRejectsSymlinkBelowConfiguredRoot(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "token"), []byte("sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(root, "project")); err != nil {
		t.Fatal(err)
	}
	if _, err := (Directory{Root: root}).Resolve(context.Background(), "project", "token"); err == nil {
		t.Fatal("Resolve() followed a project symlink outside the configured root")
	}
}

func TestDirectoryRejectsValueReadableByGroupOrOthers(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "project"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "project", "token"), []byte("sentinel"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := (Directory{Root: root}).Resolve(context.Background(), "project", "token")
	if !errors.Is(err, jobsecrets.ErrRevoked) {
		t.Fatalf("Resolve() error = %v", err)
	}
}
