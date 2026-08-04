package version

import (
	"os"
	"strings"
	"testing"
)

func TestSetupActionDefaultsToCurrentRelease(t *testing.T) {
	data, err := os.ReadFile("../../action/setup-autback/action.yml")
	if err != nil {
		t.Fatal(err)
	}
	want := "default: \"" + Current + "\""
	if !strings.Contains(string(data), want) {
		t.Fatalf("setup action does not contain %q", want)
	}
}
