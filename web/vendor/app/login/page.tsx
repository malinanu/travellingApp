'use client';

// Vendor auth — login or register a business (registration creates a
// pending vendor that an admin approves from the command center).

import { useRouter } from 'next/navigation';
import { useState } from 'react';
import { Coffee, Martini, Palette, Store, Utensils, Waves, LucideIcon } from 'lucide-react';

import type { AuthResponse } from '@spotto/shared';
import { api } from '../../lib/api';

const BUSINESS_TYPES: { id: string; label: string; icon: LucideIcon }[] = [
  { id: 'restaurant', label: 'Restaurant', icon: Utensils },
  { id: 'cafe', label: 'Café', icon: Coffee },
  { id: 'bar', label: 'Bar & Sunset', icon: Martini },
  { id: 'surf', label: 'Surf & Water', icon: Waves },
  { id: 'craft', label: 'Craft & Studio', icon: Palette },
  { id: 'shop', label: 'Shop', icon: Store },
];

export default function LoginPage() {
  const router = useRouter();
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [name, setName] = useState('');
  const [businessName, setBusinessName] = useState('');
  const [category, setCategory] = useState('cafe');
  const [town, setTown] = useState('Weligama');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const res =
        mode === 'login'
          ? await api.post<AuthResponse>('/auth/login', { email, password })
          : await api.post<AuthResponse>('/auth/register', {
              email, password, name, role: 'vendor', businessName, category, town,
            });
      if (res.user.role !== 'vendor') {
        setError('This portal is for business accounts. Use the Spotto app instead.');
        return;
      }
      api.setToken(res.token);
      router.replace('/dashboard');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="min-h-screen grid lg:grid-cols-2">
      <div className="hidden lg:flex flex-col justify-between p-12 text-white"
        style={{ background: 'linear-gradient(160deg, #0e8c8a 0%, #0b6f86 55%, #0b4a54 100%)' }}>
        <div className="flex items-center gap-3">
          <span className="w-10 h-10 rounded-xl bg-white/15 flex items-center justify-center font-extrabold text-lg">S</span>
          <span className="font-extrabold text-lg">Spotto for Business</span>
        </div>
        <div>
          <h1 className="text-4xl font-extrabold leading-tight max-w-md">
            Put your business on every traveller&rsquo;s map of Weligama.
          </h1>
          <p className="mt-4 text-white/80 font-semibold max-w-md">
            Offers, live redemptions, growth tools and reports — one sunny dashboard.
          </p>
        </div>
        <p className="text-white/60 text-sm font-semibold">Demo login: vendor@driftwood.dev · spotto123</p>
      </div>

      <div className="flex items-center justify-center p-8">
        <form onSubmit={submit} className="w-full max-w-md">
          <h2 className="text-2xl font-extrabold">
            {mode === 'login' ? 'Welcome back' : 'Register your business'}
          </h2>
          <p className="text-muted font-semibold mt-1 mb-6">
            {mode === 'login'
              ? 'Sign in to your vendor dashboard.'
              : 'Submit your business for review — approval usually takes a day.'}
          </p>

          {mode === 'register' && (
            <>
              <label className="block text-[13px] font-extrabold mb-1.5">Your name</label>
              <input className="input mb-4" value={name} onChange={(e) => setName(e.target.value)} required />
              <label className="block text-[13px] font-extrabold mb-1.5">Business name</label>
              <input className="input mb-4" value={businessName} onChange={(e) => setBusinessName(e.target.value)} required />
              <label className="block text-[13px] font-extrabold mb-1.5">Category</label>
              <div className="grid grid-cols-3 gap-2 mb-4">
                {BUSINESS_TYPES.map((t) => {
                  const Icon = t.icon;
                  const on = category === t.id;
                  return (
                    <button type="button" key={t.id} onClick={() => setCategory(t.id)}
                      className={`rounded-xl border px-2 py-2.5 flex flex-col items-center gap-1 text-[12px] font-extrabold transition-colors ${
                        on ? 'border-accent bg-accent-tint text-accent' : 'border-line bg-white hover:border-accent'
                      }`}>
                      <Icon size={17} />
                      {t.label}
                    </button>
                  );
                })}
              </div>
              <label className="block text-[13px] font-extrabold mb-1.5">Town</label>
              <input className="input mb-4" value={town} onChange={(e) => setTown(e.target.value)} required />
            </>
          )}

          <label className="block text-[13px] font-extrabold mb-1.5">Email</label>
          <input className="input mb-4" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
          <label className="block text-[13px] font-extrabold mb-1.5">Password</label>
          <input className="input mb-4" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={6} />

          {error && <p className="text-[13.5px] font-bold text-coral mb-3">{error}</p>}

          <button className="btn-accent w-full py-3" disabled={busy}>
            {busy ? 'Please wait…' : mode === 'login' ? 'Sign in' : 'Submit for review'}
          </button>

          <button
            type="button"
            className="w-full text-center mt-4 text-accent font-extrabold text-[13.5px] hover:underline"
            onClick={() => { setMode(mode === 'login' ? 'register' : 'login'); setError(null); }}
          >
            {mode === 'login' ? 'New here? Register your business' : 'Have an account? Sign in'}
          </button>
        </form>
      </div>
    </div>
  );
}
