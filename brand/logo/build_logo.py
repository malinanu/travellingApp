"""Spotto passport-stamp logo family. Geometry measured from the approved
ChatGPT render (1932 px canvas) and cleaned up; all units are those pixels."""
import math, os, sys
OUT = sys.argv[1]; os.makedirs(OUT, exist_ok=True)
INK, TEAL, LAGOON, SUN, CREAM = "#103D3A", "#0B6F86", "#0E8C8A", "#F6B81E", "#FBF6ED"
f = lambda v: (f"{v:.1f}").rstrip("0").rstrip(".")

def svg(name, body, x, y, w, h, title="Spotto logo", bg=None, rx=0):
    back = f'<rect x="{f(x)}" y="{f(y)}" width="{f(w)}" height="{f(h)}" rx="{f(rx)}" fill="{bg}"/>' if bg else ""
    open(os.path.join(OUT, name), "w", newline="\n").write(
        f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="{f(x)} {f(y)} {f(w)} {f(h)}" role="img">'
        f'<title>{title}</title>{back}{body}</svg>\n')

# ---- symbol: half sun on a horizon, two shorter sea lines ----
CX, HY, SR, LW = 966, 939, 206, 30          # centre x, horizon centre y, sun radius, line weight
LINES = [(375, HY), (271, HY + 60), (157, HY + 120)]  # half-length, y
def symbol(c_line, c_sea, c_sun, lines=LINES, lw=LW, sr=SR):
    top = HY - lw / 2
    s = f'<path d="M{f(CX-sr)} {f(top)} A{sr} {sr} 0 0 1 {f(CX+sr)} {f(top)} Z" fill="{c_sun}"/>'
    for i, (hl, y) in enumerate(lines):
        s += (f'<line x1="{f(CX-hl)}" y1="{f(y)}" x2="{f(CX+hl)}" y2="{f(y)}" stroke="{c_line if i == 0 else c_sea}" '
              f'stroke-width="{lw}" stroke-linecap="round"/>')
    return s

# ---- lettering: rounded geometric caps, stroke-built ----
CAP, SW = 92, 25          # cap height, stem weight
def letter(ch, x, y, c):
    w = {"S": 72, "P": 64, "O": 78, "T": 80}[ch]
    h = CAP; i = SW / 2; st = f'fill="none" stroke="{c}" stroke-width="{SW}"'
    if ch == "O":
        return f'<rect x="{f(x+i)}" y="{f(y+i)}" width="{f(w-SW)}" height="{f(h-SW)}" rx="{f((w-SW)/2)}" {st}/>', w
    if ch == "T":
        return (f'<path d="M{f(x)} {f(y+i)} H{f(x+w)} M{f(x+w/2)} {f(y+i)} V{f(y+h)}" {st}/>'), w
    if ch == "P":
        bh = h * 0.62; r = (bh - SW) / 2
        return (f'<path d="M{f(x+i)} {f(y+h)} V{f(y+i)} H{f(x+w-i-r)} A{f(r)} {f(r)} 0 0 1 {f(x+w-i-r)} {f(y+bh-i)} H{f(x+i)}" '
                f'{st} stroke-linejoin="round"/>'), w
    if ch == "S":
        r = (h - SW) / 4; L, R = x + i, x + w - i; t, m, b = y + i, y + h / 2, y + h - i
        d = (f"M{f(R)} {f(t)} H{f(L+r)} A{f(r)} {f(r)} 0 0 0 {f(L+r)} {f(m)} H{f(R-r)} "
             f"A{f(r)} {f(r)} 0 0 1 {f(R-r)} {f(b)} H{f(x)}")
        return f'<path d="{d}" {st}/>', w
def word(x, y, c, gap=40, scale=1.0):
    g, cx = "", 0
    for ch in "SPOTTO":
        s, w = letter(ch, cx, 0, c); g += s; cx += w + gap
    width = cx - gap
    return f'<g transform="translate({f(x)} {f(y)}) scale({scale})">{g}</g>', width * scale

def star(cx, cy, r, c):
    pts = []
    for k in range(10):
        rr = r if k % 2 == 0 else r * 0.45; a = math.radians(-90 + 36 * k)
        pts.append(f"{f(cx + rr*math.cos(a))},{f(cy + rr*math.sin(a))}")
    return f'<polygon points="{" ".join(pts)}" fill="{c}" stroke="{c}" stroke-width="5" stroke-linejoin="round"/>'

# ---- stamp ----
def stamp(c_ink, c_sea, c_sun):
    o = (430, 570, 1505, 1362, 205, 34)   # outer frame centreline box, radius, weight
    n = (482, 625, 1452, 1307, 150, 13)
    g = ""
    for x0, y0, x1, y1, rx, sw in (o, n):
        g += f'<rect x="{x0}" y="{y0}" width="{x1-x0}" height="{y1-y0}" rx="{rx}" fill="none" stroke="{c_ink}" stroke-width="{sw}"/>'
    g += symbol(c_ink, c_sea, c_sun)
    _, ww = word(0, 0, c_ink)
    wx = CX - ww / 2; ty = 1136
    w_, _ = word(wx, ty, c_ink); g += w_
    g += star(wx - 72, ty + CAP / 2, 26, c_ink) + star(wx + ww + 72, ty + CAP / 2, 26, c_ink)
    return g
SB = (396, 536, 1144, 860)   # stamp viewBox x y w h (tight + 17 px margin)
svg("spotto-stamp.svg", stamp(INK, TEAL, SUN), *SB, title="Spotto")
svg("spotto-stamp-reversed.svg", stamp(CREAM, LAGOON, SUN), *SB, title="Spotto")
svg("spotto-stamp-ink.svg", stamp(INK, INK, INK), *SB, title="Spotto")
svg("spotto-stamp-white.svg", stamp("#FFFFFF", "#FFFFFF", "#FFFFFF"), *SB, title="Spotto")

# ---- symbol files: square, centred ----
sym_w = 2 * LINES[0][0] + LW; sym_top = HY - LW / 2 - SR; sym_bot = LINES[-1][1] + LW / 2
sym_cy = (sym_top + sym_bot) / 2 - 8
def square(pad):
    side = sym_w + 2 * pad; return (CX - side / 2, sym_cy - side / 2, side, side)
SQ = square(70)
svg("spotto-symbol.svg", symbol(INK, TEAL, SUN), *SQ, title="Spotto")
svg("spotto-symbol-reversed.svg", symbol(CREAM, LAGOON, SUN), *SQ, title="Spotto")
svg("spotto-symbol-ink.svg", symbol(INK, INK, INK), *SQ, title="Spotto")
svg("spotto-symbol-white.svg", symbol("#FFFFFF", "#FFFFFF", "#FFFFFF"), *SQ, title="Spotto")
# small-size cut: bigger sun share, two short heavy lines, cropped to its own width
SM_LW, SM_R = 76, 250
SMALL = [(300, HY + 6), (175, HY + 112)]
def small(c1, c2, c3): return symbol(c1, c2, c3, lines=SMALL, lw=SM_LW, sr=SM_R)
sm_w = 2 * SMALL[0][0] + SM_LW; sm_top = HY + 6 - SM_LW / 2 - SM_R; sm_bot = SMALL[-1][1] + SM_LW / 2
sm_cy = (sm_top + sm_bot) / 2
def sm_square(pad): side = sm_w + 2 * pad; return (CX - side / 2, sm_cy - side / 2, side, side)
svg("spotto-symbol-small.svg", small(INK, TEAL, SUN), *sm_square(40), title="Spotto")
svg("spotto-symbol-small-reversed.svg", small(CREAM, LAGOON, SUN), *sm_square(40), title="Spotto")
Ts = sm_w / 0.74
svg("spotto-app-icon-small.svg", small(INK, TEAL, SUN), CX - Ts/2, sm_cy - Ts/2, Ts, Ts, title="Spotto", bg=CREAM)
# app icon: cream tile, symbol at ~62 % width (inside Android adaptive safe zone)
T = sym_w / 0.68; TB = (CX - T / 2, sym_cy - T / 2, T, T)
svg("spotto-app-icon.svg", symbol(INK, TEAL, SUN), *TB, title="Spotto", bg=CREAM)
svg("spotto-app-icon-rounded.svg", symbol(INK, TEAL, SUN), *TB, title="Spotto", bg=CREAM, rx=T * 0.22)

# ---- horizontal lockup: symbol + wordmark ----
def horizontal(c1, c2, c3):
    g = symbol(c1, c2, c3)
    sc = 1.55; _, ww = word(0, 0, c1, gap=34, scale=sc)
    wy = (sym_top + sym_bot) / 2 - CAP * sc / 2 + 6
    w_, _ = word(CX + sym_w / 2 + 90, wy, c1, gap=34, scale=sc)
    return g + w_, CX + sym_w / 2 + 90 + ww
for suf, cols in (("", (INK, TEAL, SUN)), ("-reversed", (CREAM, LAGOON, SUN)), ("-ink", (INK,)*3), ("-white", ("#FFFFFF",)*3)):
    g, right = horizontal(*cols); left = CX - sym_w / 2
    svg(f"spotto-horizontal{suf}.svg", g, left - 30, sym_top - 40, right - left + 60, sym_bot - sym_top + 80, title="Spotto")
print("ok")

# ---- Android adaptive-icon foreground: transparent 108dp canvas, symbol inside the 66dp safe circle ----
F = sym_w / 0.52
svg("spotto-android-foreground.svg", symbol(INK, TEAL, SUN), CX - F / 2, sym_cy - F / 2, F, F, title="Spotto")
print("android ok")
