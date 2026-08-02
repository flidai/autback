package buildkit_test

import (
	"os"
	"strings"
	"testing"
)

func TestBuildPushSmokeRecipeUsesStandardBuildxMetadataAndImmutableImage(t *testing.T) {
	contents, err := os.ReadFile("../../docs/build-push-smoke.md")
	if err != nil {
		t.Fatal(err)
	}

	document := string(contents)
	for _, required := range []string{
		"outback build --",
		"--push",
		"--metadata-file",
		`containerimage.digest`,
		"@${digest}",
		"outback exec",
	} {
		if !strings.Contains(document, required) {
			t.Errorf("recipe does not contain %q", required)
		}
	}

	trustedRecipeStart := strings.Index(document, "## Trusted CI recipe")
	if trustedRecipeStart < 0 {
		t.Fatal("recipe does not define the trusted CI workflow")
	}
	trustedRecipe := document[trustedRecipeStart:]
	if strings.Contains(trustedRecipe, "--load") {
		t.Error("trusted CI recipe must not transfer the complete image back to the CI runner")
	}
	if !strings.Contains(trustedRecipe, "Never pass registry credentials through Outback command arguments or job environment variables") {
		t.Error("recipe does not document the registry credential boundary")
	}
}
