package branding_test

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestLegacyProductNameIsAbsent(t *testing.T) {
	t.Parallel()

	rootOutput, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}

	legacyName := "out" + "back"
	command := exec.Command(
		"git", "grep", "-n", "-i", legacyName,
		"--", ".", ":!internal/branding/branding_test.go",
	)
	command.Dir = strings.TrimSpace(string(rootOutput))
	output, err := command.CombinedOutput()
	if err == nil {
		t.Fatalf("legacy product name remains in tracked files:\n%s", output)
	}

	var exitError *exec.ExitError
	if !errors.As(err, &exitError) || exitError.ExitCode() != 1 {
		t.Fatalf("search tracked files: %v\n%s", err, output)
	}
}
