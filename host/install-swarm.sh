#!/usr/bin/env bash

set -euo pipefail

cas_image='buchgr/bazel-remote-cache@sha256:d9b104d02bea731f5a8ce6d3c518f814953ef54c2e0218744ce7643ff9d85ca8'
buildkit_image='moby/buildkit@sha256:0168606be2315b7c807a03b3d8aa79beefdb31c98740cebdffdfeebf31190c9f'
server_names="${AUTBACK_SERVER_NAMES:?AUTBACK_SERVER_NAMES is required}"
public_name="${server_names%%,*}"
project_slug="${AUTBACK_BOOTSTRAP_PROJECT:-default}"
project_name="${AUTBACK_BOOTSTRAP_PROJECT_NAME:-Default}"

if ! id autback >/dev/null 2>&1; then
  useradd --system --home-dir /var/lib/autback --shell /usr/sbin/nologin --groups docker autback
fi
usermod -aG docker autback

install -d -o root -g root -m 0700 /etc/autback
install -d -o autback -g autback -m 0700 /var/lib/autback /var/lib/autback/control /var/lib/autback/pki /var/lib/autback/jobs /var/lib/autback/cache
install -d -o 65532 -g 65532 -m 0700 /var/lib/autback/cas
install -d -o root -g root -m 0755 /usr/local/lib/autback
install -o root -g root -m 0755 /tmp/autback-server /usr/local/bin/autback-server
install -o root -g root -m 0755 /tmp/autback-job-entrypoint /usr/local/lib/autback/autback-job-entrypoint
install -o root -g root -m 0644 /tmp/autback-server.service /etc/systemd/system/autback-server.service
install -o root -g root -m 0644 /tmp/autback-cas.service /etc/systemd/system/autback-cas.service
install -o root -g root -m 0644 /tmp/autback-buildkit.service /etc/systemd/system/autback-buildkit.service
install -o root -g root -m 0644 /tmp/autback-maintenance.service /etc/systemd/system/autback-maintenance.service
install -o root -g root -m 0644 /tmp/autback-maintenance.timer /etc/systemd/system/autback-maintenance.timer

umask 077
disk_bytes="$(df --output=size --block-size=1 /var/lib/autback | tail -1 | tr -d ' ')"
gib=1073741824
cas_max="$(( disk_bytes / 4 / gib ))"
(( cas_max < 1 )) && cas_max=1
(( cas_max > 40 )) && cas_max=40
cas_hard="$(( cas_max + (cas_max + 19) / 20 ))"
buildkit_max="$(( disk_bytes / 10 ))"
(( buildkit_max > 10 * gib )) && buildkit_max="$(( 10 * gib ))"
buildkit_reserved="$(( 2 * gib ))"
(( buildkit_reserved >= buildkit_max )) && buildkit_reserved="$(( buildkit_max / 2 ))"
cat >/etc/autback/buildkitd.toml <<EOF
[worker.oci]
  gc = true
  reservedSpace = ${buildkit_reserved}
  maxUsedSpace = ${buildkit_max}
  minFreeSpace = "20%"

  [[worker.oci.gcpolicy]]
    reservedSpace = "512MB"
    maxUsedSpace = ${buildkit_max}
    minFreeSpace = "20%"
    keepDuration = "48h"
    filters = ["type==source.local", "type==exec.cachemount", "type==source.git.checkout"]

  [[worker.oci.gcpolicy]]
    all = true
    reservedSpace = ${buildkit_reserved}
    maxUsedSpace = ${buildkit_max}
    minFreeSpace = "20%"
EOF
chmod 0600 /etc/autback/buildkitd.toml
{
  printf 'AUTBACK_CAS_IMAGE=%s\n' "$cas_image"
  printf 'AUTBACK_CAS_MAX_SIZE=%s\n' "$cas_max"
  printf 'AUTBACK_CAS_HARD_LIMIT=%s\n' "$cas_hard"
  printf 'AUTBACK_BUILDKIT_IMAGE=%s\n' "$buildkit_image"
} >/etc/autback/swarm.env
{
  printf 'AUTBACK_DATA_DIR=/var/lib/autback\n'
  printf 'AUTBACK_WORKER_OWNERSHIP=exclusive\n'
  printf 'AUTBACK_SERVER_NAMES=%s\n' "$server_names"
  printf 'AUTBACK_LISTEN=:443\n'
  printf 'AUTBACK_CAS_INTERNAL=127.0.0.1:50051\n'
  printf 'AUTBACK_CAS_LISTEN=:50052\n'
  printf 'AUTBACK_CAS_ENDPOINT=%s:50052\n' "$public_name"
  printf 'AUTBACK_BUILDKIT_INTERNAL=127.0.0.1:1234\n'
  printf 'AUTBACK_BUILDKIT_LISTEN=:1235\n'
  printf 'AUTBACK_BUILDKIT_ENDPOINT=%s:1235\n' "$public_name"
  printf 'AUTBACK_JOB_ENTRYPOINT=/usr/local/lib/autback/autback-job-entrypoint\n'
  printf 'AUTBACK_GITHUB_OIDC_AUDIENCE=https://%s\n' "$public_name"
} >/etc/autback/service.env

systemctl daemon-reload
systemctl stop autback-maintenance.timer autback-maintenance.service autback-server

docker pull "$cas_image"
docker pull "$buildkit_image"

swarm_state="$(docker info --format '{{.Swarm.LocalNodeState}}')"
if [[ "$swarm_state" == "inactive" ]]; then
  advertise_address="$(hostname -I | awk '{print $1}')"
  docker swarm init --advertise-addr "$advertise_address"
elif [[ "$swarm_state" != "active" ]]; then
  printf 'Docker Swarm state is %s; expected inactive or active\n' "$swarm_state" >&2
  exit 1
fi

if [[ ! -f /var/lib/autback/control/control.db ]]; then
  bootstrap_output="$(runuser -u autback -- /usr/local/bin/autback-server bootstrap \
    --data-dir /var/lib/autback --user owner --project "$project_slug" --project-name "$project_name" --token-name bootstrap-device)"
  bootstrap_token="$(sed -n 's/^Token: //p' <<<"$bootstrap_output")"
  [[ -n "$bootstrap_token" ]] || { printf 'bootstrap did not return a token\n' >&2; exit 1; }
  printf '%s\n' "$bootstrap_token" >/etc/autback/bootstrap-token
  chmod 0600 /etc/autback/bootstrap-token
fi

systemctl enable --now autback-cas autback-buildkit autback-server autback-maintenance.timer
systemctl restart autback-cas autback-buildkit autback-server
systemctl --no-pager --quiet is-active autback-cas autback-buildkit autback-server autback-maintenance.timer

if command -v ufw >/dev/null 2>&1 && ufw status | grep -q '^Status: active'; then
  ufw allow 443/tcp >/dev/null
  ufw allow 50052/tcp >/dev/null
  ufw allow 1235/tcp >/dev/null
fi

printf 'autback service installed for https://%s\n' "$public_name"
