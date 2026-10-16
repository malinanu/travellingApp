'use client';

// Local Guide conversations — every traveller who has chatted with the
// guide, with their full transcript. Split out of the Local Guide dashboard
// so the usage stats stay scannable. ?user=<travellerId> opens a transcript
// directly.

import { useQuery } from '@tanstack/react-query';
import { ArrowLeft, MessageCircle, Search, Sparkles, XCircle } from 'lucide-react';
import Link from 'next/link';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';
import { Suspense, type ReactNode } from 'react';

import { timeAgo, type AdminGuideUserSummary, type AdminGuideMessage } from '@spotto/shared';
import Shell from '../../../components/shell';
import { InlineErrorBanner } from '../../../components/route-status';
import { api } from '../../../lib/api';

export default function LocalGuideConversationsPage() {
  return (
    <Shell>
      <Suspense>
        <Conversations />
      </Suspense>
    </Shell>
  );
}

function Conversations() {
  const { data: users = [], isError, isLoading } = useQuery<AdminGuideUserSummary[]>({
    queryKey: ['admin-guide-users'],
    queryFn: () => api.get<AdminGuideUserSummary[]>('/admin/local-guide/users'),
    refetchInterval: 30000,
  });

  const router = useRouter();
  const pathname = usePathname();
  const selectedId = useSearchParams().get('user');
  const selected = users.find((u) => u.travellerId === selectedId) ?? null;
  const select = (id: string) => router.replace(`${pathname}?user=${encodeURIComponent(id)}`, { scroll: false });

  return (
    <>
      <Link href="/local-guide" className="inline-flex items-center gap-1.5 text-[12px] font-bold text-muted hover:text-[var(--ink)] mb-3">
        <ArrowLeft size={13} /> Local Guide
      </Link>
      {isError && <InlineErrorBanner message="Couldn't load Local Guide conversations — try refreshing." />}

      <SectionLabel>Conversations{users.length > 0 && ` · ${users.length.toLocaleString()} travellers`}</SectionLabel>
      <div className="grid grid-cols-1 lg:grid-cols-[340px_1fr] gap-5 items-start">
        <div className="panel overflow-hidden">
          <div className="max-h-[calc(100vh-220px)] overflow-y-auto">
            {users.map((u) => {
              return (
                <button
                  key={u.travellerId}
                  onClick={() => select(u.travellerId)}
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
            {isLoading && (
              <p className="px-4 py-8 text-center text-muted font-semibold text-[13px]">Loading…</p>
            )}
            {!isLoading && users.length === 0 && (
              <p className="px-4 py-8 text-center text-muted font-semibold text-[13px]">No users yet.</p>
            )}
          </div>
        </div>

        {selected ? (
          <UserTranscript travellerId={selected.travellerId} travellerName={selected.travellerName} />
        ) : (
          <div className="panel p-10 text-center text-muted font-semibold">
            {selectedId && !isLoading ? 'That traveller has no Local Guide history.' : 'Select a user to see their full chat history.'}
          </div>
        )}
      </div>
    </>
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

function SectionLabel({ children }: { children: ReactNode }) {
  return <p className="text-[11.5px] font-bold text-muted uppercase tracking-wide mb-2">{children}</p>;
}
