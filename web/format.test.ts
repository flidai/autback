import { describe, expect, test } from 'bun:test'
import { duration, relativeTime, shortDigest, shortID, successRate } from './format'

describe('console formatting', () => {
  test('keeps operation identifiers recognizable', () => {
    expect(shortID('job_0123456789abcdef')).toBe('job_0123456…')
    expect(shortDigest(`runner@sha256:${'a'.repeat(64)}`)).toBe(`sha256:${'a'.repeat(9)}…${'a'.repeat(6)}`)
  })

  test('formats relative time and elapsed duration without locale drift', () => {
    const now = Date.parse('2026-08-02T18:00:10Z')
    expect(relativeTime('2026-08-02T18:00:00Z', now)).toBe('10s ago')
    expect(duration('2026-08-02T17:59:00Z', '2026-08-02T18:00:05Z', now)).toBe('1m 5s')
  })

  test('computes success only from terminal operations', () => {
    expect(successRate(['succeeded', 'failed', 'running'])).toBe('50%')
    expect(successRate(['queued'])).toBe('—')
  })
})
