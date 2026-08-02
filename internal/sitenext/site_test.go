package sitenext_test

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

func rootPath(parts ...string) string {
	return filepath.Join(append([]string{"..", "..", "site-next"}, parts...)...)
}

func TestIndependentSiteContract(t *testing.T) {
	t.Parallel()

	contents, err := os.ReadFile(rootPath("index.html"))
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
		t.Error("the comparison site must not copy Railway branding or copy")
	}
}

func TestInternalLinksAndAssetsResolve(t *testing.T) {
	t.Parallel()

	contents, err := os.ReadFile(rootPath("index.html"))
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
			t.Errorf("reference %q escapes the comparison artifact", reference)
			continue
		}
		if _, err := os.Stat(rootPath(clean)); err != nil {
			t.Errorf("reference %q does not resolve: %v", reference, err)
		}
	}
}

func TestLandscapeIsVendored(t *testing.T) {
	t.Parallel()

	info, err := os.Stat(rootPath("assets", "landscape.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() < 100_000 {
		t.Fatalf("landscape image is unexpectedly small: %d bytes", info.Size())
	}
}

func TestInteractionContract(t *testing.T) {
	t.Parallel()

	contents, err := os.ReadFile(rootPath("script.js"))
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
