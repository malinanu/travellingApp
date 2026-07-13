// AdminMobileApp.jsx — mobile-first Command Center: iOS frame, top bar,
// bottom tab bar, route switching, device scaler + tweaks.

const { useState: useStateM, useEffect: useEffectM, useRef: useRefM } = React;

const PALETTES_M = {
  '#0b6f86': { accent: '#0b6f86' },
  '#0e8c8a': { accent: '#0e8c8a' },
  '#0b4a54': { accent: '#0b4a54' },
  '#c07a3c': { accent: '#c07a3c' },
};

const TWEAK_DEFAULTS_M = /*EDITMODE-BEGIN*/{
  "accent": "#0b6f86",
  "theme": "Light",
  "density": "Comfortable"
}/*EDITMODE-END*/;

const TABS_M = [
  { id: 'queue',   label: 'Queue',   icon: 'UserCheck',   badge: 6 },
  { id: 'monitor', label: 'Live',    icon: 'Radio',       live: true },
  { id: 'geo',     label: 'Geo',     icon: 'Map' },
  { id: 'subs',    label: 'Subs',    icon: 'CreditCard' },
  { id: 'audio',   label: 'Audio',   icon: 'Headphones' },
];

// ── Top bar: brand + notifications + admin avatar ──
function TopBar({ accent }) {
  return (
    <header className="flex items-center gap-2.5 flex-shrink-0"
      style={{ paddingTop: 52, paddingLeft: 18, paddingRight: 16, paddingBottom: 10,
               background: 'var(--surface)', borderBottom: '1px solid var(--line)' }}>
      <svg width="24" height="24" viewBox="0 0 32 32" fill="none">
        <path d="M16 3C10.5 3 6 7.3 6 12.7 6 19.5 16 29 16 29s10-9.5 10-16.3C26 7.3 21.5 3 16 3z" fill={accent} />
        <circle cx="16" cy="12.5" r="4.4" fill="#f6b81e" />
      </svg>
      <div className="leading-none">
        <div className="font-bold text-[15px] tracking-tight" style={{ color: 'var(--ink)' }}>Spotto</div>
        <div className="text-[8.5px] font-bold uppercase tracking-[0.18em] mt-0.5" style={{ color: accent }}>Command Center</div>
      </div>
      <div className="ml-auto flex items-center gap-2">
        <button className="w-9 h-9 rounded-full flex items-center justify-center relative"
          style={{ background: 'var(--bg)', border: '1px solid var(--line)', color: 'var(--ink-soft)' }}>
          <Icon name="Bell" size={17} />
          <span className="absolute top-1.5 right-1.5 w-2 h-2 rounded-full" style={{ background: '#e11d48', boxShadow: '0 0 0 2px var(--surface)' }} />
        </button>
        <span className="w-9 h-9 rounded-full flex items-center justify-center text-[12px] font-bold text-white" style={{ background: '#103d3a' }}>AD</span>
      </div>
    </header>
  );
}

// ── Bottom tab bar ──
function TabBar({ route, setRoute, accent }) {
  return (
    <nav className="flex-shrink-0 flex items-stretch"
      style={{ paddingTop: 8, paddingBottom: 26, paddingLeft: 6, paddingRight: 6,
               background: 'var(--surface)', borderTop: '1px solid var(--line)' }}>
      {TABS_M.map((tb) => {
        const on = route === tb.id;
        return (
          <button key={tb.id} onClick={() => setRoute(tb.id)}
            className="flex-1 flex flex-col items-center gap-1 py-1 relative active:scale-[0.94] transition-transform"
            style={{ color: on ? accent : 'var(--muted)' }}>
            <div className="relative">
              <Icon name={tb.icon} size={23} strokeWidth={on ? 2.5 : 2} />
              {tb.badge && (
                <span className="absolute -top-1.5 -right-2 rounded-full text-[9px] font-bold flex items-center justify-center text-white"
                  style={{ background: '#e11d48', minWidth: 15, height: 15, padding: '0 3px' }}>{tb.badge}</span>
              )}
              {tb.live && (
                <span className="absolute -top-0.5 -right-1.5 w-2.5 h-2.5 rounded-full" style={{ background: '#16a34a', boxShadow: '0 0 0 2px var(--surface)' }} />
              )}
            </div>
            <span className="text-[10px]" style={{ fontWeight: on ? 700 : 600 }}>{tb.label}</span>
          </button>
        );
      })}
    </nav>
  );
}

function AdminMobileApp() {
  const [t, setTweak] = useTweaks(TWEAK_DEFAULTS_M);
  const pal = PALETTES_M[t.accent] || PALETTES_M['#0b6f86'];
  const accent = pal.accent;
  const [route, setRoute] = useStateM('queue');
  const dark = t.theme === 'Dark';
  const compact = t.density === 'Compact';
  const scaleRef = useRefM(null);

  const cssVars = dark ? {
    '--bg': '#0b2320', '--surface': '#10302c', '--line': 'rgba(255,255,255,0.09)',
    '--ink': '#eaf2ef', '--ink-soft': '#c3d6d1', '--muted': '#7f9d97', '--accent': accent,
    '--pad': compact ? '12px' : '16px', '--gap': compact ? '8px' : '12px',
  } : {
    '--bg': '#f7f2e8', '--surface': '#ffffff', '--line': '#e7e1d4',
    '--ink': '#103d3a', '--ink-soft': '#33524f', '--muted': '#6c8582', '--accent': accent,
    '--pad': compact ? '12px' : '16px', '--gap': compact ? '8px' : '12px',
  };

  // scale the device to fit the viewport
  useEffectM(() => {
    function fit() {
      const el = scaleRef.current;
      if (!el) return;
      const s = Math.min(1, (window.innerWidth - 24) / 402, (window.innerHeight - 24) / 874);
      el.style.transform = `scale(${s})`;
    }
    fit();
    window.addEventListener('resize', fit);
    return () => window.removeEventListener('resize', fit);
  }, []);

  const views = {
    queue:   <MobileQueue accent={accent} />,
    monitor: <MobileMonitor accent={accent} />,
    geo:     <MobileGeo accent={accent} />,
    subs:    <MobileSubs accent={accent} />,
    audio:   <MobileAudio accent={accent} />,
  };

  return (
    <div style={{ position: 'fixed', inset: 0, display: 'grid', placeItems: 'center', background: 'radial-gradient(120% 120% at 50% 0%, #ffffff, #f3ead9)' }}>
      <div ref={scaleRef} style={{ transformOrigin: 'center', flexShrink: 0 }}>
        <IOSDevice dark={dark} width={402} height={874}>
          <div style={{ ...cssVars, height: '100%', display: 'flex', flexDirection: 'column',
                        background: 'var(--bg)', color: 'var(--ink)', fontFamily: "'IBM Plex Sans', system-ui" }}>
            <TopBar accent={accent} />
            <main key={route} className="flex-1 min-h-0 overflow-hidden">
              {views[route]}
            </main>
            <TabBar route={route} setRoute={setRoute} accent={accent} />
          </div>
        </IOSDevice>
      </div>

      <TweaksPanel>
        <TweakSection label="Appearance" />
        <TweakRadio label="Theme" value={t.theme} options={['Light', 'Dark']} onChange={(v) => setTweak('theme', v)} />
        <TweakRadio label="Density" value={t.density} options={['Comfortable', 'Compact']} onChange={(v) => setTweak('density', v)} />
        <TweakColor label="Accent" value={t.accent}
          options={['#0b6f86', '#0e8c8a', '#0b4a54', '#c07a3c']}
          onChange={(v) => setTweak('accent', v)} />
      </TweaksPanel>
    </div>
  );
}

ReactDOM.createRoot(document.getElementById('root')).render(<AdminMobileApp />);
