import { css } from 'lit'

export const runsTableStyles = css`
  :host {
    --surface: #141417;
    --surface-hover: #1e1e23;
    --canvas-soft: #101013;
    --line: rgba(255,255,255,.09);
    --line-strong: rgba(255,255,255,.15);
    --text: #f5f3ed;
    --text-soft: #aaa7af;
    --text-faint: #74727a;
    --ember: #e38242;
    --green: #70d6a2;
    --green-soft: rgba(112,214,162,.11);
    --red: #f08282;
    --red-soft: rgba(240,130,130,.11);
    --yellow: #e7c66d;
    --yellow-soft: rgba(231,198,109,.11);
    --mono: "SFMono-Regular", "Cascadia Code", "Roboto Mono", Consolas, monospace;
    display: block;
    color: var(--text);
  }
  * { box-sizing: border-box; }
  .runs-panel { overflow: hidden; border: 1px solid var(--line); border-radius: 11px; background: linear-gradient(180deg,rgba(255,255,255,.018),transparent),var(--surface); box-shadow: 0 16px 42px rgba(0,0,0,.12); }
  .runs-head { display: flex; min-height: 66px; align-items: center; justify-content: space-between; gap: 20px; padding: 10px 14px 10px 18px; border-bottom: 1px solid var(--line); }
  .runs-head > div:first-child { display: flex; align-items: baseline; gap: 10px; white-space: nowrap; }
  .runs-head strong { font-size: 12px; font-weight: 620; }
  .runs-head span { color: var(--text-faint); font: 10px var(--mono); }
  .runs-tools { display: flex; justify-content: flex-end; gap: 7px; width: min(100%, 620px); }
  label { min-width: 0; }
  .search { flex: 1 1 240px; }
  input, select { width: 100%; height: 34px; border: 1px solid var(--line); border-radius: 7px; outline: none; background: #0e0e11; color: var(--text-soft); font: inherit; font-size: 11px; }
  input { padding: 0 11px; }
  select { min-width: 128px; padding: 0 28px 0 10px; }
  input::placeholder { color: var(--text-faint); }
  input:focus, select:focus { border-color: rgba(227,130,66,.5); box-shadow: 0 0 0 3px rgba(227,130,66,.09); }
  .table-wrap { overflow-x: auto; }
  table { width: 100%; border-collapse: collapse; }
  th { height: 38px; padding: 0 16px; color: var(--text-faint); font-size: 9px; font-weight: 700; letter-spacing: .08em; text-align: left; text-transform: uppercase; }
  td { height: 56px; padding: 8px 16px; border-top: 1px solid var(--line); color: var(--text-soft); font-size: 11px; }
  tbody tr { transition: background 120ms ease; }
  tbody tr:hover { background: rgba(255,255,255,.018); }
  td.primary { min-width: 220px; color: var(--text); }
  td.primary a { display: flex; align-items: center; gap: 9px; color: inherit; text-decoration: none; }
  td.primary a:hover { color: var(--ember); }
  .kind-icon { display: grid; flex: 0 0 auto; width: 27px; height: 27px; place-items: center; border: 1px solid var(--line); border-radius: 6px; background: var(--canvas-soft); color: var(--text-faint); font: 10px var(--mono); }
  .mono { font-family: var(--mono); font-size: 10px; }
  .muted { color: var(--text-faint); }
  .badge { display: inline-flex; align-items: center; gap: 6px; padding: 3px 7px; border: 1px solid var(--line); border-radius: 999px; color: var(--text-soft); font-size: 10px; font-weight: 560; white-space: nowrap; }
  .badge::before { content: ""; width: 5px; height: 5px; border-radius: 50%; background: currentColor; }
  .badge.succeeded, .badge.success, .badge.running, .badge.active { border-color: rgba(112,214,162,.18); background: var(--green-soft); color: var(--green); }
  .badge.queued, .badge.preparing { border-color: rgba(231,198,109,.18); background: var(--yellow-soft); color: var(--yellow); }
  .badge.failed, .badge.cancelled { border-color: rgba(240,130,130,.18); background: var(--red-soft); color: var(--red); }
  .position { display: inline-grid; min-width: 20px; height: 20px; margin-left: 6px; place-items: center; border: 1px solid var(--line); border-radius: 5px; }
  .empty { display: grid; min-height: 180px; place-content: center; padding: 32px; text-align: center; }
  .empty strong { font-size: 12px; font-weight: 580; }
  .empty span { display: block; margin-top: 4px; color: var(--text-faint); font-size: 10px; }
  .sr-only { position: absolute; width: 1px; height: 1px; padding: 0; overflow: hidden; clip: rect(0,0,0,0); white-space: nowrap; border: 0; }
  @media (max-width: 900px) {
    .runs-head { align-items: stretch; flex-direction: column; padding: 14px 16px; }
    .runs-tools { width: 100%; justify-content: stretch; }
    .search { flex-basis: auto; }
    th:nth-child(3), td:nth-child(3), th:nth-child(5), td:nth-child(5), th:nth-child(6), td:nth-child(6) { display: none; }
  }
  @media (max-width: 620px) {
    .runs-tools { flex-wrap: wrap; }
    .search { flex: 1 0 100%; }
    label:not(.search) { flex: 1; }
    select { min-width: 0; }
    th:nth-child(7), td:nth-child(7) { display: none; }
  }
`
