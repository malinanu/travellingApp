'use client';

// Local Guide dashboard — usage/reliability stats plus a browsable
// conversation transcript viewer, so an admin can see not just "how much"
// but "what actually happened" when something goes wrong (e.g. a burst of
// rate-limited requests).

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  AlertTriangle, CheckCircle2, Clock, Database, Gauge, MessageCircle, RefreshCw, Search, Sparkles, Wand2,
  XCircle, Zap,
} from 'lucide-react';
import { useState } from 'react';

import {
  timeAgo,
  type AdminGuideUserSummary, type AdminGuideMessage, type AdminGuideStatsResponse,
  type AdminGuideErrorEntry, type AdminAIQuota, type AdminAIHealthResponse, type AdminAIUsageRow,
  type AdminAIDailyPoint,
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
              sub={stats ? 'Gemini quota hit (429)' : undefined} />
            <StatTile icon={XCircle} label="Errors"
              value={stats ? `${stats.errorCount7d}` : '—'}
              warn={!!stats && stats.errorCount7d > 0} />
            <StatTile icon={Wand2} label="Avg latency"
              value={stats ? `${Math.round(stats.avgLatencyMs7d)}ms` : '—'}
              sub={stats ? `${stats.semanticQueryCount7d} semantic queries` : undefined} />
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
              <form className="flex items-center gap-2" onSubmit={(e) => {
                e.preventDefault();
                const fd = new FormData(e.currentTarget);
                const limit = parseInt(fd.get('limit') as string, 10);
                if (limit > 0) {
                  api.put('/admin/local-guide/rate-limit', { rateLimitRPM: limit })
                    .then(() => alert('Rate limit updated!'))
                    .catch((err) => alert('Failed to update: ' + err.message));
                }
              }}>
                <input name="limit" type="number" min="1" defaultValue={data.rateLimitRPM} className="input flex-1 py-1.5" />
                <button type="submit" className="btn btn-primary py-1.5 px-4 text-[12px]">Save</button>
              </form>
              {data.quota && <QuotaForm quota={data.quota} />}
            </div>
          </div>
        )}
      </div>

      {data && !data.configured && (
        <InlineErrorBanner message="GEMINI_API_KEY isn't set on the server — Local Guide, semantic search and embeddings are all off." />
      )}
      {data?.quota && <AIUsageSection data={data} />}

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
            <XCircle size={12} className="text-red-500" /> Recent AI errors — Google's own error text
          </p>
          <div className="max-h-48 overflow-y-auto space-y-1.5">
            {recentErrors.map((e, i) => (
              <div key={i} className="flex items-start gap-2 text-[12px]">
                <span className="font-semibold text-muted shrink-0 w-24">{timeAgo(e.createdAt)}</span>
                <span className="font-bold shrink-0 w-28 truncate" title={e.model}>{featureLabel(e.feature)}</span>
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
                    <p className="text-[10px] font-bold uppercase tracking-wider">{u.inputTokens.toLocaleString()} in · {u.outputTokens.toLocaleString()} out</p>
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

// ── AI usage (Gemini) ─────────────────────────────────────────────────────
// Every upstream Gemini call is recorded server-side (ai_usage_events), so
// this section covers chat, embeddings, BR-cert OCR and inspiration alike.
// "Left to use" is measured against the admin-entered daily quota —
// Google exposes no remaining-quota API.

const FEATURE_LABELS: Record<string, string> = {
  guide_chat: 'Local Guide chat',
  guide_query_embed: 'Guide search embed',
  search_embed: 'Search embed',
  embed_worker: 'Place embeddings',
  br_cert: 'BR cert OCR',
  inspiration: 'Inspiration',
  health_check: 'Health check',
};

function featureLabel(feature: string) {
  return FEATURE_LABELS[feature] ?? feature;
}

const compact = new Intl.NumberFormat('en', { notation: 'compact', maximumFractionDigits: 1 });

function AIUsageSection({ data }: { data: AdminGuideStatsResponse }) {
  const { quota, quotaUsed, aiUsage, daily, embeddings, embedWorker, models } = data;
  const sum = (pick: (r: AdminAIUsageRow) => number) => aiUsage.reduce((n, r) => n + pick(r), 0);
  const windows = [
    { label: 'Today', inTok: sum((r) => r.inputTokensToday), outTok: sum((r) => r.outputTokensToday), req: sum((r) => r.requestsToday) },
    { label: '7 days', inTok: sum((r) => r.inputTokens7d), outTok: sum((r) => r.outputTokens7d), req: sum((r) => r.requests7d) },
    { label: '30 days', inTok: sum((r) => r.inputTokens30d), outTok: sum((r) => r.outputTokens30d), req: sum((r) => r.requests30d) },
  ];

  return (
    <>
      <p className="text-[11.5px] font-bold text-muted uppercase tracking-wide mb-2">AI usage · Gemini</p>
      <div className="grid grid-cols-1 xl:grid-cols-3 gap-4 mb-4">
        <div className="panel px-4 py-3.5">
          <p className="flex items-center gap-1.5 text-[12px] font-bold mb-3"><Gauge size={13} /> Quota left today</p>
          <QuotaBar label="Chat requests" used={quotaUsed.chatRequestsToday} limit={quota.chatRequestsPerDay} />
          <QuotaBar label="Chat tokens" used={quotaUsed.chatTokensToday} limit={quota.chatTokensPerDay} />
          <QuotaBar label="Embedding requests" used={quotaUsed.embedRequestsToday} limit={quota.embedRequestsPerDay} />
          <p className="text-[11px] text-muted font-semibold mt-2">
            Resets at midnight Pacific. Measured against the limits set under Configuration — match them to your AI Studio tier.
          </p>
        </div>

        <div className="panel px-4 py-3.5">
          <p className="flex items-center gap-1.5 text-[12px] font-bold mb-3"><Sparkles size={13} /> Tokens used</p>
          <div className="space-y-2.5">
            {windows.map((w) => (
              <div key={w.label} className="flex items-baseline justify-between gap-2">
                <span className="text-[12px] font-bold w-16">{w.label}</span>
                <span className="text-[18px] font-bold">{compact.format(w.inTok + w.outTok)}</span>
                <span className="text-[11px] text-muted font-semibold text-right">
                  {compact.format(w.inTok)} in · {compact.format(w.outTok)} out<br />{w.req.toLocaleString()} calls
                </span>
              </div>
            ))}
          </div>
          <p className="text-[11px] text-muted font-semibold mt-2">Output includes Gemini&apos;s thinking tokens.</p>
        </div>

        <HealthCard models={models} />
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-[2fr_1fr] gap-4 mb-4">
        <DailyChart points={daily} />
        <EmbeddingsCard embeddings={embeddings} worker={embedWorker} />
      </div>

      {aiUsage.length > 0 && (
        <div className="panel px-4 py-3.5 mb-4 overflow-x-auto">
          <p className="text-[11.5px] font-bold text-muted uppercase tracking-wide mb-2">By feature (30 days)</p>
          <table className="w-full text-[12px]">
            <thead>
              <tr className="text-muted text-left">
                <th className="font-bold py-1 pr-3">Feature</th>
                <th className="font-bold py-1 pr-3">Model</th>
                <th className="font-bold py-1 pr-3 text-right">Today</th>
                <th className="font-bold py-1 pr-3 text-right">7d calls</th>
                <th className="font-bold py-1 pr-3 text-right">7d tokens</th>
                <th className="font-bold py-1 pr-3 text-right">30d tokens</th>
                <th className="font-bold py-1 text-right">7d failures</th>
              </tr>
            </thead>
            <tbody>
              {aiUsage.map((r) => (
                <tr key={`${r.feature}-${r.model}`} className="border-t border-line">
                  <td className="py-1.5 pr-3 font-bold">{featureLabel(r.feature)}</td>
                  <td className="py-1.5 pr-3 font-mono text-muted">{r.model}</td>
                  <td className="py-1.5 pr-3 text-right font-semibold">{r.requestsToday.toLocaleString()}</td>
                  <td className="py-1.5 pr-3 text-right font-semibold">{r.requests7d.toLocaleString()}</td>
                  <td className="py-1.5 pr-3 text-right font-semibold">
                    {r.tokensEstimated && '~'}{compact.format(r.inputTokens7d + r.outputTokens7d)}
                  </td>
                  <td className="py-1.5 pr-3 text-right font-semibold">
                    {r.tokensEstimated && '~'}{compact.format(r.inputTokens30d + r.outputTokens30d)}
                  </td>
                  <td className="py-1.5 text-right font-semibold"
                    style={{ color: r.errors7d + r.rateLimited7d > 0 ? '#e11d48' : 'inherit' }}>
                    {r.errors7d}{r.rateLimited7d > 0 && ` + ${r.rateLimited7d} rate-limited`}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <p className="text-[11px] text-muted font-semibold mt-2">~ embedding tokens are estimated (characters ÷ 4) — Google doesn&apos;t report them.</p>
        </div>
      )}
    </>
  );
}

function QuotaBar({ label, used, limit }: { label: string; used: number; limit: number }) {
  const pct = limit > 0 ? Math.min(100, (used / limit) * 100) : 0;
  const left = Math.max(0, limit - used);
  const color = pct >= 90 ? '#e11d48' : pct >= 70 ? '#d97706' : 'var(--accent)';
  return (
    <div className="mb-2.5">
      <div className="flex items-baseline justify-between gap-2 mb-1">
        <span className="text-[12px] font-bold">{label}</span>
        <span className="text-[11.5px] font-semibold text-muted">
          <span style={{ color: 'var(--ink)' }}>{compact.format(left)}</span> left · {compact.format(used)} / {compact.format(limit)}
        </span>
      </div>
      <div className="h-2 rounded-full" style={{ background: 'var(--line)' }}>
        <div className="h-2 rounded-full" style={{ width: `${pct}%`, background: color }} />
      </div>
    </div>
  );
}

function DailyChart({ points }: { points: AdminAIDailyPoint[] }) {
  const max = Math.max(1, ...points.map((p) => p.inputTokens + p.outputTokens));
  const outColor = 'color-mix(in srgb, var(--accent) 50%, transparent)';
  return (
    <div className="panel px-4 py-3.5">
      <p className="text-[11.5px] font-bold text-muted uppercase tracking-wide mb-3">Tokens per day (14 days, Pacific)</p>
      <div className="flex items-end gap-1.5 h-32">
        {points.map((p) => {
          const total = p.inputTokens + p.outputTokens;
          return (
            <div key={p.day} className="flex-1 flex flex-col justify-end h-full"
              title={`${p.day}: ${total.toLocaleString()} tokens (${p.inputTokens.toLocaleString()} in, ${p.outputTokens.toLocaleString()} out), ${p.requests} calls`}>
              <div className="rounded-t" style={{ height: `${(p.outputTokens / max) * 100}%`, background: outColor }} />
              <div style={{ height: `${(p.inputTokens / max) * 100}%`, background: 'var(--accent)', minHeight: total > 0 ? 2 : 0 }} />
            </div>
          );
        })}
      </div>
      <div className="flex justify-between text-[10.5px] font-semibold text-muted mt-1.5">
        <span>{points[0]?.day.slice(5)}</span>
        <span className="flex items-center gap-3">
          <span className="flex items-center gap-1"><span className="inline-block w-2 h-2 rounded-sm" style={{ background: 'var(--accent)' }} /> input</span>
          <span className="flex items-center gap-1"><span className="inline-block w-2 h-2 rounded-sm" style={{ background: outColor }} /> output</span>
        </span>
        <span>{points[points.length - 1]?.day.slice(5)}</span>
      </div>
    </div>
  );
}

function EmbeddingsCard({ embeddings, worker }: {
  embeddings: AdminGuideStatsResponse['embeddings'];
  worker: AdminGuideStatsResponse['embedWorker'];
}) {
  const qc = useQueryClient();
  const [confirming, setConfirming] = useState(false);
  const reembed = useMutation({
    mutationFn: () => api.post<{ cleared: number }>('/admin/local-guide/reembed'),
    onSuccess: () => {
      setConfirming(false);
      qc.invalidateQueries({ queryKey: ['admin-guide-stats'] });
    },
  });
  const pct = embeddings.total > 0 ? Math.round((embeddings.embedded / embeddings.total) * 100) : 0;

  return (
    <div className="panel px-4 py-3.5">
      <p className="flex items-center gap-1.5 text-[12px] font-bold mb-2"><Database size={13} /> Place embeddings</p>
      <p className="text-[18px] font-bold">{embeddings.embedded.toLocaleString()} / {embeddings.total.toLocaleString()}</p>
      <div className="h-2 rounded-full my-2" style={{ background: 'var(--line)' }}>
        <div className="h-2 rounded-full" style={{ width: `${pct}%`, background: 'var(--accent)' }} />
      </div>
      <p className="text-[11.5px] font-semibold text-muted">
        {pct}% done · {embeddings.pending.toLocaleString()} pending · <span className="font-mono">{embeddings.model}</span>
      </p>
      <p className="text-[11.5px] font-semibold text-muted mt-1">
        Worker: {worker.lastRunAt ? `last ran ${timeAgo(worker.lastRunAt)}, embedded ${worker.lastEmbedded}` : 'no run since last deploy'}
      </p>
      {worker.lastError && (
        <p className="text-[11.5px] font-mono mt-1 break-words" style={{ color: '#e11d48' }}>{worker.lastError}</p>
      )}
      <div className="flex items-center gap-2 mt-3">
        {confirming ? (
          <>
            <button className="btn btn-primary py-1 px-3 text-[12px]" disabled={reembed.isPending} onClick={() => reembed.mutate()}>
              {reembed.isPending ? 'Clearing…' : 'Yes, rebuild all'}
            </button>
            <button className="btn py-1 px-3 text-[12px]" onClick={() => setConfirming(false)}>Cancel</button>
          </>
        ) : (
          <button className="btn py-1 px-3 text-[12px] flex items-center gap-1.5" onClick={() => setConfirming(true)}>
            <RefreshCw size={12} /> Re-embed all
          </button>
        )}
      </div>
      {reembed.isError && <p className="text-[11.5px] font-semibold mt-1" style={{ color: '#e11d48' }}>{(reembed.error as Error).message}</p>}
    </div>
  );
}

function HealthCard({ models }: { models: { chat: string; embed: string } }) {
  const health = useMutation({
    mutationFn: () => api.post<AdminAIHealthResponse>('/admin/local-guide/health'),
  });
  const rows = health.data
    ? [{ label: 'Chat', r: health.data.chat }, { label: 'Embedding', r: health.data.embed }]
    : null;

  return (
    <div className="panel px-4 py-3.5">
      <p className="flex items-center gap-1.5 text-[12px] font-bold mb-2"><Zap size={13} /> Connection</p>
      <p className="text-[11.5px] font-semibold text-muted">Chat <span className="font-mono" style={{ color: 'var(--ink)' }}>{models.chat}</span></p>
      <p className="text-[11.5px] font-semibold text-muted">Embedding <span className="font-mono" style={{ color: 'var(--ink)' }}>{models.embed}</span></p>
      <button className="btn btn-primary py-1.5 px-4 text-[12px] mt-3" disabled={health.isPending} onClick={() => health.mutate()}>
        {health.isPending ? 'Testing…' : 'Test connection'}
      </button>
      {health.isError && (
        <p className="text-[11.5px] font-mono mt-2 break-words" style={{ color: '#e11d48' }}>{(health.error as Error).message}</p>
      )}
      {rows && (
        <div className="mt-2 space-y-1.5">
          {rows.map(({ label, r }) => (
            <div key={label} className="text-[11.5px]">
              <p className="font-bold flex items-center gap-1.5" style={{ color: r.ok ? '#059669' : '#e11d48' }}>
                {r.ok ? <CheckCircle2 size={12} /> : <XCircle size={12} />} {label} {r.ok ? 'OK' : 'failed'} · {r.latencyMs}ms
              </p>
              {r.error && <p className="font-mono break-words" style={{ color: '#e11d48' }}>{r.error}</p>}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function QuotaForm({ quota }: { quota: AdminAIQuota }) {
  const qc = useQueryClient();
  const save = useMutation({
    mutationFn: (q: AdminAIQuota) => api.put<AdminAIQuota>('/admin/local-guide/quota', q),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-guide-stats'] }),
  });
  const fields: { name: keyof AdminAIQuota; label: string }[] = [
    { name: 'chatRequestsPerDay', label: 'Chat requests / day' },
    { name: 'chatTokensPerDay', label: 'Chat tokens / day' },
    { name: 'embedRequestsPerDay', label: 'Embedding requests / day' },
  ];

  return (
    <form className="mt-4 pt-3 border-t border-line" onSubmit={(e) => {
      e.preventDefault();
      const fd = new FormData(e.currentTarget);
      const num = (k: keyof AdminAIQuota) => parseInt(String(fd.get(k)), 10);
      save.mutate({
        chatRequestsPerDay: num('chatRequestsPerDay'),
        chatTokensPerDay: num('chatTokensPerDay'),
        embedRequestsPerDay: num('embedRequestsPerDay'),
      });
    }}>
      <label className="text-[12px] font-bold block mb-1">Gemini daily quota</label>
      <p className="text-[11.5px] font-semibold text-muted mb-2 leading-relaxed">
        Copy these from AI Studio → Rate limits for your project&apos;s tier.
      </p>
      {fields.map((f) => (
        <div key={f.name} className="flex items-center gap-2 mb-1.5">
          <span className="text-[11.5px] font-semibold flex-1">{f.label}</span>
          <input name={f.name} type="number" min="1" defaultValue={quota[f.name]} className="input w-28 py-1" />
        </div>
      ))}
      <div className="flex items-center gap-2 mt-2">
        <button type="submit" className="btn btn-primary py-1.5 px-4 text-[12px]" disabled={save.isPending}>
          {save.isPending ? 'Saving…' : 'Save quota'}
        </button>
        {save.isSuccess && <span className="text-[11.5px] font-semibold" style={{ color: '#059669' }}>Saved</span>}
        {save.isError && <span className="text-[11.5px] font-semibold" style={{ color: '#e11d48' }}>{(save.error as Error).message}</span>}
      </div>
    </form>
  );
}
