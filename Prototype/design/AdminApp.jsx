// AdminApp.jsx — Command Center shell: dark rail + topbar + routing + tweaks.

const { useState: useStateAA, useEffect: useEffectAA } = React;

const PALETTES_A = {
  '#0b6f86': { accent: '#0b6f86', tint: 'rgba(11,111,134,0.12)' },
  '#0e8c8a': { accent: '#0e8c8a', tint: 'rgba(14,140,138,0.12)' },
  '#0b4a54': { accent: '#0b4a54', tint: 'rgba(11,74,84,0.12)' },
  '#c07a3c': { accent: '#c07a3c', tint: 'rgba(192,122,60,0.12)' },
};

const TWEAK_DEFAULTS_A = /*EDITMODE-BEGIN*/{
  "accent": "#0b6f86",
  "theme": "Light",
  "authLayout": "split"
}/*EDITMODE-END*/;

const NAV_A = [
  { id: 'queue',   label: 'Approval queue',  icon: 'UserCheck', badge: 6 },
  { id: 'monitor', label: 'Content monitor',  icon: 'Radio', live: true },
  { id: 'geo',     label: 'Geo-analytics',    icon: 'Map' },
  { id: 'subs',    label: 'Subscriptions',    icon: 'CreditCard' },
  { id: 'audio',   label: 'Audio guides',     icon: 'Headphones' },
];

function Rail({ route, setRoute, accent }) {
  return (
    <aside className="flex flex-col flex-shrink-0" style={{ width: 232, background: '#103d3a', borderRight: '1px solid rgba(255,255,255,0.06)' }}>
      {/* brand */}
      <div className="flex items-center gap-2.5 px-5" style={{ height: 58, borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
        <svg width="26" height="26" viewBox="0 0 32 32" fill="none">
          <path d="M16 3C10.5 3 6 7.3 6 12.7 6 19.5 16 29 16 29s10-9.5 10-16.3C26 7.3 21.5 3 16 3z" fill={accent} />
          <circle cx="16" cy="12.5" r="4.4" fill="#f6b81e" />
        </svg>
        <div className="leading-none">
          <div className="font-bold text-[16px] tracking-tight text-white">Spotto</div>
          <div className="text-[9.5px] font-bold uppercase tracking-[0.18em] mt-0.5" style={{ color: accent }}>Command Center</div>
        </div>
      </div>

      <nav className="flex-1 px-3 py-4 space-y-1">
        <div className="px-3 pb-2 text-[10px] font-bold uppercase tracking-[0.15em]" style={{ color: 'rgba(255,255,255,0.3)' }}>Operations</div>
        {NAV_A.map((n) => {
          const on = route === n.id;
          return (
            <button key={n.id} onClick={() => setRoute(n.id)}
              className="w-full flex items-center gap-3 rounded-lg px-3 py-2.5 transition-all"
              style={{ background: on ? 'rgba(255,255,255,0.07)' : 'transparent', color: on ? '#fff' : 'rgba(255,255,255,0.55)',
                       boxShadow: on ? `inset 2px 0 0 ${accent}` : 'none' }}>
              <Icon name={n.icon} size={18} strokeWidth={on ? 2.4 : 2} style={{ color: on ? accent : 'inherit' }} />
              <span className="font-semibold text-[13.5px]">{n.label}</span>
              {n.badge && <span className="ml-auto rounded-full px-1.5 py-0.5 text-[10px] font-bold" style={{ background: accent, color: '#fff' }}>{n.badge}</span>}
              {n.live && <span className="ml-auto w-2 h-2 rounded-full" style={{ background: '#16a34a', boxShadow: '0 0 0 3px rgba(22,163,74,0.25)' }} />}
            </button>
          );
        })}
      </nav>

      {/* health footer */}
      <div className="p-3">
        <div className="rounded-lg p-3" style={{ background: 'rgba(255,255,255,0.04)' }}>
          <div className="flex items-center gap-2 text-[11px] font-bold" style={{ color: '#16a34a' }}>
            <Icon name="Activity" size={13} /> All systems operational
          </div>
          <div className="text-[10.5px] font-medium mt-1" style={{ color: 'rgba(255,255,255,0.4)' }}>API 42ms · 99.98% uptime</div>
        </div>
      </div>
    </aside>
  );
}

function CommandBar({ accent, route, onProfile }) {
  const { ADMIN_KPIS } = window;
  return (
    <header className="flex items-center gap-4 px-5 flex-shrink-0" style={{ height: 58, background: 'var(--surface)', borderBottom: '1px solid var(--line)' }}>
      <div className="flex items-center gap-2 rounded-lg px-3 py-1.5" style={{ background: 'var(--bg)', border: '1px solid var(--line)', width: 280 }}>
        <Icon name="Search" size={16} style={{ color: 'var(--muted)' }} />
        <input placeholder="Search vendors, content, IDs…" className="bg-transparent outline-none text-[13px] font-medium w-full" style={{ color: 'var(--ink)' }} />
        <kbd className="text-[10px] font-mono rounded px-1 py-0.5" style={{ background: 'var(--surface)', border: '1px solid var(--line)', color: 'var(--muted)' }}>⌘K</kbd>
      </div>

      {/* mini KPI strip */}
      <div className="hidden xl:flex items-center gap-1 ml-2">
        {ADMIN_KPIS.map((k) => {
          const c = { amber: '#d97706', red: '#e11d48', teal: accent, green: '#16a34a' }[k.tone];
          return (
            <div key={k.id} className="flex items-center gap-2 px-3 py-1 rounded-lg" title={k.label}>
              <span className="w-7 h-7 rounded-md flex items-center justify-center" style={{ background: c + '1a', color: c }}><Icon name={k.icon} size={14} /></span>
              <div className="leading-none">
                <div className="font-bold text-[14px]" style={{ color: 'var(--ink)' }}>{k.value}</div>
                <div className="text-[10px] font-semibold mt-0.5" style={{ color: 'var(--muted)' }}>{k.label}</div>
              </div>
            </div>
          );
        })}
      </div>

      <div className="ml-auto flex items-center gap-2">
        <button className="w-9 h-9 rounded-lg flex items-center justify-center relative" style={{ background: 'var(--bg)', border: '1px solid var(--line)', color: 'var(--ink-soft)' }}>
          <Icon name="Bell" size={17} /><span className="absolute top-1.5 right-1.5 w-2 h-2 rounded-full" style={{ background: '#e11d48' }} />
        </button>
        <button onClick={onProfile} className="flex items-center gap-2 rounded-lg px-2 py-1.5 transition-colors" style={{ background: route === 'profile' ? accent + '1a' : 'var(--bg)', border: `1px solid ${route === 'profile' ? accent : 'var(--line)'}` }}>
          <span className="w-7 h-7 rounded-md flex items-center justify-center text-[11px] font-bold text-white" style={{ background: '#103d3a' }}>AD</span>
          <div className="leading-none mr-1 text-left">
            <div className="font-bold text-[12px]" style={{ color: 'var(--ink)' }}>Admin</div>
            <div className="text-[10px] font-semibold" style={{ color: 'var(--muted)' }}>Super-admin</div>
          </div>
        </button>
      </div>
    </header>
  );
}

function AdminApp() {
  const [t, setTweak] = useTweaks(TWEAK_DEFAULTS_A);
  const pal = PALETTES_A[t.accent] || PALETTES_A['#0b6f86'];
  const [route, setRoute] = useStateAA('queue');
  const dark = t.theme === 'Dark';
  const [authed, setAuthed] = useStateAA(() => {
    try { return localStorage.getItem('spotto_admin_auth') === '1'; } catch { return false; }
  });
  function finishAuth() { setAuthed(true); try { localStorage.setItem('spotto_admin_auth', '1'); } catch {} }
  function logout() { try { localStorage.removeItem('spotto_admin_auth'); } catch {} setAuthed(false); }

  const cssVars = dark ? {
    '--bg': '#0b2320', '--surface': '#10302c', '--line': 'rgba(255,255,255,0.09)',
    '--ink': '#eaf2ef', '--ink-soft': '#c3d6d1', '--muted': '#7f9d97', '--accent': pal.accent,
  } : {
    '--bg': '#f7f2e8', '--surface': '#ffffff', '--line': '#e7e1d4',
    '--ink': '#103d3a', '--ink-soft': '#33524f', '--muted': '#6c8582', '--accent': pal.accent,
  };

  // keyboard shortcuts for the approval queue
  useEffectAA(() => {
    function onKey(e) {
      if (route !== 'queue') return;
      const tag = (e.target.tagName || '').toLowerCase();
      if (tag === 'input' || tag === 'textarea') return;
      const map = { a: 'approve', r: 'reject', e: 'edits' };
      const action = map[e.key.toLowerCase()];
      if (action) { const btn = document.querySelector(`[data-qa="${action}"]`); btn && btn.click(); }
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [route]);

  return (
    <div style={{ ...cssVars, position: 'fixed', inset: 0, display: 'flex', background: 'var(--bg)', fontFamily: "'IBM Plex Sans', system-ui", color: 'var(--ink)' }}>
      {!authed ? (
        <AdminAuth onAuthed={finishAuth} accent={pal.accent} layout={t.authLayout} />
      ) : (
      <>
      <Rail route={route} setRoute={setRoute} accent={pal.accent} />
      <div className="flex flex-col min-w-0 flex-1">
        <CommandBar accent={pal.accent} route={route} onProfile={() => setRoute('profile')} />
        <main className="flex-1 overflow-hidden" style={{ padding: 24 }}>
          {route === 'queue' && <VendorQueue accent={pal.accent} />}
          {route === 'monitor' && <ContentMonitor accent={pal.accent} />}
          {route === 'geo' && <GeoAnalytics accent={pal.accent} />}
          {route === 'subs' && <Subscriptions accent={pal.accent} />}
          {route === 'audio' && <AudioCMS accent={pal.accent} />}
          {route === 'profile' && <AdminProfile accent={pal.accent} onLogout={logout} />}
        </main>
      </div>
      </>
      )}

      <TweaksPanel>
        <TweakSection label="Appearance" />
        <TweakRadio label="Theme" value={t.theme} options={['Light', 'Dark']} onChange={(v) => setTweak('theme', v)} />
        <TweakColor label="Accent" value={t.accent}
          options={['#0b6f86', '#0e8c8a', '#0b4a54', '#c07a3c']}
          onChange={(v) => setTweak('accent', v)} />
        <TweakSection label="Sign-in" />
        <TweakRadio label="Layout" value={t.authLayout} options={['split', 'centered']} onChange={(v) => setTweak('authLayout', v)} />
        <TweakButton label="Log out" onClick={logout} secondary />
      </TweaksPanel>
    </div>
  );
}

ReactDOM.createRoot(document.getElementById('root')).render(<AdminApp />);
