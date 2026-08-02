#!/bin/zsh

source "${0:A:h}/lib.zsh"

: "${OUTBACK_SERVER_IP:?set OUTBACK_SERVER_IP to an existing host; this script never provisions infrastructure}"
: "${OUTBACK_SSH_KEY:?set OUTBACK_SSH_KEY to the existing host identity file}"
[[ -f "${OUTBACK_SSH_KEY}" ]] || { print -u2 "missing ${OUTBACK_SSH_KEY}"; exit 1; }

host="${OUTBACK_SERVER_IP}"
ssh_user="${OUTBACK_SSH_USER:-root}"
server_names="${OUTBACK_SERVER_NAMES:-${host}}"
project_slug="${OUTBACK_PROJECT:-default}"
project_name="${OUTBACK_PROJECT_NAME:-Default}"
ssh_args
ssh "${reply[@]}" "${ssh_user}@${host}" 'docker version >/dev/null'

build_dir="${OUTBACK_TMP_DIR}/service-build"
config_dir="${XDG_CONFIG_HOME:-${HOME}/.config}/outback"
install_dir="${OUTBACK_INSTALL_DIR:-${HOME}/.local/bin}"
config_file="${config_dir}/config.json"
ca_file="${config_dir}/ca.pem"
mkdir -p "${build_dir}" "${config_dir}" "${install_dir}"
env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go -C "${OUTBACK_DIR}" build -trimpath -ldflags='-s -w' \
  -o "${build_dir}/outback-server" ./cmd/outback-server
env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go -C "${OUTBACK_DIR}" build -trimpath -ldflags='-s -w' \
  -o "${build_dir}/outback-job-entrypoint" ./cmd/outback-job-entrypoint
go -C "${OUTBACK_DIR}" build -trimpath -o "${build_dir}/outback" ./cmd/outback

scp "${reply[@]}" \
  "${build_dir}/outback-server" \
  "${build_dir}/outback-job-entrypoint" \
  "${OUTBACK_DIR}/host/install-swarm.sh" \
  "${OUTBACK_DIR}/host/outback-server.service" \
  "${OUTBACK_DIR}/host/outback-cas.service" \
  "${OUTBACK_DIR}/host/outback-buildkit.service" \
  "${OUTBACK_DIR}/host/maintain.sh" \
  "${OUTBACK_DIR}/host/outback-maintenance.service" \
  "${OUTBACK_DIR}/host/outback-maintenance.timer" \
  "${ssh_user}@${host}:/tmp/"

ssh "${reply[@]}" "${ssh_user}@${host}" \
  "sudo -n env OUTBACK_SERVER_NAMES=${(q)server_names} OUTBACK_BOOTSTRAP_PROJECT=${(q)project_slug} OUTBACK_BOOTSTRAP_PROJECT_NAME=${(q)project_name} bash /tmp/install-swarm.sh"

umask 077
ssh "${reply[@]}" "${ssh_user}@${host}" 'sudo -n cat /var/lib/outback/pki/ca.pem' > "${ca_file}"
chmod 0600 "${ca_file}"
jq -n \
  --arg url "https://${server_names%%,*}" \
  --arg image "${OUTBACK_PROJECT_IMAGE:-}" \
  --arg ca "${ca_file}" \
	'{url:$url,service:{image:$image,ca_cert_file:$ca,oidc_audience:$url}}' \
  > "${config_file}"
chmod 0600 "${config_file}"
install -m 0755 "${build_dir}/outback" "${install_dir}/outback"

if ssh "${reply[@]}" "${ssh_user}@${host}" 'sudo -n test -f /etc/outback/bootstrap-token'; then
  bootstrap_token="$(ssh "${reply[@]}" "${ssh_user}@${host}" 'sudo -n cat /etc/outback/bootstrap-token')"
  OUTBACK_CONFIG="${config_file}" "${build_dir}/outback" login --token "${bootstrap_token}"
  unset bootstrap_token
  ssh "${reply[@]}" "${ssh_user}@${host}" 'sudo -n shred -u /etc/outback/bootstrap-token 2>/dev/null || sudo -n rm -f /etc/outback/bootstrap-token'
fi

OUTBACK_CONFIG="${config_file}" "${build_dir}/outback" doctor
print "deployed outback shared service to existing host ${host}"
print "configured ${config_file}; the device token is stored in the OS keychain"
