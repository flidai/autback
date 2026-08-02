# Build, push, and smoke-test an immutable image

Autback deliberately keeps image builds on the standard Docker Buildx interface. A
trusted project can push directly from remote BuildKit to its OCI registry, capture the
digest reported by Buildx, and ask Autback to run the project's smoke test against that
exact image. The CI runner receives only logs and a small metadata document.

`autback image build` is a convenience for projects that use an activated runner image.
The explicit recipe below is preferable when a workflow builds several deployable
images or wants to own its tags and smoke commands.

## Trusted CI recipe

The example assumes the workflow has already authenticated Buildx to the registry and
that `jq` is available:

```sh
set -eu

image_repository="ghcr.io/example/application"
image_tag="${image_repository}:ci-${GITHUB_SHA}"
metadata_file="${RUNNER_TEMP}/application-image.json"

autback build -- \
  --push \
  --metadata-file "${metadata_file}" \
  --tag "${image_tag}" \
  .

digest="$(jq -er '."containerimage.digest" | select(startswith("sha256:"))' "${metadata_file}")"
immutable_image="${image_repository}@${digest}"

autback exec -- ./scripts/smoke-image.sh "${immutable_image}"
```

The digest is the hand-off contract. Deployment, signing, attestation, and later tests
should consume `repository@sha256:...`, never resolve the mutable CI tag again.

Registry authentication has two independent boundaries:

- Buildx uses the calling machine's normal Docker credential session to push.
- The remote Docker host running the smoke test must be able to pull the image. Use a
  public package or install a narrowly scoped, pull-only credential helper on that host.

Never pass registry credentials through Autback command arguments or job environment variables.
Arguments and non-secret environment configuration may be retained as job metadata and audit
records. GitHub Actions should use OIDC for Autback itself and the platform's normal registry
login mechanism for Buildx.
