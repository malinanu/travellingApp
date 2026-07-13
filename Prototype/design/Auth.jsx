// Auth.jsx — Spotto consumer sign-in / sign-up gate (mobile).
// Sits BEFORE onboarding. Modes: login → register → phone → otp → forgot.
// 3 layouts via the `layout` tweak: 'hero' | 'immersive' | 'minimal'.

const { useState: useStateAuth, useRef: useRefAuth, useEffect: useEffectAuth } = React;

// ── brand marks for social providers ───────────────────────────
function GoogleG({ size = 19 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 48 48" aria-hidden="true">
      <path fill="#4285F4" d="M45.1 24.5c0-1.6-.1-3.1-.4-4.5H24v8.5h11.8c-.5 2.7-2 5-4.4 6.6v5.5h7.1c4.1-3.8 6.6-9.4 6.6-16.1z"/>
      <path fill="#34A853" d="M24 46c5.9 0 10.9-2 14.5-5.3l-7.1-5.5c-2 1.3-4.5 2.1-7.4 2.1-5.7 0-10.5-3.8-12.2-9h-7.3v5.7C8.1 41.1 15.4 46 24 46z"/>
      <path fill="#FBBC05" d="M11.8 28.3c-.4-1.3-.7-2.7-.7-4.3s.3-3 .7-4.3v-5.7H4.5C3 17.1 2.1 20.4 2.1 24s.9 6.9 2.4 9.7l7.3-5.4z"/>
      <path fill="#EA4335" d="M24 10.7c3.2 0 6.1 1.1 8.4 3.3l6.3-6.3C34.9 4.1 29.9 2 24 2 15.4 2 8.1 6.9 4.5 14.3l7.3 5.7c1.7-5.2 6.5-9.3 12.2-9.3z"/>
    </svg>
  );
}
function AppleLogo({ size = 19 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M16.4 12.8c0-2.3 1.9-3.4 2-3.5-1.1-1.6-2.8-1.8-3.4-1.8-1.4-.1-2.8.9-3.5.9-.7 0-1.9-.8-3.1-.8-1.6 0-3.1.9-3.9 2.4-1.7 2.9-.4 7.2 1.2 9.5.8 1.1 1.7 2.4 2.9 2.4 1.2-.1 1.6-.8 3-.8s1.8.8 3.1.7c1.3 0 2.1-1.1 2.9-2.3.9-1.3 1.3-2.6 1.3-2.7-.1 0-2.5-1-2.5-3.8zM14.1 5.9c.6-.8 1.1-1.9 1-3-.9.1-2.1.6-2.7 1.4-.6.7-1.2 1.8-1 2.9 1 .1 2-.5 2.7-1.3z"/>
    </svg>
  );
}

// ── small primitives ───────────────────────────────────────────
function Field({ label, type = 'text', value, onChange, placeholder, icon, reveal, onReveal, autoFocus }) {
  return (
    <label className="block">
      <span className="block text-[12.5px] font-extrabold mb-1.5" style={{ color: 'var(--ink-soft)' }}>{label}</span>
      <div className="flex items-center gap-2.5 rounded-2xl px-3.5"
           style={{ height: 52, background: '#fff', border: '1.5px solid rgba(15,61,58,0.12)' }}>
        {icon && <Icon name={icon} size={18} style={{ color: 'var(--muted)' }} />}
        <input
          type={type} value={value} placeholder={placeholder} autoFocus={autoFocus}
          onChange={(e) => onChange(e.target.value)}
          className="flex-1 bg-transparent outline-none text-[15px] font-bold w-full"
          style={{ color: 'var(--ink)' }} />
        {onReveal && (
          <button type="button" onClick={onReveal} className="flex-shrink-0" style={{ color: 'var(--muted)' }}>
            <Icon name={reveal ? 'EyeOff' : 'Eye'} size={18} />
          </button>
        )}
      </div>
    </label>
  );
}

function PrimaryBtn({ children, onClick, icon = 'ArrowRight' }) {
  return (
    <button onClick={onClick}
      className="w-full rounded-2xl py-4 font-extrabold text-[16px] flex items-center justify-center gap-2 whitespace-nowrap transition-all duration-200 active:scale-[0.98]"
      style={{ background: 'var(--accent)', color: '#fff', boxShadow: '0 14px 28px -12px var(--accent-shadow)' }}>
      {children}
      {icon && <Icon name={icon} size={19} strokeWidth={2.5} />}
    </button>
  );
}

function SocialButtons({ onSocial }) {
  const btn = (label, mark, onClick) => (
    <button onClick={onClick}
      className="flex-1 flex items-center justify-center gap-2 rounded-2xl py-3.5 font-extrabold text-[14px] transition-all active:scale-[0.97]"
      style={{ background: '#fff', border: '1.5px solid rgba(15,61,58,0.12)', color: 'var(--ink)' }}>
      {mark}
      <span>{label}</span>
    </button>
  );
  return (
    <div className="flex gap-2.5">
      {btn('Google', <GoogleG />, () => onSocial('google'))}
      {btn('Apple', <span style={{ color: 'var(--ink)' }}><AppleLogo /></span>, () => onSocial('apple'))}
    </div>
  );
}

function Divider({ label = 'or' }) {
  return (
    <div className="flex items-center gap-3 my-1">
      <div className="flex-1 h-px" style={{ background: 'rgba(15,61,58,0.12)' }} />
      <span className="text-[12px] font-bold" style={{ color: 'var(--muted)' }}>{label}</span>
      <div className="flex-1 h-px" style={{ background: 'rgba(15,61,58,0.12)' }} />
    </div>
  );
}

// 6-digit OTP entry
function OtpInput({ value, onChange }) {
  const refs = useRefAuth([]);
  const set = (i, ch) => {
    const d = ch.replace(/\D/g, '').slice(-1);
    const next = value.split('');
    next[i] = d;
    const joined = next.join('').slice(0, 6);
    onChange(joined);
    if (d && i < 5) refs.current[i + 1]?.focus();
  };
  const onKey = (i, e) => {
    if (e.key === 'Backspace' && !value[i] && i > 0) refs.current[i - 1]?.focus();
  };
  return (
    <div className="flex gap-2 justify-between">
      {[0, 1, 2, 3, 4, 5].map((i) => (
        <input key={i} ref={(el) => (refs.current[i] = el)} inputMode="numeric" maxLength={1}
          value={value[i] || ''} onChange={(e) => set(i, e.target.value)} onKeyDown={(e) => onKey(i, e)}
          className="text-center rounded-2xl outline-none font-extrabold transition-all"
          style={{ width: 48, height: 58, fontSize: 24, background: '#fff', color: 'var(--ink)',
                   border: `1.5px solid ${value[i] ? 'var(--accent)' : 'rgba(15,61,58,0.12)'}` }} />
      ))}
    </div>
  );
}

// ── the form body (mode machine) ───────────────────────────────
function AuthForm({ onAuthed, dark = false }) {
  const [mode, setMode] = useStateAuth('login');     // login | register | phone | otp | forgot
  const [show, setShow] = useStateAuth(false);
  const [remember, setRemember] = useStateAuth(true);
  const [otp, setOtp] = useStateAuth('');
  const [sent, setSent] = useStateAuth(false);
  const titleColor = dark ? '#fff' : 'var(--ink)';
  const subColor = dark ? 'rgba(255,255,255,0.75)' : 'var(--muted)';

  const finish = () => onAuthed();

  const TextLink = ({ children, onClick }) => (
    <button onClick={onClick} className="font-extrabold whitespace-nowrap" style={{ color: 'var(--accent)' }}>{children}</button>
  );

  // header copy per mode
  const heads = {
    login:    ['Welcome back', 'Pick up right where the bay left off.'],
    register: ['Create your account', 'Save spots, perks and your travel style.'],
    phone:    ['Continue with phone', "We'll text you a 6-digit code."],
    otp:      ['Enter the code', 'Sent to +94 76 •• •• 891'],
    forgot:   ['Reset password', "Enter your email and we'll send a link."],
  };
  const [h1, h2] = heads[mode];

  return (
    <div className="auth-anim">
      <div className="mb-5">
        <h1 className="font-extrabold leading-[1.05]" style={{ fontSize: 30, color: titleColor }}>{h1}</h1>
        <p className="mt-2 text-[14.5px] font-semibold leading-snug" style={{ color: subColor }}>{h2}</p>
      </div>

      {(mode === 'login' || mode === 'register') && (
        <div className="space-y-3.5">
          {mode === 'register' && (
            <Field label="Name" value="" onChange={() => {}} placeholder="Your name" icon="User" />
          )}
          <Field label="Email" type="email" value="" onChange={() => {}} placeholder="you@email.com" icon="Mail" />
          <Field label="Password" type={show ? 'text' : 'password'} value="" onChange={() => {}}
                 placeholder="••••••••" icon="Lock" reveal={show} onReveal={() => setShow((s) => !s)} />

          {mode === 'login' && (
            <div className="flex items-center justify-between pt-0.5">
              <button onClick={() => setRemember((r) => !r)} className="flex items-center gap-2">
                <span className="rounded-md flex items-center justify-center transition-all"
                      style={{ width: 20, height: 20,
                               background: remember ? 'var(--accent)' : '#fff',
                               border: `1.5px solid ${remember ? 'var(--accent)' : 'rgba(15,61,58,0.2)'}` }}>
                  {remember && <Icon name="Check" size={13} strokeWidth={3.5} style={{ color: '#fff' }} />}
                </span>
                <span className="text-[13px] font-bold" style={{ color: dark ? 'rgba(255,255,255,0.8)' : 'var(--ink-soft)' }}>Remember me</span>
              </button>
              <TextLink onClick={() => setMode('forgot')}>Forgot?</TextLink>
            </div>
          )}

          <div className="pt-1">
            <PrimaryBtn onClick={finish}>{mode === 'login' ? 'Log in' : 'Create account'}</PrimaryBtn>
          </div>

          {mode === 'register' && (
            <p className="text-[11.5px] font-semibold text-center leading-snug px-2" style={{ color: subColor }}>
              By continuing you agree to Spotto's <span style={{ color: 'var(--accent)' }}>Terms</span> & <span style={{ color: 'var(--accent)' }}>Privacy Policy</span>.
            </p>
          )}

          <Divider />
          <SocialButtons onSocial={finish} />
          <button onClick={() => { setSent(false); setMode('phone'); }}
            className="w-full flex items-center justify-center gap-2 rounded-2xl py-3.5 font-extrabold text-[14px] active:scale-[0.97] transition-all"
            style={{ background: dark ? 'rgba(255,255,255,0.1)' : 'var(--accent-tint)', color: 'var(--accent)' }}>
            <Icon name="Smartphone" size={17} /> Continue with phone
          </button>

          <p className="text-center text-[13.5px] font-bold pt-1" style={{ color: subColor }}>
            {mode === 'login' ? 'New to Spotto? ' : 'Already have an account? '}
            <TextLink onClick={() => setMode(mode === 'login' ? 'register' : 'login')}>
              {mode === 'login' ? 'Create account' : 'Log in'}
            </TextLink>
          </p>
        </div>
      )}

      {mode === 'phone' && (
        <div className="space-y-4">
          <div className="flex gap-2.5">
            <div className="flex items-center gap-1.5 rounded-2xl px-3.5 flex-shrink-0"
                 style={{ height: 52, background: '#fff', border: '1.5px solid rgba(15,61,58,0.12)' }}>
              <span className="text-[16px]">🇱🇰</span>
              <span className="font-extrabold text-[15px]" style={{ color: 'var(--ink)' }}>+94</span>
            </div>
            <div className="flex items-center gap-2.5 rounded-2xl px-3.5 flex-1"
                 style={{ height: 52, background: '#fff', border: '1.5px solid rgba(15,61,58,0.12)' }}>
              <Icon name="Smartphone" size={18} style={{ color: 'var(--muted)' }} />
              <input autoFocus inputMode="tel" placeholder="76 123 4891"
                     className="flex-1 bg-transparent outline-none text-[15px] font-bold w-full" style={{ color: 'var(--ink)' }} />
            </div>
          </div>
          <PrimaryBtn onClick={() => { setSent(true); setMode('otp'); }} icon="ArrowRight">Send code</PrimaryBtn>
          <button onClick={() => setMode('login')} className="w-full text-center text-[13.5px] font-extrabold" style={{ color: subColor }}>
            ← Back to log in
          </button>
        </div>
      )}

      {mode === 'otp' && (
        <div className="space-y-4">
          <OtpInput value={otp} onChange={setOtp} />
          <PrimaryBtn onClick={finish} icon="Check">Verify & continue</PrimaryBtn>
          <p className="text-center text-[13.5px] font-bold" style={{ color: subColor }}>
            Didn't get it? <TextLink onClick={() => {}}>Resend</TextLink>
          </p>
          <button onClick={() => setMode('phone')} className="w-full text-center text-[13.5px] font-extrabold" style={{ color: subColor }}>
            ← Change number
          </button>
        </div>
      )}

      {mode === 'forgot' && (
        <div className="space-y-4">
          {sent ? (
            <div className="rounded-2xl p-4 flex items-start gap-3" style={{ background: 'var(--accent-tint)' }}>
              <Icon name="MailCheck" size={20} style={{ color: 'var(--accent)' }} />
              <p className="text-[13.5px] font-bold leading-snug" style={{ color: 'var(--ink)' }}>
                Check your inbox — we've sent a reset link to your email.
              </p>
            </div>
          ) : (
            <Field label="Email" type="email" value="" onChange={() => {}} placeholder="you@email.com" icon="Mail" autoFocus />
          )}
          {!sent && <PrimaryBtn onClick={() => setSent(true)} icon="Send">Send reset link</PrimaryBtn>}
          <button onClick={() => { setSent(false); setMode('login'); }} className="w-full text-center text-[13.5px] font-extrabold" style={{ color: subColor }}>
            ← Back to log in
          </button>
        </div>
      )}
    </div>
  );
}

// ── brand lockup ───────────────────────────────────────────────
function BrandLockup({ light = false, tagline = true }) {
  return (
    <div>
      <div className="flex items-center gap-2.5">
        <SpottoMark size={34} />
        <span className="font-extrabold tracking-tight" style={{ fontSize: 26, color: light ? '#fff' : 'var(--ink)' }}>Spotto</span>
      </div>
      {tagline && (
        <p className="mt-3 font-bold text-[15px] leading-snug max-w-[260px]"
           style={{ color: light ? 'rgba(255,255,255,0.92)' : 'var(--muted)' }}>
          Your local friend in the bay — the good coffee, the right break, the spot for sunset.
        </p>
      )}
    </div>
  );
}

// ── layout shells ──────────────────────────────────────────────
function Auth({ onAuthed, layout = 'hero' }) {
  const animStyle = (
    <style>{`
      @keyframes authIn { from { opacity: 0; transform: translateY(14px); } to { opacity: 1; transform: none; } }
      @media (prefers-reduced-motion: no-preference) { .auth-anim { animation: authIn 0.5s cubic-bezier(0.22,1,0.36,1) both; } }
    `}</style>
  );

  if (layout === 'immersive') {
    return (
      <div className="h-full relative" style={{ background: '#103d3a' }}>
        {animStyle}
        <div style={{ position: 'absolute', inset: 0 }}><SatelliteMap time="evening" /></div>
        <div style={{ position: 'absolute', inset: 0, background: 'linear-gradient(180deg, rgba(16,61,58,0.35) 0%, rgba(16,61,58,0.55) 45%, rgba(16,61,58,0.92) 100%)' }} />
        <div className="relative h-full flex flex-col overflow-y-auto">
          <div className="px-7" style={{ paddingTop: 76 }}>
            <BrandLockup light />
          </div>
          <div className="flex-1" />
          <div className="px-5 pb-10 pt-6">
            <div className="rounded-[28px] p-6" style={{ background: 'rgba(255,255,255,0.97)', boxShadow: '0 24px 60px -20px rgba(0,0,0,0.6)', backdropFilter: 'blur(8px)' }}>
              <AuthForm onAuthed={onAuthed} />
            </div>
          </div>
        </div>
      </div>
    );
  }

  if (layout === 'minimal') {
    return (
      <div className="h-full overflow-y-auto" style={{ background: 'var(--sand)' }}>
        {animStyle}
        <div className="min-h-full flex flex-col px-7" style={{ paddingTop: 92, paddingBottom: 40 }}>
          <div className="flex items-center gap-2.5 mb-9">
            <SpottoMark size={30} />
            <span className="font-extrabold tracking-tight" style={{ fontSize: 22, color: 'var(--ink)' }}>Spotto</span>
          </div>
          <AuthForm onAuthed={onAuthed} />
        </div>
      </div>
    );
  }

  // default: hero
  return (
    <div className="h-full overflow-y-auto" style={{ background: 'var(--sand)' }}>
      {animStyle}
      <div className="relative" style={{ height: 286 }}>
        <Illustration kind="sunset" scrim={false} />
        <div className="absolute inset-0" style={{ background: 'linear-gradient(180deg, transparent 55%, var(--sand) 100%)' }} />
        <div className="absolute left-0 right-0 px-7" style={{ top: 60 }}>
          <BrandLockup tagline={false} />
        </div>
      </div>
      <div className="px-6 pb-10" style={{ marginTop: -26 }}>
        <div className="rounded-[28px] p-6" style={{ background: '#fff', boxShadow: '0 20px 44px -22px rgba(16,61,58,0.35)' }}>
          <AuthForm onAuthed={onAuthed} />
        </div>
      </div>
    </div>
  );
}

Object.assign(window, { Auth });
