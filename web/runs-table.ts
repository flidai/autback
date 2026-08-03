import { LitElement, html, type TemplateResult } from 'lit'
import {
  TableController,
  columnFilteringFeature,
  createCoreRowModel,
  createFilteredRowModel,
  filterFns,
  globalFilteringFeature,
  tableFeatures,
  type ColumnDef,
  type ColumnFiltersState,
} from '@tanstack/lit-table'
import { DatastarLit } from './datastar-lit'
import { duration, formatBytes, formatPercent, relativeTime, shortID } from './format'
import type { ClockView, OperationView, QueueView } from './generated/console'
import { orderedRunRows, type RunRow } from './runs-table-model'
import { runsTableStyles } from './runs-table-styles'

const features = tableFeatures({
  columnFilteringFeature,
  globalFilteringFeature,
  filteredRowModel: createFilteredRowModel(),
  filterFns,
})

const columns: Array<ColumnDef<typeof features, RunRow, unknown>> = [
  { id: 'search', accessorFn: (row) => [row.id, row.command, row.projectName, row.project, row.status, row.kind].join(' '), enableGlobalFilter: true },
  { id: 'status', accessorFn: (row) => row.status, filterFn: 'equalsString', enableGlobalFilter: false },
  { id: 'kind', accessorFn: (row) => row.kind, filterFn: 'equalsString', enableGlobalFilter: false },
]

const EMPTY_CLOCK: ClockView = { now: '' }

class AutbackRunsTable extends DatastarLit(LitElement) {
  static override styles = runsTableStyles

  private tableController = new TableController<typeof features, RunRow>(this)
  private query = ''
  private statusFilter = ''
  private kindFilter = ''
  private rowsFingerprint = ''
  private rowsCache: RunRow[] = []

  override render(): TemplateResult {
    const operations = this.signal<OperationView[]>('operations', [])
    const queue = this.signal<QueueView[]>('queue', [])
    const clock = this.signal<ClockView>('clock', EMPTY_CLOCK)
    const rows = this.rows(operations, queue)
    const columnFilters: ColumnFiltersState = [
      ...(this.statusFilter ? [{ id: 'status', value: this.statusFilter }] : []),
      ...(this.kindFilter ? [{ id: 'kind', value: this.kindFilter }] : []),
    ]
    const table = this.tableController.table({
      features,
      columns,
      data: rows,
      getCoreRowModel: createCoreRowModel(),
      globalFilterFn: 'includesString',
      getColumnCanGlobalFilter: (column: { id: string }) => column.id === 'search',
      state: { globalFilter: this.query, columnFilters },
    } as any) as any
    const visible: RunRow[] = table.getRowModel().rows.map((row: any) => row.original as RunRow)
    const now = Date.parse(clock.now)
    return html`
      <article class="runs-panel">
        <header class="runs-head">
          <div><strong>Runs</strong><span>${visible.length === rows.length ? `${rows.length} total` : `${visible.length} of ${rows.length}`}</span></div>
          <div class="runs-tools">
            <label class="search"><span class="sr-only">Search runs</span><input type="search" placeholder="Search runs…" .value=${this.query} @input=${this.onSearch}></label>
            <label><span class="sr-only">Filter by status</span><select .value=${this.statusFilter} @change=${this.onStatusFilter}>
              <option value="">All statuses</option><option value="active">Active</option><option value="queued">Queued</option><option value="succeeded">Succeeded</option><option value="failed">Failed</option><option value="cancelled">Cancelled</option>
            </select></label>
            <label><span class="sr-only">Filter by kind</span><select .value=${this.kindFilter} @change=${this.onKindFilter}>
              <option value="">Jobs and builds</option><option value="job">Jobs</option><option value="build">Builds</option>
            </select></label>
          </div>
        </header>
        ${visible.length === 0 ? this.empty(rows.length > 0) : html`
          <div class="table-wrap"><table>
            <thead><tr><th>Run</th><th>Status</th><th>Project</th><th>Duration</th><th>CPU peak</th><th>Memory peak</th><th>Created</th></tr></thead>
            <tbody>${visible.map((run) => this.row(run, now))}</tbody>
          </table></div>
        `}
      </article>
    `
  }

  private rows(operations: OperationView[], queue: QueueView[]): RunRow[] {
    const fingerprint = JSON.stringify([operations, queue])
    if (fingerprint !== this.rowsFingerprint) {
      this.rowsFingerprint = fingerprint
      this.rowsCache = orderedRunRows(operations, queue)
    }
    return this.rowsCache
  }

  private row(run: RunRow, now: number): TemplateResult {
    return html`<tr>
      <td class="primary"><a href=${runURL(run.kind, run.id)}><span class="kind-icon">${run.kind === 'build' ? '◇' : '›_'}</span><span><span class="mono">${shortID(run.id, 22)}</span><br><span class="muted">${run.command || capitalize(run.kind)}</span></span></a></td>
      <td><span class="badge ${run.status}">${run.status}</span>${run.queuePosition != null ? html`<span class="position">${run.queuePosition}</span>` : ''}</td>
      <td>${run.projectName}</td>
      <td class="mono">${duration(run.startedAt, run.finishedAt, now)}</td>
      <td class="mono">${run.resources.sampleCount ? formatPercent(run.resources.cpuPeak) : '—'}</td>
      <td class="mono">${run.resources.sampleCount ? formatBytes(run.resources.memoryBytesPeak) : '—'}</td>
      <td>${relativeTime(run.createdAt, now)}</td>
    </tr>`
  }

  private empty(filtered: boolean): TemplateResult {
    return html`<div class="empty"><strong>${filtered ? 'No matching runs' : 'No runs yet'}</strong><span>${filtered ? 'Try a different search or filter.' : 'Submit a repository command with autback exec.'}</span></div>`
  }

  private onSearch = (event: Event): void => {
    this.query = (event.currentTarget as HTMLInputElement).value
    this.requestUpdate()
  }

  private onStatusFilter = (event: Event): void => {
    this.statusFilter = (event.currentTarget as HTMLSelectElement).value
    this.requestUpdate()
  }

  private onKindFilter = (event: Event): void => {
    this.kindFilter = (event.currentTarget as HTMLSelectElement).value
    this.requestUpdate()
  }
}

function runURL(kind: string, id: string): string { return `/app/runs/${encodeURIComponent(kind)}/${encodeURIComponent(id)}` }
function capitalize(value: string): string { return value ? value[0]!.toUpperCase() + value.slice(1) : '—' }

customElements.define('autback-runs-table', AutbackRunsTable)

declare global { interface HTMLElementTagNameMap { 'autback-runs-table': AutbackRunsTable } }
