// Generate 1200×630 OG/social cards in the vabc brand style.
// Renders an HTML template (with embedded fonts) via headless Chrome.
//   node scripts/gen-og.mjs
import { execFileSync } from "node:child_process";
import { mkdirSync, readFileSync, writeFileSync, rmSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";
import { glob } from "node:fs/promises";

const root = join(dirname(fileURLToPath(import.meta.url)), "..");
const outDir = join(root, "public", "og");
mkdirSync(outDir, { recursive: true });

const CHROME = process.env.CHROME || "google-chrome";

async function findFont(pkg, file) {
  for await (const m of glob(join(root, "node_modules/.pnpm", `*${pkg}*`, "**", file))) return m;
  throw new Error("font not found: " + file);
}
const b64 = (p) => readFileSync(p).toString("base64");
const display = b64(await findFont("bricolage-grotesque", "bricolage-grotesque-latin-800-normal.woff2"));
const mono = b64(await findFont("ibm-plex-mono", "ibm-plex-mono-latin-500-normal.woff2"));

const cards = [
  {
    slug: "default",
    title: "vabc",
    sub: "Virginia ABC product search & store inventory — from your terminal.",
  },
  {
    slug: "landing",
    title: "Find the bottle.<br/><span class='amber'>Skip the website.</span>",
    sub: "Live Virginia ABC catalog + store inventory. Agent-friendly, read-only.",
  },
];

function html({ title, sub }) {
  return `<!doctype html><html><head><meta charset="utf-8"><style>
  @font-face{font-family:'D';src:url(data:font/woff2;base64,${display}) format('woff2');font-weight:800}
  @font-face{font-family:'M';src:url(data:font/woff2;base64,${mono}) format('woff2');font-weight:500}
  *{margin:0;box-sizing:border-box}
  body{width:1200px;height:630px;overflow:hidden;
    background:radial-gradient(820px 460px at 84% -12%,rgba(232,177,90,.18),transparent 60%),#0b0a07;
    color:#efe7d6;font-family:'M';padding:72px 76px;position:relative}
  .grain{position:absolute;inset:0;opacity:.05;background-image:url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='120' height='120'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='.8' numOctaves='2'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E")}
  .brand{font-family:'M';font-size:24px;letter-spacing:.18em;text-transform:uppercase;color:#e8b15a}
  .brand b{color:#efe7d6}
  h1{font-family:'D';font-weight:800;font-size:84px;line-height:.98;letter-spacing:-.03em;margin-top:120px}
  .amber{color:#e8b15a;text-shadow:0 0 50px rgba(232,177,90,.4)}
  p{font-family:'M';font-size:27px;color:#b6a98f;margin-top:28px;max-width:22em}
  .board{position:absolute;right:76px;bottom:64px;font-family:'M';font-size:21px;color:#8a7d64;text-align:right;line-height:1.9}
  .g{color:#6bd49a}.a{color:#e8604a}
  .bar{position:absolute;left:0;bottom:0;height:8px;width:100%;
    background:linear-gradient(90deg,#e8b15a,#b9863a 60%,#6bd49a)}
  </style></head><body>
  <div class="grain"></div>
  <div class="brand"><b>vabc</b> · virginia abc cli</div>
  <h1>${title}</h1>
  <p>${sub}</p>
  <div class="board">store #219 vienna&nbsp;&nbsp;qty <span class="g">21</span><br/>store #76 falls church&nbsp;&nbsp;qty <span class="g">33</span><br/>store #267 mclean&nbsp;&nbsp;qty <span class="a">out</span></div>
  <div class="bar"></div>
  </body></html>`;
}

const tmp = join(root, ".og-tmp");
mkdirSync(tmp, { recursive: true });
for (const c of cards) {
  const f = join(tmp, c.slug + ".html");
  writeFileSync(f, html(c));
  const out = join(outDir, c.slug + ".png");
  execFileSync(CHROME, [
    "--headless=new", "--disable-gpu", "--hide-scrollbars",
    "--force-device-scale-factor=1", "--window-size=1200,630",
    "--screenshot=" + out, "file://" + f,
  ], { stdio: "ignore" });
  console.log("wrote", out);
}
rmSync(tmp, { recursive: true, force: true });
