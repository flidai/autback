package site_test

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	attributePattern = regexp.MustCompile(`(?:href|src)="([^"]+)"`)
	idPattern        = regexp.MustCompile(`\sid="([^"]+)"`)
)

func sitePath(parts ...string) string {
	return filepath.Join(append([]string{"..", "..", "site"}, parts...)...)
}

func TestPageContract(t *testing.T) {
	t.Parallel()

	contents, err := os.ReadFile(sitePath("index.html"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(contents)

	for _, required := range []string{
		`<html lang="en">`,
		`<meta name="viewport"`,
		`class="skip-link"`,
		`<main id="main-content">`,
		`Heavy work.`,
		`Light laptop.`,
		`role="tablist"`,
		`aria-selected="true"`,
		`REAPI CAS`,
		`Buildx`,
		`Strict FIFO`,
	} {
		if !strings.Contains(page, required) {
			t.Errorf("index.html is missing %q", required)
		}
	}
	if strings.Contains(strings.ToLower(page), "railway") {
		t.Error("the site must not copy reference-site branding or copy")
	}
}

func TestInternalLinksAndAssetsResolve(t *testing.T) {
	t.Parallel()

	contents, err := os.ReadFile(sitePath("index.html"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(contents)

	ids := make(map[string]bool)
	for _, match := range idPattern.FindAllStringSubmatch(page, -1) {
		if ids[match[1]] {
			t.Errorf("duplicate id %q", match[1])
		}
		ids[match[1]] = true
	}

	for _, match := range attributePattern.FindAllStringSubmatch(page, -1) {
		reference := match[1]
		parsed, err := url.Parse(reference)
		if err != nil {
			t.Errorf("parse reference %q: %v", reference, err)
			continue
		}
		if parsed.Scheme != "" || parsed.Host != "" {
			continue
		}
		if strings.HasPrefix(reference, "/") {
			t.Errorf("root-relative reference %q breaks GitHub project Pages", reference)
			continue
		}
		if parsed.Path == "" {
			if parsed.Fragment != "" && !ids[parsed.Fragment] {
				t.Errorf("fragment %q has no matching id", reference)
			}
			continue
		}
		clean := filepath.Clean(filepath.FromSlash(parsed.Path))
		if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			t.Errorf("reference %q escapes the site artifact", reference)
			continue
		}
		if _, err := os.Stat(sitePath(clean)); err != nil {
			t.Errorf("reference %q does not resolve: %v", reference, err)
		}
	}
}

func TestBackgroundArtworkIsVendoredAndOptimized(t *testing.T) {
	t.Parallel()

	info, err := os.Stat(sitePath("assets", "background.webp"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() < 100_000 {
		t.Fatalf("background artwork is unexpectedly small: %d bytes", info.Size())
	}
	if info.Size() > 2_000_000 {
		t.Fatalf("background artwork is too large for the landing page: %d bytes", info.Size())
	}
}

func TestInteractionContract(t *testing.T) {
	t.Parallel()

	contents, err := os.ReadFile(sitePath("script.js"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(contents)

	for _, required := range []string{
		`event.key === 'Escape'`,
		`navigation?.removeAttribute('data-open')`,
		`menuButton?.focus()`,
	} {
		if !strings.Contains(script, required) {
			t.Errorf("script.js is missing accessible menu behavior %q", required)
		}
	}
}

func TestPagesWorkflowPublishesCanonicalSite(t *testing.T) {
	t.Parallel()

	contents, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "pages.yml"))
	if err != nil {
		t.Fatal(err)
	}
	workflow := string(contents)

	for _, required := range []string{
		"actions/configure-pages@",
		"actions/upload-pages-artifact@",
		"actions/deploy-pages@",
		"path: site",
		"pages: write",
		"id-token: write",
		"name: github-pages",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("pages workflow is missing %q", required)
		}
	}
}
