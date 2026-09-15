'use client';

// Local Guide dashboard — usage/reliability stats plus a browsable
// conversation transcript viewer, so an admin can see not just "how much"
// but "what actually happened" when something goes wrong (e.g. a burst of
// rate-limited requests).

import { useQuery } from '@tanstack/react-query';
import {
  AlertTriangle, CheckCircle2, Clock, MessageCircle, Search, Sparkles, Wand2, XCircle,
} from 'lucide-react';
import { useState } from 'react';

import {
  timeAgo,
  type AdminGuideUserSummary, type AdminGuideMessage, type AdminGuideStatsResponse,
  type AdminGuideErrorEntry,
} from '@spotto/shared';
import Shell from '../../components/shell';
import { InlineErrorBanner } from '../../components/route-status';
import { api } from '../../lib/api';

export default function LocalGuidePage() {
  const { data, isError: statsError } = useQuery<AdminGuideStatsResponse>({
    queryKey: ['admin-guide-stats'],
    queryFn: () => api.get<AdminGuideStatsResponse>('/admin/local-guide/stats'),
    refetchInterval: 30000,
  });
  const { data: users = [], isError: usersError } = useQuery<AdminGuideUserSummary[]>({
    queryKey: ['admin-guide-users'],
    queryFn: () => api.get<AdminGuideUserSummary[]>('/admin/local-guide/users'),
    refetchInterval: 30000,
  });
  const { data: recentErrors = [] } = useQuery<AdminGuideErrorEntry[]>({
    queryKey: ['admin-guide-errors'],
    queryFn: () => api.get<AdminGuideErrorEntry[]>('/admin/local-guide/errors'),
    refetchInterval: 30000,
  });

  const [selectedId, setSelectedId] = useState<string | null>(null);
  const selected = users.find((u) => u.travellerId === selectedId) ?? null;

  const stats = data?.stats;
  const toolUsage = data?.toolUsage ?? [];
  const maxTool = Math.max(1, ...toolUsage.map((t) => t.count));
  const successRate = stats && stats.requests7d > 0
    ? `${Math.round((stats.okCount7d / stats.requests7d) * 100)}%`
    : '—';

  return (
    <Shell>
      {(statsError || usersError) && (
        <InlineErrorBanner message="Couldn't load Local Guide data — try refreshing." />
      )}

      <div className="flex flex-col xl:flex-row gap-5 mb-5 items-start">
        <div className="flex-1 w-full">
          <p className="text-[11.5px] font-bold text-muted uppercase tracking-wide mb-2">Local Guide · last 7 days</p>
          <div className="grid grid-cols-2 sm:grid-cols-4 xl:grid-cols-4 gap-3">
            <StatTile icon={MessageCircle} label="Conversations"
              value={stats ? `${stats.totalConversations}` : '—'}
              sub={stats ? `${stats.totalMessages} messages total` : undefined} />
            <StatTile icon={Clock} label="Messages (7d)"
              value={stats ? `${stats.messages7d}` : '—'}
              sub={stats ? `${stats.messagesToday} today` : undefined} />
            <StatTile icon={CheckCircle2} label="Success rate"
              value={successRate}
              sub={stats ? `${stats.requests7d} requests` : undefined} />
            <StatTile icon={AlertTriangle} label="Rate-limited"
              value={stats ? `${stats.rateLimitedCount7d}` : '—'}
              warn={!!stats && stats.rateLimitedCount7d > 0}
              sub={stats ? 'OpenRouter free-tier limit hit' : undefined} />
            <StatTile icon={XCircle} label="Errors"
              value={stats ? `${stats.errorCount7d}` : '—'}
              warn={!!stats && stats.errorCount7d > 0} />
            <StatTile icon={Wand2} label="Avg latency"
              value={stats ? `${Math.round(stats.avgLatencyMs7d)}ms` : '—'}
              sub={stats ? `${stats.semanticQueryCount7d} semantic queries` : undefined} />
            <StatTile icon={Sparkles} label="Embeddings"
              value={stats ? `${stats.placesEmbedded}/${stats.placesTotal}` : '—'}
              sub={stats && stats.placesTotal > 0
                ? `${Math.round((stats.placesEmbedded / stats.placesTotal) * 100)}% backfilled`
                : undefined} />
          </div>
        </div>

        {data?.rateLimitRPM !== undefined && (
          <div className="w-full xl:w-80 shrink-0">
            <p className="text-[11.5px] font-bold text-muted uppercase tracking-wide mb-2">Configuration</p>
            <div className="panel p-4 h-[calc(100%-24px)] flex flex-col justify-center">
              <label className="text-[12px] font-bold block mb-1">Max Requests Per Minute (Per User)</label>
              <p className="text-[11.5px] font-semibold text-muted mb-3 leading-relaxed">
                Controls the Local Guide chat rate limit.
              </p>
              <form onSubmit={(e) => {
                e.preventDefault();
                const fd = new FormData(e.currentTarget);
                const limit = parseInt(fd.get('limit') as string, 10);
                if (limit > 0) {
                  api.put('/admin/local-guide/rate-limit', { rateLimitRPM: limit })
                    .then(() => alert('Rate limit updated!'))
                    .catch((err) => alert('Failed to update: ' + err.message));
                }
              }} className="flex items-center gap-2">
                <input name="limit" type="number" min="1" defaultValue={data.rateLimitRPM} className="input flex-1 py-1.5" />
                <button type="submit" className="btn btn-primary py-1.5 px-4 text-[12px]">Save</button>
              </form>
            </div>
          </div>
        )}
      </div>

      {toolUsage.length > 0 && (
        <div className="panel px-4 py-3.5 mb-4">
          <p className="text-[11.5px] font-bold text-muted uppercase tracking-wide mb-2">Tool usage (7d)</p>
          {toolUsage.map((t) => (
            <div key={t.tool} className="flex items-center gap-2 mb-1 last:mb-0">
              <span className="text-[12px] font-bold w-32 shrink-0 truncate">{t.tool}</span>
              <div className="flex-1 h-2 rounded-full" style={{ background: 'var(--line)' }}>
                <div className="h-2 rounded-full" style={{ width: `${(t.count / maxTool) * 100}%`, background: 'var(--accent)' }} />
              </div>
              <span className="text-[12px] font-semibold text-muted w-10 text-right">{t.count}</span>
            </div>
          ))}
        </div>
      )}

      {recentErrors.length > 0 && (
        <div className="panel px-4 py-3.5 mb-4">
          <p className="text-[11.5px] font-bold text-muted uppercase tracking-wide mb-2 flex items-center gap-1.5">
            <XCircle size={12} className="text-red-500" /> Recent errors — real upstream failure reason
          </p>
          <div className="max-h-48 overflow-y-auto space-y-1.5">
            {recentErrors.map((e, i) => (
              <div key={i} className="flex items-start gap-2 text-[12px]">
                <span className="font-semibold text-muted shrink-0 w-24">{timeAgo(e.createdAt)}</span>
                <span className="font-mono flex-1 break-words" style={{ color: '#e11d48' }}>{e.errorMessage || '(no message recorded)'}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="grid grid-cols-[380px_1fr] gap-5 items-start">
        <div className="panel overflow-hidden">
          <div className="max-h-[calc(100vh-220px)] overflow-y-auto">
            {users.map((u) => {
              const cost = (u.inputTokens * 0.075 / 1000000 + u.outputTokens * 0.3 / 1000000).toFixed(4);
              return (
                <button
                  key={u.travellerId}
                  onClick={() => setSelectedId(u.travellerId)}
                  className="w-full text-left px-4 py-3 border-b border-line last:border-0 transition-colors"
                  style={{ background: selected?.travellerId === u.travellerId ? 'color-mix(in srgb, var(--accent) 10%, transparent)' : 'transparent' }}
                >
                  <div className="flex items-center justify-between gap-2">
                    <p className="font-bold text-[13.5px] truncate">{u.travellerName || 'Traveller'}</p>
                    <span className="text-[11px] text-muted font-semibold shrink-0">{timeAgo(u.lastActiveAt)}</span>
                  </div>
                  <div className="flex items-center gap-3 mt-1.5 opacity-80">
                    <p className="text-[11px] font-semibold flex items-center gap-1"><MessageCircle size={10} /> {u.totalMessages} msgs</p>
                    <p className="text-[11px] font-semibold flex items-center gap-1" style={{ color: u.errorCount > 0 ? '#e11d48' : 'inherit' }}><XCircle size={10} /> {u.errorCount} errs</p>
                    <p className="text-[11px] font-semibold flex items-center gap-1"><Sparkles size={10} /> {u.tripsPlanned} trips</p>
                  </div>
                  <div className="flex items-center justify-between gap-2 mt-1 opacity-70">
                    <p className="text-[10px] font-bold uppercase tracking-wider">{(u.inputTokens + u.outputTokens).toLocaleString()} tokens</p>
                    <p className="text-[10px] font-bold uppercase tracking-wider">${cost} est.</p>
                  </div>
                </button>
              );
            })}
            {users.length === 0 && (
              <p className="px-4 py-8 text-center text-muted font-semibold text-[13px]">No users yet.</p>
            )}
          </div>
        </div>

        {selected ? (
          <UserTranscript travellerId={selected.travellerId} travellerName={selected.travellerName} />
        ) : (
          <div className="panel p-10 text-center text-muted font-semibold">
            Select a user to see their full chat history.
          </div>
        )}
      </div>
    </Shell>
  );
}

function UserTranscript({ travellerId, travellerName }: { travellerId: string; travellerName: string }) {
  const { data: messages = [], isError, isLoading } = useQuery<AdminGuideMessage[]>({
    queryKey: ['admin-guide-user-messages', travellerId],
    queryFn: () => api.get<AdminGuideMessage[]>(`/admin/local-guide/users/${travellerId}/messages`),
  });

  let currentConvId = '';

  return (
    <div className="panel p-6">
      <h2 className="text-xl font-bold flex items-center gap-2">
        <Search size={16} className="text-muted" /> {travellerName || 'Traveller'}
      </h2>
      {isError && <InlineErrorBanner message="Couldn't load this transcript." />}
      {isLoading && <p className="text-muted font-semibold text-[13px] mt-4">Loading transcript…</p>}
      <div className="space-y-4 mt-4">
        {messages.map((m, i) => {
          const isNewConv = m.conversationId !== currentConvId;
          if (isNewConv) currentConvId = m.conversationId;
          
          return (
            <div key={i}>
              {isNewConv && (
                <div className="flex items-center gap-2 my-6">
                  <div className="flex-1 h-px bg-line" />
                  <span className="text-[10px] font-bold text-muted uppercase tracking-wider">Conversation {m.conversationId.split('_').pop()}</span>
                  <div className="flex-1 h-px bg-line" />
                </div>
              )}
              <div className="flex" style={{ justifyContent: m.role === 'user' ? 'flex-end' : 'flex-start' }}>
                <div className="max-w-[75%] rounded-xl px-3.5 py-2.5"
                  style={{
                    background: m.role === 'user' ? 'var(--accent)' : 'var(--bg)',
                    color: m.role === 'user' ? '#fff' : 'inherit',
                    border: m.role === 'user' ? 'none' : '1px solid var(--line)',
                  }}>
                  <p className="text-[13px] font-medium leading-relaxed whitespace-pre-wrap">{m.content}</p>
                  {m.placeIds && m.placeIds.length > 0 && (
                    <p className="text-[11px] font-semibold mt-1 opacity-70">{m.placeIds.length} place(s) referenced</p>
                  )}
                  <p className="text-[10.5px] font-semibold mt-1 opacity-60">{timeAgo(m.createdAt)}</p>
                </div>
              </div>
            </div>
          );
        })}
        {!isLoading && messages.length === 0 && (
          <p className="text-muted font-semibold text-[13px]">No messages in history.</p>
        )}
      </div>
    </div>
  );
}

function StatTile({
  icon: Icon,
  label,
  value,
  sub,
  warn,
}: {
  icon: typeof MessageCircle;
  label: string;
  value: string;
  sub?: string;
  warn?: boolean;
}) {
  return (
    <div className="rounded-xl p-3.5" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
      <p className="flex items-center gap-1.5 text-[11.5px] font-bold text-muted"><Icon size={13} /> {label}</p>
      <p className="text-[18px] font-bold mt-1" style={{ color: warn ? '#e11d48' : 'var(--ink)' }}>{value}</p>
      {sub && <p className="text-[11px] text-muted font-semibold mt-0.5">{sub}</p>}
    </div>
  );
}
