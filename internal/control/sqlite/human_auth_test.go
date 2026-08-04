package sqlite_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/flidai/autback/internal/control"
)

func TestExternalIdentityUsesImmutableProviderSubject(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{UserName: "Owner", ProjectSlug: "example", TokenName: "owner-device"})
	if err != nil {
		t.Fatal(err)
	}
	owner, err := store.Authenticate(ctx, bootstrap.Token)
	if err != nil {
		t.Fatal(err)
	}
	member, err := store.CreateUser(ctx, owner, "Coworker", false)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := store.BindExternalIdentity(ctx, owner, member.ID, control.ExternalIdentity{
		Provider: "github", Subject: "12345678", Login: "coworker",
	})
	if err != nil {
		t.Fatal(err)
	}
	if identity.UserID != member.ID || identity.Subject != "12345678" || identity.Login != "coworker" {
		t.Fatalf("identity = %#v", identity)
	}
	resolved, err := store.UserByExternalIdentity(ctx, "github", "12345678", "coworker-renamed")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ID != member.ID {
		t.Fatalf("resolved user = %#v", resolved)
	}
	updated, err := store.ExternalIdentity(ctx, "github", "12345678")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Login != "coworker-renamed" || updated.LastAuthenticatedAt == nil {
		t.Fatalf("updated identity = %#v", updated)
	}

	other, err := store.CreateUser(ctx, owner, "Other", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.BindExternalIdentity(ctx, owner, other.ID, control.ExternalIdentity{Provider: "github", Subject: "12345678", Login: "other"}); !errors.Is(err, control.ErrAlreadyExists) {
		t.Fatalf("duplicate subject error = %v", err)
	}
}

func TestExternalIdentityBindingRequiresAnAdminDevice(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{UserName: "Owner", ProjectSlug: "example", TokenName: "owner-device"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.BindExternalIdentity(ctx, control.Principal{Kind: control.PrincipalBrowser, UserID: bootstrap.User.ID, Admin: true}, bootstrap.User.ID, control.ExternalIdentity{Provider: "github", Subject: "1", Login: "owner"}); !errors.Is(err, control.ErrForbidden) {
		t.Fatalf("browser binding error = %v", err)
	}
}

func TestRevokeExternalIdentityEndsHumanAndDeviceAccess(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{UserName: "Owner", ProjectSlug: "example", TokenName: "owner-device"})
	if err != nil {
		t.Fatal(err)
	}
	owner, err := store.Authenticate(ctx, bootstrap.Token)
	if err != nil {
		t.Fatal(err)
	}
	member, err := store.CreateUser(ctx, owner, "Coworker", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.BindExternalIdentity(ctx, owner, member.ID, control.ExternalIdentity{Provider: "github", Subject: "12345678", Login: "coworker"}); err != nil {
		t.Fatal(err)
	}
	session, err := store.CreateBrowserSession(ctx, member.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	device, err := store.CreateDeviceToken(ctx, owner, control.CreateDeviceToken{Name: "coworker-laptop", UserID: member.ID})
	if err != nil {
		t.Fatal(err)
	}

	if err := store.RevokeExternalIdentity(ctx, owner, member.ID, "github"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ExternalIdentity(ctx, "github", "12345678"); !errors.Is(err, control.ErrNotFound) {
		t.Fatalf("ExternalIdentity error = %v, want not found", err)
	}
	if _, err := store.UserByExternalIdentity(ctx, "github", "12345678", "coworker"); !errors.Is(err, control.ErrForbidden) {
		t.Fatalf("UserByExternalIdentity error = %v, want forbidden", err)
	}
	if _, err := store.Authenticate(ctx, session.Token); !errors.Is(err, control.ErrUnauthenticated) {
		t.Fatalf("browser session authentication = %v, want unauthenticated", err)
	}
	if _, err := store.Authenticate(ctx, device.Secret); !errors.Is(err, control.ErrUnauthenticated) {
		t.Fatalf("device authentication = %v, want unauthenticated", err)
	}
}

func TestOAuthLoginStateIsShortLivedAndSingleUse(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	now := time.Now().UTC()
	issued, err := store.CreateOAuthLoginState(ctx, "/app/projects/example", "verifier", now.Add(10*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	consumed, err := store.ConsumeOAuthLoginState(ctx, issued.State, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if consumed.ReturnTo != "/app/projects/example" || consumed.CodeVerifier != "verifier" {
		t.Fatalf("consumed state = %#v", consumed)
	}
	if _, err := store.ConsumeOAuthLoginState(ctx, issued.State, now.Add(2*time.Minute)); !errors.Is(err, control.ErrUnauthenticated) {
		t.Fatalf("second consume error = %v", err)
	}

	expired, err := store.CreateOAuthLoginState(ctx, "/app", "expired-verifier", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ConsumeOAuthLoginState(ctx, expired.State, now.Add(2*time.Minute)); !errors.Is(err, control.ErrUnauthenticated) {
		t.Fatalf("expired consume error = %v", err)
	}
}

func TestBrowserSessionAuthenticatesAsTheBoundUser(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{UserName: "Owner", ProjectSlug: "example", TokenName: "owner-device"})
	if err != nil {
		t.Fatal(err)
	}
	session, err := store.CreateBrowserSession(ctx, bootstrap.User.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	principal, err := store.Authenticate(ctx, session.Token)
	if err != nil {
		t.Fatal(err)
	}
	if principal.Kind != control.PrincipalBrowser || principal.UserID != bootstrap.User.ID || !principal.Admin {
		t.Fatalf("principal = %#v", principal)
	}
}

func TestRevokeBrowserSessionInvalidatesItsToken(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{UserName: "Owner", ProjectSlug: "example", TokenName: "owner-device"})
	if err != nil {
		t.Fatal(err)
	}
	session, err := store.CreateBrowserSession(ctx, bootstrap.User.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeBrowserSession(ctx, session.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Authenticate(ctx, session.Token); !errors.Is(err, control.ErrUnauthenticated) {
		t.Fatalf("authenticate revoked session: %v", err)
	}
}

func TestApprovedDeviceLoginIssuesOneRevocableDeviceToken(t *testing.T) {
	store := openStore(t)
	ctx := context.Background()
	bootstrap, err := store.Bootstrap(ctx, control.Bootstrap{UserName: "Owner", ProjectSlug: "example", TokenName: "owner-device"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	login, err := store.CreateDeviceLogin(ctx, "work-laptop", now.Add(10*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if login.Login.UserCode == "" || login.DeviceCode == "" {
		t.Fatalf("issued login = %#v", login)
	}
	if _, err := store.ExchangeDeviceLogin(ctx, login.DeviceCode, now.Add(time.Minute)); !errors.Is(err, control.ErrLoginPending) {
		t.Fatalf("pending exchange error = %v", err)
	}
	principal := control.Principal{Kind: control.PrincipalBrowser, UserID: bootstrap.User.ID, Admin: true}
	if err := store.ApproveDeviceLogin(ctx, login.Login.UserCode, principal, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	token, err := store.ExchangeDeviceLogin(ctx, login.DeviceCode, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	authenticated, err := store.Authenticate(ctx, token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	if authenticated.Kind != control.PrincipalDevice || authenticated.UserID != bootstrap.User.ID {
		t.Fatalf("authenticated principal = %#v", authenticated)
	}
	if _, err := store.ExchangeDeviceLogin(ctx, login.DeviceCode, now.Add(3*time.Minute)); !errors.Is(err, control.ErrUnauthenticated) {
		t.Fatalf("second exchange error = %v", err)
	}
}
