'use client';

// Admin auth — split layout, login then a real TOTP verification step
// (only shown when the account actually has 2FA enabled), plus a
// forgot/reset-password flow (same global /auth/forgot-password and
// /auth/reset-password routes the vendor portal's login page uses — see
// web/vendor/app/login/page.tsx for the reference implementation). Port of
// AdminAuth.jsx, now backed by /auth/2fa/login-verify instead of accepting
// any 6 digits.

import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useRef, useState } from 'react';

import { isTwoFactorRequired, type AuthResponse, type TwoFactorRequiredResponse } from '@spotto/shared';
import { SpottoMark } from '../../components/shell';
import { api } from '../../lib/api';

export default function AdminLoginPage() {
  return (
    <Suspense fallback={null}>
      <AdminLoginPageInner />
    </Suspense>
  );
}

function AdminLoginPageInner() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [mode, setMode] = useState<'creds' | 'otp' | 'forgot'>('creds');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [otp, setOtp] = useState(['', '', '', '', '', '']);
  const [twoFactorToken, setTwoFactorToken] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const otpRefs = useRef<(HTMLInputElement | null)[]>([]);

  const [forgotEmail, setForgotEmail] = useState('');
  const [forgotSent, setForgotSent] = useState(false);
  const [resetToken, setResetToken] = useState<string | null>(null);
  const [newPassword, setNewPassword] = useState('');
  const [resetDone, setResetDone] = useState(false);

  // A password-reset email always links back here with ?resetToken=... —
  // jump straight into the "set a new password" view.
  useEffect(() => {
    const rToken = searchParams.get('resetToken');
    if (rToken) {
      setResetToken(rToken);
      setMode('forgot');
      router.replace('/login');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const finishSession = (res: AuthResponse) => {
    if (res.user.role !== 'admin') {
      api.setToken(null);
      api.logout(res.refreshToken);
      setError('This account is not an admin.');
      return;
    }
    api.setToken(res.token, true, res.refreshToken);
    router.replace('/queue');
  };

  const submitCreds = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const res = await api.post<AuthResponse | TwoFactorRequiredResponse>('/auth/login', { email, password });
      if (isTwoFactorRequired(res)) {
        setTwoFactorToken(res.twoFactorToken);
        setMode('otp');
        setTimeout(() => otpRefs.current[0]?.focus(), 50);
        return;
      }
      finishSession(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Sign-in failed');
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
      const res = await api.post<AuthResponse>('/auth/2fa/login-verify', {
        twoFactorToken,
        code: otp.join(''),
      });
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

  return (
    <div className="min-h-screen grid lg:grid-cols-2" style={{ background: 'var(--bg)' }}>
      <div className="hidden lg:flex flex-col justify-between p-12 text-white" style={{ background: '#103d3a' }}>
        <div className="flex items-center gap-3">
          <SpottoMark size={30} />
          <span className="font-bold text-lg">Spotto Command Center</span>
        </div>
        <div>
          <h1 className="text-4xl font-bold leading-tight max-w-md">
            Run the whole platform from one sunny cockpit.
          </h1>
          <p className="mt-4 text-white/70 font-medium max-w-md">
            Approve vendors, moderate content, watch geo-demand, manage subscriptions.
          </p>
        </div>
        <p className="text-white/50 text-sm font-medium">Demo login: admin@spotto.dev · spotto123</p>
      </div>

      <div className="flex items-center justify-center p-8">
        {mode === 'creds' && (
          <form onSubmit={submitCreds} className="w-full max-w-md">
            <h2 className="text-2xl font-bold" style={{ color: 'var(--ink)' }}>Admin sign in</h2>
            <p className="text-muted font-medium mt-1 mb-6">Restricted to Spotto staff accounts.</p>
            <label className="block text-[13px] font-bold mb-1.5">Email</label>
            <input className="input mb-4" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required suppressHydrationWarning />
            <label className="block text-[13px] font-bold mb-1.5">Password</label>
            <input className="input mb-2" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={6} suppressHydrationWarning />
            <button type="button" onClick={() => { setError(null); setForgotSent(false); setMode('forgot'); }}
              className="block text-[12.5px] font-bold mb-4" style={{ color: 'var(--accent)' }}>
              Forgot password?
            </button>
            {error && <p className="text-[13.5px] font-bold mb-3" style={{ color: '#e11d48' }}>{error}</p>}
            <button className="btn btn-primary w-full py-3" disabled={busy}>{busy ? 'Please wait…' : 'Continue'}</button>
          </form>
        )}

        {mode === 'otp' && (
          <div className="w-full max-w-md">
            <h2 className="text-2xl font-bold" style={{ color: 'var(--ink)' }}>Two-factor verification</h2>
            <p className="text-muted font-medium mt-1 mb-6">Enter the 6-digit code from your authenticator app.</p>
            <div className="flex gap-2 mb-4">
              {otp.map((d, i) => (
                <input
                  key={i}
                  ref={(el) => { otpRefs.current[i] = el; }}
                  value={d}
                  onChange={(e) => onOtpChange(i, e.target.value)}
                  inputMode="numeric"
                  maxLength={1}
                  className="input text-center text-xl font-bold w-12 h-14"
                />
              ))}
            </div>
            {error && <p className="text-[13.5px] font-bold mb-3" style={{ color: '#e11d48' }}>{error}</p>}
            <button className="btn btn-primary w-full py-3" disabled={otp.join('').length !== 6 || busy} onClick={submitOtp}>
              {busy ? 'Verifying…' : 'Verify & continue'}
            </button>
          </div>
        )}

        {mode === 'forgot' && !resetToken && (
          <div className="w-full max-w-md">
            <h2 className="text-2xl font-bold" style={{ color: 'var(--ink)' }}>Reset password</h2>
            <p className="text-muted font-medium mt-1 mb-6">We&rsquo;ll email you a secure reset link.</p>
            {forgotSent ? (
              <p className="text-[13.5px] font-bold mb-5" style={{ color: 'var(--accent)' }}>
                Check your inbox — the reset link is on its way.
              </p>
            ) : (
              <>
                <label className="block text-[13px] font-bold mb-1.5">Email</label>
                <input className="input mb-4" type="email" value={forgotEmail} onChange={(e) => setForgotEmail(e.target.value)} required autoFocus />
              </>
            )}
            {error && <p className="text-[13.5px] font-bold mb-3" style={{ color: '#e11d48' }}>{error}</p>}
            {!forgotSent && (
              <button className="btn btn-primary w-full py-3 mb-3" disabled={!forgotEmail || busy} onClick={submitForgot}>
                {busy ? 'Sending…' : 'Send reset link'}
              </button>
            )}
            <button type="button" onClick={() => { setForgotSent(false); setMode('creds'); }} className="w-full text-center text-[13px] font-bold text-muted">
              ← Back to sign in
            </button>
          </div>
        )}

        {mode === 'forgot' && resetToken && (
          <div className="w-full max-w-md">
            <h2 className="text-2xl font-bold" style={{ color: 'var(--ink)' }}>Set a new password</h2>
            <p className="text-muted font-medium mt-1 mb-6">Choose a new password for your account.</p>
            {resetDone ? (
              <p className="text-[13.5px] font-bold mb-5" style={{ color: 'var(--accent)' }}>
                Password updated — sign in with your new password.
              </p>
            ) : (
              <>
                <label className="block text-[13px] font-bold mb-1.5">New password</label>
                <input className="input mb-4" type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)}
                  placeholder="Min 10 characters" autoFocus />
              </>
            )}
            {error && <p className="text-[13.5px] font-bold mb-3" style={{ color: '#e11d48' }}>{error}</p>}
            {resetDone ? (
              <button className="btn btn-primary w-full py-3" onClick={() => { setMode('creds'); setResetToken(null); setResetDone(false); }}>
                Back to sign in
              </button>
            ) : (
              <button className="btn btn-primary w-full py-3" disabled={newPassword.length < 10 || busy} onClick={submitReset}>
                {busy ? 'Saving…' : 'Save new password'}
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
