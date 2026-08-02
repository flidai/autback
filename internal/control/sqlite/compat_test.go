package sqlite

import (
	"testing"

	"github.com/flidai/outback/internal/control"
)

func TestParseTokenAcceptsLegacyRtestCredentials(t *testing.T) {
	tests := []struct {
		secret string
		kind   control.PrincipalKind
		id     string
	}{
		{secret: "rtest_dt_tokdevice_secret", kind: control.PrincipalDevice, id: "tokdevice"},
		{secret: "rtest_gh_tokgithub_secret", kind: control.PrincipalGitHub, id: "tokgithub"},
	}
	for _, test := range tests {
		kind, id, ok := parseToken(test.secret)
		if !ok || kind != test.kind || id != test.id {
			t.Fatalf("parseToken(%q) = (%q, %q, %t)", test.secret, kind, id, ok)
		}
	}
}
