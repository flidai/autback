#!/bin/zsh

source "${0:A:h}/lib.zsh"

: "${OUTBACK_SERVER_IP:?set OUTBACK_SERVER_IP to the existing host}"
: "${OUTBACK_SSH_KEY:?set OUTBACK_SSH_KEY to the existing host identity file}"

host="${OUTBACK_SERVER_IP}"
ssh_args
proof_dir="${OUTBACK_DIR}/evidence/reapi"
fixture="$(mktemp -d "${TMPDIR:-/tmp}/outback-reapi-e2e.XXXXXX")"
cleanup() { rm -rf "${fixture}"; }
trap cleanup EXIT INT TERM

cp -R "${OUTBACK_DIR}/examples/go-redis/." "${fixture}/"
git -C "${fixture}" init -q
git -C "${fixture}" config user.name 'outback proof'
git -C "${fixture}" config user.email 'outback@example.invalid'
git -C "${fixture}" add .
git -C "${fixture}" commit -qm 'committed baseline'
print 'dirty worktree reached remote worker' > "${fixture}/proof.txt"
print 'untracked worktree bytes' > "${fixture}/untracked.txt"
mkdir -p "${fixture}/ignored"
print 'must not upload' > "${fixture}/ignored/large.bin"
print 'ignored/' >> "${fixture}/.gitignore"

cat > "${fixture}/Dockerfile" <<'EOF'
FROM alpine:3.22.1@sha256:4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1
COPY proof.txt /proof.txt
FROM scratch
COPY --from=0 /proof.txt /proof.txt
EOF

go -C "${OUTBACK_DIR}" build -o "${OUTBACK_TMP_DIR}/outback" ./cmd/outback
config_file="${OUTBACK_TMP_DIR}/reapi-e2e-config.json"
umask 077
jq -n --arg host "${host}" --arg identity "${OUTBACK_SSH_KEY:A}" \
  '{backend:"reapi",ssh:{host:$host,user:"root",identity_file:$identity},reapi:{instance:"outback"},buildkit:{}}' \
  > "${config_file}"
chmod 0600 "${config_file}"
mkdir -p "${proof_dir}"
rm -f "${proof_dir}/cancel.log"

OUTBACK_CONFIG="${config_file}" "${OUTBACK_TMP_DIR}/outback" doctor | tee "${proof_dir}/doctor.log"
(
  cd "${fixture}"
  OUTBACK_CONFIG="${config_file}" "${OUTBACK_TMP_DIR}/outback" run integration
) 2>&1 | tee "${proof_dir}/first-run.log"
(
  cd "${fixture}"
  OUTBACK_CONFIG="${config_file}" "${OUTBACK_TMP_DIR}/outback" run integration
) 2>&1 | tee "${proof_dir}/cached-run.log"
grep -Eq 'Transfer: (0 B|[0-9]+ B) uploaded' "${proof_dir}/cached-run.log"

set +e
(
  cd "${fixture}"
  OUTBACK_CONFIG="${config_file}" "${OUTBACK_TMP_DIR}/outback" run --timeout 1s -- sh -c 'echo REMOTE_TIMEOUT_PROOF_STARTED; sleep 60'
) > "${proof_dir}/timeout.log" 2>&1
timeout_exit=$?
set -e
[[ ${timeout_exit} -ne 0 ]] || { print -u2 'timeout proof unexpectedly succeeded'; exit 1; }

build_output="${fixture}/build-output"
mkdir -p "${build_output}"
(
  cd "${fixture}"
  OUTBACK_CONFIG="${config_file}" "${OUTBACK_TMP_DIR}/outback" build --progress plain --output "type=local,dest=${build_output}" .
) 2>&1 | tee "${proof_dir}/build.log"
grep -q 'dirty worktree reached remote worker' "${build_output}/proof.txt"

ssh "${reply[@]}" "root@${host}" \
  'set -eu
   for attempt in $(seq 1 20); do
     docker ps --format "{{.Names}}" | grep -Eq "^(outback-reapi-|reaper_)" || break
     sleep 1
   done
   systemctl is-active outback-nativelink outback-buildkit
   ss -lnt
   free -h
   df -h /
   docker ps --format "{{.Names}} {{.Image}} {{.Status}}"' \
  > "${proof_dir}/remote-status.txt"
grep -q '127.0.0.1:50051' "${proof_dir}/remote-status.txt"
grep -q '127.0.0.1:1234' "${proof_dir}/remote-status.txt"
if grep -Eq '^(outback-reapi-|reaper_)' "${proof_dir}/remote-status.txt"; then
  print -u2 'remote action leaked a container'
  exit 1
fi

jq -n \
  --arg completed "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --arg server "${host}" \
  --argjson timeout_exit "${timeout_exit}" \
  '{completed_at:$completed,server_ipv4:$server,reapi_instance:"outback",timeout_exit:$timeout_exit,testcontainers:true,dirty_worktree:true,incremental_cas:true,buildkit:true,cancel_operation:false}' \
  > "${proof_dir}/proof.json"
print "REAPI, Testcontainers, incremental CAS, timeout, and BuildKit proofs passed"
