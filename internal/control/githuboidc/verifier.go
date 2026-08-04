package githuboidc

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/flidai/autback/internal/control"
)

const Issuer = "https://token.actions.githubusercontent.com"

type Verifier struct {
	verifier  *oidc.IDTokenVerifier
	audiences map[string]struct{}
}

func New(ctx context.Context, issuer, audience string) (*Verifier, error) {
	return NewWithAudiences(ctx, issuer, []string{audience})
}

func NewWithAudiences(ctx context.Context, issuer string, audiences []string) (*Verifier, error) {
	if issuer == "" {
		issuer = Issuer
	}
	allowed := make(map[string]struct{}, len(audiences))
	for _, audience := range audiences {
		if audience = strings.TrimSpace(audience); audience != "" {
			allowed[audience] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		return nil, errors.New("GitHub OIDC audience is required")
	}
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}
	return &Verifier{verifier: provider.Verifier(&oidc.Config{SkipClientIDCheck: true}), audiences: allowed}, nil
}

func (v *Verifier) Verify(ctx context.Context, raw string) (control.GitHubClaims, error) {
	if strings.TrimSpace(raw) == "" {
		return control.GitHubClaims{}, errors.New("GitHub OIDC token is required")
	}
	token, err := v.verifier.Verify(ctx, raw)
	if err != nil {
		return control.GitHubClaims{}, err
	}
	audienceAllowed := false
	for _, audience := range token.Audience {
		if _, ok := v.audiences[audience]; ok {
			audienceAllowed = true
			break
		}
	}
	if !audienceAllowed {
		return control.GitHubClaims{}, errors.New("GitHub OIDC token has an unexpected audience")
	}
	var rawClaims struct {
		RepositoryOwnerID string `json:"repository_owner_id"`
		RepositoryID      string `json:"repository_id"`
		Repository        string `json:"repository"`
		WorkflowRef       string `json:"workflow_ref"`
		Ref               string `json:"ref"`
		Environment       string `json:"environment"`
		EventName         string `json:"event_name"`
		NotBefore         int64  `json:"nbf"`
	}
	if err := token.Claims(&rawClaims); err != nil {
		return control.GitHubClaims{}, err
	}
	if token.Subject == "" || rawClaims.RepositoryOwnerID == "" || rawClaims.RepositoryID == "" || rawClaims.WorkflowRef == "" || rawClaims.Ref == "" || rawClaims.EventName == "" {
		return control.GitHubClaims{}, errors.New("GitHub OIDC token is missing required immutable identity or policy claims")
	}
	if rawClaims.NotBefore == 0 || time.Now().Add(30*time.Second).Before(time.Unix(rawClaims.NotBefore, 0)) {
		return control.GitHubClaims{}, errors.New("GitHub OIDC token is not yet valid")
	}
	return control.GitHubClaims{
		Subject: token.Subject, RepositoryOwnerID: rawClaims.RepositoryOwnerID, RepositoryID: rawClaims.RepositoryID,
		Repository: rawClaims.Repository, WorkflowRef: rawClaims.WorkflowRef, Ref: rawClaims.Ref,
		Environment: rawClaims.Environment, EventName: rawClaims.EventName, ExpiresAt: token.Expiry,
	}, nil
}

var _ interface {
	Verify(context.Context, string) (control.GitHubClaims, error)
} = (*Verifier)(nil)
