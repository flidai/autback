#!/usr/bin/env bash

set -euo pipefail

nativelink_image='ghcr.io/tracemachina/nativelink@sha256:56e4a01ebb3cc58c984627f9f284746b48b29abc95ea4363b8720bc8da5e50d4'
buildkit_image='moby/buildkit@sha256:0168606be2315b7c807a03b3d8aa79beefdb31c98740cebdffdfeebf31190c9f'
job_cpus="${OUTBACK_JOB_CPUS:-1.5}"
job_memory="${OUTBACK_JOB_MEMORY:-2500m}"

if ! getent group outback >/dev/null; then
  groupadd --system outback
fi
if ! id outback >/dev/null 2>&1; then
  useradd --system --gid outback --groups docker --home-dir /var/lib/outback --shell /usr/sbin/nologin outback
else
  usermod --append --groups docker outback
fi

install -d -o root -g outback -m 0750 /etc/outback
install -d -o outback -g outback -m 0700 \
  /var/lib/outback/reapi/cas/content /var/lib/outback/reapi/cas/tmp \
  /var/lib/outback/reapi/ac/content /var/lib/outback/reapi/ac/tmp \
  /var/lib/outback/reapi/worker/content /var/lib/outback/reapi/worker/tmp \
  /var/lib/outback/reapi/work
chown -R outback:outback /var/lib/outback/reapi

install -o root -g root -m 0755 /tmp/outback-reapi-entrypoint /usr/local/bin/outback-reapi-entrypoint
install -o root -g outback -m 0640 /tmp/nativelink.json5 /etc/outback/nativelink.json5
install -o root -g root -m 0644 /tmp/outback-nativelink.service /etc/systemd/system/outback-nativelink.service
install -o root -g root -m 0644 /tmp/outback-buildkit.service /etc/systemd/system/outback-buildkit.service

umask 027
{
  printf 'OUTBACK_RUNNER_IMAGE=%s\n' 'outback-runner-standard:local'
  printf 'OUTBACK_JOB_CPUS=%s\n' "$job_cpus"
  printf 'OUTBACK_JOB_MEMORY=%s\n' "$job_memory"
  printf 'OUTBACK_BUILDKIT_IMAGE=%s\n' "$buildkit_image"
} >/etc/outback/standards.env
chown root:outback /etc/outback/standards.env
chmod 0640 /etc/outback/standards.env

docker pull "$nativelink_image"
container_id="$(docker create "$nativelink_image")"
trap 'docker rm -f "$container_id" >/dev/null 2>&1 || true' EXIT
binary_path="$(docker image inspect --format '{{index .Config.Entrypoint 0}}' "$nativelink_image")"
docker cp "${container_id}:${binary_path}" /tmp/nativelink
install -o root -g root -m 0755 /tmp/nativelink /usr/local/bin/nativelink
docker rm "$container_id" >/dev/null
trap - EXIT

docker pull "$buildkit_image"
docker build --pull -t outback-runner-standard:local -f /tmp/Dockerfile /tmp

systemctl daemon-reload
systemctl enable --now outback-nativelink outback-buildkit
systemctl restart outback-nativelink outback-buildkit
systemctl --no-pager --quiet is-active outback-nativelink outback-buildkit
