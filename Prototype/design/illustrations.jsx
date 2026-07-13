// illustrations.jsx — Sunny Postcard flat SVG scenes that replace the old
// gradient "photo" placeholders. One scene per spot category, all locked to
// the brand palette: teal #0b6f86 / #0e8c8a, cream #fbf6ed, ink #103d3a,
// sun #f6b81e. Renders edge-to-edge like the gradient div it replaces.

const ILLO_TONE = {
  cowork: '#0b6f86',
  surf:   '#0e8c8a',
  sunset: '#0b4a54',
  clay:   '#0a5f6e',
  batik:  '#0b6f86',
};

// icon that accompanies each category (was PHOTO[..].icon in BusinessDrawer)
const PHOTO_ICON = {
  cowork: 'Coffee',
  surf:   'Waves',
  sunset: 'Martini',
  clay:   'Palette',
  batik:  'Brush',
};

const ILLO_SCENES = {
  cowork: (
    <g>
      <rect width="400" height="240" fill="#0b6f86" />
      <rect y="0" width="400" height="120" fill="#0e8c8a" />
      {/* window with sun */}
      <circle cx="316" cy="62" r="42" fill="#fbf6ed" />
      <circle cx="316" cy="62" r="33" fill="#bde4e0" />
      <circle cx="330" cy="50" r="12" fill="#f6b81e" />
      <path d="M283 74 Q300 66 316 74 T349 74" stroke="#0e8c8a" strokeWidth="4" fill="none" />
      {/* shelf */}
      <rect x="36" y="52" width="110" height="6" rx="3" fill="#fbf6ed" opacity="0.55" />
      <rect x="48" y="30" width="12" height="22" rx="3" fill="#fbf6ed" opacity="0.8" />
      <rect x="66" y="24" width="12" height="28" rx="3" fill="#f6b81e" />
      <rect x="84" y="34" width="12" height="18" rx="3" fill="#fbf6ed" opacity="0.8" />
      {/* counter */}
      <rect y="150" width="400" height="90" fill="#fbf6ed" />
      <rect y="150" width="400" height="10" fill="#103d3a" opacity="0.12" />
      {/* laptop */}
      <path d="M96 154 L188 154 L188 106 Q188 100 182 100 L102 100 Q96 100 96 106 Z" fill="#103d3a" />
      <rect x="103" y="107" width="78" height="40" rx="3" fill="#bde4e0" />
      <path d="M112 133 Q126 122 142 133 T172 133" stroke="#0b6f86" strokeWidth="4" fill="none" />
      <rect x="84" y="154" width="116" height="8" rx="4" fill="#0a2e33" />
      {/* coffee cup + steam */}
      <path d="M242 128 h34 v20 a17 17 0 0 1 -34 0 Z" fill="#0b6f86" />
      <path d="M276 132 h8 a8 8 0 0 1 0 16 h-8" stroke="#0b6f86" strokeWidth="5" fill="none" />
      <rect x="236" y="158" width="46" height="6" rx="3" fill="#103d3a" opacity="0.18" />
      <path d="M251 118 c0-6 5-6 5-12 M263 118 c0-6 5-6 5-12" stroke="#f6b81e" strokeWidth="4" strokeLinecap="round" fill="none" />
      {/* plant */}
      <rect x="330" y="140" width="26" height="22" rx="5" fill="#f6b81e" />
      <path d="M343 140 C336 124 324 120 314 124 C326 114 340 122 343 140Z" fill="#103d3a" />
      <path d="M343 140 C350 122 362 118 372 124 C360 112 346 122 343 140Z" fill="#0a5f6e" />
    </g>
  ),

  surf: (
    <g>
      <rect width="400" height="240" fill="#bde4e0" />
      <circle cx="322" cy="56" r="30" fill="#f6b81e" />
      <circle cx="322" cy="56" r="42" fill="#f6b81e" opacity="0.25" />
      {/* clouds */}
      <path d="M64 52 a14 14 0 0 1 28 0 a11 11 0 0 1 20 6 h-64 a12 12 0 0 1 16 -6Z" fill="#fbf6ed" />
      <path d="M170 84 a10 10 0 0 1 20 0 a8 8 0 0 1 15 4 h-47 a9 9 0 0 1 12 -4Z" fill="#fbf6ed" opacity="0.85" />
      {/* sea */}
      <path d="M0 128 Q50 114 100 128 T200 128 T300 128 T400 128 V240 H0 Z" fill="#0e8c8a" />
      <path d="M0 158 Q50 146 100 158 T200 158 T300 158 T400 158 V240 H0 Z" fill="#0b6f86" />
      <path d="M0 196 Q50 186 100 196 T200 196 T300 196 T400 196 V240 H0 Z" fill="#0a5f6e" />
      {/* breaking wave crest */}
      <path d="M28 128 Q60 100 106 108 Q80 112 70 128 Z" fill="#fbf6ed" opacity="0.9" />
      {/* surfboard */}
      <g transform="rotate(-24 250 176)">
        <ellipse cx="250" cy="176" rx="86" ry="20" fill="#fbf6ed" />
        <ellipse cx="250" cy="176" rx="86" ry="20" fill="none" stroke="#103d3a" strokeWidth="4" opacity="0.15" />
        <rect x="166" y="172" width="168" height="7" rx="3.5" fill="#f6b81e" />
      </g>
      {/* gulls */}
      <path d="M120 60 q6 -8 12 0 M132 60 q6 -8 12 0" stroke="#103d3a" strokeWidth="4" strokeLinecap="round" fill="none" opacity="0.7" />
    </g>
  ),

  sunset: (
    <g>
      <rect width="400" height="240" fill="#fbf6ed" />
      <rect y="0" width="400" height="150" fill="#f8e3b4" />
      <rect y="0" width="400" height="86" fill="#fbf6ed" />
      {/* setting sun */}
      <circle cx="200" cy="150" r="52" fill="#f6b81e" />
      <circle cx="200" cy="150" r="68" fill="#f6b81e" opacity="0.28" />
      {/* sea */}
      <rect y="150" width="400" height="90" fill="#0b4a54" />
      <path d="M0 150 Q50 142 100 150 T200 150 T300 150 T400 150 V162 Q350 156 300 162 T200 162 T100 162 T0 162 Z" fill="#0e8c8a" />
      {/* sun glints */}
      <rect x="168" y="176" width="64" height="6" rx="3" fill="#f6b81e" opacity="0.85" />
      <rect x="182" y="194" width="38" height="5" rx="2.5" fill="#f6b81e" opacity="0.6" />
      <rect x="176" y="212" width="50" height="5" rx="2.5" fill="#f6b81e" opacity="0.4" />
      {/* palm silhouette */}
      <path d="M66 226 C64 186 60 160 46 134" stroke="#103d3a" strokeWidth="9" fill="none" strokeLinecap="round" />
      <g fill="#103d3a">
        <path d="M46 134 C26 120 6 120 -8 130 C12 110 34 114 46 134Z" />
        <path d="M46 134 C34 112 38 92 54 80 C40 100 44 118 46 134Z" />
        <path d="M46 134 C62 114 84 110 100 120 C78 106 56 116 46 134Z" />
        <path d="M46 134 C66 128 84 136 92 152 C76 136 56 132 46 134Z" />
        <circle cx="56" cy="140" r="7" />
        <circle cx="40" cy="146" r="7" />
      </g>
      {/* gulls */}
      <path d="M300 66 q6 -8 12 0 M312 66 q6 -8 12 0" stroke="#103d3a" strokeWidth="4" strokeLinecap="round" fill="none" opacity="0.55" />
    </g>
  ),

  clay: (
    <g>
      <rect width="400" height="240" fill="#f3e6cf" />
      {/* back wall arch */}
      <path d="M270 240 V130 a62 62 0 0 1 124 0 V240 Z" fill="#0e8c8a" opacity="0.28" />
      {/* shelf */}
      <rect x="24" y="96" width="220" height="8" rx="4" fill="#103d3a" opacity="0.7" />
      {/* small pots on shelf */}
      <path d="M46 96 v-8 a4 4 0 0 1 4 -4 h16 a4 4 0 0 1 4 4 v8 Z" fill="#0b6f86" />
      <path d="M88 96 v-16 q0 -14 14 -14 q14 0 14 14 v16 Z" fill="#f6b81e" />
      <path d="M148 96 v-10 a5 5 0 0 1 5 -5 h20 a5 5 0 0 1 5 5 v10 Z" fill="#103d3a" />
      {/* big pots on ground */}
      <path d="M60 240 q-26 -10 -26 -44 q0 -34 34 -34 q34 0 34 34 q0 34 -26 44 Z" fill="#0b6f86" />
      <ellipse cx="68" cy="162" rx="26" ry="8" fill="#0a5f6e" />
      <path d="M48 188 q20 -10 40 0" stroke="#fbf6ed" strokeWidth="5" fill="none" strokeLinecap="round" opacity="0.7" />
      <path d="M170 240 q-32 -12 -32 -56 q0 -42 42 -42 q42 0 42 42 q0 44 -32 56 Z" fill="#0a5f6e" />
      <ellipse cx="180" cy="142" rx="32" ry="9" fill="#103d3a" />
      <path d="M152 176 q28 -12 56 0 M156 200 q24 -10 48 0" stroke="#f6b81e" strokeWidth="5" fill="none" strokeLinecap="round" />
      {/* potter's small stool pot */}
      <path d="M300 240 q-18 -8 -18 -32 q0 -24 24 -24 q24 0 24 24 q0 24 -18 32 Z" fill="#103d3a" />
      <path d="M288 212 q18 -8 36 0" stroke="#fbf6ed" strokeWidth="4" fill="none" strokeLinecap="round" opacity="0.6" />
    </g>
  ),

  batik: (
    <g>
      <rect width="400" height="240" fill="#0b6f86" />
      {/* fabric bands */}
      <rect y="0" width="400" height="52" fill="#fbf6ed" />
      <path d="M0 52 Q25 38 50 52 T100 52 T150 52 T200 52 T250 52 T300 52 T350 52 T400 52 V60 H0 Z" fill="#0b6f86" />
      <g fill="#0b6f86">
        <circle cx="36" cy="26" r="7" /><circle cx="108" cy="26" r="7" />
        <circle cx="180" cy="26" r="7" /><circle cx="252" cy="26" r="7" />
        <circle cx="324" cy="26" r="7" /><circle cx="396" cy="26" r="7" />
      </g>
      <rect y="92" width="400" height="46" fill="#0e8c8a" />
      <path d="M0 92 Q25 80 50 92 T100 92 T150 92 T200 92 T250 92 T300 92 T350 92 T400 92" stroke="#fbf6ed" strokeWidth="5" fill="none" />
      <path d="M0 138 Q25 126 50 138 T100 138 T150 138 T200 138 T250 138 T300 138 T350 138 T400 138" stroke="#fbf6ed" strokeWidth="5" fill="none" />
      <g fill="#f6b81e">
        <circle cx="52" cy="115" r="8" /><circle cx="136" cy="115" r="8" />
        <circle cx="220" cy="115" r="8" /><circle cx="304" cy="115" r="8" />
        <circle cx="388" cy="115" r="8" />
      </g>
      <rect y="170" width="400" height="70" fill="#103d3a" />
      <path d="M0 190 Q25 176 50 190 T100 190 T150 190 T200 190 T250 190 T300 190 T350 190 T400 190" stroke="#0e8c8a" strokeWidth="6" fill="none" />
      <path d="M0 216 Q25 202 50 216 T100 216 T150 216 T200 216 T250 216 T300 216 T350 216 T400 216" stroke="#fbf6ed" strokeWidth="5" fill="none" opacity="0.5" />
    </g>
  ),
};

// Drop-in replacement for the gradient placeholder divs. Fills its parent
// (parent must be position:relative). scrim adds the bottom ink fade that
// keeps existing white overlay text readable.
function Illustration({ kind, scrim = true, style, className }) {
  const scene = ILLO_SCENES[kind] || ILLO_SCENES.cowork;
  return (
    <div className={className} style={{ position: 'absolute', inset: 0, overflow: 'hidden', ...style }}>
      <svg viewBox="0 0 400 240" preserveAspectRatio="xMidYMid slice"
           style={{ position: 'absolute', inset: 0, width: '100%', height: '100%', display: 'block' }}>
        {scene}
      </svg>
      {scrim && (
        <div style={{ position: 'absolute', inset: 0,
          background: 'linear-gradient(180deg, transparent 42%, rgba(16,61,58,0.72))' }} />
      )}
    </div>
  );
}

Object.assign(window, { Illustration, ILLO_TONE, PHOTO_ICON });
