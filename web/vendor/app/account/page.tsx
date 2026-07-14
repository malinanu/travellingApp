'use client';

// Account — business identity, plan, owner, two-factor auth, sign-out.
// Port of AccountProfile.jsx (billing actions are demo stubs).

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { BadgeCheck, KeyRound, LogOut, Mail, MapPin, ShieldCheck, Store, Wallet } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { QRCodeSVG } from 'qrcode.react';
import { useState } from 'react';

import type { TwoFactorSetupResponse } from '@spotto/shared';
import Shell, { useBusiness, useMe } from '../../components/shell';
import { api } from '../../lib/api';

export default function AccountPage() {
  const router = useRouter();
  const { data: me } = useMe();
  const { data: business } = useBusiness();

  return (
    <Shell>
      <div className="max-w-2xl space-y-4">
        <div className="card p-6 flex items-center gap-4">
          <span className="w-16 h-16 rounded-2xl bg-accent text-white flex items-center justify-center text-xl font-extrabold">
            {(business?.name ?? 'S').split(/\s+/).slice(0, 2).map((w) => w[0]).join('').toUpperCase()}
          </span>
          <div className="flex-1">
            <h2 className="text-lg font-extrabold flex items-center gap-2">
              {business?.name ?? '…'}
              <BadgeCheck size={18} className="text-accent" />
            </h2>
            <p className="text-[13.5px] text-muted font-semibold flex items-center gap-3 mt-0.5">
              <span className="flex items-center gap-1"><Store size={13} /> {business?.category}</span>
              <span className="flex items-center gap-1"><MapPin size={13} /> {business?.town}</span>
            </p>
          </div>
          <span className="rounded-full bg-sun text-ink text-[12px] font-extrabold px-3 py-1">
            {business?.tier ?? 'Free'}
          </span>
        </div>

        <div className="card p-6">
          <h3 className="font-extrabold text-[15px] mb-4">Owner</h3>
          <div className="space-y-2.5 text-[14px] font-semibold">
            <p className="flex items-center gap-2.5"><Store size={15} className="text-muted" /> {business?.owner}</p>
            <p className="flex items-center gap-2.5"><Mail size={15} className="text-muted" /> {me?.email}</p>
          </div>
        </div>

        <TwoFactorSection enabled={me?.totpEnabled ?? false} />

        <div className="card p-6">
          <h3 className="font-extrabold text-[15px] mb-2">Plan & billing</h3>
          <p className="text-[13.5px] text-muted font-semibold mb-4">
            {business?.tier === 'Premium'
              ? 'Premium — $49/mo. Priority placement, growth tools, boosted events.'
              : 'Free plan. Upgrade for priority placement and growth tools.'}
          </p>
          <button className="rounded-xl border border-line bg-white px-4 py-2.5 font-extrabold text-[13.5px] flex items-center gap-2 hover:border-accent">
            <Wallet size={16} /> Manage billing
          </button>
        </div>

        <button
          className="rounded-xl border border-coral/40 bg-[#fde4d8] text-coral px-5 py-3 font-extrabold text-[14px] flex items-center gap-2"
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
      qc.invalidateQueries({ queryKey: ['me'] });
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Invalid code'),
  });
  const disable = useMutation({
    mutationFn: () => api.post('/auth/2fa/disable', { code: disableCode }),
    onSuccess: () => {
      setDisableCode('');
      qc.invalidateQueries({ queryKey: ['me'] });
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Invalid code'),
  });

  return (
    <div className="card p-6">
      <h3 className="font-extrabold text-[15px] mb-1 flex items-center gap-2">
        <ShieldCheck size={17} className={enabled ? 'text-accent' : 'text-muted'} />
        Two-factor authentication
      </h3>
      <p className="text-[13.5px] text-muted font-semibold mb-4">
        {enabled
          ? 'Enabled — a code from your authenticator app is required at every sign-in.'
          : 'Add an authenticator app (Google Authenticator, Authy, 1Password) for a second sign-in step.'}
      </p>

      {enabled ? (
        <>
          <input className="input mb-3" placeholder="Current 6-digit code to disable" value={disableCode}
            onChange={(e) => { setDisableCode(e.target.value); setError(null); }} maxLength={6} inputMode="numeric" />
          {error && <p className="text-[13px] font-bold text-coral mb-3">{error}</p>}
          <button className="rounded-xl border border-coral/40 bg-[#fde4d8] text-coral px-4 py-2.5 font-extrabold text-[13.5px]"
            disabled={disableCode.length !== 6 || disable.isPending}
            onClick={() => disable.mutate()}>
            {disable.isPending ? 'Disabling…' : 'Disable 2FA'}
          </button>
        </>
      ) : setup ? (
        <div>
          <div className="flex items-start gap-4 mb-4">
            <div className="bg-white p-2 rounded-lg border border-line">
              <QRCodeSVG value={setup.otpauthUrl} size={120} />
            </div>
            <div className="text-[12.5px] text-muted font-semibold">
              <p className="mb-2">Scan with your authenticator app, or enter this key manually:</p>
              <code className="block bg-sand rounded-lg px-2 py-1.5 font-mono text-[11.5px] break-all">{setup.secret}</code>
            </div>
          </div>
          <input className="input mb-3" placeholder="6-digit code from the app" value={code}
            onChange={(e) => { setCode(e.target.value); setError(null); }} maxLength={6} inputMode="numeric" />
          {error && <p className="text-[13px] font-bold text-coral mb-3">{error}</p>}
          <div className="flex gap-2">
            <button className="btn-accent px-4 py-2.5 flex items-center gap-2 text-[13.5px]"
              disabled={code.length !== 6 || confirmEnable.isPending}
              onClick={() => confirmEnable.mutate()}>
              <KeyRound size={15} /> {confirmEnable.isPending ? 'Verifying…' : 'Confirm & enable'}
            </button>
            <button className="rounded-xl border border-line bg-white px-4 py-2.5 font-extrabold text-[13.5px]"
              onClick={() => setSetup(null)}>
              Cancel
            </button>
          </div>
        </div>
      ) : (
        <button className="rounded-xl border border-line bg-white px-4 py-2.5 font-extrabold text-[13.5px] flex items-center gap-2 hover:border-accent"
          disabled={startSetup.isPending}
          onClick={() => startSetup.mutate()}>
          <ShieldCheck size={16} /> {startSetup.isPending ? 'Generating…' : 'Set up 2FA'}
        </button>
      )}
    </div>
  );
}
