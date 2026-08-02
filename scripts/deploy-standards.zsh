#!/bin/zsh

source "${0:A:h}/lib.zsh"

: "${OUTBACK_SERVER_IP:?set OUTBACK_SERVER_IP to an existing host; this script never provisions infrastructure}"
: "${OUTBACK_SSH_KEY:?set OUTBACK_SSH_KEY to the existing host identity file}"
[[ -f "${OUTBACK_SSH_KEY}" ]] || { print -u2 "missing ${OUTBACK_SSH_KEY}"; exit 1; }

host="${OUTBACK_SERVER_IP}"
ssh_args
ssh "${reply[@]}" "root@${host}" 'docker version >/dev/null'

build_dir="${OUTBACK_TMP_DIR}/build"
mkdir -p "${build_dir}"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go -C "${OUTBACK_DIR}" build -trimpath -ldflags='-s -w' \
  -o "${build_dir}/outback-reapi-entrypoint" ./cmd/outback-reapi-entrypoint

scp "${reply[@]}" \
  "${build_dir}/outback-reapi-entrypoint" \
  "${OUTBACK_DIR}/backend/reapi/nativelink.json5" \
  "${OUTBACK_DIR}/runner/standard/Dockerfile" \
  "${OUTBACK_DIR}/host/install-standards.sh" \
  "${OUTBACK_DIR}/host/outback-nativelink.service" \
  "${OUTBACK_DIR}/host/outback-buildkit.service" \
  "root@${host}:/tmp/"

ssh "${reply[@]}" "root@${host}" \
  "OUTBACK_JOB_CPUS=${OUTBACK_JOB_CPUS:-1.5} OUTBACK_JOB_MEMORY=${OUTBACK_JOB_MEMORY:-2500m} bash /tmp/install-standards.sh"

OUTBACK_BACKEND=reapi "${OUTBACK_DIR}/scripts/install-standards-cli.zsh"
print "deployed REAPI and BuildKit to existing host ${host}"
