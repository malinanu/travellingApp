// AccountProfile.jsx — Vendor Portal OWNER account page (distinct from business profile).
// Light/sand theme. Rendered as the 'account' route. Props: accent, onLogout.

const { useState: useStateAcc } = React;

function AccCard({ title, sub, icon, accent, children, action }) {
  return (
    <div className="rounded-2xl bg-white" style={{ border: '1px solid var(--line)' }}>
      <div className="flex items-center gap-2.5 px-5 py-4" style={{ borderBottom: '1px solid var(--line)' }}>
        {icon && <span className="w-9 h-9 rounded-xl flex items-center justify-center" style={{ background: 'var(--accent-tint)', color: accent }}><Icon name={icon} size={18} /></span>}
        <div className="min-w-0 flex-1">
          <h3 className="font-extrabold text-[15px]" style={{ color: 'var(--ink)' }}>{title}</h3>
          {sub && <p className="text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>{sub}</p>}
        </div>
        {action}
      </div>
      <div className="p-5">{children}</div>
    </div>
  );
}

function AccField({ label, value, onChange, icon, type = 'text', prefix }) {
  return (
    <label className="block">
      <span className="block text-[12px] font-extrabold mb-1.5" style={{ color: 'var(--ink-soft)' }}>{label}</span>
      <div className="flex items-center gap-2.5 rounded-xl px-3.5" style={{ height: 48, background: 'var(--sand)', border: '1.5px solid var(--line)' }}>
        {icon && <Icon name={icon} size={17} style={{ color: 'var(--muted)' }} />}
        {prefix && <span className="font-extrabold text-[14px]" style={{ color: 'var(--ink-soft)' }}>{prefix}</span>}
        <input type={type} value={value} onChange={(e) => onChange?.(e.target.value)}
          className="flex-1 bg-transparent outline-none text-[14.5px] font-bold w-full" style={{ color: 'var(--ink)' }} />
      </div>
    </label>
  );
}

function AccToggle({ on, onClick, accent }) {
  return (
    <button onClick={onClick} className="relative rounded-full transition-colors flex-shrink-0"
      style={{ width: 46, height: 27, background: on ? accent : 'rgba(15,61,58,0.18)' }}>
      <span className="absolute top-1 rounded-full bg-white transition-all" style={{ width: 19, height: 19, left: on ? 23 : 4, boxShadow: '0 1px 3px rgba(0,0,0,0.2)' }} />
    </button>
  );
}

// ── team management ────────────────────────────────────────────
const TEAM_SEED = [
  { id: 't1', name: 'Tharindu Perera', email: 'tharindu@driftwood.lk', role: 'owner',   color: '#0b6f86' },
  { id: 't2', name: 'Nadeesha Fonseka', email: 'nadeesha@driftwood.lk', role: 'manager', color: '#c07a3c' },
  { id: 't3', name: 'Kasun Silva',      email: 'kasun@driftwood.lk',    role: 'staff',   color: '#5a5f9e' },
];
const INVITES_SEED = [
  { id: 'i1', email: 'amaya@driftwood.lk', role: 'staff', sent: '2 days ago' },
];

const ROLE_META = {
  owner:   { label: 'Owner',   desc: 'Full access, billing & team' },
  manager: { label: 'Manager', desc: 'Create & edit offers, view reports, scan' },
  staff:   { label: 'Staff',   desc: 'Scan & validate QR codes at the counter' },
};

function TeamAvatar({ name, color, size = 38 }) {
  const initials = name.split(' ').map((w) => w[0]).join('').slice(0, 2).toUpperCase();
  return (
    <span className="rounded-xl flex items-center justify-center font-extrabold text-white flex-shrink-0"
      style={{ width: size, height: size, fontSize: size * 0.34, background: color }}>{initials}</span>
  );
}

function RoleChip({ role, accent }) {
  const owner = role === 'owner';
  return (
    <span className="text-[10.5px] font-extrabold uppercase tracking-wide px-2 py-0.5 rounded-full whitespace-nowrap"
      style={{ background: owner ? 'var(--accent-tint)' : 'var(--sand)', color: owner ? accent : 'var(--ink-soft)',
               border: owner ? 'none' : '1px solid var(--line)' }}>
      {ROLE_META[role].label}
    </span>
  );
}

function TeamPanel({ accent, team, setTeam, invites, setInvites, onClose }) {
  const [email, setEmail] = useStateAcc('');
  const [role, setRole] = useStateAcc('staff');
  const [justSent, setJustSent] = useStateAcc(false);
  const valid = /^\S+@\S+\.\S+$/.test(email);

  function sendInvite() {
    if (!valid) return;
    setInvites((v) => [{ id: 'i' + Date.now(), email: email.trim(), role, sent: 'just now' }, ...v]);
    setEmail(''); setJustSent(true);
    setTimeout(() => setJustSent(false), 1800);
  }
  function revoke(id) { setInvites((v) => v.filter((i) => i.id !== id)); }
  function remove(id) { setTeam((t) => t.filter((m) => m.id !== id)); }

  return (
    <>
      <div onClick={onClose} style={{ position: 'fixed', inset: 0, zIndex: 60, background: 'rgba(16,61,58,0.4)', backdropFilter: 'blur(1.5px)' }} />
      <div className="flex flex-col" style={{ position: 'fixed', top: 0, right: 0, bottom: 0, zIndex: 61, width: 'min(430px, 100vw)',
        background: 'var(--sand)', boxShadow: '-24px 0 60px -30px rgba(16,61,58,0.45)' }}>
        {/* header */}
        <div className="flex items-center gap-3 px-5 bg-white" style={{ height: 64, borderBottom: '1px solid var(--line)', flexShrink: 0 }}>
          <span className="w-9 h-9 rounded-xl flex items-center justify-center" style={{ background: 'var(--accent-tint)', color: accent }}><Icon name="Users" size={18} /></span>
          <div className="flex-1 min-w-0">
            <h3 className="font-extrabold text-[16px]" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Your team</h3>
            <p className="text-[12px] font-semibold" style={{ color: 'var(--muted)' }}>{window.VENDOR.name} · {team.length} of 5 seats used</p>
          </div>
          <button onClick={onClose} className="w-9 h-9 rounded-xl flex items-center justify-center" style={{ background: 'var(--sand)', border: '1px solid var(--line)', color: 'var(--ink)' }}>
            <Icon name="X" size={17} strokeWidth={2.5} />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-5 space-y-5">
          {/* invite form */}
          <div className="rounded-2xl bg-white p-4" style={{ border: '1px solid var(--line)' }}>
            <div className="font-extrabold text-[14px] mb-3" style={{ color: 'var(--ink)' }}>Invite a team member</div>
            <div className="flex items-center gap-2.5 rounded-xl px-3.5" style={{ height: 48, background: 'var(--sand)', border: '1.5px solid var(--line)' }}>
              <Icon name="Mail" size={17} style={{ color: 'var(--muted)' }} />
              <input type="email" value={email} placeholder="name@email.com"
                onChange={(e) => setEmail(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && sendInvite()}
                className="flex-1 bg-transparent outline-none text-[14.5px] font-bold w-full" style={{ color: 'var(--ink)' }} />
            </div>
            {/* role picker */}
            <div className="flex gap-2 mt-3">
              {['staff', 'manager'].map((r) => {
                const on = role === r;
                return (
                  <button key={r} onClick={() => setRole(r)}
                    className="flex-1 rounded-xl px-3 py-2.5 text-left transition-all active:scale-[0.98]"
                    style={{ background: on ? 'var(--accent-tint)' : 'var(--sand)',
                             border: `1.5px solid ${on ? accent : 'var(--line)'}` }}>
                    <div className="flex items-center gap-1.5">
                      <Icon name={r === 'staff' ? 'QrCode' : 'Settings2'} size={14} style={{ color: on ? accent : 'var(--muted)' }} />
                      <span className="font-extrabold text-[13px]" style={{ color: on ? accent : 'var(--ink)' }}>{ROLE_META[r].label}</span>
                    </div>
                    <div className="text-[11px] font-semibold mt-1 leading-snug" style={{ color: 'var(--muted)' }}>{ROLE_META[r].desc}</div>
                  </button>
                );
              })}
            </div>
            <button onClick={sendInvite} disabled={!valid}
              className="w-full mt-3 rounded-xl py-3 font-extrabold text-[14px] flex items-center justify-center gap-2 text-white transition-all active:scale-[0.98]"
              style={{ background: justSent ? '#16a34a' : accent, opacity: valid || justSent ? 1 : 0.45,
                       boxShadow: valid ? '0 8px 18px -8px var(--accent-shadow)' : 'none' }}>
              {justSent ? <><Icon name="MailCheck" size={17} strokeWidth={2.5} /> Invite sent</> : <><Icon name="Send" size={16} /> Send invite</>}
            </button>
          </div>

          {/* pending invites */}
          {invites.length > 0 && (
            <div>
              <div className="text-[11.5px] font-extrabold uppercase tracking-wide mb-2 px-1" style={{ color: 'var(--muted)' }}>Pending · {invites.length}</div>
              <div className="rounded-2xl bg-white overflow-hidden" style={{ border: '1px dashed var(--line)' }}>
                {invites.map((inv, i) => (
                  <div key={inv.id} className="flex items-center gap-3 px-4 py-3" style={{ borderTop: i ? '1px solid var(--line)' : 'none' }}>
                    <span className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0" style={{ background: 'rgba(246,184,30,0.16)', color: '#c98a00' }}>
                      <Icon name="Clock" size={16} />
                    </span>
                    <div className="flex-1 min-w-0">
                      <div className="font-extrabold text-[13.5px] truncate" style={{ color: 'var(--ink)' }}>{inv.email}</div>
                      <div className="text-[11.5px] font-semibold" style={{ color: 'var(--muted)' }}>{ROLE_META[inv.role].label} · invited {inv.sent}</div>
                    </div>
                    <button className="text-[12px] font-extrabold whitespace-nowrap" style={{ color: accent }}>Resend</button>
                    <button onClick={() => revoke(inv.id)} className="w-7 h-7 rounded-lg flex items-center justify-center" style={{ color: 'var(--muted)' }} title="Revoke invite">
                      <Icon name="X" size={15} />
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* members */}
          <div>
            <div className="text-[11.5px] font-extrabold uppercase tracking-wide mb-2 px-1" style={{ color: 'var(--muted)' }}>Members · {team.length}</div>
            <div className="rounded-2xl bg-white overflow-hidden" style={{ border: '1px solid var(--line)' }}>
              {team.map((m, i) => (
                <div key={m.id} className="flex items-center gap-3 px-4 py-3" style={{ borderTop: i ? '1px solid var(--line)' : 'none' }}>
                  <TeamAvatar name={m.name} color={m.color} />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-extrabold text-[13.5px] truncate" style={{ color: 'var(--ink)' }}>{m.name}</span>
                      <RoleChip role={m.role} accent={accent} />
                    </div>
                    <div className="text-[11.5px] font-semibold truncate" style={{ color: 'var(--muted)' }}>{m.email}</div>
                  </div>
                  {m.role !== 'owner' && (
                    <button onClick={() => remove(m.id)} className="w-7 h-7 rounded-lg flex items-center justify-center" style={{ color: 'var(--muted)' }} title="Remove member">
                      <Icon name="Trash2" size={15} />
                    </button>
                  )}
                </div>
              ))}
            </div>
          </div>

          {/* permissions hint */}
          <div className="rounded-2xl p-4 flex items-start gap-3" style={{ background: 'var(--accent-tint)' }}>
            <Icon name="ShieldCheck" size={18} style={{ color: accent, flexShrink: 0, marginTop: 1 }} />
            <p className="text-[12.5px] font-semibold leading-snug" style={{ color: 'var(--ink-soft)' }}>
              Staff can only validate QR codes. Managers can also create and edit offers and view reports. Only you can change billing or the team.
            </p>
          </div>
        </div>
      </div>
    </>
  );
}

function AccountProfile({ accent, onLogout }) {
  const { VENDOR } = window;
  const [name, setName] = useStateAcc(VENDOR.owner || 'Tharindu Perera');
  const [email, setEmail] = useStateAcc('tharindu@driftwood.lk');
  const [phone, setPhone] = useStateAcc('76 123 4891');
  const [twofa, setTwofa] = useStateAcc(false);
  const [notif, setNotif] = useStateAcc({ redeem: true, reviews: true, payout: true, tips: false });
  const [saved, setSaved] = useStateAcc(false);
  const [showTeam, setShowTeam] = useStateAcc(false);
  const [team, setTeam] = useStateAcc(TEAM_SEED);
  const [invites, setInvites] = useStateAcc(INVITES_SEED);

  const initials = name.split(' ').map((w) => w[0]).join('').slice(0, 2).toUpperCase();
  function save() { setSaved(true); setTimeout(() => setSaved(false), 1800); }

  return (
    <div className="space-y-6">
      {/* header */}
      <div className="flex items-end justify-between gap-4">
        <div className="min-w-0">
          <h1 className="font-extrabold tracking-tight" style={{ fontSize: 28, lineHeight: 1.15, color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>My account</h1>
          <p className="text-[14.5px] font-semibold mt-1" style={{ color: 'var(--muted)' }}>Your personal login, security and alerts.</p>
        </div>
        <button onClick={save}
          className="rounded-xl px-5 py-2.5 font-extrabold text-[14px] flex items-center gap-2 text-white active:scale-[0.98] transition-transform flex-shrink-0 whitespace-nowrap"
          style={{ background: saved ? '#16a34a' : accent, boxShadow: '0 8px 18px -8px var(--accent-shadow)' }}>
          {saved ? <><Icon name="Check" size={17} strokeWidth={3} /> Saved</> : <><Icon name="Save" size={17} /> Save changes</>}
        </button>
      </div>

      {/* identity hero */}
      <div className="rounded-2xl bg-white p-5 flex items-center gap-4" style={{ border: '1px solid var(--line)' }}>
        <div className="relative flex-shrink-0">
          <div className="w-16 h-16 rounded-2xl flex items-center justify-center font-extrabold text-white text-[22px]" style={{ background: `linear-gradient(140deg, ${accent}, var(--accent-2))` }}>{initials}</div>
          <button className="absolute -bottom-1 -right-1 w-7 h-7 rounded-lg flex items-center justify-center" style={{ background: '#fff', color: accent, boxShadow: '0 3px 10px rgba(15,61,58,0.2)', border: '1.5px solid var(--accent-tint)' }}><Icon name="Camera" size={13} /></button>
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <h2 className="font-extrabold text-[18px] truncate" style={{ color: 'var(--ink)' }}>{name}</h2>
            <span className="text-[11px] font-extrabold uppercase tracking-wide px-2 py-0.5 rounded-full whitespace-nowrap flex-shrink-0" style={{ background: 'var(--accent-tint)', color: accent }}>Owner</span>
          </div>
          <p className="text-[13.5px] font-semibold mt-0.5 truncate" style={{ color: 'var(--muted)' }}>{VENDOR.name} · {VENDOR.area}</p>
        </div>
        <div className="hidden sm:flex items-center gap-1.5 text-[11.5px] font-extrabold px-3 py-1.5 rounded-lg whitespace-nowrap" style={{ background: 'rgba(22,163,74,0.1)', color: '#16a34a' }}>
          <Icon name="BadgeCheck" size={14} /> Verified
        </div>
      </div>

      <div className="grid gap-6" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(340px, 100%), 1fr))' }}>
        {/* personal details */}
        <AccCard title="Personal details" sub="Your contact info" icon="User" accent={accent}>
          <div className="space-y-3.5">
            <AccField label="Full name" value={name} onChange={setName} icon="User" />
            <AccField label="Email" value={email} onChange={setEmail} icon="Mail" type="email" />
            <AccField label="Phone" value={phone} onChange={setPhone} icon="Phone" prefix="+94" />
          </div>
        </AccCard>

        {/* security */}
        <AccCard title="Sign-in & security" sub="Keep your account safe" icon="Lock" accent={accent}>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <div>
                <div className="font-extrabold text-[14px]" style={{ color: 'var(--ink)' }}>Password</div>
                <div className="text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>Last changed 3 months ago</div>
              </div>
              <button className="rounded-xl px-3.5 py-2 text-[13px] font-extrabold" style={{ background: 'var(--sand)', border: '1.5px solid var(--line)', color: 'var(--ink)' }}>Change</button>
            </div>
            <div className="flex items-center justify-between pt-3.5" style={{ borderTop: '1px solid var(--line)' }}>
              <div className="min-w-0 pr-3">
                <div className="font-extrabold text-[14px]" style={{ color: 'var(--ink)' }}>Two-factor auth</div>
                <div className="text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>Add an extra step at login</div>
              </div>
              <AccToggle on={twofa} onClick={() => setTwofa(!twofa)} accent={accent} />
            </div>
          </div>
        </AccCard>

        {/* notifications */}
        <AccCard title="Notifications" sub="What we email & push you" icon="Bell" accent={accent}>
          <div className="space-y-1">
            {[
              { k: 'redeem', icon: 'Ticket', title: 'Redemption alerts', sub: 'When a traveler claims an offer' },
              { k: 'reviews', icon: 'Star', title: 'New reviews', sub: 'When someone rates your spot' },
              { k: 'payout', icon: 'Wallet', title: 'Payout updates', sub: 'Subscription & billing changes' },
              { k: 'tips', icon: 'Lightbulb', title: 'Growth tips', sub: 'Occasional marketing ideas' },
            ].map((n, i) => (
              <div key={n.k} className="flex items-center gap-3 py-2.5" style={{ borderTop: i ? '1px solid var(--line)' : 'none' }}>
                <span className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0" style={{ background: 'var(--accent-tint)', color: accent }}><Icon name={n.icon} size={17} /></span>
                <div className="flex-1 min-w-0">
                  <div className="font-extrabold text-[14px]" style={{ color: 'var(--ink)' }}>{n.title}</div>
                  <div className="text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>{n.sub}</div>
                </div>
                <AccToggle on={notif[n.k]} onClick={() => setNotif((p) => ({ ...p, [n.k]: !p[n.k] }))} accent={accent} />
              </div>
            ))}
          </div>
        </AccCard>

        {/* plan + team */}
        <AccCard title="Plan & team" sub="Subscription and access" icon="Sparkles" accent={accent}>
          <div className="rounded-xl p-3.5 flex items-center gap-3 mb-3" style={{ background: 'var(--accent-tint)' }}>
            <span className="w-10 h-10 rounded-xl flex items-center justify-center text-white flex-shrink-0" style={{ background: accent }}><Icon name="Crown" size={19} /></span>
            <div className="flex-1 min-w-0">
              <div className="font-extrabold text-[14px]" style={{ color: 'var(--ink)' }}>Premium plan</div>
              <div className="text-[12.5px] font-semibold" style={{ color: 'var(--ink-soft)' }}>Renews Jul 1 · $24/mo</div>
            </div>
            <button className="rounded-lg px-3 py-1.5 text-[12.5px] font-extrabold text-white whitespace-nowrap" style={{ background: accent }}>Manage</button>
          </div>
          <button onClick={() => setShowTeam(true)}
            className="w-full flex items-center gap-3 rounded-xl px-3.5 py-3 text-left active:scale-[0.99] transition-transform"
            style={{ background: 'var(--sand)', border: '1.5px solid var(--line)' }}>
            <span className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0" style={{ background: '#fff', color: accent }}><Icon name="UserPlus" size={17} /></span>
            <div className="flex-1 min-w-0">
              <div className="font-extrabold text-[13.5px]" style={{ color: 'var(--ink)' }}>Invite a team member</div>
              <div className="text-[12px] font-semibold" style={{ color: 'var(--muted)' }}>
                {team.length} members{invites.length ? ` · ${invites.length} pending` : ''} · staff scan & manage offers
              </div>
            </div>
            <span className="flex -space-x-2 items-center flex-shrink-0 mr-1">
              {team.slice(0, 3).map((m) => (
                <span key={m.id} style={{ border: '2px solid #fff', borderRadius: 10, display: 'inline-flex' }}>
                  <TeamAvatar name={m.name} color={m.color} size={26} />
                </span>
              ))}
            </span>
            <Icon name="ChevronRight" size={18} style={{ color: 'var(--muted)' }} />
          </button>
        </AccCard>
      </div>

      {/* sign out */}
      <div className="rounded-2xl bg-white p-4 flex items-center justify-between" style={{ border: '1px solid var(--line)' }}>
        <div className="flex items-center gap-3">
          <span className="w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0" style={{ background: 'rgba(225,29,72,0.1)', color: '#e11d48' }}><Icon name="LogOut" size={18} /></span>
          <div>
            <div className="font-extrabold text-[14px]" style={{ color: 'var(--ink)' }}>Log out</div>
            <div className="text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>Sign out of the Vendor Portal</div>
          </div>
        </div>
        <button onClick={onLogout} className="rounded-xl px-4 py-2.5 font-extrabold text-[13.5px] text-white active:scale-[0.98] transition-transform whitespace-nowrap" style={{ background: '#e11d48' }}>Log out</button>
      </div>

      {showTeam && (
        <TeamPanel accent={accent} team={team} setTeam={setTeam}
                   invites={invites} setInvites={setInvites} onClose={() => setShowTeam(false)} />
      )}
    </div>
  );
}

Object.assign(window, { AccountProfile });
