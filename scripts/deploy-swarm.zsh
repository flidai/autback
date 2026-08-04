#!/bin/zsh

source "${0:A:h}/lib.zsh"

: "${AUTBACK_SERVER_IP:?set AUTBACK_SERVER_IP to an existing host; this script never provisions infrastructure}"
: "${AUTBACK_SSH_KEY:?set AUTBACK_SSH_KEY to the existing host identity file}"
[[ -f "${AUTBACK_SSH_KEY}" ]] || { print -u2 "missing ${AUTBACK_SSH_KEY}"; exit 1; }

host="${AUTBACK_SERVER_IP}"
ssh_user="${AUTBACK_SSH_USER:-root}"
server_names="${AUTBACK_SERVER_NAMES:-${host}}"
public_url="${AUTBACK_PUBLIC_URL:-}"
acme_domain="${AUTBACK_ACME_DOMAIN:-}"
acme_email="${AUTBACK_ACME_EMAIL:-}"
project_slug="${AUTBACK_PROJECT:-default}"
project_name="${AUTBACK_PROJECT_NAME:-Default}"
ssh_args
ssh "${reply[@]}" "${ssh_user}@${host}" 'docker version >/dev/null'

build_dir="${AUTBACK_TMP_DIR}/service-build"
config_dir="${XDG_CONFIG_HOME:-${HOME}/.config}/autback"
install_dir="${AUTBACK_INSTALL_DIR:-${HOME}/.local/bin}"
config_file="${config_dir}/config.json"
ca_file="${config_dir}/ca.pem"
target_url="${public_url:-https://${server_names%%,*}}"
mkdir -p "${build_dir}" "${config_dir}" "${install_dir}"
env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go -C "${AUTBACK_DIR}" build -trimpath -ldflags='-s -w' \
  -o "${build_dir}/autback-server" ./cmd/autback-server
env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go -C "${AUTBACK_DIR}" build -trimpath -ldflags='-s -w' \
  -o "${build_dir}/autback-job-entrypoint" ./cmd/autback-job-entrypoint
go -C "${AUTBACK_DIR}" build -trimpath -o "${build_dir}/autback" ./cmd/autback

migration_token=""
if [[ -f "${config_file}" ]]; then
  previous_url="$(jq -r '.url // empty' "${config_file}")"
  if [[ -n "${previous_url}" && "${previous_url}" != "${target_url}" ]]; then
    migration_token="$(AUTBACK_CONFIG="${config_file}" "${build_dir}/autback" token create --name domain-migration 2>/dev/null)"
    [[ -n "${migration_token}" ]] || { print -u2 'failed to create a credential for the service URL migration'; exit 1; }
  fi
fi

scp "${reply[@]}" \
  "${build_dir}/autback-server" \
  "${build_dir}/autback-job-entrypoint" \
  "${AUTBACK_DIR}/host/install-swarm.sh" \
  "${AUTBACK_DIR}/host/autback-server.service" \
  "${AUTBACK_DIR}/host/autback-cas.service" \
  "${AUTBACK_DIR}/host/autback-buildkit.service" \
  "${AUTBACK_DIR}/host/autback-maintenance.service" \
  "${AUTBACK_DIR}/host/autback-maintenance.timer" \
  "${ssh_user}@${host}:/tmp/"

if [[ -n "${AUTBACK_GITHUB_CLIENT_ID:-}" || -n "${AUTBACK_GITHUB_CLIENT_SECRET:-}" ]]; then
  : "${AUTBACK_GITHUB_CLIENT_ID:?set AUTBACK_GITHUB_CLIENT_ID with AUTBACK_GITHUB_CLIENT_SECRET}"
  : "${AUTBACK_GITHUB_CLIENT_SECRET:?set AUTBACK_GITHUB_CLIENT_SECRET with AUTBACK_GITHUB_CLIENT_ID}"
  [[ "${AUTBACK_GITHUB_CLIENT_ID}${AUTBACK_GITHUB_CLIENT_SECRET}" != *$'\n'* ]] || { print -u2 'GitHub credentials cannot contain newlines'; exit 1; }
  auth_file="$(mktemp)"
  chmod 0600 "${auth_file}"
  trap 'rm -f "${auth_file}"' EXIT
  {
    print -r -- "AUTBACK_GITHUB_CLIENT_ID=${AUTBACK_GITHUB_CLIENT_ID}"
    print -r -- "AUTBACK_GITHUB_CLIENT_SECRET=${AUTBACK_GITHUB_CLIENT_SECRET}"
  } > "${auth_file}"
  scp "${reply[@]}" "${auth_file}" "${ssh_user}@${host}:/tmp/autback-auth.env"
fi

ssh "${reply[@]}" "${ssh_user}@${host}" \
  "sudo -n env AUTBACK_SERVER_NAMES=${(q)server_names} AUTBACK_PUBLIC_URL=${(q)public_url} AUTBACK_ACME_DOMAIN=${(q)acme_domain} AUTBACK_ACME_EMAIL=${(q)acme_email} AUTBACK_BOOTSTRAP_PROJECT=${(q)project_slug} AUTBACK_BOOTSTRAP_PROJECT_NAME=${(q)project_name} bash /tmp/install-swarm.sh"

umask 077
ssh "${reply[@]}" "${ssh_user}@${host}" 'sudo -n cat /var/lib/autback/pki/ca.pem' > "${ca_file}"
chmod 0600 "${ca_file}"
jq -n \
	--arg url "${target_url}" \
  --arg image "${AUTBACK_PROJECT_IMAGE:-}" \
  --arg ca "${ca_file}" \
	'{url:$url,service:{image:$image,ca_cert_file:$ca,oidc_audience:$url}}' \
  > "${config_file}"
chmod 0600 "${config_file}"
install -m 0755 "${build_dir}/autback" "${install_dir}/autback"

if ssh "${reply[@]}" "${ssh_user}@${host}" 'sudo -n test -f /etc/autback/bootstrap-token'; then
  bootstrap_token="$(ssh "${reply[@]}" "${ssh_user}@${host}" 'sudo -n cat /etc/autback/bootstrap-token')"
  AUTBACK_CONFIG="${config_file}" "${build_dir}/autback" login --token "${bootstrap_token}"
  unset bootstrap_token
  ssh "${reply[@]}" "${ssh_user}@${host}" 'sudo -n shred -u /etc/autback/bootstrap-token 2>/dev/null || sudo -n rm -f /etc/autback/bootstrap-token'
fi

if [[ -n "${migration_token}" ]]; then
  AUTBACK_CONFIG="${config_file}" "${build_dir}/autback" login --token "${migration_token}"
  unset migration_token
fi

AUTBACK_CONFIG="${config_file}" "${build_dir}/autback" doctor
print "deployed autback shared service to existing host ${host}"
print "configured ${config_file}; the device token is stored in the OS keychain"
