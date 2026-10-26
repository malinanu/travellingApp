'use client';

// Pieces of the Content editor (app/content/edit): the Markdown body with a
// preview, the place picker, audio chapters and transcript, and the itinerary
// day builder.

import { useQuery } from '@tanstack/react-query';
import { ArrowDown, ArrowUp, Bold, Clock, Heading2, ImagePlus, Italic, Link2, List, Loader2, MapPin, Play, Plus, Quote, RotateCcw, Search, Trash2 } from 'lucide-react';
import { useEffect, useRef, useState, type ReactNode, type RefObject } from 'react';

import {
  renderMarkdown, resolveMediaUrl,
  type ContentChapter, type ContentPlace, type ContentStatus, type ContentTranscript, type ItineraryDay, type ItineraryStop,
} from '@spotto/shared';
import { api } from '../lib/api';
import { STATUS_META, formatDuration, languageName, parseDuration } from './content-meta';

export function StatusPill({ status }: { status: ContentStatus }) {
  const meta = STATUS_META[status];
  return (
    <span className="text-[10.5px] font-bold uppercase tracking-wide rounded-full px-2 py-0.5 shrink-0"
      style={{ background: `${meta.color}1f`, color: meta.color }}>
      {meta.label}
    </span>
  );
}

export function Field({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <label className="block">
      <span className="block text-[12.5px] font-bold mb-1.5">{label}</span>
      {children}
      {hint && <span className="block text-[11.5px] text-muted font-semibold mt-1">{hint}</span>}
    </label>
  );
}

// ---- Markdown body ----

export function MarkdownEditor({ value, onChange, disabled, onUploadImage, rows = 18 }: {
  value: string;
  onChange: (v: string) => void;
  disabled?: boolean;
  /** Uploads a picked image and returns its URL, for ![…](url). */
  onUploadImage?: (file: File) => Promise<string>;
  rows?: number;
}) {
  const [tab, setTab] = useState<'write' | 'preview'>('write');
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const area = useRef<HTMLTextAreaElement>(null);
  const fileInput = useRef<HTMLInputElement>(null);

  // Wraps the selection (or inserts at the cursor) and keeps it selected.
  const wrap = (before: string, after = '', placeholder = '') => {
    const el = area.current;
    if (!el) return;
    const { selectionStart: a, selectionEnd: b } = el;
    const picked = value.slice(a, b) || placeholder;
    const next = value.slice(0, a) + before + picked + after + value.slice(b);
    onChange(next);
    requestAnimationFrame(() => {
      el.focus();
      el.setSelectionRange(a + before.length, a + before.length + picked.length);
    });
  };
  // Starts the current line with prefix (headings, lists, quotes).
  const linePrefix = (prefix: string) => {
    const el = area.current;
    if (!el) return;
    const start = value.lastIndexOf('\n', el.selectionStart - 1) + 1;
    onChange(value.slice(0, start) + prefix + value.slice(start));
    requestAnimationFrame(() => {
      el.focus();
      el.setSelectionRange(el.selectionStart + prefix.length, el.selectionEnd + prefix.length);
    });
  };

  const pickImage = async (file: File) => {
    if (!onUploadImage) return;
    setUploading(true);
    setError(null);
    try {
      const url = await onUploadImage(file);
      wrap(`\n![`, `](${url})\n`, file.name.replace(/\.[^.]+$/, ''));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Upload failed');
    } finally {
      setUploading(false);
    }
  };

  const tools: { icon: typeof Bold; label: string; run: () => void }[] = [
    { icon: Heading2, label: 'Heading', run: () => linePrefix('## ') },
    { icon: Bold, label: 'Bold', run: () => wrap('**', '**', 'bold text') },
    { icon: Italic, label: 'Italic', run: () => wrap('*', '*', 'italic text') },
    { icon: Link2, label: 'Link', run: () => wrap('[', '](https://)', 'link text') },
    { icon: List, label: 'List', run: () => linePrefix('- ') },
    { icon: Quote, label: 'Quote', run: () => linePrefix('> ') },
  ];

  return (
    <div className="rounded-xl overflow-hidden" style={{ border: '1px solid var(--line)' }}>
      <div className="flex items-center gap-1 px-2 py-1.5" style={{ background: 'var(--bg)', borderBottom: '1px solid var(--line)' }}>
        {(['write', 'preview'] as const).map((t) => (
          <button key={t} type="button" onClick={() => setTab(t)}
            className="rounded-md px-2.5 py-1 text-[12px] font-bold"
            style={{ background: tab === t ? 'var(--surface)' : 'transparent', color: tab === t ? 'var(--ink)' : 'var(--muted)' }}>
            {t === 'write' ? 'Write' : 'Preview'}
          </button>
        ))}
        {tab === 'write' && !disabled && (
          <div className="ml-auto flex items-center gap-0.5">
            {tools.map((t) => (
              <button key={t.label} type="button" title={t.label} aria-label={t.label} onClick={t.run}
                className="w-7 h-7 rounded-md flex items-center justify-center text-ink-soft hover:bg-[color:var(--surface)]">
                <t.icon size={14} />
              </button>
            ))}
            {onUploadImage && (
              <>
                <button type="button" title="Add a photo" aria-label="Add a photo" onClick={() => fileInput.current?.click()}
                  disabled={uploading}
                  className="w-7 h-7 rounded-md flex items-center justify-center text-ink-soft hover:bg-[color:var(--surface)]">
                  <ImagePlus size={14} />
                </button>
                <input ref={fileInput} type="file" accept="image/jpeg,image/png,image/webp" className="hidden"
                  onChange={(e) => { const f = e.target.files?.[0]; e.target.value = ''; if (f) pickImage(f); }} />
              </>
            )}
          </div>
        )}
      </div>
      {tab === 'write' ? (
        <textarea ref={area} value={value} onChange={(e) => onChange(e.target.value)} disabled={disabled} rows={rows}
          className="w-full p-3.5 outline-none text-[13.5px] leading-relaxed font-mono resize-y"
          style={{ background: 'var(--surface)', color: 'var(--ink)' }}
          placeholder={'Write here. Leave an empty line between paragraphs.\n\n## A section heading\n**bold**, *italic*, [a link](https://…), - a list item, > a quote'} />
      ) : (
        <div className="md p-4 min-h-[200px]" style={{ background: 'var(--surface)' }}
          dangerouslySetInnerHTML={{ __html: value.trim() ? renderMarkdown(value, resolveMediaUrl) : '<p style="opacity:.6">Nothing to preview yet.</p>' }} />
      )}
      {(uploading || error) && (
        <p className="px-3 py-1.5 text-[11.5px] font-semibold" style={{ color: error ? '#e11d48' : 'var(--muted)', borderTop: '1px solid var(--line)' }}>
          {error ?? 'Uploading photo…'}
        </p>
      )}
    </div>
  );
}

// ---- Places ----

/** Finds Spotto places by name; onPick gets the chosen one. */
export function PlaceSearch({ onPick, placeholder = 'Search Spotto places by name', exclude = [] }: {
  onPick: (p: ContentPlace) => void;
  placeholder?: string;
  exclude?: string[];
}) {
  const [input, setInput] = useState('');
  const [q, setQ] = useState('');
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const t = setTimeout(() => setQ(input.trim()), 250);
    return () => clearTimeout(t);
  }, [input]);
  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', onDoc);
    return () => document.removeEventListener('mousedown', onDoc);
  }, []);
  const { data: results = [], isFetching } = useQuery<ContentPlace[]>({
    queryKey: ['studio-places', q],
    queryFn: () => api.get<ContentPlace[]>(`/studio/places?q=${encodeURIComponent(q)}`),
    enabled: q.length >= 2,
  });
  const shown = results.filter((p) => !exclude.includes(p.id));

  return (
    <div className="relative" ref={ref}>
      <label className="flex items-center gap-2 input py-2">
        <Search size={14} className="text-muted shrink-0" />
        <input value={input} onChange={(e) => { setInput(e.target.value); setOpen(true); }} onFocus={() => setOpen(true)}
          placeholder={placeholder} className="bg-transparent outline-none w-full text-[13px]" />
      </label>
      {open && q.length >= 2 && (
        <div className="absolute left-0 right-0 top-full mt-1 rounded-xl overflow-hidden z-30 max-h-[280px] overflow-y-auto"
          style={{ background: 'var(--surface)', border: '1px solid var(--line)', boxShadow: '0 20px 40px -20px rgba(0,0,0,0.35)' }}>
          {shown.length === 0 && (
            <p className="px-3 py-3 text-[12.5px] text-muted font-semibold">{isFetching ? 'Searching…' : 'No places match.'}</p>
          )}
          {shown.map((p) => (
            <button key={p.id} type="button"
              onClick={() => { onPick(p); setInput(''); setQ(''); setOpen(false); }}
              className="w-full flex items-center gap-2.5 px-3 py-2 text-left hover:bg-[color:var(--bg)]">
              <MapPin size={13} className="text-muted shrink-0" />
              <span className="min-w-0">
                <span className="block text-[13px] font-bold truncate">{p.name}</span>
                {p.category && <span className="block text-[11px] text-muted font-semibold truncate">{p.category}</span>}
              </span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

/** A travel guide's linked places, in order. */
export function LinkedPlaces({ ids, onChange, disabled }: { ids: string[]; onChange: (ids: string[]) => void; disabled?: boolean }) {
  const [known, setKnown] = useState<Record<string, ContentPlace>>({});
  const missing = ids.filter((id) => !known[id]);
  const { data: resolved } = useQuery<ContentPlace[]>({
    queryKey: ['studio-places-ids', missing.join(',')],
    queryFn: () => api.get<ContentPlace[]>(`/studio/places?ids=${encodeURIComponent(missing.join(','))}`),
    enabled: missing.length > 0,
  });
  useEffect(() => {
    if (resolved?.length) setKnown((k) => ({ ...k, ...Object.fromEntries(resolved.map((p) => [p.id, p])) }));
  }, [resolved]);

  const move = (i: number, d: number) => {
    const next = [...ids];
    [next[i], next[i + d]] = [next[i + d], next[i]];
    onChange(next);
  };

  return (
    <div className="space-y-2">
      {ids.length === 0 && <p className="text-[12.5px] text-muted font-semibold">No places linked yet.</p>}
      {ids.map((id, i) => (
        <div key={id} className="flex items-center gap-2 rounded-lg px-3 py-2" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
          <MapPin size={13} className="text-muted shrink-0" />
          <span className="min-w-0 flex-1">
            <span className="block text-[13px] font-bold truncate">{known[id]?.name ?? id}</span>
            {known[id]?.category && <span className="block text-[11px] text-muted font-semibold">{known[id].category}</span>}
          </span>
          {!disabled && (
            <>
              <IconBtn label="Move up" disabled={i === 0} onClick={() => move(i, -1)}><ArrowUp size={13} /></IconBtn>
              <IconBtn label="Move down" disabled={i === ids.length - 1} onClick={() => move(i, 1)}><ArrowDown size={13} /></IconBtn>
              <IconBtn label="Remove" onClick={() => onChange(ids.filter((x) => x !== id))}><Trash2 size={13} /></IconBtn>
            </>
          )}
        </div>
      ))}
      {!disabled && (
        <PlaceSearch exclude={ids} onPick={(p) => { setKnown((k) => ({ ...k, [p.id]: p })); onChange([...ids, p.id]); }}
          placeholder="Add a place: search Spotto places by name" />
      )}
    </div>
  );
}

function IconBtn({ label, onClick, disabled, children }: { label: string; onClick: () => void; disabled?: boolean; children: ReactNode }) {
  return (
    <button type="button" title={label} aria-label={label} onClick={onClick} disabled={disabled}
      className="w-7 h-7 rounded-md flex items-center justify-center text-ink-soft hover:bg-[color:var(--surface)] disabled:opacity-30 shrink-0">
      {children}
    </button>
  );
}

// ---- Audio chapters ----

/** Chapters in the order listeners hear them. */
const byStart = (cs: ContentChapter[]) => [...cs].sort((a, b) => a.startSec - b.startSec);

/** An audio guide's chapters. Each runs until the next one starts and can
 * have a photo of the spot it describes, which listeners see while it plays.
 * With the editor's player, "Add chapter here" and "Set to now" take the
 * time from where it's paused. */
export function ChaptersEditor({ chapters, onChange, disabled, lengthSec, audio, onUploadPhoto }: {
  chapters: ContentChapter[];
  onChange: (c: ContentChapter[]) => void;
  disabled?: boolean;
  lengthSec: number;
  /** The editor's player, once the audio is uploaded. */
  audio?: RefObject<HTMLAudioElement | null>;
  /** Uploads a chapter photo and returns its link. */
  onUploadPhoto?: (file: File) => Promise<string>;
}) {
  const [draft, setDraft] = useState<Record<number, string>>({});
  const [uploading, setUploading] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const fileInput = useRef<HTMLInputElement>(null);
  const list = useRef<HTMLDivElement>(null);
  const pickFor = useRef<number | null>(null);
  // An upload finishes after the user may have edited other chapters.
  const latest = useRef(chapters);
  latest.current = chapters;

  const set = (i: number, patch: Partial<ContentChapter>) => onChange(chapters.map((c, j) => (j === i ? { ...c, ...patch } : c)));
  const now = () => Math.floor(audio?.current?.currentTime ?? 0);
  const focusTitle = (startSec: number) =>
    requestAnimationFrame(() => list.current?.querySelector<HTMLInputElement>(`[data-start="${startSec}"]`)?.focus());
  const addAt = (startSec: number) => {
    if (!chapters.some((c) => c.startSec === startSec)) onChange(byStart([...chapters, { title: '', startSec }]));
    focusTitle(startSec);
  };
  const playFrom = (sec: number) => {
    const el = audio?.current;
    if (!el) return;
    el.currentTime = sec;
    el.play().catch(() => {});
  };
  const pickPhoto = (i: number) => {
    pickFor.current = i;
    setError(null);
    fileInput.current?.click();
  };
  const upload = async (file: File) => {
    const i = pickFor.current;
    const target = i === null ? undefined : latest.current[i];
    if (!target || !onUploadPhoto) return;
    setUploading(i);
    setError(null);
    try {
      const url = await onUploadPhoto(file);
      onChange(latest.current.map((c) => (c.startSec === target.startSec ? { ...c, imageUrl: url } : c)));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Upload failed');
    } finally {
      setUploading(null);
    }
  };

  return (
    <div className="space-y-2" ref={list}>
      {chapters.length === 0 && (
        <p className="text-[12.5px] text-muted font-semibold">No chapters yet. Add one where each new spot starts, so listeners can see where they are.</p>
      )}
      {chapters.map((c, i) => {
        const end = chapters[i + 1]?.startSec ?? (lengthSec > 0 ? lengthSec : null);
        return (
          <div key={i} className="flex items-center gap-2">
            <button type="button" title={c.imageUrl ? 'Change photo' : 'Add a photo of this spot'} aria-label={c.imageUrl ? 'Change photo' : 'Add a photo'}
              disabled={disabled || !onUploadPhoto || uploading !== null} onClick={() => pickPhoto(i)}
              className="w-[72px] h-[48px] rounded-lg shrink-0 overflow-hidden flex items-center justify-center"
              style={{ background: 'var(--bg)', border: `1px ${c.imageUrl ? 'solid' : 'dashed'} var(--line)` }}>
              {uploading === i ? (
                <Loader2 size={15} className="animate-spin text-muted" />
              ) : c.imageUrl ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={resolveMediaUrl(c.imageUrl)} alt="" className="w-full h-full object-cover" />
              ) : (
                <ImagePlus size={15} className="text-muted" />
              )}
            </button>
            {audio && <IconBtn label="Play from here" onClick={() => playFrom(c.startSec)}><Play size={13} /></IconBtn>}
            <input className="input py-1.5 !w-[84px] font-mono text-center" disabled={disabled} aria-label="Starts at"
              value={draft[i] ?? formatDuration(c.startSec)}
              onChange={(e) => setDraft((d) => ({ ...d, [i]: e.target.value }))}
              onBlur={() => {
                const sec = parseDuration(draft[i] ?? '');
                if (draft[i] !== undefined && sec !== null) onChange(byStart(chapters.map((x, j) => (j === i ? { ...x, startSec: sec } : x))));
                setDraft((d) => { const n = { ...d }; delete n[i]; return n; });
              }} />
            <span className="text-[11.5px] text-muted font-semibold font-mono shrink-0 w-[52px]" title="Runs until">
              → {end !== null ? formatDuration(end) : 'end'}
            </span>
            <input className="input py-1.5" disabled={disabled} placeholder="What's here, e.g. The Lion's Paw" value={c.title}
              data-start={c.startSec} onChange={(e) => set(i, { title: e.target.value })} />
            {!disabled && audio && (
              <IconBtn label="Set the start to where the player is"
                onClick={() => onChange(byStart(chapters.map((x, j) => (j === i ? { ...x, startSec: now() } : x))))}>
                <Clock size={13} />
              </IconBtn>
            )}
            {!disabled && <IconBtn label="Remove chapter" onClick={() => onChange(chapters.filter((_, j) => j !== i))}><Trash2 size={13} /></IconBtn>}
          </div>
        );
      })}
      {!disabled && (
        <div className="flex flex-wrap gap-2 pt-1">
          {audio && (
            <button type="button" className="btn btn-primary py-1.5 px-3 text-[12px] flex items-center gap-1.5" onClick={() => addAt(now())}>
              <Plus size={12} /> Add chapter here
            </button>
          )}
          <button type="button" className="btn py-1.5 px-3 text-[12px] flex items-center gap-1.5"
            onClick={() => {
              const last = chapters[chapters.length - 1]?.startSec ?? -60;
              addAt(Math.min(last + 60, Math.max(lengthSec, 0) || last + 60));
            }}>
            <Plus size={12} /> Add chapter
          </button>
        </div>
      )}
      {error && <p className="text-[12px] font-semibold" style={{ color: '#e11d48' }}>{error}</p>}
      <input ref={fileInput} type="file" accept="image/jpeg,image/png,image/webp" className="hidden"
        onChange={(e) => { const f = e.target.files?.[0]; e.target.value = ''; if (f) upload(f); }} />
      <p className="text-[11.5px] text-muted font-semibold">
        {audio
          ? 'Play the audio, pause where a new spot starts and click Add chapter here, then add its photo. Listeners see the photo while that part plays. Times are minutes:seconds, e.g. 4:30.'
          : 'Start times are minutes:seconds into the recording, e.g. 4:30. Upload the audio to add chapters from the player.'}
      </p>
    </div>
  );
}

// ---- Audio transcript ----

/** The line being heard at sec: the last one starting by then. */
export function lineAt(segments: { start: number }[], sec: number): number {
  let lo = 0;
  let hi = segments.length - 1;
  let found = -1;
  while (lo <= hi) {
    const mid = (lo + hi) >> 1;
    if (segments[mid].start <= sec + 0.05) {
      found = mid;
      lo = mid + 1;
    } else {
      hi = mid - 1;
    }
  }
  return found;
}

/** A finished transcript, to read along with the recording and correct.
 * Clicking a time plays from there and the line being played is
 * highlighted. Only the words change: the timings stay as transcribed. */
export function TranscriptEditor({ transcript, audio, disabled, onSave, onRedo }: {
  transcript: ContentTranscript;
  /** The editor's player. */
  audio: RefObject<HTMLAudioElement | null>;
  disabled?: boolean;
  onSave: (texts: string[]) => Promise<void>;
  /** Makes a new transcript from the audio. */
  onRedo?: () => void;
}) {
  const segments = transcript.segments;
  const [texts, setTexts] = useState(() => segments.map((sg) => sg.text));
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<{ error: boolean; text: string } | null>(null);
  const [active, setActive] = useState(-1);
  const list = useRef<HTMLDivElement>(null);

  // A new or saved transcript replaces the lines being edited; a refetch of
  // the same one (window focus) keeps unsaved corrections.
  const [shown, setShown] = useState(transcript.updatedAt);
  if (shown !== transcript.updatedAt) {
    setShown(transcript.updatedAt);
    setTexts(segments.map((sg) => sg.text));
  }

  useEffect(() => {
    const el = audio.current;
    if (!el) return;
    const onTime = () => setActive(lineAt(segments, el.currentTime));
    el.addEventListener('timeupdate', onTime);
    el.addEventListener('seeked', onTime);
    return () => {
      el.removeEventListener('timeupdate', onTime);
      el.removeEventListener('seeked', onTime);
    };
  }, [audio, segments]);

  // While it plays, keep the line in view, unless you're typing in the list.
  useEffect(() => {
    const box = list.current;
    const row = box?.children[active] as HTMLElement | undefined;
    if (!box || !row || audio.current?.paused || box.contains(document.activeElement)) return;
    if (row.offsetTop < box.scrollTop || row.offsetTop + row.offsetHeight > box.scrollTop + box.clientHeight) {
      box.scrollTo({ top: row.offsetTop - box.clientHeight / 3, behavior: 'smooth' });
    }
  }, [active, audio]);

  const changed = texts.some((t, i) => t !== segments[i]?.text);
  const playFrom = (sec: number) => {
    const el = audio.current;
    if (!el) return;
    el.currentTime = sec;
    el.play().catch(() => {});
  };
  const save = async () => {
    setSaving(true);
    setMessage(null);
    try {
      await onSave(texts.map((t) => t.trim()));
      setMessage({ error: false, text: 'Transcript saved.' });
    } catch (e) {
      setMessage({ error: true, text: e instanceof Error ? e.message : "Couldn't save. Please try again." });
    } finally {
      setSaving(false);
    }
  };
  const redo = () => {
    const lose = transcript.edited || changed ? ' Your corrections will be lost.' : '';
    if (window.confirm(`Make a new transcript from the audio?${lose} It takes a few minutes.`)) onRedo?.();
  };

  return (
    <div>
      <p className="text-[12px] text-muted font-semibold mb-3">
        {segments.length} lines · {languageName(transcript.language)}{transcript.edited && ' · corrected'}.{' '}
        {disabled ? 'Click a time to hear that line.' : 'Fix names and words the transcriber misheard. Click a time to hear that line.'}
      </p>
      <div ref={list} className="relative max-h-[560px] overflow-y-auto space-y-1 pr-1 -mx-1.5">
        {segments.map((sg, i) => {
          const on = i === active;
          const edited = texts[i] !== sg.text;
          return (
            <div key={i} className="flex items-start gap-2 rounded-lg px-1.5 py-1"
              style={{ background: on ? 'color-mix(in srgb, var(--accent) 12%, transparent)' : 'transparent' }}>
              <button type="button" title="Play from here" onClick={() => playFrom(sg.start)}
                className="font-mono text-[11.5px] font-bold w-[52px] text-right shrink-0 pt-[7px] hover:underline"
                style={{ color: on ? 'var(--accent)' : 'var(--muted)' }}>
                {formatDuration(sg.start)}
              </button>
              {disabled ? (
                <p className="text-[13px] leading-snug py-1.5">{sg.text}</p>
              ) : (
                <textarea className="input py-1.5 resize-none text-[13px] leading-snug" aria-label={`Line at ${formatDuration(sg.start)}`}
                  rows={Math.max(1, Math.ceil((texts[i]?.length ?? 0) / 90))} maxLength={2000} value={texts[i] ?? ''}
                  style={edited ? { borderColor: 'var(--accent)' } : undefined}
                  onChange={(e) => { const v = e.target.value; setTexts((ts) => ts.map((t, j) => (j === i ? v : t))); }} />
              )}
            </div>
          );
        })}
      </div>
      {!disabled && (
        <div className="flex flex-wrap items-center gap-2 mt-3">
          <button type="button" className="btn btn-primary py-1.5 px-3 text-[12px] flex items-center gap-1.5" disabled={!changed || saving} onClick={save}>
            {saving && <Loader2 size={12} className="animate-spin" />} Save transcript
          </button>
          {changed && (
            <button type="button" className="btn py-1.5 px-3 text-[12px]" disabled={saving}
              onClick={() => setTexts(segments.map((sg) => sg.text))}>Undo changes</button>
          )}
          {onRedo && (
            <button type="button" className="btn py-1.5 px-3 text-[12px] flex items-center gap-1.5 ml-auto" disabled={saving} onClick={redo}>
              <RotateCcw size={12} /> Run again from audio
            </button>
          )}
        </div>
      )}
      {message && (
        <p className="text-[12px] font-semibold mt-2" style={{ color: message.error ? '#e11d48' : '#059669' }}>{message.text}</p>
      )}
    </div>
  );
}

// ---- Itinerary ----

export function ItineraryEditor({ days, onChange, disabled }: { days: ItineraryDay[]; onChange: (d: ItineraryDay[]) => void; disabled?: boolean }) {
  const setDay = (i: number, patch: Partial<ItineraryDay>) => onChange(days.map((d, j) => (j === i ? { ...d, ...patch } : d)));
  const setStop = (di: number, si: number, patch: Partial<ItineraryStop>) =>
    setDay(di, { stops: days[di].stops.map((s, j) => (j === si ? { ...s, ...patch } : s)) });

  return (
    <div className="space-y-4">
      {days.length === 0 && <p className="text-[12.5px] text-muted font-semibold">Add the first day to start the plan.</p>}
      {days.map((day, di) => (
        <div key={di} className="rounded-xl p-3.5" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
          <div className="flex items-center gap-2 mb-3">
            <span className="text-[11px] font-bold uppercase tracking-wide text-muted shrink-0">Day {di + 1}</span>
            <input className="input py-1.5" disabled={disabled} placeholder="e.g. Old town and the fort" value={day.title}
              onChange={(e) => setDay(di, { title: e.target.value })} />
            {!disabled && <IconBtn label="Remove day" onClick={() => onChange(days.filter((_, j) => j !== di))}><Trash2 size={13} /></IconBtn>}
          </div>
          <div className="space-y-2">
            {day.stops.map((stop, si) => (
              <div key={si} className="rounded-lg p-2.5 space-y-2" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
                <div className="flex items-center gap-2">
                  <input type="time" className="input py-1.5 !w-[116px]" disabled={disabled} aria-label="Time" value={stop.time ?? ''}
                    onChange={(e) => setStop(di, si, { time: e.target.value })} />
                  <input className="input py-1.5" disabled={disabled} placeholder="Stop name" value={stop.name}
                    onChange={(e) => setStop(di, si, { name: e.target.value })} />
                  {stop.placeId && (
                    <span className="text-[10.5px] font-bold rounded-full px-2 py-0.5 shrink-0" title="Linked to a Spotto place"
                      style={{ background: 'color-mix(in srgb, var(--accent) 14%, transparent)', color: 'var(--accent)' }}>Spotto place</span>
                  )}
                  {!disabled && (
                    <>
                      <IconBtn label="Move up" disabled={si === 0}
                        onClick={() => { const s = [...day.stops]; [s[si - 1], s[si]] = [s[si], s[si - 1]]; setDay(di, { stops: s }); }}><ArrowUp size={13} /></IconBtn>
                      <IconBtn label="Remove stop" onClick={() => setDay(di, { stops: day.stops.filter((_, j) => j !== si) })}><Trash2 size={13} /></IconBtn>
                    </>
                  )}
                </div>
                <input className="input py-1.5" disabled={disabled} placeholder="Note (optional): what to do, tips" value={stop.note ?? ''}
                  onChange={(e) => setStop(di, si, { note: e.target.value })} />
              </div>
            ))}
            {!disabled && (
              <div className="grid grid-cols-1 md:grid-cols-[1fr_auto] gap-2">
                <PlaceSearch placeholder="Add a stop: search Spotto places"
                  onPick={(p) => setDay(di, { stops: [...day.stops, { placeId: p.id, name: p.name, lat: p.lat, lng: p.lng }] })} />
                <button type="button" className="btn py-1.5 px-3 text-[12px] flex items-center gap-1.5 justify-center"
                  onClick={() => setDay(di, { stops: [...day.stops, { name: '' }] })}>
                  <Plus size={12} /> Other stop
                </button>
              </div>
            )}
          </div>
        </div>
      ))}
      {!disabled && days.length < 30 && (
        <button type="button" className="btn py-1.5 px-3 text-[12px] flex items-center gap-1.5"
          onClick={() => onChange([...days, { title: '', stops: [] }])}>
          <Plus size={12} /> Add day {days.length + 1}
        </button>
      )}
    </div>
  );
}
