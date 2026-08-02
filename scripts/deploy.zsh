#!/bin/zsh

source "${0:A:h}/lib.zsh"
load_secrets
ensure_ssh_key

host="$(server_ip)"
ssh_args
ssh "${reply[@]}" "root@${host}" 'cloud-init status --wait >/dev/null 2>&1 || true'

build_dir="${OUTBACK_TMP_DIR}/build"
mkdir -p "${build_dir}"
for binary in outback-server outback-worker; do
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go -C "${OUTBACK_DIR}" build -trimpath -ldflags='-s -w' -o "${build_dir}/${binary}" "./cmd/${binary}"
done

scp "${reply[@]}" "${build_dir}/outback-server" "${build_dir}/outback-worker" \
  "${OUTBACK_DIR}/runner/standard/Dockerfile" \
  "${OUTBACK_DIR}/host/install.sh" \
  "${OUTBACK_DIR}/host/outback-server.service" \
  "${OUTBACK_DIR}/host/outback-worker.service" \
  "${OUTBACK_DIR}/host/maintain.sh" \
  "${OUTBACK_DIR}/host/outback-maintenance.service" \
  "${OUTBACK_DIR}/host/outback-maintenance.timer" \
  "root@${host}:/tmp/"
ssh "${reply[@]}" "root@${host}" \
  "OUTBACK_JOB_CPUS=${OUTBACK_JOB_CPUS:-1.5} OUTBACK_JOB_MEMORY=${OUTBACK_JOB_MEMORY:-2500m} bash /tmp/install.sh && \
   docker build --pull -t outback-runner-standard:local -f /tmp/Dockerfile /tmp && \
   systemctl enable --now outback-server outback-worker outback-maintenance.timer && \
   systemctl restart outback-server outback-worker"

client_env="${OUTBACK_TMP_DIR}/client.env"
umask 077
ssh "${reply[@]}" "root@${host}" 'cat /etc/outback/outback.env' > "${client_env}"
print "OUTBACK_URL=http://127.0.0.1:18080" >> "${client_env}"
chmod 0600 "${client_env}"

ssh "${reply[@]}" "root@${host}" 'systemctl --no-pager --quiet is-active outback-server outback-worker'
"${OUTBACK_DIR}/scripts/install-cli.zsh"
print "deployed coordinator and worker to ${host}"
