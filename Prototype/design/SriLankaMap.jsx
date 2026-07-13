// SriLankaMap.jsx — stylised silhouette of Sri Lanka with region bubbles.
// Each region shows traveler demand (outer ring) vs vendor supply (inner dot).
// Big demand + low supply => "gap" => pulsing accent ring.

const { useState: useStateSL } = React;

// silhouette in a 0..100 x, 0..100 y space (scaled to viewBox)
const ISLE = `
M46,3
C54,3 58,7 60,12
C63,18 60,23 62,28
C66,33 72,33 74,40
C78,47 80,52 81,59
C82,66 79,72 74,78
C68,85 60,90 52,93
C47,95 43,95 39,92
C33,88 29,84 27,78
C23,71 20,64 20,56
C20,48 22,42 24,37
C27,31 30,27 32,22
C35,15 38,9 42,5
C43,4 44,3 46,3 Z`;

function tone(gap) {
  if (gap >= 28) return { ring: '#e11d48', label: 'Critical gap' };
  if (gap >= 12) return { ring: '#d97706', label: 'Under-served' };
  if (gap <= -10) return { ring: '#0b6f86', label: 'Saturated' };
  return { ring: '#64748b', label: 'Balanced' };
}

function SriLankaMap({ regions, selected, onSelect, accent }) {
  const VB = 400, H = 520, padX = 36, padY = 24;
  const px = (x) => padX + (x / 100) * (VB - padX * 2);
  const py = (y) => padY + (y / 100) * (H - padY * 2);

  return (
    <svg viewBox={`0 0 ${VB} ${H}`} style={{ width: '100%', height: '100%', display: 'block' }}>
      <defs>
        <linearGradient id="isleFill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor="#1a423c" />
          <stop offset="1" stopColor="#123430" />
        </linearGradient>
        <filter id="isleGlow" x="-20%" y="-20%" width="140%" height="140%">
          <feGaussianBlur stdDeviation="6" result="b" />
          <feColorMatrix in="b" type="matrix" values="0 0 0 0 0.05  0 0 0 0 0.58  0 0 0 0 0.53  0 0 0 0.5 0" />
        </filter>
      </defs>

      {/* sea grid */}
      {Array.from({ length: 9 }).map((_, i) => (
        <line key={'h'+i} x1="0" x2={VB} y1={(i+1)*(H/10)} y2={(i+1)*(H/10)} stroke="rgba(255,255,255,0.04)" />
      ))}
      {Array.from({ length: 7 }).map((_, i) => (
        <line key={'v'+i} y1="0" y2={H} x1={(i+1)*(VB/8)} x2={(i+1)*(VB/8)} stroke="rgba(255,255,255,0.04)" />
      ))}

      {/* island */}
      <g transform={`scale(${VB/100}, ${H/100})`}>
        <path d={ISLE} fill="#0b6f86" opacity="0.18" filter="url(#isleGlow)" />
      </g>
      <g transform={`scale(${VB/100}, ${H/100})`}>
        <path d={ISLE} fill="url(#isleFill)" stroke="rgba(45,212,191,0.55)" strokeWidth="0.5" />
      </g>

      {/* region bubbles */}
      {regions.map((r) => {
        const gap = r.travelers - r.vendors;
        const tn = tone(gap);
        const isGap = gap >= 12;
        const cx = px(r.x), cy = py(r.y);
        const demandR = 6 + (r.travelers / 100) * 16;
        const supplyR = 3 + (r.vendors / 100) * 11;
        const isSel = selected === r.id;
        return (
          <g key={r.id} onClick={() => onSelect(r.id)} style={{ cursor: 'pointer' }}>
            {isGap && (
              <circle cx={cx} cy={cy} r={demandR + 6} fill="none" stroke={tn.ring} strokeWidth="1.5" opacity="0.5">
                <animate attributeName="r" values={`${demandR};${demandR+10};${demandR}`} dur="2.4s" repeatCount="indefinite" />
                <animate attributeName="opacity" values="0.6;0;0.6" dur="2.4s" repeatCount="indefinite" />
              </circle>
            )}
            {/* demand ring */}
            <circle cx={cx} cy={cy} r={demandR} fill={tn.ring} fillOpacity="0.16" stroke={tn.ring}
                    strokeWidth={isSel ? 2.5 : 1.5} strokeOpacity={isSel ? 1 : 0.7} />
            {/* supply core */}
            <circle cx={cx} cy={cy} r={supplyR} fill={tn.ring} fillOpacity="0.9" />
            {/* label */}
            <text x={cx} y={cy - demandR - 5} textAnchor="middle" fontSize="11" fontWeight="700"
                  fill={isSel ? '#fff' : 'rgba(255,255,255,0.78)'} fontFamily="'IBM Plex Sans', sans-serif">{r.name}</text>
          </g>
        );
      })}
    </svg>
  );
}

Object.assign(window, { SriLankaMap, regionTone: tone });
