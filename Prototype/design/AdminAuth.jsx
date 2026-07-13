// AdminAuth.jsx — Spotto Command Center sign-in. Always dark/branded.
// Flow: credentials → 2FA verification code → onAuthed.
// Layouts via `layout` tweak: 'split' | 'centered'.

const { useState: useStateAdAuth, useRef: useRefAdAuth } = React;

const AD_ACCENT = 'var(--accent)';

function AdLogo({ size = 28, accent }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" fill="none">
      <path d="M16 3C10.5 3 6 7.3 6 12.7 6 19.5 16 29 16 29s10-9.5 10-16.3C26 7.3 21.5 3 16 3z" fill={accent} />
      <circle cx="16" cy="12.5" r="4.4" fill="#f6b81e" />
    </svg>
  );
}

function AdField({ label, type = 'text', placeholder, icon, reveal, onReveal, autoFocus }) {
  return (
    <label className="block">
      <span className="block text-[10.5px] font-bold uppercase tracking-[0.13em] mb-1.5" style={{ color: '#7f9d97' }}>{label}</span>
      <div className="flex items-center gap-2.5 rounded-lg px-3"
           style={{ height: 46, background: '#0b2320', border: '1px solid rgba(255,255,255,0.1)' }}>
        {icon && <Icon name={icon} size={16} style={{ color: '#7f9d97' }} />}
        <input type={type} placeholder={placeholder} autoFocus={autoFocus}
          className="flex-1 bg-transparent outline-none text-[14px] font-medium w-full" style={{ color: '#eaf2ef' }} />
        {onReveal && (
          <button type="button" onClick={onReveal} style={{ color: '#7f9d97' }}>
            <Icon name={reveal ? 'EyeOff' : 'Eye'} size={16} />
          </button>
        )}
      </div>
    </label>
  );
}

function AdPrimary({ children, onClick, accent, icon }) {
  return (
    <button onClick={onClick}
      className="w-full rounded-lg py-3 font-bold text-[14px] flex items-center justify-center gap-2 whitespace-nowrap transition-all active:scale-[0.99]"
      style={{ background: accent, color: '#fff', boxShadow: `0 10px 26px -12px ${accent}` }}>
      {children}
      {icon && <Icon name={icon} size={16} strokeWidth={2.5} />}
    </button>
  );
}

function AdCodeInput({ value, onChange, accent }) {
  const refs = useRefAdAuth([]);
  const set = (i, ch) => {
    const d = ch.replace(/\D/g, '').slice(-1);
    const next = value.split(''); next[i] = d;
    const joined = next.join('').slice(0, 6); onChange(joined);
    if (d && i < 5) refs.current[i + 1]?.focus();
  };
  const onKey = (i, e) => { if (e.key === 'Backspace' && !value[i] && i > 0) refs.current[i - 1]?.focus(); };
  return (
    <div className="flex gap-2 justify-between">
      {[0, 1, 2, 3, 4, 5].map((i) => (
        <input key={i} ref={(el) => (refs.current[i] = el)} inputMode="numeric" maxLength={1}
          value={value[i] || ''} onChange={(e) => set(i, e.target.value)} onKeyDown={(e) => onKey(i, e)}
          className="font-mono text-center rounded-lg outline-none transition-all"
          style={{ width: 46, height: 54, fontSize: 22, fontWeight: 600, color: '#eaf2ef', background: '#0b2320',
                   border: `1px solid ${value[i] ? accent : 'rgba(255,255,255,0.12)'}` }} />
      ))}
    </div>
  );
}

function AdminAuthForm({ onAuthed, accent }) {
  const [stage, setStage] = useStateAdAuth('login');   // login | twofa | forgot
  const [show, setShow] = useStateAdAuth(false);
  const [trust, setTrust] = useStateAdAuth(true);
  const [code, setCode] = useStateAdAuth('');
  const [sent, setSent] = useStateAdAuth(false);

  return (
    <div className="ad-auth-anim w-full" style={{ maxWidth: 380 }}>
      <div className="flex items-center gap-2 mb-1.5">
        <span className="font-mono text-[10.5px] font-semibold px-2 py-0.5 rounded" style={{ background: 'rgba(255,255,255,0.06)', color: '#7f9d97' }}>
          {stage === 'twofa' ? 'STEP 2 / 2' : 'SECURE SIGN-IN'}
        </span>
        <span className="flex items-center gap-1 text-[10.5px] font-bold" style={{ color: '#16a34a' }}>
          <span className="w-1.5 h-1.5 rounded-full" style={{ background: '#16a34a', boxShadow: '0 0 0 3px rgba(22,163,74,0.2)' }} /> Encrypted
        </span>
      </div>

      {stage === 'login' && (
        <>
          <h1 className="font-bold tracking-tight leading-tight" style={{ fontSize: 25, color: '#eaf2ef' }}>Sign in to the Command Center</h1>
          <p className="text-[13px] font-medium mt-1.5 mb-6" style={{ color: '#7f9d97' }}>Operator access · admin & moderation tools.</p>
          <div className="space-y-3.5">
            <AdField label="Work email" type="email" placeholder="you@spotto.app" icon="Mail" autoFocus />
            <AdField label="Password" type={show ? 'text' : 'password'} placeholder="••••••••••" icon="Lock" reveal={show} onReveal={() => setShow((s) => !s)} />
            <div className="flex items-center justify-between pt-0.5">
              <button onClick={() => setTrust((r) => !r)} className="flex items-center gap-2">
                <span className="rounded flex items-center justify-center" style={{ width: 18, height: 18, background: trust ? accent : 'transparent', border: `1px solid ${trust ? accent : 'rgba(255,255,255,0.25)'}` }}>
                  {trust && <Icon name="Check" size={12} strokeWidth={3.5} style={{ color: '#fff' }} />}
                </span>
                <span className="text-[12.5px] font-semibold" style={{ color: '#c3d6d1' }}>Remember device</span>
              </button>
              <button onClick={() => { setSent(false); setStage('forgot'); }} className="text-[12.5px] font-bold whitespace-nowrap" style={{ color: accent }}>Forgot?</button>
            </div>
            <div className="pt-1.5">
              <AdPrimary onClick={() => setStage('twofa')} accent={accent} icon="ArrowRight">Continue</AdPrimary>
            </div>
          </div>
          <div className="mt-5 pt-4 flex items-center gap-2.5" style={{ borderTop: '1px solid rgba(255,255,255,0.08)' }}>
            <Icon name="ShieldCheck" size={15} style={{ color: '#7f9d97' }} />
            <p className="text-[11.5px] font-medium" style={{ color: '#7f9d97' }}>
              Admin access is invite-only. <span style={{ color: accent }}>Request access</span>
            </p>
          </div>
        </>
      )}

      {stage === 'twofa' && (
        <>
          <h1 className="font-bold tracking-tight leading-tight" style={{ fontSize: 25, color: '#eaf2ef' }}>Two-factor verification</h1>
          <p className="text-[13px] font-medium mt-1.5 mb-6" style={{ color: '#7f9d97' }}>
            Enter the 6-digit code from your authenticator app.
          </p>
          <AdCodeInput value={code} onChange={setCode} accent={accent} />
          <label className="flex items-center gap-2 mt-4 mb-5 cursor-pointer" onClick={() => setTrust((r) => !r)}>
            <span className="rounded flex items-center justify-center" style={{ width: 18, height: 18, background: trust ? accent : 'transparent', border: `1px solid ${trust ? accent : 'rgba(255,255,255,0.25)'}` }}>
              {trust && <Icon name="Check" size={12} strokeWidth={3.5} style={{ color: '#fff' }} />}
            </span>
            <span className="text-[12.5px] font-semibold" style={{ color: '#c3d6d1' }}>Trust this device for 30 days</span>
          </label>
          <AdPrimary onClick={onAuthed} accent={accent} icon="ArrowRight">Verify & enter</AdPrimary>
          <div className="flex items-center justify-between mt-4 text-[12.5px] font-semibold">
            <button onClick={() => setStage('login')} style={{ color: '#7f9d97' }}>← Back</button>
            <button style={{ color: accent }}>Use SMS instead</button>
          </div>
        </>
      )}

      {stage === 'forgot' && (
        <>
          <h1 className="font-bold tracking-tight leading-tight" style={{ fontSize: 25, color: '#eaf2ef' }}>Reset your password</h1>
          <p className="text-[13px] font-medium mt-1.5 mb-6" style={{ color: '#7f9d97' }}>We'll email a secure reset link to your work address.</p>
          {sent ? (
            <div className="rounded-lg p-4 flex items-start gap-3 mb-5" style={{ background: 'rgba(22,163,74,0.1)', border: '1px solid rgba(22,163,74,0.25)' }}>
              <Icon name="MailCheck" size={18} style={{ color: '#16a34a' }} />
              <p className="text-[12.5px] font-medium leading-snug" style={{ color: '#c3d6d1' }}>If an account exists, a reset link is on its way.</p>
            </div>
          ) : (
            <div className="mb-5"><AdField label="Work email" type="email" placeholder="you@spotto.app" icon="Mail" autoFocus /></div>
          )}
          {!sent && <AdPrimary onClick={() => setSent(true)} accent={accent} icon="Send">Send reset link</AdPrimary>}
          <button onClick={() => { setSent(false); setStage('login'); }} className="w-full text-center text-[12.5px] font-bold mt-4" style={{ color: '#7f9d97' }}>← Back to sign in</button>
        </>
      )}
    </div>
  );
}

// brand / marketing panel
function AdBrandPanel({ accent }) {
  const kpis = (window.ADMIN_KPIS || []).slice(0, 4);
  return (
    <div className="relative h-full flex flex-col justify-between overflow-hidden" style={{ padding: '48px 44px' }}>
      {/* glow + grid */}
      <div style={{ position: 'absolute', inset: 0, background: `radial-gradient(120% 90% at 15% 0%, ${accent}22, transparent 60%)` }} />
      <div style={{ position: 'absolute', inset: 0, backgroundImage: 'linear-gradient(rgba(255,255,255,0.04) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.04) 1px, transparent 1px)', backgroundSize: '44px 44px', maskImage: 'radial-gradient(80% 80% at 30% 20%, #000, transparent)' }} />
      <div className="relative">
        <div className="flex items-center gap-3">
          <AdLogo size={32} accent={accent} />
          <div className="leading-none">
            <div className="font-bold text-[20px] tracking-tight text-white">Spotto</div>
            <div className="text-[10px] font-bold uppercase tracking-[0.2em] mt-1" style={{ color: accent }}>Command Center</div>
          </div>
        </div>
      </div>

      <div className="relative">
        <h2 className="font-bold leading-[1.12] text-white" style={{ fontSize: 30 }}>
          Run the platform<br />from one console.
        </h2>
        <p className="text-[14px] font-medium mt-3 leading-relaxed" style={{ color: '#9db4ae', maxWidth: 320 }}>
          Approve vendors, watch live content, track geo-demand and subscriptions across the south coast — in real time.
        </p>
        <div className="grid grid-cols-2 gap-3 mt-7" style={{ maxWidth: 340 }}>
          {kpis.map((k) => {
            const c = { amber: '#d97706', red: '#e11d48', teal: accent, green: '#16a34a' }[k.tone] || accent;
            return (
              <div key={k.id} className="rounded-xl p-3.5" style={{ background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.07)' }}>
                <div className="flex items-center gap-2">
                  <span className="w-7 h-7 rounded-md flex items-center justify-center" style={{ background: c + '22', color: c }}><Icon name={k.icon} size={14} /></span>
                  <span className="font-bold text-[18px] text-white">{k.value}</span>
                </div>
                <div className="text-[11px] font-semibold mt-2" style={{ color: '#7f9d97' }}>{k.label}</div>
              </div>
            );
          })}
        </div>
      </div>

      <div className="relative flex items-center gap-2 text-[11.5px] font-medium" style={{ color: '#6d8a84' }}>
        <Icon name="Activity" size={13} style={{ color: '#16a34a' }} /> All systems operational · API 42ms · 99.98% uptime
      </div>
    </div>
  );
}

function AdminAuth({ onAuthed, accent, layout = 'split' }) {
  const animStyle = (
    <style>{`
      @keyframes adAuthIn { from { opacity: 0; transform: translateY(12px); } to { opacity: 1; transform: none; } }
      @media (prefers-reduced-motion: no-preference) { .ad-auth-anim { animation: adAuthIn 0.5s cubic-bezier(0.22,1,0.36,1) both; } }
    `}</style>
  );

  if (layout === 'centered') {
    return (
      <div className="h-full w-full flex items-center justify-center relative overflow-hidden" style={{ background: '#0b2320' }}>
        {animStyle}
        <div style={{ position: 'absolute', inset: 0, background: `radial-gradient(90% 70% at 50% -10%, ${accent}1f, transparent 55%)` }} />
        <div style={{ position: 'absolute', inset: 0, backgroundImage: 'linear-gradient(rgba(255,255,255,0.035) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.035) 1px, transparent 1px)', backgroundSize: '46px 46px', maskImage: 'radial-gradient(70% 60% at 50% 30%, #000, transparent)' }} />
        <div className="relative w-full flex flex-col items-center px-6" style={{ maxWidth: 440 }}>
          <div className="flex items-center gap-2.5 mb-7">
            <AdLogo size={30} accent={accent} />
            <div className="leading-none">
              <div className="font-bold text-[18px] tracking-tight text-white">Spotto</div>
              <div className="text-[9.5px] font-bold uppercase tracking-[0.2em] mt-1" style={{ color: accent }}>Command Center</div>
            </div>
          </div>
          <div className="w-full rounded-2xl p-7" style={{ background: '#10302c', border: '1px solid rgba(255,255,255,0.08)', boxShadow: '0 30px 70px -30px rgba(0,0,0,0.8)' }}>
            <AdminAuthForm onAuthed={onAuthed} accent={accent} />
          </div>
        </div>
      </div>
    );
  }

  // default: split
  return (
    <div className="h-full w-full flex" style={{ background: '#0b2320' }}>
      {animStyle}
      <div className="hidden md:block" style={{ flex: '0 0 50%', background: '#103d3a', borderRight: '1px solid rgba(255,255,255,0.07)' }}>
        <AdBrandPanel accent={accent} />
      </div>
      <div className="flex-1 flex items-center justify-center px-7">
        <AdminAuthForm onAuthed={onAuthed} accent={accent} />
      </div>
    </div>
  );
}

Object.assign(window, { AdminAuth });
