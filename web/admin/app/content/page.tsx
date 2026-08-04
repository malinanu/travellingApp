'use client';

// Content monitor — polled live feed (3s), pause/resume toggle, filter tabs,
// optimistic approve/remove/warn. Port of ContentMonitor.jsx.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, CheckCircle2, Flag, Pause, Play, ShieldAlert, XCircle } from 'lucide-react';
import { useState } from 'react';

import { timeAgo, type FeedItem, type FeedStatus } from '@spotto/shared';
import Shell from '../../components/shell';
import { InlineErrorBanner } from '../../components/route-status';
import { api } from '../../lib/api';

const TABS: { id: FeedStatus | 'all'; label: string }[] = [
  { id: 'all', label: 'All Statuses' },
  { id: 'ok', label: 'OK' },
  { id: 'review', label: 'Needs review' },
  { id: 'flag', label: 'Flagged' },
];

const CONTENT_TYPES = [
  { id: 'all', label: 'All Content' },
  { id: 'Review', label: 'Reviews' },
  { id: 'Photo', label: 'Photos' },
  { id: 'Checkin', label: 'Check-ins' },
  { id: 'Offer', label: 'Offers' },
];

const STATUS_STYLE: Record<FeedStatus, string> = {
  ok: 'text-[color:var(--muted)]',
  review: 'text-[#d97706]',
  flag: 'text-[#e11d48]',
};

export default function ContentPage() {
  const qc = useQueryClient();
  const [live, setLive] = useState(true);
  const [tab, setTab] = useState<FeedStatus | 'all'>('all');
  const [contentType, setContentType] = useState<string>('all');

  const { data: items = [], isError } = useQuery<FeedItem[]>({
    queryKey: ['admin-feed'],
    queryFn: () => api.get<FeedItem[]>('/admin/feed?after=0'),
    refetchInterval: live ? 3000 : false,
  });

  const resolve = useMutation({
    mutationFn: ({ id, action }: { id: number; action: 'approve' | 'remove' | 'warn' }) =>
      api.post(`/admin/feed/${id}/${action}`),
    onMutate: async ({ id, action }) => {
      await qc.cancelQueries({ queryKey: ['admin-feed'] });
      const prev = qc.getQueryData<FeedItem[]>(['admin-feed']);
      qc.setQueryData<FeedItem[]>(['admin-feed'], (old = []) =>
        old.map((i) => (i.id === id ? { ...i, resolved: action === 'approve' ? 'approved' : action === 'remove' ? 'removed' : 'warned' } : i)),
      );
      return { prev };
    },
    onError: (_e, _v, ctx) => ctx?.prev && qc.setQueryData(['admin-feed'], ctx.prev),
  });

  const filtered = items.filter(
    (i) => (tab === 'all' || i.status === tab) && (contentType === 'all' || i.type === contentType),
  );

  return (
    <Shell>
      {isError && <InlineErrorBanner message="Couldn't load the content feed — try refreshing." />}
      <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
        <div className="flex flex-wrap gap-1.5">
          {TABS.map((t) => (
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              className="btn text-[13px] font-bold"
              style={{
                background: tab === t.id ? 'var(--accent)' : 'var(--surface)',
                color: tab === t.id ? '#fff' : 'var(--ink)',
                borderColor: tab === t.id ? 'var(--accent)' : 'var(--line)',
              }}
            >
              {t.label}
            </button>
          ))}
          <div className="w-[1px] h-6 bg-line self-center mx-1" />
          {CONTENT_TYPES.map((ct) => (
            <button
              key={ct.id}
              onClick={() => setContentType(ct.id)}
              className="btn text-[13px] font-bold"
              style={{
                background: contentType === ct.id ? 'var(--ink)' : 'var(--surface)',
                color: contentType === ct.id ? '#fff' : 'var(--ink)',
                borderColor: contentType === ct.id ? 'var(--ink)' : 'var(--line)',
              }}
            >
              {ct.label}
            </button>
          ))}
        </div>
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              className="btn"
              style={{
                background: tab === t.id ? 'var(--accent)' : 'var(--surface)',
                color: tab === t.id ? '#fff' : 'var(--ink)',
                borderColor: tab === t.id ? 'var(--accent)' : 'var(--line)',
              }}
            >
              {t.label}
            </button>
          ))}
        </div>
        <button className="btn flex items-center gap-2" onClick={() => setLive((l) => !l)}>
          {live ? <Pause size={14} /> : <Play size={14} />} {live ? 'Live' : 'Paused'}
        </button>
      </div>

      <div className="panel divide-y" style={{ borderColor: 'var(--line)' }}>
        {filtered.map((i) => (
          <div key={i.id} className="feed-in flex items-start gap-4 px-5 py-4" style={{ borderColor: 'var(--line)' }}>
            <span className={`mt-0.5 ${STATUS_STYLE[i.status]}`}>
              {i.status === 'flag' ? <ShieldAlert size={18} /> : i.status === 'review' ? <AlertTriangle size={18} /> : <Flag size={18} />}
            </span>
            <div className="flex-1 min-w-0">
              <p className="font-bold text-[14px]">
                {i.vendor} <span className="text-muted font-medium">· {i.town} · {i.type}</span>
              </p>
              <p className="text-[13.5px] font-medium mt-0.5">{i.text}</p>
              {i.reason && <p className="text-[12.5px] font-semibold mt-1" style={{ color: '#d97706' }}>{i.reason}</p>}
              {i.resolved && (
                <p className="text-[12px] font-bold mt-1.5 uppercase" style={{ color: 'var(--accent)' }}>{i.resolved}</p>
              )}
            </div>
            <span className="text-[11.5px] text-muted font-semibold whitespace-nowrap mt-0.5">{timeAgo(i.createdAt)}</span>
            {!i.resolved && (
              <div className="flex gap-1.5 shrink-0">
                <button aria-label="Approve" className="btn !p-2" onClick={() => resolve.mutate({ id: i.id, action: 'approve' })}>
                  <CheckCircle2 size={15} className="text-accent" />
                </button>
                <button aria-label="Warn" className="btn !p-2" onClick={() => resolve.mutate({ id: i.id, action: 'warn' })}>
                  <AlertTriangle size={15} style={{ color: '#d97706' }} />
                </button>
                <button aria-label="Remove" className="btn !p-2" onClick={() => resolve.mutate({ id: i.id, action: 'remove' })}>
                  <XCircle size={15} style={{ color: '#e11d48' }} />
                </button>
              </div>
            )}
          </div>
        ))}
        {filtered.length === 0 && <p className="px-5 py-8 text-center text-muted font-semibold">Nothing here.</p>}
      </div>
    </Shell>
  );
}
