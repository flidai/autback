package sitemotion_test

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
	return filepath.Join(append([]string{"..", "..", "site-motion"}, parts...)...)
}

func TestMotionSiteContract(t *testing.T) {
	t.Parallel()

	contents, err := os.ReadFile(rootPath("index.html"))
	if err != nil {
		t.Fatal(err)
	}
	page := string(contents)

	for _, required := range []string{
		`<html lang="en">`,
		`class="skip-link"`,
		`<main id="main-content">`,
		`data-motion-stage`,
		`id="scene-canvas"`,
		`aria-hidden="true"`,
		`data-scene="0"`,
		`data-scene="5"`,
		`Only the change crosses the wire.`,
		`One FIFO queue. The whole machine.`,
	} {
		if !strings.Contains(page, required) {
			t.Errorf("index.html is missing %q", required)
		}
	}

	forbidden := strings.Join([]string{"data", "curve"}, "")
	if strings.Contains(strings.ToLower(page), forbidden) {
		t.Error("the motion concept must not copy reference-site branding or copy")
	}
}

func TestReducedMotionHasStaticReadingMode(t *testing.T) {
	t.Parallel()

	contents, err := os.ReadFile(rootPath("styles.css"))
	if err != nil {
		t.Fatal(err)
	}
	styles := string(contents)

	for _, required := range []string{
		`@media (prefers-reduced-motion: reduce)`,
		`.motion-stage`,
		`height: auto`,
		`.chapter.is-active`,
	} {
		if !strings.Contains(styles, required) {
			t.Errorf("styles.css is missing reduced-motion behavior %q", required)
		}
	}
	reduced := strings.SplitN(styles, `@media (prefers-reduced-motion: reduce)`, 2)[1]
	if !strings.Contains(reduced, `visibility: visible`) {
		t.Error("reduced-motion reading mode must reveal every chapter")
	}
}

func TestMotionEngineContract(t *testing.T) {
	t.Parallel()

	contents, err := os.ReadFile(rootPath("script.js"))
	if err != nil {
		t.Fatal(err)
	}
	script := string(contents)

	for _, required := range []string{
		`requestAnimationFrame`,
		`devicePixelRatio`,
		`prefers-reduced-motion: reduce`,
		`createImageData`,
		`drawLaptop`,
		`drawTransfer`,
		`drawWorker`,
		`drawQueue`,
		`drawDigest`,
		`event.key === 'Escape'`,
	} {
		if !strings.Contains(script, required) {
			t.Errorf("script.js is missing motion or accessibility behavior %q", required)
		}
	}
}

func TestInternalReferencesResolve(t *testing.T) {
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
			t.Errorf("reference %q escapes the motion artifact", reference)
			continue
		}
		if _, err := os.Stat(rootPath(clean)); err != nil {
			t.Errorf("reference %q does not resolve: %v", reference, err)
		}
	}
}
