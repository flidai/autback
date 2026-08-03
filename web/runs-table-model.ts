import type { OperationView, QueueView } from './generated/console'

export type RunRow = OperationView & {
  queuePosition: number | null
  orderGroup: number
}

const EMPTY_RESOURCES = {
  sampleCount: 0,
  cpuAverage: 0,
  cpuPeak: 0,
  memoryAverage: 0,
  memoryPeak: 0,
  memoryBytesPeak: 0,
}

export function orderedRunRows(operations: OperationView[], queue: QueueView[]): RunRow[] {
  const rows = new Map<string, RunRow>()
  for (const operation of operations) {
    rows.set(runKey(operation.kind, operation.id), {
      ...operation,
      resources: { ...operation.resources },
      queuePosition: null,
      orderGroup: 2,
    })
  }
  for (const item of queue) {
    const key = runKey(item.kind, item.id)
    const operation = rows.get(key)
    rows.set(key, {
      ...(operation ?? queueOperation(item)),
      status: item.status === 'active' ? 'running' : item.status,
      startedAt: item.leasedAt ?? operation?.startedAt,
      queuePosition: item.position,
      orderGroup: item.status === 'active' ? 0 : 1,
    })
  }
  return [...rows.values()].sort(compareRuns)
}

function queueOperation(item: QueueView): OperationView {
  return {
    kind: item.kind,
    id: item.id,
    project: item.project,
    projectName: item.projectName,
    status: item.status,
    command: '',
    image: '',
    createdAt: item.acceptedAt,
    startedAt: item.leasedAt,
    finishedAt: undefined,
    exitCode: undefined,
    queueWaitMillis: undefined,
    resources: { ...EMPTY_RESOURCES },
  }
}

function compareRuns(left: RunRow, right: RunRow): number {
  if (left.orderGroup !== right.orderGroup) return left.orderGroup - right.orderGroup
  if (left.orderGroup < 2) return (left.queuePosition ?? 0) - (right.queuePosition ?? 0)
  const created = Date.parse(right.createdAt) - Date.parse(left.createdAt)
  return created || right.id.localeCompare(left.id)
}

function runKey(kind: string, id: string): string { return `${kind}:${id}` }
