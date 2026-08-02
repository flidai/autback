#!/usr/bin/env bash

set -euo pipefail

job_cpus="${OUTBACK_JOB_CPUS:-1.5}"
job_memory="${OUTBACK_JOB_MEMORY:-2500m}"

if ! getent group outback >/dev/null; then
  groupadd --system outback
fi
if ! id outback >/dev/null 2>&1; then
  useradd --system --gid outback --groups docker --home-dir /var/lib/outback \
    --shell /usr/sbin/nologin outback
else
  usermod --append --groups docker outback
fi

install -d -o root -g outback -m 0750 /etc/outback
install -d -o outback -g outback -m 0700 /var/lib/outback /var/lib/outback/jobs /var/lib/outback/cache

if [[ ! -s /etc/outback/outback.env ]]; then
  umask 077
  printf 'OUTBACK_TOKEN=%s\n' "$(openssl rand -hex 32)" >/etc/outback/outback.env
fi

sed -i '/^OUTBACK_JOB_CPUS=/d; /^OUTBACK_JOB_MEMORY=/d' /etc/outback/outback.env
printf 'OUTBACK_JOB_CPUS=%s\nOUTBACK_JOB_MEMORY=%s\n' \
  "$job_cpus" "$job_memory" >>/etc/outback/outback.env
chown root:outback /etc/outback/outback.env
chmod 0640 /etc/outback/outback.env

install -o root -g root -m 0755 /tmp/outback-server /usr/local/bin/outback-server
install -o root -g root -m 0755 /tmp/outback-worker /usr/local/bin/outback-worker
install -o root -g root -m 0644 /tmp/outback-server.service /etc/systemd/system/outback-server.service
install -o root -g root -m 0644 /tmp/outback-worker.service /etc/systemd/system/outback-worker.service
install -o root -g root -m 0755 /tmp/maintain.sh /usr/local/sbin/outback-maintain
install -o root -g root -m 0644 /tmp/outback-maintenance.service /etc/systemd/system/outback-maintenance.service
install -o root -g root -m 0644 /tmp/outback-maintenance.timer /etc/systemd/system/outback-maintenance.timer

systemctl daemon-reload
systemctl enable outback-server outback-worker outback-maintenance.timer
