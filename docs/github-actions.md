# GitHub Actions

GitHub Actions authenticates directly to the autback HTTPS control plane with OIDC. There
is no SSH key, shared autback token, Tailscale identity, Docker credential, or persistent CI
secret.

## One-time service policy

Create a project trust with immutable GitHub owner and repository IDs and the narrowest
workflow policy that fits the repository:

```console
autback trust github create \
  --project poc \
  --owner-id <repository_owner_id> \
  --repository-id <repository_id> \
  --workflow-ref 'flidai/autback/.github/workflows/poc.yml@refs/heads/*' \
  --ref 'refs/heads/*' \
  --environment autback-poc \
  --event workflow_dispatch
```

The server validates GitHub's issuer, signature/JWKS, audience, expiry, not-before time,
immutable IDs, workflow, ref, environment, and event before issuing a project-scoped
session lasting at most 15 minutes.

## Repository configuration

Create a protected GitHub environment named `autback-poc`, then configure variables:

- `AUTBACK_SERVICE_URL`: the control-plane URL, such as `https://autback.example.com`;
- `AUTBACK_PROJECT_IMAGE`: an optional digest-pinned per-run override; normally omit it and
  use the digest-pinned image activated on the autback project;
- `AUTBACK_CA_CERTIFICATE`: the service CA PEM when HTTPS uses the autback private CA. Omit
  this variable after placing the control plane behind a publicly trusted certificate.

The workflow needs `id-token: write` and `contents: read`. The composite action writes a
mode-0600 configuration under `RUNNER_TEMP`; the CLI detects the standard Actions OIDC
environment, requests an ID token for the autback audience, and exchanges it on demand.
The action exposes its required `project` input as `AUTBACK_PROJECT`, so the OIDC exchange
and every subsequent operation are bound to that selected project. It does not create or
rely on a user-wide default. Before a long build records completion or activates its
result, the CLI requests a fresh OIDC identity and project session; a short-lived bootstrap
session therefore never becomes the lifetime limit for an otherwise healthy build.

## CLI distribution

`action/setup-autback` installs an exact `version` from the repository's `v*` GitHub
release. Release archives cover Linux and macOS on amd64 and arm64. The action downloads
the release checksum manifest, verifies SHA-256 before extraction, verifies the binary's
reported version, and caches only that verified release under an OS/architecture/version
key. A restored cache entry must report the requested version before it is trusted.

Installation fails closed when the requested release or checksum is unavailable; the
action never compiles repository source or stores an unverified binary under a release
cache key. Publishing a `v0.1.0` tag runs `release.yml`; the job refuses a tag that
disagrees with `autback version` and publishes checksummed archives.

The POC stays `workflow_dispatch`-only until its manual hosted proof passes. For a
`pull_request` trust, autback requires an `--environment`; configure that GitHub environment
with required reviewers or an equally strong deployment protection rule. GitHub's signed
OIDC claims include `head_ref` but do not include the head repository's immutable ID, so
the autback server cannot honestly distinguish a same-repository PR from a fork by JWT
claims alone. Environment approval is therefore the explicit transition from untrusted to
trusted. Do not approve a fork, Dependabot run, or workflow change until its code is safe
to execute on the shared Docker worker.

Repository names remain audit metadata rather than authorization keys. A rename continues
to work when the immutable `repository_id` and `repository_owner_id` still match; reusing
an old name with different IDs does not grant access.
