#!/bin/zsh

set -euo pipefail

OUTBACK_DIR="${0:A:h:h}"
OUTBACK_REPO_ROOT="${OUTBACK_DIR:h}"
OUTBACK_INFRA_DIR="${OUTBACK_DIR}/infra"
OUTBACK_TMP_DIR="${OUTBACK_DIR}/.tmp"
OUTBACK_SSH_KEY="${OUTBACK_SSH_KEY:-${HOME}/.ssh/outback-poc}"

mkdir -p "${OUTBACK_TMP_DIR}"

load_secrets() {
  local secrets_file="${HOME}/.zshrc.secrets"
  if [[ ! -r "${secrets_file}" ]]; then
    print -u2 "missing ${secrets_file}"
    return 1
  fi
  source "${secrets_file}"
  : "${HCP_API_TOKEN:?HCP_API_TOKEN is not set}"
  : "${HETZNER_API_KEY:?HETZNER_API_KEY is not set}"
  export TF_TOKEN_app_terraform_io="${HCP_API_TOKEN}"
  export TF_VAR_hcloud_token="${HETZNER_API_KEY}"
}

ensure_ssh_key() {
  if [[ ! -f "${OUTBACK_SSH_KEY}" ]]; then
    ssh-keygen -q -t ed25519 -N '' -C 'leapview-outback-poc' -f "${OUTBACK_SSH_KEY}"
  fi
}

server_ip() {
  if [[ -n "${OUTBACK_SERVER_IP:-}" ]]; then
    print -r -- "${OUTBACK_SERVER_IP}"
    return
  fi
  terraform -chdir="${OUTBACK_INFRA_DIR}" output -raw server_ipv4
}

ssh_args() {
  reply=(-i "${OUTBACK_SSH_KEY}" -o UseKeychain=yes -o AddKeysToAgent=yes -o IdentitiesOnly=yes -o BatchMode=yes -o StrictHostKeyChecking=accept-new -o ConnectTimeout=10)
}

wait_for_job_status() {
  local config_file="$1"
  local binary="$2"
  local job_id="$3"
  local wanted="$4"
  local evidence_file="$5"
  local job_status=""
  local attempt

  for attempt in {1..80}; do
    OUTBACK_CONFIG="${config_file}" "${binary}" status --json "${job_id}" > "${evidence_file}"
    job_status="$(jq -r '.status' "${evidence_file}")"
    if [[ "${job_status}" == "${wanted}" ]]; then
      return 0
    fi
    if [[ "${job_status}" == "succeeded" || "${job_status}" == "failed" || "${job_status}" == "cancelled" || "${job_status}" == "timed_out" ]]; then
      print -u2 "job ${job_id} reached terminal status ${job_status}, wanted ${wanted}"
      return 1
    fi
    sleep 0.25
  done
  print -u2 "job ${job_id} remained ${job_status}, wanted ${wanted}"
  return 1
}
