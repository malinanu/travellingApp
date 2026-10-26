'use client';

// One travel post in the Content editor. ?id=<post> opens a post; ?type=<type>
// starts a new one, created on its first save. Publishers save drafts and
// send them for review; admins publish (their own posts straight away), send
// a publisher's post back with a note, or take a live post down.

import { useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, CheckCircle2, Eye, Headphones, ImagePlus, Loader2, Send, Trash2, Upload } from 'lucide-react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { Suspense, useEffect, useRef, useState, type ReactNode } from 'react';

import {
  resolveMediaUrl, timeAgo, uploadMedia, validateFile,
  type ContentChapter, type ContentItem, type ContentType, type ItineraryDay, type StudioTown,
} from '@spotto/shared';
import { ChaptersEditor, Field, ItineraryEditor, LinkedPlaces, MarkdownEditor, StatusPill } from '../../../components/content-fields';
import { CONTENT_TYPES, LANGUAGES, contentTypeMeta, formatDuration } from '../../../components/content-meta';
import { InlineErrorBanner } from '../../../components/route-status';
import Shell, { useMe } from '../../../components/shell';
import { api } from '../../../lib/api';

const RED = '#e11d48';
const GREEN = '#059669';

export default function ContentEditPage() {
  return (
    <Shell>
      <Suspense>
        <Editor />
      </Suspense>
    </Shell>
  );
}

type Form = {
  type: ContentType;
  title: string;
  summary: string;
  body: string;
  audioSeconds: number;
  chapters: ContentChapter[];
  placeIds: string[];
  days: ItineraryDay[];
  /** Comma-separated while editing. */
  tags: string;
  regionId: string;
  language: string;
};

const emptyForm = (type: ContentType): Form => ({
  type, title: '', summary: '', body: '', audioSeconds: 0, chapters: [], placeIds: [], days: [], tags: '', regionId: '', language: 'en',
});

const toForm = (c: ContentItem): Form => ({
  type: c.type, title: c.title, summary: c.summary, body: c.body ?? '', audioSeconds: c.audioSeconds,
  chapters: c.chapters, placeIds: c.placeIds, days: c.days, tags: c.tags.join(', '), regionId: c.regionId ?? '', language: c.language,
});

const toInput = (f: Form) => ({ ...f, tags: f.tags.split(',').map((t) => t.trim()).filter(Boolean) });

const BODY_LABEL: Record<ContentType, { label: string; hint: string }> = {
  audio: { label: 'Description', hint: 'What listeners will hear, shown under the player.' },
  blog: { label: 'Post', hint: '' },
  guide: { label: 'Guide', hint: 'Use ## headings for each section.' },
  itinerary: { label: 'Introduction', hint: 'Who this plan suits, how to get there, what to bring.' },
  tip: { label: 'Tip', hint: 'Keep it short: a few sentences.' },
};

function readAudioDuration(file: File): Promise<number | null> {
  return new Promise((resolve) => {
    const url = URL.createObjectURL(file);
    const audio = new Audio();
    audio.preload = 'metadata';
    const done = (v: number | null) => {
      URL.revokeObjectURL(url);
      resolve(v);
    };
    audio.onloadedmetadata = () => done(Number.isFinite(audio.duration) ? audio.duration : null);
    audio.onerror = () => done(null);
    audio.src = url;
  });
}

function Editor() {
  const params = useSearchParams();
  const router = useRouter();
  const qc = useQueryClient();
  const { data: me } = useMe();
  const isAdmin = me?.role === 'admin';
  const [id, setId] = useState<string | null>(params.get('id'));
  const startType = (CONTENT_TYPES.find((t) => t.type === params.get('type'))?.type ?? 'blog') as ContentType;

  const { data: item, isError } = useQuery<ContentItem>({
    queryKey: ['studio-item', id],
    queryFn: () => api.get<ContentItem>(`/studio/content/${encodeURIComponent(id ?? '')}`),
    enabled: !!id,
  });
  const { data: towns = [] } = useQuery<StudioTown[]>({
    queryKey: ['studio-regions'],
    queryFn: () => api.get<StudioTown[]>('/studio/regions'),
    staleTime: 10 * 60_000,
  });

  const [form, setForm] = useState<Form | null>(id ? null : emptyForm(startType));
  const [saved, setSaved] = useState(() => (id ? '' : JSON.stringify(emptyForm(startType))));
  useEffect(() => {
    if (item && !form) {
      const f = toForm(item);
      setForm(f);
      setSaved(JSON.stringify(f));
    }
  }, [item, form]);

  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [rejecting, setRejecting] = useState(false);
  const [note, setNote] = useState('');
  // The post's id as of the latest save. A handler that saves twice (an
  // upload to a new post) must not create it twice while `id` is still the
  // value from before the first save.
  const idRef = useRef(id);

  if (isError) return <InlineErrorBanner message="Couldn't open this post. It may have been deleted." />;
  if (!form || (id && !item)) return <p className="text-muted font-semibold text-[13px] py-10 text-center">Loading…</p>;

  const status = item?.status ?? 'draft';
  const isOwn = !item || item.authorId === me?.id;
  const editable = isAdmin || status === 'draft' || status === 'rejected' || status === 'archived';
  const dirty = JSON.stringify(form) !== saved;
  const meta = contentTypeMeta(form.type);
  const set = (patch: Partial<Form>) => setForm((f) => (f ? { ...f, ...patch } : f));

  const applyItem = (it: ContentItem) => {
    qc.setQueryData(['studio-item', it.id], it);
    const f = toForm(it);
    setForm(f);
    setSaved(JSON.stringify(f));
    qc.invalidateQueries({ queryKey: ['studio-content'] });
  };

  const save = async (override?: Form): Promise<ContentItem> => {
    const body = toInput(override ?? form);
    const current = idRef.current;
    const it = current
      ? await api.put<ContentItem>(`/studio/content/${encodeURIComponent(current)}`, body)
      : await api.post<ContentItem>('/studio/content', body);
    if (!current) {
      idRef.current = it.id;
      setId(it.id);
      window.history.replaceState(null, '', `/content/edit?id=${encodeURIComponent(it.id)}`);
    }
    applyItem(it);
    return it;
  };
  const ensureId = async () => idRef.current ?? (await save()).id;

  const run = async (label: string, fn: () => Promise<string | void>) => {
    setBusy(label);
    setError(null);
    setNotice(null);
    try {
      const msg = await fn();
      if (msg) setNotice(msg);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Something went wrong. Please try again.');
    } finally {
      setBusy(null);
    }
  };

  const transition = (label: string, path: string, body?: unknown, done?: string) =>
    run(label, async () => {
      const pid = dirty || !idRef.current ? (await save()).id : idRef.current;
      applyItem(await api.post<ContentItem>(path.replace(':id', encodeURIComponent(pid)), body));
      return done;
    });

  const uploadCover = (file: File) =>
    run('cover', async () => {
      const bad = validateFile(file, 'image');
      if (bad) throw new Error(bad);
      const pid = await ensureId();
      await uploadMedia(api, file, { ownerType: 'content', ownerId: pid, role: 'cover' });
      await qc.invalidateQueries({ queryKey: ['studio-item', pid] });
    });

  const uploadAudio = (file: File) =>
    run('audio', async () => {
      const bad = validateFile(file, 'audio');
      if (bad) throw new Error(bad);
      const seconds = await readAudioDuration(file);
      const pid = await ensureId();
      await uploadMedia(api, file, { ownerType: 'content', ownerId: pid, role: 'audio' });
      const next = { ...form, audioSeconds: seconds ? Math.round(seconds) : form.audioSeconds };
      await save(next);
      await qc.invalidateQueries({ queryKey: ['studio-item', pid] });
      return 'Audio uploaded.';
    });

  const uploadInline = async (file: File) => {
    const bad = validateFile(file, 'image');
    if (bad) throw new Error(bad);
    const pid = await ensureId();
    const m = await uploadMedia(api, file, { ownerType: 'content', ownerId: pid, role: 'inline' });
    return m.url;
  };

  const remove = () =>
    run('delete', async () => {
      if (!id) {
        router.push('/content');
        return;
      }
      if (!window.confirm('Delete this post for good? Its files are deleted too.')) return;
      await api.del(`/studio/content/${encodeURIComponent(id)}`);
      qc.invalidateQueries({ queryKey: ['studio-content'] });
      router.push('/content');
    });

  const canPublish = isAdmin && status !== 'published' && (isOwn || status === 'in_review' || status === 'archived');
  const bodyLabel = BODY_LABEL[form.type];

  return (
    <>
      <div className="flex flex-wrap items-center gap-2.5 mb-4">
        <Link href="/content" className="btn py-1.5 px-3 text-[12px] flex items-center gap-1.5"><ArrowLeft size={13} /> All posts</Link>
        <span className="flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11.5px] font-bold" style={{ background: `${meta.color}1a`, color: meta.color }}>
          <meta.icon size={12} /> {meta.label}
        </span>
        <StatusPill status={status} />
        {item?.authorName && isAdmin && !isOwn && <span className="text-[12px] text-muted font-semibold">by {item.authorName}</span>}
        {dirty && editable && <span className="text-[12px] font-semibold" style={{ color: '#c98a00' }}>Unsaved changes</span>}
      </div>

      {status === 'rejected' && item?.reviewNote && (
        <Banner color={RED} title={isOwn ? 'An admin sent this back' : 'Sent back to the author'}>
          &ldquo;{item.reviewNote}&rdquo;{isOwn && ' Make the changes, then send it for review again.'}
        </Banner>
      )}
      {!isAdmin && status === 'in_review' && (
        <Banner color="#c98a00" title="Waiting for review">An admin is checking this post. You can edit it again if they send it back.</Banner>
      )}
      {!isAdmin && status === 'published' && (
        <Banner color={GREEN} title="Live">Travellers can read this now. Ask an admin to take it down if it needs changes.</Banner>
      )}
      {isAdmin && status === 'published' && (
        <Banner color={GREEN} title="Live">Changes you save here show to travellers straight away.</Banner>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-[1fr_330px] gap-5 items-start">
        <div className="space-y-4 min-w-0">
          <div className="panel p-5 space-y-4">
            <input value={form.title} onChange={(e) => set({ title: e.target.value })} disabled={!editable} maxLength={160}
              placeholder={`${meta.label} title`} className="w-full bg-transparent outline-none text-[22px] font-bold" />
            <Field label="Summary" hint="One or two sentences, shown in lists and at the top of the post.">
              <textarea className="input resize-y" rows={2} maxLength={400} disabled={!editable} value={form.summary}
                onChange={(e) => set({ summary: e.target.value })} />
            </Field>
          </div>

          {form.type === 'audio' && (
            <>
              <div className="panel p-5">
                <PanelTitle>Audio file</PanelTitle>
                {item?.audioUrl ? (
                  <audio controls preload="metadata" src={resolveMediaUrl(item.audioUrl)} className="w-full mb-2" />
                ) : (
                  <p className="text-[12.5px] text-muted font-semibold mb-2">No file yet. MP3 or M4A, up to 200 MB.</p>
                )}
                <div className="flex flex-wrap items-center gap-3">
                  {editable && (
                    <FilePick accept="audio/mpeg,audio/mp4,audio/x-m4a,audio/aac,.mp3,.m4a" onPick={uploadAudio} busy={busy === 'audio'}
                      label={item?.audioUrl ? 'Replace audio' : 'Upload audio'} icon={<Upload size={13} />} />
                  )}
                  {form.audioSeconds > 0 && <span className="text-[12px] text-muted font-semibold">Length {formatDuration(form.audioSeconds)}</span>}
                  {busy === 'audio' && <span className="text-[12px] text-muted font-semibold">Uploading — large files take a few minutes.</span>}
                </div>
              </div>
              <div className="panel p-5">
                <PanelTitle>Chapters</PanelTitle>
                <ChaptersEditor chapters={form.chapters} onChange={(chapters) => set({ chapters })} disabled={!editable} lengthSec={form.audioSeconds} />
              </div>
            </>
          )}

          {form.type === 'guide' && (
            <div className="panel p-5">
              <PanelTitle>Places in this guide</PanelTitle>
              <p className="text-[12px] text-muted font-semibold mb-3">Travellers can open each place in the app.</p>
              <LinkedPlaces ids={form.placeIds} onChange={(placeIds) => set({ placeIds })} disabled={!editable} />
            </div>
          )}

          {form.type === 'itinerary' && (
            <div className="panel p-5">
              <PanelTitle>Day by day</PanelTitle>
              <p className="text-[12px] text-muted font-semibold mb-3">Travellers can add this plan to their trip in the app.</p>
              <ItineraryEditor days={form.days} onChange={(days) => set({ days })} disabled={!editable} />
            </div>
          )}

          <div className="panel p-5">
            <PanelTitle>{bodyLabel.label}</PanelTitle>
            {bodyLabel.hint && <p className="text-[12px] text-muted font-semibold mb-3">{bodyLabel.hint}</p>}
            <MarkdownEditor value={form.body} onChange={(body) => set({ body })} disabled={!editable}
              onUploadImage={editable ? uploadInline : undefined} rows={form.type === 'tip' ? 6 : 18} />
          </div>
        </div>

        <div className="space-y-4 lg:sticky lg:top-0">
          <div className="panel p-5 space-y-2.5">
            <PanelTitle>{isAdmin ? 'Publish' : 'Review'}</PanelTitle>
            {editable && (
              <button className="btn w-full flex items-center justify-center gap-2" disabled={!!busy || (!dirty && !!id)}
                onClick={() => run('save', async () => { await save(); return 'Saved.'; })}>
                {busy === 'save' ? <Loader2 size={14} className="animate-spin" /> : null}
                {status === 'published' ? 'Save changes' : id ? 'Save draft' : 'Create draft'}
              </button>
            )}
            {!isAdmin && editable && (
              <button className="btn btn-primary w-full flex items-center justify-center gap-2" disabled={!!busy}
                onClick={() => transition('submit', '/studio/content/:id/submit', undefined, 'Sent for review. You’ll see here when it’s live or sent back.')}>
                <Send size={14} /> Send for review
              </button>
            )}
            {canPublish && (
              <button className="btn btn-primary w-full flex items-center justify-center gap-2" disabled={!!busy}
                onClick={() => transition('publish', '/admin/content/:id/publish', undefined, 'Live — travellers can read it now.')}>
                <CheckCircle2 size={14} /> {status === 'in_review' ? 'Approve and publish' : 'Publish'}
              </button>
            )}
            {isAdmin && status === 'in_review' && !rejecting && (
              <button className="btn w-full" disabled={!!busy} onClick={() => setRejecting(true)}>Send back with a note</button>
            )}
            {rejecting && (
              <form className="space-y-2" onSubmit={(e) => {
                e.preventDefault();
                transition('reject', '/admin/content/:id/reject', { note: note.trim() }, 'Sent back to the author.').then(() => { setRejecting(false); setNote(''); });
              }}>
                <textarea autoFocus required className="input" rows={3} value={note} onChange={(e) => setNote(e.target.value)}
                  placeholder="What should the author change?" />
                <div className="flex gap-2 justify-end">
                  <button type="button" className="btn py-1 px-3 text-[12px]" onClick={() => setRejecting(false)}>Cancel</button>
                  <button type="submit" className="btn btn-danger py-1 px-3 text-[12px]" disabled={!note.trim() || !!busy}>Send back</button>
                </div>
              </form>
            )}
            {isAdmin && status === 'published' && (
              <button className="btn w-full" disabled={!!busy}
                onClick={() => transition('unpublish', '/admin/content/:id/unpublish', undefined, 'Taken down. Travellers no longer see it.')}>
                Take down
              </button>
            )}
            {busy && busy !== 'save' && busy !== 'cover' && busy !== 'audio' && (
              <p className="text-[12px] text-muted font-semibold flex items-center gap-1.5"><Loader2 size={12} className="animate-spin" /> Working…</p>
            )}
            {notice && <p className="text-[12px] font-semibold flex items-start gap-1.5" style={{ color: GREEN }}><CheckCircle2 size={13} className="mt-[1px] shrink-0" /> {notice}</p>}
            {error && <p className="text-[12px] font-semibold" style={{ color: RED }}>{error}</p>}
            {!isAdmin && (
              <p className="text-[11.5px] text-muted font-semibold pt-1">An admin checks every post before travellers see it.</p>
            )}
          </div>

          <div className="panel p-5 space-y-3.5">
            <PanelTitle>Details</PanelTitle>
            <Field label="Type">
              <select className="input" value={form.type} disabled={!editable} onChange={(e) => set({ type: e.target.value as ContentType })}>
                {CONTENT_TYPES.map((t) => <option key={t.type} value={t.type}>{t.label}</option>)}
              </select>
            </Field>
            <Field label="Town" hint="Travellers near it see the post first.">
              <select className="input" value={form.regionId} disabled={!editable} onChange={(e) => set({ regionId: e.target.value })}>
                <option value="">All of Sri Lanka</option>
                {towns.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
              </select>
            </Field>
            <Field label="Language">
              <select className="input" value={form.language} disabled={!editable} onChange={(e) => set({ language: e.target.value })}>
                {LANGUAGES.map((l) => <option key={l.code} value={l.code}>{l.label}</option>)}
                {!LANGUAGES.some((l) => l.code === form.language) && <option value={form.language}>{form.language}</option>}
              </select>
            </Field>
            <Field label="Tags" hint="Separate with commas, e.g. food, beaches, history.">
              <input className="input" value={form.tags} disabled={!editable} onChange={(e) => set({ tags: e.target.value })} />
            </Field>
          </div>

          <div className="panel p-5">
            <PanelTitle>Cover photo</PanelTitle>
            {item?.coverUrl ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={resolveMediaUrl(item.coverUrl)} alt="" className="w-full aspect-[16/9] object-cover rounded-lg mb-3" />
            ) : (
              <div className="w-full aspect-[16/9] rounded-lg mb-3 flex items-center justify-center" style={{ background: `${meta.color}14` }}>
                <meta.icon size={30} style={{ color: meta.color, opacity: 0.55 }} />
              </div>
            )}
            {editable && (
              <FilePick accept="image/jpeg,image/png,image/webp" onPick={uploadCover} busy={busy === 'cover'}
                label={item?.coverUrl ? 'Replace photo' : 'Add a photo'} icon={<ImagePlus size={13} />} />
            )}
            <p className="text-[11.5px] text-muted font-semibold mt-2">A wide photo works best. JPEG, PNG or WebP, up to 8 MB.</p>
          </div>

          {item && (item.publishedAt || item.views > 0) && (
            <div className="panel p-5 text-[12.5px] font-semibold space-y-1.5">
              <PanelTitle>Readers</PanelTitle>
              {item.publishedAt && <p className="text-muted">First published {timeAgo(item.publishedAt)}</p>}
              <p className="flex items-center gap-1.5"><Eye size={13} className="text-muted" /> {item.views.toLocaleString()} views</p>
              {item.type === 'audio' && <p className="flex items-center gap-1.5"><Headphones size={13} className="text-muted" /> {item.plays.toLocaleString()} listens</p>}
            </div>
          )}

          {(isAdmin || editable) && (
            <button className="btn w-full flex items-center justify-center gap-2 text-[12.5px]" style={{ color: RED }} disabled={!!busy} onClick={remove}>
              <Trash2 size={13} /> {id ? 'Delete post' : 'Discard'}
            </button>
          )}
        </div>
      </div>
    </>
  );
}

function PanelTitle({ children }: { children: ReactNode }) {
  return <h2 className="font-bold text-[14px] mb-3">{children}</h2>;
}

function Banner({ color, title, children }: { color: string; title: string; children: ReactNode }) {
  return (
    <div className="rounded-xl px-4 py-3 mb-4" style={{ background: `${color}14`, border: `1px solid ${color}40` }}>
      <p className="font-bold text-[13px]" style={{ color }}>{title}</p>
      <p className="text-[12.5px] font-semibold mt-0.5">{children}</p>
    </div>
  );
}

function FilePick({ accept, onPick, busy, label, icon }: { accept: string; onPick: (f: File) => void; busy: boolean; label: string; icon: ReactNode }) {
  const input = useRef<HTMLInputElement>(null);
  return (
    <>
      <button type="button" className="btn py-1.5 px-3 text-[12px] flex items-center gap-1.5" disabled={busy} onClick={() => input.current?.click()}>
        {busy ? <Loader2 size={13} className="animate-spin" /> : icon} {busy ? 'Uploading…' : label}
      </button>
      <input ref={input} type="file" accept={accept} className="hidden"
        onChange={(e) => { const f = e.target.files?.[0]; e.target.value = ''; if (f) onPick(f); }} />
    </>
  );
}
