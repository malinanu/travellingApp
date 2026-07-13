// VendorAuth.jsx — Spotto Vendor Portal sign-in + multi-step "list your business".
// Modes: login | register (3-step application) | forgot. Layouts: 'split' | 'centered'.

const { useState: useStateVAuth, useRef: useRefVAuth } = React;

function VGoogleG({ size = 18 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 48 48" aria-hidden="true">
      <path fill="#4285F4" d="M45.1 24.5c0-1.6-.1-3.1-.4-4.5H24v8.5h11.8c-.5 2.7-2 5-4.4 6.6v5.5h7.1c4.1-3.8 6.6-9.4 6.6-16.1z"/>
      <path fill="#34A853" d="M24 46c5.9 0 10.9-2 14.5-5.3l-7.1-5.5c-2 1.3-4.5 2.1-7.4 2.1-5.7 0-10.5-3.8-12.2-9h-7.3v5.7C8.1 41.1 15.4 46 24 46z"/>
      <path fill="#FBBC05" d="M11.8 28.3c-.4-1.3-.7-2.7-.7-4.3s.3-3 .7-4.3v-5.7H4.5C3 17.1 2.1 20.4 2.1 24s.9 6.9 2.4 9.7l7.3-5.4z"/>
      <path fill="#EA4335" d="M24 10.7c3.2 0 6.1 1.1 8.4 3.3l6.3-6.3C34.9 4.1 29.9 2 24 2 15.4 2 8.1 6.9 4.5 14.3l7.3 5.7c1.7-5.2 6.5-9.3 12.2-9.3z"/>
    </svg>
  );
}
function VAppleLogo({ size = 18 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M16.4 12.8c0-2.3 1.9-3.4 2-3.5-1.1-1.6-2.8-1.8-3.4-1.8-1.4-.1-2.8.9-3.5.9-.7 0-1.9-.8-3.1-.8-1.6 0-3.1.9-3.9 2.4-1.7 2.9-.4 7.2 1.2 9.5.8 1.1 1.7 2.4 2.9 2.4 1.2-.1 1.6-.8 3-.8s1.8.8 3.1.7c1.3 0 2.1-1.1 2.9-2.3.9-1.3 1.3-2.6 1.3-2.7-.1 0-2.5-1-2.5-3.8zM14.1 5.9c.6-.8 1.1-1.9 1-3-.9.1-2.1.6-2.7 1.4-.6.7-1.2 1.8-1 2.9 1 .1 2-.5 2.7-1.3z"/>
    </svg>
  );
}

const VBIZ_CATS = [
  { id: 'cafe',  label: 'Café / Restaurant', icon: 'Coffee' },
  { id: 'surf',  label: 'Surf School',       icon: 'Waves' },
  { id: 'stay',  label: 'Stay / Hotel',      icon: 'BedDouble' },
  { id: 'well',  label: 'Wellness / Yoga',   icon: 'Flower2' },
  { id: 'tour',  label: 'Tours / Activity',  icon: 'Compass' },
  { id: 'shop',  label: 'Shop / Artisan',    icon: 'Palette' },
];

function VField({ label, type = 'text', placeholder, icon, reveal, onReveal, autoFocus, textarea }) {
  return (
    <label className="block">
      <span className="block text-[12.5px] font-extrabold mb-1.5" style={{ color: 'var(--ink-soft)' }}>{label}</span>
      {textarea ? (
        <textarea rows={3} placeholder={placeholder} autoFocus={autoFocus}
          className="w-full rounded-2xl px-3.5 py-3 bg-transparent outline-none text-[15px] font-bold resize-none"
          style={{ background: '#fff', border: '1.5px solid var(--line)', color: 'var(--ink)' }} />
      ) : (
        <div className="flex items-center gap-2.5 rounded-2xl px-3.5" style={{ height: 50, background: '#fff', border: '1.5px solid var(--line)' }}>
          {icon && <Icon name={icon} size={18} style={{ color: 'var(--muted)' }} />}
          <input type={type} placeholder={placeholder} autoFocus={autoFocus}
            className="flex-1 bg-transparent outline-none text-[15px] font-bold w-full" style={{ color: 'var(--ink)' }} />
          {onReveal && (
            <button type="button" onClick={onReveal} style={{ color: 'var(--muted)' }}>
              <Icon name={reveal ? 'EyeOff' : 'Eye'} size={18} />
            </button>
          )}
        </div>
      )}
    </label>
  );
}

function VPrimary({ children, onClick, icon = 'ArrowRight' }) {
  return (
    <button onClick={onClick}
      className="w-full rounded-2xl py-3.5 font-extrabold text-[15.5px] flex items-center justify-center gap-2 whitespace-nowrap transition-all active:scale-[0.98]"
      style={{ background: 'var(--accent)', color: '#fff', boxShadow: '0 14px 28px -12px var(--accent-shadow)' }}>
      {children}
      {icon && <Icon name={icon} size={18} strokeWidth={2.5} />}
    </button>
  );
}

function VSocial({ onSocial }) {
  const btn = (label, mark, onClick) => (
    <button onClick={onClick}
      className="flex-1 flex items-center justify-center gap-2 rounded-2xl py-3 font-extrabold text-[14px] transition-all active:scale-[0.97]"
      style={{ background: '#fff', border: '1.5px solid var(--line)', color: 'var(--ink)' }}>
      {mark}<span>{label}</span>
    </button>
  );
  return (
    <div className="flex gap-2.5">
      {btn('Google', <VGoogleG />, () => onSocial('google'))}
      {btn('Apple', <span style={{ color: 'var(--ink)' }}><VAppleLogo /></span>, () => onSocial('apple'))}
    </div>
  );
}

function VMark({ size = 30, light = false }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" fill="none">
      <path d="M16 3C10.5 3 6 7.3 6 12.7 6 19.5 16 29 16 29s10-9.5 10-16.3C26 7.3 21.5 3 16 3z" fill={light ? '#fff' : 'var(--accent)'} />
      <circle cx="16" cy="12.5" r="4.4" fill={light ? 'var(--sun)' : '#f6b81e'} />
    </svg>
  );
}

// ── login / forgot / multi-step register ───────────────────────
function VendorAuthForm({ onAuthed }) {
  const [mode, setMode] = useStateVAuth('login');   // login | register | forgot
  const [step, setStep] = useStateVAuth(0);          // register: 0,1,2 → 3 = submitted
  const [show, setShow] = useStateVAuth(false);
  const [remember, setRemember] = useStateVAuth(true);
  const [cat, setCat] = useStateVAuth(null);
  const [sent, setSent] = useStateVAuth(false);

  const TextLink = ({ children, onClick }) => (
    <button onClick={onClick} className="font-extrabold whitespace-nowrap" style={{ color: 'var(--accent)' }}>{children}</button>
  );

  // ── LOGIN ──
  if (mode === 'login') {
    return (
      <div className="vd-auth-anim w-full" style={{ maxWidth: 380 }}>
        <h1 className="font-extrabold leading-tight" style={{ fontSize: 28, color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Welcome back</h1>
        <p className="text-[14px] font-semibold mt-1.5 mb-6" style={{ color: 'var(--muted)' }}>Sign in to manage your offers and redemptions.</p>
        <div className="space-y-3.5">
          <VField label="Email" type="email" placeholder="you@yourbusiness.lk" icon="Mail" autoFocus />
          <VField label="Password" type={show ? 'text' : 'password'} placeholder="••••••••" icon="Lock" reveal={show} onReveal={() => setShow((s) => !s)} />
          <div className="flex items-center justify-between pt-0.5">
            <button onClick={() => setRemember((r) => !r)} className="flex items-center gap-2">
              <span className="rounded-md flex items-center justify-center" style={{ width: 20, height: 20, background: remember ? 'var(--accent)' : '#fff', border: `1.5px solid ${remember ? 'var(--accent)' : 'rgba(15,61,58,0.2)'}` }}>
                {remember && <Icon name="Check" size={13} strokeWidth={3.5} style={{ color: '#fff' }} />}
              </span>
              <span className="text-[13px] font-bold whitespace-nowrap" style={{ color: 'var(--ink-soft)' }}>Remember me</span>
            </button>
            <button onClick={() => { setSent(false); setMode('forgot'); }} className="text-[13px] font-extrabold" style={{ color: 'var(--accent)' }}>Forgot?</button>
          </div>
          <div className="pt-1"><VPrimary onClick={onAuthed}>Sign in</VPrimary></div>
          <div className="flex items-center gap-3 my-1">
            <div className="flex-1 h-px" style={{ background: 'var(--line)' }} />
            <span className="text-[12px] font-bold" style={{ color: 'var(--muted)' }}>or</span>
            <div className="flex-1 h-px" style={{ background: 'var(--line)' }} />
          </div>
          <VSocial onSocial={onAuthed} />
        </div>
        <div className="mt-6 rounded-2xl p-4 flex items-center gap-3" style={{ background: 'var(--accent-tint)' }}>
          <Icon name="Store" size={20} style={{ color: 'var(--accent)' }} />
          <div className="flex-1">
            <div className="text-[13.5px] font-extrabold" style={{ color: 'var(--ink)' }}>New to Spotto?</div>
            <button onClick={() => { setStep(0); setMode('register'); }} className="text-[13px] font-extrabold whitespace-nowrap" style={{ color: 'var(--accent)' }}>List your business →</button>
          </div>
        </div>
      </div>
    );
  }

  // ── FORGOT ──
  if (mode === 'forgot') {
    return (
      <div className="vd-auth-anim w-full" style={{ maxWidth: 380 }}>
        <h1 className="font-extrabold leading-tight" style={{ fontSize: 28, color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Reset password</h1>
        <p className="text-[14px] font-semibold mt-1.5 mb-6" style={{ color: 'var(--muted)' }}>We'll email you a secure reset link.</p>
        {sent ? (
          <div className="rounded-2xl p-4 flex items-start gap-3 mb-5" style={{ background: 'var(--accent-tint)' }}>
            <Icon name="MailCheck" size={20} style={{ color: 'var(--accent)' }} />
            <p className="text-[13.5px] font-bold leading-snug" style={{ color: 'var(--ink)' }}>Check your inbox — the reset link is on its way.</p>
          </div>
        ) : (
          <div className="mb-5"><VField label="Email" type="email" placeholder="you@yourbusiness.lk" icon="Mail" autoFocus /></div>
        )}
        {!sent && <VPrimary onClick={() => setSent(true)} icon="Send">Send reset link</VPrimary>}
        <button onClick={() => { setSent(false); setMode('login'); }} className="w-full text-center text-[13.5px] font-extrabold mt-4" style={{ color: 'var(--muted)' }}>← Back to sign in</button>
      </div>
    );
  }

  // ── REGISTER (multi-step) ──
  const STEPS = ['Your account', 'Your business', 'A few details'];
  const submitted = step >= 3;

  return (
    <div className="vd-auth-anim w-full" style={{ maxWidth: 400 }}>
      {!submitted ? (
        <>
          {/* progress */}
          <div className="flex items-center gap-2 mb-1">
            <span className="text-[11.5px] font-extrabold uppercase tracking-wider" style={{ color: 'var(--accent)' }}>Step {step + 1} of 3</span>
          </div>
          <div className="flex gap-1.5 mb-5">
            {[0, 1, 2].map((i) => (
              <div key={i} className="h-1.5 rounded-full transition-all duration-300"
                style={{ flex: 1, background: i <= step ? 'var(--accent)' : 'rgba(15,61,58,0.12)' }} />
            ))}
          </div>
          <h1 className="font-extrabold leading-tight mb-5" style={{ fontSize: 25, color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>{STEPS[step]}</h1>

          {step === 0 && (
            <div className="space-y-3.5">
              <VField label="Owner / contact name" placeholder="e.g. Tharindu Perera" icon="User" autoFocus />
              <VField label="Email" type="email" placeholder="you@yourbusiness.lk" icon="Mail" />
              <VField label="Password" type={show ? 'text' : 'password'} placeholder="Create a password" icon="Lock" reveal={show} onReveal={() => setShow((s) => !s)} />
            </div>
          )}

          {step === 1 && (
            <div className="space-y-4">
              <VField label="Business name" placeholder="e.g. Driftwood Coffee & Co-Work" icon="Store" autoFocus />
              <div>
                <span className="block text-[12.5px] font-extrabold mb-2" style={{ color: 'var(--ink-soft)' }}>Category</span>
                <div className="grid grid-cols-2 gap-2.5">
                  {VBIZ_CATS.map((c) => {
                    const on = cat === c.id;
                    return (
                      <button key={c.id} onClick={() => setCat(c.id)}
                        className="flex items-center gap-2.5 rounded-2xl px-3 py-3 text-left transition-all active:scale-[0.98]"
                        style={{ background: on ? 'var(--accent)' : '#fff', border: `1.5px solid ${on ? 'var(--accent)' : 'var(--line)'}`, color: on ? '#fff' : 'var(--ink)' }}>
                        <Icon name={c.icon} size={18} />
                        <span className="font-extrabold text-[12.5px] leading-tight">{c.label}</span>
                      </button>
                    );
                  })}
                </div>
              </div>
            </div>
          )}

          {step === 2 && (
            <div className="space-y-3.5">
              <div className="flex gap-2.5">
                <div className="flex items-center gap-1.5 rounded-2xl px-3 flex-shrink-0" style={{ height: 50, background: '#fff', border: '1.5px solid var(--line)' }}>
                  <span className="text-[15px]">🇱🇰</span><span className="font-extrabold text-[15px]" style={{ color: 'var(--ink)' }}>+94</span>
                </div>
                <div className="flex items-center gap-2.5 rounded-2xl px-3.5 flex-1" style={{ height: 50, background: '#fff', border: '1.5px solid var(--line)' }}>
                  <Icon name="Phone" size={18} style={{ color: 'var(--muted)' }} />
                  <input inputMode="tel" placeholder="76 123 4891" className="flex-1 bg-transparent outline-none text-[15px] font-bold w-full" style={{ color: 'var(--ink)' }} />
                </div>
              </div>
              <VField label="Town / area" placeholder="e.g. Weligama" icon="MapPin" />
              <VField label="Tell travelers about your place" textarea placeholder="One or two lines on what makes it special…" />
            </div>
          )}

          <div className="flex gap-3 mt-6">
            {step > 0 && (
              <button onClick={() => setStep((s) => s - 1)} className="rounded-2xl px-5 py-3.5 font-extrabold text-[15px] active:scale-[0.98]" style={{ background: '#fff', border: '1.5px solid var(--line)', color: 'var(--ink)' }}>Back</button>
            )}
            <div className="flex-1">
              <VPrimary onClick={() => setStep((s) => s + 1)} icon={step === 2 ? 'Check' : 'ArrowRight'}>
                {step === 2 ? 'Submit application' : 'Continue'}
              </VPrimary>
            </div>
          </div>

          <p className="text-center text-[13.5px] font-bold mt-5" style={{ color: 'var(--muted)' }}>
            Already a partner? <TextLink onClick={() => setMode('login')}>Sign in</TextLink>
          </p>
        </>
      ) : (
        <div className="text-center">
          <div className="w-16 h-16 rounded-full flex items-center justify-center mx-auto mb-5" style={{ background: 'var(--accent-tint)' }}>
            <Icon name="PartyPopper" size={30} style={{ color: 'var(--accent)' }} />
          </div>
          <h1 className="font-extrabold leading-tight" style={{ fontSize: 26, color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Application received!</h1>
          <p className="text-[14.5px] font-semibold mt-2.5 mb-6 leading-relaxed" style={{ color: 'var(--muted)' }}>
            Our team reviews new listings within 24 hours. Meanwhile, take a look around your dashboard.
          </p>
          <div className="rounded-2xl p-4 mb-6 text-left flex items-center gap-3" style={{ background: 'var(--sand)', border: '1.5px solid var(--line)' }}>
            <Icon name="Clock" size={20} style={{ color: 'var(--accent)' }} />
            <div>
              <div className="text-[13.5px] font-extrabold" style={{ color: 'var(--ink)' }}>Status: Pending review</div>
              <div className="text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>We'll email you once you're verified.</div>
            </div>
          </div>
          <VPrimary onClick={onAuthed} icon="ArrowRight">Explore the dashboard</VPrimary>
        </div>
      )}
    </div>
  );
}

// ── brand panel ────────────────────────────────────────────────
function VBrandPanel() {
  const props = [
    { icon: 'Users', t: 'Reach travelers near you', s: 'Get discovered by nomads, surfers & explorers in the bay.' },
    { icon: 'Tag', t: 'Run offers in seconds', s: 'Launch perks, discounts and bundles — live instantly.' },
    { icon: 'QrCode', t: 'Validate with a scan', s: 'Redeem codes at the counter, track every claim.' },
  ];
  return (
    <div className="relative h-full flex flex-col justify-between overflow-hidden" style={{ padding: '48px 44px', background: 'linear-gradient(160deg, var(--accent) 0%, var(--accent-2) 100%)' }}>
      <div style={{ position: 'absolute', right: -60, top: -40, width: 280, height: 280, borderRadius: '50%', background: 'rgba(255,255,255,0.1)' }} />
      <div style={{ position: 'absolute', left: -50, bottom: -60, width: 220, height: 220, borderRadius: '50%', background: 'rgba(255,255,255,0.08)' }} />
      <div className="relative flex items-center gap-2.5">
        <VMark size={30} light />
        <div className="leading-none">
          <div className="font-extrabold text-[20px] tracking-tight text-white" style={{ fontFamily: 'var(--font-display)' }}>Spotto</div>
          <div className="text-[10px] font-extrabold uppercase tracking-[0.2em] mt-1" style={{ color: 'rgba(255,255,255,0.85)' }}>Vendor Portal</div>
        </div>
      </div>

      <div className="relative">
        <h2 className="font-extrabold text-white leading-[1.1]" style={{ fontSize: 32, fontFamily: 'var(--font-display)' }}>
          Turn passers-by<br />into regulars.
        </h2>
        <div className="mt-7 space-y-4">
          {props.map((p) => (
            <div key={p.t} className="flex items-start gap-3.5">
              <span className="w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0" style={{ background: 'rgba(255,255,255,0.18)', color: '#fff' }}>
                <Icon name={p.icon} size={19} />
              </span>
              <div>
                <div className="font-extrabold text-[15px] text-white">{p.t}</div>
                <div className="text-[13px] font-semibold leading-snug" style={{ color: 'rgba(255,255,255,0.82)' }}>{p.s}</div>
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="relative rounded-2xl px-4 py-3.5 flex items-center gap-3" style={{ background: 'rgba(255,255,255,0.15)' }}>
        <Icon name="TrendingUp" size={20} style={{ color: '#fff' }} />
        <p className="text-[13px] font-bold text-white leading-snug">Partners see <span className="font-extrabold">+18% redemptions</span> in their first month.</p>
      </div>
    </div>
  );
}

function VendorAuth({ onAuthed, layout = 'split' }) {
  const animStyle = (
    <style>{`
      @keyframes vdAuthIn { from { opacity: 0; transform: translateY(14px); } to { opacity: 1; transform: none; } }
      @media (prefers-reduced-motion: no-preference) { .vd-auth-anim { animation: vdAuthIn 0.5s cubic-bezier(0.22,1,0.36,1) both; } }
    `}</style>
  );

  if (layout === 'centered') {
    return (
      <div className="h-full w-full flex items-center justify-center px-6 overflow-y-auto" style={{ background: 'var(--sand)' }}>
        {animStyle}
        <div className="w-full flex flex-col items-center py-10" style={{ maxWidth: 440 }}>
          <div className="flex items-center gap-2.5 mb-7">
            <VMark size={30} />
            <div className="leading-none">
              <div className="font-extrabold text-[20px] tracking-tight" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Spotto</div>
              <div className="text-[10px] font-extrabold uppercase tracking-[0.18em] mt-1" style={{ color: 'var(--accent)' }}>Vendor Portal</div>
            </div>
          </div>
          <div className="w-full rounded-[28px] p-7" style={{ background: '#fff', boxShadow: '0 24px 56px -26px rgba(15,61,58,0.4)' }}>
            <VendorAuthForm onAuthed={onAuthed} />
          </div>
        </div>
      </div>
    );
  }

  // default: split
  return (
    <div className="h-full w-full flex" style={{ background: 'var(--sand)' }}>
      {animStyle}
      <div className="hidden md:block" style={{ flex: '0 0 46%' }}>
        <VBrandPanel />
      </div>
      <div className="flex-1 flex items-center justify-center px-7 overflow-y-auto" style={{ paddingTop: 32, paddingBottom: 32 }}>
        <VendorAuthForm onAuthed={onAuthed} />
      </div>
    </div>
  );
}

Object.assign(window, { VendorAuth });
