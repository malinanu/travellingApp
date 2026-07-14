'use client';

// Admin profile — identity, real 2FA setup/disable, sessions stub, sign out.
// Port of AdminProfile.jsx.

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { KeyRound, LogOut, Mail, Shield, ShieldCheck, User as UserIcon } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { QRCodeSVG } from 'qrcode.react';
import { useState } from 'react';

import type { TwoFactorSetupResponse } from '@spotto/shared';
import Shell, { useMe } from '../../components/shell';
import { api } from '../../lib/api';

export default function AdminProfilePage() {
  const router = useRouter();
  const { data: me } = useMe();

  return (
    <Shell>
      <div className="max-w-xl space-y-4">
        <div className="panel p-6 flex items-center gap-4">
          <span className="w-14 h-14 rounded-full bg-accent text-white flex items-center justify-center text-xl font-bold">
            {(me?.name ?? 'A').slice(0, 1)}
          </span>
          <div>
            <h2 className="font-bold text-lg">{me?.name ?? 'Admin'}</h2>
            <p className="text-muted font-semibold text-[13px] flex items-center gap-1.5">
              <Mail size={13} /> {me?.email}
            </p>
          </div>
        </div>

        <TwoFactorSection enabled={me?.totpEnabled ?? false} />

        <div className="panel p-6">
          <h3 className="font-bold text-[15px] mb-3 flex items-center gap-2"><Shield size={16} /> Security</h3>
          <button className="btn">Change password</button>
        </div>

        <div className="panel p-6">
          <h3 className="font-bold text-[15px] mb-3 flex items-center gap-2"><UserIcon size={16} /> Role</h3>
          <p className="text-[13.5px] text-muted font-medium">Super-admin — full platform access.</p>
        </div>

        <button
          className="btn btn-danger px-5 py-2.5 flex items-center gap-2"
          onClick={() => {
            api.setToken(null);
            router.replace('/login');
          }}
        >
          <LogOut size={16} /> Sign out
        </button>
      </div>
    </Shell>
  );
}

function TwoFactorSection({ enabled }: { enabled: boolean }) {
  const qc = useQueryClient();
  const [setup, setSetup] = useState<TwoFactorSetupResponse | null>(null);
  const [code, setCode] = useState('');
  const [disableCode, setDisableCode] = useState('');
  const [error, setError] = useState<string | null>(null);

  const startSetup = useMutation({
    mutationFn: () => api.post<TwoFactorSetupResponse>('/auth/2fa/setup'),
    onSuccess: setSetup,
  });
  const confirmEnable = useMutation({
    mutationFn: () => api.post('/auth/2fa/enable', { code }),
    onSuccess: () => {
      setSetup(null);
      setCode('');
      qc.invalidateQueries({ queryKey: ['admin-me'] });
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Invalid code'),
  });
  const disable = useMutation({
    mutationFn: () => api.post('/auth/2fa/disable', { code: disableCode }),
    onSuccess: () => {
      setDisableCode('');
      qc.invalidateQueries({ queryKey: ['admin-me'] });
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Invalid code'),
  });

  return (
    <div className="panel p-6">
      <h3 className="font-bold text-[15px] mb-1 flex items-center gap-2">
        <ShieldCheck size={16} className={enabled ? 'text-accent' : 'text-muted'} />
        Two-factor authentication
      </h3>
      <p className="text-[13.5px] text-muted font-medium mb-4">
        {enabled
          ? 'Enabled — a code from your authenticator app is required at every sign-in.'
          : 'Add an authenticator app (Google Authenticator, Authy, 1Password) for a second sign-in step.'}
      </p>

      {enabled ? (
        <>
          <input className="input mb-3" placeholder="Current 6-digit code to disable" value={disableCode}
            onChange={(e) => { setDisableCode(e.target.value); setError(null); }} maxLength={6} inputMode="numeric" />
          {error && <p className="text-[13px] font-bold mb-3" style={{ color: '#e11d48' }}>{error}</p>}
          <button className="btn btn-danger" disabled={disableCode.length !== 6 || disable.isPending}
            onClick={() => disable.mutate()}>
            {disable.isPending ? 'Disabling…' : 'Disable 2FA'}
          </button>
        </>
      ) : setup ? (
        <div>
          <div className="flex items-start gap-4 mb-4">
            <div className="bg-white p-2 rounded-lg border" style={{ borderColor: 'var(--line)' }}>
              <QRCodeSVG value={setup.otpauthUrl} size={120} />
            </div>
            <div className="text-[12.5px] text-muted font-medium">
              <p className="mb-2">Scan with your authenticator app, or enter this key manually:</p>
              <code className="block rounded-lg px-2 py-1.5 font-mono text-[11.5px] break-all" style={{ background: 'var(--bg)' }}>
                {setup.secret}
              </code>
            </div>
          </div>
          <input className="input mb-3" placeholder="6-digit code from the app" value={code}
            onChange={(e) => { setCode(e.target.value); setError(null); }} maxLength={6} inputMode="numeric" />
          {error && <p className="text-[13px] font-bold mb-3" style={{ color: '#e11d48' }}>{error}</p>}
          <div className="flex gap-2">
            <button className="btn btn-primary flex items-center gap-2" disabled={code.length !== 6 || confirmEnable.isPending}
              onClick={() => confirmEnable.mutate()}>
              <KeyRound size={15} /> {confirmEnable.isPending ? 'Verifying…' : 'Confirm & enable'}
            </button>
            <button className="btn" onClick={() => setSetup(null)}>Cancel</button>
          </div>
        </div>
      ) : (
        <button className="btn flex items-center gap-2" disabled={startSetup.isPending} onClick={() => startSetup.mutate()}>
          <ShieldCheck size={16} /> {startSetup.isPending ? 'Generating…' : 'Set up 2FA'}
        </button>
      )}
    </div>
  );
}
