#!/bin/zsh

source "${0:A:h}/lib.zsh"

: "${OUTBACK_SERVER_IP:?set OUTBACK_SERVER_IP to the existing host}"
: "${OUTBACK_SSH_KEY:?set OUTBACK_SSH_KEY to the existing host identity file}"

install_dir="${OUTBACK_INSTALL_DIR:-${HOME}/.local/bin}"
config_dir="${XDG_CONFIG_HOME:-${HOME}/.config}/outback"
config_file="${config_dir}/config.json"
mkdir -p "${install_dir}" "${config_dir}"

go -C "${OUTBACK_DIR}" build -trimpath -o "${OUTBACK_TMP_DIR}/outback" ./cmd/outback
install -m 0755 "${OUTBACK_TMP_DIR}/outback" "${install_dir}/outback"

temporary="$(mktemp "${config_dir}/config.XXXXXX")"
trap 'rm -f "${temporary}"' EXIT
umask 077
jq -n \
  --arg host "${OUTBACK_SSH_HOST:-${OUTBACK_SERVER_IP}}" \
  --arg user "${OUTBACK_SSH_USER:-root}" \
  --arg identity "${OUTBACK_SSH_KEY:A}" \
  --arg backend "${OUTBACK_BACKEND:-swarm}" \
  '{
    backend:$backend,
    ssh:{host:$host,user:$user,identity_file:$identity},
    cas:{instance:"outback",remote_address:"127.0.0.1:50051",job_address:"127.0.0.1:50051"},
    swarm:{jobs_root:"/var/lib/outback/jobs",image:"outback-runner-standard:local",cpus:"2.5",memory:"5g"},
    buildkit:{remote_address:"127.0.0.1:1234"}
  }' > "${temporary}"
chmod 0600 "${temporary}"
mv "${temporary}" "${config_file}"
trap - EXIT

print "installed ${install_dir}/outback"
print "configured ${config_file} for CAS, Docker Swarm jobs, and BuildKit over SSH"
