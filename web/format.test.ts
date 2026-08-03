import { describe, expect, test } from 'bun:test'
import { duration, formatBytes, formatMilliseconds, formatPercent, relativeTime, shortDigest, shortID, successRate } from './format'

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

  test('formats capacity and resource values compactly', () => {
		expect(formatBytes(8 * 1024 ** 3)).toBe('8 GB')
		expect(formatBytes(1536 * 1024 ** 2)).toBe('1.5 GB')
		expect(formatPercent(0.754)).toBe('75%')
		expect(formatMilliseconds(65432)).toBe('1m 5s')
	})
})
