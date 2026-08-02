package githubaction_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSetupActionHasSecureProjectAwareContract(t *testing.T) {
	data := read(t, filepath.Join("..", "..", "action", "setup-outback", "action.yml"))
	for _, want := range []string{
		"using: composite",
		"version:",
		"repository:",
		"service-url:",
		"project:",
		"image:",
		"ca-certificate:",
		"oidc-audience:",
		"actions/cache/restore@",
		"actions/cache/save@",
		"install-release.sh",
		"chmod 0600",
		"OUTBACK_CONFIG",
		"GITHUB_PATH",
	} {
		if !strings.Contains(data, want) {
			t.Fatalf("action.yml missing %q", want)
		}
	}
	if strings.Contains(data, "go build -trimpath -o \"${bin_dir}/outback\"") {
		t.Fatal("action.yml must not unconditionally compile outback")
	}
	for _, removed := range []string{"allow-source-fallback", "OUTBACK_ALLOW_SOURCE_FALLBACK", "OUTBACK_ACTION_ROOT", "backend: \"service\""} {
		if strings.Contains(data, removed) {
			t.Fatalf("action.yml retains removed compatibility path %q", removed)
		}
	}
}

func TestReleaseInstallerDownloadsAndVerifiesPinnedArchive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("release installer targets POSIX GitHub runners")
	}
	releaseRoot := t.TempDir()
	installRoot := t.TempDir()
	version := "9.8.7"
	asset := "outback_" + version + "_" + installerOS(runtime.GOOS) + "_" + installerArch(runtime.GOARCH) + ".tar.gz"
	assetDir := filepath.Join(releaseRoot, "v"+version)
	if err := os.MkdirAll(filepath.Join(assetDir, "payload"), 0o755); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(assetDir, "payload", "outback")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nprintf '%s\\n' '"+version+"'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	run(t, assetDir, "tar", "-czf", asset, "-C", "payload", "outback")
	archive, err := os.ReadFile(filepath.Join(assetDir, asset))
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(archive)
	checksums := hex.EncodeToString(digest[:]) + "  " + asset + "\n"
	if err := os.WriteFile(filepath.Join(assetDir, "checksums.txt"), []byte(checksums), 0o644); err != nil {
		t.Fatal(err)
	}

	result := runInstaller(t, map[string]string{
		"OUTBACK_VERSION":          version,
		"OUTBACK_REPOSITORY":       "example/outback",
		"OUTBACK_RELEASE_BASE_URL": "file://" + releaseRoot,
		"OUTBACK_INSTALL_ROOT":     installRoot,
	})
	if !strings.Contains(result, "source=release") {
		t.Fatalf("installer output = %q, want release source", result)
	}
	got := run(t, t.TempDir(), filepath.Join(installRoot, "bin", "outback"), "version")
	if strings.TrimSpace(got) != version {
		t.Fatalf("installed version = %q, want %q", strings.TrimSpace(got), version)
	}
	cached := runInstaller(t, map[string]string{
		"OUTBACK_VERSION":          version,
		"OUTBACK_REPOSITORY":       "example/outback",
		"OUTBACK_RELEASE_BASE_URL": "file://" + t.TempDir(),
		"OUTBACK_INSTALL_ROOT":     installRoot,
	})
	if !strings.Contains(cached, "source=cache") {
		t.Fatalf("second installer output = %q, want cache source", cached)
	}
}

func TestReleaseInstallerRejectsChecksumMismatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("release installer targets POSIX GitHub runners")
	}
	releaseRoot := t.TempDir()
	version := "9.8.7"
	asset := "outback_" + version + "_" + installerOS(runtime.GOOS) + "_" + installerArch(runtime.GOARCH) + ".tar.gz"
	assetDir := filepath.Join(releaseRoot, "v"+version)
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, asset), []byte("corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "checksums.txt"), []byte(strings.Repeat("0", 64)+"  "+asset+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output, err := installerCommand(map[string]string{
		"OUTBACK_VERSION":          version,
		"OUTBACK_REPOSITORY":       "example/outback",
		"OUTBACK_RELEASE_BASE_URL": "file://" + releaseRoot,
		"OUTBACK_INSTALL_ROOT":     t.TempDir(),
	}).CombinedOutput()
	if err == nil {
		t.Fatalf("installer unexpectedly accepted mismatched checksum: %s", output)
	}
	if !strings.Contains(string(output), "checksum") {
		t.Fatalf("installer error %q does not identify checksum failure", output)
	}
}

func TestReleaseInstallerNeverCompilesSourceWhenReleaseIsUnavailable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("release installer targets POSIX GitHub runners")
	}
	output, err := installerCommand(map[string]string{
		"OUTBACK_VERSION":               "9.8.7",
		"OUTBACK_REPOSITORY":            "example/outback",
		"OUTBACK_RELEASE_BASE_URL":      "file://" + t.TempDir(),
		"OUTBACK_INSTALL_ROOT":          t.TempDir(),
		"OUTBACK_ACTION_ROOT":           t.TempDir(),
		"OUTBACK_ALLOW_SOURCE_FALLBACK": "true",
	}).CombinedOutput()
	if err == nil {
		t.Fatalf("installer unexpectedly compiled source: %s", output)
	}
	if strings.Contains(string(output), "source=source") || !strings.Contains(string(output), "release 9.8.7 is unavailable") {
		t.Fatalf("installer output = %q", output)
	}
}

func TestReleaseInstallerSelectsGitHubRunnerPlatform(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("release installer targets POSIX GitHub runners")
	}
	releaseRoot := t.TempDir()
	version := "9.8.7"
	asset := "outback_" + version + "_linux_arm64.tar.gz"
	assetDir := filepath.Join(releaseRoot, "v"+version)
	if err := os.MkdirAll(filepath.Join(assetDir, "payload"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "payload", "outback"), []byte("#!/bin/sh\necho "+version+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	run(t, assetDir, "tar", "-czf", asset, "-C", "payload", "outback")
	archive, err := os.ReadFile(filepath.Join(assetDir, asset))
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(archive)
	if err := os.WriteFile(filepath.Join(assetDir, "checksums.txt"), []byte(hex.EncodeToString(digest[:])+"  "+asset+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := runInstaller(t, map[string]string{
		"RUNNER_OS":                "Linux",
		"RUNNER_ARCH":              "ARM64",
		"OUTBACK_VERSION":          version,
		"OUTBACK_REPOSITORY":       "example/outback",
		"OUTBACK_RELEASE_BASE_URL": "file://" + releaseRoot,
		"OUTBACK_INSTALL_ROOT":     t.TempDir(),
	})
	if !strings.Contains(result, "source=release") {
		t.Fatalf("installer output = %q, want linux/arm64 release", result)
	}
}

func TestReleaseWorkflowPackagesPortableChecksummedAssets(t *testing.T) {
	data := read(t, filepath.Join("..", "..", ".github", "workflows", "release.yml"))
	for _, want := range []string{
		"v*",
		"linux amd64",
		"linux arm64",
		"darwin amd64",
		"darwin arm64",
		"checksums.txt",
		"gh release create",
	} {
		if !strings.Contains(data, want) {
			t.Fatalf("outback-release.yml missing %q", want)
		}
	}
}

func TestPOCWorkflowIsManualAndUsesOIDC(t *testing.T) {
	data := read(t, filepath.Join("..", "..", ".github", "workflows", "poc.yml"))
	for _, want := range []string{
		"workflow_dispatch:",
		"id-token: write",
		"environment: outback-poc",
		"uses: ./action/setup-outback",
		"service-url: ${{ vars.OUTBACK_SERVICE_URL }}",
		"project: poc",
		"outback doctor",
		"outback exec -- go test -count=1 -v ./...",
	} {
		if !strings.Contains(data, want) {
			t.Fatalf("outback-poc.yml missing %q", want)
		}
	}
	if strings.Contains(data, "pull_request:") {
		t.Fatal("POC workflow must not run automatically for pull requests")
	}
	for _, forbidden := range []string{"github.repository ==", "tailscale/github-action", "ssh-private-key", "OUTBACK_TOKEN"} {
		if strings.Contains(data, forbidden) {
			t.Fatalf("POC workflow still contains legacy authentication %q", forbidden)
		}
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
