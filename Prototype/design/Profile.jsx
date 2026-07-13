// Profile.jsx — Spotto consumer ACCOUNT profile (mobile). Full-screen overlay.
// Edits avatar, name, email, travel style, interests, notifications + account rows.
// Props: prefs {style, interests}, onSave(prefs), onClose(), onLogout().

const { useState: useStateProf } = React;

function ProfRow({ icon, title, sub, right, onClick, danger }) {
  return (
    <button onClick={onClick}
      className="w-full flex items-center gap-3 px-4 py-3.5 text-left active:opacity-70 transition-opacity">
      <span className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0"
            style={{ background: danger ? 'rgba(225,29,72,0.1)' : 'var(--accent-tint)', color: danger ? '#e11d48' : 'var(--accent)' }}>
        <Icon name={icon} size={18} />
      </span>
      <div className="flex-1 min-w-0">
        <div className="font-extrabold text-[14.5px]" style={{ color: danger ? '#e11d48' : 'var(--ink)' }}>{title}</div>
        {sub && <div className="text-[12px] font-semibold truncate" style={{ color: 'var(--muted)' }}>{sub}</div>}
      </div>
      {right || <Icon name="ChevronRight" size={18} style={{ color: 'var(--muted)' }} />}
    </button>
  );
}

function ProfToggle({ on, onClick }) {
  return (
    <button onClick={onClick} className="relative rounded-full transition-colors flex-shrink-0"
      style={{ width: 46, height: 27, background: on ? 'var(--accent)' : 'rgba(15,61,58,0.18)' }}>
      <span className="absolute top-1 rounded-full bg-white transition-all"
            style={{ width: 19, height: 19, left: on ? 23 : 4, boxShadow: '0 1px 3px rgba(0,0,0,0.2)' }} />
    </button>
  );
}

function ProfField({ label, value, onChange, icon, type = 'text' }) {
  return (
    <label className="block">
      <span className="block text-[12px] font-extrabold mb-1.5" style={{ color: 'var(--ink-soft)' }}>{label}</span>
      <div className="flex items-center gap-2.5 rounded-2xl px-3.5" style={{ height: 50, background: 'var(--sand)', border: '1.5px solid var(--line, rgba(15,61,58,0.1))' }}>
        {icon && <Icon name={icon} size={17} style={{ color: 'var(--muted)' }} />}
        <input type={type} value={value} onChange={(e) => onChange(e.target.value)}
          className="flex-1 bg-transparent outline-none text-[14.5px] font-bold w-full" style={{ color: 'var(--ink)' }} />
      </div>
    </label>
  );
}

function SectionCard({ title, children, pad = true }) {
  return (
    <div>
      {title && <div className="px-1 mb-2 text-[12px] font-extrabold uppercase tracking-wide" style={{ color: 'var(--muted)' }}>{title}</div>}
      <div className="rounded-3xl bg-white overflow-hidden" style={{ boxShadow: '0 2px 14px -8px rgba(15,61,58,0.2)', padding: pad ? 16 : 0 }}>
        {children}
      </div>
    </div>
  );
}

// ── shared sub-page shell ──────────────────────────────────────
function SubPage({ title, onBack, children }) {
  return (
    <div className="absolute inset-0 z-50 flex flex-col sub-anim" style={{ background: 'var(--sand)' }}>
      <div className="flex items-center gap-3 px-4 flex-shrink-0" style={{ paddingTop: 60, paddingBottom: 10, background: 'var(--sand)' }}>
        <button onClick={onBack} className="w-10 h-10 rounded-full flex items-center justify-center flex-shrink-0" style={{ background: '#fff', color: 'var(--ink)', boxShadow: '0 2px 8px rgba(15,61,58,0.08)' }}>
          <Icon name="ChevronLeft" size={22} />
        </button>
        <span className="font-extrabold text-[17px] truncate" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>{title}</span>
      </div>
      <div className="flex-1 overflow-y-auto px-5 pb-10" style={{ paddingTop: 6 }}>{children}</div>
      <style>{`@keyframes subIn{from{opacity:0;transform:translateX(16px)}to{opacity:1;transform:none}}@media (prefers-reduced-motion: no-preference){.sub-anim{animation:subIn .28s cubic-bezier(.22,1,.36,1) both}}`}</style>
    </div>
  );
}

function PwField({ label, value, onChange, placeholder }) {
  const [show, setShow] = useStateProf(false);
  return (
    <label className="block">
      <span className="block text-[12px] font-extrabold mb-1.5" style={{ color: 'var(--ink-soft)' }}>{label}</span>
      <div className="flex items-center gap-2.5 rounded-2xl px-3.5" style={{ height: 50, background: 'var(--sand)', border: '1.5px solid var(--line, rgba(15,61,58,0.1))' }}>
        <Icon name="Lock" size={17} style={{ color: 'var(--muted)' }} />
        <input type={show ? 'text' : 'password'} value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder}
          className="flex-1 bg-transparent outline-none text-[14.5px] font-bold w-full" style={{ color: 'var(--ink)' }} />
        <button type="button" onClick={() => setShow((s) => !s)} style={{ color: 'var(--muted)' }}><Icon name={show ? 'EyeOff' : 'Eye'} size={18} /></button>
      </div>
    </label>
  );
}

function pwStrength(v) {
  let s = 0;
  if (v.length >= 8) s++;
  if (/[A-Z]/.test(v) && /[a-z]/.test(v)) s++;
  if (/\d/.test(v)) s++;
  if (/[^A-Za-z0-9]/.test(v)) s++;
  const map = [
    { label: 'Too short', color: '#cbd5d3' },
    { label: 'Weak', color: '#e11d48' },
    { label: 'Fair', color: '#eaa200' },
    { label: 'Good', color: '#0b6f86' },
    { label: 'Strong', color: '#16a34a' },
  ];
  return { score: s, ...map[s] };
}

// ── Password & security ────────────────────────────────────────
function SecurityPage({ onBack }) {
  const [cur, setCur] = useStateProf('');
  const [pw, setPw] = useStateProf('');
  const [pw2, setPw2] = useStateProf('');
  const [twofa, setTwofa] = useStateProf(true);
  const [method, setMethod] = useStateProf('app');
  const [saved, setSaved] = useStateProf(false);
  const st = pwStrength(pw);
  const match = pw && pw === pw2;
  const devices = [
    { name: 'iPhone 15', meta: 'Weligama · Active now', icon: 'Smartphone', current: true },
    { name: 'MacBook Air · Safari', meta: 'Colombo · Yesterday', icon: 'Laptop', current: false },
  ];
  function update() { setSaved(true); setTimeout(() => setSaved(false), 1800); setCur(''); setPw(''); setPw2(''); }

  return (
    <SubPage title="Password & security" onBack={onBack}>
      <div className="space-y-6">
        <SectionCard title="Change password">
          <div className="space-y-3.5">
            <PwField label="Current password" value={cur} onChange={setCur} placeholder="••••••••" />
            <PwField label="New password" value={pw} onChange={setPw} placeholder="At least 8 characters" />
            {pw && (
              <div>
                <div className="flex gap-1.5">
                  {[1, 2, 3, 4].map((i) => (
                    <div key={i} className="h-1.5 flex-1 rounded-full transition-all" style={{ background: i <= st.score ? st.color : 'rgba(15,61,58,0.12)' }} />
                  ))}
                </div>
                <div className="text-[12px] font-extrabold mt-1.5" style={{ color: st.color }}>{st.label}</div>
              </div>
            )}
            <PwField label="Confirm new password" value={pw2} onChange={setPw2} placeholder="Re-enter password" />
            {pw2 && !match && <div className="text-[12px] font-bold" style={{ color: '#e11d48' }}>Passwords don't match</div>}
            <button onClick={update} disabled={!match || st.score < 2}
              className="w-full rounded-2xl py-3.5 font-extrabold text-[15px] flex items-center justify-center gap-2 whitespace-nowrap transition-all active:scale-[0.98]"
              style={{ background: saved ? '#16a34a' : (match && st.score >= 2 ? 'var(--accent)' : 'rgba(15,61,58,0.1)'), color: (saved || (match && st.score >= 2)) ? '#fff' : 'rgba(15,61,58,0.35)', boxShadow: (match && st.score >= 2) ? '0 12px 24px -12px var(--accent-shadow)' : 'none' }}>
              {saved ? <><Icon name="Check" size={17} strokeWidth={3} /> Password updated</> : 'Update password'}
            </button>
          </div>
        </SectionCard>

        <SectionCard title="Two-factor authentication" pad={false}>
          <div className="flex items-center gap-3 px-4 py-3.5">
            <span className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0" style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}><Icon name="ShieldCheck" size={18} /></span>
            <div className="flex-1 min-w-0">
              <div className="font-extrabold text-[14px]" style={{ color: 'var(--ink)' }}>Require a code at login</div>
              <div className="text-[12px] font-semibold" style={{ color: 'var(--muted)' }}>Extra protection for your account</div>
            </div>
            <ProfToggle on={twofa} onClick={() => setTwofa(!twofa)} />
          </div>
          {twofa && (
            <div className="px-4 pb-4 pt-1" style={{ borderTop: '1px solid rgba(15,61,58,0.07)' }}>
              <div className="text-[11.5px] font-extrabold uppercase tracking-wide mb-2 mt-2.5" style={{ color: 'var(--muted)' }}>Method</div>
              <div className="grid grid-cols-2 gap-2.5">
                {[{ id: 'app', icon: 'Smartphone', label: 'Authenticator' }, { id: 'sms', icon: 'MessageSquare', label: 'SMS code' }].map((m) => {
                  const on = method === m.id;
                  return (
                    <button key={m.id} onClick={() => setMethod(m.id)} className="flex items-center gap-2 rounded-2xl px-3 py-3 transition-all active:scale-[0.98]"
                      style={{ background: on ? 'var(--accent)' : 'var(--sand)', color: on ? '#fff' : 'var(--ink)', border: `1.5px solid ${on ? 'var(--accent)' : 'transparent'}` }}>
                      <Icon name={m.icon} size={17} /><span className="font-extrabold text-[13px]">{m.label}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          )}
        </SectionCard>

        <SectionCard title="Where you're logged in" pad={false}>
          {devices.map((d, i) => (
            <div key={d.name} className="flex items-center gap-3 px-4 py-3.5" style={{ borderTop: i ? '1px solid rgba(15,61,58,0.07)' : 'none' }}>
              <span className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0" style={{ background: 'var(--sand)', color: 'var(--ink-soft)' }}><Icon name={d.icon} size={17} /></span>
              <div className="flex-1 min-w-0">
                <div className="font-extrabold text-[14px] truncate" style={{ color: 'var(--ink)' }}>{d.name}</div>
                <div className="text-[12px] font-semibold" style={{ color: 'var(--muted)' }}>{d.meta}</div>
              </div>
              {d.current ? <span className="text-[10.5px] font-extrabold px-2 py-1 rounded-full whitespace-nowrap" style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}>This device</span>
                : <button className="text-[12px] font-extrabold whitespace-nowrap" style={{ color: '#e11d48' }}>Log out</button>}
            </div>
          ))}
        </SectionCard>

        <SectionCard pad={false}>
          <ProfRow icon="Trash2" title="Delete account" sub="Permanently remove your data" danger right={<Icon name="ChevronRight" size={18} style={{ color: '#e11d48' }} />} />
        </SectionCard>
      </div>
    </SubPage>
  );
}

// ── Saved payment ──────────────────────────────────────────────
function PaymentPage({ onBack }) {
  const [cards, setCards] = useStateProf([
    { id: 'c1', brand: 'Visa', last4: '4291', exp: '08/27', grad: 'linear-gradient(135deg,#1a4ba3,#3f86f5)' },
    { id: 'c2', brand: 'Mastercard', last4: '8830', exp: '02/26', grad: 'linear-gradient(135deg,#3a3a44,#16161c)' },
  ]);
  const [def, setDef] = useStateProf('c1');
  const activity = [
    { id: 'a1', title: 'Surf lesson · Bay Surf School', date: 'Jun 6', amt: 'Rs 4,500', icon: 'Waves' },
    { id: 'a2', title: 'Wheel-throwing taster · Clay & Co', date: 'Jun 3', amt: 'Rs 1,500', icon: 'Palette' },
    { id: 'a3', title: 'Day pass · Driftwood Co-Work', date: 'May 30', amt: 'Rs 2,000', icon: 'Coffee' },
  ];
  const defaultCard = cards.find((c) => c.id === def) || cards[0];

  return (
    <SubPage title="Saved payment" onBack={onBack}>
      <div className="space-y-6">
        {/* featured default card */}
        <div className="rounded-[22px] p-5 relative overflow-hidden" style={{ background: defaultCard.grad, minHeight: 168, boxShadow: '0 18px 36px -18px rgba(16,61,58,0.6)' }}>
          <div style={{ position: 'absolute', right: -30, top: -30, width: 150, height: 150, borderRadius: '50%', background: 'rgba(255,255,255,0.1)' }} />
          <div className="relative flex items-start justify-between">
            <span className="text-[11px] font-extrabold uppercase tracking-wider px-2 py-0.5 rounded-full" style={{ background: 'rgba(255,255,255,0.22)', color: '#fff' }}>Default</span>
            <Icon name="Wifi" size={20} style={{ color: 'rgba(255,255,255,0.7)', transform: 'rotate(90deg)' }} />
          </div>
          <div className="relative mt-7 font-bold text-white tracking-[0.18em]" style={{ fontSize: 19 }}>•••• •••• •••• {defaultCard.last4}</div>
          <div className="relative mt-4 flex items-end justify-between">
            <div>
              <div className="text-[10px] font-bold uppercase tracking-wide" style={{ color: 'rgba(255,255,255,0.6)' }}>Expires</div>
              <div className="text-[14px] font-extrabold text-white">{defaultCard.exp}</div>
            </div>
            <div className="font-extrabold text-white italic text-[17px]">{defaultCard.brand}</div>
          </div>
        </div>

        <SectionCard title="Your cards" pad={false}>
          {cards.map((c, i) => {
            const isDef = c.id === def;
            return (
              <div key={c.id} className="flex items-center gap-3 px-4 py-3.5" style={{ borderTop: i ? '1px solid rgba(15,61,58,0.07)' : 'none' }}>
                <span className="w-10 h-7 rounded-md flex items-center justify-center flex-shrink-0" style={{ background: c.grad }}><Icon name="CreditCard" size={15} style={{ color: '#fff' }} /></span>
                <div className="flex-1 min-w-0">
                  <div className="font-extrabold text-[14px]" style={{ color: 'var(--ink)' }}>{c.brand} ·· {c.last4}</div>
                  <div className="text-[12px] font-semibold" style={{ color: 'var(--muted)' }}>Expires {c.exp}</div>
                </div>
                {isDef ? <span className="text-[10.5px] font-extrabold px-2 py-1 rounded-full whitespace-nowrap" style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}>Default</span>
                  : <button onClick={() => setDef(c.id)} className="text-[12px] font-extrabold whitespace-nowrap" style={{ color: 'var(--accent)' }}>Set default</button>}
              </div>
            );
          })}
          <button className="w-full flex items-center gap-3 px-4 py-3.5" style={{ borderTop: '1px solid rgba(15,61,58,0.07)' }}>
            <span className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0" style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}><Icon name="Plus" size={18} /></span>
            <span className="font-extrabold text-[14px]" style={{ color: 'var(--accent)' }}>Add payment method</span>
          </button>
        </SectionCard>

        <SectionCard title="Recent activity" pad={false}>
          {activity.map((a, i) => (
            <div key={a.id} className="flex items-center gap-3 px-4 py-3.5" style={{ borderTop: i ? '1px solid rgba(15,61,58,0.07)' : 'none' }}>
              <span className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0" style={{ background: 'var(--sand)', color: 'var(--ink-soft)' }}><Icon name={a.icon} size={17} /></span>
              <div className="flex-1 min-w-0">
                <div className="font-extrabold text-[13.5px] truncate" style={{ color: 'var(--ink)' }}>{a.title}</div>
                <div className="text-[12px] font-semibold" style={{ color: 'var(--muted)' }}>{a.date}</div>
              </div>
              <div className="font-extrabold text-[13.5px] whitespace-nowrap" style={{ color: 'var(--ink)' }}>{a.amt}</div>
            </div>
          ))}
        </SectionCard>
      </div>
    </SubPage>
  );
}

// ── Privacy ────────────────────────────────────────────────────
function PrivacyPage({ onBack }) {
  const [p, setP] = useStateProf({
    locShare: true, locBg: false, groupMap: true,
    recs: true, analytics: true, marketing: false,
    checkins: true, publicProfile: false,
  });
  const set = (k) => setP((s) => ({ ...s, [k]: !s[k] }));
  const group = (title, sub, rows) => (
    <SectionCard title={title} pad={false}>
      {rows.map((r, i) => (
        <div key={r.k} className="flex items-center gap-3 px-4 py-3.5" style={{ borderTop: i ? '1px solid rgba(15,61,58,0.07)' : 'none' }}>
          <span className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0" style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}><Icon name={r.icon} size={17} /></span>
          <div className="flex-1 min-w-0">
            <div className="font-extrabold text-[14px]" style={{ color: 'var(--ink)' }}>{r.title}</div>
            <div className="text-[12px] font-semibold" style={{ color: 'var(--muted)' }}>{r.sub}</div>
          </div>
          <ProfToggle on={p[r.k]} onClick={() => set(r.k)} />
        </div>
      ))}
    </SectionCard>
  );
  return (
    <SubPage title="Privacy" onBack={onBack}>
      <div className="space-y-6">
        {group('Location', null, [
          { k: 'locShare', icon: 'MapPin', title: 'Share while using Spotto', sub: 'Power live distances & nearby perks' },
          { k: 'locBg', icon: 'Navigation', title: 'Background location', sub: 'Get deal alerts when the app is closed' },
          { k: 'groupMap', icon: 'Users', title: 'Show me on group map', sub: 'Let friends in your group see your pin' },
        ])}
        {group('Data & personalization', null, [
          { k: 'recs', icon: 'Sparkles', title: 'Personalized picks', sub: 'Tune the map to your style & interests' },
          { k: 'analytics', icon: 'BarChart3', title: 'Usage analytics', sub: 'Help improve Spotto with anonymous data' },
          { k: 'marketing', icon: 'Mail', title: 'Marketing emails', sub: 'Travel tips & partner offers' },
        ])}
        {group('Visibility', null, [
          { k: 'checkins', icon: 'CheckCircle', title: 'Show my check-ins', sub: 'Visible to friends you travel with' },
          { k: 'publicProfile', icon: 'Globe', title: 'Public profile', sub: 'Anyone with your link can view it' },
        ])}
        <SectionCard pad={false}>
          <ProfRow icon="Download" title="Download my data" sub="Get a copy of your Spotto data" />
          <div style={{ borderTop: '1px solid rgba(15,61,58,0.07)' }} />
          <ProfRow icon="Trash2" title="Delete my data" sub="Erase activity & history" danger right={<Icon name="ChevronRight" size={18} style={{ color: '#e11d48' }} />} />
        </SectionCard>
      </div>
    </SubPage>
  );
}

// ── Help & support ─────────────────────────────────────────────
function FaqItem({ q, a, isLast }) {
  const [open, setOpen] = useStateProf(false);
  return (
    <div style={{ borderBottom: isLast ? 'none' : '1px solid rgba(15,61,58,0.07)' }}>
      <button onClick={() => setOpen((o) => !o)} className="w-full flex items-center gap-3 px-4 py-3.5 text-left">
        <span className="flex-1 font-extrabold text-[14px]" style={{ color: 'var(--ink)' }}>{q}</span>
        <Icon name="ChevronDown" size={18} style={{ color: 'var(--muted)', transform: open ? 'rotate(180deg)' : 'none', transition: 'transform 0.2s' }} />
      </button>
      {open && <p className="px-4 pb-4 -mt-1 text-[13px] font-semibold leading-relaxed" style={{ color: 'var(--ink-soft)' }}>{a}</p>}
    </div>
  );
}

function HelpPage({ onBack }) {
  const topics = ['Redeeming a perk', 'Account & login', 'Payments', 'Group mode', 'Daily quests'];
  const faqs = [
    { q: 'How do I redeem a deal?', a: 'Open a spot from the map, tap its perk, and show the QR code at the counter — the vendor scans it to confirm.' },
    { q: 'Why are some pins faded?', a: 'Spotto highlights places that fit the current time of day, weather and your travel style. Faded pins are still tappable.' },
    { q: 'How do daily quests work?', a: 'Each day brings a small challenge — like visiting two local artisans — that unlocks a reward when you complete it.' },
    { q: 'Is my location always tracked?', a: "Only while you're using the app, unless you enable background location in Privacy settings." },
  ];
  const contacts = [
    { icon: 'MessageCircle', title: 'Chat with us', sub: 'Usually replies in a few minutes', tag: 'Online', tagColor: '#16a34a' },
    { icon: 'Mail', title: 'Email support', sub: 'help@spotto.app' },
    { icon: 'Phone', title: 'Call us', sub: '+94 11 234 5678 · 9am–6pm' },
  ];
  return (
    <SubPage title="Help & support" onBack={onBack}>
      <div className="space-y-6">
        <div className="flex items-center gap-2.5 rounded-2xl px-4" style={{ height: 50, background: '#fff', border: '1.5px solid var(--line, rgba(15,61,58,0.1))', boxShadow: '0 2px 12px -8px rgba(15,61,58,0.2)' }}>
          <Icon name="Search" size={18} style={{ color: 'var(--muted)' }} />
          <input placeholder="Search help articles…" className="flex-1 bg-transparent outline-none text-[14.5px] font-bold" style={{ color: 'var(--ink)' }} />
        </div>

        <div>
          <div className="px-1 mb-2 text-[12px] font-extrabold uppercase tracking-wide" style={{ color: 'var(--muted)' }}>Popular topics</div>
          <div className="flex flex-wrap gap-2">
            {topics.map((t) => (
              <button key={t} className="rounded-full px-3.5 py-2 text-[13px] font-extrabold" style={{ background: '#fff', color: 'var(--ink)', border: '1.5px solid var(--line, rgba(15,61,58,0.1))' }}>{t}</button>
            ))}
          </div>
        </div>

        <SectionCard title="Frequently asked" pad={false}>
          {faqs.map((f, i) => <FaqItem key={f.q} q={f.q} a={f.a} isLast={i === faqs.length - 1} />)}
        </SectionCard>

        <SectionCard title="Get in touch" pad={false}>
          {contacts.map((c, i) => (
            <button key={c.title} className="w-full flex items-center gap-3 px-4 py-3.5 text-left" style={{ borderTop: i ? '1px solid rgba(15,61,58,0.07)' : 'none' }}>
              <span className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0" style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}><Icon name={c.icon} size={18} /></span>
              <div className="flex-1 min-w-0">
                <div className="font-extrabold text-[14px] flex items-center gap-2" style={{ color: 'var(--ink)' }}>
                  {c.title}
                  {c.tag && <span className="text-[10px] font-extrabold px-1.5 py-0.5 rounded-full flex items-center gap-1" style={{ background: 'rgba(22,163,74,0.12)', color: c.tagColor }}><span className="w-1.5 h-1.5 rounded-full" style={{ background: c.tagColor }} />{c.tag}</span>}
                </div>
                <div className="text-[12px] font-semibold truncate" style={{ color: 'var(--muted)' }}>{c.sub}</div>
              </div>
              <Icon name="ChevronRight" size={18} style={{ color: 'var(--muted)' }} />
            </button>
          ))}
        </SectionCard>

        <SectionCard pad={false}>
          <ProfRow icon="Users" title="Community forum" />
          <div style={{ borderTop: '1px solid rgba(15,61,58,0.07)' }} />
          <ProfRow icon="FileText" title="Terms of service" />
          <div style={{ borderTop: '1px solid rgba(15,61,58,0.07)' }} />
          <ProfRow icon="ShieldCheck" title="Privacy policy" />
        </SectionCard>

        <p className="text-center text-[12px] font-bold" style={{ color: 'var(--muted)' }}>Spotto · v2.4 · Weligama</p>
      </div>
    </SubPage>
  );
}

function Profile({ prefs, onSave, onClose, onLogout }) {
  const [sub, setSub] = useStateProf(null);
  const [name, setName] = useStateProf('Lena Müller');
  const [email, setEmail] = useStateProf('lena@email.com');
  const [style, setStyle] = useStateProf(prefs?.style || 'nomad');
  const [interests, setInterests] = useStateProf(prefs?.interests || []);
  const [notif, setNotif] = useStateProf({ deals: true, quest: true, group: true, email: false });
  const [saved, setSaved] = useStateProf(false);

  const toggleInterest = (id) => setInterests((p) => p.includes(id) ? p.filter((x) => x !== id) : [...p, id]);
  const initials = name.split(' ').map((w) => w[0]).join('').slice(0, 2).toUpperCase();
  const styleObj = (window.TRAVEL_STYLES || []).find((s) => s.id === style) || {};

  function save() {
    onSave?.({ style, interests });
    setSaved(true);
    setTimeout(() => setSaved(false), 1800);
  }

  if (sub === 'security') return <SecurityPage onBack={() => setSub(null)} />;
  if (sub === 'payment') return <PaymentPage onBack={() => setSub(null)} />;
  if (sub === 'privacy') return <PrivacyPage onBack={() => setSub(null)} />;
  if (sub === 'help') return <HelpPage onBack={() => setSub(null)} />;

  return (
    <div className="absolute inset-0 z-40 flex flex-col" style={{ background: 'var(--sand)' }}>
      {/* app bar */}
      <div className="flex items-center justify-between px-4 flex-shrink-0" style={{ paddingTop: 60, paddingBottom: 10, background: 'var(--sand)' }}>
        <button onClick={onClose} className="w-10 h-10 rounded-full flex items-center justify-center" style={{ background: '#fff', color: 'var(--ink)', boxShadow: '0 2px 8px rgba(15,61,58,0.08)' }}>
          <Icon name="ChevronLeft" size={22} />
        </button>
        <span className="font-extrabold text-[17px]" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Profile</span>
        <button onClick={save}
          className="rounded-full px-4 h-10 flex items-center gap-1.5 font-extrabold text-[13.5px] text-white whitespace-nowrap transition-all active:scale-95"
          style={{ background: saved ? '#16a34a' : 'var(--accent)', boxShadow: '0 8px 18px -10px var(--accent-shadow)' }}>
          {saved ? <><Icon name="Check" size={15} strokeWidth={3} /> Saved</> : 'Save'}
        </button>
      </div>

      <div className="flex-1 overflow-y-auto px-5 pb-10" style={{ paddingTop: 6 }}>
        {/* hero */}
        <div className="flex flex-col items-center text-center mb-7 mt-2">
          <div className="relative">
            <div className="rounded-full flex items-center justify-center font-extrabold text-white"
                 style={{ width: 92, height: 92, fontSize: 32, background: 'linear-gradient(140deg, var(--accent), var(--accent-2))', boxShadow: '0 14px 30px -12px var(--accent-shadow)' }}>
              {initials}
            </div>
            <button className="absolute bottom-0 right-0 w-8 h-8 rounded-full flex items-center justify-center" style={{ background: '#fff', color: 'var(--accent)', boxShadow: '0 3px 10px rgba(15,61,58,0.2)', border: '1.5px solid var(--accent-tint)' }}>
              <Icon name="Camera" size={15} />
            </button>
          </div>
          <h2 className="font-extrabold mt-3.5" style={{ fontSize: 21, color: 'var(--ink)' }}>{name}</h2>
          <div className="flex items-center gap-1.5 mt-1 rounded-full px-3 py-1" style={{ background: 'var(--accent-tint)' }}>
            {styleObj.icon && <Icon name={styleObj.icon} size={13} style={{ color: 'var(--accent)' }} />}
            <span className="text-[12.5px] font-extrabold" style={{ color: 'var(--accent)' }}>{styleObj.label || 'Traveler'}</span>
            <span className="text-[11px] font-bold" style={{ color: 'var(--muted)' }}>· Weligama</span>
          </div>
        </div>

        <div className="space-y-6">
          {/* details */}
          <SectionCard title="Your details">
            <div className="space-y-3.5">
              <ProfField label="Name" value={name} onChange={setName} icon="User" />
              <ProfField label="Email" value={email} onChange={setEmail} icon="Mail" type="email" />
            </div>
          </SectionCard>

          {/* travel style */}
          <SectionCard title="Travel style">
            <div className="grid grid-cols-2 gap-2.5">
              {(window.TRAVEL_STYLES || []).map((s) => {
                const on = style === s.id;
                return (
                  <button key={s.id} onClick={() => setStyle(s.id)}
                    className="flex items-center gap-2.5 rounded-2xl px-3 py-3 text-left transition-all active:scale-[0.98]"
                    style={{ background: on ? 'var(--accent)' : 'var(--sand)', border: `1.5px solid ${on ? 'var(--accent)' : 'transparent'}`, color: on ? '#fff' : 'var(--ink)' }}>
                    <Icon name={s.icon} size={18} />
                    <span className="font-extrabold text-[13px] leading-tight">{s.label}</span>
                  </button>
                );
              })}
            </div>
          </SectionCard>

          {/* interests */}
          <SectionCard title="Interests">
            <div className="flex flex-wrap gap-2">
              {(window.INTERESTS || []).map((it) => {
                const on = interests.includes(it.id);
                return (
                  <button key={it.id} onClick={() => toggleInterest(it.id)}
                    className="flex items-center gap-1.5 rounded-full pl-2.5 pr-3.5 py-2 transition-all active:scale-95"
                    style={{ background: on ? 'var(--accent)' : 'var(--sand)', color: on ? '#fff' : 'var(--ink)', border: `1.5px solid ${on ? 'var(--accent)' : 'transparent'}` }}>
                    <Icon name={it.icon} size={15} />
                    <span className="font-bold text-[13px]">{it.label}</span>
                  </button>
                );
              })}
            </div>
          </SectionCard>

          {/* notifications */}
          <SectionCard title="Notifications" pad={false}>
            {[
              { k: 'deals', icon: 'Tag', title: 'Deal alerts near me', sub: 'When a perk is within walking distance' },
              { k: 'quest', icon: 'Compass', title: 'Daily quest reminders', sub: 'Nudge me to finish the day\u2019s quest' },
              { k: 'group', icon: 'Users', title: 'Group activity', sub: 'When friends check in nearby' },
              { k: 'email', icon: 'Mail', title: 'Email updates', sub: 'Occasional travel tips & news' },
            ].map((n, i, arr) => (
              <div key={n.k} className="flex items-center gap-3 px-4 py-3" style={{ borderTop: i ? '1px solid rgba(15,61,58,0.07)' : 'none' }}>
                <span className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0" style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}><Icon name={n.icon} size={17} /></span>
                <div className="flex-1 min-w-0">
                  <div className="font-extrabold text-[14px]" style={{ color: 'var(--ink)' }}>{n.title}</div>
                  <div className="text-[12px] font-semibold" style={{ color: 'var(--muted)' }}>{n.sub}</div>
                </div>
                <ProfToggle on={notif[n.k]} onClick={() => setNotif((p) => ({ ...p, [n.k]: !p[n.k] }))} />
              </div>
            ))}
          </SectionCard>

          {/* account */}
          <SectionCard title="Account" pad={false}>
            <ProfRow icon="Lock" title="Password & security" sub="Change password, 2FA" onClick={() => setSub('security')} />
            <div style={{ borderTop: '1px solid rgba(15,61,58,0.07)' }} />
            <ProfRow icon="CreditCard" title="Saved payment" sub="Visa ·· 4291" onClick={() => setSub('payment')} />
            <div style={{ borderTop: '1px solid rgba(15,61,58,0.07)' }} />
            <ProfRow icon="ShieldCheck" title="Privacy" sub="Location & data sharing" onClick={() => setSub('privacy')} />
            <div style={{ borderTop: '1px solid rgba(15,61,58,0.07)' }} />
            <ProfRow icon="HelpCircle" title="Help & support" onClick={() => setSub('help')} />
          </SectionCard>

          <SectionCard pad={false}>
            <ProfRow icon="LogOut" title="Log out" danger onClick={onLogout} right={<span />} />
          </SectionCard>

          <p className="text-center text-[12px] font-bold pt-1" style={{ color: 'var(--muted)' }}>Spotto · v2.4 · Weligama</p>
        </div>
      </div>
    </div>
  );
}

Object.assign(window, { Profile });
