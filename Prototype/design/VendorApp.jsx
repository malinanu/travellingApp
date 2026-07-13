// VendorApp.jsx — portal shell: sidebar + topbar + routing + tweaks.

const { useState: useStateVA, useEffect: useEffectVA } = React;

const PALETTES_V = {
  '#0b6f86': { accent: '#0b6f86', accent2: '#0e8c8a', tint: '#ddedef', shadow: 'rgba(11,111,134,0.35)' },
  '#0e8c8a': { accent: '#0e8c8a', accent2: '#2ba69a', tint: '#dcefec', shadow: 'rgba(14,140,138,0.35)' },
  '#0b4a54': { accent: '#0b4a54', accent2: '#0b6f86', tint: '#dbe8ea', shadow: 'rgba(11,74,84,0.35)' },
  '#c07a3c': { accent: '#c07a3c', accent2: '#d9974f', tint: '#f6e8d6', shadow: 'rgba(192,122,60,0.35)' },
};

const TWEAK_DEFAULTS_V = /*EDITMODE-BEGIN*/{
  "accent": "#0b6f86",
  "displayFont": "Fredoka",
  "density": "Comfortable",
  "authLayout": "split"
}/*EDITMODE-END*/;

const NAV = [
  { id: 'dashboard', label: 'Dashboard', icon: 'LayoutDashboard' },
  { id: 'offers',    label: 'Offers',    icon: 'Tag' },
  { id: 'menu',      label: 'Menu',      icon: 'UtensilsCrossed', foodOnly: true },
  { id: 'growth',    label: 'Growth',    icon: 'Rocket' },
  { id: 'validate',  label: 'Validate',  icon: 'QrCode' },
  { id: 'reports',   label: 'Reports',   icon: 'BarChart3' },
  { id: 'profile',   label: 'Profile',   icon: 'Store' },
];

function SpottoMarkV({ size = 26 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" fill="none">
      <path d="M16 3C10.5 3 6 7.3 6 12.7 6 19.5 16 29 16 29s10-9.5 10-16.3C26 7.3 21.5 3 16 3z" fill="var(--accent)" />
      <circle cx="16" cy="12.5" r="4.4" fill="var(--sun)" />
    </svg>
  );
}

function Sidebar({ route, setRoute, accent, open, setOpen, isFood }) {
  const { VENDOR } = window;
  return (
    <>
      {/* mobile overlay */}
      <div onClick={() => setOpen(false)} data-nav-overlay
        style={{ position: 'fixed', inset: 0, zIndex: 30, background: 'rgba(16,61,58,0.4)',
                 opacity: open ? 1 : 0, pointerEvents: open ? 'auto' : 'none', transition: 'opacity 200ms' }} />
      <aside className="flex flex-col flex-shrink-0"
        style={{ width: 248, background: '#fff', borderRight: '1px solid var(--line)',
                 position: 'fixed', top: 0, bottom: 0, left: 0, zIndex: 31,
                 transform: open ? 'translateX(0)' : 'translateX(-100%)',
                 transition: 'transform 240ms cubic-bezier(0.22,1,0.36,1)' }}
        data-sidebar>
        {/* brand */}
        <div className="flex items-center gap-2.5 px-5" style={{ height: 64, borderBottom: '1px solid var(--line)' }}>
          <SpottoMarkV size={28} />
          <div className="leading-none">
            <div className="font-extrabold text-[18px] tracking-tight" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Spotto</div>
            <div className="text-[10.5px] font-extrabold uppercase tracking-widest mt-0.5" style={{ color: accent }}>Vendor</div>
          </div>
        </div>
        {/* nav */}
        <nav className="flex-1 px-3 py-4 space-y-1">
          {NAV.filter((n) => !n.foodOnly || isFood).map((n) => {
            const on = route === n.id;
            return (
              <button key={n.id} onClick={() => { setRoute(n.id); setOpen(false); }}
                className="w-full flex items-center gap-3 rounded-xl px-3.5 py-2.5 transition-all"
                style={{ background: on ? 'var(--accent-tint)' : 'transparent', color: on ? accent : 'var(--ink-soft)' }}>
                <Icon name={n.icon} size={20} strokeWidth={on ? 2.5 : 2} />
                <span className="font-extrabold text-[14.5px]">{n.label}</span>
                {n.id === 'validate' && <span className="ml-auto text-[10px] font-extrabold rounded-full px-1.5 py-0.5" style={{ background: accent, color: '#fff' }}>SCAN</span>}
              </button>
            );
          })}
        </nav>
        {/* vendor card */}
        <div className="p-3">
          <button onClick={() => { setRoute('account'); setOpen(false); }} className="w-full rounded-xl p-3 flex items-center gap-3 text-left transition-colors" style={{ background: route === 'account' ? 'var(--accent-tint)' : 'var(--sand)' }}>
            <div className="w-10 h-10 rounded-lg flex items-center justify-center text-[14px] font-extrabold text-white flex-shrink-0"
                 style={{ background: accent }}>{VENDOR.initials}</div>
            <div className="min-w-0">
              <div className="font-extrabold text-[13px] truncate" style={{ color: 'var(--ink)' }}>{VENDOR.name}</div>
              <div className="text-[11.5px] font-semibold truncate" style={{ color: 'var(--muted)' }}>{VENDOR.area} · Verified</div>
            </div>
          </button>
        </div>
      </aside>
    </>
  );
}

// seed notifications for the bell dropdown
const NOTIFS_SEED = [
  { id: 'n1', icon: 'Ticket',     tone: 'accent', title: 'Lena redeemed "Free flat white"',        sub: 'Scanned at the counter',            time: '2 min ago',  unread: true },
  { id: 'n2', icon: 'Star',       tone: 'sun',    title: 'New 5-star review from Marco',           sub: '"Best pour-over on the south coast"', time: '38 min ago', unread: true },
  { id: 'n3', icon: 'Clock',      tone: 'sun',    title: '"Buy 5 day-passes" ends tonight',        sub: '47 of 50 claimed — almost gone',     time: '2 hr ago',   unread: true },
  { id: 'n4', icon: 'TrendingUp', tone: 'accent', title: 'Profile views up 12% this week',         sub: 'Your evening photos are working',    time: 'Yesterday',  unread: false },
  { id: 'n5', icon: 'Wallet',     tone: 'muted',  title: 'Premium plan renewed',                   sub: '$24.00 · receipt emailed',           time: '3 days ago', unread: false },
];

function NotifBell({ accent }) {
  const [open, setOpen] = useStateVA(false);
  const [notifs, setNotifs] = useStateVA(NOTIFS_SEED);
  const unread = notifs.filter((n) => n.unread).length;

  const toneStyle = (t) =>
    t === 'sun' ? { background: 'rgba(246,184,30,0.16)', color: '#c98a00' }
    : t === 'muted' ? { background: 'rgba(15,61,58,0.06)', color: 'var(--muted)' }
    : { background: 'var(--accent-tint)', color: accent };

  function markAll() { setNotifs(notifs.map((n) => ({ ...n, unread: false }))); }
  function markOne(id) { setNotifs(notifs.map((n) => n.id === id ? { ...n, unread: false } : n)); }

  return (
    <div style={{ position: 'relative' }}>
      <button onClick={() => setOpen((o) => !o)} aria-label="Notifications"
        className="w-10 h-10 rounded-xl flex items-center justify-center relative active:scale-95 transition-transform"
        style={{ background: open ? 'var(--accent-tint)' : '#fff', border: `1px solid ${open ? accent : 'var(--line)'}`, color: open ? accent : 'var(--ink)' }}>
        <Icon name="Bell" size={19} />
        {unread > 0 && <span className="absolute top-2 right-2.5 w-2 h-2 rounded-full" style={{ background: 'var(--sun)', boxShadow: '0 0 0 2px #fff' }} />}
      </button>

      {open && (
        <>
          {/* click-away backdrop */}
          <div onClick={() => setOpen(false)} style={{ position: 'fixed', inset: 0, zIndex: 40 }} />
          <div className="rounded-2xl bg-white overflow-hidden"
               style={{ position: 'absolute', right: 0, top: 48, zIndex: 41, width: 'min(360px, calc(100vw - 32px))',
                        border: '1px solid var(--line)', boxShadow: '0 24px 50px -20px rgba(16,61,58,0.35)' }}>
            {/* header */}
            <div className="flex items-center gap-2 px-4 py-3" style={{ borderBottom: '1px solid var(--line)' }}>
              <span className="font-extrabold text-[14.5px]" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Notifications</span>
              {unread > 0 && (
                <span className="rounded-full px-1.5 py-0.5 text-[10.5px] font-extrabold" style={{ background: 'var(--sun)', color: '#5a3b00' }}>{unread} new</span>
              )}
              {unread > 0 && (
                <button onClick={markAll} className="ml-auto text-[12px] font-extrabold" style={{ color: accent }}>Mark all read</button>
              )}
            </div>
            {/* list */}
            <div style={{ maxHeight: 380, overflowY: 'auto' }}>
              {notifs.map((n, i) => (
                <button key={n.id} onClick={() => markOne(n.id)}
                  className="w-full flex items-start gap-3 px-4 py-3 text-left transition-colors"
                  style={{ borderTop: i ? '1px solid var(--line)' : 'none', background: n.unread ? 'rgba(246,184,30,0.05)' : '#fff' }}>
                  <span className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0" style={toneStyle(n.tone)}>
                    <Icon name={n.icon} size={16} />
                  </span>
                  <span className="flex-1 min-w-0">
                    <span className="block font-extrabold text-[13px] leading-snug" style={{ color: 'var(--ink)' }}>{n.title}</span>
                    <span className="block text-[12px] font-semibold mt-0.5 truncate" style={{ color: 'var(--muted)' }}>{n.sub}</span>
                    <span className="block text-[10.5px] font-bold mt-1" style={{ color: 'var(--muted)', opacity: 0.8 }}>{n.time}</span>
                  </span>
                  {n.unread && <span className="w-2 h-2 rounded-full flex-shrink-0 mt-1.5" style={{ background: 'var(--sun)' }} />}
                </button>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  );
}

function Topbar({ onMenu, accent, route, onAccount }) {
  const title = NAV.find((n) => n.id === route)?.label || '';
  return (
    <header className="flex items-center gap-3 px-5 flex-shrink-0 sticky top-0 z-20"
            style={{ height: 64, background: 'rgba(251,246,237,0.85)', backdropFilter: 'blur(10px)', borderBottom: '1px solid var(--line)' }}>
      <button onClick={onMenu} className="lg:hidden w-9 h-9 rounded-lg flex items-center justify-center" style={{ color: 'var(--ink)' }}>
        <Icon name="Menu" size={22} />
      </button>
      <div className="hidden sm:flex items-center gap-2 rounded-xl px-3 py-2 flex-1 max-w-xs" style={{ background: '#fff', border: '1px solid var(--line)' }}>
        <Icon name="Search" size={17} style={{ color: 'var(--muted)' }} />
        <input placeholder="Search offers, codes…" className="bg-transparent outline-none text-[13.5px] font-semibold w-full" style={{ color: 'var(--ink)' }} />
      </div>
      <div className="flex-1 sm:hidden font-extrabold text-[17px]" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>{title}</div>
      <div className="ml-auto flex items-center gap-2">
        <NotifBell accent={accent} />
        <button onClick={onAccount} className="rounded-xl px-3 py-2 flex items-center gap-2" style={{ background: '#fff', border: `1px solid ${route === 'account' ? accent : 'var(--line)'}` }}>
          <span className="w-7 h-7 rounded-lg flex items-center justify-center text-[12px] font-extrabold text-white" style={{ background: accent }}>{window.VENDOR.initials}</span>
          <Icon name="ChevronDown" size={15} style={{ color: 'var(--muted)' }} />
        </button>
      </div>
    </header>
  );
}

function VendorApp() {
  const [t, setTweak] = useTweaks(TWEAK_DEFAULTS_V);
  const pal = PALETTES_V[t.accent] || PALETTES_V['#0b6f86'];
  const [route, setRoute] = useStateVA('dashboard');
  const [navOpen, setNavOpen] = useStateVA(false);
  const [offers, setOffers] = useStateVA(window.OFFERS);
  const [businessType, setBusinessType] = useStateVA(window.VENDOR.typeId);
  const [menu, setMenu] = useStateVA(window.MENU);
  const isFood = !!(window.BUSINESS_TYPES.find((b) => b.id === businessType) || {}).food;
  const [authed, setAuthed] = useStateVA(() => {
    try { return localStorage.getItem('spotto_vendor_auth') === '1'; } catch { return false; }
  });
  function finishAuth() { setAuthed(true); try { localStorage.setItem('spotto_vendor_auth', '1'); } catch {} }
  function logout() { try { localStorage.removeItem('spotto_vendor_auth'); } catch {} setAuthed(false); }

  useEffectVA(() => { const el = document.getElementById('scroll-main'); if (el) el.scrollTop = 0; }, [route]);

  const compact = t.density === 'Compact';
  const cssVars = {
    '--accent': pal.accent, '--accent-2': pal.accent2, '--accent-tint': pal.tint, '--accent-shadow': pal.shadow,
    '--sun': '#f6b81e', '--ink': '#103d3a', '--ink-soft': '#33524f', '--muted': '#6c8582',
    '--sand': '#fbf6ed', '--line': '#e7e1d4',
    '--font-display': `'${t.displayFont}', system-ui`,
  };

  return (
    <div style={{ ...cssVars, position: 'fixed', inset: 0, display: 'flex', background: 'var(--sand)', fontFamily: "'Nunito', system-ui" }}>
      {!authed ? (
        <VendorAuth onAuthed={finishAuth} layout={t.authLayout} />
      ) : (
      <>
      <div className="lg-static"><Sidebar route={route} setRoute={setRoute} accent={pal.accent} open={navOpen} setOpen={setNavOpen} isFood={isFood} /></div>

      <div className="flex flex-col min-w-0 flex-1" style={{ marginLeft: 0 }} data-content>
        <Topbar onMenu={() => setNavOpen(true)} accent={pal.accent} route={route} onAccount={() => setRoute('account')} />
        <main id="scroll-main" className="flex-1 overflow-y-auto">
          <div style={{ maxWidth: 1180, margin: '0 auto', padding: compact ? '20px 20px 60px' : '28px 28px 80px' }}>
            {route === 'dashboard' && <VendorDashboard accent={pal.accent} onNav={setRoute} />}
            {route === 'offers' && <OfferCreator accent={pal.accent} offers={offers} setOffers={setOffers} />}
            {route === 'menu' && <MenuManager accent={pal.accent} menu={menu} setMenu={setMenu} isFood={isFood} onNav={setRoute} />}
            {route === 'growth' && <GrowthTools accent={pal.accent} />}
            {route === 'validate' && <QRValidator accent={pal.accent} />}
            {route === 'reports' && <Reports accent={pal.accent} />}
            {route === 'profile' && <ProfileEditor accent={pal.accent} businessType={businessType} setBusinessType={setBusinessType} />}
            {route === 'account' && <AccountProfile accent={pal.accent} onLogout={logout} />}
          </div>
        </main>
      </div>
      </>
      )}

      <TweaksPanel>
        <TweakSection label="Brand" />
        <TweakColor label="Accent" value={t.accent}
          options={['#0b6f86', '#0e8c8a', '#0b4a54', '#c07a3c']}
          onChange={(v) => setTweak('accent', v)} />
        <TweakSelect label="Heading font" value={t.displayFont}
          options={['Fredoka', 'Baloo 2', 'Quicksand', 'Nunito']}
          onChange={(v) => setTweak('displayFont', v)} />
        <TweakSection label="Layout" />
        <TweakRadio label="Density" value={t.density} options={['Comfortable', 'Compact']} onChange={(v) => setTweak('density', v)} />
        <TweakSection label="Sign-in" />
        <TweakRadio label="Layout" value={t.authLayout} options={['split', 'centered']} onChange={(v) => setTweak('authLayout', v)} />
        <TweakButton label="Log out" onClick={logout} secondary />
      </TweaksPanel>
    </div>
  );
}

ReactDOM.createRoot(document.getElementById('root')).render(<VendorApp />);
