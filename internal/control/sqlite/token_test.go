package sqlite

import (
	"testing"

	"github.com/flidai/autback/internal/control"
)

func TestParseTokenAcceptsOnlyAutbackCredentials(t *testing.T) {
	tests := []struct {
		token string
		kind  control.PrincipalKind
		id    string
	}{
		{token: "autback_dt_tokdevice_secret", kind: control.PrincipalDevice, id: "tokdevice"},
		{token: "autback_gh_tokgithub_secret", kind: control.PrincipalGitHub, id: "tokgithub"},
	}
	for _, test := range tests {
		kind, id, ok := parseToken(test.token)
		if !ok || kind != test.kind || id != test.id {
			t.Fatalf("parseToken(%q) = (%q, %q, %t)", test.token, kind, id, ok)
		}
	}
	for _, token := range []string{
		"rtest_dt_tokdevice_secret",
		"rtest_gh_tokgithub_secret",
		"autback_enr_token_secret",
		"autback_unknown_token_secret",
	} {
		if kind, id, ok := parseToken(token); ok {
			t.Fatalf("parseToken(%q) = (%q, %q, true), want rejected", token, kind, id)
		}
	}
}
