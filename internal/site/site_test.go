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
		`<meta name="description"`,
		`class="skip-link"`,
		`<main id="main-content">`,
		`Remote compute for the work your laptop shouldn’t carry.`,
		`REAPI CAS`,
		`Buildx`,
		`FIFO`,
	} {
		if !strings.Contains(page, required) {
			t.Errorf("index.html is missing %q", required)
		}
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

func TestPagesWorkflowPublishesSiteArtifact(t *testing.T) {
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
