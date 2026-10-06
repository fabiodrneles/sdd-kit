#!/usr/bin/env python3
# Gera docs/assets/relay-savings-{pt,en}-{light,dark}.svg com os números do
# sdd-report.sh phase --compare (spec 019 FR-4). Uso: python3 docs/assets/relay-savings.py docs/assets
import sys
out=sys.argv[1]
T={
 'pt':dict(title='Custo por ticket: sessão longa (padrão) contra o relé do sdd-kit',
  h1='menos tokens por ticket', h2='menos contexto relido por chamada',
  c1='Tokens gastos por ticket', c2='Contexto relido a cada chamada',
  base='Sessão longa (padrão)', kit='sdd-kit com relé',
  v1a='6,50 milhões', v1b='0,40 milhão', v2a='231 mil', v2b='43 mil',
  foot1='Medido com sdd-report.sh, sem LLM: Fase 12 (4 tickets numa sessão longa) contra Fase 13 (3 tickets, um agente novo por ticket), 2026-10-06.',
  foot2='Os tickets da Fase 13 eram menores; o contexto por chamada não depende do tamanho do ticket.',
  x16='16×', x54='5,4×'),
 'en':dict(title='Cost per ticket: one long session (default) vs the sdd-kit relay',
  h1='fewer tokens per ticket', h2='less context reread per call',
  c1='Tokens spent per ticket', c2='Context reread on every call',
  base='Long session (default)', kit='sdd-kit relay',
  v1a='6.50 million', v1b='0.40 million', v2a='231 thousand', v2b='43 thousand',
  foot1='Measured with sdd-report.sh, no LLM: Phase 12 (4 tickets in one long session) vs Phase 13 (3 tickets, one fresh agent each), 2026-10-06.',
  foot2='Phase 13 tickets were smaller; context per call does not depend on ticket size.',
  x16='16×', x54='5.4×'),
}
C={
 'light':dict(bg='#fcfcfb',card='#ffffff',border='#e6e5e0',t1='#0b0b0b',t2='#52514e',t3='#77766f',grid='#ecebe6',base='#eb6834',kit='#2a78d6'),
 'dark':dict(bg='#1a1a19',card='#232322',border='#3a3a37',t1='#ffffff',t2='#c3c2b7',t3='#9a998f',grid='#33332f',base='#d95926',kit='#3987e5'),
}
W,H=880,500
def bars(y,label,va,vb,a,b,c,t):
    # a,b in same units; one axis per chart
    x0,x1=200,700; mx=max(a,b)
    s=f'<text x="40" y="{y}" font-size="15" font-weight="600" fill="{c["t1"]}">{label}</text>'
    for i,(name,v,val,col) in enumerate(((t['base'],a,va,c['base']),(t['kit'],b,vb,c['kit']))):
        yy=y+18+i*40; w=max(6,(x1-x0)*v/mx)
        s+=f'<text x="40" y="{yy+19}" font-size="13" fill="{c["t2"]}">{name}</text>'
        s+=f'<path d="M{x0} {yy+4} h{w-4} a4 4 0 0 1 4 4 v12 a4 4 0 0 1 -4 4 h-{w-4} z" fill="{col}"><title>{name}: {val}</title></path>'
        s+=f'<text x="{x0+w+10}" y="{yy+19}" font-size="13" font-weight="600" fill="{c["t1"]}">{val}</text>'
    return s
for lang,t in T.items():
    for mode,c in C.items():
        s=f'''<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}" viewBox="0 0 {W} {H}" font-family="-apple-system, Segoe UI, Helvetica, Arial, sans-serif" role="img" aria-label="{t['title']}: {t['x16']} {t['h1']}, {t['x54']} {t['h2']}">
<rect x="0.5" y="0.5" width="{W-1}" height="{H-1}" rx="12" fill="{c['bg']}" stroke="{c['border']}"/>
<text x="40" y="48" font-size="19" font-weight="700" fill="{c['t1']}">{t['title']}</text>
'''
        for i,(big,small) in enumerate(((t['x16'],t['h1']),(t['x54'],t['h2']))):
            x=40+i*410
            s+=f'<rect x="{x}" y="72" width="390" height="104" rx="10" fill="{c["card"]}" stroke="{c["border"]}"/>'
            s+=f'<text x="{x+24}" y="138" font-size="52" font-weight="700" fill="{c["t1"]}">{big}</text>'
            s+=f'<text x="{x+24}" y="162" font-size="14" fill="{c["t2"]}">{small}</text>'
            s+=f'<rect x="{x}" y="72" width="6" height="104" rx="3" fill="{c["kit"]}"/>'
        s+=bars(222,t['c1'],t['v1a'],t['v1b'],6.50,0.403,c,t)
        s+=bars(334,t['c2'],t['v2a'],t['v2b'],231,43,c,t)
        s+=f'<text x="40" y="456" font-size="12" fill="{c["t3"]}">{t["foot1"]}</text>'
        s+=f'<text x="40" y="474" font-size="12" fill="{c["t3"]}">{t["foot2"]}</text>'
        s+='</svg>\n'
        open(f'{out}/relay-savings-{lang}-{mode}.svg','w').write(s)
