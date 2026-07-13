// AdminProfile.jsx — Command Center operator account & security page.
// Rendered as a routed main view. Respects --bg/--surface/--line/--ink/--accent (light+dark).

const { useState: useStateAdProf } = React;

function AdpCard({ title, sub, icon, accent, children, action }) {
  return (
    <div className="rounded-xl" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
      <div className="flex items-center gap-3 px-5 py-4" style={{ borderBottom: '1px solid var(--line)' }}>
        {icon && <span className="w-8 h-8 rounded-lg flex items-center justify-center" style={{ background: accent + '1a', color: accent }}><Icon name={icon} size={16} /></span>}
        <div className="min-w-0 flex-1">
          <h3 className="font-bold text-[14.5px]" style={{ color: 'var(--ink)' }}>{title}</h3>
          {sub && <p className="text-[12px] font-medium" style={{ color: 'var(--muted)' }}>{sub}</p>}
        </div>
        {action}
      </div>
      <div className="p-5">{children}</div>
    </div>
  );
}

function AdpField({ label, value, onChange, icon, type = 'text', readOnly }) {
  return (
    <label className="block">
      <span className="block text-[10.5px] font-bold uppercase tracking-[0.12em] mb-1.5" style={{ color: 'var(--muted)' }}>{label}</span>
      <div className="flex items-center gap-2.5 rounded-lg px-3" style={{ height: 42, background: 'var(--bg)', border: '1px solid var(--line)' }}>
        {icon && <Icon name={icon} size={15} style={{ color: 'var(--muted)' }} />}
        <input type={type} value={value} onChange={(e) => onChange?.(e.target.value)} readOnly={readOnly}
          className="flex-1 bg-transparent outline-none text-[13.5px] font-medium w-full" style={{ color: 'var(--ink)', cursor: readOnly ? 'default' : 'text' }} />
        {readOnly && <Icon name="Lock" size={13} style={{ color: 'var(--muted)' }} />}
      </div>
    </label>
  );
}

function AdpToggle({ on, onClick, accent }) {
  return (
    <button onClick={onClick} className="relative rounded-full transition-colors flex-shrink-0"
      style={{ width: 40, height: 23, background: on ? accent : 'var(--line)' }}>
      <span className="absolute top-[3px] rounded-full bg-white transition-all" style={{ width: 17, height: 17, left: on ? 20 : 3, boxShadow: '0 1px 2px rgba(0,0,0,0.3)' }} />
    </button>
  );
}

function AdminProfile({ accent, onLogout }) {
  const [name, setName] = useStateAdProf('Asha Dilshan');
  const [email, setEmail] = useStateAdProf('asha@spotto.app');
  const [twofa, setTwofa] = useStateAdProf(true);
  const [notif, setNotif] = useStateAdProf({ queue: true, flagged: true, weekly: false, login: true });
  const [saved, setSaved] = useStateAdProf(false);

  const sessions = [
    { id: 's1', device: 'MacBook Pro · Chrome', loc: 'Colombo, LK', when: 'Active now', current: true },
    { id: 's2', device: 'iPhone 15 · Spotto app', loc: 'Colombo, LK', when: '2 hours ago', current: false },
    { id: 's3', device: 'Windows · Edge', loc: 'Galle, LK', when: '3 days ago', current: false },
  ];

  function save() { setSaved(true); setTimeout(() => setSaved(false), 1800); }

  return (
    <div className="h-full overflow-y-auto" style={{ margin: -24, padding: 24 }}>
      <div style={{ maxWidth: 900, margin: '0 auto' }}>
        {/* header */}
        <div className="flex items-end justify-between gap-4 mb-6">
          <div>
            <h1 className="font-bold tracking-tight" style={{ fontSize: 24, color: 'var(--ink)' }}>Account & security</h1>
            <p className="text-[13px] font-medium mt-1" style={{ color: 'var(--muted)' }}>Manage your operator profile, sign-in and alerts.</p>
          </div>
          <button onClick={save} className="rounded-lg px-4 py-2.5 font-bold text-[13px] flex items-center gap-2 text-white whitespace-nowrap active:scale-[0.98] transition-transform"
            style={{ background: saved ? '#16a34a' : accent, boxShadow: `0 8px 18px -10px ${accent}` }}>
            {saved ? <><Icon name="Check" size={15} strokeWidth={3} /> Saved</> : <><Icon name="Save" size={15} /> Save changes</>}
          </button>
        </div>

        {/* identity hero */}
        <div className="rounded-xl p-5 mb-5 flex items-center gap-4" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
          <div className="relative flex-shrink-0">
            <div className="w-16 h-16 rounded-2xl flex items-center justify-center font-bold text-white text-[22px]" style={{ background: '#103d3a' }}>AD</div>
            <button className="absolute -bottom-1 -right-1 w-7 h-7 rounded-lg flex items-center justify-center" style={{ background: accent, color: '#fff', border: '2px solid var(--surface)' }}><Icon name="Camera" size={13} /></button>
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <h2 className="font-bold text-[18px] truncate" style={{ color: 'var(--ink)' }}>{name}</h2>
              <span className="text-[10.5px] font-bold uppercase tracking-wide px-2 py-0.5 rounded-full whitespace-nowrap" style={{ background: accent + '1a', color: accent }}>Super-admin</span>
            </div>
            <p className="text-[13px] font-medium mt-0.5" style={{ color: 'var(--muted)' }}>{email} · South-coast operations</p>
          </div>
          <div className="hidden sm:flex items-center gap-1.5 text-[11.5px] font-bold px-3 py-1.5 rounded-lg" style={{ background: 'rgba(22,163,74,0.1)', color: '#16a34a' }}>
            <span className="w-1.5 h-1.5 rounded-full" style={{ background: '#16a34a' }} /> Verified
          </div>
        </div>

        <div className="grid gap-5" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(380px, 1fr))' }}>
          {/* details */}
          <AdpCard title="Profile details" sub="Your name & contact" icon="User" accent={accent}>
            <div className="space-y-3.5">
              <AdpField label="Full name" value={name} onChange={setName} icon="User" />
              <AdpField label="Work email" value={email} onChange={setEmail} icon="Mail" type="email" />
              <div className="grid grid-cols-2 gap-3">
                <AdpField label="Role" value="Super-admin" readOnly icon="ShieldCheck" />
                <AdpField label="Region" value="South coast" onChange={() => {}} icon="Map" />
              </div>
            </div>
          </AdpCard>

          {/* security */}
          <AdpCard title="Security" sub="Password & two-factor" icon="Lock" accent={accent}>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <div className="font-bold text-[13.5px]" style={{ color: 'var(--ink)' }}>Password</div>
                  <div className="text-[12px] font-medium" style={{ color: 'var(--muted)' }}>Last changed 42 days ago</div>
                </div>
                <button className="rounded-lg px-3 py-2 text-[12.5px] font-bold" style={{ background: 'var(--bg)', border: '1px solid var(--line)', color: 'var(--ink)' }}>Change</button>
              </div>
              <div className="flex items-center justify-between pt-3" style={{ borderTop: '1px solid var(--line)' }}>
                <div>
                  <div className="font-bold text-[13.5px] flex items-center gap-1.5 whitespace-nowrap" style={{ color: 'var(--ink)' }}>Two-factor {twofa && <span className="text-[10px] font-bold px-1.5 py-0.5 rounded" style={{ background: 'rgba(22,163,74,0.12)', color: '#16a34a' }}>ON</span>}</div>
                  <div className="text-[12px] font-medium" style={{ color: 'var(--muted)' }}>Authenticator app · required at sign-in</div>
                </div>
                <AdpToggle on={twofa} onClick={() => setTwofa(!twofa)} accent={accent} />
              </div>
              <div className="flex items-center justify-between pt-3" style={{ borderTop: '1px solid var(--line)' }}>
                <div>
                  <div className="font-bold text-[13.5px]" style={{ color: 'var(--ink)' }}>Recovery codes</div>
                  <div className="text-[12px] font-medium" style={{ color: 'var(--muted)' }}>8 unused codes</div>
                </div>
                <button className="rounded-lg px-3 py-2 text-[12.5px] font-bold" style={{ background: 'var(--bg)', border: '1px solid var(--line)', color: 'var(--ink)' }}>View</button>
              </div>
            </div>
          </AdpCard>

          {/* sessions */}
          <AdpCard title="Active sessions" sub="Devices signed into your account" icon="MonitorSmartphone" accent={accent}
            action={<button className="text-[12px] font-bold" style={{ color: '#e11d48' }}>Revoke all</button>}>
            <div className="space-y-2.5">
              {sessions.map((s) => (
                <div key={s.id} className="flex items-center gap-3 rounded-lg px-3 py-2.5" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
                  <span className="w-8 h-8 rounded-lg flex items-center justify-center" style={{ background: 'var(--surface)', color: 'var(--ink-soft)' }}><Icon name={s.device.includes('iPhone') ? 'Smartphone' : 'Monitor'} size={15} /></span>
                  <div className="min-w-0 flex-1">
                    <div className="font-bold text-[12.5px] truncate" style={{ color: 'var(--ink)' }}>{s.device}</div>
                    <div className="text-[11px] font-medium" style={{ color: 'var(--muted)' }}>{s.loc} · {s.when}</div>
                  </div>
                  {s.current ? <span className="text-[10.5px] font-bold px-2 py-0.5 rounded" style={{ background: accent + '1a', color: accent }}>This device</span>
                    : <button className="text-[11.5px] font-bold" style={{ color: '#e11d48' }}>Revoke</button>}
                </div>
              ))}
            </div>
          </AdpCard>

          {/* notifications */}
          <AdpCard title="Notifications" sub="What pings you" icon="Bell" accent={accent}>
            <div className="space-y-1">
              {[
                { k: 'queue', title: 'Approval queue', sub: 'New vendor submissions' },
                { k: 'flagged', title: 'Flagged content', sub: 'Auto-moderation escalations' },
                { k: 'weekly', title: 'Weekly report', sub: 'Platform summary every Monday' },
                { k: 'login', title: 'Sign-in alerts', sub: 'New device or location' },
              ].map((n, i) => (
                <div key={n.k} className="flex items-center justify-between py-2.5" style={{ borderTop: i ? '1px solid var(--line)' : 'none' }}>
                  <div>
                    <div className="font-bold text-[13.5px]" style={{ color: 'var(--ink)' }}>{n.title}</div>
                    <div className="text-[12px] font-medium" style={{ color: 'var(--muted)' }}>{n.sub}</div>
                  </div>
                  <AdpToggle on={notif[n.k]} onClick={() => setNotif((p) => ({ ...p, [n.k]: !p[n.k] }))} accent={accent} />
                </div>
              ))}
            </div>
          </AdpCard>
        </div>

        {/* sign out */}
        <div className="mt-5 rounded-xl p-4 flex items-center justify-between" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
          <div className="flex items-center gap-3">
            <span className="w-9 h-9 rounded-lg flex items-center justify-center" style={{ background: 'rgba(225,29,72,0.1)', color: '#e11d48' }}><Icon name="LogOut" size={17} /></span>
            <div>
              <div className="font-bold text-[13.5px]" style={{ color: 'var(--ink)' }}>Sign out</div>
              <div className="text-[12px] font-medium" style={{ color: 'var(--muted)' }}>End your session on this device</div>
            </div>
          </div>
          <button onClick={onLogout} className="rounded-lg px-4 py-2.5 font-bold text-[13px] text-white active:scale-[0.98] transition-transform" style={{ background: '#e11d48' }}>Sign out</button>
        </div>
      </div>
    </div>
  );
}

Object.assign(window, { AdminProfile });
