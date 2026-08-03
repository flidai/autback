import { describe, expect, test } from 'bun:test'

describe('console product language', () => {
  test('does not expose implementation vocabulary', async () => {
    const source = await Bun.file(new URL('./console.ts', import.meta.url)).text()

    for (const phrase of [
      'Connecting to SQLite',
      'Live from SQLite',
      'live projection',
      'durable state',
      'worker lease',
      'single lease',
      'immutable inputs',
      'authoritative',
      'wall-clock execution',
      'process result',
      '64 KiB',
			'Strict FIFO',
    ]) {
      expect(source).not.toContain(phrase)
    }
  })
})
