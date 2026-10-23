'use client';

// Users — find any account, see the devices it's signed in on and its quest
// history, and sign it out of every device (e.g. when someone reports a lost
// phone). ?user=<id> opens a user directly. Super-admins can also change an
// account between traveller and vendor, or delete it.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowRightLeft, CheckCircle2, KeyRound, LogOut, MonitorSmartphone, Search, ShieldAlert, Trash2, Trophy } from 'lucide-react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useState, type ReactNode } from 'react';

import {
  timeAgo,
  type AdminChangeRoleRequest,
  type AdminChangeRoleResponse,
  type AdminUserDetail,
  type AdminUserSession,
  type AdminUserSummary,
  type AdminVendorTeam,
  type User,
  type Vendor,
  type VendorCategory,
  type VendorRole,
} from '@spotto/shared';
import Shell, { useMe } from '../../components/shell';
import { InlineErrorBanner } from '../../components/route-status';
import { api } from '../../lib/api';

const RED = '#e11d48';
const GREEN = '#059669';

const CLIENT_LABEL: Record<AdminUserSession['client'], string> = {
  app: 'Spotto app',
  admin: 'Admin web',
  vendor: 'Vendor web',
};

export default function UsersPage() {
  return (
    <Shell>
      <Suspense>
        <Users />
      </Suspense>
    </Shell>
  );
}

function Users() {
  const [input, setInput] = useState('');
  const [q, setQ] = useState('');
  useEffect(() => {
    const t = setTimeout(() => setQ(input.trim()), 300);
    return () => clearTimeout(t);
  }, [input]);

  const { data: users = [], isLoading, isError } = useQuery<AdminUserSummary[]>({
    queryKey: ['admin-users', q],
    queryFn: () => api.get<AdminUserSummary[]>(`/admin/users?q=${encodeURIComponent(q)}`),
  });

  const router = useRouter();
  const pathname = usePathname();
  const selectedId = useSearchParams().get('user');
  const select = (id: string) => router.replace(`${pathname}?user=${encodeURIComponent(id)}`, { scroll: false });

  return (
    <>
      <SectionLabel>Users</SectionLabel>
      <div className="grid grid-cols-1 lg:grid-cols-[360px_1fr] gap-5 items-start">
        <div className="panel overflow-hidden">
          <div className="p-3 border-b border-line">
            <label className="flex items-center gap-2 input py-2">
              <Search size={14} className="text-muted shrink-0" />
              <input value={input} onChange={(e) => setInput(e.target.value)} placeholder="Name, email or user id"
                className="bg-transparent outline-none w-full text-[13px]" />
            </label>
          </div>
          {isError && <div className="p-3"><InlineErrorBanner message="Couldn't search users — try again." /></div>}
          <div className="max-h-[calc(100vh-260px)] overflow-y-auto">
            {users.map((u) => (
              <button key={u.id} onClick={() => select(u.id)}
                className="w-full text-left px-4 py-3 border-b border-line last:border-0 transition-colors"
                style={{ background: selectedId === u.id ? 'color-mix(in srgb, var(--accent) 10%, transparent)' : 'transparent' }}>
                <div className="flex items-center justify-between gap-2">
                  <p className="font-bold text-[13.5px] truncate">{u.name || u.email}</p>
                  <RolePill role={u.role} />
                </div>
                <p className="text-[12px] text-muted font-semibold truncate mt-0.5">{u.email}{u.vendorName && ` · ${u.vendorName}`}</p>
                <p className="text-[11px] text-muted font-semibold mt-1">
                  {u.activeSessions} {u.activeSessions === 1 ? 'device' : 'devices'}
                  {u.lastActiveAt && ` · active ${timeAgo(u.lastActiveAt)}`}
                </p>
              </button>
            ))}
            {isLoading && <p className="px-4 py-8 text-center text-muted font-semibold text-[13px]">Loading…</p>}
            {!isLoading && users.length === 0 && (
              <p className="px-4 py-8 text-center text-muted font-semibold text-[13px]">{q ? 'No users match.' : 'No users yet.'}</p>
            )}
          </div>
        </div>

        {selectedId ? (
          <UserDetail key={selectedId} id={selectedId} />
        ) : (
          <div className="panel p-10 text-center text-muted font-semibold">Select a user to see their devices and history.</div>
        )}
      </div>
    </>
  );
}

function UserDetail({ id }: { id: string }) {
  const { data: me } = useMe();
  const { data, isLoading, isError } = useQuery<AdminUserDetail>({
    queryKey: ['admin-user', id],
    queryFn: () => api.get<AdminUserDetail>(`/admin/users/${encodeURIComponent(id)}`),
  });

  if (isError) return <div className="panel p-6"><InlineErrorBanner message="Couldn't load this user." /></div>;
  if (isLoading || !data) return <div className="panel p-10 text-center text-muted font-semibold">Loading…</div>;

  const { user, sessions, questCompletions, audit } = data;
  return (
    <div className="space-y-4">
      <div className="panel p-5">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h2 className="text-xl font-bold truncate">{user.name || user.email}</h2>
            <p className="text-[13px] text-muted font-semibold truncate">{user.email}</p>
          </div>
          <RolePill role={user.role} />
        </div>
        <p className="text-[12px] text-muted font-semibold mt-2">
          Joined {new Date(user.createdAt).toLocaleDateString()} · <span className="font-mono">{user.id}</span>
        </p>
      </div>

      {user.role === 'admin' && me?.isSuperAdmin && (
        <AdminPermissionsPanel userId={user.id} isSuperAdmin={data.isSuperAdmin} permissions={data.adminPermissions} />
      )}

      <div className="panel p-5">
        <CardTitle icon={MonitorSmartphone}>Signed-in devices · {sessions.length}</CardTitle>
        {sessions.length === 0 ? (
          <p className="text-[12.5px] text-muted font-semibold">
            Not signed in on any device. (Sign-ins from before sessions were tracked don&apos;t show here.)
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-[12.5px] min-w-[480px]">
              <thead>
                <tr className="text-muted text-left text-[11px] uppercase tracking-wide">
                  <th className="font-bold pb-2 pr-3">Device</th>
                  <th className="font-bold pb-2 pr-3">Signed in</th>
                  <th className="font-bold pb-2">Last used</th>
                </tr>
              </thead>
              <tbody>
                {sessions.map((s) => (
                  <tr key={s.id} className="border-t border-line">
                    <td className="py-2 pr-3">
                      <div className="font-bold">{CLIENT_LABEL[s.client] ?? s.client}</div>
                      {s.deviceLabel && <div className="text-[11.5px] text-muted truncate max-w-[280px]" title={s.deviceLabel}>{s.deviceLabel}</div>}
                    </td>
                    <td className="py-2 pr-3 text-muted font-semibold" title={new Date(s.createdAt).toLocaleString()}>{timeAgo(s.createdAt)}</td>
                    <td className="py-2 text-muted font-semibold" title={new Date(s.lastUsedAt).toLocaleString()}>{timeAgo(s.lastUsedAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <SignOutEverywhere userId={user.id} />
      </div>

      <div className="panel p-5">
        <CardTitle icon={Trophy}>Quest history · {questCompletions.length}</CardTitle>
        {questCompletions.length === 0 && <p className="text-[12.5px] text-muted font-semibold">No quests completed.</p>}
        <div className="space-y-2 max-h-80 overflow-y-auto pr-1">
          {questCompletions.map((c) => (
            <div key={c.date} className="flex items-center justify-between gap-3 rounded-lg px-3 py-2" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
              <div className="min-w-0">
                <p className="text-[12.5px] font-bold">{c.date} · {c.sponsorName}</p>
                <p className="text-[11.5px] text-muted font-semibold truncate">{c.rewardDescription} · goal {c.goal} · completed {timeAgo(c.completedAt)}</p>
              </div>
              {c.redeemedAt ? (
                <span className="flex items-center gap-1 text-[11.5px] font-bold shrink-0" style={{ color: GREEN }} title={new Date(c.redeemedAt).toLocaleString()}>
                  <CheckCircle2 size={12} /> Redeemed
                </span>
              ) : (
                <span className="text-[11.5px] font-bold text-muted shrink-0">Not redeemed</span>
              )}
            </div>
          ))}
        </div>
      </div>

      {audit.length > 0 && (
        <div className="panel p-5">
          <CardTitle icon={LogOut}>Admin actions</CardTitle>
          <div className="space-y-2">
            {audit.map((e) => {
              const meta = (e.metadata ?? {}) as { reason?: string; sessionsEnded?: number };
              return (
                <div key={e.id} className="text-[12.5px]">
                  <span className="font-bold">{USER_ACTION_LABEL[e.action] ?? e.action}</span>
                  <span className="text-muted font-semibold"> · {e.actorName || e.actorEmail} · {timeAgo(e.createdAt)}</span>
                  {meta.reason && <p className="text-[12px] text-muted">&ldquo;{meta.reason}&rdquo;{meta.sessionsEnded !== undefined && ` · ${meta.sessionsEnded} ended`}</p>}
                </div>
              );
            })}
          </div>
        </div>
      )}

      {me?.isSuperAdmin && user.role !== 'admin' && <SuperAdminControls user={user} />}
    </div>
  );
}

const USER_ACTION_LABEL: Record<string, string> = {
  'user.revoke_sessions': 'Signed out of all devices',
  'user.permissions': 'Permissions changed',
  'user.role_change': 'Account type changed',
};

// Super-admins grant individual admins extra powers. Today that's only
// handing vendor logins over to their real owners (vendor Team tab).
function AdminPermissionsPanel({ userId, isSuperAdmin, permissions }: { userId: string; isSuperAdmin: boolean; permissions: string[] }) {
  const qc = useQueryClient();
  const granted = permissions.includes('vendor_logins');
  const [confirming, setConfirming] = useState(false);
  const [reason, setReason] = useState('');
  const toggle = useMutation({
    mutationFn: () => api.put(`/admin/users/${encodeURIComponent(userId)}/permissions`, { vendorLogins: !granted, reason: reason.trim() }),
    onSuccess: () => {
      setConfirming(false);
      setReason('');
      qc.invalidateQueries({ queryKey: ['admin-user', userId] });
    },
  });

  return (
    <div className="panel p-5">
      <CardTitle icon={KeyRound}>Admin permissions</CardTitle>
      {isSuperAdmin ? (
        <p className="text-[12.5px] text-muted font-semibold">Super-admin: has every permission, including changing vendor logins.</p>
      ) : (
        <>
          <p className="text-[12.5px] font-semibold">
            Change vendor logins: <span style={{ color: granted ? GREEN : undefined }} className={granted ? '' : 'text-muted'}>{granted ? 'Allowed' : 'Not allowed'}</span>
          </p>
          <p className="text-[11.5px] text-muted font-semibold mt-0.5">Lets this admin move a business account to its real owner&apos;s email and reset its password.</p>
          {confirming ? (
            <form className="space-y-2 mt-3" onSubmit={(e) => { e.preventDefault(); toggle.mutate(); }}>
              <label className="block">
                <span className="text-[12px] font-bold block mb-1">Reason (saved in the audit log)</span>
                <input autoFocus required value={reason} onChange={(e) => setReason(e.target.value)} className="input w-full py-1.5"
                  placeholder={granted ? 'e.g. Moved off the support team' : 'e.g. Handles vendor onboarding'} />
              </label>
              <div className="flex items-center justify-end gap-2">
                <button type="button" className="btn py-1 px-3 text-[12px]" onClick={() => setConfirming(false)}>Cancel</button>
                <button type="submit" className={`btn ${granted ? 'btn-danger' : 'btn-primary'} py-1 px-3 text-[12px]`} disabled={toggle.isPending || !reason.trim()}>
                  {toggle.isPending ? 'Saving…' : granted ? 'Remove permission' : 'Allow'}
                </button>
              </div>
            </form>
          ) : (
            <button className="btn py-1.5 px-3 text-[12px] mt-3" onClick={() => { toggle.reset(); setConfirming(true); }}>
              {granted ? 'Remove permission' : 'Allow changing vendor logins'}
            </button>
          )}
          {toggle.isError && <p className="text-[11.5px] font-semibold mt-2" style={{ color: RED }}>{(toggle.error as Error).message}</p>}
        </>
      )}
    </div>
  );
}

// Ends every session the user has: all their devices are back at the login
// screen on their next request.
function SignOutEverywhere({ userId }: { userId: string }) {
  const qc = useQueryClient();
  const [confirming, setConfirming] = useState(false);
  const [reason, setReason] = useState('');
  const revoke = useMutation({
    mutationFn: () => api.post<{ sessionsEnded: number }>(`/admin/users/${encodeURIComponent(userId)}/revoke-sessions`, { reason: reason.trim() }),
    onSuccess: () => {
      setConfirming(false);
      setReason('');
      qc.invalidateQueries({ queryKey: ['admin-user', userId] });
      qc.invalidateQueries({ queryKey: ['admin-users'] });
    },
  });

  return (
    <div className="mt-4 pt-3 border-t border-line">
      {confirming ? (
        <form className="space-y-2" onSubmit={(e) => { e.preventDefault(); revoke.mutate(); }}>
          <label className="block">
            <span className="text-[12px] font-bold block mb-1">Reason (saved in the audit log)</span>
            <input autoFocus required value={reason} onChange={(e) => setReason(e.target.value)}
              placeholder="e.g. User reported a lost phone" className="input w-full py-1.5" />
          </label>
          <div className="flex items-center gap-2">
            <span className="text-[11.5px] font-semibold text-muted mr-auto">Every device, including the apps and web, will need to sign in again.</span>
            <button type="button" className="btn py-1 px-3 text-[12px]" onClick={() => setConfirming(false)}>Cancel</button>
            <button type="submit" className="btn btn-danger py-1 px-3 text-[12px]" disabled={revoke.isPending || !reason.trim()}>
              {revoke.isPending ? 'Signing out…' : 'Sign out everywhere'}
            </button>
          </div>
        </form>
      ) : (
        <div className="flex items-center gap-3">
          <button className="btn py-1.5 px-3 text-[12px] flex items-center gap-1.5" onClick={() => { revoke.reset(); setConfirming(true); }}>
            <LogOut size={12} /> Sign out all devices
          </button>
          {revoke.isSuccess && (
            <span className="text-[11.5px] font-semibold flex items-center gap-1" style={{ color: GREEN }}>
              <CheckCircle2 size={12} /> Signed out of {revoke.data.sessionsEnded} {revoke.data.sessionsEnded === 1 ? 'device' : 'devices'}
            </span>
          )}
        </div>
      )}
      {revoke.isError && <p className="text-[11.5px] font-semibold mt-2" style={{ color: RED }}>{(revoke.error as Error).message}</p>}
    </div>
  );
}

const VENDOR_ROLE_LABEL: Record<VendorRole, string> = { owner: 'Owner', manager: 'Manager', staff: 'Staff' };

const isOwner = (u: User) => u.role === 'vendor' && !!u.vendorId && (u.vendorRole === 'owner' || !u.vendorRole);

// Super-admin only, never for admin accounts: move the account between
// traveller and vendor, or delete it for good. An owner who leaves a
// business that still has other members hands it to one of them; a sole
// owner leaves it with no owner until an admin fixes or deletes it.
function SuperAdminControls({ user }: { user: User }) {
  const owner = isOwner(user);
  const { data: team, isLoading: teamLoading } = useQuery<AdminVendorTeam>({
    queryKey: ['admin-vendor-team', user.vendorId],
    queryFn: () => api.get<AdminVendorTeam>(`/admin/vendors/${encodeURIComponent(user.vendorId ?? '')}/team`),
    enabled: owner,
  });
  const { data: vendors = [] } = useQuery<Vendor[]>({
    queryKey: ['admin-vendors', 'all'],
    queryFn: () => api.get<Vendor[]>('/admin/vendors'),
  });
  const business = vendors.find((v) => v.id === user.vendorId);
  const others = (team?.members ?? []).filter((m) => m.id !== user.id);
  const handover: Handover = !owner
    ? { kind: 'none' }
    : teamLoading
      ? { kind: 'loading' }
      : others.some(isOwner)
        ? { kind: 'none' }
        : others.length === 0
          ? { kind: 'ownerless', businessName: business?.name ?? 'Their business' }
          : { kind: 'choose', candidates: others };

  const [open, setOpen] = useState<'vendor' | 'traveller' | 'delete' | null>(null);
  const [done, setDone] = useState<string | null>(null);
  const finish = (message: string) => {
    setOpen(null);
    setDone(message);
  };

  return (
    <div className="panel p-5">
      <CardTitle icon={ShieldAlert}>Super-admin controls</CardTitle>
      <p className="text-[12.5px] font-semibold">
        Account type:{' '}
        {user.role === 'traveller' ? 'Traveller' : (
          <>
            Vendor ·{' '}
            {user.vendorId
              ? `${VENDOR_ROLE_LABEL[user.vendorRole || 'owner']} of ${business?.name ?? user.vendorId}`
              : <span className="text-muted">No business linked</span>}
          </>
        )}
      </p>

      {open === 'vendor' && <MakeVendorForm user={user} vendors={vendors} onCancel={() => setOpen(null)} onDone={finish} />}
      {open === 'traveller' && (
        <MakeTravellerForm user={user} businessName={business?.name} handover={handover} onCancel={() => setOpen(null)} onDone={finish} />
      )}
      {open === null && (
        <div className="flex flex-wrap items-center gap-2 mt-3">
          {(user.role === 'traveller' || !user.vendorId) && (
            <button className="btn py-1.5 px-3 text-[12px] flex items-center gap-1.5" onClick={() => { setDone(null); setOpen('vendor'); }}>
              <ArrowRightLeft size={12} /> {user.role === 'traveller' ? 'Make vendor' : 'Link to a business'}
            </button>
          )}
          {user.role === 'vendor' && (
            <button className="btn py-1.5 px-3 text-[12px] flex items-center gap-1.5" onClick={() => { setDone(null); setOpen('traveller'); }}>
              <ArrowRightLeft size={12} /> Make traveller
            </button>
          )}
        </div>
      )}
      {done && open === null && (
        <p className="text-[11.5px] font-semibold mt-2 flex items-start gap-1" style={{ color: GREEN }}>
          <CheckCircle2 size={12} className="mt-[2px] shrink-0" /> {done}
        </p>
      )}

      <div className="mt-4 rounded-lg p-3" style={{ border: `1px solid color-mix(in srgb, ${RED} 35%, transparent)`, background: `color-mix(in srgb, ${RED} 5%, transparent)` }}>
        {open === 'delete' ? (
          <DeleteUserForm user={user} businessName={business?.name} handover={handover} onCancel={() => setOpen(null)} />
        ) : (
          <div className="flex items-center gap-3">
            <div className="min-w-0 mr-auto">
              <p className="text-[12.5px] font-bold" style={{ color: RED }}>Delete account permanently</p>
              <p className="text-[11.5px] text-muted font-semibold">Only when it&apos;s absolutely necessary. This can&apos;t be undone.</p>
            </div>
            <button className="btn py-1.5 px-3 text-[12px] flex items-center gap-1.5 shrink-0" style={{ color: RED }}
              onClick={() => { setDone(null); setOpen('delete'); }}>
              <Trash2 size={12} /> Delete…
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

type Handover =
  | { kind: 'none' }
  | { kind: 'loading' }
  | { kind: 'ownerless'; businessName: string }
  | { kind: 'choose'; candidates: User[] };

function invalidateAccount(qc: ReturnType<typeof useQueryClient>, userId: string) {
  qc.invalidateQueries({ queryKey: ['admin-user', userId] });
  qc.invalidateQueries({ queryKey: ['admin-users'] });
  qc.invalidateQueries({ queryKey: ['admin-vendors'] });
  qc.invalidateQueries({ queryKey: ['admin-vendor-team'] });
}

function MakeVendorForm({ user, vendors, onCancel, onDone }: {
  user: User;
  vendors: Vendor[];
  onCancel: () => void;
  onDone: (message: string) => void;
}) {
  const qc = useQueryClient();
  const [mode, setMode] = useState<'existing' | 'new'>('existing');
  const [search, setSearch] = useState('');
  const [vendorId, setVendorId] = useState('');
  const [vendorRole, setVendorRole] = useState<VendorRole>('staff');
  const [name, setName] = useState('');
  const [category, setCategory] = useState('');
  const [town, setTown] = useState('');
  const [reason, setReason] = useState('');
  const { data: categories = [] } = useQuery<VendorCategory[]>({
    queryKey: ['admin_categories'],
    queryFn: () => api.get<VendorCategory[]>('/admin/categories'),
    enabled: mode === 'new',
  });

  const picked = vendors.find((v) => v.id === vendorId);
  const needle = search.trim().toLowerCase();
  const matches = needle
    ? vendors.filter((v) => v.name.toLowerCase().includes(needle) || v.town.toLowerCase().includes(needle) || v.id === needle).slice(0, 8)
    : [];

  const change = useMutation({
    mutationFn: () => {
      const body: AdminChangeRoleRequest = mode === 'existing'
        ? { role: 'vendor', reason: reason.trim(), vendorId, vendorRole }
        : { role: 'vendor', reason: reason.trim(), newBusiness: { name: name.trim(), category, town: town.trim() } };
      return api.post<AdminChangeRoleResponse>(`/admin/users/${encodeURIComponent(user.id)}/role`, body);
    },
    onSuccess: (res) => {
      invalidateAccount(qc, user.id);
      const where = mode === 'existing' ? `${VENDOR_ROLE_LABEL[vendorRole]} of ${picked?.name ?? 'the business'}` : `owner of ${name.trim()} (pending approval)`;
      onDone(`Now a vendor — ${where}. They've been signed out everywhere and sign in again on the vendor site.` +
        (res.noPassword ? ' They signed up with Google or Apple, so they need "Forgot password" on the vendor site to set a password first.' : ''));
    },
  });

  const ready = reason.trim() !== '' && (mode === 'existing' ? !!picked : name.trim() !== '' && category !== '');

  return (
    <form className="space-y-3 mt-3" onSubmit={(e) => { e.preventDefault(); if (ready) change.mutate(); }}>
      <div className="flex gap-2">
        <button type="button" className={`btn py-1 px-3 text-[12px] ${mode === 'existing' ? 'btn-primary' : ''}`} onClick={() => setMode('existing')}>
          Link to an existing business
        </button>
        <button type="button" className={`btn py-1 px-3 text-[12px] ${mode === 'new' ? 'btn-primary' : ''}`} onClick={() => setMode('new')}>
          Create a new business
        </button>
      </div>

      {mode === 'existing' ? (
        <>
          {picked ? (
            <div className="flex items-center justify-between gap-2 rounded-lg px-3 py-2" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
              <div className="min-w-0">
                <p className="text-[12.5px] font-bold truncate">{picked.name}</p>
                <p className="text-[11.5px] text-muted font-semibold truncate">{picked.town} · {picked.status}</p>
              </div>
              <button type="button" className="text-[11.5px] font-bold text-muted shrink-0" onClick={() => { setVendorId(''); setSearch(''); }}>Change</button>
            </div>
          ) : (
            <label className="block">
              <span className="text-[12px] font-bold block mb-1">Business</span>
              <input autoFocus value={search} onChange={(e) => setSearch(e.target.value)} className="input w-full py-1.5" placeholder="Search by name or town" />
              {matches.length > 0 && (
                <div className="mt-1 rounded-lg overflow-hidden" style={{ border: '1px solid var(--line)' }}>
                  {matches.map((v) => (
                    <button type="button" key={v.id} onClick={() => setVendorId(v.id)}
                      className="w-full text-left px-3 py-1.5 border-b border-line last:border-0 text-[12.5px] hover:bg-[var(--bg)]">
                      <span className="font-bold">{v.name}</span>
                      <span className="text-muted font-semibold"> · {v.town} · {v.status}</span>
                    </button>
                  ))}
                </div>
              )}
              {needle && matches.length === 0 && <p className="text-[11.5px] text-muted font-semibold mt-1">No business matches.</p>}
            </label>
          )}
          <label className="block">
            <span className="text-[12px] font-bold block mb-1">Team role</span>
            <select value={vendorRole} onChange={(e) => setVendorRole(e.target.value as VendorRole)} className="input w-full py-1.5">
              <option value="owner">Owner</option>
              <option value="manager">Manager</option>
              <option value="staff">Staff</option>
            </select>
          </label>
        </>
      ) : (
        <>
          <label className="block">
            <span className="text-[12px] font-bold block mb-1">Business name</span>
            <input autoFocus value={name} onChange={(e) => setName(e.target.value)} className="input w-full py-1.5" maxLength={120} />
          </label>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
            <label className="block">
              <span className="text-[12px] font-bold block mb-1">Category</span>
              <select value={category} onChange={(e) => setCategory(e.target.value)} className="input w-full py-1.5">
                <option value="">Choose…</option>
                {[...categories].sort((a, b) => a.label.localeCompare(b.label)).map((c) => <option key={c.id} value={c.id}>{c.label}</option>)}
              </select>
            </label>
            <label className="block">
              <span className="text-[12px] font-bold block mb-1">Town (optional)</span>
              <input value={town} onChange={(e) => setTown(e.target.value)} className="input w-full py-1.5" maxLength={80} />
            </label>
          </div>
          <p className="text-[11.5px] text-muted font-semibold">They&apos;ll own it. It starts as pending, like a new sign-up — approve it on the Vendors page.</p>
        </>
      )}

      <ReasonField value={reason} onChange={setReason} placeholder="e.g. Signed up in the app by mistake — runs a cafe" />
      <FormButtons
        note="They'll be signed out everywhere."
        onCancel={onCancel}
        submitLabel={change.isPending ? 'Changing…' : 'Make vendor'}
        disabled={change.isPending || !ready}
      />
      {change.isError && <ErrorText error={change.error} />}
    </form>
  );
}

function MakeTravellerForm({ user, businessName, handover, onCancel, onDone }: {
  user: User;
  businessName?: string;
  handover: Handover;
  onCancel: () => void;
  onDone: (message: string) => void;
}) {
  const qc = useQueryClient();
  const [newOwnerId, setNewOwnerId] = useState('');
  const [reason, setReason] = useState('');
  const change = useMutation({
    mutationFn: () => {
      const body: AdminChangeRoleRequest = { role: 'traveller', reason: reason.trim(), ...(newOwnerId && { newOwnerId }) };
      return api.post<AdminChangeRoleResponse>(`/admin/users/${encodeURIComponent(user.id)}/role`, body);
    },
    onSuccess: () => {
      invalidateAccount(qc, user.id);
      onDone("Now a traveller. They've been signed out everywhere.");
    },
  });
  const ready = reason.trim() !== '' && handover.kind !== 'loading' && (handover.kind !== 'choose' || newOwnerId !== '');

  return (
    <form className="space-y-3 mt-3" onSubmit={(e) => { e.preventDefault(); if (ready) change.mutate(); }}>
      {user.vendorId && handover.kind === 'none' && (
        <p className="text-[12px] font-semibold">They&apos;ll be removed from {businessName ?? 'their business'}&apos;s team.</p>
      )}
      <HandoverField handover={handover} value={newOwnerId} onChange={setNewOwnerId} />
      <ReasonField value={reason} onChange={setReason} placeholder="e.g. Sold the business, keeps using the app" />
      <FormButtons
        note="They'll be signed out everywhere."
        onCancel={onCancel}
        submitLabel={change.isPending ? 'Changing…' : 'Make traveller'}
        disabled={change.isPending || !ready}
      />
      {change.isError && <ErrorText error={change.error} />}
    </form>
  );
}

function DeleteUserForm({ user, businessName, handover, onCancel }: {
  user: User;
  businessName?: string;
  handover: Handover;
  onCancel: () => void;
}) {
  const qc = useQueryClient();
  const router = useRouter();
  const pathname = usePathname();
  const [newOwnerId, setNewOwnerId] = useState('');
  const [reason, setReason] = useState('');
  const [confirm, setConfirm] = useState('');
  const del = useMutation({
    mutationFn: () => api.post(`/admin/users/${encodeURIComponent(user.id)}/delete`, {
      reason: reason.trim(),
      confirm: confirm.trim(),
      ...(newOwnerId && { newOwnerId }),
    }),
    onSuccess: () => {
      router.replace(pathname, { scroll: false });
      qc.removeQueries({ queryKey: ['admin-user', user.id] });
      invalidateAccount(qc, user.id);
    },
  });
  const ready = reason.trim() !== '' && confirm.trim().toLowerCase() === user.email.toLowerCase() &&
    handover.kind !== 'loading' && (handover.kind !== 'choose' || newOwnerId !== '');

  return (
    <form className="space-y-3" onSubmit={(e) => { e.preventDefault(); if (ready) del.mutate(); }}>
      <div>
        <p className="text-[12.5px] font-bold" style={{ color: RED }}>Delete {user.name || user.email} permanently?</p>
        <p className="text-[11.5px] text-muted font-semibold mt-0.5">
          Removes the account, its sign-ins, trips, groups, saved places and photo.
          {user.vendorId && ` They're taken off ${businessName ?? 'their business'}'s team.`}
          {' '}Reviews, claims and bookings they made stay, shown without a name. This can&apos;t be undone.
        </p>
      </div>
      <HandoverField handover={handover} value={newOwnerId} onChange={setNewOwnerId} />
      <ReasonField value={reason} onChange={setReason} placeholder="e.g. Asked by email to delete their data" />
      <label className="block">
        <span className="text-[12px] font-bold block mb-1">Type <span className="font-mono">{user.email}</span> to confirm</span>
        <input value={confirm} onChange={(e) => setConfirm(e.target.value)} className="input w-full py-1.5" autoComplete="off" spellCheck={false} />
      </label>
      <FormButtons onCancel={onCancel} danger submitLabel={del.isPending ? 'Deleting…' : 'Delete permanently'} disabled={del.isPending || !ready} />
      {del.isError && <ErrorText error={del.error} />}
    </form>
  );
}

// Who takes over the business when its owner leaves.
function HandoverField({ handover, value, onChange }: { handover: Handover; value: string; onChange: (id: string) => void }) {
  if (handover.kind === 'loading') return <p className="text-[12px] text-muted font-semibold">Checking their team…</p>;
  if (handover.kind === 'ownerless') {
    return (
      <p className="text-[12px] font-semibold">
        They&apos;re the only one on {handover.businessName}&apos;s team, so it will have no owner — nobody can manage its team
        until an admin adds one, or deletes the business on the Vendors page.
      </p>
    );
  }
  if (handover.kind !== 'choose') return null;
  return (
    <label className="block">
      <span className="text-[12px] font-bold block mb-1">New owner of the business</span>
      <select value={value} onChange={(e) => onChange(e.target.value)} className="input w-full py-1.5" required>
        <option value="">Choose a team member…</option>
        {handover.candidates.map((m) => (
          <option key={m.id} value={m.id}>{m.name || m.email} · {VENDOR_ROLE_LABEL[m.vendorRole || 'staff']} · {m.email}</option>
        ))}
      </select>
      <span className="text-[11.5px] text-muted font-semibold block mt-1">They own the business and its team from now on.</span>
    </label>
  );
}

function ReasonField({ value, onChange, placeholder }: { value: string; onChange: (v: string) => void; placeholder: string }) {
  return (
    <label className="block">
      <span className="text-[12px] font-bold block mb-1">Reason (saved in the audit log)</span>
      <input required value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder} className="input w-full py-1.5" maxLength={500} />
    </label>
  );
}

function FormButtons({ note, onCancel, submitLabel, disabled, danger }: {
  note?: string;
  onCancel: () => void;
  submitLabel: string;
  disabled: boolean;
  danger?: boolean;
}) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-[11.5px] font-semibold text-muted mr-auto">{note}</span>
      <button type="button" className="btn py-1 px-3 text-[12px]" onClick={onCancel}>Cancel</button>
      <button type="submit" className={`btn ${danger ? 'btn-danger' : 'btn-primary'} py-1 px-3 text-[12px]`} disabled={disabled}
        style={disabled ? { opacity: 0.55, cursor: 'not-allowed' } : undefined}>
        {submitLabel}
      </button>
    </div>
  );
}

function ErrorText({ error }: { error: unknown }) {
  return <p className="text-[11.5px] font-semibold" style={{ color: RED }}>{(error as Error).message}</p>;
}

function RolePill({ role }: { role: string }) {
  return (
    <span className="rounded-full px-2 py-0.5 text-[10.5px] font-bold uppercase tracking-wide shrink-0"
      style={{ color: 'var(--accent)', background: 'color-mix(in srgb, var(--accent) 12%, transparent)' }}>
      {role}
    </span>
  );
}

function SectionLabel({ children }: { children: ReactNode }) {
  return <p className="text-[11.5px] font-bold text-muted uppercase tracking-wide mb-2">{children}</p>;
}

function CardTitle({ icon: Icon, children }: { icon: typeof Search; children: ReactNode }) {
  return (
    <p className="flex items-center gap-1.5 text-[12.5px] font-bold mb-3"><Icon size={13} className="text-muted" /> {children}</p>
  );
}
