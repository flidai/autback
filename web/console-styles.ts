import { css } from 'lit'

export const consoleStyles = css`
  :host {
    color-scheme: dark;
    --canvas: #0b0b0d;
    --canvas-soft: #101013;
    --surface: #141417;
    --surface-raised: #19191d;
    --surface-hover: #1e1e23;
    --line: rgba(255, 255, 255, 0.09);
    --line-strong: rgba(255, 255, 255, 0.15);
    --text: #f5f3ed;
    --text-soft: #aaa7af;
    --text-faint: #74727a;
    --ember: #e38242;
    --ember-soft: rgba(227, 130, 66, 0.12);
    --violet: #a99af8;
    --violet-soft: rgba(169, 154, 248, 0.12);
    --green: #70d6a2;
    --green-soft: rgba(112, 214, 162, 0.11);
    --red: #f08282;
    --red-soft: rgba(240, 130, 130, 0.11);
    --yellow: #e7c66d;
    --yellow-soft: rgba(231, 198, 109, 0.11);
    --blue: #7bb8f0;
    --blue-soft: rgba(123, 184, 240, 0.11);
    --sans: Inter, ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    --mono: "SFMono-Regular", "Cascadia Code", "Roboto Mono", Consolas, monospace;
    display: block;
    min-height: 100vh;
    background: var(--canvas);
    color: var(--text);
    font-family: var(--sans);
    font-size: 14px;
    line-height: 1.5;
    -webkit-font-smoothing: antialiased;
  }

  * { box-sizing: border-box; }
  a { color: inherit; }
  a:focus-visible { outline: 2px solid var(--ember); outline-offset: 3px; }

  .shell {
    min-height: 100vh;
    display: grid;
    grid-template-columns: 244px minmax(0, 1fr);
    background:
      radial-gradient(circle at 70% -20%, rgba(143, 91, 59, 0.08), transparent 35%),
      var(--canvas);
  }

  .sidebar {
    position: sticky;
    top: 0;
    height: 100vh;
    display: flex;
    flex-direction: column;
    padding: 18px 12px 14px;
    border-right: 1px solid var(--line);
    background: rgba(12, 12, 14, 0.94);
    backdrop-filter: blur(18px);
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 10px;
    height: 42px;
    padding: 0 10px;
    color: var(--text);
    font-size: 14px;
    font-weight: 680;
    letter-spacing: -0.02em;
    text-decoration: none;
  }

  .brand-mark {
    display: grid;
    width: 27px;
    height: 27px;
    place-items: center;
    border: 1px solid rgba(255, 255, 255, 0.22);
    border-radius: 50%;
    background: var(--text);
    color: #111114;
    font-family: Georgia, serif;
    font-size: 13px;
    font-weight: 800;
  }

  .nav-section { margin-top: 22px; }
  .nav-label {
    padding: 0 11px 7px;
    color: var(--text-faint);
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.09em;
    text-transform: uppercase;
  }

  .nav-link {
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 36px;
    padding: 0 10px;
    border-radius: 7px;
    color: var(--text-soft);
    font-size: 13px;
    text-decoration: none;
    transition: 140ms ease;
  }
  .nav-link:hover { background: var(--surface-hover); color: var(--text); }
  .nav-link.active { background: var(--surface-raised); color: var(--text); box-shadow: inset 0 0 0 1px var(--line); }
  .nav-link svg { width: 15px; height: 15px; color: var(--text-faint); }
  .nav-link.active svg { color: var(--ember); }
  .nav-link .count { margin-left: auto; color: var(--text-faint); font: 10px var(--mono); }

  .sidebar-foot {
    margin-top: auto;
    padding: 12px 10px 4px;
    border-top: 1px solid var(--line);
  }
  .identity { display: flex; align-items: center; gap: 10px; }
  .avatar {
    display: grid;
    width: 28px;
    height: 28px;
    place-items: center;
    border: 1px solid var(--line-strong);
    border-radius: 50%;
    background: linear-gradient(145deg, #30241f, #19191d);
    color: #f0b486;
    font-size: 11px;
    font-weight: 750;
  }
  .identity-name { color: var(--text); font-size: 12px; font-weight: 600; }
  .identity-role { color: var(--text-faint); font-size: 10px; }

  .workspace { min-width: 0; }
  .topbar {
    position: sticky;
    z-index: 5;
    top: 0;
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 60px;
    padding: 0 clamp(22px, 4vw, 54px);
    border-bottom: 1px solid var(--line);
    background: rgba(11, 11, 13, 0.82);
    backdrop-filter: blur(18px);
  }
  .breadcrumb { display: flex; align-items: center; gap: 8px; min-width: 0; color: var(--text-faint); font-size: 12px; }
  .breadcrumb strong { overflow: hidden; color: var(--text-soft); font-weight: 550; text-overflow: ellipsis; white-space: nowrap; }
  .slash { color: #45434a; }
  .live { display: inline-flex; align-items: center; gap: 7px; color: var(--text-soft); font-size: 11px; }
  .live-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--green); box-shadow: 0 0 0 4px var(--green-soft); }
  .live.degraded .live-dot { background: var(--yellow); box-shadow: 0 0 0 4px var(--yellow-soft); }

  .content { width: min(100%, 1440px); margin: 0 auto; padding: 38px clamp(22px, 4vw, 54px) 72px; }
  .page-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 24px; margin-bottom: 28px; }
  .eyebrow { margin: 0 0 7px; color: var(--text-faint); font-size: 10px; font-weight: 750; letter-spacing: 0.1em; text-transform: uppercase; }
  h1 { margin: 0; font-size: clamp(25px, 3vw, 34px); font-weight: 620; letter-spacing: -0.045em; line-height: 1.1; }
  .lede { max-width: 600px; margin: 10px 0 0; color: var(--text-soft); font-size: 13px; }
  .read-only {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 7px 10px;
    border: 1px solid var(--line);
    border-radius: 7px;
    background: var(--surface);
    color: var(--text-soft);
    font-size: 11px;
    white-space: nowrap;
  }
  .read-only svg { width: 13px; color: var(--text-faint); }

  .metrics { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; margin-bottom: 18px; }
  .metric, .panel {
    border: 1px solid var(--line);
    background: linear-gradient(180deg, rgba(255,255,255,0.018), transparent), var(--surface);
    box-shadow: 0 16px 42px rgba(0, 0, 0, 0.12);
  }
  .metric { min-height: 112px; padding: 17px 18px; border-radius: 10px; }
  .metric-top { display: flex; align-items: center; justify-content: space-between; color: var(--text-faint); font-size: 11px; }
  .metric-top svg { width: 14px; height: 14px; }
  .metric-value { margin-top: 18px; font-size: 24px; font-weight: 570; letter-spacing: -0.04em; }
  .metric-note { margin-top: 2px; color: var(--text-faint); font-size: 10px; }

  .panel { overflow: hidden; border-radius: 11px; }
  .panel-head { display: flex; min-height: 53px; align-items: center; justify-content: space-between; gap: 16px; padding: 0 18px; border-bottom: 1px solid var(--line); }
  .panel-title { display: flex; align-items: center; gap: 9px; font-size: 12px; font-weight: 620; }
  .panel-title svg { width: 14px; height: 14px; color: var(--text-faint); }
  .panel-meta { color: var(--text-faint); font: 10px var(--mono); }
  .panel-body { padding: 18px; }

  .resource-panel { margin-bottom: 14px; }
  .chart-legend { display: flex; justify-content: flex-end; gap: 22px; padding: 14px 18px 0; color: var(--text-faint); font-size: 10px; }
  .legend { display: inline-flex; align-items: center; gap: 7px; }
  .legend i { width: 7px; height: 7px; border-radius: 50%; background: currentColor; }
  .legend strong { color: var(--text-soft); font-weight: 530; }
  .legend.cpu { color: var(--violet); }
  .legend.memory { color: var(--green); }
  .resource-chart { padding: 3px 18px 14px; }
  .resource-chart svg { display: block; width: 100%; height: 230px; overflow: visible; }
  .grid-line { stroke: rgba(255,255,255,.075); stroke-width: 1; vector-effect: non-scaling-stroke; }
  .axis-label { fill: var(--text-faint); font: 9px var(--mono); }
  .series { fill: none; stroke-width: 1.7; vector-effect: non-scaling-stroke; }
  .series.cpu { stroke: var(--violet); }
  .series.memory { stroke: var(--green); opacity: .9; }
  .chart-times { display: flex; justify-content: space-between; padding-left: 42px; color: var(--text-faint); font: 9px var(--mono); }

  .badge { display: inline-flex; align-items: center; gap: 6px; padding: 3px 7px; border: 1px solid var(--line); border-radius: 999px; color: var(--text-soft); font-size: 10px; font-weight: 560; white-space: nowrap; }
  .badge::before { content: ""; width: 5px; height: 5px; border-radius: 50%; background: currentColor; }
  .badge.success, .badge.running, .badge.active, .badge.online { border-color: rgba(112,214,162,.18); background: var(--green-soft); color: var(--green); }
  .badge.queued, .badge.preparing { border-color: rgba(231,198,109,.18); background: var(--yellow-soft); color: var(--yellow); }
  .badge.failed, .badge.cancelled, .badge.degraded { border-color: rgba(240,130,130,.18); background: var(--red-soft); color: var(--red); }
  .badge.build { border-color: rgba(169,154,248,.18); background: var(--violet-soft); color: var(--violet); }

  .table-wrap { overflow-x: auto; }
  table { width: 100%; border-collapse: collapse; }
  th { height: 38px; padding: 0 16px; color: var(--text-faint); font-size: 9px; font-weight: 700; letter-spacing: .08em; text-align: left; text-transform: uppercase; }
  td { height: 54px; padding: 8px 16px; border-top: 1px solid var(--line); color: var(--text-soft); font-size: 11px; }
  td.primary { min-width: 200px; color: var(--text); }
  td.primary a { display: flex; align-items: center; gap: 9px; text-decoration: none; }
  td.primary a:hover { color: var(--ember); }
  .mono { font-family: var(--mono); font-size: 10px; }
  .muted { color: var(--text-faint); }

  .empty { display: grid; min-height: 180px; place-items: center; padding: 32px; color: var(--text-faint); text-align: center; }
  .empty svg { width: 24px; margin-bottom: 10px; }
  .empty strong { display: block; color: var(--text-soft); font-size: 12px; font-weight: 580; }
  .empty span { display: block; margin-top: 4px; font-size: 10px; }

  .project-banner { position: relative; display: grid; grid-template-columns: 1fr auto; gap: 24px; margin-bottom: 18px; padding: 24px; overflow: hidden; border: 1px solid var(--line); border-radius: 11px; background: linear-gradient(115deg, rgba(227,130,66,.10), transparent 40%), var(--surface); }
  .project-banner::after { content: ""; position: absolute; right: -50px; bottom: -90px; width: 280px; height: 180px; border: 1px solid rgba(227,130,66,.12); border-radius: 50%; transform: rotate(-11deg); }
  .project-name { font-size: 19px; font-weight: 600; letter-spacing: -.03em; }
  .project-slug { margin-top: 5px; color: var(--text-faint); font: 10px var(--mono); }
  .project-facts { position: relative; z-index: 1; display: flex; align-items: center; gap: 24px; }
  .fact strong { display: block; font-size: 16px; font-weight: 580; }
  .fact span { color: var(--text-faint); font-size: 9px; letter-spacing: .06em; text-transform: uppercase; }
  .digest { overflow: hidden; max-width: 100%; color: var(--text-soft); font: 10px var(--mono); text-overflow: ellipsis; white-space: nowrap; }

  .trend-grid { display: grid; grid-template-columns: minmax(0, 1.4fr) minmax(280px, .6fr); gap: 14px; margin-bottom: 14px; }
  .trend-panel { min-height: 190px; }
  .duration-bars { display: flex; height: 136px; align-items: end; gap: 4px; padding: 22px 18px 18px; }
  .duration-bars > i { min-width: 3px; flex: 1; border-radius: 2px 2px 0 0; background: linear-gradient(180deg, var(--ember), rgba(227,130,66,.35)); }
  .duration-bars .empty { width: 100%; min-height: 90px; }
  .project-health { display: grid; grid-template-columns: 1fr 1fr; }
  .project-health > div { display: grid; place-content: center; padding: 20px; border-right: 1px solid var(--line); text-align: center; }
  .project-health > div:last-child { border-right: 0; }
  .project-health span { color: var(--text-faint); font-size: 10px; }
  .project-health strong { display: block; margin-top: 9px; font-size: 24px; font-weight: 570; letter-spacing: -.04em; }

  .detail-grid { display: grid; grid-template-columns: minmax(0, 1.3fr) minmax(280px, .7fr); gap: 14px; }
  .detail-stack { display: grid; gap: 14px; align-content: start; }
  .command { margin: 0; padding: 18px; overflow-x: auto; border: 1px solid var(--line); border-radius: 8px; background: #0d0d10; color: #d8d5dc; font: 11px/1.7 var(--mono); white-space: pre-wrap; }
  .command .prompt { color: var(--ember); user-select: none; }
  .definition { display: grid; grid-template-columns: 120px minmax(0,1fr); gap: 0; margin: 0; }
  .definition dt, .definition dd { min-height: 42px; margin: 0; padding: 11px 14px; border-bottom: 1px solid var(--line); }
  .definition dt { color: var(--text-faint); font-size: 10px; }
  .definition dd { overflow: hidden; color: var(--text-soft); font: 10px var(--mono); text-overflow: ellipsis; }
  .definition > :nth-last-child(-n+2) { border-bottom: 0; }
  .log { max-height: 430px; margin: 0; padding: 18px; overflow: auto; background: #09090b; color: #c9c6ce; font: 10.5px/1.7 var(--mono); tab-size: 2; white-space: pre-wrap; }
  .log-note { padding: 8px 18px; border-top: 1px solid var(--line); color: var(--text-faint); font-size: 9px; }

  .audit-action { color: var(--text); font: 10px var(--mono); }
  .metadata { display: flex; max-width: 360px; flex-wrap: wrap; gap: 4px; }
  .metadata span { padding: 2px 5px; border: 1px solid var(--line); border-radius: 4px; color: var(--text-faint); font: 9px var(--mono); }

  .loading { display: grid; min-height: calc(100vh - 60px); place-items: center; color: var(--text-faint); }
  .loader { display: flex; align-items: center; gap: 9px; }
  .loader::before { content: ""; width: 8px; height: 8px; border-radius: 50%; background: var(--ember); animation: pulse 1.4s ease-in-out infinite; }
  @keyframes pulse { 50% { opacity: .28; transform: scale(.72); } }

  @media (max-width: 1040px) {
    .metrics { grid-template-columns: repeat(2, 1fr); }
    .detail-grid, .trend-grid { grid-template-columns: 1fr; }
  }

  @media (max-width: 760px) {
    .shell { display: block; }
    .sidebar { position: static; width: 100%; height: auto; padding: 10px 12px; border-right: 0; border-bottom: 1px solid var(--line); }
    .sidebar .brand { padding: 0 4px; }
    .nav-section { display: flex; gap: 4px; margin-top: 8px; overflow-x: auto; }
    .nav-section .nav-label { display: none; }
    .nav-link { flex: 0 0 auto; min-height: 32px; }
    .projects-nav { padding-bottom: 3px; }
    .sidebar-foot { display: none; }
    .topbar { top: 0; height: 50px; padding: 0 16px; }
    .content { padding: 26px 16px 56px; }
    .page-head { display: block; }
    .read-only { margin-top: 16px; }
    .metrics { grid-template-columns: 1fr 1fr; }
    .metric { min-height: 96px; padding: 14px; }
    .metric-value { margin-top: 12px; font-size: 20px; }
    .project-banner { grid-template-columns: 1fr; }
    .project-facts { justify-content: space-between; }
    .resource-chart svg { height: 190px; }
    .chart-legend { justify-content: flex-start; flex-wrap: wrap; }
    th:nth-child(3), td:nth-child(3), th:nth-child(5), td:nth-child(5), th:nth-child(6), td:nth-child(6) { display: none; }
  }

  @media (max-width: 430px) {
    .metrics { grid-template-columns: 1fr; }
    .project-facts { gap: 12px; }
    .definition { grid-template-columns: 96px minmax(0,1fr); }
  }

  @media (prefers-reduced-motion: reduce) {
    *, *::before, *::after { scroll-behavior: auto !important; animation-duration: .01ms !important; animation-iteration-count: 1 !important; transition-duration: .01ms !important; }
  }
`
