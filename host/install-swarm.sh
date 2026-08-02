#!/usr/bin/env bash

set -euo pipefail

cas_image='buchgr/bazel-remote-cache@sha256:d9b104d02bea731f5a8ce6d3c518f814953ef54c2e0218744ce7643ff9d85ca8'
buildkit_image='moby/buildkit@sha256:0168606be2315b7c807a03b3d8aa79beefdb31c98740cebdffdfeebf31190c9f'
server_names="${OUTBACK_SERVER_NAMES:?OUTBACK_SERVER_NAMES is required}"
public_name="${server_names%%,*}"
project_slug="${OUTBACK_BOOTSTRAP_PROJECT:-default}"
project_name="${OUTBACK_BOOTSTRAP_PROJECT_NAME:-Default}"

if ! id outback >/dev/null 2>&1; then
  useradd --system --home-dir /var/lib/outback --shell /usr/sbin/nologin --groups docker outback
fi
usermod -aG docker outback

install -d -o root -g root -m 0700 /etc/outback
install -d -o outback -g outback -m 0700 /var/lib/outback /var/lib/outback/control /var/lib/outback/pki /var/lib/outback/jobs /var/lib/outback/cache
install -d -o 65532 -g 65532 -m 0700 /var/lib/outback/cas
install -d -o root -g root -m 0755 /usr/local/lib/outback
install -o root -g root -m 0755 /tmp/outback-server /usr/local/bin/outback-server
install -o root -g root -m 0755 /tmp/outback-job-entrypoint /usr/local/lib/outback/outback-job-entrypoint
install -o root -g root -m 0644 /tmp/outback-server.service /etc/systemd/system/outback-server.service
install -o root -g root -m 0644 /tmp/outback-cas.service /etc/systemd/system/outback-cas.service
install -o root -g root -m 0644 /tmp/outback-buildkit.service /etc/systemd/system/outback-buildkit.service
install -o root -g root -m 0755 /tmp/maintain.sh /usr/local/sbin/outback-maintain
install -o root -g root -m 0644 /tmp/outback-maintenance.service /etc/systemd/system/outback-maintenance.service
install -o root -g root -m 0644 /tmp/outback-maintenance.timer /etc/systemd/system/outback-maintenance.timer

umask 077
{
  printf 'OUTBACK_CAS_IMAGE=%s\n' "$cas_image"
  printf 'OUTBACK_BUILDKIT_IMAGE=%s\n' "$buildkit_image"
} >/etc/outback/swarm.env
{
  printf 'OUTBACK_DATA_DIR=/var/lib/outback\n'
  printf 'OUTBACK_SERVER_NAMES=%s\n' "$server_names"
  printf 'OUTBACK_LISTEN=:443\n'
  printf 'OUTBACK_CAS_INTERNAL=127.0.0.1:50051\n'
  printf 'OUTBACK_CAS_LISTEN=:50052\n'
  printf 'OUTBACK_CAS_ENDPOINT=%s:50052\n' "$public_name"
  printf 'OUTBACK_BUILDKIT_INTERNAL=127.0.0.1:1234\n'
  printf 'OUTBACK_BUILDKIT_LISTEN=:1235\n'
  printf 'OUTBACK_BUILDKIT_ENDPOINT=%s:1235\n' "$public_name"
  printf 'OUTBACK_JOB_ENTRYPOINT=/usr/local/lib/outback/outback-job-entrypoint\n'
  printf 'OUTBACK_GITHUB_OIDC_AUDIENCE=https://%s\n' "$public_name"
} >/etc/outback/service.env

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

if [[ ! -f /var/lib/outback/control/control.db ]]; then
  bootstrap_output="$(runuser -u outback -- /usr/local/bin/outback-server bootstrap \
    --data-dir /var/lib/outback --user owner --project "$project_slug" --project-name "$project_name" --token-name bootstrap-device)"
  bootstrap_token="$(sed -n 's/^Token: //p' <<<"$bootstrap_output")"
  [[ -n "$bootstrap_token" ]] || { printf 'bootstrap did not return a token\n' >&2; exit 1; }
  printf '%s\n' "$bootstrap_token" >/etc/outback/bootstrap-token
  chmod 0600 /etc/outback/bootstrap-token
fi

systemctl daemon-reload
systemctl enable --now outback-cas outback-buildkit outback-server outback-maintenance.timer
systemctl restart outback-cas outback-buildkit outback-server
systemctl --no-pager --quiet is-active outback-cas outback-buildkit outback-server outback-maintenance.timer

if command -v ufw >/dev/null 2>&1 && ufw status | grep -q '^Status: active'; then
  ufw allow 443/tcp >/dev/null
  ufw allow 50052/tcp >/dev/null
  ufw allow 1235/tcp >/dev/null
fi

printf 'outback service installed for https://%s\n' "$public_name"
