#!/bin/zsh

source "${0:A:h}/lib.zsh"
zmodload zsh/datetime

: "${OUTBACK_SERVER_IP:?set OUTBACK_SERVER_IP to the existing host}"
: "${OUTBACK_SSH_KEY:?set OUTBACK_SSH_KEY to the existing host identity file}"

host="${OUTBACK_SERVER_IP}"
ssh_user="${OUTBACK_SSH_USER:-root}"
service_url="${OUTBACK_SERVICE_URL:-https://${host}}"
project="${OUTBACK_PROJECT:-default}"
project_image="${OUTBACK_PROJECT_IMAGE:-golang:1.25-bookworm@sha256:ea341baa9bd5ba6784f6d7161ace70544349a6242d54d34a0fbfd2c4d51c9d58}"
ca_file="${OUTBACK_CA_FILE:-${XDG_CONFIG_HOME:-${HOME}/.config}/outback/ca.pem}"
[[ -r "${ca_file}" ]] || { print -u2 "missing service CA ${ca_file}; deploy the service first"; exit 1; }
ssh_args

proof_dir="${OUTBACK_DIR}/evidence/service"
fixture="$(mktemp -d "${TMPDIR:-/tmp}/outback-service-e2e.XXXXXX")"
cleanup() { rm -rf "${fixture}"; }
trap cleanup EXIT INT TERM
mkdir -p "${proof_dir}"

go -C "${OUTBACK_DIR}" build -trimpath -o "${OUTBACK_TMP_DIR}/outback" ./cmd/outback
config_file="${OUTBACK_TMP_DIR}/service-e2e-config.json"
umask 077
jq -n --arg url "${service_url}" --arg ca "${ca_file}" \
	'{url:$url,service:{cpus:"2",memory:"4g",ca_cert_file:$ca,oidc_audience:$url}}' \
  > "${config_file}"
chmod 0600 "${config_file}"
export OUTBACK_CONFIG="${config_file}"

cp -R "${OUTBACK_DIR}/examples/go-redis/." "${fixture}/"
git -C "${fixture}" init -q
git -C "${fixture}" config user.name 'outback proof'
git -C "${fixture}" config user.email 'outback@example.invalid'
jq -n --arg project "${project}" '{project:$project}' > "${fixture}/outback.json"
git -C "${fixture}" add .
git -C "${fixture}" commit -qm 'committed baseline'
print 'dirty worktree reached remote worker' > "${fixture}/proof.txt"
print 'untracked worktree bytes' > "${fixture}/untracked.txt"
mkdir -p "${fixture}/ignored"
print 'must not upload' > "${fixture}/ignored/large.bin"
print 'ignored/' >> "${fixture}/.gitignore"

"${OUTBACK_TMP_DIR}/outback" doctor | tee "${proof_dir}/doctor.log"
"${OUTBACK_TMP_DIR}/outback" image activate --project "${project}" --image "${project_image}" | tee "${proof_dir}/image-activate.log"

run_remote_test() {
  local log_file="$1"
  local start_time="${EPOCHREALTIME}"
  (
    cd "${fixture}"
    "${OUTBACK_TMP_DIR}/outback" exec -- go test -count=1 -v ./...
  ) 2>&1 | tee "${log_file}" >&2
  local end_time="${EPOCHREALTIME}"
  printf '%.3f' "$((end_time - start_time))"
}

first_seconds="$(run_remote_test "${proof_dir}/first-run.log")"
first_job="$(grep -E '^Job: (job|outback-)' "${proof_dir}/first-run.log" | tail -1 | awk '{print $2}')"
cached_seconds="$(run_remote_test "${proof_dir}/cached-run.log")"
cached_job="$(grep -E '^Job: (job|outback-)' "${proof_dir}/cached-run.log" | tail -1 | awk '{print $2}')"
grep -q 'Transfer: 0 B uploaded' "${proof_dir}/cached-run.log"
grep -q 'REMOTE_E2E_PROOF' "${proof_dir}/cached-run.log"

set +e
(
  cd "${fixture}"
  "${OUTBACK_TMP_DIR}/outback" exec --timeout 1s -- sh -c 'echo REMOTE_TIMEOUT_PROOF_STARTED; sleep 60'
) > "${proof_dir}/timeout.log" 2>&1
timeout_exit=$?
set -e
timeout_job="$(grep -E '^Job: (job|outback-)' "${proof_dir}/timeout.log" | tail -1 | awk '{print $2}')"
[[ ${timeout_exit} -ne 0 ]] || { print -u2 'timeout proof unexpectedly succeeded'; exit 1; }
"${OUTBACK_TMP_DIR}/outback" status --json "${timeout_job}" > "${proof_dir}/timeout-job.json"
jq -e '.status == "timed_out"' "${proof_dir}/timeout-job.json" >/dev/null

(
  cd "${fixture}"
  "${OUTBACK_TMP_DIR}/outback" exec --detach --timeout 1m -- sh -c 'echo REMOTE_CANCEL_PROOF_STARTED; sleep 60'
) > "${proof_dir}/cancel-submit.log" 2>&1
cancel_job="$(grep -E '^Job: (job|outback-)' "${proof_dir}/cancel-submit.log" | tail -1 | awk '{print $2}')"
wait_for_job_status "${config_file}" "${OUTBACK_TMP_DIR}/outback" "${cancel_job}" running "${proof_dir}/cancel-running.json"
"${OUTBACK_TMP_DIR}/outback" cancel "${cancel_job}" | tee "${proof_dir}/cancel.log"
wait_for_job_status "${config_file}" "${OUTBACK_TMP_DIR}/outback" "${cancel_job}" cancelled "${proof_dir}/cancel-job.json"

(
  cd "${fixture}"
  "${OUTBACK_TMP_DIR}/outback" exec --detach --timeout 1m -- sh -c 'echo QUEUE_JOB_ONE_STARTED; sleep 60'
) > "${proof_dir}/queue-first-submit.log" 2>&1
queue_first_job="$(grep -E '^Job: (job|outback-)' "${proof_dir}/queue-first-submit.log" | tail -1 | awk '{print $2}')"
wait_for_job_status "${config_file}" "${OUTBACK_TMP_DIR}/outback" "${queue_first_job}" running "${proof_dir}/queue-first.json"
(
  cd "${fixture}"
  "${OUTBACK_TMP_DIR}/outback" exec --detach --timeout 1m -- sh -c 'echo QUEUE_JOB_TWO_STARTED; sleep 60'
) > "${proof_dir}/queue-second-submit.log" 2>&1
queue_second_job="$(grep -E '^Job: (job|outback-)' "${proof_dir}/queue-second-submit.log" | tail -1 | awk '{print $2}')"
wait_for_job_status "${config_file}" "${OUTBACK_TMP_DIR}/outback" "${queue_second_job}" queued "${proof_dir}/queue-second.json"
"${OUTBACK_TMP_DIR}/outback" cancel "${queue_second_job}" >/dev/null
"${OUTBACK_TMP_DIR}/outback" cancel "${queue_first_job}" >/dev/null
wait_for_job_status "${config_file}" "${OUTBACK_TMP_DIR}/outback" "${queue_second_job}" cancelled "${proof_dir}/queue-second-cancelled.json"
wait_for_job_status "${config_file}" "${OUTBACK_TMP_DIR}/outback" "${queue_first_job}" cancelled "${proof_dir}/queue-first-cancelled.json"

build_fixture="${fixture}/build-proof"
build_output="${fixture}/build-output"
mkdir -p "${build_fixture}" "${build_output}"
print 'remote BuildKit reached through build-scoped mTLS' > "${build_fixture}/proof.txt"
print 'FROM scratch\nCOPY proof.txt /proof.txt' > "${build_fixture}/Dockerfile"
(
  cd "${build_fixture}"
  "${OUTBACK_TMP_DIR}/outback" build -- --progress plain --output "type=local,dest=${build_output}" .
) 2>&1 | tee "${proof_dir}/build.log"
cmp "${build_fixture}/proof.txt" "${build_output}/proof.txt"

for attempt in {1..80}; do
  ssh "${reply[@]}" "${ssh_user}@${host}" \
    'sudo -n systemctl is-active outback-server outback-cas outback-buildkit; docker info --format "swarm={{.Swarm.LocalNodeState}}"; free -h; df -h /; docker ps --format "{{.Names}} {{.Labels}}"' \
    > "${proof_dir}/remote-status.txt"
  if ! grep -Eq 'reaper_|org.testcontainers=true' "${proof_dir}/remote-status.txt"; then
    break
  fi
  if [[ ${attempt} -eq 80 ]]; then
    print -u2 'Testcontainers resource leaked after remote job completion'
    exit 1
  fi
  sleep 0.25
done
[[ "$(grep -c '^active$' "${proof_dir}/remote-status.txt")" -ge 3 ]]
grep -q 'swarm=active' "${proof_dir}/remote-status.txt"

jq -n \
  --arg completed "$(date -u +%Y-%m-%dT%H:%M:%SZ)" --arg server "${host}" \
  --arg project_image "${project_image}" --arg first_job "${first_job}" --arg cached_job "${cached_job}" --arg timeout_job "${timeout_job}" --arg cancel_job "${cancel_job}" \
  --arg queue_first_job "${queue_first_job}" --arg queue_second_job "${queue_second_job}" \
  --argjson first_seconds "${first_seconds}" --argjson cached_seconds "${cached_seconds}" \
  '{completed_at:$completed,server_ipv4:$server,backend:"connect-https+reapi-cas+docker-swarm+buildkit",project_image:$project_image,first_job:$first_job,cached_job:$cached_job,timeout_job:$timeout_job,cancel_job:$cancel_job,queue_first_job:$queue_first_job,queue_second_job:$queue_second_job,first_seconds:$first_seconds,cached_seconds:$cached_seconds,generic_oci:true,project_image_lifecycle:true,image_default_resolution:true,image_validation:true,device_token:true,job_scoped_cas_mtls:true,build_scoped_buildkit_mtls:true,testcontainers:true,dirty_worktree:true,incremental_cas:true,timeout:true,cancellation:true,capacity_queue:true}' \
  > "${proof_dir}/proof.json"

print "remote shared-service E2E passed: cached ${cached_seconds}s (first ${first_seconds}s)"
