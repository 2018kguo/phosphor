#!/usr/bin/env python3
"""Writes docs/bench-light.svg and docs/bench-dark.svg from the README's
comparison table. Rerun after updating the numbers: python3 docs/chart.py"""
import math, os

LIBS = ["phosphor", "charmbracelet/x/vt", "hinshun/vt10x"]
THROUGHPUT = [  # MiB/s
    ("vim", [350, 23, 30]),
    ("less", [266, 2, 27]),
    ("agent TUI", [408, 36, 43]),
    ("scrolling logs", [404, 5, 39]),
    ("progress bars", [1126, 27, 42]),
]
MEMORY = [3.0, 4.5 * 1024, 66]  # KiB per idle 80x24 terminal

THEMES = {
    "light": dict(surface="#fcfcfb", ink="#0b0b0b", ink2="#52514e", grid="#e4e3df",
                  series=["#2a78d6", "#eb6834", "#1baf7a"]),
    "dark": dict(surface="#1a1a19", ink="#f0efec", ink2="#c3c2b7", grid="#34332f",
                 series=["#3987e5", "#d95926", "#199e70"]),
}

W, LEFT, RIGHT = 760, 128, 60
PLOT = W - LEFT - RIGHT
BAR, GAP, GROUP = 10, 2, 16


def fmt_rate(v):
    return f"{v / 1024:.1f} GiB/s" if v >= 1024 else f"{v:g} MiB/s"


def fmt_mem(v):
    return f"{v / 1024:.1f} MiB" if v >= 1024 else f"{v:g} KiB"


def bar(x0, y, w, color):
    # Square at the baseline, 4px rounded at the data end.
    r = min(4, w, BAR / 2)
    return (f'<path d="M{x0},{y} h{w - r} a{r},{r} 0 0 1 {r},{r} v{BAR - 2 * r} '
            f'a{r},{r} 0 0 1 -{r},{r} h-{w - r} z" fill="{color}"/>')


def panel(t, top, title, note, rows, lo, hi, ticks, fmt, label_all):
    out = [f'<text x="0" y="{top}" class="h">{title}</text>',
           f'<text x="0" y="{top + 18}" class="s">{note}</text>']
    y0 = top + 38
    height = sum(len(v) * (BAR + GAP) - GAP + GROUP for _, v in rows) - GROUP
    scale = lambda v: PLOT * (math.log10(v) - math.log10(lo)) / (math.log10(hi) - math.log10(lo))
    for v, lab in ticks:
        x = LEFT + scale(v)
        out.append(f'<line x1="{x:.1f}" y1="{y0 - 6}" x2="{x:.1f}" y2="{y0 + height + 4}" stroke="{t["grid"]}"/>')
        out.append(f'<text x="{x:.1f}" y="{y0 + height + 18}" class="s" text-anchor="middle">{lab}</text>')
    y = y0
    for name, vals in rows:
        gh = len(vals) * (BAR + GAP) - GAP
        out.append(f'<text x="{LEFT - 10}" y="{y + gh / 2 + 4}" class="l" text-anchor="end">{name}</text>')
        for i, v in enumerate(vals):
            w = max(scale(v), 2)
            out.append(bar(LEFT, y, w, t["series"][i]))
            if i == 0 or label_all:
                out.append(f'<text x="{LEFT + w + 6:.1f}" y="{y + BAR - 1}" class="v">{fmt(v)}</text>')
            y += BAR + GAP
        y += GROUP - GAP
    out.append(f'<line x1="{LEFT}" y1="{y0 - 6}" x2="{LEFT}" y2="{y0 + height + 4}" stroke="{t["ink2"]}"/>')
    return out, y0 + height + 24


def svg(t):
    legend, x = [], 0
    for i, name in enumerate(LIBS):
        legend.append(f'<rect x="{x}" y="0" width="10" height="10" rx="2" fill="{t["series"][i]}"/>'
                      f'<text x="{x + 16}" y="9.5" class="l">{name}</text>')
        x += 16 + 8 * len(name) + 24
    p1, end = panel(t, 44, "Throughput on real program output",
                    "MiB/s on one core, log scale, higher is better", THROUGHPUT, 1, 2000,
                    [(1, "1"), (10, "10"), (100, "100"), (1000, "1,000")], fmt_rate, False)
    mem_rows = [(name, [v]) for name, v in zip(LIBS, MEMORY)]
    p2, end = panel(t, end + 30, "Memory per idle 80×24 terminal",
                    "Heap at a shell prompt, log scale, lower is better", [("", MEMORY)], 1, 10240,
                    [(1, "1 KiB"), (10, "10 KiB"), (100, "100 KiB"), (1024, "1 MiB"), (10240, "10 MiB")],
                    fmt_mem, True)
    h = end + 8
    style = (f'<style>text{{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Helvetica,Arial,sans-serif}}'
             f'.h{{font-size:15px;font-weight:600;fill:{t["ink"]}}}.s{{font-size:11px;fill:{t["ink2"]}}}'
             f'.l{{font-size:12px;fill:{t["ink"]}}}.v{{font-size:11px;fill:{t["ink2"]}}}</style>')
    body = "\n".join(legend + p1 + p2)
    return (f'<svg xmlns="http://www.w3.org/2000/svg" width="{W + 24}" height="{h + 24}" '
            f'viewBox="-12 -12 {W + 24} {h + 24}" role="img" '
            f'aria-label="phosphor vs charmbracelet/x/vt and hinshun/vt10x: throughput and idle memory">'
            f'{style}<rect x="-12" y="-12" width="{W + 24}" height="{h + 24}" rx="8" fill="{t["surface"]}"/>'
            f'{body}</svg>\n')


here = os.path.dirname(os.path.abspath(__file__))
for mode, theme in THEMES.items():
    with open(os.path.join(here, f"bench-{mode}.svg"), "w") as f:
        f.write(svg(theme))
