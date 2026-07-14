'use client';

// Account — business identity, plan, owner and sign-out.
// Port of AccountProfile.jsx (billing actions are demo stubs).

import { BadgeCheck, LogOut, Mail, MapPin, Store, Wallet } from 'lucide-react';
import { useRouter } from 'next/navigation';

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
