'use client';

// Users — find any account, see the devices it's signed in on and its quest
// history, and sign it out of every device (e.g. when someone reports a lost
// phone). ?user=<id> opens a user directly.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { CheckCircle2, KeyRound, LogOut, MonitorSmartphone, Search, Trophy } from 'lucide-react';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useState, type ReactNode } from 'react';

import { timeAgo, type AdminUserDetail, type AdminUserSession, type AdminUserSummary } from '@spotto/shared';
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
    </div>
  );
}

const USER_ACTION_LABEL: Record<string, string> = {
  'user.revoke_sessions': 'Signed out of all devices',
  'user.permissions': 'Permissions changed',
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
