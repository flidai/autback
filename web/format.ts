export function shortID(value: string, size = 12): string {
  if (value.length <= size) return value
  return `${value.slice(0, size - 1)}…`
}

export function shortDigest(value: string): string {
  if (!value) return 'Not configured'
  const digest = value.includes('@') ? value.split('@').at(-1)! : value
  return digest.length > 23 ? `${digest.slice(0, 16)}…${digest.slice(-6)}` : digest
}

export function relativeTime(value: string, now = Date.now()): string {
  const timestamp = Date.parse(value)
  if (!Number.isFinite(timestamp)) return '—'
  const seconds = Math.max(0, Math.round((now - timestamp) / 1000))
  if (seconds < 5) return 'now'
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}

export function duration(started: string | null | undefined, finished: string | null | undefined, now = Date.now()): string {
  if (!started) return '—'
  const start = Date.parse(started)
  const end = finished ? Date.parse(finished) : now
  if (!Number.isFinite(start) || !Number.isFinite(end) || end < start) return '—'
  const milliseconds = end - start
  if (milliseconds < 1000) return `${milliseconds}ms`
  const seconds = milliseconds / 1000
  if (seconds < 60) return `${seconds.toFixed(seconds < 10 ? 1 : 0)}s`
  const minutes = Math.floor(seconds / 60)
  return `${minutes}m ${Math.floor(seconds % 60)}s`
}

export function successRate(statuses: string[]): string {
  const terminal = statuses.filter((status) => ['succeeded', 'success', 'failed', 'cancelled'].includes(status))
  if (terminal.length === 0) return '—'
  const passed = terminal.filter((status) => status === 'succeeded' || status === 'success').length
  return `${Math.round((passed / terminal.length) * 100)}%`
}

export function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '—'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
  const scaled = value / 1024 ** index
  return `${scaled >= 10 || Number.isInteger(scaled) ? scaled.toFixed(0) : scaled.toFixed(1)} ${units[index]}`
}

export function formatPercent(value: number): string {
  if (!Number.isFinite(value)) return '—'
  return `${Math.round(Math.max(0, Math.min(1, value)) * 100)}%`
}

export function formatMilliseconds(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value) || value < 0) return '—'
  if (value < 1000) return `${Math.round(value)}ms`
  const seconds = value / 1000
  if (seconds < 60) return `${seconds.toFixed(seconds < 10 ? 1 : 0)}s`
  return `${Math.floor(seconds / 60)}m ${Math.floor(seconds % 60)}s`
}
