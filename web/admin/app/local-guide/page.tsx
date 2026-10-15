'use client';

// Local Guide dashboard — usage/reliability stats plus a browsable
// conversation transcript viewer, so an admin can see not just "how much"
// but "what actually happened" when something goes wrong (e.g. a burst of
// rate-limited requests).

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  AlertTriangle, CheckCircle2, Clock, Database, Gauge, MessageCircle, RefreshCw, Search, SlidersHorizontal,
  Sparkles, Wand2, Wrench, XCircle, Zap,
} from 'lucide-react';
import { useState, type ReactNode } from 'react';

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
      {data && !data.configured && (
        <InlineErrorBanner message="GEMINI_API_KEY isn't set on the server — Local Guide, semantic search and embeddings are all off." />
      )}

      <SectionLabel>Local Guide · last 7 days</SectionLabel>
      <div className="grid grid-cols-2 sm:grid-cols-3 xl:grid-cols-6 gap-3 mb-6">
        <StatTile icon={MessageCircle} label="Conversations"
          value={stats ? stats.totalConversations.toLocaleString() : '—'}
          sub={stats ? `${stats.totalMessages.toLocaleString()} messages total` : undefined} />
        <StatTile icon={Clock} label="Messages"
          value={stats ? stats.messages7d.toLocaleString() : '—'}
          sub={stats ? `${stats.messagesToday} today` : undefined} />
        <StatTile icon={CheckCircle2} label="Success rate"
          value={successRate}
          tone={stats && stats.requests7d > 0 && stats.okCount7d < stats.requests7d ? 'bad' : undefined}
          sub={stats ? `${stats.okCount7d} of ${stats.requests7d} requests` : undefined} />
        <StatTile icon={AlertTriangle} label="Rate-limited"
          value={stats ? `${stats.rateLimitedCount7d}` : '—'}
          tone={stats && stats.rateLimitedCount7d > 0 ? 'warn' : undefined}
          sub="Gemini quota hit (429)" />
        <StatTile icon={XCircle} label="Errors"
          value={stats ? `${stats.errorCount7d}` : '—'}
          tone={stats && stats.errorCount7d > 0 ? 'bad' : undefined}
          sub="Other failures" />
        <StatTile icon={Wand2} label="Avg latency"
          value={stats ? formatMs(stats.avgLatencyMs7d) : '—'}
          sub={stats ? `${stats.semanticQueryCount7d} semantic queries` : undefined} />
      </div>

      {data?.quota && <AIUsageSection data={data} />}

      {(toolUsage.length > 0 || recentErrors.length > 0) && (
        <div className="grid grid-cols-1 xl:grid-cols-[1fr_2fr] gap-4 mb-6 items-start">
          <div className="panel px-4 py-4">
            <CardTitle icon={Wrench}>Tool usage · 7 days</CardTitle>
            {toolUsage.length === 0 && <p className="text-[12px] text-muted font-semibold">No tool calls yet.</p>}
            {toolUsage.map((t) => (
              <div key={t.tool} className="mb-2.5 last:mb-0">
                <div className="flex items-baseline justify-between gap-2 mb-1">
                  <span className="text-[12px] font-bold font-mono truncate">{t.tool}</span>
                  <span className="text-[12px] font-semibold text-muted">{t.count.toLocaleString()}</span>
                </div>
                <Meter pct={(t.count / maxTool) * 100} color="var(--accent)" />
              </div>
            ))}
          </div>

          <div className="panel px-4 py-4">
            <CardTitle icon={XCircle} right={<span className="text-[11px] font-semibold text-muted">Click a row for Google&apos;s details</span>}>
              Recent AI errors
            </CardTitle>
            {recentErrors.length === 0 && <p className="text-[12px] text-muted font-semibold">No errors recorded.</p>}
            <div className="max-h-80 overflow-y-auto space-y-2">
              {recentErrors.map((e, i) => (
                <ErrorRow key={i} feature={e.feature} createdAt={e.createdAt} message={e.errorMessage} />
              ))}
            </div>
          </div>
        </div>
      )}

      <SectionLabel>Conversations</SectionLabel>
      <div className="grid grid-cols-1 lg:grid-cols-[340px_1fr] gap-5 items-start">
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

// ── Shared bits ───────────────────────────────────────────────────────────

const RED = '#e11d48';
const AMBER = '#d97706';
const GREEN = '#059669';

type Tone = 'ok' | 'warn' | 'bad' | 'idle';
const TONE_COLOR: Record<Tone, string> = { ok: GREEN, warn: AMBER, bad: RED, idle: 'var(--muted)' };

function SectionLabel({ children }: { children: ReactNode }) {
  return <p className="text-[11.5px] font-bold text-muted uppercase tracking-wide mb-2">{children}</p>;
}

function CardTitle({ icon: Icon, children, right }: { icon: typeof MessageCircle; children: ReactNode; right?: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-2 mb-3">
      <p className="flex items-center gap-1.5 text-[12.5px] font-bold"><Icon size={13} className="text-muted" /> {children}</p>
      {right}
    </div>
  );
}

function Pill({ tone, children }: { tone: Tone; children: ReactNode }) {
  const color = TONE_COLOR[tone];
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-bold whitespace-nowrap"
      style={{ color, background: `color-mix(in srgb, ${color} 14%, transparent)` }}>
      <span className="w-1.5 h-1.5 rounded-full" style={{ background: color }} />
      {children}
    </span>
  );
}

function Meter({ pct, color, height = 6 }: { pct: number; color: string; height?: number }) {
  return (
    <div className="rounded-full overflow-hidden" style={{ height, background: 'var(--line)' }}>
      <div className="h-full rounded-full transition-[width] duration-500"
        style={{ width: `${Math.max(0, Math.min(100, pct))}%`, background: color, minWidth: pct > 0 ? 3 : 0 }} />
    </div>
  );
}

function StatTile({
  icon: Icon,
  label,
  value,
  sub,
  tone,
}: {
  icon: typeof MessageCircle;
  label: string;
  value: string;
  sub?: string;
  tone?: Tone;
}) {
  return (
    <div className="panel px-3.5 py-3">
      <p className="flex items-center gap-1.5 text-[11.5px] font-bold text-muted"><Icon size={12} /> {label}</p>
      <p className="text-[22px] leading-tight font-bold mt-1.5 tabular-nums" style={{ color: tone ? TONE_COLOR[tone] : 'var(--ink)' }}>{value}</p>
      {sub && <p className="text-[11px] text-muted font-semibold mt-0.5 truncate">{sub}</p>}
    </div>
  );
}

function formatMs(ms: number) {
  return ms >= 1000 ? `${(ms / 1000).toFixed(1)}s` : `${Math.round(ms)}ms`;
}

// Google's daily quotas reset at midnight Pacific; ai_usage_events' "today"
// uses the same day boundary.
function untilPacificMidnight() {
  const now = new Date();
  const la = new Date(now.toLocaleString('en-US', { timeZone: 'America/Los_Angeles' }));
  const next = new Date(la);
  next.setHours(24, 0, 0, 0);
  const mins = Math.max(0, Math.round((next.getTime() - la.getTime()) / 60000));
  return `${Math.floor(mins / 60)}h ${mins % 60}m`;
}

// parseGeminiError splits the server's upstream error text (see
// server/internal/gemini/client.go upstreamError) into readable parts, e.g.
// "gemini: rate limited (429) (daily quota used up): chat: quota
// generativelanguage.googleapis.com/generate_content_free_tier_requests
// limit 20 model gemini-3.8-flash; retry in 4h58m44s".
function parseGeminiError(msg: string) {
  const quota = /quota (\S+) limit (\d+) model (\S+?)(?:; retry in (\S+))?$/.exec(msg);
  const daily = msg.includes('daily quota used up');
  const limited = msg.includes('rate limited (429)');
  const headline = daily ? 'Daily quota used up' : limited ? 'Rate limited (429)' : 'Request failed';
  if (!quota) {
    const detail = msg.replace(/^gemini: rate limited \(429\)(?: \(daily quota used up\))?: \w+: /, '');
    return { headline, detail, fields: [] as { k: string; v: string }[], tone: (limited ? 'warn' : 'bad') as Tone };
  }
  const fields = [
    { k: 'Model', v: quota[3] },
    { k: 'Limit', v: quota[2] },
    { k: 'Quota', v: quota[1].replace('generativelanguage.googleapis.com/', '') },
  ];
  if (quota[4]) fields.push({ k: 'Resets in', v: quota[4] });
  return { headline, detail: '', fields, tone: (daily ? 'bad' : 'warn') as Tone };
}

function ErrorDetail({ message }: { message: string }) {
  const e = parseGeminiError(message);
  return (
    <div className="rounded-lg px-3 py-2.5 mt-2" style={{ background: `color-mix(in srgb, ${TONE_COLOR[e.tone]} 8%, transparent)`, border: `1px solid color-mix(in srgb, ${TONE_COLOR[e.tone]} 25%, transparent)` }}>
      <p className="text-[12px] font-bold" style={{ color: TONE_COLOR[e.tone] }}>{e.headline}</p>
      {e.fields.length > 0 && (
        <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5 mt-1.5 text-[11.5px]">
          {e.fields.map((f) => (
            <div key={f.k} className="contents">
              <dt className="font-semibold text-muted">{f.k}</dt>
              <dd className="font-mono break-all">{f.v}</dd>
            </div>
          ))}
        </dl>
      )}
      {e.detail && <p className="text-[11.5px] font-mono break-words mt-1 text-ink-soft">{e.detail}</p>}
    </div>
  );
}

function ErrorRow({ feature, createdAt, message }: { feature: string; createdAt: string; message: string }) {
  const [open, setOpen] = useState(false);
  const e = parseGeminiError(message || '');
  const model = e.fields.find((f) => f.k === 'Model')?.v;
  return (
    <div className="rounded-lg px-3 py-2" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
      <button className="w-full flex items-center gap-2 text-left" onClick={() => setOpen((o) => !o)}>
        <Pill tone={e.tone}>{e.headline}</Pill>
        <span className="text-[12px] font-bold truncate">{featureLabel(feature)}</span>
        {model && <span className="text-[11px] font-mono text-muted truncate hidden sm:inline">{model}</span>}
        <span className="text-[11px] font-semibold text-muted ml-auto shrink-0">{timeAgo(createdAt)}</span>
      </button>
      {open && (message ? <ErrorDetail message={message} /> : <p className="text-[11.5px] text-muted mt-2">(no message recorded)</p>)}
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
  const { quota, quotaUsed, aiUsage, daily, embeddings, embedWorker, models, rateLimitRPM } = data;
  const sum = (pick: (r: AdminAIUsageRow) => number) => aiUsage.reduce((n, r) => n + pick(r), 0);
  const windows = [
    { label: 'Today', inTok: sum((r) => r.inputTokensToday), outTok: sum((r) => r.outputTokensToday), req: sum((r) => r.requestsToday) },
    { label: '7 days', inTok: sum((r) => r.inputTokens7d), outTok: sum((r) => r.outputTokens7d), req: sum((r) => r.requests7d) },
    { label: '30 days', inTok: sum((r) => r.inputTokens30d), outTok: sum((r) => r.outputTokens30d), req: sum((r) => r.requests30d) },
  ];

  return (
    <>
      <SectionLabel>AI usage · Gemini</SectionLabel>
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 mb-4">
        <QuotaCard quota={quota} used={quotaUsed} />
        <HealthCard models={models} />
        <EmbeddingsCard embeddings={embeddings} worker={embedWorker} quota={quota} used={quotaUsed} />
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-[2fr_1fr] gap-4 mb-4 items-start">
        <TokensCard windows={windows} points={daily} />
        <LimitsForm quota={quota} rateLimitRPM={rateLimitRPM} />
      </div>

      {aiUsage.length > 0 && (
        <div className="panel px-4 py-4 mb-6 overflow-x-auto">
          <CardTitle icon={Sparkles}>By feature · 30 days</CardTitle>
          <table className="w-full text-[12px] min-w-[640px]">
            <thead>
              <tr className="text-muted text-left text-[11px] uppercase tracking-wide">
                <th className="font-bold pb-2 pr-3">Feature</th>
                <th className="font-bold pb-2 pr-3">Model</th>
                <th className="font-bold pb-2 pr-3 text-right">Calls today</th>
                <th className="font-bold pb-2 pr-3 text-right">Calls 7d</th>
                <th className="font-bold pb-2 pr-3 text-right">Tokens 7d</th>
                <th className="font-bold pb-2 pr-3 text-right">Tokens 30d</th>
                <th className="font-bold pb-2 text-right">Failures 7d</th>
              </tr>
            </thead>
            <tbody>
              {aiUsage.map((r) => {
                const failures = r.errors7d + r.rateLimited7d;
                return (
                  <tr key={`${r.feature}-${r.model}`} className="border-t border-line">
                    <td className="py-2 pr-3 font-bold">{featureLabel(r.feature)}</td>
                    <td className="py-2 pr-3 font-mono text-muted">{r.model}</td>
                    <td className="py-2 pr-3 text-right font-semibold tabular-nums">{r.requestsToday.toLocaleString()}</td>
                    <td className="py-2 pr-3 text-right font-semibold tabular-nums">{r.requests7d.toLocaleString()}</td>
                    <td className="py-2 pr-3 text-right font-semibold tabular-nums">
                      {r.tokensEstimated && '~'}{compact.format(r.inputTokens7d + r.outputTokens7d)}
                    </td>
                    <td className="py-2 pr-3 text-right font-semibold tabular-nums">
                      {r.tokensEstimated && '~'}{compact.format(r.inputTokens30d + r.outputTokens30d)}
                    </td>
                    <td className="py-2 text-right">
                      {failures === 0 ? <span className="font-semibold text-muted">0</span> : (
                        <span className="inline-flex gap-1.5 justify-end flex-wrap">
                          {r.errors7d > 0 && <Pill tone="bad">{r.errors7d} errors</Pill>}
                          {r.rateLimited7d > 0 && <Pill tone="warn">{r.rateLimited7d} rate-limited</Pill>}
                        </span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
          <p className="text-[11px] text-muted font-semibold mt-3">~ Embedding tokens are estimated (characters ÷ 4); Google doesn&apos;t report them.</p>
        </div>
      )}
    </>
  );
}

function QuotaCard({ quota, used }: { quota: AdminAIQuota; used: AdminGuideStatsResponse['quotaUsed'] }) {
  const rows = [
    { label: 'Chat requests', used: used.chatRequestsToday, limit: quota.chatRequestsPerDay },
    { label: 'Chat tokens', used: used.chatTokensToday, limit: quota.chatTokensPerDay },
    { label: 'Embedding requests', used: used.embedRequestsToday, limit: quota.embedRequestsPerDay },
  ];
  const worst = Math.max(...rows.map((r) => (r.limit > 0 ? r.used / r.limit : 0)));
  const tone: Tone = worst >= 1 ? 'bad' : worst >= 0.8 ? 'warn' : 'ok';
  return (
    <div className="panel px-4 py-4 flex flex-col">
      <CardTitle icon={Gauge} right={<Pill tone={tone}>{tone === 'bad' ? 'Limit reached' : tone === 'warn' ? 'Running low' : 'Healthy'}</Pill>}>
        Quota left today
      </CardTitle>
      <div className="flex-1 flex flex-col gap-4">
        {rows.map((r) => {
          const pct = r.limit > 0 ? (r.used / r.limit) * 100 : 0;
          const color = pct >= 100 ? RED : pct >= 80 ? AMBER : 'var(--accent)';
          return (
            <div key={r.label}>
              <div className="flex items-baseline justify-between gap-2 mb-1.5">
                <span className="text-[12px] font-bold">{r.label}</span>
                <span className="text-[11.5px] font-semibold text-muted tabular-nums">
                  <span className="text-[14px] font-bold" style={{ color: pct >= 100 ? RED : 'var(--ink)' }}>{compact.format(Math.max(0, r.limit - r.used))}</span>
                  {' '}left of {compact.format(r.limit)}
                </span>
              </div>
              <Meter pct={pct} color={color} />
            </div>
          );
        })}
      </div>
      <p className="text-[11px] text-muted font-semibold mt-4 pt-3 border-t border-line flex items-center gap-1.5">
        <Clock size={11} /> Resets in {untilPacificMidnight()} (midnight Pacific)
      </p>
    </div>
  );
}

function HealthCard({ models }: { models: { chat: string; embed: string } }) {
  const health = useMutation({
    mutationFn: () => api.post<AdminAIHealthResponse>('/admin/local-guide/health'),
  });
  const results: Record<string, AdminAIHealthResponse['chat'] | undefined> = {
    Chat: health.data?.chat,
    Embedding: health.data?.embed,
  };
  const anyFailed = !!health.data && (!health.data.chat.ok || !health.data.embed.ok);
  const tone: Tone = !health.data ? 'idle' : anyFailed ? 'bad' : 'ok';

  return (
    <div className="panel px-4 py-4 flex flex-col">
      <CardTitle icon={Zap} right={<Pill tone={tone}>{!health.data ? 'Not tested' : anyFailed ? 'Failing' : 'Connected'}</Pill>}>
        Connection
      </CardTitle>
      <div className="space-y-2 flex-1">
        {([['Chat', models.chat], ['Embedding', models.embed]] as const).map(([label, model]) => {
          const r = results[label];
          return (
            <div key={label} className="rounded-lg px-3 py-2" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
              <div className="flex items-center justify-between gap-2">
                <div className="min-w-0">
                  <p className="text-[11px] font-bold text-muted uppercase tracking-wide">{label}</p>
                  <p className="text-[12.5px] font-mono font-semibold truncate">{model}</p>
                </div>
                {r && (
                  <span className="flex items-center gap-1 text-[11.5px] font-bold shrink-0" style={{ color: r.ok ? GREEN : RED }}>
                    {r.ok ? <CheckCircle2 size={13} /> : <XCircle size={13} />} {formatMs(r.latencyMs)}
                  </span>
                )}
              </div>
              {r?.error && <ErrorDetail message={r.error} />}
            </div>
          );
        })}
      </div>
      <button className="btn btn-primary py-1.5 px-4 text-[12px] mt-4 self-start flex items-center gap-1.5" disabled={health.isPending} onClick={() => health.mutate()}>
        <Zap size={12} /> {health.isPending ? 'Testing…' : 'Test connection'}
      </button>
      {health.isError && <ErrorDetail message={(health.error as Error).message} />}
    </div>
  );
}

function EmbeddingsCard({ embeddings, worker, quota, used }: {
  embeddings: AdminGuideStatsResponse['embeddings'];
  worker: AdminGuideStatsResponse['embedWorker'];
  quota: AdminAIQuota;
  used: AdminGuideStatsResponse['quotaUsed'];
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
  const pct = embeddings.total > 0 ? (embeddings.embedded / embeddings.total) * 100 : 0;
  // Mirrors embeddingQueryReserve in server/internal/http/embedding_worker.go.
  const reserve = Math.max(50, Math.floor(quota.embedRequestsPerDay / 5));
  const shareUsed = quota.embedRequestsPerDay > 0 && used.embedRequestsToday >= quota.embedRequestsPerDay - reserve;
  const paused = !!worker.pausedUntil && new Date(worker.pausedUntil).getTime() > Date.now();

  let tone: Tone = 'ok';
  let status = 'Running';
  let note = worker.lastRunAt ? `Last batch ${timeAgo(worker.lastRunAt)} · ${worker.lastEmbedded} embedded` : 'Waiting for its first batch since the last deploy';
  if (embeddings.pending === 0) {
    status = 'Complete';
    note = 'Every place has an embedding for the current model.';
  } else if (paused) {
    tone = 'warn';
    status = 'Paused';
    note = `Google rate-limited the last batch. Resumes at ${new Date(worker.pausedUntil!).toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })}.`;
  } else if (shareUsed) {
    tone = 'warn';
    status = "Today's share used";
    note = `The last ${reserve} embedding requests of the day are kept for live searches. Resumes in ${untilPacificMidnight()}.`;
  } else if (worker.lastError) {
    tone = 'bad';
    status = 'Failing';
  }

  return (
    <div className="panel px-4 py-4 flex flex-col">
      <CardTitle icon={Database} right={<Pill tone={tone}>{status}</Pill>}>Place embeddings</CardTitle>
      <div className="flex items-baseline gap-2">
        <p className="text-[24px] leading-none font-bold tabular-nums">{Math.floor(pct)}%</p>
        <p className="text-[12px] font-semibold text-muted tabular-nums">
          {embeddings.embedded.toLocaleString()} of {embeddings.total.toLocaleString()} places
        </p>
      </div>
      <div className="my-3"><Meter pct={pct} color="var(--accent)" height={8} /></div>
      <p className="text-[11.5px] font-semibold text-muted">
        {embeddings.pending.toLocaleString()} pending · <span className="font-mono">{embeddings.model}</span>
      </p>
      <p className="text-[11.5px] font-semibold text-ink-soft mt-1.5 flex-1">{note}</p>
      {worker.lastError && !paused && !shareUsed && <ErrorDetail message={worker.lastError} />}
      <div className="flex items-center gap-2 mt-4 pt-3 border-t border-line">
        {confirming ? (
          <>
            <span className="text-[11.5px] font-semibold text-muted mr-auto">Clear all {embeddings.total.toLocaleString()} and rebuild?</span>
            <button className="btn py-1 px-3 text-[12px]" onClick={() => setConfirming(false)}>Cancel</button>
            <button className="btn btn-danger py-1 px-3 text-[12px]" disabled={reembed.isPending} onClick={() => reembed.mutate()}>
              {reembed.isPending ? 'Clearing…' : 'Rebuild'}
            </button>
          </>
        ) : (
          <button className="btn py-1 px-3 text-[12px] flex items-center gap-1.5" onClick={() => setConfirming(true)}>
            <RefreshCw size={12} /> Re-embed all
          </button>
        )}
      </div>
      {reembed.isError && <p className="text-[11.5px] font-semibold mt-1" style={{ color: RED }}>{(reembed.error as Error).message}</p>}
    </div>
  );
}

function TokensCard({ windows, points }: {
  windows: { label: string; inTok: number; outTok: number; req: number }[];
  points: AdminAIDailyPoint[];
}) {
  const max = Math.max(1, ...points.map((p) => p.inputTokens + p.outputTokens));
  const empty = points.every((p) => p.inputTokens + p.outputTokens === 0);
  const outColor = 'color-mix(in srgb, var(--accent) 45%, transparent)';
  return (
    <div className="panel px-4 py-4">
      <CardTitle icon={Sparkles}>Tokens used</CardTitle>
      <div className="grid grid-cols-3 gap-3 mb-5">
        {windows.map((w) => (
          <div key={w.label} className="rounded-lg px-3 py-2.5" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
            <p className="text-[11px] font-bold text-muted uppercase tracking-wide">{w.label}</p>
            <p className="text-[20px] leading-tight font-bold mt-1 tabular-nums">{compact.format(w.inTok + w.outTok)}</p>
            <p className="text-[11px] text-muted font-semibold mt-0.5 tabular-nums">
              {compact.format(w.inTok)} in · {compact.format(w.outTok)} out
            </p>
            <p className="text-[11px] text-muted font-semibold tabular-nums">{w.req.toLocaleString()} calls</p>
          </div>
        ))}
      </div>

      <div className="flex items-center justify-between mb-2">
        <p className="text-[11px] font-bold text-muted uppercase tracking-wide">Per day · last 14 days (Pacific)</p>
        <span className="flex items-center gap-3 text-[11px] font-semibold text-muted">
          <span className="flex items-center gap-1"><span className="inline-block w-2 h-2 rounded-sm" style={{ background: 'var(--accent)' }} /> input</span>
          <span className="flex items-center gap-1"><span className="inline-block w-2 h-2 rounded-sm" style={{ background: outColor }} /> output</span>
        </span>
      </div>
      <p className="text-[10.5px] font-semibold text-muted tabular-nums mb-1 h-3.5">{!empty && `${compact.format(max)} max`}</p>
      <div className="relative h-36 border-b border-line">
        <div className="absolute inset-0 border-t border-dashed border-line" />
        <div className="absolute inset-0 flex items-end gap-1.5">
          {points.map((p) => {
            const total = p.inputTokens + p.outputTokens;
            return (
              <div key={p.day} className="group flex-1 flex flex-col justify-end h-full relative"
                title={`${p.day}: ${total.toLocaleString()} tokens (${p.inputTokens.toLocaleString()} in, ${p.outputTokens.toLocaleString()} out), ${p.requests} calls`}>
                <div className="rounded-t-sm" style={{ height: `${(p.outputTokens / max) * 100}%`, background: outColor }} />
                <div className={p.outputTokens > 0 ? '' : 'rounded-t-sm'}
                  style={{ height: `${(p.inputTokens / max) * 100}%`, background: 'var(--accent)', minHeight: total > 0 ? 3 : 0 }} />
                {total === 0 && <div className="h-[3px] rounded-sm" style={{ background: 'var(--line)' }} />}
              </div>
            );
          })}
        </div>
        {empty && (
          <p className="absolute inset-0 flex items-center justify-center text-[12px] font-semibold text-muted">No Gemini calls in the last 14 days</p>
        )}
      </div>
      <div className="flex gap-1.5 mt-1.5">
        {points.map((p, i) => (
          <span key={p.day} className="flex-1 text-center text-[10px] font-semibold text-muted tabular-nums">
            {i % 2 === points.length % 2 ? '' : p.day.slice(5).replace('-', '/')}
          </span>
        ))}
      </div>
    </div>
  );
}

function LimitsForm({ quota, rateLimitRPM }: { quota: AdminAIQuota; rateLimitRPM: number }) {
  const qc = useQueryClient();
  const save = useMutation({
    mutationFn: async (v: { rpm: number; quota: AdminAIQuota }) => {
      await api.put('/admin/local-guide/rate-limit', { rateLimitRPM: v.rpm });
      await api.put<AdminAIQuota>('/admin/local-guide/quota', v.quota);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin-guide-stats'] }),
  });
  const fields: { name: string; label: string; hint: string; value: number }[] = [
    { name: 'rpm', label: 'Messages / min', hint: 'Per traveller · chat rate limit', value: rateLimitRPM },
    { name: 'chatRequestsPerDay', label: 'Chat requests / day', hint: 'Chat model RPD', value: quota.chatRequestsPerDay },
    { name: 'chatTokensPerDay', label: 'Chat tokens / day', hint: 'Input + output', value: quota.chatTokensPerDay },
    { name: 'embedRequestsPerDay', label: 'Embeds / day', hint: 'Embedding model RPD', value: quota.embedRequestsPerDay },
  ];

  return (
    <form className="panel px-4 py-4" onSubmit={(e) => {
      e.preventDefault();
      const fd = new FormData(e.currentTarget);
      const num = (k: string) => parseInt(String(fd.get(k)), 10);
      save.mutate({
        rpm: num('rpm'),
        quota: {
          chatRequestsPerDay: num('chatRequestsPerDay'),
          chatTokensPerDay: num('chatTokensPerDay'),
          embedRequestsPerDay: num('embedRequestsPerDay'),
        },
      });
    }}>
      <CardTitle icon={SlidersHorizontal}>Limits</CardTitle>
      <p className="text-[11.5px] font-semibold text-muted leading-relaxed mb-4">
        Daily quotas only drive the meters; Google enforces the real ones. Copy them from AI Studio → Rate limits.
      </p>
      <div className="grid grid-cols-2 gap-x-3 gap-y-3.5">
        {fields.map((f) => (
          <label key={f.name} className="block min-w-0">
            <span className="text-[12px] font-bold block truncate">{f.label}</span>
            <span className="text-[11px] font-semibold text-muted block mb-1 truncate">{f.hint}</span>
            <input name={f.name} type="number" min="1" required defaultValue={f.value} className="input w-full py-1.5 tabular-nums" />
          </label>
        ))}
      </div>
      <div className="flex items-center gap-3 mt-4">
        <button type="submit" className="btn btn-primary py-1.5 px-4 text-[12px]" disabled={save.isPending}>
          {save.isPending ? 'Saving…' : 'Save limits'}
        </button>
        {save.isSuccess && <span className="text-[11.5px] font-semibold flex items-center gap-1" style={{ color: GREEN }}><CheckCircle2 size={12} /> Saved</span>}
        {save.isError && <span className="text-[11.5px] font-semibold" style={{ color: RED }}>{(save.error as Error).message}</span>}
      </div>
    </form>
  );
}
