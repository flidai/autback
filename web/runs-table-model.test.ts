import { describe, expect, test } from 'bun:test'
import type { OperationView, QueueView } from './generated/console'
import { orderedRunRows } from './runs-table-model'

const resources = { sampleCount: 0, cpuAverage: 0, cpuPeak: 0, memoryAverage: 0, memoryPeak: 0, memoryBytesPeak: 0 }

function operation(id: string, status: string, createdAt: string): OperationView {
  return {
    kind: 'job', id, project: 'autback', projectName: 'Autback', status, command: 'task ci', image: '', createdAt,
    startedAt: undefined, finishedAt: undefined, exitCode: undefined, queueWaitMillis: undefined, resources,
  }
}

describe('unified runs table model', () => {
  test('orders active, queued FIFO, then completed newest first without duplicates', () => {
    const operations = [
      operation('completed_old', 'succeeded', '2026-08-03T07:00:00Z'),
      operation('queued_second', 'queued', '2026-08-03T08:01:00Z'),
      operation('active', 'running', '2026-08-03T08:00:00Z'),
      operation('completed_new', 'failed', '2026-08-03T07:30:00Z'),
    ]
    const queue: QueueView[] = [
      { position: 1, kind: 'job', id: 'active', project: 'autback', projectName: 'Autback', status: 'active', acceptedAt: '2026-08-03T08:00:00Z', leasedAt: '2026-08-03T08:00:05Z' },
      { position: 3, kind: 'job', id: 'queued_second', project: 'autback', projectName: 'Autback', status: 'queued', acceptedAt: '2026-08-03T08:01:00Z', leasedAt: undefined },
      { position: 2, kind: 'build', id: 'queued_first', project: 'autback', projectName: 'Autback', status: 'queued', acceptedAt: '2026-08-03T08:00:30Z', leasedAt: undefined },
    ]

    const rows = orderedRunRows(operations, queue)

    expect(rows.map((row) => row.id)).toEqual(['active', 'queued_first', 'queued_second', 'completed_new', 'completed_old'])
    expect(rows.find((row) => row.id === 'active')?.status).toBe('active')
    expect(rows.filter((row) => row.id === 'queued_second')).toHaveLength(1)
  })
})
