// SatelliteMap.jsx — stylised top-down "satellite" view of Weligama Bay.
// Renders a fixed 400x700 SVG that scales to fill its container.
// `time` ('morning'|'afternoon'|'evening') drives a CSS lighting overlay.

// deterministic pseudo-random so scatter is stable across renders
function _rng(seed) {
  let s = seed % 2147483647;
  if (s <= 0) s += 2147483646;
  return () => (s = (s * 16807) % 2147483647) / 2147483647;
}

// coastline: gentle curve, land above, ocean below
const COAST = 'M0,300 C70,278 140,322 210,318 C280,314 340,286 400,300 L400,0 L0,0 Z';
const BEACH = 'M0,300 C70,278 140,322 210,318 C280,314 340,286 400,300 L400,338 C340,326 280,360 210,358 C140,360 70,318 0,340 Z';

function _buildings() {
  const r = _rng(7);
  const out = [];
  // dense town band hugging the shore, thinning inland
  for (let i = 0; i < 120; i++) {
    const x = r() * 400;
    const yTop = 40 + r() * 230;          // land only
    const coastY = 300 - 18 * Math.sin((x / 400) * Math.PI); // approx
    if (yTop > coastY - 12) continue;
    const near = 1 - (coastY - yTop) / 260; // closer to coast => bigger/denser
    if (r() > 0.35 + near * 0.5) continue;
    const w = 4 + r() * (6 + near * 7);
    const h = 4 + r() * (6 + near * 7);
    out.push({ x, y: yTop, w, h, rot: (r() - 0.5) * 30, tone: r() });
  }
  return out;
}
function _trees() {
  const r = _rng(23);
  const out = [];
  for (let i = 0; i < 150; i++) {
    const x = r() * 400;
    const y = 20 + r() * 270;
    const coastY = 300 - 18 * Math.sin((x / 400) * Math.PI);
    if (y > coastY - 6) continue;
    out.push({ x, y, rad: 2.4 + r() * 3.6, tone: r() });
  }
  return out;
}
const _B = _buildings();
const _T = _trees();

function SatelliteMap({ time = 'afternoon' }) {
  // lighting overlays keyed by time of day
  const overlays = {
    morning: {
      bg: 'radial-gradient(120% 90% at 80% -10%, rgba(255,224,166,0.42), rgba(255,238,205,0.10) 45%, rgba(150,190,200,0.0) 70%)',
      blend: 'soft-light', extra: 'linear-gradient(180deg, rgba(255,247,230,0.18), rgba(120,160,170,0.05))',
    },
    afternoon: {
      bg: 'radial-gradient(120% 90% at 50% 0%, rgba(255,255,255,0.20), rgba(255,255,255,0.0) 55%)',
      blend: 'soft-light', extra: 'none',
    },
    evening: {
      bg: 'radial-gradient(120% 80% at 50% 18%, rgba(255,150,70,0.55), rgba(255,94,120,0.30) 42%, rgba(70,40,90,0.40) 78%, rgba(35,22,55,0.55))',
      blend: 'overlay', extra: 'linear-gradient(180deg, rgba(40,24,60,0.30), rgba(20,12,35,0.42))',
    },
  };
  const ov = overlays[time] || overlays.afternoon;

  return (
    <div style={{ position: 'absolute', inset: 0, overflow: 'hidden', background: '#0e7c8a' }}>
      <svg viewBox="0 0 400 700" preserveAspectRatio="xMidYMid slice"
           style={{ position: 'absolute', inset: 0, width: '100%', height: '100%' }}>
        <defs>
          <linearGradient id="ocean" x1="0" y1="0.2" x2="0.2" y2="1">
            <stop offset="0" stopColor="#2bb6c4" />
            <stop offset="0.45" stopColor="#1593a8" />
            <stop offset="1" stopColor="#0b6f86" />
          </linearGradient>
          <linearGradient id="landg" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stopColor="#6f8f3f" />
            <stop offset="1" stopColor="#557636" />
          </linearGradient>
          <linearGradient id="sandg" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0" stopColor="#efe0b8" />
            <stop offset="1" stopColor="#e8d49a" />
          </linearGradient>
          <filter id="terr" x="-5%" y="-5%" width="110%" height="110%">
            <feTurbulence type="fractalNoise" baseFrequency="0.16 0.14" numOctaves="3" seed="11" result="n" />
            <feColorMatrix in="n" type="matrix"
              values="0 0 0 0 0.30  0 0 0 0 0.40  0 0 0 0 0.16  0 0 0 0.6 0" result="t" />
            <feComposite in="t" in2="SourceGraphic" operator="in" result="ti" />
            <feBlend in="SourceGraphic" in2="ti" mode="multiply" />
          </filter>
          <filter id="water" x="-5%" y="-5%" width="110%" height="110%">
            <feTurbulence type="fractalNoise" baseFrequency="0.012 0.05" numOctaves="2" seed="4" result="n" />
            <feColorMatrix in="n" type="matrix"
              values="0 0 0 0 1  0 0 0 0 1  0 0 0 0 1  0 0 0 0.10 0" result="t" />
            <feComposite in="t" in2="SourceGraphic" operator="in" result="ti" />
            <feBlend in="SourceGraphic" in2="ti" mode="screen" />
          </filter>
        </defs>

        {/* ocean */}
        <rect x="0" y="280" width="400" height="420" fill="url(#ocean)" />
        <rect x="0" y="280" width="400" height="420" fill="url(#ocean)" filter="url(#water)" opacity="0.9" />
        {/* shallows / reef near shore */}
        <path d="M0,300 C70,280 140,322 210,318 C280,314 340,288 400,300 L400,360 C320,344 250,378 180,372 C110,366 60,336 0,356 Z"
              fill="#54cdd2" opacity="0.45" />
        {/* Taprobane-style islet */}
        <g>
          <ellipse cx="306" cy="430" rx="26" ry="20" fill="#5d7d39" />
          <ellipse cx="306" cy="430" rx="26" ry="20" fill="#5d7d39" filter="url(#terr)" />
          <ellipse cx="306" cy="430" rx="33" ry="26" fill="none" stroke="#7fdadd" strokeWidth="3" opacity="0.5" />
        </g>
        {/* breaking-wave lines */}
        {[372, 392, 415].map((y, i) => (
          <path key={i} d={`M${-10 + i*8},${y} C100,${y-9} 300,${y+9} 410,${y-3}`}
                fill="none" stroke="#eafcfd" strokeWidth={2.2 - i*0.4} opacity={0.5 - i*0.12} />
        ))}

        {/* beach */}
        <path d={BEACH} fill="url(#sandg)" />
        {/* land */}
        <path d={COAST} fill="url(#landg)" />
        <path d={COAST} fill="url(#landg)" filter="url(#terr)" />

        {/* roads */}
        <g stroke="#d9c7a4" fill="none" strokeLinecap="round">
          <path d="M40,0 C70,90 60,180 120,250" strokeWidth="6" opacity="0.85" />
          <path d="M0,150 C120,140 260,170 400,120" strokeWidth="7" opacity="0.9" />
          <path d="M250,0 C240,80 280,160 250,250" strokeWidth="5" opacity="0.8" />
          <path d="M120,250 C200,235 300,250 360,235" strokeWidth="4" opacity="0.7" />
        </g>
        <g stroke="#f3ead0" fill="none" strokeLinecap="round" opacity="0.5">
          <path d="M0,150 C120,140 260,170 400,120" strokeWidth="2.2" />
          <path d="M40,0 C70,90 60,180 120,250" strokeWidth="2" />
        </g>

        {/* trees */}
        {_T.map((t, i) => (
          <circle key={'t'+i} cx={t.x} cy={t.y} r={t.rad}
                  fill={t.tone > 0.5 ? '#3f5e26' : '#4d7030'} opacity="0.9" />
        ))}
        {/* buildings (with tiny shadow for relief) */}
        {_B.map((b, i) => (
          <g key={'b'+i} transform={`translate(${b.x},${b.y}) rotate(${b.rot})`}>
            <rect x="1.2" y="1.2" width={b.w} height={b.h} rx="1" fill="#000" opacity="0.16" />
            <rect width={b.w} height={b.h} rx="1"
                  fill={b.tone > 0.7 ? '#d7d2c7' : b.tone > 0.4 ? '#c66b4e' : '#e8e3d8'} />
          </g>
        ))}
      </svg>

      {/* time-of-day lighting */}
      <div style={{
        position: 'absolute', inset: 0, pointerEvents: 'none',
        background: ov.bg, mixBlendMode: ov.blend,
        transition: 'background 700ms ease, opacity 700ms ease',
      }} />
      {ov.extra !== 'none' && (
        <div style={{
          position: 'absolute', inset: 0, pointerEvents: 'none',
          background: ov.extra, transition: 'background 700ms ease',
        }} />
      )}
      {/* sun glint on water at evening */}
      {time === 'evening' && (
        <div style={{
          position: 'absolute', left: '50%', top: '46%', width: '46%', height: '40%',
          transform: 'translateX(-50%)', pointerEvents: 'none',
          background: 'radial-gradient(60% 80% at 50% 0%, rgba(255,196,120,0.85), rgba(255,150,90,0.0) 70%)',
          mixBlendMode: 'screen',
        }} />
      )}
    </div>
  );
}

Object.assign(window, { SatelliteMap });
