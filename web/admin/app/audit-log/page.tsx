'use client';

// Cross-admin history of destructive actions (vendor approve/reject/
// suspend, feed moderation, subscription upgrades, vendor-grant admin
// overrides) — super-admin only, since it names which specific admin did
// what. Port of the access-grants page's flat-table convention.

import { useQuery } from '@tanstack/react-query';
import { timeAgo, type AdminAuditLogEntry } from '@spotto/shared';
import Shell, { useMe } from '../../components/shell';
import { api } from '../../lib/api';

function targetLabel(e: AdminAuditLogEntry) {
  return e.targetId ? `${e.targetType} · ${e.targetId}` : e.targetType;
}

export default function AuditLogPage() {
  const { data: me } = useMe();
  const { data: entries = [] } = useQuery<AdminAuditLogEntry[]>({
    queryKey: ['admin-audit-log'],
    queryFn: () => api.get<AdminAuditLogEntry[]>('/admin/audit-log'),
    enabled: !!me?.isSuperAdmin,
  });

  if (me && !me.isSuperAdmin) {
    return (
      <Shell>
        <div className="panel p-8 text-center text-muted font-semibold">
          Super-admin only — ask a super-admin for access if you need to review this history.
        </div>
      </Shell>
    );
  }

  return (
    <Shell>
      <div className="panel overflow-hidden">
        <table className="w-full text-[13.5px]">
          <thead className="sticky top-0" style={{ background: 'var(--surface)' }}>
            <tr className="text-left border-b border-line">
              <Th>When</Th><Th>Admin</Th><Th>Action</Th><Th>Target</Th>
            </tr>
          </thead>
          <tbody>
            {entries.map((e) => (
              <tr key={e.id} className="border-b border-line last:border-0">
                <td className="px-4 py-3 text-muted font-medium whitespace-nowrap">{timeAgo(e.createdAt)}</td>
                <td className="px-4 py-3 font-bold">{e.actorName || e.actorEmail || e.actorId}</td>
                <td className="px-4 py-3 font-mono text-[12.5px]">{e.action}</td>
                <td className="px-4 py-3 text-muted font-medium">{targetLabel(e)}</td>
              </tr>
            ))}
            {entries.length === 0 && (
              <tr><td colSpan={4} className="px-4 py-8 text-center text-muted font-semibold">No admin actions recorded yet.</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </Shell>
  );
}

function Th({ children }: { children?: React.ReactNode }) {
  return <th className="px-4 py-3 text-[11px] font-bold uppercase text-muted">{children}</th>;
}
