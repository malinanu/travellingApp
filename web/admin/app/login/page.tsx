'use client';

// Admin auth — split layout, login then a real TOTP verification step
// (only shown when the account actually has 2FA enabled). Port of
// AdminAuth.jsx, now backed by /auth/2fa/login-verify instead of accepting
// any 6 digits.

import { useRouter } from 'next/navigation';
import { useRef, useState } from 'react';

import { isTwoFactorRequired, type AuthResponse, type TwoFactorRequiredResponse } from '@spotto/shared';
import { api } from '../../lib/api';

export default function AdminLoginPage() {
  const router = useRouter();
  const [step, setStep] = useState<'creds' | 'otp'>('creds');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [otp, setOtp] = useState(['', '', '', '', '', '']);
  const [twoFactorToken, setTwoFactorToken] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const otpRefs = useRef<(HTMLInputElement | null)[]>([]);

  const finishSession = (res: AuthResponse) => {
    if (res.user.role !== 'admin') {
      api.setToken(null);
      setError('This account is not an admin.');
      return;
    }
    api.setToken(res.token);
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
        setStep('otp');
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

  return (
    <div className="min-h-screen grid lg:grid-cols-2" style={{ background: 'var(--bg)' }}>
      <div className="hidden lg:flex flex-col justify-between p-12 text-white" style={{ background: '#103d3a' }}>
        <div className="flex items-center gap-3">
          <svg width="30" height="30" viewBox="0 0 32 32" fill="none">
            <path d="M16 3C10.5 3 6 7.3 6 12.7 6 19.5 16 29 16 29s10-9.5 10-16.3C26 7.3 21.5 3 16 3z" fill="#0b6f86" />
            <circle cx="16" cy="12.5" r="4.4" fill="#f6b81e" />
          </svg>
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
        {step === 'creds' ? (
          <form onSubmit={submitCreds} className="w-full max-w-md">
            <h2 className="text-2xl font-bold" style={{ color: 'var(--ink)' }}>Admin sign in</h2>
            <p className="text-muted font-medium mt-1 mb-6">Restricted to Spotto staff accounts.</p>
            <label className="block text-[13px] font-bold mb-1.5">Email</label>
            <input className="input mb-4" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
            <label className="block text-[13px] font-bold mb-1.5">Password</label>
            <input className="input mb-4" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={6} />
            {error && <p className="text-[13.5px] font-bold mb-3" style={{ color: '#e11d48' }}>{error}</p>}
            <button className="btn btn-primary w-full py-3" disabled={busy}>{busy ? 'Please wait…' : 'Continue'}</button>
          </form>
        ) : (
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
      </div>
    </div>
  );
}
