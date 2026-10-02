// Desenha a demonstração do README (spec 013 FR-2) a partir da transcrição real
// gerada pelo demo.sh: os comandos ("$ " e "> ") aparecem sendo digitados, a
// saída aparece linha a linha. Precisa do playwright (NODE_PATH) e do ffmpeg.
//
// Uso: node docs/demo/render.cjs transcript.txt demo.gif
// FONT_CSS=arquivo.css troca as fontes (padrão: Google Fonts).
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { execFileSync } = require('node:child_process');
const { chromium } = require('playwright');

const [input, output] = process.argv.slice(2);
if (!input || !output) {
  console.error('uso: render.cjs transcript.txt demo.gif');
  process.exit(2);
}
const lines = fs.readFileSync(input, 'utf8').trimEnd().split('\n');
const fontCss = process.env.FONT_CSS
  ? `<link rel="stylesheet" href="file://${path.resolve(process.env.FONT_CSS)}">`
  : '<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@500;700&family=Inter:wght@500;600&display=swap">';

const esc = (s) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;');
function lineHtml(text, cursor) {
  const c = cursor ? '<span class="cur"></span>' : '';
  if (text.startsWith('$ ')) return `<div><span class="p">$</span> <span class="cmd">${esc(text.slice(2))}</span>${c}</div>`;
  if (text.startsWith('> ')) return `<div><span class="q">&gt;</span> <span class="ask">${esc(text.slice(2))}</span>${c}</div>`;
  let cls = 'out';
  if (/^criado: /.test(text)) cls = 'ok';
  if (/0 issues\.|^ok\s|cobertura: /.test(text)) cls = 'ok';
  if (/^sdd-kit .* em /.test(text)) cls = 'sum';
  return `<div class="${cls}">${esc(text)}${c}</div>`;
}
function page(shown) {
  return `<!doctype html><html><head><meta charset="utf-8">${fontCss}<style>
  *{box-sizing:border-box;margin:0;padding:0}
  body{width:1000px;height:660px;background:#0b1020;font-family:Inter,sans-serif;padding:28px}
  .win{height:100%;background:#070b17;border:1px solid #243055;border-radius:14px;overflow:hidden;display:flex;flex-direction:column}
  .bar{display:flex;gap:8px;align-items:center;padding:12px 16px;border-bottom:1px solid #243055;color:#9aa6c7;font-size:15px}
  .bar i{width:12px;height:12px;border-radius:50%;display:block}
  .bar span{margin-left:10px}
  .term{flex:1;padding:16px 20px;font:500 16px/1.55 'JetBrains Mono','DejaVu Sans Mono',monospace;color:#c9d2ea;
    white-space:pre-wrap;word-break:break-all;display:flex;flex-direction:column;justify-content:flex-end;overflow:hidden}
  .p{color:#3ddc97;font-weight:700}.q{color:#7c9cff;font-weight:700}
  .cmd{color:#e8ecf8}.ask{color:#e8ecf8}
  .ok{color:#3ddc97}.sum{color:#ffc857}.out{color:#9aa6c7}
  .cur{display:inline-block;width:9px;height:18px;background:#e8ecf8;vertical-align:-3px;margin-left:2px}
  </style></head><body><div class="win"><div class="bar"><i style="background:#ff5f57"></i>
  <i style="background:#febc2e"></i><i style="background:#28c840"></i><span>sdd-kit · adoção e primeiro make ci</span></div>
  <div class="term">${shown}</div></div></body></html>`;
}

// Quadros: [html, segundos]. Comandos digitados em pedaços; saída linha a linha.
const frames = [];
const done = [];
for (const text of lines) {
  const typed = text.startsWith('$ ') || text.startsWith('> ');
  if (typed) {
    const prefix = 2;
    const step = Math.max(3, Math.ceil((text.length - prefix) / 18));
    for (let i = prefix; i < text.length; i += step) {
      frames.push([page(done.join('') + lineHtml(text.slice(0, i), true)), 0.05]);
    }
    done.push(lineHtml(text, false));
    frames.push([page(done.join('')), 0.5]);
  } else {
    done.push(lineHtml(text, false));
    frames.push([page(done.join('')), text === '…' ? 0.3 : 0.12]);
  }
}
frames[frames.length - 1][1] = 4;

(async () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'demo-'));
  const browser = await chromium.launch();
  const tab = await browser.newPage({ viewport: { width: 1000, height: 660 } });
  const list = [];
  for (let i = 0; i < frames.length; i++) {
    await tab.setContent(frames[i][0]);
    await tab.evaluate(() => document.fonts.ready);
    const file = path.join(dir, `f${String(i).padStart(4, '0')}.png`);
    await tab.screenshot({ path: file });
    list.push(`file '${file}'`, `duration ${frames[i][1]}`);
  }
  list.push(`file '${path.join(dir, `f${String(frames.length - 1).padStart(4, '0')}.png`)}'`);
  await browser.close();
  fs.writeFileSync(path.join(dir, 'list.txt'), list.join('\n') + '\n');
  execFileSync('ffmpeg', ['-y', '-loglevel', 'error', '-f', 'concat', '-safe', '0', '-i', path.join(dir, 'list.txt'),
    '-vf', 'fps=20,split[a][b];[a]palettegen=max_colors=64[p];[b][p]paletteuse=dither=none', '-loop', '0', output]);
  fs.rmSync(dir, { recursive: true, force: true });
})();
