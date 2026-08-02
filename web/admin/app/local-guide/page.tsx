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
  type AdminGuideConversationSummary, type AdminGuideMessage, type AdminGuideStatsResponse,
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
  const { data: conversations = [], isError: convError } = useQuery<AdminGuideConversationSummary[]>({
    queryKey: ['admin-guide-conversations'],
    queryFn: () => api.get<AdminGuideConversationSummary[]>('/admin/local-guide/conversations'),
    refetchInterval: 30000,
  });

  const [selectedId, setSelectedId] = useState<string | null>(null);
  const selected = conversations.find((c) => c.id === selectedId) ?? null;

  const stats = data?.stats;
  const toolUsage = data?.toolUsage ?? [];
  const maxTool = Math.max(1, ...toolUsage.map((t) => t.count));
  const successRate = stats && stats.requests7d > 0
    ? `${Math.round((stats.okCount7d / stats.requests7d) * 100)}%`
    : '—';

  return (
    <Shell>
      {(statsError || convError) && (
        <InlineErrorBanner message="Couldn't load Local Guide data — try refreshing." />
      )}

      <div className="mb-4">
        <p className="text-[11.5px] font-bold text-muted uppercase tracking-wide mb-2">Local Guide · last 7 days</p>
        <div className="grid grid-cols-2 sm:grid-cols-4 xl:grid-cols-7 gap-3">
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

      <div className="grid grid-cols-[340px_1fr] gap-5 items-start">
        <div className="panel overflow-hidden">
          <div className="max-h-[calc(100vh-220px)] overflow-y-auto">
            {conversations.map((c) => (
              <button
                key={c.id}
                onClick={() => setSelectedId(c.id)}
                className="w-full text-left px-4 py-3 border-b border-line last:border-0 transition-colors"
                style={{ background: selected?.id === c.id ? 'color-mix(in srgb, var(--accent) 10%, transparent)' : 'transparent' }}
              >
                <div className="flex items-center justify-between gap-2">
                  <p className="font-bold text-[13.5px] truncate">{c.travellerName || 'Traveller'}</p>
                  <span className="text-[11px] text-muted font-semibold shrink-0">{c.messageCount} msgs</span>
                </div>
                <p className="text-[12px] text-muted font-medium truncate mt-0.5">{c.lastMessage || '—'}</p>
                <p className="text-[11px] text-muted font-semibold mt-0.5">{timeAgo(c.updatedAt)}</p>
              </button>
            ))}
            {conversations.length === 0 && (
              <p className="px-4 py-8 text-center text-muted font-semibold text-[13px]">No conversations yet.</p>
            )}
          </div>
        </div>

        {selected ? (
          <ConversationTranscript conversationId={selected.id} travellerName={selected.travellerName} />
        ) : (
          <div className="panel p-10 text-center text-muted font-semibold">
            Select a conversation to see the full transcript.
          </div>
        )}
      </div>
    </Shell>
  );
}

function ConversationTranscript({ conversationId, travellerName }: { conversationId: string; travellerName: string }) {
  const { data: messages = [], isError, isLoading } = useQuery<AdminGuideMessage[]>({
    queryKey: ['admin-guide-conversation', conversationId],
    queryFn: () => api.get<AdminGuideMessage[]>(`/admin/local-guide/conversations/${conversationId}`),
  });

  return (
    <div className="panel p-6">
      <h2 className="text-xl font-bold flex items-center gap-2">
        <Search size={16} className="text-muted" /> {travellerName || 'Traveller'}
      </h2>
      {isError && <InlineErrorBanner message="Couldn't load this conversation." />}
      {isLoading && <p className="text-muted font-semibold text-[13px] mt-4">Loading transcript…</p>}
      <div className="space-y-3 mt-4">
        {messages.map((m, i) => (
          <div key={i} className="flex" style={{ justifyContent: m.role === 'user' ? 'flex-end' : 'flex-start' }}>
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
        ))}
        {!isLoading && messages.length === 0 && (
          <p className="text-muted font-semibold text-[13px]">No messages in this conversation.</p>
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
