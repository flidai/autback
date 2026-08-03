import { LitElement, html, nothing, svg, type TemplateResult } from 'lit'
import { DatastarLit } from './datastar-lit'
import { consoleStyles } from './console-styles'
import { duration, formatBytes, formatMilliseconds, formatPercent, relativeTime, shortDigest, shortID, successRate } from './format'
import type {
  AuditView,
  ConsoleSignals,
  OperationDetailView,
  OperationView,
  QueueView,
  ResourceSampleView,
  ResourceView,
} from './generated/console'

const EMPTY_RESOURCES: ResourceView = {
  samples: [], sampleCount: 0, activeSampleCount: 0, cpuCores: 0, memoryTotalBytes: 0,
  diskUsageBytes: 0, diskTotalBytes: 0, busyRatio: 0, cpuAverage: 0, cpuPeak: 0,
  memoryAverage: 0, memoryPeak: 0, memoryBytesPeak: 0, queueWaitP95Millis: 0,
}

const EMPTY: ConsoleSignals = {
  session: { user: '', admin: false, projects: [] },
  service: { name: 'Autback', version: '', control: 'CLI only', admission: 'One at a time', startedAt: '' },
  worker: { status: 'connecting', capacity: '1 operation', activeId: '', updatedAt: '' },
  clock: { now: '' },
  resources: EMPTY_RESOURCES,
  queue: [], operations: [], operation: null,
  log: { available: false, truncated: false, content: '' },
  audit: [], status: { ready: false, route: '', message: 'Connecting', updatedAt: '' },
}

class AutbackConsole extends DatastarLit(LitElement) {
  static override styles = consoleStyles

  get routeKind(): string { return this.getAttribute('route-kind') || 'overview' }
  get project(): string { return this.getAttribute('project') || '' }
  get operationID(): string { return this.getAttribute('operation-id') || '' }

  override render(): TemplateResult {
    const signals = this.signals()
    return html`<div class="shell">
      ${this.sidebar(signals)}
      <section class="workspace">
        ${this.topbar(signals)}
        ${signals.status.ready
          ? html`<main class="content" id="content">${this.page(signals)}</main>`
          : html`<main class="loading" id="content"><div class="loader">Opening console</div></main>`}
      </section>
    </div>`
  }

  private signals(): ConsoleSignals {
    return {
      session: this.signal('session', EMPTY.session), service: this.signal('service', EMPTY.service),
      worker: this.signal('worker', EMPTY.worker), resources: this.signal('resources', EMPTY.resources),
      clock: this.signal('clock', EMPTY.clock),
      queue: this.signal('queue', EMPTY.queue), operations: this.signal('operations', EMPTY.operations),
      operation: this.signal('operation', EMPTY.operation), log: this.signal('log', EMPTY.log),
      audit: this.signal('audit', EMPTY.audit), status: this.signal('status', EMPTY.status),
    }
  }

  private sidebar(signals: ConsoleSignals): TemplateResult {
    return html`<aside class="sidebar" aria-label="Console navigation">
      <a class="brand" href="/app"><span class="brand-mark">A</span><span>Autback</span></a>
      <nav class="nav-section" aria-label="Primary">
        <div class="nav-label">Console</div>
        ${this.navLink('/app', 'overview', 'Runs', 'activity')}
        ${this.navLink('/app/audit', 'audit', 'Audit log', 'shield')}
      </nav>
      <nav class="nav-section projects-nav" aria-label="Projects">
        <div class="nav-label">Projects</div>
        ${signals.session.projects.map((project) => html`<a class="nav-link ${this.routeKind === 'project' && this.project === project.slug ? 'active' : ''}" href=${`/app/projects/${encodeURIComponent(project.slug)}`}>
          ${icon('cube')}<span>${project.name}</span><span class="count">${project.trusts}</span>
        </a>`)}
      </nav>
      <div class="sidebar-foot"><div class="identity"><span class="avatar">${initials(signals.session.user)}</span><div>
        <div class="identity-name">${signals.session.user || 'Connecting'}</div><div class="identity-role">${signals.session.admin ? 'Administrator' : 'Member'}</div>
      </div></div></div>
    </aside>`
  }

  private navLink(href: string, route: string, label: string, iconName: IconName): TemplateResult {
    return html`<a class="nav-link ${this.routeKind === route ? 'active' : ''}" href=${href}>${icon(iconName)}<span>${label}</span></a>`
  }

  private topbar(signals: ConsoleSignals): TemplateResult {
    const label = this.routeKind === 'project' ? this.project : this.routeKind === 'operation' ? shortID(this.operationID, 18) : this.routeKind === 'audit' ? 'Audit log' : 'Runs'
    return html`<header class="topbar">
      <div class="breadcrumb"><span>Autback</span><span class="slash">/</span><strong>${label}</strong></div>
      <div class="live ${signals.worker.status}" aria-live="polite"><span class="live-dot"></span><span>${signals.status.message}</span></div>
    </header>`
  }

  private page(signals: ConsoleSignals): TemplateResult {
    const now = Date.parse(signals.clock.now)
    switch (this.routeKind) {
      case 'project': return this.projectPage(signals, now)
      case 'operation': return this.runPage(signals, now)
      case 'audit': return this.auditPage(signals, now)
      default: return this.overview(signals, now)
    }
  }

  private overview(signals: ConsoleSignals, now: number): TemplateResult {
    return html`
      ${this.pageHead('Shared runner', 'Runs and capacity', 'See what is running, what is next, and whether the machine has room to do more.')}
      ${this.resourceChart(signals.resources, 'Runner utilization')}
      ${this.resourceMetrics(signals.resources)}
      <section class="grid">
        ${this.jobsPanel(signals.queue, now)}
        ${this.runnerPanel(signals)}
      </section>
      ${this.runsPanel(signals.operations, 'Recent runs', now)}
    `
  }

  private projectPage(signals: ConsoleSignals, now: number): TemplateResult {
    const project = signals.session.projects.find((item) => item.slug === this.project)
    if (!project) return this.notFound('You do not have access to this project.')
    return html`
      ${this.pageHead('Project', project.name, 'Runs, demand, and runner use for this project.')}
      <section class="project-banner">
        <div><div class="project-name">${project.name}</div><div class="project-slug">${project.slug}</div><p class="digest">${shortDigest(project.activeImage)}</p></div>
        <div class="project-facts">
          <div class="fact"><strong>${project.members}</strong><span>Members</span></div>
          <div class="fact"><strong>${project.trusts}</strong><span>GitHub trusts</span></div>
          <div class="fact"><strong>${project.allowImageOverrides ? 'Flexible' : 'Pinned'}</strong><span>Runner image</span></div>
        </div>
      </section>
      ${this.projectTrends(signals.operations)}
      ${this.resourceChart(signals.resources, 'Resource utilization')}
      <section class="grid">${this.jobsPanel(signals.queue, now)}${this.runnerPanel(signals)}</section>
      ${this.runsPanel(signals.operations, 'Project runs', now)}
    `
  }

  private runPage(signals: ConsoleSignals, now: number): TemplateResult {
    const run = signals.operation
    if (!run) return this.notFound('You do not have access to this run.')
    const title = run.command || `${capitalize(run.kind)} ${shortID(run.id, 18)}`
    return html`
      ${this.pageHead(`${capitalize(run.kind)} run`, title, `${run.projectName} · ${shortID(run.id, 26)}`)}
      ${this.resourceChart(signals.resources, 'Resource utilization')}
      <section class="metrics" aria-label="Run summary">
        ${metric('Status', run.status, capitalize(run.kind), 'pulse')}
        ${metric('Queue wait', formatMilliseconds(run.queueWaitMillis), 'before starting', 'queue')}
        ${metric('Duration', duration(run.startedAt, run.finishedAt, now), run.startedAt ? 'elapsed time' : 'not started', 'clock')}
        ${metric('Exit code', run.exitCode == null ? '—' : String(run.exitCode), run.finishedAt ? 'result' : 'pending', 'terminal')}
      </section>
      <section class="detail-grid">
        <div class="detail-stack">
          <article class="panel"><header class="panel-head"><div class="panel-title">${icon('terminal')}Command</div><span class="badge ${run.status}">${run.status}</span></header>
            <div class="panel-body"><pre class="command"><span class="prompt">$</span> ${run.command || 'docker buildx build'}</pre></div>
          </article>
          ${this.logPanel(signals, run)}
        </div>
        <div class="detail-stack">${this.runSummaryPanel(run, now)}${this.provenancePanel(run)}
          <article class="panel"><header class="panel-head"><div class="panel-title">${icon('terminal')}Continue in CLI</div><span class="panel-meta">CLI</span></header>
            <div class="panel-body"><p class="lede">View the full log or inspect this run from your terminal.</p><pre class="command"><span class="prompt">$</span> autback ${run.kind === 'job' ? 'logs' : 'build status'} ${run.id}</pre></div>
          </article>
        </div>
      </section>
    `
  }

  private resourceMetrics(resources: ResourceView): TemplateResult {
    return html`<section class="metrics" aria-label="Runner capacity summary">
      ${metric('Busy', formatPercent(resources.busyRatio), 'of the selected hour', 'pulse')}
      ${metric('CPU while active', formatPercent(resources.cpuAverage), `${formatPercent(resources.cpuPeak)} peak`, 'cpu')}
      ${metric('Memory while active', formatPercent(resources.memoryAverage), `${formatBytes(resources.memoryBytesPeak)} peak`, 'memory')}
      ${metric('Queue wait p95', formatMilliseconds(resources.queueWaitP95Millis), 'recent runs', 'queue')}
    </section>`
  }

  private resourceChart(resources: ResourceView, title: string): TemplateResult {
    const cpu = chartPoints(resources.samples, (sample) => sample.cpuUtilization)
    const memory = chartPoints(resources.samples, (sample) => sample.memoryUtilization)
    const first = resources.samples.at(0)
    const last = resources.samples.at(-1)
    return html`<article class="panel resource-panel">
      <header class="panel-head"><div class="panel-title">${icon('activity')}${title}</div>
        <span class="panel-meta">${capacity(resources)}</span></header>
      ${resources.samples.length < 2 ? emptyState('activity', 'Collecting runner data', 'Utilization will appear after the next samples arrive.') : html`
        <div class="chart-legend">
          <span class="legend cpu"><i></i>CPU <strong>${formatPercent(resources.cpuAverage)} avg · ${formatPercent(resources.cpuPeak)} peak</strong></span>
          <span class="legend memory"><i></i>Memory <strong>${formatPercent(resources.memoryAverage)} avg · ${formatPercent(resources.memoryPeak)} peak</strong></span>
        </div>
        <div class="resource-chart">
          <svg viewBox="0 0 900 230" preserveAspectRatio="none" role="img" aria-label="CPU and memory utilization over time">
            ${[0, .25, .5, .75, 1].map((value) => svg`<line class="grid-line" x1="42" y1=${chartY(value)} x2="892" y2=${chartY(value)}></line><text class="axis-label" x="4" y=${chartY(value) + 4}>${Math.round(value * 100)}%</text>`)}
            <polyline class="series memory" points=${memory}></polyline>
            <polyline class="series cpu" points=${cpu}></polyline>
          </svg>
          <div class="chart-times"><span>${clockTime(first?.observedAt)}</span><span>${clockTime(last?.observedAt)}</span></div>
        </div>
      `}
    </article>`
  }

  private projectTrends(operations: OperationView[]): TemplateResult {
    const finished = operations.filter((item) => item.startedAt && item.finishedAt).slice(0, 20).reverse()
    const values = finished.map((item) => Date.parse(item.finishedAt!) - Date.parse(item.startedAt!))
    const maximum = Math.max(...values, 1)
    return html`<section class="trend-grid">
      <article class="panel trend-panel"><header class="panel-head"><div class="panel-title">${icon('clock')}Run duration</div><span class="panel-meta">Last ${finished.length}</span></header>
        <div class="duration-bars">${values.length === 0 ? emptyState('clock', 'No completed runs', 'Duration history will appear here.') : values.map((value) => html`<i style=${`height:${Math.max(5, value / maximum * 100)}%`} title=${formatMilliseconds(value)}></i>`)}</div>
      </article>
      <article class="panel project-health"><div><span>Success rate</span><strong>${successRate(operations.map((item) => item.status))}</strong></div><div><span>Queue wait p95</span><strong>${formatMilliseconds(percentile(operations.map((item) => item.queueWaitMillis)))}</strong></div></article>
    </section>`
  }

  private jobsPanel(queue: QueueView[], now: number): TemplateResult {
    return html`<article class="panel">
      <header class="panel-head"><div class="panel-title">${icon('queue')}Jobs</div><span class="panel-meta">${queue.length}</span></header>
      <div class="queue-list">${queue.length === 0 ? emptyState('queue', 'No jobs queued or active', 'The next submitted job can start immediately.') : queue.map((item) => html`
        <div class="queue-row"><span class="position">${item.position}</span>
          <div class="queue-main"><a href=${runURL(item.kind, item.id)}>${shortID(item.id, 24)}</a><div class="queue-sub">${item.projectName} · ${relativeTime(item.acceptedAt, now)}</div></div>
          <span class="badge ${item.status}">${item.status}</span>
        </div>`)}
      </div>
    </article>`
  }

  private runnerPanel(signals: ConsoleSignals): TemplateResult {
    const resources = signals.resources
    return html`<article class="panel runner-panel"><header class="panel-head"><div class="panel-title">${icon('cpu')}Runner</div><span class="badge ${signals.worker.status}">${signals.worker.status}</span></header>
      <div class="runner-capacity"><div><strong>${resources.cpuCores || '—'}</strong><span>vCPU</span></div><div><strong>${formatBytes(resources.memoryTotalBytes)}</strong><span>Memory</span></div><div><strong>${formatBytes(resources.diskTotalBytes)}</strong><span>Disk</span></div></div>
      <div class="runner-now"><span class="live-dot"></span><div><strong>${signals.worker.activeId ? shortID(signals.worker.activeId, 22) : 'Ready'}</strong><span>${signals.worker.activeId ? 'active now' : 'waiting for work'}</span></div></div>
    </article>`
  }

  private runsPanel(operations: OperationView[], title: string, now: number): TemplateResult {
    return html`<article class="panel"><header class="panel-head"><div class="panel-title">${icon('activity')}${title}</div><span class="panel-meta">${operations.length} shown</span></header>
      ${operations.length === 0 ? emptyState('activity', 'No runs yet', 'Submit a repository command with autback exec.') : html`<div class="table-wrap"><table>
        <thead><tr><th>Run</th><th>Status</th><th>Project</th><th>Duration</th><th>CPU peak</th><th>Memory peak</th><th>Created</th></tr></thead>
        <tbody>${operations.map((run) => html`<tr>
          <td class="primary"><a href=${runURL(run.kind, run.id)}><span class="kind-icon">${icon(run.kind === 'build' ? 'cube' : 'terminal')}</span><span><span class="mono">${shortID(run.id, 20)}</span><br><span class="muted">${run.command || capitalize(run.kind)}</span></span></a></td>
          <td><span class="badge ${run.status}">${run.status}</span></td><td>${run.projectName}</td>
          <td class="mono">${duration(run.startedAt, run.finishedAt, now)}</td>
          <td class="mono">${run.resources?.sampleCount ? formatPercent(run.resources.cpuPeak) : '—'}</td>
          <td class="mono">${run.resources?.sampleCount ? formatBytes(run.resources.memoryBytesPeak) : '—'}</td>
          <td>${relativeTime(run.createdAt, now)}</td>
        </tr>`)}</tbody>
      </table></div>`}
    </article>`
  }

  private runSummaryPanel(run: OperationDetailView, now: number): TemplateResult {
    return html`<article class="panel"><header class="panel-head"><div class="panel-title">${icon('activity')}Run summary</div><span class="panel-meta">${run.resources.sampleCount} samples</span></header>
      <dl class="definition"><dt>Started</dt><dd>${run.startedAt ? relativeTime(run.startedAt, now) : '—'}</dd><dt>CPU peak</dt><dd>${formatPercent(run.resources.cpuPeak)}</dd><dt>Memory peak</dt><dd>${formatBytes(run.resources.memoryBytesPeak)}</dd><dt>Queue wait</dt><dd>${formatMilliseconds(run.queueWaitMillis)}</dd></dl>
    </article>`
  }

  private logPanel(signals: ConsoleSignals, run: OperationDetailView): TemplateResult {
    return html`<article class="panel"><header class="panel-head"><div class="panel-title">${icon('terminal')}Output</div><span class="panel-meta">${signals.log.available ? 'Following' : 'Unavailable'}</span></header>
      ${signals.log.available ? html`<pre class="log">${signals.log.content || 'Waiting for output…'}</pre>${signals.log.truncated ? html`<div class="log-note">Showing the latest output. Use <span class="mono">autback logs ${run.id}</span> for the full log.</div>` : nothing}` : emptyState('terminal', 'No output available', run.kind === 'build' ? 'Build progress remains in the invoking terminal.' : 'The runner has not produced output yet.')}
    </article>`
  }

  private provenancePanel(run: OperationDetailView): TemplateResult {
    const caches = run.caches?.length ? run.caches.map((cache) => cache.name).join(', ') : 'None declared'
    return html`<article class="panel"><header class="panel-head"><div class="panel-title">${icon('fingerprint')}Provenance</div><span class="panel-meta">Inputs</span></header>
      <dl class="definition"><dt>Run</dt><dd>${run.id}</dd><dt>Project</dt><dd>${run.project}</dd><dt>Image</dt><dd title=${run.image}>${shortDigest(run.image)}</dd><dt>Workdir</dt><dd>${run.workingDirectory || '—'}</dd><dt>Root</dt><dd>${run.rootDigest || '—'}</dd><dt>Caches</dt><dd>${caches}</dd></dl>
    </article>`
  }

  private auditPage(signals: ConsoleSignals, now: number): TemplateResult {
    return html`${this.pageHead('Governance', 'Audit log', 'Project, access, image, job, and build activity across Autback.')}
      <article class="panel"><header class="panel-head"><div class="panel-title">${icon('shield')}Recent events</div><span class="panel-meta">${signals.audit.length} records</span></header>
      ${signals.audit.length === 0 ? emptyState('shield', 'No audit events yet', 'Changes made with the Autback CLI will appear here.') : this.auditTable(signals.audit, now)}</article>`
  }

  private auditTable(events: AuditView[], now: number): TemplateResult {
    return html`<div class="table-wrap"><table><thead><tr><th>Event</th><th>Actor</th><th>Project</th><th>Target</th><th>When</th></tr></thead>
      <tbody>${events.map((event) => html`<tr><td><span class="audit-action">${event.action}</span>${metadata(event)}</td><td>${event.actor}</td><td>${event.project || 'Service'}</td><td class="mono">${shortID(event.target, 18)}</td><td>${relativeTime(event.createdAt, now)}</td></tr>`)}</tbody>
    </table></div>`
  }

  private pageHead(eyebrow: string, title: string, description: string): TemplateResult {
    return html`<header class="page-head"><div><p class="eyebrow">${eyebrow}</p><h1>${title}</h1><p class="lede">${description}</p></div><div class="read-only">${icon('eye')}CLI-managed</div></header>`
  }

  private notFound(message: string): TemplateResult {
    return html`${this.pageHead('Not found', 'Unavailable', message)}<article class="panel">${emptyState('shield', 'Nothing to show', 'Return to the console overview.')}</article>`
  }
}

type IconName = 'activity' | 'clock' | 'cpu' | 'cube' | 'disk' | 'eye' | 'fingerprint' | 'memory' | 'pulse' | 'queue' | 'shield' | 'terminal' | 'trend'

function icon(name: IconName): TemplateResult {
  const paths: Record<IconName, TemplateResult> = {
    activity: svg`<path d="M3 12h4l2.2-6 4.2 12 2.2-6H21"/>`, clock: svg`<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>`,
    cpu: svg`<rect x="7" y="7" width="10" height="10" rx="2"/><path d="M9 1v3M15 1v3M9 20v3M15 20v3M20 9h3M20 14h3M1 9h3M1 14h3"/>`,
    cube: svg`<path d="m12 3 8 4.5v9L12 21l-8-4.5v-9Z"/><path d="m4.5 7.8 7.5 4.3 7.5-4.3M12 12v9"/>`, disk: svg`<ellipse cx="12" cy="5" rx="8" ry="3"/><path d="M4 5v7c0 1.7 3.6 3 8 3s8-1.3 8-3V5M4 12v7c0 1.7 3.6 3 8 3s8-1.3 8-3v-7"/>`,
    eye: svg`<path d="M2 12s3.6-6 10-6 10 6 10 6-3.6 6-10 6S2 12 2 12Z"/><circle cx="12" cy="12" r="2.5"/>`,
    fingerprint: svg`<path d="M8 11a4 4 0 0 1 8 0c0 5-1 8-3 10M5 11a7 7 0 0 1 14 0c0 4-.5 7-2 10M11 14c0 3-.5 5-1.5 7M8 15c0 2-.4 3.5-1 5M12 2a9 9 0 0 0-9 9"/>`,
    memory: svg`<rect x="5" y="7" width="14" height="10" rx="2"/><path d="M8 3v4M12 3v4M16 3v4M8 17v4M12 17v4M16 17v4M9 11h6"/>`,
    pulse: svg`<path d="M3 12h4l2-5 4 10 2-5h6"/>`, queue: svg`<path d="M9 6h12M9 12h12M9 18h12"/><circle cx="4" cy="6" r="1"/><circle cx="4" cy="12" r="1"/><circle cx="4" cy="18" r="1"/>`,
    shield: svg`<path d="M12 3 20 6v6c0 5-3.4 8-8 10-4.6-2-8-5-8-10V6Z"/><path d="m9 12 2 2 4-5"/>`, terminal: svg`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="m7 9 3 3-3 3M13 15h4"/>`, trend: svg`<path d="m3 17 6-6 4 4 8-9"/><path d="M15 6h6v6"/>`,
  }
  return svg`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${paths[name]}</svg>`
}

function metric(label: string, value: string, note: string, iconName: IconName): TemplateResult {
  return html`<article class="metric"><div class="metric-top"><span>${label}</span>${icon(iconName)}</div><div class="metric-value">${capitalize(value)}</div><div class="metric-note">${note}</div></article>`
}

function emptyState(iconName: IconName, title: string, description: string): TemplateResult {
  return html`<div class="empty"><div>${icon(iconName)}<strong>${title}</strong><span>${description}</span></div></div>`
}

function runURL(kind: string, id: string): string { return `/app/runs/${encodeURIComponent(kind)}/${encodeURIComponent(id)}` }
function capacity(resources: ResourceView): string { return resources.cpuCores ? `${resources.cpuCores} vCPU · ${formatBytes(resources.memoryTotalBytes)} · ${formatBytes(resources.diskTotalBytes)} disk` : 'Waiting for capacity data' }
function chartY(value: number): number { return 216 - Math.max(0, Math.min(1, value)) * 196 }
function chartPoints(samples: ResourceSampleView[], value: (sample: ResourceSampleView) => number): string {
  if (samples.length === 0) return ''
  return samples.map((sample, index) => `${42 + (index / Math.max(1, samples.length - 1)) * 850},${chartY(value(sample))}`).join(' ')
}
function clockTime(value: string | undefined): string {
  if (!value) return '—'
  return new Intl.DateTimeFormat(undefined, { hour: '2-digit', minute: '2-digit' }).format(new Date(value))
}
function percentile(values: Array<number | null | undefined>): number | undefined {
  const sorted = values.filter((value): value is number => value != null && Number.isFinite(value)).sort((a, b) => a - b)
  return sorted.length ? sorted[Math.ceil(sorted.length * .95) - 1] : undefined
}
function metadata(event: AuditView): TemplateResult | typeof nothing {
  const entries = Object.entries(event.metadata ?? {}).slice(0, 3)
  return entries.length === 0 ? nothing : html`<div class="metadata">${entries.map(([key, value]) => html`<span>${key}=${shortID(value, 28)}</span>`)}</div>`
}
function capitalize(value: string): string { return value ? value[0]!.toUpperCase() + value.slice(1) : '—' }
function initials(value: string): string { return value.split(/\s+/).filter(Boolean).slice(0, 2).map((part) => part[0]?.toUpperCase()).join('') || 'A' }

customElements.define('autback-console', AutbackConsole)

declare global { interface HTMLElementTagNameMap { 'autback-console': AutbackConsole } }
