// Generate the GitHub repo social-preview card (1280×640) for vabc — a proof-forward card
// showing a real catalog search returning clean JSON, in the warm amber/whiskey brand.
// Output: .github/social-preview.png (uploaded manually in repo Settings → Social preview).
// Run from the site dir (has @fontsource fonts + sharp): node scripts/gen-social.mjs
import { execFileSync } from 'node:child_process';
import { readFileSync, mkdirSync, writeFileSync, globSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import sharp from 'sharp';

const SITE = resolve(dirname(fileURLToPath(import.meta.url)), '..'); // site
const REPO = resolve(SITE, '..');
const OUT = join(REPO, '.github', 'social-preview.png');
const CHROME = process.env.CHROME || '/usr/bin/google-chrome';
const W = 1280, H = 640;

// @fontsource (non-variable): files are weight-specific (*-latin-<w>-normal.woff2). Pick the
// first available weight from the preference list.
function fontB64(family, weights) {
  for (const w of weights) {
    const [f] = globSync(`node_modules/@fontsource/${family}/files/*-latin-${w}-normal.woff2`, { cwd: SITE });
    if (f) return readFileSync(join(SITE, f)).toString('base64');
  }
  throw new Error(`font not found: ${family}`);
}
const bric = fontB64('bricolage-grotesque', [800, 700, 600]);
const mono = fontB64('ibm-plex-mono', [500, 400]);

const html = `<!doctype html><html><head><meta charset="utf-8"><style>
@font-face{font-family:'Bric';src:url(data:font/woff2;base64,${bric}) format('woff2');font-weight:800}
@font-face{font-family:'Mono';src:url(data:font/woff2;base64,${mono}) format('woff2');font-weight:500}
*{margin:0;box-sizing:border-box}
html,body{width:${W}px;height:${H}px}
body{background:#0b0a07;color:#f7f1e4;font-family:'Mono',monospace;position:relative;overflow:hidden}
.bg{position:absolute;inset:0;background:
  radial-gradient(900px 520px at 8% -12%, rgba(232,177,90,.26), transparent 60%),
  radial-gradient(760px 520px at 112% 122%, rgba(232,96,74,.16), transparent 55%),
  radial-gradient(620px 460px at 95% 6%, rgba(232,177,90,.12), transparent 60%)}
.frame{position:absolute;inset:22px;border:1px solid rgba(232,177,90,.20);border-radius:18px}
.wrap{position:absolute;inset:0;padding:52px 60px;display:flex;flex-direction:column;justify-content:space-between}
.top{display:flex;align-items:center;justify-content:space-between;font-size:24px}
.brand{display:flex;align-items:center;gap:14px;font-weight:600}
.dot{width:15px;height:15px;border-radius:50%;background:#e8b15a;box-shadow:0 0 22px 4px rgba(232,177,90,.7)}
.meta{color:#8a7c5e;letter-spacing:.12em;text-transform:uppercase;font-size:19px}
.title{font-family:'Bric',sans-serif;font-weight:800;font-size:62px;line-height:1.04;letter-spacing:-.02em;max-width:1110px}
.title .ac{color:#e8b15a}
.term{background:#080705;border:1px solid rgba(232,177,90,.20);border-radius:14px;padding:22px 26px;font-size:23px;line-height:1.5}
.dots{display:flex;gap:8px;margin-bottom:14px}
.dots i{width:13px;height:13px;border-radius:50%;display:inline-block}
.cmd{color:#f7f1e4}.cmd .p{color:#e8b15a}.k{color:#f7cf83}.s{color:#6bd49a}.n{color:#e8b15a}.t{color:#6bd49a}.muted{color:#8a7c5e}
.bottom{display:flex;align-items:center;justify-content:space-between;gap:24px;font-size:21px}
.tags{display:flex;gap:10px;flex-shrink:0}
.tag{border:1px solid rgba(232,177,90,.34);color:#f1d9a8;border-radius:999px;padding:6px 13px;font-size:18px;white-space:nowrap}
.install{color:#e8b15a;white-space:nowrap}
</style></head><body>
<div class="bg"></div><div class="frame"></div>
<div class="wrap">
  <div class="top">
    <div class="brand"><span class="dot"></span>vabc</div>
    <div class="meta">Virginia ABC · no API key</div>
  </div>
  <div class="title">Virginia ABC catalog &amp; live store stock,<br><span class="ac">as clean JSON.</span></div>
  <div class="term">
    <div class="dots"><i style="background:#ff5f56"></i><i style="background:#ffbd2e"></i><i style="background:#27c93f"></i></div>
    <div class="cmd"><span class="p">$</span> vabc product search "blanton's" --json | jq '.[0]'</div>
    <div class="cmd"><span class="muted">{</span> <span class="k">"name"</span>: <span class="s">"Blanton's Single Barrel"</span>, <span class="k">"price"</span>: <span class="n">64.99</span>,</div>
    <div class="cmd">  <span class="k">"proof"</span>: <span class="n">93</span>, <span class="k">"in_stock"</span>: <span class="t">true</span>, <span class="k">"stores_nearby"</span>: <span class="n">3</span> <span class="muted">}</span></div>
  </div>
  <div class="bottom">
    <div class="tags"><span class="tag">read-only</span><span class="tag">no API key</span><span class="tag">live store stock</span><span class="tag">MIT</span></div>
    <div class="install">$ brew install rnwolfe/tap/vabc</div>
  </div>
</div></body></html>`;

mkdirSync(dirname(OUT), { recursive: true });
const tmp = join(tmpdir(), 'vabc-social.html');
const raw = join(tmpdir(), 'vabc-social-raw.png');
writeFileSync(tmp, html);
// This headless Chrome paints ~half the window height → render at 2× and crop the top.
execFileSync(CHROME, [
  '--headless=new', '--no-sandbox', '--hide-scrollbars', '--force-device-scale-factor=1',
  `--window-size=${W},${H * 2}`, '--default-background-color=00000000',
  '--virtual-time-budget=1500', `--screenshot=${raw}`, `file://${tmp}`,
], { stdio: 'ignore' });
await sharp(raw).extract({ left: 0, top: 0, width: W, height: H }).toFile(OUT);
console.log('wrote', OUT);
