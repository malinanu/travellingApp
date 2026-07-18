'use client';

// My account — personal login, security and alerts. Port of the vendor
// account mockup: editable personal details, password change, two-factor
// auth, notification preferences, plan & team (real invite-based team
// management, now emailed via the server's mailer with a staff/manager
// role picker, falling back to a copy-link if email isn't configured or
// bounces), sign out. Billing management stays a disabled "coming soon"
// stub — see SKIPPED.md for why.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  BadgeCheck, Bell, Camera, Check, Clock, Copy, Crown, KeyRound, Lightbulb, Lock,
  LogOut, Mail, MapPin, Phone, QrCode, Send, Settings2, ShieldCheck, Star, Store,
  Ticket, Trash2, User as UserIcon, UserPlus, Wallet, X, LucideIcon,
} from 'lucide-react';
import { useRouter } from 'next/navigation';
import { QRCodeSVG } from 'qrcode.react';
import { useEffect, useState } from 'react';

import type { TeamResponse, TwoFactorSetupResponse, User } from '@spotto/shared';
import Shell, { useBusiness, useMe } from '../../components/shell';
import { api } from '../../lib/api';

function initialsOf(name: string) {
  return name.split(/\s+/).slice(0, 2).map((w) => w[0]).join('').toUpperCase() || '?';
}

function timeAgo(iso?: string) {
  if (!iso) return 'a while ago';
  const days = Math.floor((Date.now() - new Date(iso).getTime()) / 86_400_000);
  if (days < 1) return 'today';
  if (days < 30) return `${days} day${days === 1 ? '' : 's'} ago`;
  const months = Math.floor(days / 30);
  return `${months} month${months === 1 ? '' : 's'} ago`;
}

function roleLabel(role?: string) {
  if (role === 'manager') return 'Manager';
  if (role === 'staff') return 'Staff';
  return 'Owner';
}

function CardHeader({ icon: Icon, title, sub }: { icon: LucideIcon; title: string; sub: string }) {
  return (
    <div className="flex items-center gap-2.5 mb-4">
      <span className="w-9 h-9 rounded-xl bg-accent-tint text-accent flex items-center justify-center shrink-0">
        <Icon size={18} />
      </span>
      <div>
        <h3 className="font-extrabold text-[15px]">{title}</h3>
        <p className="text-[12.5px] text-muted font-semibold">{sub}</p>
      </div>
    </div>
  );
}

export default function AccountPage() {
  const router = useRouter();
  const qc = useQueryClient();
  const { data: me } = useMe();
  const { data: business } = useBusiness();

  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [contactError, setContactError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    if (me) {
      setName(me.name);
      setEmail(me.email);
      setPhone(me.phone ?? '');
    }
  }, [me]);

  const saveContact = useMutation({
    mutationFn: () => api.put<User>('/me/contact', { name, email, phone }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['me'] });
      setContactError(null);
      setSaved(true);
      setTimeout(() => setSaved(false), 1800);
    },
    onError: (e) => setContactError(e instanceof Error ? e.message : 'Could not save changes'),
  });

  return (
    <Shell>
      <div>
        <div className="flex items-end justify-between gap-4 mb-5">
          <div>
            <h1 className="text-2xl font-extrabold">My account</h1>
            <p className="text-muted font-semibold mt-0.5">Your personal login, security and alerts.</p>
          </div>
          <button className="btn-accent flex items-center gap-2 px-4 py-2.5 shrink-0 whitespace-nowrap" disabled={saveContact.isPending}
            onClick={() => saveContact.mutate()}>
            {saveContact.isPending ? 'Saving…' : saved ? <><Check size={16} /> Saved</> : <><Send size={16} /> Save changes</>}
          </button>
        </div>

        {contactError && (
          <p className="text-[13.5px] font-bold text-coral mb-4">{contactError}</p>
        )}

        <div className="card p-5 flex items-center gap-4 mb-4">
          <div className="relative shrink-0">
            <div className="w-16 h-16 rounded-2xl flex items-center justify-center text-white text-[22px] font-extrabold"
              style={{ background: 'linear-gradient(140deg, var(--accent), var(--accent-2))' }}>
              {initialsOf(me?.name ?? '?')}
            </div>
            <button className="absolute -bottom-1 -right-1 w-7 h-7 rounded-lg bg-white border border-accent-tint text-accent flex items-center justify-center"
              style={{ boxShadow: '0 3px 10px rgba(15,61,58,0.2)' }} title="Photo upload coming soon">
              <Camera size={13} />
            </button>
          </div>
          <div className="flex-1 min-w-0">
            <h2 className="text-lg font-extrabold flex items-center gap-2">
              {me?.name ?? '…'}
              <span className="text-[11px] font-extrabold uppercase tracking-wide rounded-full px-2 py-0.5 bg-accent-tint text-accent shrink-0">
                {roleLabel(me?.vendorRole)}
              </span>
            </h2>
            <p className="text-[13.5px] text-muted font-semibold flex items-center gap-3 mt-0.5">
              <span className="flex items-center gap-1"><Store size={13} /> {business?.name}</span>
              <span className="flex items-center gap-1"><MapPin size={13} /> {business?.town}</span>
            </p>
          </div>
          {business?.status === 'approved' && (
            <span className="rounded-lg text-[11.5px] font-extrabold px-3 py-1.5 flex items-center gap-1.5 shrink-0" style={{ background: 'rgba(22,163,74,0.1)', color: '#16a34a' }}>
              <BadgeCheck size={14} /> Verified
            </span>
          )}
        </div>

        <div className="grid gap-4 mb-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(340px, 100%), 1fr))' }}>
          <div className="card p-5">
            <CardHeader icon={UserIcon} title="Personal details" sub="Your contact info" />
            <label className="block text-[12.5px] font-extrabold mb-1.5 text-muted">Full name</label>
            <div className="relative mb-3.5">
              <UserIcon size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
              <input className="input pl-9" value={name} onChange={(e) => setName(e.target.value)} />
            </div>

            <label className="block text-[12.5px] font-extrabold mb-1.5 text-muted">Email</label>
            <div className="relative mb-3.5">
              <Mail size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
              <input className="input pl-9" type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
            </div>

            <label className="block text-[12.5px] font-extrabold mb-1.5 text-muted">Phone</label>
            <div className="relative">
              <Phone size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
              <input className="input pl-9" value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="+94 …" />
            </div>
          </div>

          <SignInSecurityCard me={me} />
          <NotificationsCard me={me} />

          <div className="card p-5">
            <CardHeader icon={Crown} title="Plan & team" sub="Subscription and access" />

            <div className="rounded-xl bg-accent-tint p-3.5 flex items-center gap-3 mb-3">
              <span className="w-10 h-10 rounded-xl bg-accent text-white flex items-center justify-center shrink-0"><Crown size={19} /></span>
              <div className="flex-1 min-w-0">
                <p className="font-extrabold text-[14px]">{business?.tier === 'Premium' ? 'Premium plan' : 'Free plan'}</p>
                <p className="text-[12.5px] text-ink-soft font-semibold">
                  {business?.tier === 'Premium' ? 'Priority placement, growth tools, boosted events' : 'Upgrade for priority placement and growth tools'}
                </p>
              </div>
              <button
                className="rounded-lg px-3 py-1.5 text-[12.5px] font-extrabold text-white bg-accent opacity-50 cursor-not-allowed shrink-0"
                disabled
                title="Billing management is coming soon — see SKIPPED.md"
              >
                <span className="flex items-center gap-1.5"><Wallet size={14} /> Manage</span>
              </button>
            </div>

            <TeamSection isOwner={me?.vendorRole !== 'staff' && me?.vendorRole !== 'manager'} />
          </div>
        </div>

        <div className="card p-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="w-10 h-10 rounded-xl flex items-center justify-center shrink-0" style={{ background: 'rgba(225,29,72,0.1)', color: '#e11d48' }}>
              <LogOut size={18} />
            </span>
            <div>
              <p className="font-extrabold text-[14px]">Log out</p>
              <p className="text-[12.5px] text-muted font-semibold">Sign out of the Vendor Portal</p>
            </div>
          </div>
          <button
            className="rounded-xl px-4 py-2.5 font-extrabold text-[13.5px] text-white shrink-0"
            style={{ background: '#e11d48' }}
            onClick={() => {
              api.setToken(null);
              router.replace('/login');
            }}
          >
            Log out
          </button>
        </div>
      </div>
    </Shell>
  );
}

function SignInSecurityCard({ me }: { me?: User }) {
  return (
    <div className="card p-5">
      <CardHeader icon={Lock} title="Sign-in & security" sub="Keep your account safe" />
      <PasswordSection />
      <div className="h-px bg-line my-4" />
      <TwoFactorSection enabled={me?.totpEnabled ?? false} />
    </div>
  );
}

function PasswordSection() {
  const [open, setOpen] = useState(false);
  const [current, setCurrent] = useState('');
  const [next, setNext] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const change = useMutation({
    mutationFn: () => api.put<{ token: string }>('/auth/password', { currentPassword: current, newPassword: next }),
    onSuccess: (res) => {
      api.setToken(res.token);
      setCurrent('');
      setNext('');
      setOpen(false);
      setSuccess(true);
      setTimeout(() => setSuccess(false), 2500);
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Could not change password'),
  });

  if (!open) {
    return (
      <div className="flex items-center justify-between">
        <div>
          <p className="font-extrabold text-[14px]">Password</p>
          <p className="text-[12.5px] text-muted font-semibold">{success ? 'Changed just now' : 'Keep this up to date'}</p>
        </div>
        <button className="rounded-xl border border-line bg-white px-4 py-2 font-extrabold text-[13px]" onClick={() => setOpen(true)}>
          Change
        </button>
      </div>
    );
  }

  return (
    <div>
      <p className="font-extrabold text-[14px] mb-3">Change password</p>
      <input className="input mb-2.5" type="password" placeholder="Current password" value={current}
        onChange={(e) => { setCurrent(e.target.value); setError(null); }} />
      <input className="input mb-2.5" type="password" placeholder="New password (min 10 characters)" value={next}
        onChange={(e) => { setNext(e.target.value); setError(null); }} />
      {error && <p className="text-[13px] font-bold text-coral mb-2.5">{error}</p>}
      <div className="flex gap-2">
        <button className="btn-accent px-4 py-2 text-[13px]" disabled={!current || next.length < 10 || change.isPending}
          onClick={() => change.mutate()}>
          {change.isPending ? 'Updating…' : 'Update password'}
        </button>
        <button className="rounded-xl border border-line bg-white px-4 py-2 font-extrabold text-[13px]"
          onClick={() => { setOpen(false); setError(null); }}>
          Cancel
        </button>
      </div>
    </div>
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
    mutationFn: () => api.post<{ token: string }>('/auth/2fa/disable', { code: disableCode }),
    onSuccess: (res) => {
      api.setToken(res.token);
      setDisableCode('');
      qc.invalidateQueries({ queryKey: ['me'] });
    },
    onError: (e) => setError(e instanceof Error ? e.message : 'Invalid code'),
  });

  return (
    <div>
      <div className="flex items-center justify-between mb-1">
        <div className="flex items-center gap-2.5">
          <ShieldCheck size={17} className={enabled ? 'text-accent' : 'text-muted'} />
          <div>
            <p className="font-extrabold text-[14px]">Two-factor auth</p>
            <p className="text-[12.5px] text-muted font-semibold">Add an extra step at login</p>
          </div>
        </div>
        {!enabled && !setup && (
          <label className="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" className="sr-only peer" checked={false} onChange={() => startSetup.mutate()} disabled={startSetup.isPending} />
            <span className="w-10 h-6 rounded-full bg-line peer-checked:bg-accent transition-colors relative">
              <span className="absolute left-0.5 top-0.5 w-5 h-5 rounded-full bg-white transition-transform" />
            </span>
          </label>
        )}
        {enabled && (
          <label className="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" className="sr-only peer" checked readOnly />
            <span className="w-10 h-6 rounded-full bg-accent relative">
              <span className="absolute left-[22px] top-0.5 w-5 h-5 rounded-full bg-white transition-transform" />
            </span>
          </label>
        )}
      </div>

      {enabled && (
        <div className="mt-3">
          <input className="input mb-2.5" placeholder="Current 6-digit code to disable" value={disableCode}
            onChange={(e) => { setDisableCode(e.target.value); setError(null); }} maxLength={6} inputMode="numeric" />
          {error && <p className="text-[13px] font-bold text-coral mb-2.5">{error}</p>}
          <button className="rounded-xl border border-coral/40 bg-[#fde4d8] text-coral px-4 py-2 font-extrabold text-[13px]"
            disabled={disableCode.length !== 6 || disable.isPending}
            onClick={() => disable.mutate()}>
            {disable.isPending ? 'Disabling…' : 'Disable 2FA'}
          </button>
        </div>
      )}

      {!enabled && setup && (
        <div className="mt-3">
          <div className="flex items-start gap-4 mb-3">
            <div className="bg-white p-2 rounded-lg border border-line">
              <QRCodeSVG value={setup.otpauthUrl} size={110} />
            </div>
            <div className="text-[12.5px] text-muted font-semibold">
              <p className="mb-2">Scan with your authenticator app, or enter this key manually:</p>
              <code className="block bg-sand rounded-lg px-2 py-1.5 font-mono text-[11.5px] break-all">{setup.secret}</code>
            </div>
          </div>
          <input className="input mb-2.5" placeholder="6-digit code from the app" value={code}
            onChange={(e) => { setCode(e.target.value); setError(null); }} maxLength={6} inputMode="numeric" />
          {error && <p className="text-[13px] font-bold text-coral mb-2.5">{error}</p>}
          <div className="flex gap-2">
            <button className="btn-accent px-4 py-2 flex items-center gap-2 text-[13px]"
              disabled={code.length !== 6 || confirmEnable.isPending}
              onClick={() => confirmEnable.mutate()}>
              <KeyRound size={15} /> {confirmEnable.isPending ? 'Verifying…' : 'Confirm & enable'}
            </button>
            <button className="rounded-xl border border-line bg-white px-4 py-2 font-extrabold text-[13px]"
              onClick={() => setSetup(null)}>
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

const NOTIF_TOGGLES: { key: keyof Pick<User, 'notifyRedemptions' | 'notifyReviews' | 'notifyPayouts' | 'notifyGrowthTips'>; title: string; sub: string; icon: LucideIcon }[] = [
  { key: 'notifyRedemptions', title: 'Redemption alerts', sub: 'When a traveler claims an offer', icon: Ticket },
  { key: 'notifyReviews', title: 'New reviews', sub: 'When someone rates your spot', icon: Star },
  { key: 'notifyPayouts', title: 'Payout updates', sub: 'Subscription & billing changes', icon: Wallet },
  { key: 'notifyGrowthTips', title: 'Growth tips', sub: 'Occasional marketing ideas', icon: Lightbulb },
];

function NotificationsCard({ me }: { me?: User }) {
  const qc = useQueryClient();
  const update = useMutation({
    mutationFn: (next: Record<string, boolean>) => api.put('/me/notification-prefs', next),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['me'] }),
  });

  const toggle = (key: string, value: boolean) => {
    if (!me) return;
    update.mutate({
      notifyRedemptions: me.notifyRedemptions,
      notifyReviews: me.notifyReviews,
      notifyPayouts: me.notifyPayouts,
      notifyGrowthTips: me.notifyGrowthTips,
      [key]: value,
    });
  };

  return (
    <div className="card p-5">
      <CardHeader icon={Bell} title="Notifications" sub="What we email & push you" />
      <div>
        {NOTIF_TOGGLES.map((t, i) => {
          const on = me?.[t.key] ?? false;
          const Icon = t.icon;
          return (
            <div key={t.key} className={`flex items-center gap-3 py-2.5 ${i ? 'border-t border-line' : ''}`}>
              <span className="w-9 h-9 rounded-xl bg-accent-tint text-accent flex items-center justify-center shrink-0"><Icon size={17} /></span>
              <div className="flex-1 min-w-0">
                <p className="font-extrabold text-[14px]">{t.title}</p>
                <p className="text-[12.5px] text-muted font-semibold">{t.sub}</p>
              </div>
              <label className="relative inline-flex items-center cursor-pointer shrink-0">
                <input type="checkbox" className="sr-only peer" checked={on}
                  onChange={(e) => toggle(t.key, e.target.checked)} disabled={update.isPending} />
                <span className={`w-10 h-6 rounded-full relative transition-colors ${on ? 'bg-accent' : 'bg-line'}`}>
                  <span className={`absolute top-0.5 w-5 h-5 rounded-full bg-white transition-all ${on ? 'left-[22px]' : 'left-0.5'}`} />
                </span>
              </label>
            </div>
          );
        })}
      </div>
    </div>
  );
}

const ROLE_META: Record<string, { label: string; sub: string; icon: LucideIcon }> = {
  staff: { label: 'Staff', sub: 'Scan & validate QR codes at the counter', icon: QrCode },
  manager: { label: 'Manager', sub: 'Create & edit offers, view reports, scan', icon: Settings2 },
};

function TeamSection({ isOwner }: { isOwner: boolean }) {
  const qc = useQueryClient();
  const [expanded, setExpanded] = useState(false);
  const [inviteEmail, setInviteEmail] = useState('');
  const [inviteRole, setInviteRole] = useState<'staff' | 'manager'>('staff');
  const [newInviteLink, setNewInviteLink] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const { data: team } = useQuery<TeamResponse>({
    queryKey: ['vendor-team'],
    queryFn: () => api.get<TeamResponse>('/vendor/team'),
  });

  const createInvite = useMutation({
    mutationFn: () => api.post<{ code: string }>('/vendor/team/invite', { email: inviteEmail.trim(), role: inviteRole }),
    onSuccess: (inv) => {
      setNewInviteLink(`${window.location.origin}/join/${inv.code}`);
      setInviteEmail('');
      qc.invalidateQueries({ queryKey: ['vendor-team'] });
    },
  });
  const removeMember = useMutation({
    mutationFn: (userId: string) => api.del(`/vendor/team/members/${userId}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['vendor-team'] }),
  });
  const revokeInvite = useMutation({
    mutationFn: (id: string) => api.del(`/vendor/team/invites/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['vendor-team'] }),
  });

  const members = team?.members ?? [];
  const invites = team?.invites ?? [];
  const palette = ['#0B6F86', '#F0682F', '#7C5CFF', '#16A34A', '#0369A1'];

  return (
    <div className="rounded-2xl border border-line p-4">
      <button className="w-full flex items-center justify-between text-left" onClick={() => setExpanded((v) => !v)}>
        <div className="flex items-center gap-3">
          <span className="w-9 h-9 rounded-xl bg-white border border-line flex items-center justify-center">
            <UserPlus size={16} className="text-accent" />
          </span>
          <div>
            <p className="font-extrabold text-[14px]">Invite a team member</p>
            <p className="text-[12.5px] text-muted font-semibold">
              {`${members.length} member${members.length === 1 ? '' : 's'} · ${invites.length} pending · staff scan & manage offers`}
            </p>
          </div>
        </div>
        <div className="flex -space-x-2">
          {members.slice(0, 4).map((m, i) => (
            <span key={m.id} className="w-8 h-8 rounded-full border-2 border-white text-white text-[11px] font-extrabold flex items-center justify-center"
              style={{ background: palette[i % palette.length] }}>
              {initialsOf(m.name)}
            </span>
          ))}
        </div>
      </button>

      {expanded && (
        <div className="mt-4 pt-4 border-t border-line">
          <div className="space-y-2 mb-3">
            {members.map((m) => (
              <div key={m.id} className="flex items-center justify-between rounded-xl bg-sand px-3 py-2.5">
                <div className="flex items-center gap-2.5">
                  <span className="w-8 h-8 rounded-full text-white text-[11px] font-extrabold flex items-center justify-center" style={{ background: '#0B6F86' }}>
                    {initialsOf(m.name)}
                  </span>
                  <div>
                    <p className="font-extrabold text-[13px]">{m.name} <span className="text-muted font-semibold">· {roleLabel(m.vendorRole)}</span></p>
                    <p className="text-[11.5px] text-muted font-semibold">{m.email}</p>
                  </div>
                </div>
                {isOwner && m.vendorRole !== 'owner' && (
                  <button className="text-coral" title="Remove" onClick={() => removeMember.mutate(m.id)} disabled={removeMember.isPending}>
                    <Trash2 size={16} />
                  </button>
                )}
              </div>
            ))}
            {invites.map((inv) => (
              <div key={inv.id} className="flex items-center gap-3 rounded-xl border border-dashed border-line px-3 py-2.5">
                <span className="w-9 h-9 rounded-xl flex items-center justify-center shrink-0" style={{ background: 'rgba(246,184,30,0.16)', color: '#c98a00' }}>
                  <Clock size={16} />
                </span>
                <div className="flex-1 min-w-0">
                  <p className="font-extrabold text-[13px] truncate">{inv.email || 'Pending invite'}</p>
                  <p className="text-[11.5px] text-muted font-semibold">{roleLabel(inv.role)} · expires {timeAgo(inv.expiresAt)}</p>
                </div>
                {isOwner && (
                  <button className="text-coral shrink-0" title="Revoke" onClick={() => revokeInvite.mutate(inv.id)} disabled={revokeInvite.isPending}>
                    <X size={16} />
                  </button>
                )}
              </div>
            ))}
          </div>

          {isOwner && (
            <div className="rounded-xl bg-sand p-3.5">
              <div className="relative mb-2.5">
                <Mail size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-muted" />
                <input className="input pl-9" type="email" placeholder="name@email.com" value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                  onKeyDown={(e) => { if (e.key === 'Enter' && inviteEmail.trim()) createInvite.mutate(); }} />
              </div>
              <div className="grid grid-cols-2 gap-2 mb-2.5">
                {(['staff', 'manager'] as const).map((r) => {
                  const on = inviteRole === r;
                  const Icon = ROLE_META[r].icon;
                  return (
                    <button key={r} type="button" onClick={() => setInviteRole(r)}
                      className={`rounded-xl px-3 py-2.5 text-left transition-colors border ${on ? 'border-accent bg-accent-tint' : 'border-line bg-white'}`}>
                      <span className={`flex items-center gap-1.5 font-extrabold text-[13px] ${on ? 'text-accent' : 'text-ink'}`}>
                        <Icon size={14} /> {ROLE_META[r].label}
                      </span>
                      <span className="block text-[11px] text-muted font-semibold mt-1 leading-snug">{ROLE_META[r].sub}</span>
                    </button>
                  );
                })}
              </div>

              {newInviteLink ? (
                <div className="rounded-xl bg-white border border-line p-3 flex items-center justify-between gap-2">
                  <code className="text-[12px] font-mono break-all">{newInviteLink}</code>
                  <button
                    className="shrink-0 rounded-lg bg-sand border border-line px-2.5 py-1.5 text-[12px] font-extrabold flex items-center gap-1"
                    onClick={() => {
                      navigator.clipboard.writeText(newInviteLink);
                      setCopied(true);
                      setTimeout(() => setCopied(false), 1500);
                    }}
                  >
                    {copied ? <Check size={13} /> : <Copy size={13} />} {copied ? 'Copied' : 'Copy'}
                  </button>
                </div>
              ) : (
                <button className="btn-accent w-full px-4 py-2.5 text-[13.5px] flex items-center justify-center gap-2"
                  disabled={createInvite.isPending} onClick={() => createInvite.mutate()}>
                  <Send size={15} /> {createInvite.isPending ? 'Sending…' : 'Send invite'}
                </button>
              )}
              <p className="text-[11px] text-muted font-semibold mt-2">
                {inviteEmail.trim() ? "We'll email the invite — the link above still works if it bounces." : "Leave email blank to just generate a copy-link."}
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
