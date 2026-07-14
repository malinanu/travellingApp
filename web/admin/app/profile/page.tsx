'use client';

// Admin profile — identity, security stub, sessions stub, sign out.
// Port of AdminProfile.jsx.

import { LogOut, Mail, Shield, User as UserIcon } from 'lucide-react';
import { useRouter } from 'next/navigation';

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

        <div className="panel p-6">
          <h3 className="font-bold text-[15px] mb-3 flex items-center gap-2"><Shield size={16} /> Security</h3>
          <p className="text-[13.5px] text-muted font-medium mb-3">Two-factor authentication is required at every sign-in.</p>
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
