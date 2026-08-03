import { describe, expect, test } from 'bun:test'

describe('runs table web component contract', () => {
  test('uses TanStack filtering over live Datastar signals', async () => {
    const source = await Bun.file(new URL('./runs-table.ts', import.meta.url)).text()
    for (const contract of [
      'TableController',
      'columnFilteringFeature',
      'globalFilteringFeature',
      'createFilteredRowModel',
    ]) expect(source).toContain(contract)
    for (const signal of ['queue', 'operations', 'clock']) {
      expect(source).toMatch(new RegExp(`this\\.signal(?:<[^>]+>)?\\('${signal}'`))
    }
  })

  test('the console mounts one unified list instead of separate jobs and runs panels', async () => {
    const source = await Bun.file(new URL('./console.ts', import.meta.url)).text()
    expect(source.match(/<autback-runs-table/g)).toHaveLength(2)
    expect(source).not.toContain('jobsPanel(')
    expect(source).not.toContain('runsPanel(')
  })
})
