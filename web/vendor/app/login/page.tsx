'use client';

// Vendor auth — sign in, forgot/reset password, Google sign-in, and a
// 3-step "list your business" registration wizard (registration creates a
// pending vendor that an admin approves from the command center). A real
// TOTP step is inserted after credentials whenever the account has 2FA
// enabled (set up from Account → Two-factor authentication). Port of the
// VendorAuth mockup — Apple sign-in is intentionally not included, see
// SKIPPED.md.

import { API_URL, isTwoFactorRequired, type AuthResponse, type TwoFactorRequiredResponse } from '@spotto/shared';
import {
  ArrowLeft, ArrowRight, Check, Clock, Eye, EyeOff, Lock, Mail, MapPin,
  MailCheck, PartyPopper, Phone, QrCode, Send, Store, Tag,
  TrendingUp, User as UserIcon, Users, LucideIcon,
} from 'lucide-react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useRef, useState } from 'react';

import { SpottoMark } from '../../components/shell';
import { api } from '../../lib/api';
import { BUSINESS_CATEGORIES } from '../../lib/business-categories';

const REGISTER_STEPS = ['Your account', 'Your business', 'A few details'];

function GoogleG({ size = 18 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 48 48" aria-hidden="true">
      <path fill="#4285F4" d="M45.1 24.5c0-1.6-.1-3.1-.4-4.5H24v8.5h11.8c-.5 2.7-2 5-4.4 6.6v5.5h7.1c4.1-3.8 6.6-9.4 6.6-16.1z" />
      <path fill="#34A853" d="M24 46c5.9 0 10.9-2 14.5-5.3l-7.1-5.5c-2 1.3-4.5 2.1-7.4 2.1-5.7 0-10.5-3.8-12.2-9h-7.3v5.7C8.1 41.1 15.4 46 24 46z" />
      <path fill="#FBBC05" d="M11.8 28.3c-.4-1.3-.7-2.7-.7-4.3s.3-3 .7-4.3v-5.7H4.5C3 17.1 2.1 20.4 2.1 24s.9 6.9 2.4 9.7l7.3-5.4z" />
      <path fill="#EA4335" d="M24 10.7c3.2 0 6.1 1.1 8.4 3.3l6.3-6.3C34.9 4.1 29.9 2 24 2 15.4 2 8.1 6.9 4.5 14.3l7.3 5.7c1.7-5.2 6.5-9.3 12.2-9.3z" />
    </svg>
  );
}

function Field({
  label, type = 'text', value, onChange, icon: Icon, reveal, onReveal, autoFocus, placeholder,
}: {
  label: string; type?: string; value: string; onChange: (v: string) => void; icon?: LucideIcon;
  reveal?: boolean; onReveal?: () => void; autoFocus?: boolean; placeholder?: string;
}) {
  return (
    <label className="block">
      <span className="block text-[12.5px] font-extrabold mb-1.5 text-ink-soft">{label}</span>
      <div className="flex items-center gap-2.5 rounded-2xl px-3.5 h-[50px] bg-white border-[1.5px] border-line">
        {Icon && <Icon size={18} className="text-muted shrink-0" />}
        <input type={type} value={value} onChange={(e) => onChange(e.target.value)} autoFocus={autoFocus} placeholder={placeholder}
          className="flex-1 bg-transparent outline-none text-[15px] font-bold w-full text-ink" />
        {onReveal && (
          <button type="button" onClick={onReveal} className="text-muted shrink-0">
            {reveal ? <EyeOff size={18} /> : <Eye size={18} />}
          </button>
        )}
      </div>
    </label>
  );
}

function Primary({ children, onClick, disabled, icon: Icon = ArrowRight, type }: {
  children: React.ReactNode; onClick?: () => void; disabled?: boolean; icon?: LucideIcon | null; type?: 'submit' | 'button';
}) {
  return (
    <button type={type ?? 'button'} onClick={onClick} disabled={disabled}
      className="btn-accent w-full rounded-2xl py-3.5 text-[15.5px] flex items-center justify-center gap-2 whitespace-nowrap transition-transform active:scale-[0.98] disabled:opacity-60"
      style={{ boxShadow: '0 14px 28px -12px var(--accent-shadow, rgba(11,111,134,0.35))' }}>
      {children}
      {Icon && <Icon size={18} strokeWidth={2.5} />}
    </button>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={null}>
      <LoginPageInner />
    </Suspense>
  );
}

function LoginPageInner() {
  const router = useRouter();
  const searchParams = useSearchParams();

  const [mode, setMode] = useState<'login' | 'register' | 'forgot'>('login');
  const [registerStep, setRegisterStep] = useState(0);
  const [otpMode, setOtpMode] = useState(false);
  const [twoFactorToken, setTwoFactorToken] = useState<string | null>(null);
  const [otp, setOtp] = useState(['', '', '', '', '', '']);
  const otpRefs = useRef<(HTMLInputElement | null)[]>([]);

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [remember, setRemember] = useState(true);

  const [name, setName] = useState('');
  const [businessName, setBusinessName] = useState('');
  const [category, setCategory] = useState('cafe');
  const [phone, setPhone] = useState('');
  const [town, setTown] = useState('Weligama');
  const [description, setDescription] = useState('');

  const [forgotEmail, setForgotEmail] = useState('');
  const [forgotSent, setForgotSent] = useState(false);
  const [resetToken, setResetToken] = useState<string | null>(null);
  const [newPassword, setNewPassword] = useState('');
  const [resetDone, setResetDone] = useState(false);

  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  // One-time handling of whatever landed us here via a redirect: an OAuth
  // exchange code, a password-reset link, an OAuth "no account, please
  // register" bounce, or an OAuth error.
  useEffect(() => {
    const authCode = searchParams.get('authCode');
    const rToken = searchParams.get('resetToken');
    const oauthMode = searchParams.get('mode');
    const oauthError = searchParams.get('error');

    if (authCode) {
      setBusy(true);
      api.post<AuthResponse | TwoFactorRequiredResponse>('/auth/exchange', { code: authCode })
        .then((res) => {
          if (isTwoFactorRequired(res)) {
            setTwoFactorToken(res.twoFactorToken);
            setOtpMode(true);
            setTimeout(() => otpRefs.current[0]?.focus(), 50);
          } else {
            finishSession(res, true);
          }
        })
        .catch((err) => setError(err instanceof Error ? err.message : 'Google sign-in failed'))
        .finally(() => { setBusy(false); router.replace('/login'); });
      return;
    }
    if (rToken) {
      setResetToken(rToken);
      setMode('forgot');
      router.replace('/login');
      return;
    }
    if (oauthMode === 'register') {
      setMode('register');
      setRegisterStep(0);
      setEmail(searchParams.get('email') ?? '');
      setName(searchParams.get('name') ?? '');
      router.replace('/login');
      return;
    }
    if (oauthError === 'not_a_vendor_account') {
      setError('This portal is for business accounts. Use the Spotto app instead.');
      router.replace('/login');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const finishSession = (res: AuthResponse, rememberOverride?: boolean) => {
    if (res.user.role !== 'vendor') {
      api.setToken(null);
      setError('This portal is for business accounts. Use the Spotto app instead.');
      return;
    }
    api.setToken(res.token, rememberOverride ?? remember);
    router.replace('/dashboard');
  };

  const submitLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const res = await api.post<AuthResponse | TwoFactorRequiredResponse>('/auth/login', { email, password });
      if (isTwoFactorRequired(res)) {
        setTwoFactorToken(res.twoFactorToken);
        setOtpMode(true);
        setTimeout(() => otpRefs.current[0]?.focus(), 50);
        return;
      }
      finishSession(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong');
    } finally {
      setBusy(false);
    }
  };

  const submitRegister = async () => {
    setBusy(true);
    setError(null);
    try {
      const res = await api.post<AuthResponse | TwoFactorRequiredResponse>('/auth/register', {
        email, password, name, role: 'vendor', businessName, category,
        phone: phone ? `+94 ${phone}` : '', town, description,
      });
      if (isTwoFactorRequired(res)) {
        setTwoFactorToken(res.twoFactorToken);
        setOtpMode(true);
        setTimeout(() => otpRefs.current[0]?.focus(), 50);
        return;
      }
      // Hold onto the session but don't navigate yet — show the
      // "Application received" confirmation first (registerStep 3).
      api.setToken(res.token, remember);
      setRegisterStep(3);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong');
      api.setToken(null);
    } finally {
      setBusy(false);
    }
  };

  const onOtpChange = (i: number, v: string) => {
    const digit = v.replace(/\D/g, '').slice(-1);
    const next = [...otp];
    next[i] = digit;
    setOtp(next);
    if (digit && i < 5) otpRefs.current[i + 1]?.focus();
  };

  const submitOtp = async () => {
    if (otp.join('').length !== 6 || !twoFactorToken) return;
    setBusy(true);
    setError(null);
    try {
      const res = await api.post<AuthResponse>('/auth/2fa/login-verify', { twoFactorToken, code: otp.join('') });
      finishSession(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Invalid code');
      setOtp(['', '', '', '', '', '']);
      otpRefs.current[0]?.focus();
    } finally {
      setBusy(false);
    }
  };

  const submitForgot = async () => {
    setBusy(true);
    setError(null);
    try {
      await api.post('/auth/forgot-password', { email: forgotEmail });
      setForgotSent(true);
    } catch {
      // handleForgotPassword always 200s — this only fails on network error.
      setError('Something went wrong — try again in a moment.');
    } finally {
      setBusy(false);
    }
  };

  const submitReset = async () => {
    setBusy(true);
    setError(null);
    try {
      await api.post('/auth/reset-password', { token: resetToken, newPassword });
      setResetDone(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not reset password');
    } finally {
      setBusy(false);
    }
  };

  if (otpMode) {
    return (
      <AuthShell>
        <h1 className="font-display font-extrabold leading-tight text-[28px]">Two-factor verification</h1>
        <p className="text-[14px] text-muted font-semibold mt-1.5 mb-6">Enter the 6-digit code from your authenticator app.</p>
        <div className="flex gap-2 mb-4">
          {otp.map((d, i) => (
            <input key={i} ref={(el) => { otpRefs.current[i] = el; }} value={d} onChange={(e) => onOtpChange(i, e.target.value)}
              inputMode="numeric" maxLength={1} className="input text-center text-xl font-extrabold w-12 h-14" />
          ))}
        </div>
        {error && <p className="text-[13.5px] font-bold text-coral mb-3">{error}</p>}
        <Primary onClick={submitOtp} disabled={otp.join('').length !== 6 || busy} icon={busy ? null : ArrowRight}>
          {busy ? 'Verifying…' : 'Verify & continue'}
        </Primary>
      </AuthShell>
    );
  }

  if (mode === 'forgot') {
    if (resetToken) {
      return (
        <AuthShell>
          <h1 className="font-display font-extrabold leading-tight text-[28px]">Set a new password</h1>
          <p className="text-[14px] text-muted font-semibold mt-1.5 mb-6">Choose a new password for your account.</p>
          {resetDone ? (
            <div className="rounded-2xl bg-accent-tint p-4 flex items-start gap-3 mb-5">
              <MailCheck size={20} className="text-accent shrink-0" />
              <p className="text-[13.5px] font-bold leading-snug">Password updated — sign in with your new password.</p>
            </div>
          ) : (
            <div className="mb-5">
              <Field label="New password" type="password" value={newPassword} onChange={setNewPassword} icon={Lock} placeholder="Min 10 characters" autoFocus />
            </div>
          )}
          {error && <p className="text-[13.5px] font-bold text-coral mb-3">{error}</p>}
          {resetDone ? (
            <Primary onClick={() => { setMode('login'); setResetToken(null); setResetDone(false); }}>Back to sign in</Primary>
          ) : (
            <Primary onClick={submitReset} disabled={newPassword.length < 10 || busy} icon={Send}>
              {busy ? 'Saving…' : 'Save new password'}
            </Primary>
          )}
        </AuthShell>
      );
    }
    return (
      <AuthShell>
        <h1 className="font-display font-extrabold leading-tight text-[28px]">Reset password</h1>
        <p className="text-[14px] text-muted font-semibold mt-1.5 mb-6">We&rsquo;ll email you a secure reset link.</p>
        {forgotSent ? (
          <div className="rounded-2xl bg-accent-tint p-4 flex items-start gap-3 mb-5">
            <MailCheck size={20} className="text-accent shrink-0" />
            <p className="text-[13.5px] font-bold leading-snug">Check your inbox — the reset link is on its way.</p>
          </div>
        ) : (
          <div className="mb-5">
            <Field label="Email" type="email" value={forgotEmail} onChange={setForgotEmail} icon={Mail} placeholder="you@yourbusiness.lk" autoFocus />
          </div>
        )}
        {error && <p className="text-[13.5px] font-bold text-coral mb-3">{error}</p>}
        {!forgotSent && (
          <Primary onClick={submitForgot} disabled={!forgotEmail || busy} icon={Send}>
            {busy ? 'Sending…' : 'Send reset link'}
          </Primary>
        )}
        <button onClick={() => { setForgotSent(false); setMode('login'); }} className="w-full text-center text-[13.5px] font-extrabold mt-4 text-muted">
          ← Back to sign in
        </button>
      </AuthShell>
    );
  }

  if (mode === 'register') {
    if (registerStep >= 3) {
      return (
        <AuthShell wide>
          <div className="text-center">
            <div className="w-16 h-16 rounded-full bg-accent-tint flex items-center justify-center mx-auto mb-5">
              <PartyPopper size={30} className="text-accent" />
            </div>
            <h1 className="font-display font-extrabold leading-tight text-[26px]">Application received!</h1>
            <p className="text-[14.5px] text-muted font-semibold mt-2.5 mb-6 leading-relaxed">
              Our team reviews new listings within 24 hours. Meanwhile, take a look around your dashboard.
            </p>
            <div className="rounded-2xl bg-sand border-[1.5px] border-line p-4 mb-6 text-left flex items-center gap-3">
              <Clock size={20} className="text-accent shrink-0" />
              <div>
                <p className="text-[13.5px] font-extrabold">Status: Pending review</p>
                <p className="text-[12.5px] text-muted font-semibold">We&rsquo;ll email you once you&rsquo;re verified.</p>
              </div>
            </div>
            <Primary onClick={() => router.replace('/dashboard')}>Explore the dashboard</Primary>
          </div>
        </AuthShell>
      );
    }

    return (
      <AuthShell wide>
        <div className="flex items-center gap-2 mb-1">
          <span className="text-[11.5px] font-extrabold uppercase tracking-wider text-accent">Step {registerStep + 1} of 3</span>
        </div>
        <div className="flex gap-1.5 mb-5">
          {[0, 1, 2].map((i) => (
            <div key={i} className={`h-1.5 rounded-full flex-1 transition-colors ${i <= registerStep ? 'bg-accent' : 'bg-[rgba(15,61,58,0.12)]'}`} />
          ))}
        </div>
        <h1 className="font-display font-extrabold leading-tight text-[25px] mb-5">{REGISTER_STEPS[registerStep]}</h1>

        {registerStep === 0 && (
          <div className="space-y-3.5">
            <Field label="Owner / contact name" value={name} onChange={setName} icon={UserIcon} placeholder="e.g. Tharindu Perera" autoFocus />
            <Field label="Email" type="email" value={email} onChange={setEmail} icon={Mail} placeholder="you@yourbusiness.lk" />
            <Field label="Password" type={showPassword ? 'text' : 'password'} value={password} onChange={setPassword} icon={Lock}
              reveal={showPassword} onReveal={() => setShowPassword((s) => !s)} placeholder="Create a password (min 10 characters)" />
          </div>
        )}

        {registerStep === 1 && (
          <div className="space-y-4">
            <Field label="Business name" value={businessName} onChange={setBusinessName} icon={Store} placeholder="e.g. Driftwood Coffee & Co-Work" autoFocus />
            <div>
              <span className="block text-[12.5px] font-extrabold mb-2 text-ink-soft">Category</span>
              <div className="grid grid-cols-2 gap-2.5">
                {BUSINESS_CATEGORIES.map((t) => {
                  const Icon = t.icon;
                  const on = category === t.id;
                  return (
                    <button key={t.id} type="button" onClick={() => setCategory(t.id)}
                      className={`flex items-center gap-2.5 rounded-2xl px-3 py-3 text-left transition-all active:scale-[0.98] border-[1.5px] ${
                        on ? 'bg-accent border-accent text-white' : 'bg-white border-line text-ink'
                      }`}>
                      <Icon size={18} />
                      <span className="font-extrabold text-[12.5px] leading-tight">{t.label}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          </div>
        )}

        {registerStep === 2 && (
          <div className="space-y-3.5">
            <div className="flex gap-2.5">
              <div className="flex items-center gap-1.5 rounded-2xl px-3 h-[50px] bg-white border-[1.5px] border-line shrink-0">
                <span className="text-[15px]">🇱🇰</span><span className="font-extrabold text-[15px]">+94</span>
              </div>
              <div className="flex-1">
                <Field label="Phone number" type="tel" value={phone} onChange={setPhone} icon={Phone} placeholder="76 123 4891" />
              </div>
            </div>
            <Field label="Town / area" value={town} onChange={setTown} icon={MapPin} placeholder="e.g. Weligama" />
            <label className="block">
              <span className="block text-[12.5px] font-extrabold mb-1.5 text-ink-soft">Tell travelers about your place</span>
              <textarea rows={3} value={description} onChange={(e) => setDescription(e.target.value)}
                placeholder="One or two lines on what makes it special…"
                className="w-full rounded-2xl px-3.5 py-3 bg-white border-[1.5px] border-line outline-none text-[15px] font-bold resize-none text-ink" />
            </label>
          </div>
        )}

        {error && <p className="text-[13.5px] font-bold text-coral mt-4">{error}</p>}

        <div className="flex gap-3 mt-6">
          {registerStep > 0 && (
            <button type="button" onClick={() => setRegisterStep((s) => s - 1)}
              className="rounded-2xl px-5 py-3.5 font-extrabold text-[15px] bg-white border-[1.5px] border-line active:scale-[0.98]">
              <ArrowLeft size={18} />
            </button>
          )}
          <div className="flex-1">
            <Primary
              disabled={busy || (registerStep === 0 && (!name || !email || password.length < 10)) || (registerStep === 1 && !businessName)}
              icon={busy ? null : registerStep === 2 ? Check : ArrowRight}
              onClick={() => (registerStep === 2 ? submitRegister() : setRegisterStep((s) => s + 1))}
            >
              {busy ? 'Submitting…' : registerStep === 2 ? 'Submit application' : 'Continue'}
            </Primary>
          </div>
        </div>

        <p className="text-center text-[13.5px] font-bold mt-5 text-muted">
          Already a partner?{' '}
          <button type="button" onClick={() => { setMode('login'); setError(null); }} className="font-extrabold text-accent">Sign in</button>
        </p>
      </AuthShell>
    );
  }

  // ── login ──
  return (
    <AuthShell>
      <form onSubmit={submitLogin}>
        <h1 className="font-display font-extrabold leading-tight text-[28px]">Welcome back</h1>
        <p className="text-[14px] text-muted font-semibold mt-1.5 mb-6">Sign in to manage your offers and redemptions.</p>
        <div className="space-y-3.5">
          <Field label="Email" type="email" value={email} onChange={setEmail} icon={Mail} placeholder="you@yourbusiness.lk" autoFocus />
          <Field label="Password" type={showPassword ? 'text' : 'password'} value={password} onChange={setPassword} icon={Lock}
            reveal={showPassword} onReveal={() => setShowPassword((s) => !s)} placeholder="••••••••" />
          <div className="flex items-center justify-between pt-0.5">
            <button type="button" onClick={() => setRemember((r) => !r)} className="flex items-center gap-2">
              <span className={`rounded-md w-5 h-5 flex items-center justify-center border-[1.5px] ${remember ? 'bg-accent border-accent' : 'bg-white border-line'}`}>
                {remember && <Check size={13} strokeWidth={3.5} className="text-white" />}
              </span>
              <span className="text-[13px] font-bold text-ink-soft whitespace-nowrap">Remember me</span>
            </button>
            <button type="button" onClick={() => { setForgotSent(false); setMode('forgot'); }} className="text-[13px] font-extrabold text-accent">Forgot?</button>
          </div>
          {error && <p className="text-[13.5px] font-bold text-coral">{error}</p>}
          <div className="pt-1">
            <Primary type="submit" disabled={busy}>
              {busy ? 'Please wait…' : 'Sign in'}
            </Primary>
          </div>
          <div className="flex items-center gap-3 my-1">
            <div className="flex-1 h-px bg-line" />
            <span className="text-[12px] font-bold text-muted">or</span>
            <div className="flex-1 h-px bg-line" />
          </div>
          <a href={`${API_URL}/auth/google/login`}
            className="w-full flex items-center justify-center gap-2 rounded-2xl py-3 font-extrabold text-[14px] bg-white border-[1.5px] border-line text-ink transition-transform active:scale-[0.97]">
            <GoogleG /> Continue with Google
          </a>
        </div>
        <div className="mt-6 rounded-2xl bg-accent-tint p-4 flex items-center gap-3">
          <Store size={20} className="text-accent shrink-0" />
          <div className="flex-1">
            <p className="text-[13.5px] font-extrabold">New to Spotto?</p>
            <button type="button" onClick={() => { setRegisterStep(0); setMode('register'); setError(null); }} className="text-[13px] font-extrabold text-accent whitespace-nowrap">
              List your business →
            </button>
          </div>
        </div>
      </form>
    </AuthShell>
  );
}

function AuthShell({ children, wide }: { children: React.ReactNode; wide?: boolean }) {
  const features = [
    { icon: Users, t: 'Reach travelers near you', s: 'Get discovered by nomads, surfers & explorers in the bay.' },
    { icon: Tag, t: 'Run offers in seconds', s: 'Launch perks, discounts and bundles — live instantly.' },
    { icon: QrCode, t: 'Validate with a scan', s: 'Redeem codes at the counter, track every claim.' },
  ];
  return (
    <div className="min-h-screen flex bg-sand">
      <div className="hidden md:flex flex-col justify-between relative overflow-hidden shrink-0 p-11"
        style={{ flex: '0 0 46%', background: 'linear-gradient(160deg, var(--accent) 0%, var(--accent-2) 100%)' }}>
        <span className="absolute rounded-full bg-white/10" style={{ right: -60, top: -40, width: 280, height: 280 }} />
        <span className="absolute rounded-full bg-white/[0.08]" style={{ left: -50, bottom: -60, width: 220, height: 220 }} />
        <div className="relative flex items-center gap-2.5">
          <SpottoMark size={30} light />
          <div className="leading-none">
            <p className="font-display font-extrabold text-[20px] tracking-tight text-white">Spotto</p>
            <p className="text-[10px] font-extrabold uppercase tracking-[0.2em] mt-1 text-white/85">Vendor Portal</p>
          </div>
        </div>
        <div className="relative">
          <h2 className="font-display font-extrabold text-white leading-[1.1] text-[32px]">Turn passers-by<br />into regulars.</h2>
          <div className="mt-7 space-y-4">
            {features.map((f) => (
              <div key={f.t} className="flex items-start gap-3.5">
                <span className="w-10 h-10 rounded-xl bg-white/[0.18] text-white flex items-center justify-center shrink-0"><f.icon size={19} /></span>
                <div>
                  <p className="font-extrabold text-[15px] text-white">{f.t}</p>
                  <p className="text-[13px] font-semibold leading-snug text-white/80">{f.s}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
        <div className="relative rounded-2xl px-4 py-3.5 flex items-center gap-3 bg-white/15">
          <TrendingUp size={20} className="text-white shrink-0" />
          <p className="text-[13px] font-bold text-white leading-snug">Partners see <span className="font-extrabold">+18% redemptions</span> in their first month.</p>
        </div>
      </div>

      <div className="flex-1 flex items-center justify-center px-7 py-10 overflow-y-auto">
        <div className="w-full" style={{ maxWidth: wide ? 400 : 380 }}>{children}</div>
      </div>
    </div>
  );
}
