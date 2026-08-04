'use client';

import { useState } from 'react';
import { Key, Save, Shield, Sliders, Globe } from 'lucide-react';
import Shell from '../../components/shell';

export default function AdminSettingsPage() {
  const [saved, setSaved] = useState(false);
  const [platformFee, setPlatformFee] = useState('10');
  const [defaultRegion, setDefaultRegion] = useState('Weligama Bay');
  const [requireBrCert, setRequireBrCert] = useState(true);

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault();
    setSaved(true);
    setTimeout(() => setSaved(false), 3000);
  };

  return (
    <Shell>
      <div className="max-w-3xl">
        <h1 className="font-display font-extrabold text-[28px] text-ink mb-1">System Settings</h1>
        <p className="text-[14px] text-muted font-semibold mb-6">
          Global platform configuration, region defaults, and security policies.
        </p>

        {saved && (
          <div className="rounded-xl p-4 mb-6 bg-accent-tint text-accent font-extrabold text-[14px]">
            ✓ System settings saved successfully.
          </div>
        )}

        <form onSubmit={handleSave} className="space-y-6">
          {/* Platform Settings */}
          <div className="card p-6">
            <div className="flex items-center gap-2.5 mb-4">
              <span className="w-9 h-9 rounded-xl bg-accent-tint text-accent flex items-center justify-center">
                <Sliders size={18} />
              </span>
              <div>
                <h2 className="font-extrabold text-[16px]">Platform Parameters</h2>
                <p className="text-[12.5px] text-muted font-semibold">General operational settings</p>
              </div>
            </div>

            <div className="space-y-4">
              <div>
                <label className="block text-[13px] font-extrabold mb-1.5">Platform Fee (%)</label>
                <input
                  type="number"
                  className="input w-48"
                  value={platformFee}
                  onChange={(e) => setPlatformFee(e.target.value)}
                />
              </div>

              <div>
                <label className="block text-[13px] font-extrabold mb-1.5">Default Region</label>
                <input
                  type="text"
                  className="input w-72"
                  value={defaultRegion}
                  onChange={(e) => setDefaultRegion(e.target.value)}
                />
              </div>
            </div>
          </div>

          {/* Security & Verification */}
          <div className="card p-6">
            <div className="flex items-center gap-2.5 mb-4">
              <span className="w-9 h-9 rounded-xl bg-accent-tint text-accent flex items-center justify-center">
                <Shield size={18} />
              </span>
              <div>
                <h2 className="font-extrabold text-[16px]">Security & Verification</h2>
                <p className="text-[12.5px] text-muted font-semibold">Vendor verification rules</p>
              </div>
            </div>

            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={requireBrCert}
                onChange={(e) => setRequireBrCert(e.target.checked)}
                className="w-4 h-4 accent-accent"
              />
              <span className="text-[14px] font-extrabold">Require BR Certificate verification for vendor payout</span>
            </label>
          </div>

          <div className="flex justify-end">
            <button type="submit" className="btn-accent rounded-xl px-6 py-3 font-extrabold text-[14px] flex items-center gap-2">
              <Save size={16} /> Save Changes
            </button>
          </div>
        </form>
      </div>
    </Shell>
  );
}
