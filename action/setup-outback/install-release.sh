#!/usr/bin/env bash

set -euo pipefail

: "${OUTBACK_VERSION:?OUTBACK_VERSION is required}"
: "${OUTBACK_REPOSITORY:?OUTBACK_REPOSITORY is required}"
: "${OUTBACK_INSTALL_ROOT:?OUTBACK_INSTALL_ROOT is required}"

if [[ ! "${OUTBACK_VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.][0-9A-Za-z.-]+)?$ ]]; then
  printf 'invalid outback version: %s\n' "${OUTBACK_VERSION}" >&2
  exit 2
fi
if [[ ! "${OUTBACK_REPOSITORY}" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]]; then
  printf 'invalid outback release repository: %s\n' "${OUTBACK_REPOSITORY}" >&2
  exit 2
fi

runner_os="${RUNNER_OS:-$(uname -s)}"
case "${runner_os}" in
  Linux) os=linux ;;
  macOS|Darwin) os=darwin ;;
  *) printf 'unsupported outback release OS: %s\n' "${runner_os}" >&2; exit 2 ;;
esac

runner_arch="${RUNNER_ARCH:-$(uname -m)}"
case "${runner_arch}" in
  X64|x86_64|amd64) arch=amd64 ;;
  ARM64|aarch64|arm64) arch=arm64 ;;
  *) printf 'unsupported outback release architecture: %s\n' "${runner_arch}" >&2; exit 2 ;;
esac

bin_dir="${OUTBACK_INSTALL_ROOT}/bin"
binary="${bin_dir}/outback"
output_file="${GITHUB_OUTPUT:-}"
install -d -m 0755 "${bin_dir}"

record_source() {
  printf 'source=%s\n' "$1"
  if [[ -n "${output_file}" ]]; then
    printf 'source=%s\n' "$1" >> "${output_file}"
  fi
}

if [[ -x "${binary}" ]] && [[ "$("${binary}" version 2>/dev/null)" == "${OUTBACK_VERSION}" ]]; then
  record_source cache
  exit 0
fi
rm -f "${binary}"

asset="outback_${OUTBACK_VERSION}_${os}_${arch}.tar.gz"
release_base="${OUTBACK_RELEASE_BASE_URL:-https://github.com/${OUTBACK_REPOSITORY}/releases/download}"
release_url="${release_base}/v${OUTBACK_VERSION}"
temporary="$(mktemp -d)"
trap 'rm -rf "${temporary}"' EXIT

downloaded=true
curl --fail --location --silent --show-error --retry 3 \
  --output "${temporary}/${asset}" "${release_url}/${asset}" || downloaded=false
if [[ "${downloaded}" == true ]]; then
  curl --fail --location --silent --show-error --retry 3 \
    --output "${temporary}/checksums.txt" "${release_url}/checksums.txt" || downloaded=false
fi

if [[ "${downloaded}" == true ]]; then
  expected="$(awk -v asset="${asset}" '$2 == asset || $2 == "*" asset {print $1; exit}' "${temporary}/checksums.txt")"
  if [[ ! "${expected}" =~ ^[0-9a-fA-F]{64}$ ]]; then
    printf 'outback release checksum is missing for %s\n' "${asset}" >&2
    exit 1
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "${temporary}/${asset}" | awk '{print $1}')"
  else
    actual="$(shasum -a 256 "${temporary}/${asset}" | awk '{print $1}')"
  fi
  if [[ "${actual}" != "${expected}" ]]; then
    printf 'outback release checksum verification failed for %s\n' "${asset}" >&2
    exit 1
  fi
  tar -xzf "${temporary}/${asset}" -C "${temporary}" outback
  if [[ "$("${temporary}/outback" version)" != "${OUTBACK_VERSION}" ]]; then
    printf 'outback release binary does not report version %s\n' "${OUTBACK_VERSION}" >&2
    exit 1
  fi
  install -m 0755 "${temporary}/outback" "${binary}"
  record_source release
  exit 0
fi

printf 'outback release %s is unavailable\n' "${OUTBACK_VERSION}" >&2
exit 1
