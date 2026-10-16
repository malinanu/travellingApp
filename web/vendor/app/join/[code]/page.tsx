'use client';

// Accept a team invite — the counterpart to Account → Plan & team → Invite
// a team member. There's no email system in this app, so an invite is a
// copy-link code (this route's [code] segment), not a sent email. Joining
// creates a new staff login on the inviting vendor's account (see
// POST /auth/register's inviteCode handling).

import { useParams, useRouter } from 'next/navigation';
import { useState } from 'react';

import type { AuthResponse } from '@spotto/shared';
import { api } from '../../../lib/api';

export default function JoinPage() {
  const router = useRouter();
  const params = useParams<{ code: string }>();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const res = await api.post<AuthResponse>('/auth/register', {
        email, password, name, role: 'vendor', inviteCode: params.code,
      });
      api.setToken(res.token, true, res.refreshToken);
      router.replace('/dashboard');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-8">
      <form onSubmit={submit} className="w-full max-w-md">
        <h2 className="text-2xl font-extrabold">Join your team on Spotto</h2>
        <p className="text-muted font-semibold mt-1 mb-6">
          Create your staff login to start scanning codes and managing offers.
        </p>

        <label className="block text-[13px] font-extrabold mb-1.5">Your name</label>
        <input className="input mb-4" value={name} onChange={(e) => setName(e.target.value)} required />
        <label className="block text-[13px] font-extrabold mb-1.5">Email</label>
        <input className="input mb-4" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
        <label className="block text-[13px] font-extrabold mb-1.5">Password</label>
        <input className="input mb-4" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required minLength={10} />

        {error && <p className="text-[13.5px] font-bold text-coral mb-3">{error}</p>}

        <button className="btn-accent w-full py-3" disabled={busy}>
          {busy ? 'Please wait…' : 'Join team'}
        </button>
      </form>
    </div>
  );
}
