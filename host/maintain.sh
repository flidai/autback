#!/usr/bin/env bash

set -euo pipefail

service_retention_seconds="${AUTBACK_ORPHAN_RETENTION_SECONDS:-86400}"
job_retention_minutes="${AUTBACK_JOB_RETENTION_MINUTES:-10080}"
cache_high_bytes="${AUTBACK_CACHE_HIGH_BYTES:-12884901888}"
cache_low_bytes="${AUTBACK_CACHE_LOW_BYTES:-8589934592}"
disk_high_percent="${AUTBACK_DISK_HIGH_PERCENT:-85}"

cutoff="$(( $(date +%s) - service_retention_seconds ))"
while IFS= read -r service; do
  [[ -n "$service" ]] || continue
  task_state="$(docker service ps --no-trunc --format '{{.CurrentState}}' "$service" | head -1)"
  case "$task_state" in
    Complete*|Failed*|Rejected*|Shutdown*|Orphaned*) ;;
    *) continue ;;
  esac
  created="$(docker service inspect --format '{{.CreatedAt}}' "$service")"
  if (( $(date -d "$created" +%s) < cutoff )); then
    docker service rm "$service" >/dev/null
  fi
done < <(docker service ls --filter 'label=autback.managed=true' --format '{{.Name}}')

worker_busy=false
while IFS= read -r service; do
  [[ -n "$service" ]] || continue
  task_state="$(docker service ps --no-trunc --format '{{.CurrentState}}' "$service" | head -1)"
  case "$task_state" in
    Running*|Starting*|Preparing*|Assigned*|Accepted*|Ready*|Pending*|New*) worker_busy=true; break ;;
  esac
done < <(docker service ls --filter 'label=autback.managed=true' --format '{{.Name}}')

if [[ "$worker_busy" == false ]]; then
  find /var/lib/autback/jobs -mindepth 1 -maxdepth 1 -type d -mmin "+${job_retention_minutes}" -exec rm -rf -- {} +

  cache_bytes="$(du --summarize --bytes /var/lib/autback/cache | awk '{print $1}')"
  disk_percent="$(df --output=pcent /var/lib/autback | tail -1 | tr -cd '0-9')"
  if (( cache_bytes > cache_high_bytes || disk_percent >= disk_high_percent )); then
    while IFS= read -r -d '' cache_record; do
      cache_path="${cache_record#* }"
      rm -rf -- "$cache_path"
      cache_bytes="$(du --summarize --bytes /var/lib/autback/cache | awk '{print $1}')"
      (( cache_bytes <= cache_low_bytes )) && break
    done < <(find /var/lib/autback/cache -mindepth 2 -maxdepth 2 -type d -printf '%T@ %p\0' | sort --zero-terminated --numeric-sort)
  fi
fi

docker container prune --force --filter 'label=org.testcontainers=true' --filter 'until=24h' >/dev/null
docker volume prune --force >/dev/null
docker image prune --force --filter 'until=24h' >/dev/null
if (( $(df --output=pcent /var/lib/autback | tail -1 | tr -cd '0-9') >= disk_high_percent )); then
  docker exec autback-buildkit buildctl --addr tcp://127.0.0.1:1234 prune \
    --all --keep-storage 4000 >/dev/null
else
  docker exec autback-buildkit buildctl --addr tcp://127.0.0.1:1234 prune \
    --all --keep-storage 10000 >/dev/null
fi
