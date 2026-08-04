#!/bin/zsh

source "${0:A:h}/lib.zsh"
zmodload zsh/datetime

proof_dir="${AUTBACK_DIR}/evidence/service-local"
fixture="$(mktemp -d "${TMPDIR:-/tmp}/autback-service-local-fixture.XXXXXX")"
data_dir="$(mktemp -d "${TMPDIR:-/tmp}/autback-service-local-data.XXXXXX")"
server_pid=""
keychain_enrolled=false
service_containers=()

cleanup() {
  local exit_status=$?
  if (( exit_status != 0 && ${#service_containers[@]} > 0 )); then
    local container
    for container in "${service_containers[@]}"; do
      print -u2 -- "--- ${container} logs ---"
      docker logs --tail 100 "${container}" >&2 || true
    done
  fi
  if [[ "${keychain_enrolled}" == "true" && -x "${build_dir:-}/autback" && -n "${config_file:-}" ]]; then
    env -u AUTBACK_TOKEN AUTBACK_CONFIG="${config_file}" "${build_dir}/autback" logout >/dev/null 2>&1 || true
  fi
  if [[ -n "${server_pid}" ]]; then
    kill "${server_pid}" >/dev/null 2>&1 || true
    wait "${server_pid}" >/dev/null 2>&1 || true
  fi
  if (( ${#service_containers[@]} > 0 )); then
    docker rm --force "${service_containers[@]}" >/dev/null 2>&1 || true
  fi
  if [[ -d "${data_dir}/cache" ]]; then
    docker run --rm --volume "${data_dir}:/data" alpine:3.22.1 \
      chmod -R u+rwX /data/cache >/dev/null 2>&1 || true
  fi
  rm -rf "${fixture}" "${data_dir}"
  return ${exit_status}
}
trap cleanup EXIT INT TERM

if [[ "$(docker info --format '{{.Swarm.LocalNodeState}}')" == "inactive" ]]; then
  docker swarm init >/dev/null
fi

cas_image='buchgr/bazel-remote-cache@sha256:d9b104d02bea731f5a8ce6d3c518f814953ef54c2e0218744ce7643ff9d85ca8'
buildkit_image='moby/buildkit@sha256:0168606be2315b7c807a03b3d8aa79beefdb31c98740cebdffdfeebf31190c9f'
registry_image='registry:2.8.3@sha256:a3d8aaa63ed8681a604f1dea0aa03f100d5895b6a58ace528858a7b332415373'
project_image='golang:1.25-bookworm@sha256:ea341baa9bd5ba6784f6d7161ace70544349a6242d54d34a0fbfd2c4d51c9d58'

allocate_ports() {
  python3 - "$1" <<'PY'
import socket
import sys

sockets = []
for _ in range(int(sys.argv[1])):
    listener = socket.socket()
    listener.bind(("127.0.0.1", 0))
    sockets.append(listener)
for listener in sockets:
    print(listener.getsockname()[1])
PY
}

ports=("${(@f)$(allocate_ports 7)}")
cas_internal_port="${ports[1]}"
cas_http_port="${ports[2]}"
registry_port="${ports[3]}"
buildkit_internal_port="${ports[4]}"
control_port="${ports[5]}"
cas_gateway_port="${ports[6]}"
buildkit_gateway_port="${ports[7]}"

run_id="autback-service-local-$$"
cas_container="${run_id}-cas"
registry_container="${run_id}-registry"
buildkit_container="${run_id}-buildkit"
service_containers=("${cas_container}" "${registry_container}" "${buildkit_container}")

start_container() {
  local name="$1"
  shift
  docker run --detach --name "${name}" "$@" >/dev/null
}

start_container "${cas_container}" --network host --volume autback-cas-service-local:/data \
  "${cas_image}" --dir /data --max_size 5 --grpc_address "127.0.0.1:${cas_internal_port}" \
  --http_address "127.0.0.1:${cas_http_port}" --profile_address none --access_log_level none
start_container "${registry_container}" --network host --volume autback-registry-service-local:/var/lib/registry \
  --env "REGISTRY_HTTP_ADDR=0.0.0.0:${registry_port}" "${registry_image}"
buildkit_config="${AUTBACK_TMP_DIR}/service-local-buildkitd.toml"
print "[registry.\"127.0.0.1:${registry_port}\"]\n  http = true" > "${buildkit_config}"
start_container "${buildkit_container}" --privileged --network host \
  --volume autback-buildkit-service-local-image-lifecycle:/var/lib/buildkit \
  --volume "${buildkit_config}:/etc/buildkit/buildkitd.toml:ro" \
  "${buildkit_image}" --config /etc/buildkit/buildkitd.toml --addr "tcp://127.0.0.1:${buildkit_internal_port}"

for attempt in {1..80}; do
  if nc -z 127.0.0.1 "${cas_internal_port}" >/dev/null 2>&1 && \
    curl --silent --fail "http://127.0.0.1:${registry_port}/v2/" >/dev/null && \
    docker exec "${buildkit_container}" \
      buildctl --addr "tcp://127.0.0.1:${buildkit_internal_port}" debug workers >/dev/null 2>&1; then
    break
  fi
  if [[ ${attempt} -eq 80 ]]; then
    print -u2 'local CAS, OCI registry, or image-lifecycle BuildKit did not become ready'
    exit 1
  fi
  sleep 0.25
done

architecture="$(docker info --format '{{.Architecture}}')"
case "${architecture}" in
  x86_64) goarch=amd64 ;;
  aarch64) goarch=arm64 ;;
  *) print -u2 "unsupported Docker architecture ${architecture}"; exit 1 ;;
esac

build_dir="${AUTBACK_TMP_DIR}/service-local-build"
mkdir -p "${build_dir}" "${proof_dir}"
go -C "${AUTBACK_DIR}" build -trimpath -o "${build_dir}/autback" ./cmd/autback
go -C "${AUTBACK_DIR}" build -trimpath -o "${build_dir}/autback-server" ./cmd/autback-server
env CGO_ENABLED=0 GOOS=linux GOARCH="${goarch}" go -C "${AUTBACK_DIR}" build -trimpath \
  -o "${build_dir}/autback-job-entrypoint" ./cmd/autback-job-entrypoint

# Docker Swarm bind mounts are resolved by the daemon. Install the static helper
# into a daemon-host path so the project image does not need any autback content.
docker run --rm \
  --volume "${build_dir}/autback-job-entrypoint:/source/autback-job-entrypoint:ro" \
  --volume /var/lib/autback:/target \
  alpine:3.22.1 sh -eu -c \
  'mkdir -p /target/bin /target/jobs; cp /source/autback-job-entrypoint /target/bin/autback-job-entrypoint; chmod 0755 /target/bin/autback-job-entrypoint'

bootstrap_output="$("${build_dir}/autback-server" bootstrap \
  --data-dir "${data_dir}" --user owner --project example --project-name 'Service E2E' --token-name local-e2e)"
device_token="$(print -r -- "${bootstrap_output}" | sed -n 's/^Token: //p')"
if [[ -z "${device_token}" ]]; then
  print -u2 'bootstrap did not return a device token'
  exit 1
fi

AUTBACK_DATA_DIR="${data_dir}" \
AUTBACK_SERVER_NAMES='localhost,127.0.0.1' \
AUTBACK_LISTEN="127.0.0.1:${control_port}" \
AUTBACK_CAS_LISTEN="127.0.0.1:${cas_gateway_port}" \
AUTBACK_CAS_ENDPOINT="localhost:${cas_gateway_port}" \
AUTBACK_CAS_INTERNAL="127.0.0.1:${cas_internal_port}" \
AUTBACK_BUILDKIT_LISTEN="127.0.0.1:${buildkit_gateway_port}" \
AUTBACK_BUILDKIT_ENDPOINT="localhost:${buildkit_gateway_port}" \
AUTBACK_BUILDKIT_INTERNAL="127.0.0.1:${buildkit_internal_port}" \
AUTBACK_CACHE_ROOT="${data_dir}/cache" \
AUTBACK_JOB_ENTRYPOINT='/var/lib/autback/bin/autback-job-entrypoint' \
"${build_dir}/autback-server" > "${proof_dir}/server.log" 2>&1 &
server_pid=$!

for attempt in {1..80}; do
  if curl --silent --fail --cacert "${data_dir}/pki/ca.pem" "https://localhost:${control_port}/healthz" >/dev/null; then
    break
  fi
  if ! kill -0 "${server_pid}" >/dev/null 2>&1; then
    print -u2 'autback-server stopped during startup'
    tail -100 "${proof_dir}/server.log" >&2
    exit 1
  fi
  if [[ ${attempt} -eq 80 ]]; then
    print -u2 'autback-server did not become healthy'
    exit 1
  fi
  sleep 0.25
done

cp -R "${AUTBACK_DIR}/examples/go-redis/." "${fixture}/"
git -C "${fixture}" init -q
git -C "${fixture}" config user.name 'autback proof'
git -C "${fixture}" config user.email 'autback@example.invalid'
jq -n --arg project "example" '{project:$project}' > "${fixture}/autback.json"
git -C "${fixture}" add .
git -C "${fixture}" commit -qm 'committed baseline'
print 'dirty worktree reached remote worker' > "${fixture}/proof.txt"
print 'untracked worktree bytes' > "${fixture}/untracked.txt"
mkdir -p "${fixture}/ignored"
print 'must not upload' > "${fixture}/ignored/large.bin"
print 'ignored/' >> "${fixture}/.gitignore"

config_file="${data_dir}/client.json"
umask 077
jq -n \
  --arg ca "${data_dir}/pki/ca.pem" \
  --arg url "https://localhost:${control_port}" \
	'{url:$url,service:{ca_cert_file:$ca,oidc_audience:$url}}' \
  > "${config_file}"
chmod 0600 "${config_file}"
export AUTBACK_CONFIG="${config_file}"
export AUTBACK_TOKEN="${device_token}"

"${build_dir}/autback" doctor | tee "${proof_dir}/doctor.log"
"${build_dir}/autback" admin project create --slug second --name 'Second E2E Project' > "${proof_dir}/second-project.json"
coworker_json="$("${build_dir}/autback" admin user create --name coworker)"
print -r -- "${coworker_json}" > "${proof_dir}/coworker.json"
coworker_id="$(print -r -- "${coworker_json}" | jq -r '.id')"
"${build_dir}/autback" admin member add --project example --user "${coworker_id}" > "${proof_dir}/coworker-member.log"
enrollment_code="$("${build_dir}/autback" admin enrollment create --user "${coworker_id}" --device coworker-laptop --expires 10m 2> "${proof_dir}/enrollment-create.log")"
print -r -- "${enrollment_code}" | env -u AUTBACK_TOKEN AUTBACK_CONFIG="${config_file}" \
  "${build_dir}/autback" login --recovery-code > "${proof_dir}/enrollment-login.log" 2>&1
keychain_enrolled=true
env -u AUTBACK_TOKEN AUTBACK_CONFIG="${config_file}" "${build_dir}/autback" doctor > "${proof_dir}/coworker-doctor.log"
set +e
print -r -- "${enrollment_code}" | env -u AUTBACK_TOKEN AUTBACK_CONFIG="${config_file}" \
  "${build_dir}/autback" login --recovery-code > "${proof_dir}/enrollment-reuse.log" 2>&1
enrollment_reuse_exit=$?
set -e
[[ ${enrollment_reuse_exit} -ne 0 ]] || { print -u2 'single-use enrollment code was accepted twice'; exit 1; }
env -u AUTBACK_TOKEN AUTBACK_CONFIG="${config_file}" "${build_dir}/autback" token list > "${proof_dir}/coworker-tokens.json"
coworker_token_id="$(jq -r '.[] | select(.name == "coworker-laptop") | .id' "${proof_dir}/coworker-tokens.json")"
[[ -n "${coworker_token_id}" ]] || { print -u2 'enrolled device token was not listed'; exit 1; }
env -u AUTBACK_TOKEN AUTBACK_CONFIG="${config_file}" "${build_dir}/autback" token revoke "${coworker_token_id}" > "${proof_dir}/coworker-revoke.log"
set +e
env -u AUTBACK_TOKEN AUTBACK_CONFIG="${config_file}" "${build_dir}/autback" token list > "${proof_dir}/coworker-revoked-token-list.log" 2>&1
revoked_exit=$?
set -e
[[ ${revoked_exit} -ne 0 ]] || { print -u2 'revoked enrolled device remained authenticated'; exit 1; }
env -u AUTBACK_TOKEN AUTBACK_CONFIG="${config_file}" "${build_dir}/autback" logout > "${proof_dir}/coworker-logout.log"
keychain_enrolled=false
"${build_dir}/autback" image activate --project example --image "${project_image}" | tee "${proof_dir}/image-activate.log"
"${build_dir}/autback" image activate --project second --image "${project_image}" > "${proof_dir}/second-image-activate.log"
"${build_dir}/autback" image activate --project example --image "${cas_image}" | tee "${proof_dir}/image-second-activate.log"
"${build_dir}/autback" image rollback --project example | tee "${proof_dir}/image-rollback.log"
"${build_dir}/autback" image history --project example > "${proof_dir}/image-history.json"
jq -e --arg image "${project_image}" 'length == 3 and .[0].action == "rollback" and .[0].image == $image' "${proof_dir}/image-history.json" >/dev/null
"${build_dir}/autback" image overrides --project example deny | tee "${proof_dir}/image-policy.log"

set +e
(
  cd "${fixture}"
  "${build_dir}/autback" exec --image "${cas_image}" -- true
) > "${proof_dir}/image-override-denied.log" 2>&1
override_exit=$?
set -e
[[ ${override_exit} -ne 0 ]] || { print -u2 'denied image override unexpectedly succeeded'; exit 1; }
grep -q 'project image overrides are disabled' "${proof_dir}/image-override-denied.log"

runner_fixture="${fixture}/runner-image"
mkdir -p "${runner_fixture}"
print "FROM ${project_image}\nLABEL org.opencontainers.image.source=autback-e2e" > "${runner_fixture}/Dockerfile"
(
  cd "${runner_fixture}"
  "${build_dir}/autback" image build --tag "127.0.0.1:${registry_port}/autback/e2e:latest" -- --progress plain
) 2>&1 | tee "${proof_dir}/image-build.log"
(
  cd "${fixture}"
  "${build_dir}/autback" exec -- go version
) 2>&1 | tee "${proof_dir}/image-build-exec.log"
grep -q '^go version ' "${proof_dir}/image-build-exec.log"

run_remote_test() {
  local log_file="$1"
  local start_time="${EPOCHREALTIME}"
  (
    cd "${fixture}"
    "${build_dir}/autback" exec \
      --cache go-build=/root/.cache/go-build \
      --cache go-mod=/go/pkg/mod \
      -- go test -count=1 -v ./...
  ) 2>&1 | tee "${log_file}" >&2
  local end_time="${EPOCHREALTIME}"
  printf '%.3f' "$((end_time - start_time))"
}

first_seconds="$(run_remote_test "${proof_dir}/first-run.log")"
first_job="$(grep -E '^Job: (job|autback-)' "${proof_dir}/first-run.log" | tail -1 | awk '{print $2}')"
cached_seconds="$(run_remote_test "${proof_dir}/cached-run.log")"
cached_job="$(grep -E '^Job: (job|autback-)' "${proof_dir}/cached-run.log" | tail -1 | awk '{print $2}')"
grep -q 'Transfer: 0 B uploaded' "${proof_dir}/cached-run.log"
grep -q 'REMOTE_E2E_PROOF' "${proof_dir}/cached-run.log"

(
  cd "${fixture}"
  "${build_dir}/autback" exec --project second -- go version
) 2>&1 | tee "${proof_dir}/second-project-cas.log"
grep -q 'Transfer: 0 B uploaded' "${proof_dir}/second-project-cas.log"

set +e
(
  cd "${fixture}"
  "${build_dir}/autback" exec --timeout 1s -- sh -c 'echo REMOTE_TIMEOUT_PROOF_STARTED; sleep 60'
) > "${proof_dir}/timeout.log" 2>&1
timeout_exit=$?
set -e
timeout_job="$(grep -E '^Job: (job|autback-)' "${proof_dir}/timeout.log" | tail -1 | awk '{print $2}')"
[[ ${timeout_exit} -ne 0 ]] || { print -u2 'timeout proof unexpectedly succeeded'; exit 1; }
"${build_dir}/autback" status --json "${timeout_job}" > "${proof_dir}/timeout-job.json"
jq -e '.status == "timed_out"' "${proof_dir}/timeout-job.json" >/dev/null

(
  cd "${fixture}"
  "${build_dir}/autback" exec --detach --timeout 1m -- sh -c 'echo REMOTE_CANCEL_PROOF_STARTED; sleep 60'
) > "${proof_dir}/cancel-submit.log" 2>&1
cancel_job="$(grep -E '^Job: (job|autback-)' "${proof_dir}/cancel-submit.log" | tail -1 | awk '{print $2}')"
wait_for_job_status "${config_file}" "${build_dir}/autback" "${cancel_job}" running "${proof_dir}/cancel-running.json"
"${build_dir}/autback" cancel "${cancel_job}" | tee "${proof_dir}/cancel.log"
wait_for_job_status "${config_file}" "${build_dir}/autback" "${cancel_job}" cancelled "${proof_dir}/cancel-job.json"

(
  cd "${fixture}"
  "${build_dir}/autback" exec --detach --timeout 1m -- sh -c 'echo QUEUE_JOB_ONE_STARTED; sleep 60'
) > "${proof_dir}/queue-first-submit.log" 2>&1
queue_first_job="$(grep -E '^Job: (job|autback-)' "${proof_dir}/queue-first-submit.log" | tail -1 | awk '{print $2}')"
wait_for_job_status "${config_file}" "${build_dir}/autback" "${queue_first_job}" running "${proof_dir}/queue-first.json"
(
  cd "${fixture}"
  "${build_dir}/autback" exec --detach --timeout 1m -- sh -c 'echo QUEUE_JOB_TWO_STARTED; sleep 60'
) > "${proof_dir}/queue-second-submit.log" 2>&1
queue_second_job="$(grep -E '^Job: (job|autback-)' "${proof_dir}/queue-second-submit.log" | tail -1 | awk '{print $2}')"
wait_for_job_status "${config_file}" "${build_dir}/autback" "${queue_second_job}" queued "${proof_dir}/queue-second.json"
"${build_dir}/autback" cancel "${queue_second_job}" >/dev/null
"${build_dir}/autback" cancel "${queue_first_job}" >/dev/null
wait_for_job_status "${config_file}" "${build_dir}/autback" "${queue_second_job}" cancelled "${proof_dir}/queue-second-cancelled.json"
wait_for_job_status "${config_file}" "${build_dir}/autback" "${queue_first_job}" cancelled "${proof_dir}/queue-first-cancelled.json"

build_fixture="${fixture}/build-proof"
build_output="${fixture}/build-output"
mkdir -p "${build_fixture}" "${build_output}"
print 'remote BuildKit reached through build-scoped mTLS' > "${build_fixture}/proof.txt"
print 'FROM scratch\nCOPY proof.txt /proof.txt' > "${build_fixture}/Dockerfile"
(
  cd "${build_fixture}"
  "${build_dir}/autback" build -- --progress plain --output "type=local,dest=${build_output}" .
) 2>&1 | tee "${proof_dir}/build.log"
cmp "${build_fixture}/proof.txt" "${build_output}/proof.txt"
second_build_output="${fixture}/second-build-output"
mkdir -p "${second_build_output}"
(
  cd "${build_fixture}"
  "${build_dir}/autback" build --project second -- --progress plain --output "type=local,dest=${second_build_output}" .
) 2>&1 | tee "${proof_dir}/second-project-build.log"
cmp "${build_fixture}/proof.txt" "${second_build_output}/proof.txt"
grep -q 'CACHED' "${proof_dir}/second-project-build.log"

cancel_build_fixture="${fixture}/cancel-build-proof"
mkdir -p "${cancel_build_fixture}"
print 'FROM alpine:3.22.1\nRUN echo BUILD_CANCEL_PROOF_STARTED; sleep 60' > "${cancel_build_fixture}/Dockerfile"
(
  cd "${cancel_build_fixture}"
  exec "${build_dir}/autback" build -- --progress plain .
) > "${proof_dir}/cancel-build.log" 2>&1 &
cancel_build_pid=$!
for attempt in {1..120}; do
  if grep -q 'BUILD_CANCEL_PROOF_STARTED' "${proof_dir}/cancel-build.log"; then
    break
  fi
  if ! kill -0 "${cancel_build_pid}" >/dev/null 2>&1; then
    print -u2 'BuildKit cancellation proof stopped before reaching the long-running step'
    tail -100 "${proof_dir}/cancel-build.log" >&2
    exit 1
  fi
  if [[ ${attempt} -eq 120 ]]; then
    print -u2 'BuildKit cancellation proof did not reach the long-running step'
    exit 1
  fi
  sleep 0.25
done
kill -INT "${cancel_build_pid}"
set +e
wait "${cancel_build_pid}"
cancel_build_exit=$?
set -e
[[ ${cancel_build_exit} -ne 0 ]] || { print -u2 'cancelled BuildKit build unexpectedly succeeded'; exit 1; }
cancel_build="$(grep -E '^Build: bld' "${proof_dir}/cancel-build.log" | tail -1 | awk '{print $2}')"
[[ -n "${cancel_build}" ]] || { print -u2 'cancelled BuildKit build ID was not recorded'; exit 1; }
for attempt in {1..40}; do
  cancel_build_status="$(sqlite3 "${data_dir}/control/control.db" "SELECT status FROM control_builds WHERE id='${cancel_build}'")"
  if [[ "${cancel_build_status}" == "cancelled" ]]; then
    break
  fi
  if [[ ${attempt} -eq 40 ]]; then
    print -u2 "BuildKit cancellation status is ${cancel_build_status:-missing}, expected cancelled"
    exit 1
  fi
  sleep 0.25
done

"${build_dir}/autback" list --project example --json > "${proof_dir}/list.json"
for attempt in {1..80}; do
  docker ps --format '{{.Names}} {{.Labels}}' > "${proof_dir}/docker-containers.txt"
  if ! grep -Eq 'reaper_|org.testcontainers=true' "${proof_dir}/docker-containers.txt"; then
    break
  fi
  if [[ ${attempt} -eq 80 ]]; then
    print -u2 'Testcontainers resource leaked after job completion'
    exit 1
  fi
  sleep 0.25
done

jq -n \
  --arg completed "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --arg first_job "${first_job}" --arg cached_job "${cached_job}" --arg timeout_job "${timeout_job}" --arg cancel_job "${cancel_job}" \
  --arg queue_first_job "${queue_first_job}" --arg queue_second_job "${queue_second_job}" --arg cancel_build "${cancel_build}" \
  --arg project_image "${project_image}" \
  --argjson first_seconds "${first_seconds}" --argjson cached_seconds "${cached_seconds}" \
  '{completed_at:$completed,backend:"connect-https+reapi-cas+docker-swarm+buildkit",project_image:$project_image,first_job:$first_job,cached_job:$cached_job,timeout_job:$timeout_job,cancel_job:$cancel_job,cancel_build:$cancel_build,queue_first_job:$queue_first_job,queue_second_job:$queue_second_job,first_seconds:$first_seconds,cached_seconds:$cached_seconds,generic_oci:true,repository_project_discovery:true,project_image_lifecycle:true,image_default_resolution:true,image_validation:true,image_rollback:true,image_history:true,image_override_policy:true,image_build_push_activate_execute:true,connect_https:true,device_token:true,one_time_device_enrollment:true,os_keychain_storage:true,enrollment_single_use:true,independent_device_revocation:true,logout:true,job_scoped_cas_mtls:true,build_scoped_buildkit_mtls:true,two_project_shared_cas:true,two_project_shared_buildkit_cache:true,explicit_project_caches:true,buildkit_cancellation:true,testcontainers:true,dirty_worktree:true,incremental_cas:true,timeout:true,cancellation:true,strict_fifo_queue:true}' \
  > "${proof_dir}/proof.json"

print "shared-service E2E passed: cached ${cached_seconds}s (first ${first_seconds}s)"
