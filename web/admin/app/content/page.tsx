'use client';

// Content — travel posts for travellers: audio guides and audiobooks, blog
// posts, travel guides, itineraries and tips. Admins see everyone's posts
// and review what publishers send in; publishers see only their own.
// ?author=<user id> narrows the list to one author (from the Users page).

import { useQuery } from '@tanstack/react-query';
import { Eye, Headphones, MapPin, Plus, Search, X } from 'lucide-react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useRef, useState } from 'react';

import { resolveMediaUrl, timeAgo, type ContentItem, type ContentStatus, type StudioContentList, type StudioTown } from '@spotto/shared';
import { CONTENT_TYPES, contentTypeMeta } from '../../components/content-meta';
import { StatusPill } from '../../components/content-fields';
import { InlineErrorBanner } from '../../components/route-status';
import Shell, { useMe } from '../../components/shell';
import { api } from '../../lib/api';

type Tab = 'all' | ContentStatus;

export default function ContentPage() {
  return (
    <Shell>
      <Suspense>
        <ContentList />
      </Suspense>
    </Shell>
  );
}

function ContentList() {
  const { data: me } = useMe();
  const isAdmin = me?.role === 'admin';
  const router = useRouter();
  const author = useSearchParams().get('author') ?? '';
  const [tab, setTab] = useState<Tab>('all');
  const [type, setType] = useState('');
  const [region, setRegion] = useState('');
  const [input, setInput] = useState('');
  const [q, setQ] = useState('');
  useEffect(() => {
    const t = setTimeout(() => setQ(input.trim()), 300);
    return () => clearTimeout(t);
  }, [input]);

  const params = new URLSearchParams();
  if (tab !== 'all') params.set('status', tab);
  if (type) params.set('type', type);
  if (region) params.set('region', region);
  if (q) params.set('q', q);
  if (author) params.set('author', author);
  const { data, isLoading, isError } = useQuery<StudioContentList>({
    queryKey: ['studio-content', params.toString()],
    queryFn: () => api.get<StudioContentList>(`/studio/content?${params}`),
  });
  const { data: towns = [] } = useQuery<StudioTown[]>({
    queryKey: ['studio-regions'],
    queryFn: () => api.get<StudioTown[]>('/studio/regions'),
    staleTime: 10 * 60_000,
  });

  const counts = data?.counts ?? {};
  const total = Object.values(counts).reduce((a, b) => a + (b ?? 0), 0);
  const tabs: { id: Tab; label: string; count: number }[] = [
    { id: 'all', label: 'All', count: total },
    { id: 'in_review', label: isAdmin ? 'To review' : 'Waiting for review', count: counts.in_review ?? 0 },
    { id: 'published', label: 'Live', count: counts.published ?? 0 },
    { id: 'draft', label: 'Drafts', count: counts.draft ?? 0 },
    { id: 'rejected', label: 'Sent back', count: counts.rejected ?? 0 },
    { id: 'archived', label: 'Taken down', count: counts.archived ?? 0 },
  ];
  const items = data?.items ?? [];

  return (
    <>
      {!isAdmin && total === 0 && !isLoading && (
        <div className="panel p-5 mb-5">
          <p className="font-bold text-[15px]">Welcome — you can write for Spotto travellers here.</p>
          <p className="text-[13px] text-muted font-semibold mt-1 max-w-[720px]">
            Start a post with <span className="font-bold">New post</span>. Save it as a draft as often as you like, then send it for review.
            An admin checks it and puts it live in the Spotto app (blog posts also appear on the website), or sends it back with a note.
          </p>
        </div>
      )}

      <div className="flex flex-wrap items-center gap-2 mb-4">
        <div className="flex flex-wrap gap-1 rounded-xl p-1" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
          {tabs.map((t) => (
            <button key={t.id} onClick={() => setTab(t.id)}
              className="rounded-lg px-3 py-1.5 text-[12.5px] font-bold transition-colors"
              style={{ background: tab === t.id ? 'var(--accent)' : 'transparent', color: tab === t.id ? '#fff' : 'var(--ink-soft)' }}>
              {t.label}
              {t.count > 0 && <span className="ml-1.5 opacity-75">{t.count}</span>}
            </button>
          ))}
        </div>
        <div className="ml-auto">
          <NewPostButton onPick={(t) => router.push(`/content/edit?type=${t}`)} />
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-2 mb-5">
        <label className="flex items-center gap-2 input py-2 !w-[280px]">
          <Search size={14} className="text-muted shrink-0" />
          <input value={input} onChange={(e) => setInput(e.target.value)} placeholder="Search titles"
            className="bg-transparent outline-none w-full text-[13px]" />
        </label>
        <select className="input !w-auto py-2" value={type} onChange={(e) => setType(e.target.value)} aria-label="Type">
          <option value="">Every type</option>
          {CONTENT_TYPES.map((t) => <option key={t.type} value={t.type}>{t.label}</option>)}
        </select>
        <select className="input !w-auto py-2" value={region} onChange={(e) => setRegion(e.target.value)} aria-label="Town">
          <option value="">Every town</option>
          {towns.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
        </select>
        {author && (
          <Link href="/content" className="btn py-1.5 px-3 text-[12px] flex items-center gap-1.5">
            One author only <X size={12} />
          </Link>
        )}
      </div>

      {isError && <InlineErrorBanner message="Couldn't load posts — try refreshing." />}
      {isLoading && <p className="text-muted font-semibold text-[13px] py-10 text-center">Loading…</p>}
      {!isLoading && items.length === 0 && (
        <div className="panel p-10 text-center">
          <p className="font-bold text-[14px]">No posts here yet.</p>
          <p className="text-muted font-semibold text-[12.5px] mt-1">
            {tab === 'in_review' && isAdmin ? 'Nothing is waiting for review.' : 'Use New post to write one.'}
          </p>
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        {items.map((item) => <ContentCard key={item.id} item={item} showAuthor={isAdmin} />)}
      </div>
    </>
  );
}

function NewPostButton({ onPick }: { onPick: (type: string) => void }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', onDoc);
    return () => document.removeEventListener('mousedown', onDoc);
  }, []);
  return (
    <div className="relative" ref={ref}>
      <button className="btn btn-primary flex items-center gap-2" onClick={() => setOpen((o) => !o)}>
        <Plus size={15} /> New post
      </button>
      {open && (
        <div className="absolute right-0 mt-2 w-[320px] rounded-xl overflow-hidden z-30"
          style={{ background: 'var(--surface)', border: '1px solid var(--line)', boxShadow: '0 24px 50px -20px rgba(0,0,0,0.35)' }}>
          {CONTENT_TYPES.map((t, i) => {
            const Icon = t.icon;
            return (
              <button key={t.type} onClick={() => onPick(t.type)}
                className="w-full flex items-start gap-3 px-4 py-3 text-left hover:bg-[color:var(--bg)]"
                style={{ borderTop: i ? '1px solid var(--line)' : 'none' }}>
                <span className="w-8 h-8 rounded-lg flex items-center justify-center shrink-0" style={{ background: `${t.color}1a`, color: t.color }}>
                  <Icon size={16} />
                </span>
                <span>
                  <span className="block font-bold text-[13px]">{t.label}</span>
                  <span className="block text-[11.5px] text-muted font-semibold">{t.hint}</span>
                </span>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}

function ContentCard({ item, showAuthor }: { item: ContentItem; showAuthor: boolean }) {
  const meta = contentTypeMeta(item.type);
  const Icon = meta.icon;
  return (
    <Link href={`/content/edit?id=${encodeURIComponent(item.id)}`} className="panel overflow-hidden flex flex-col hover:border-[color:var(--accent)] transition-colors">
      <div className="h-[140px] relative flex items-center justify-center" style={{ background: `${meta.color}14` }}>
        {item.coverUrl ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={resolveMediaUrl(item.coverUrl)} alt="" className="absolute inset-0 w-full h-full object-cover" />
        ) : (
          <Icon size={34} style={{ color: meta.color, opacity: 0.6 }} />
        )}
        <span className="absolute left-3 top-3 flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-bold"
          style={{ background: 'var(--surface)', color: meta.color }}>
          <Icon size={12} /> {meta.label}
        </span>
      </div>
      <div className="p-4 flex-1 flex flex-col">
        <div className="flex items-start justify-between gap-2">
          <h3 className="font-bold text-[14.5px] leading-snug line-clamp-2">{item.title || 'Untitled'}</h3>
          <StatusPill status={item.status} />
        </div>
        {item.summary && <p className="text-[12.5px] text-muted font-medium mt-1 line-clamp-2">{item.summary}</p>}
        {item.status === 'rejected' && item.reviewNote && (
          <p className="text-[12px] font-semibold mt-2 rounded-lg px-2.5 py-1.5" style={{ background: 'rgba(225,29,72,0.08)', color: '#e11d48' }}>
            {item.reviewNote}
          </p>
        )}
        <div className="mt-auto pt-3 flex flex-wrap items-center gap-x-3 gap-y-1 text-[11.5px] text-muted font-semibold">
          {item.regionName && <span className="flex items-center gap-1"><MapPin size={11} /> {item.regionName}</span>}
          {showAuthor && <span>{item.authorName || 'Unknown author'}</span>}
          <span>Edited {timeAgo(item.updatedAt)}</span>
          {item.status === 'published' && (
            <span className="flex items-center gap-1 ml-auto">
              <Eye size={11} /> {item.views.toLocaleString()}
              {item.type === 'audio' && <><Headphones size={11} className="ml-1.5" /> {item.plays.toLocaleString()}</>}
            </span>
          )}
        </div>
      </div>
    </Link>
  );
}
