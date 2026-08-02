import { LitElement, html, nothing, svg, type TemplateResult } from 'lit'
import { DatastarLit } from './datastar-lit'
import { consoleStyles } from './console-styles'
import { duration, relativeTime, shortDigest, shortID, successRate } from './format'
import type {
  AuditView,
  ConsoleSignals,
  OperationDetailView,
  OperationView,
  ProjectView,
  QueueView,
} from './generated/console'

const EMPTY: ConsoleSignals = {
  session: { user: '', admin: false, projects: [] },
  service: { name: 'Autback', version: '', control: 'CLI only', admission: 'Strict FIFO', startedAt: '' },
  worker: { status: 'connecting', capacity: '1 operation', activeId: '', updatedAt: '' },
  queue: [],
  operations: [],
  operation: null,
  log: { available: false, truncated: false, content: '' },
  audit: [],
  status: { ready: false, route: '', message: 'Connecting to SQLite', updatedAt: '' },
}

class AutbackConsole extends DatastarLit(LitElement) {
  static override styles = consoleStyles

  get routeKind(): string { return this.getAttribute('route-kind') || 'overview' }
  get project(): string { return this.getAttribute('project') || '' }
  get operationKind(): string { return this.getAttribute('operation-kind') || '' }
  get operationID(): string { return this.getAttribute('operation-id') || '' }

  override render(): TemplateResult {
    const signals = this.signals()
    return html`
      <div class="shell">
        ${this.sidebar(signals)}
        <section class="workspace">
          ${this.topbar(signals)}
          ${signals.status.ready
            ? html`<main class="content" id="content">${this.page(signals)}</main>`
            : html`<main class="loading" id="content"><div class="loader">Opening live console</div></main>`}
        </section>
      </div>
    `
  }

  private signals(): ConsoleSignals {
    return {
      session: this.signal('session', EMPTY.session),
      service: this.signal('service', EMPTY.service),
      worker: this.signal('worker', EMPTY.worker),
      queue: this.signal('queue', EMPTY.queue),
      operations: this.signal('operations', EMPTY.operations),
      operation: this.signal('operation', EMPTY.operation),
      log: this.signal('log', EMPTY.log),
      audit: this.signal('audit', EMPTY.audit),
      status: this.signal('status', EMPTY.status),
    }
  }

  private sidebar(signals: ConsoleSignals): TemplateResult {
    return html`
      <aside class="sidebar" aria-label="Console navigation">
        <a class="brand" href="/app"><span class="brand-mark">A</span><span>Autback</span></a>
        <nav class="nav-section" aria-label="Primary">
          <div class="nav-label">Console</div>
          ${this.navLink('/app', 'overview', 'Overview', 'activity')}
          ${this.navLink('/app/audit', 'audit', 'Audit log', 'shield')}
        </nav>
        <nav class="nav-section projects-nav" aria-label="Projects">
          <div class="nav-label">Projects</div>
          ${signals.session.projects.map((project) => html`
            <a class="nav-link ${this.routeKind === 'project' && this.project === project.slug ? 'active' : ''}" href=${`/app/projects/${encodeURIComponent(project.slug)}`}>
              ${icon('cube')}<span>${project.name}</span><span class="count">${project.trusts}</span>
            </a>
          `)}
        </nav>
        <div class="sidebar-foot">
          <div class="identity">
            <span class="avatar">${initials(signals.session.user)}</span>
            <div><div class="identity-name">${signals.session.user || 'Connecting'}</div><div class="identity-role">${signals.session.admin ? 'Administrator' : 'Member'}</div></div>
          </div>
        </div>
      </aside>
    `
  }

  private navLink(href: string, route: string, label: string, iconName: IconName): TemplateResult {
    return html`<a class="nav-link ${this.routeKind === route ? 'active' : ''}" href=${href}>${icon(iconName)}<span>${label}</span></a>`
  }

  private topbar(signals: ConsoleSignals): TemplateResult {
    const label = this.routeKind === 'project' ? this.project : this.routeKind === 'operation' ? shortID(this.operationID, 18) : this.routeKind === 'audit' ? 'Audit log' : 'Overview'
    return html`
      <header class="topbar">
        <div class="breadcrumb"><span>Autback</span><span class="slash">/</span><strong>${label}</strong></div>
        <div class="live ${signals.worker.status}" aria-live="polite"><span class="live-dot"></span><span>${signals.status.message}</span></div>
      </header>
    `
  }

  private page(signals: ConsoleSignals): TemplateResult {
    switch (this.routeKind) {
      case 'project': return this.projectPage(signals)
      case 'operation': return this.operationPage(signals)
      case 'audit': return this.auditPage(signals)
      default: return this.overview(signals)
    }
  }

  private overview(signals: ConsoleSignals): TemplateResult {
    const active = signals.operations.filter((operation) => ['running', 'active', 'preparing'].includes(operation.status)).length
    const waiting = signals.queue.filter((item) => item.status === 'queued').length
    return html`
      ${this.pageHead('Shared runner', 'One trusted queue. Every heavy task gets the machine.', 'Live governance for jobs, builds, projects, and trust. All changes remain CLI-only.')}
      <section class="metrics" aria-label="Service metrics">
        ${metric('Worker', signals.worker.status, signals.worker.capacity, 'cpu')}
        ${metric('Active', String(active), active === 1 ? 'operation using the VM' : 'operations using the VM', 'pulse')}
        ${metric('Waiting', String(waiting), waiting === 1 ? 'operation in strict FIFO' : 'operations in strict FIFO', 'queue')}
        ${metric('Success', successRate(signals.operations.map((item) => item.status)), 'recent terminal operations', 'trend')}
      </section>
      <section class="grid">
        ${this.queuePanel(signals.queue)}
        <article class="panel">
          <header class="panel-head"><div class="panel-title">${icon('cpu')}Worker lease</div><span class="badge ${signals.worker.status}">${signals.worker.status}</span></header>
          <div class="worker-orbit">
            <div class="worker-core">${icon('cpu')}</div>
            <div class="worker-label"><strong>${signals.worker.activeId ? shortID(signals.worker.activeId, 16) : 'Available'}</strong><span>${signals.worker.activeId ? 'holds the single lease' : 'next job gets the machine'}</span></div>
          </div>
        </article>
      </section>
      ${this.operationsPanel(signals.operations, 'Recent operations')}
    `
  }

  private projectPage(signals: ConsoleSignals): TemplateResult {
    const project = signals.session.projects.find((item) => item.slug === this.project)
    if (!project) return this.notFound('Project is not available to this device.')
    return html`
      ${this.pageHead('Project', project.name, 'Runner image, trust posture, queue position, and recent remote work.')}
      <section class="project-banner">
        <div><div class="project-name">${project.name}</div><div class="project-slug">${project.slug}</div><p class="digest">${shortDigest(project.activeImage)}</p></div>
        <div class="project-facts">
          <div class="fact"><strong>${project.members}</strong><span>Members</span></div>
          <div class="fact"><strong>${project.trusts}</strong><span>GitHub trusts</span></div>
          <div class="fact"><strong>${project.allowImageOverrides ? 'Allowed' : 'Pinned'}</strong><span>Image policy</span></div>
        </div>
      </section>
      <section class="grid">
        ${this.queuePanel(signals.queue)}
        <article class="panel">
          <header class="panel-head"><div class="panel-title">${icon('terminal')}CLI control</div><span class="panel-meta">read only</span></header>
          <div class="panel-body"><p class="lede">The console reflects durable state. Change this project from a trusted terminal.</p><pre class="command"><span class="prompt">$</span> autback image show --project ${project.slug}</pre></div>
        </article>
      </section>
      ${this.operationsPanel(signals.operations, 'Project operations')}
    `
  }

  private operationPage(signals: ConsoleSignals): TemplateResult {
    const operation = signals.operation
    if (!operation) return this.notFound('Operation is not available to this device.')
    const title = operation.command || `${capitalize(operation.kind)} ${shortID(operation.id, 18)}`
    return html`
      ${this.pageHead(`${capitalize(operation.kind)} operation`, title, `${operation.projectName} · ${shortID(operation.id, 26)}`)}
      <section class="metrics" aria-label="Operation metrics">
        ${metric('Status', operation.status, operation.kind, 'pulse')}
        ${metric('Duration', duration(operation.startedAt, operation.finishedAt), operation.startedAt ? 'wall-clock execution' : 'not started', 'clock')}
        ${metric('Exit code', operation.exitCode == null ? '—' : String(operation.exitCode), operation.finishedAt ? 'process result' : 'pending', 'terminal')}
        ${metric('Created', relativeTime(operation.createdAt), operation.projectName, 'calendar')}
      </section>
      <section class="detail-grid">
        <div class="detail-stack">
          <article class="panel">
            <header class="panel-head"><div class="panel-title">${icon('terminal')}Command</div><span class="badge ${operation.status}">${operation.status}</span></header>
            <div class="panel-body"><pre class="command"><span class="prompt">$</span> ${operation.command || 'docker buildx build'}</pre></div>
          </article>
          ${this.logPanel(signals, operation)}
        </div>
        <div class="detail-stack">
          ${this.provenancePanel(operation)}
          <article class="panel">
            <header class="panel-head"><div class="panel-title">${icon('terminal')}Continue in CLI</div><span class="panel-meta">authoritative</span></header>
            <div class="panel-body"><p class="lede">Stream the complete log or inspect this operation from any enrolled device.</p><pre class="command"><span class="prompt">$</span> autback ${operation.kind === 'job' ? 'logs' : 'build status'} ${operation.id}</pre></div>
          </article>
        </div>
      </section>
    `
  }

  private auditPage(signals: ConsoleSignals): TemplateResult {
    return html`
      ${this.pageHead('Governance', 'Audit log', 'An append-only account of project, trust, token, image, job, and build lifecycle events.')}
      <article class="panel">
        <header class="panel-head"><div class="panel-title">${icon('shield')}Recent events</div><span class="panel-meta">${signals.audit.length} records</span></header>
        ${signals.audit.length === 0 ? emptyState('shield', 'No audit events yet', 'CLI mutations will appear here.') : this.auditTable(signals.audit)}
      </article>
    `
  }

  private pageHead(eyebrow: string, title: string, description: string): TemplateResult {
    return html`
      <header class="page-head">
        <div><p class="eyebrow">${eyebrow}</p><h1>${title}</h1><p class="lede">${description}</p></div>
        <div class="read-only">${icon('eye')}Read-only console · use the CLI to make changes</div>
      </header>
    `
  }

  private queuePanel(queue: QueueView[]): TemplateResult {
    return html`
      <article class="panel">
        <header class="panel-head"><div class="panel-title">${icon('queue')}Strict FIFO queue</div><span class="panel-meta">${queue.length} operations</span></header>
        <div class="queue-list">
          ${queue.length === 0 ? emptyState('queue', 'The queue is clear', 'The next submitted task receives the worker lease.') : queue.map((item) => html`
            <div class="queue-row">
              <span class="position">${item.position}</span>
              <div class="queue-main"><a href=${operationURL(item.kind, item.id)}>${item.id}</a><div class="queue-sub">${item.projectName} · ${relativeTime(item.acceptedAt)}</div></div>
              <span class="badge ${item.status}">${item.status}</span>
            </div>
          `)}
        </div>
      </article>
    `
  }

  private operationsPanel(operations: OperationView[], title: string): TemplateResult {
    return html`
      <article class="panel">
        <header class="panel-head"><div class="panel-title">${icon('activity')}${title}</div><span class="panel-meta">${operations.length} shown</span></header>
        ${operations.length === 0 ? emptyState('activity', 'No operations yet', 'Submit a repository command with autback exec.') : html`
          <div class="table-wrap"><table>
            <thead><tr><th>Operation</th><th>Status</th><th>Project</th><th>Duration</th><th>Created</th></tr></thead>
            <tbody>${operations.map((operation) => html`<tr>
              <td class="primary"><a href=${operationURL(operation.kind, operation.id)}><span class="kind-icon">${icon(operation.kind === 'build' ? 'cube' : 'terminal')}</span><span><span class="mono">${shortID(operation.id, 20)}</span><br><span class="muted">${operation.command || capitalize(operation.kind)}</span></span></a></td>
              <td><span class="badge ${operation.status}">${operation.status}</span></td>
              <td>${operation.projectName}</td>
              <td class="mono">${duration(operation.startedAt, operation.finishedAt)}</td>
              <td>${relativeTime(operation.createdAt)}</td>
            </tr>`)}</tbody>
          </table></div>
        `}
      </article>
    `
  }

  private logPanel(signals: ConsoleSignals, operation: OperationDetailView): TemplateResult {
    return html`
      <article class="panel">
        <header class="panel-head"><div class="panel-title">${icon('terminal')}Log tail</div><span class="panel-meta">${signals.log.available ? 'live projection' : 'not available'}</span></header>
        ${signals.log.available
          ? html`<pre class="log">${signals.log.content || 'Waiting for output…'}</pre>${signals.log.truncated ? html`<div class="log-note">Showing the newest 64 KiB. Use <span class="mono">autback logs ${operation.id}</span> for the complete stream.</div>` : nothing}`
          : emptyState('terminal', 'No log tail available', operation.kind === 'build' ? 'Build progress remains in the invoking terminal.' : 'The worker has not produced output yet.')}
      </article>
    `
  }

  private provenancePanel(operation: OperationDetailView): TemplateResult {
    const cacheSummary = operation.caches?.length ? operation.caches.map((cache) => cache.name).join(', ') : 'None declared'
    return html`
      <article class="panel">
        <header class="panel-head"><div class="panel-title">${icon('fingerprint')}Provenance</div><span class="panel-meta">immutable inputs</span></header>
        <dl class="definition">
          <dt>Operation</dt><dd>${operation.id}</dd>
          <dt>Project</dt><dd>${operation.project}</dd>
          <dt>Image</dt><dd title=${operation.image}>${shortDigest(operation.image)}</dd>
          <dt>Workdir</dt><dd>${operation.workingDirectory || '—'}</dd>
          <dt>Root</dt><dd>${operation.rootDigest || '—'}</dd>
          <dt>Caches</dt><dd>${cacheSummary}</dd>
        </dl>
      </article>
    `
  }

  private auditTable(events: AuditView[]): TemplateResult {
    return html`<div class="table-wrap"><table>
      <thead><tr><th>Event</th><th>Actor</th><th>Project</th><th>Target</th><th>When</th></tr></thead>
      <tbody>${events.map((event) => html`<tr>
        <td><span class="audit-action">${event.action}</span>${metadata(event)}</td>
        <td>${event.actor}</td><td>${event.project || 'Service'}</td><td class="mono">${shortID(event.target, 18)}</td><td>${relativeTime(event.createdAt)}</td>
      </tr>`)}</tbody>
    </table></div>`
  }

  private notFound(message: string): TemplateResult {
    return html`${this.pageHead('Not found', 'Unavailable', message)}<article class="panel">${emptyState('shield', 'Nothing to show', 'Return to the console overview.')}</article>`
  }
}

type IconName = 'activity' | 'calendar' | 'clock' | 'cpu' | 'cube' | 'eye' | 'fingerprint' | 'pulse' | 'queue' | 'shield' | 'terminal' | 'trend'

function icon(name: IconName): TemplateResult {
  const paths: Record<IconName, TemplateResult> = {
    activity: svg`<path d="M3 12h4l2.2-6 4.2 12 2.2-6H21"/>`,
    calendar: svg`<rect x="3" y="5" width="18" height="16" rx="2"/><path d="M16 3v4M8 3v4M3 10h18"/>`,
    clock: svg`<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>`,
    cpu: svg`<rect x="7" y="7" width="10" height="10" rx="2"/><path d="M9 1v3M15 1v3M9 20v3M15 20v3M20 9h3M20 14h3M1 9h3M1 14h3"/>`,
    cube: svg`<path d="m12 3 8 4.5v9L12 21l-8-4.5v-9Z"/><path d="m4.5 7.8 7.5 4.3 7.5-4.3M12 12v9"/>`,
    eye: svg`<path d="M2 12s3.6-6 10-6 10 6 10 6-3.6 6-10 6S2 12 2 12Z"/><circle cx="12" cy="12" r="2.5"/>`,
    fingerprint: svg`<path d="M8 11a4 4 0 0 1 8 0c0 5-1 8-3 10M5 11a7 7 0 0 1 14 0c0 4-.5 7-2 10M11 14c0 3-.5 5-1.5 7M8 15c0 2-.4 3.5-1 5M12 2a9 9 0 0 0-9 9"/>`,
    pulse: svg`<path d="M3 12h4l2-5 4 10 2-5h6"/>`,
    queue: svg`<path d="M9 6h12M9 12h12M9 18h12"/><circle cx="4" cy="6" r="1"/><circle cx="4" cy="12" r="1"/><circle cx="4" cy="18" r="1"/>`,
    shield: svg`<path d="M12 3 20 6v6c0 5-3.4 8-8 10-4.6-2-8-5-8-10V6Z"/><path d="m9 12 2 2 4-5"/>`,
    terminal: svg`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="m7 9 3 3-3 3M13 15h4"/>`,
    trend: svg`<path d="m3 17 6-6 4 4 8-9"/><path d="M15 6h6v6"/>`,
  }
  return svg`<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${paths[name]}</svg>`
}

function metric(label: string, value: string, note: string, iconName: IconName): TemplateResult {
  return html`<article class="metric"><div class="metric-top"><span>${label}</span>${icon(iconName)}</div><div class="metric-value">${capitalize(value)}</div><div class="metric-note">${note}</div></article>`
}

function emptyState(iconName: IconName, title: string, description: string): TemplateResult {
  return html`<div class="empty"><div>${icon(iconName)}<strong>${title}</strong><span>${description}</span></div></div>`
}

function operationURL(kind: string, id: string): string {
  return `/app/operations/${encodeURIComponent(kind)}/${encodeURIComponent(id)}`
}

function metadata(event: AuditView): TemplateResult | typeof nothing {
  const entries = Object.entries(event.metadata ?? {}).slice(0, 3)
  if (entries.length === 0) return nothing
  return html`<div class="metadata">${entries.map(([key, value]) => html`<span>${key}=${shortID(value, 28)}</span>`)}</div>`
}

function capitalize(value: string): string {
  return value ? value[0]!.toUpperCase() + value.slice(1) : '—'
}

function initials(value: string): string {
  return value.split(/\s+/).filter(Boolean).slice(0, 2).map((part) => part[0]?.toUpperCase()).join('') || 'A'
}

customElements.define('autback-console', AutbackConsole)

declare global {
  interface HTMLElementTagNameMap {
    'autback-console': AutbackConsole
  }
}
