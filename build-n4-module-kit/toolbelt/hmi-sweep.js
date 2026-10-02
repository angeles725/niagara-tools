#!/usr/bin/env node
/*
 * hmi-sweep.js — reusable HMI kiosk sweep: every navigable view of a dashboard SPA, at the panel
 * resolution, checked for (a) no document-level scroll, (b) inner scrollers only where named, and
 * (c) a target element (e.g. the global alarm banner) visible and not covered by anything else.
 *
 * Why a kit tool: an ad-hoc sweep re-derived per session drifted in WHAT it swept (33 views one
 * session, 26 the next, same app). One app-agnostic enumeration convention makes re-runs
 * comparable and proves a behavior-neutral split (run it before and after).
 * [ev: retro comppan-fase2-amps-alarms Δ3] [ev: retro dashboard-rc-file-split Δ5]
 * [ev: retro panccadia-persistent-config-hoa Δ4]
 *
 * View enumeration (app-agnostic, no app globals):
 *   1. every element matching --nav (default `.nav-item`) in document order is clicked; the view
 *      label is its data-page, else its trimmed text, else its index;
 *   2. after each nav click, every VISIBLE element matching each --subtab selector is clicked in
 *      order; label `<nav>#<subtab label>`.
 *   With no nav element the page itself is one view (`main`).
 * Scenarios: `--scenario <name>[=<query>]` (repeatable; default one `default` with no query). The
 * query is appended to every request whose URL contains --api-match (the scenario-forwarding rule
 * in types/dashboard.md: a fixed jsonUrl never sees the page's own `?scenario`), or to the page
 * URL when --api-match is absent.
 *
 * Usage:
 *   node toolbelt/hmi-sweep.js --url <page-url> [--viewport 1280x800] [--scenario name[=query]]...
 *       [--api-match <substring>] [--nav <selector>] [--subtab <selector>]...
 *       [--target <selector>] [--allow-scroller <selector>]... [--settle-ms 80]
 *       [--screenshot-dir <dir>] [--chrome-path <chrome>]
 * Row:     PASS|FAIL|WARN  hmi-sweep  <scenario>/<view>  no-scroll|target|inner-scroller|subtab: <detail>
 *          A --target that matches nothing (or only a hidden element) in a view is a FAIL row; a
 *          sub-tab that is no longer visible after the previous click is a WARN row (not swept).
 *          --settle-ms is a non-negative integer (ms).
 * Summary: hmi-sweep: N views · p PASS · f FAIL · w WARN  ->  CLEAN|ISSUES
 * Exit:    0 no FAIL · 1 any FAIL · 3 usage/env · 4 tool unavailable (puppeteer-core or Chrome
 *          missing — reported, never a pass)
 * Tools: puppeteer-core (KIT_PUPPETEER = module path/dir, else require('puppeteer-core') via
 *   NODE_PATH) and a Chrome binary (--chrome-path, else KIT_CHROME, else PUPPETEER_EXECUTABLE_PATH).
 *   The sweep runs on a desktop Chrome at the panel viewport: it proves layout fit, not the panel
 *   engine's feature floor (that is rc-scan.sh browser-floor + lint-vendor-floor.sh).
 */
'use strict';

const fs = require('fs');
const path = require('path');

const USAGE = 'usage: hmi-sweep.js --url <page-url> [--viewport WxH] [--scenario name[=query]]... ' +
  '[--api-match <substr>] [--nav <sel>] [--subtab <sel>]... [--target <sel>] ' +
  '[--allow-scroller <sel>]... [--settle-ms N] [--screenshot-dir <dir>] [--chrome-path <chrome>]';

class UsageError extends Error {}

function parseArgs(argv) {
  const out = {
    url: null, width: 1280, height: 800, nav: '.nav-item', subtabs: [], target: null,
    allowScrollers: [], scenarios: [], apiMatch: null, settleMs: 80, screenshotDir: null,
    chromePath: process.env.KIT_CHROME || process.env.PUPPETEER_EXECUTABLE_PATH || null,
  };
  const need = (i) => {
    if (i + 1 >= argv.length) throw new UsageError(`missing value for ${argv[i]}`);
    return argv[i + 1];
  };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    switch (a) {
      case '--url': out.url = need(i); i++; break;
      case '--viewport': {
        const m = /^(\d+)x(\d+)$/.exec(need(i)); i++;
        if (!m) throw new UsageError('--viewport must be WxH, e.g. 1280x800');
        out.width = Number(m[1]); out.height = Number(m[2]); break;
      }
      case '--scenario': {
        const v = need(i); i++;
        const eq = v.indexOf('=');
        out.scenarios.push(eq < 0 ? { name: v, query: '' }
          : { name: v.slice(0, eq), query: v.slice(eq + 1) });
        break;
      }
      case '--api-match': out.apiMatch = need(i); i++; break;
      case '--nav': out.nav = need(i); i++; break;
      case '--subtab': out.subtabs.push(need(i)); i++; break;
      case '--target': out.target = need(i); i++; break;
      case '--allow-scroller': out.allowScrollers.push(need(i)); i++; break;
      case '--settle-ms': {
        const v = need(i); i++;
        if (!/^\d+$/.test(v)) throw new UsageError('--settle-ms must be a non-negative integer (ms)');
        out.settleMs = Number(v); break;
      }
      case '--screenshot-dir': out.screenshotDir = need(i); i++; break;
      case '--chrome-path': out.chromePath = need(i); i++; break;
      default: throw new UsageError(`unknown argument: ${a}`);
    }
  }
  if (!out.url) throw new UsageError('--url is required');
  if (out.scenarios.length === 0) out.scenarios.push({ name: 'default', query: '' });
  return out;
}

function withQuery(url, query) {
  if (!query) return url;
  return url + (url.includes('?') ? '&' : '?') + query;
}

/*
 * Pure verdict for one measured view. m = { docScrollY, docScrollX,
 *   target: null (no --target) | { present: false, sel } | { present: true, visible, unoccluded },
 *   scrollers: [{ sel, allowed }] }, or m = { gone: true } for a sub-tab that vanished before its click.
 */
function rowsForView(scenario, label, m) {
  const id = `${scenario}/${label}`;
  const rows = [];
  if (m.gone) {
    return [`WARN  hmi-sweep  ${id}  subtab: no longer visible after the previous click (not swept)`];
  }
  if (m.docScrollY || m.docScrollX) {
    const axes = [m.docScrollY ? 'vertical' : '', m.docScrollX ? 'horizontal' : ''].filter(Boolean);
    rows.push(`FAIL  hmi-sweep  ${id}  no-scroll: document scrolls (${axes.join('+')}) at the panel viewport`);
  } else {
    rows.push(`PASS  hmi-sweep  ${id}  no-scroll: document fits`);
  }
  if (m.target && !m.target.present) {
    rows.push(`FAIL  hmi-sweep  ${id}  target: ${m.target.sel} not found (no element, or only a hidden one)`);
  } else if (m.target) {
    if (!m.target.visible) {
      rows.push(`FAIL  hmi-sweep  ${id}  target: present but not visible (zero box or off-screen)`);
    } else if (!m.target.unoccluded) {
      rows.push(`FAIL  hmi-sweep  ${id}  target: covered by another element at its center`);
    } else {
      rows.push(`PASS  hmi-sweep  ${id}  target: visible and on top`);
    }
  }
  for (const s of m.scrollers || []) {
    if (!s.allowed) {
      rows.push(`WARN  hmi-sweep  ${id}  inner-scroller: ${s.sel} scrolls and is not named by --allow-scroller`);
    }
  }
  return rows;
}

function summarize(rows, views) {
  let p = 0; let f = 0; let w = 0;
  for (const r of rows) {
    if (r.startsWith('PASS')) p++;
    else if (r.startsWith('FAIL')) f++;
    else if (r.startsWith('WARN')) w++;
  }
  return {
    line: `hmi-sweep: ${views} view${views === 1 ? '' : 's'} · ${p} PASS · ${f} FAIL · ${w} WARN  ->  ` +
      (f > 0 ? 'ISSUES' : 'CLEAN'),
    exit: f > 0 ? 1 : 0,
  };
}

function loadPuppeteer() {
  const spec = process.env.KIT_PUPPETEER || 'puppeteer-core';
  try {
    return require(spec);
  } catch (e) {
    return null;
  }
}

// Runs in the page: measure the current view.
function measureInPage(targetSel, allowSels) {
  const de = document.documentElement;
  const res = {
    docScrollY: de.scrollHeight > de.clientHeight,
    docScrollX: de.scrollWidth > de.clientWidth,
    target: null,
    scrollers: [],
  };
  if (targetSel) {
    const t = document.querySelector(targetSel);
    res.target = { present: false, sel: targetSel };
    if (t && !t.hidden) {
      const r = t.getBoundingClientRect();
      const visible = r.width > 0 && r.height > 0 && r.bottom > 0 && r.right > 0 &&
        r.top < window.innerHeight && r.left < window.innerWidth;
      let unoccluded = false;
      if (visible) {
        const top = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2);
        unoccluded = !!(top && (top === t || t.contains(top)));
      }
      res.target = { present: true, visible, unoccluded };
    }
  }
  const describe = (el) => el.tagName.toLowerCase() + (el.id ? `#${el.id}` : '') +
    (el.classList.length ? `.${Array.from(el.classList).join('.')}` : '');
  for (const el of document.body.querySelectorAll('*')) {
    const cs = getComputedStyle(el);
    const scrollY = /(auto|scroll)/.test(cs.overflowY) && el.scrollHeight > el.clientHeight + 1;
    const scrollX = /(auto|scroll)/.test(cs.overflowX) && el.scrollWidth > el.clientWidth + 1;
    if (!(scrollY || scrollX) || el.getClientRects().length === 0) continue;
    res.scrollers.push({ sel: describe(el), allowed: allowSels.some((s) => el.matches(s)) });
  }
  return res;
}

// Runs in the page (serialized by $$eval, so self-contained): click the i-th VISIBLE element and
// return its label, or null when the visible set shrank after an earlier click (no click, no throw).
function clickVisibleNth(els, i) {
  const vis = els.filter((e) => e.getClientRects().length > 0);
  const el = vis[i];
  if (!el) return null;
  el.click();
  return (el.dataset && el.dataset.tab) || (el.textContent || '').trim() || String(i);
}

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function sweepScenario(browser, args, sc) {
  const page = await browser.newPage();
  await page.setViewport({ width: args.width, height: args.height });
  if (args.apiMatch && sc.query) {
    await page.setRequestInterception(true);
    page.on('request', (req) => {
      const u = req.url();
      if (u.includes(args.apiMatch)) req.continue({ url: withQuery(u, sc.query) });
      else req.continue();
    });
  }
  const pageUrl = args.apiMatch ? args.url : withQuery(args.url, sc.query);
  await page.goto(pageUrl, { waitUntil: 'networkidle0' });
  await sleep(args.settleMs * 2);

  const out = [];
  const measure = async (label) => {
    const m = await page.evaluate(measureInPage, args.target, args.allowScrollers);
    out.push({ label, m });
    if (args.screenshotDir) {
      fs.mkdirSync(args.screenshotDir, { recursive: true });
      const file = `${sc.name}_${label}`.replace(/[^A-Za-z0-9_.-]+/g, '_') + '.png';
      await page.screenshot({ path: path.join(args.screenshotDir, file) });
    }
  };
  const navCount = await page.$$eval(args.nav, (els) => els.length);
  if (navCount === 0) {
    await measure('main');
  }
  for (let n = 0; n < navCount; n++) {
    const navLabel = await page.$$eval(args.nav, (els, i) => {
      const el = els[i];
      if (!el) return null;
      el.click();
      return el.dataset.page || (el.textContent || '').trim() || String(i);
    }, n);
    if (navLabel === null) {
      out.push({ label: `${args.nav}[${n}]`, m: { gone: true } });
      continue;
    }
    await sleep(args.settleMs);
    let subSeen = 0;
    for (const sel of args.subtabs) {
      const count = await page.$$eval(sel, (els) => els.filter((e) => e.getClientRects().length > 0).length);
      for (let s = 0; s < count; s++) {
        const subLabel = await page.$$eval(sel, clickVisibleNth, s);
        if (subLabel === null) {
          out.push({ label: `${navLabel}#${sel}[${s}]`, m: { gone: true } });
          continue;
        }
        await sleep(args.settleMs);
        await measure(`${navLabel}#${subLabel}`);
        subSeen++;
      }
    }
    if (subSeen === 0) await measure(navLabel);
  }
  await page.close();
  return out;
}

async function main(argv) {
  let args;
  try {
    args = parseArgs(argv);
  } catch (e) {
    if (e instanceof UsageError) {
      console.error(`hmi-sweep: ${e.message}\n${USAGE}`);
      return 3;
    }
    throw e;
  }
  const puppeteer = loadPuppeteer();
  if (!puppeteer) {
    console.log(`SKIP  hmi-sweep  ${args.url}  unavailable: puppeteer-core not resolvable ` +
      '(npm install puppeteer-core, then NODE_PATH or KIT_PUPPETEER)');
    return 4;
  }
  if (!args.chromePath || !fs.existsSync(args.chromePath)) {
    console.log(`SKIP  hmi-sweep  ${args.url}  unavailable: Chrome not found ` +
      '(--chrome-path, KIT_CHROME or PUPPETEER_EXECUTABLE_PATH)');
    return 4;
  }
  const browser = await puppeteer.launch({ executablePath: args.chromePath, headless: true,
    args: ['--no-sandbox', '--allow-file-access-from-files'] });
  const rows = [];
  let views = 0;
  try {
    for (const sc of args.scenarios) {
      const measured = await sweepScenario(browser, args, sc);
      for (const v of measured) {
        views++;
        rows.push(...rowsForView(sc.name, v.label, v.m));
      }
    }
  } finally {
    await browser.close();
  }
  for (const r of rows) console.log(r);
  const s = summarize(rows, views);
  console.log(s.line);
  return s.exit;
}

if (require.main === module) {
  main(process.argv.slice(2)).then((code) => process.exit(code), (e) => {
    console.error(`hmi-sweep: env fault: ${e && e.message ? e.message : e}`);
    process.exit(3);
  });
}

module.exports = { parseArgs, rowsForView, summarize, withQuery, clickVisibleNth };
