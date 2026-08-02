#!/bin/zsh

source "${0:A:h}/lib.zsh"
ensure_ssh_key

host="$(server_ip)"
client_env="${OUTBACK_TMP_DIR}/client.env"
[[ -r "${client_env}" ]] || { print -u2 'run scripts/deploy.zsh first'; exit 1; }
token="$(sed -n 's/^OUTBACK_TOKEN=//p' "${client_env}" | tail -1)"
[[ -n "${token}" ]] || { print -u2 'client environment does not contain OUTBACK_TOKEN'; exit 1; }

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
  --arg token "${token}" \
  --arg host "${host}" \
  --arg identity "${OUTBACK_SSH_KEY:A}" \
  '{token:$token,ssh:{host:$host,user:"root",identity_file:$identity,remote_address:"127.0.0.1:8080"}}' \
  > "${temporary}"
chmod 0600 "${temporary}"
mv "${temporary}" "${config_file}"
trap - EXIT

print "installed ${install_dir}/outback"
print "configured ${config_file}"
if [[ ":${PATH}:" != *":${install_dir}:"* ]]; then
  print -u2 "add ${install_dir} to PATH to invoke outback directly"
fi
