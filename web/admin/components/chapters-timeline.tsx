'use client';

// An audio guide's chapters and transcript as one timeline (the content
// editor, app/content/edit). Each chapter runs until the next one starts and
// has a photo of the spot it describes and a title. The publisher reads what's
// said, starts a chapter between two lines, drops a photo on it and corrects
// misheard words; everything saves with the post. The player bar on top (with
// the chapters as blocks, search and follow-along) drives the editor's own
// <audio>.

import {
  ChevronDown, ChevronUp, Clock, ImagePlus, Loader2, MoreHorizontal, Pause, Play, Plus, RotateCcw, Scissors, Search, Trash2, X,
} from 'lucide-react';
import { useEffect, useMemo, useRef, useState, type ReactNode, type RefObject } from 'react';

import { resolveMediaUrl, type ContentChapter, type ContentTranscript } from '@spotto/shared';
import { lineAt } from './content-fields';
import { formatDuration, languageName, parseDuration } from './content-meta';

const RED = '#e11d48';
const tint = (pct: number) => `color-mix(in srgb, var(--accent) ${pct}%, transparent)`;

/** Chapters in the order listeners hear them. */
export const byStart = (cs: ContentChapter[]) => [...cs].sort((a, b) => a.startSec - b.startSec);

/** A starting title for a chapter: the first few words of its first line. */
export function suggestTitle(text: string): string {
  const t = text.replace(/[.,;:!?"“”()…]+/g, ' ').trim().split(/\s+/).filter(Boolean).slice(0, 6).join(' ').slice(0, 60);
  return t.charAt(0).toUpperCase() + t.slice(1);
}

/** Which transcript lines (by index) each chapter holds: groups[0] are the
 * lines before the first chapter, groups[c + 1] chapter c's. Both lists are
 * in time order. */
export function groupLines(starts: number[], chapters: ContentChapter[]): number[][] {
  const groups: number[][] = Array.from({ length: chapters.length + 1 }, () => []);
  let c = -1;
  starts.forEach((s, i) => {
    while (c + 1 < chapters.length && chapters[c + 1].startSec <= s + 0.05) c++;
    groups[c + 1].push(i);
  });
  return groups;
}

/** "Mirror" in "the mirror wall" → the text with each match marked. */
function highlight(text: string, q: string): ReactNode {
  if (q.length < 2) return text;
  const lower = text.toLowerCase();
  const out: ReactNode[] = [];
  let from = 0;
  for (let at = lower.indexOf(q); at >= 0; at = lower.indexOf(q, from)) {
    out.push(text.slice(from, at));
    out.push(
      <mark key={at} className="rounded-sm" style={{ background: 'color-mix(in srgb, var(--sun, #f6b81e) 50%, transparent)', color: 'inherit' }}>
        {text.slice(at, at + q.length)}
      </mark>,
    );
    from = at + q.length;
  }
  out.push(text.slice(from));
  return out;
}

export function ChaptersTimeline({
  chapters, onChapters, transcript, texts, onTexts, audio, hasAudio, lengthSec, editable, onUploadPhoto, onRedo, onSave, saving, dirty,
}: {
  chapters: ContentChapter[];
  onChapters: (c: ContentChapter[]) => void;
  /** A finished transcript, if there is one. */
  transcript?: ContentTranscript;
  /** Its lines as currently edited (one per segment). */
  texts: string[];
  onTexts: (t: string[]) => void;
  /** The editor's player. */
  audio: RefObject<HTMLAudioElement | null>;
  hasAudio: boolean;
  lengthSec: number;
  editable: boolean;
  /** Uploads a chapter photo and returns its link. */
  onUploadPhoto?: (file: File) => Promise<string>;
  /** Makes a new transcript from the audio. */
  onRedo?: () => void;
  /** Saves the post with its chapters and corrected lines. */
  onSave?: () => void;
  saving?: boolean;
  dirty?: boolean;
}) {
  const segments = useMemo(() => transcript?.segments ?? [], [transcript]);
  const groups = useMemo(() => groupLines(segments.map((s) => s.start), chapters), [segments, chapters]);

  const [activeLine, setActiveLine] = useState(-1);
  const [playing, setPlaying] = useState(false);
  const [follow, setFollow] = useState(true);
  const [query, setQuery] = useState('');
  const [matchIdx, setMatchIdx] = useState(0);
  const [allOpen, setAllOpen] = useState(false);
  const [expanded, setExpanded] = useState<Set<number>>(new Set());
  const [editing, setEditing] = useState<{ line: number; was: string } | null>(null);
  const [uploadingFor, setUploadingFor] = useState<number | null>(null);
  const [photoError, setPhotoError] = useState<{ start: number; text: string } | null>(null);
  const [focusTitle, setFocusTitle] = useState<number | null>(null);
  const list = useRef<HTMLDivElement>(null);
  const bar = useRef<HTMLDivElement>(null);
  const scrolledAt = useRef(0);
  const fileInput = useRef<HTMLInputElement>(null);
  const pickFor = useRef<number | null>(null);
  // An upload finishes after other chapters may have been edited.
  const latest = useRef(chapters);
  latest.current = chapters;

  useEffect(() => {
    const el = audio.current;
    if (!el) return;
    const onTime = () => setActiveLine(segments.length ? lineAt(segments, el.currentTime) : -1);
    const onState = () => setPlaying(!el.paused);
    el.addEventListener('timeupdate', onTime);
    el.addEventListener('seeked', onTime);
    el.addEventListener('play', onState);
    el.addEventListener('pause', onState);
    onTime();
    onState();
    return () => {
      el.removeEventListener('timeupdate', onTime);
      el.removeEventListener('seeked', onTime);
      el.removeEventListener('play', onState);
      el.removeEventListener('pause', onState);
    };
  }, [audio, segments, hasAudio]);

  const q = query.trim().toLowerCase();
  const matches = useMemo(
    () => (q.length < 2 ? [] : texts.flatMap((t, i) => (t.toLowerCase().includes(q) ? [i] : []))),
    [q, texts],
  );
  const currentMatch = matches.length ? matches[Math.min(matchIdx, matches.length - 1)] : -1;

  const reveal = (line: number, always: boolean) =>
    requestAnimationFrame(() => {
      const row = list.current?.querySelector<HTMLElement>(`[data-line="${line}"]`);
      if (!row) return;
      const r = row.getBoundingClientRect();
      const top = (bar.current?.getBoundingClientRect().bottom ?? 0) + 8; // under the stuck player bar
      // A line in view now may not be once an earlier smooth scroll ends, so retarget that one.
      const scrolling = performance.now() - scrolledAt.current < 700;
      if (always || scrolling || r.top < top || r.bottom > window.innerHeight - 40) {
        row.scrollIntoView({ block: 'center', behavior: 'smooth' });
        scrolledAt.current = performance.now();
      }
    });

  // Follow along: keep the line being heard in view, unless you're typing.
  useEffect(() => {
    if (!follow || !playing || activeLine < 0 || list.current?.contains(document.activeElement)) return;
    reveal(activeLine, false);
  }, [activeLine, follow, playing]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (currentMatch >= 0) reveal(currentMatch, true);
  }, [currentMatch]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (focusTitle === null) return;
    list.current?.querySelector<HTMLInputElement>(`[data-chapter-title="${focusTitle}"]`)?.focus();
    setFocusTitle(null);
  }, [focusTitle, chapters]);

  const now = () => Math.floor(audio.current?.currentTime ?? 0);
  const playFrom = (sec: number) => {
    const el = audio.current;
    if (!el) return;
    el.currentTime = sec;
    el.play().catch(() => {});
  };
  const addChapterAt = (sec: number) => {
    if (!chapters.some((c) => c.startSec === sec)) onChapters(byStart([...chapters, { title: '', startSec: sec }]));
    setFocusTitle(sec);
  };
  const setChapter = (start: number, patch: Partial<ContentChapter>) =>
    onChapters(byStart(chapters.map((c) => (c.startSec === start ? { ...c, ...patch } : c))));
  const removeChapter = (c: ContentChapter) => {
    if ((c.title || c.imageUrl) && !window.confirm(`Remove the chapter${c.title ? ` “${c.title}”` : ''}? Its lines join the chapter before it.`)) return;
    onChapters(chapters.filter((x) => x !== c));
  };
  const pickPhoto = (start: number) => {
    pickFor.current = start;
    fileInput.current?.click();
  };
  const uploadPhoto = async (start: number, file: File) => {
    if (!onUploadPhoto) return;
    if (!file.type.startsWith('image/')) {
      setPhotoError({ start, text: 'Drop a photo: JPEG, PNG or WebP.' });
      return;
    }
    setUploadingFor(start);
    setPhotoError(null);
    try {
      const url = await onUploadPhoto(file);
      onChapters(latest.current.map((c) => (c.startSec === start ? { ...c, imageUrl: url } : c)));
    } catch (e) {
      setPhotoError({ start, text: e instanceof Error ? e.message : 'Upload failed' });
    } finally {
      setUploadingFor(null);
    }
  };
  const finishEdit = (revert: boolean) => {
    if (editing && revert) onTexts(texts.map((t, i) => (i === editing.line ? editing.was : t)));
    setEditing(null);
  };
  const redo = () => {
    const edited = transcript?.edited || texts.some((t, i) => t !== segments[i]?.text);
    if (window.confirm(`Make a new transcript from the audio?${edited ? ' Your corrections will be lost.' : ''} It takes a few minutes.`)) onRedo?.();
  };

  const edits = texts.filter((t, i) => t !== segments[i]?.text).length;
  const groupOpen = (key: number, lines: number[]) =>
    lines.length <= 4 || allOpen || expanded.has(key) || (playing && lines.includes(activeLine)) || lines.includes(currentMatch);
  const toggleGroup = (key: number) =>
    setExpanded((s) => {
      const n = new Set(s);
      if (n.has(key)) n.delete(key);
      else n.add(key);
      return n;
    });

  const renderLines = (lines: number[], key: number, splitFirst: boolean) => {
    const open = groupOpen(key, lines);
    const shown = open ? lines : lines.slice(0, 3);
    return (
      <div className="pl-1">
        {shown.map((i, k) => (
          <LineRow key={i} index={i} start={segments[i].start} text={texts[i] ?? ''} original={segments[i].text}
            active={i === activeLine && hasAudio} match={i === currentMatch} query={q}
            editable={editable} editing={editing?.line === i}
            canSplit={editable && (k > 0 || splitFirst)}
            onSplit={() => addChapterAt(Math.floor(segments[i].start + 0.05))}
            onPlay={() => playFrom(segments[i].start)}
            onStartEdit={() => setEditing({ line: i, was: texts[i] ?? '' })}
            onChange={(v) => onTexts(texts.map((t, j) => (j === i ? v : t)))}
            onEndEdit={finishEdit} />
        ))}
        {lines.length > shown.length && (
          <button type="button" className="ml-[60px] mt-1 text-[12px] font-bold flex items-center gap-1" style={{ color: 'var(--accent)' }}
            onClick={() => toggleGroup(key)}>
            <ChevronDown size={13} /> {lines.length - shown.length} more {lines.length - shown.length === 1 ? 'line' : 'lines'}
          </button>
        )}
        {open && lines.length > 4 && !allOpen && expanded.has(key) && (
          <button type="button" className="ml-[60px] mt-1 text-[12px] font-bold flex items-center gap-1 text-muted" onClick={() => toggleGroup(key)}>
            <ChevronUp size={13} /> Show less
          </button>
        )}
      </div>
    );
  };

  return (
    <div>
      <p className="text-[12px] text-muted font-semibold -mt-1 mb-3">
        {chapters.length} {chapters.length === 1 ? 'chapter' : 'chapters'}
        {transcript && <> · {segments.length} lines · {languageName(transcript.language)}{transcript.edited && ' · corrected'}</>}
        {editable && (transcript
          ? '. Read along, start a chapter where each new spot begins (✂ between two lines), and give it a photo and a name.'
          : '. Add a chapter where each new spot begins, and give it a photo and a name.')}
      </p>

      <PlayerBar barRef={bar} audio={audio} hasAudio={hasAudio} lengthSec={lengthSec} chapters={chapters} onJump={playFrom}>
        {segments.length > 0 && (
          <label className="flex items-center gap-2 input py-1.5 !w-auto flex-1 min-w-[200px]">
            <Search size={14} className="text-muted shrink-0" />
            <input value={query} onChange={(e) => { setQuery(e.target.value); setMatchIdx(0); }} placeholder="Search what's said…"
              aria-label="Search the transcript" className="bg-transparent outline-none w-full text-[13px]"
              onKeyDown={(e) => {
                if (e.key === 'Enter' && matches.length) setMatchIdx((m) => (m + (e.shiftKey ? matches.length - 1 : 1)) % matches.length);
                if (e.key === 'Escape') setQuery('');
              }} />
            {q.length >= 2 && (
              <span className="flex items-center gap-0.5 shrink-0 text-[11.5px] font-bold text-muted">
                {matches.length ? `${Math.min(matchIdx, matches.length - 1) + 1} of ${matches.length}` : 'No match'}
                <SmallBtn label="Previous match" disabled={!matches.length} onClick={() => setMatchIdx((m) => (m + matches.length - 1) % matches.length)}><ChevronUp size={13} /></SmallBtn>
                <SmallBtn label="Next match" disabled={!matches.length} onClick={() => setMatchIdx((m) => (m + 1) % matches.length)}><ChevronDown size={13} /></SmallBtn>
                <SmallBtn label="Clear search" onClick={() => setQuery('')}><X size={13} /></SmallBtn>
              </span>
            )}
          </label>
        )}
        {segments.length > 0 && hasAudio && (
          <label className="flex items-center gap-2 text-[12px] font-bold cursor-pointer select-none shrink-0">
            <input type="checkbox" checked={follow} onChange={(e) => setFollow(e.target.checked)} style={{ accentColor: 'var(--accent)' }} />
            Follow along
          </label>
        )}
        {segments.length > 0 && (
          <button type="button" className="btn py-1 px-2.5 text-[12px] shrink-0" onClick={() => { setAllOpen((o) => !o); setExpanded(new Set()); }}>
            {allOpen ? 'Collapse all' : 'Expand all'}
          </button>
        )}
        {editable && hasAudio && (
          <button type="button" className="btn btn-primary py-1 px-2.5 text-[12px] flex items-center gap-1.5 shrink-0" onClick={() => addChapterAt(now())}>
            <Plus size={12} /> Add chapter here
          </button>
        )}
        {editable && onRedo && transcript && (
          <Menu label="More">
            {(close) => (
              <MenuItem icon={<RotateCcw size={13} />} onClick={() => { close(); redo(); }}>Run the transcript again from the audio</MenuItem>
            )}
          </Menu>
        )}
      </PlayerBar>

      <div ref={list} className="mt-3 space-y-3">
        {!transcript && (
          <p className="text-[12.5px] text-muted font-semibold rounded-lg px-3 py-2.5" style={{ background: 'var(--bg)' }}>
            {hasAudio
              ? 'The transcript appears here when it’s ready. Then you can read along and start chapters between its lines.'
              : 'Upload the audio to play it here and get its transcript.'}
          </p>
        )}

        {groups[0].length > 0 && (
          <section className="rounded-xl p-3" style={{ border: '1px dashed var(--line)' }}>
            <div className="flex flex-wrap items-center gap-2 mb-1.5">
              <span className="text-[12.5px] font-bold text-muted">{chapters.length ? 'Before the first chapter' : 'No chapters yet'}</span>
              {editable && (
                <button type="button" className="btn py-1 px-2.5 text-[12px] flex items-center gap-1.5 ml-auto"
                  onClick={() => addChapterAt(Math.floor(segments[groups[0][0]].start + 0.05))}>
                  <Scissors size={12} /> Make this a chapter
                </button>
              )}
            </div>
            {!chapters.length && editable && (
              <p className="text-[12px] text-muted font-semibold mb-2">Point between two lines and click ✂ to start a chapter there, or use Add chapter here while the audio plays.</p>
            )}
            {renderLines(groups[0], -1, false)}
          </section>
        )}

        {chapters.map((c, ci) => {
          const lines = groups[ci + 1];
          const end = chapters[ci + 1]?.startSec ?? (lengthSec > 0 ? lengthSec : null);
          const first = lines.length ? texts[lines[0]] : '';
          const playingHere = hasAudio && playing && activeLine >= 0 && lines.includes(activeLine);
          return (
            <section key={`${c.startSec}-${ci}`} className="rounded-xl p-3"
              style={{ border: `1px solid ${playingHere ? tint(55) : 'var(--line)'}`, background: playingHere ? tint(5) : 'transparent' }}>
              <ChapterHeader number={ci + 1} chapter={c} end={end} editable={editable} canUpload={!!onUploadPhoto}
                uploading={uploadingFor === c.startSec}
                error={photoError?.start === c.startSec ? photoError.text : null}
                suggestion={!c.title && first ? suggestTitle(first) : ''}
                hasAudio={hasAudio}
                onTitle={(title) => setChapter(c.startSec, { title })}
                onStart={(startSec) => setChapter(c.startSec, { startSec })}
                onPickPhoto={() => pickPhoto(c.startSec)}
                onDropPhoto={(f) => uploadPhoto(c.startSec, f)}
                onPlay={() => playFrom(c.startSec)}
                onSetToPlayer={() => setChapter(c.startSec, { startSec: now() })}
                onRemove={() => removeChapter(c)} />
              {lines.length > 0 && <div className="mt-2">{renderLines(lines, c.startSec, false)}</div>}
            </section>
          );
        })}

        {editable && !transcript && (
          <button type="button" className="btn py-1.5 px-3 text-[12px] flex items-center gap-1.5"
            onClick={() => {
              const last = chapters[chapters.length - 1]?.startSec ?? -60;
              addChapterAt(Math.min(last + 60, Math.max(lengthSec, 0) || last + 60));
            }}>
            <Plus size={12} /> Add chapter
          </button>
        )}
      </div>

      {editable && onSave && (
        <div className="flex flex-wrap items-center gap-3 mt-4 pt-4" style={{ borderTop: '1px solid var(--line)' }}>
          <button type="button" className="btn btn-primary flex items-center gap-2 disabled:opacity-50" disabled={!dirty || saving} onClick={onSave}>
            {saving && <Loader2 size={14} className="animate-spin" />} Save chapters &amp; text
          </button>
          <span className="text-[12px] font-semibold" style={{ color: dirty ? '#c98a00' : 'var(--muted)' }}>
            {dirty ? 'Unsaved changes' : 'All changes saved'}
            {edits > 0 && ` · ${edits} ${edits === 1 ? 'line' : 'lines'} corrected`}
          </span>
        </div>
      )}

      <input ref={fileInput} type="file" accept="image/jpeg,image/png,image/webp" className="hidden"
        onChange={(e) => {
          const f = e.target.files?.[0];
          e.target.value = '';
          if (f && pickFor.current !== null) uploadPhoto(pickFor.current, f);
        }} />
    </div>
  );
}

/** Play/pause, time and scrubber, the chapters as blocks, and the tools
 * passed as children. Sticks to the top while the timeline scrolls. */
function PlayerBar({ barRef, audio, hasAudio, lengthSec, chapters, onJump, children }: {
  barRef: RefObject<HTMLDivElement>;
  audio: RefObject<HTMLAudioElement | null>;
  hasAudio: boolean;
  lengthSec: number;
  chapters: ContentChapter[];
  onJump: (sec: number) => void;
  children: ReactNode;
}) {
  const [now, setNow] = useState(0);
  const [duration, setDuration] = useState(lengthSec);
  const [playing, setPlaying] = useState(false);

  useEffect(() => {
    const el = audio.current;
    if (!el) return;
    const sync = () => {
      setNow(el.currentTime);
      setPlaying(!el.paused);
      if (Number.isFinite(el.duration) && el.duration > 0) setDuration(el.duration);
    };
    const events = ['timeupdate', 'seeked', 'play', 'pause', 'loadedmetadata', 'durationchange'];
    events.forEach((ev) => el.addEventListener(ev, sync));
    sync();
    return () => events.forEach((ev) => el.removeEventListener(ev, sync));
  }, [audio, hasAudio]);

  const total = duration || lengthSec;
  const nowChapter = chapters.reduce((at, c, i) => (c.startSec <= now + 0.05 ? i : at), -1);
  const toggle = () => {
    const el = audio.current;
    if (!el) return;
    if (el.paused) el.play().catch(() => {});
    else el.pause();
  };

  return (
    <div ref={barRef} className="sticky -top-6 z-10 rounded-xl p-3 space-y-2.5"
      style={{ background: 'var(--bg)', border: '1px solid var(--line)', boxShadow: '0 12px 24px -18px rgba(0,0,0,0.5)' }}>
      {hasAudio ? (
        <div className="flex items-center gap-3">
          <button type="button" onClick={toggle} aria-label={playing ? 'Pause' : 'Play'}
            className="w-9 h-9 rounded-full flex items-center justify-center shrink-0 text-white" style={{ background: 'var(--accent)' }}>
            {playing ? <Pause size={16} /> : <Play size={16} className="ml-0.5" />}
          </button>
          <span className="font-mono text-[12px] font-bold shrink-0 tabular-nums">{formatDuration(now)} / {formatDuration(total)}</span>
          <input type="range" min={0} max={total || 0} step={0.1} value={Math.min(now, total || 0)} aria-label="Position"
            onChange={(e) => { if (audio.current) audio.current.currentTime = Number(e.target.value); }}
            className="flex-1 min-w-0" style={{ accentColor: 'var(--accent)' }} />
        </div>
      ) : null}
      {chapters.length > 0 && total > 0 && (
        <div className="relative flex h-6 rounded-md overflow-hidden" style={{ background: 'var(--surface)' }}>
          {chapters[0].startSec > 0 && <div style={{ flexBasis: `${(chapters[0].startSec / total) * 100}%` }} />}
          {chapters.map((c, i) => {
            const end = Math.min(chapters[i + 1]?.startSec ?? total, total);
            const width = Math.max(0, ((end - c.startSec) / total) * 100);
            return (
              <button key={`${c.startSec}-${i}`} type="button" title={`${i + 1}. ${c.title || 'Untitled chapter'} · ${formatDuration(c.startSec)}`}
                onClick={() => onJump(c.startSec)}
                className="h-full text-[10.5px] font-bold truncate px-1 border-r last:border-r-0"
                style={{
                  flexBasis: `${width}%`,
                  minWidth: 4,
                  borderColor: 'var(--surface)',
                  background: i === nowChapter && hasAudio ? tint(55) : tint(i % 2 ? 16 : 26),
                  color: i === nowChapter && hasAudio ? '#fff' : 'var(--ink)',
                }}>
                {i + 1}
              </button>
            );
          })}
          {hasAudio && <div className="absolute top-0 bottom-0 w-0.5 pointer-events-none" style={{ left: `${(Math.min(now, total) / total) * 100}%`, background: 'var(--ink)' }} />}
        </div>
      )}
      <div className="flex flex-wrap items-center gap-2">{children}</div>
    </div>
  );
}

function ChapterHeader({
  number, chapter, end, editable, canUpload, uploading, error, suggestion, hasAudio,
  onTitle, onStart, onPickPhoto, onDropPhoto, onPlay, onSetToPlayer, onRemove,
}: {
  number: number;
  chapter: ContentChapter;
  end: number | null;
  editable: boolean;
  canUpload: boolean;
  uploading: boolean;
  error: string | null;
  suggestion: string;
  hasAudio: boolean;
  onTitle: (t: string) => void;
  onStart: (sec: number) => void;
  onPickPhoto: () => void;
  onDropPhoto: (f: File) => void;
  onPlay: () => void;
  onSetToPlayer: () => void;
  onRemove: () => void;
}) {
  const [over, setOver] = useState(false);
  const [startDraft, setStartDraft] = useState<string | null>(null);
  const droppable = editable && canUpload;
  const commitStart = () => {
    const sec = parseDuration(startDraft ?? '');
    if (sec !== null && sec !== chapter.startSec) onStart(sec);
    setStartDraft(null);
  };

  return (
    <div className="flex items-start gap-3"
      onDragOver={(e) => { if (droppable && e.dataTransfer.types.includes('Files')) { e.preventDefault(); setOver(true); } }}
      onDragLeave={() => setOver(false)}
      onDrop={(e) => {
        if (!droppable) return;
        e.preventDefault();
        setOver(false);
        const f = e.dataTransfer.files?.[0];
        if (f) onDropPhoto(f);
      }}>
      <button type="button" disabled={!droppable || uploading} onClick={onPickPhoto}
        aria-label={chapter.imageUrl ? 'Change photo' : 'Add a photo of this spot'} title={droppable ? 'Drop a photo here, or click to choose one' : undefined}
        className="relative w-[128px] h-[84px] rounded-lg shrink-0 overflow-hidden flex flex-col items-center justify-center gap-1"
        style={{
          background: over ? tint(14) : 'var(--bg)',
          border: `${over ? 2 : 1}px ${chapter.imageUrl ? 'solid' : 'dashed'} ${over ? 'var(--accent)' : 'var(--line)'}`,
        }}>
        {chapter.imageUrl ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img src={resolveMediaUrl(chapter.imageUrl)} alt="" className="absolute inset-0 w-full h-full object-cover" />
        ) : (
          <>
            <ImagePlus size={18} className="text-muted" />
            {droppable && <span className="text-[11px] font-bold text-muted leading-tight text-center px-2">Drop a photo<br />or click</span>}
          </>
        )}
        {uploading && (
          <span className="absolute inset-0 flex items-center justify-center" style={{ background: 'rgba(0,0,0,0.45)' }}>
            <Loader2 size={18} className="animate-spin text-white" />
          </span>
        )}
      </button>

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="w-6 h-6 rounded-full shrink-0 flex items-center justify-center text-[11.5px] font-bold" style={{ background: tint(18), color: 'var(--accent)' }}>
            {number}
          </span>
          {editable ? (
            <input data-chapter-title={chapter.startSec} value={chapter.title} onChange={(e) => onTitle(e.target.value)} maxLength={120}
              placeholder="Name this spot, e.g. The Lion's Paw" aria-label="Chapter title"
              className="flex-1 min-w-0 bg-transparent outline-none text-[15px] font-bold rounded-md px-1.5 py-1 focus:bg-[color:var(--bg)]" />
          ) : (
            <span className="flex-1 min-w-0 text-[15px] font-bold truncate">{chapter.title || 'Untitled chapter'}</span>
          )}
          {(editable || hasAudio) && <Menu label="Chapter options">
            {(close) => (
              <>
                {hasAudio && <MenuItem icon={<Play size={13} />} onClick={() => { close(); onPlay(); }}>Play from here</MenuItem>}
                {editable && hasAudio && <MenuItem icon={<Clock size={13} />} onClick={() => { close(); onSetToPlayer(); }}>Start where the player is</MenuItem>}
                {editable && canUpload && <MenuItem icon={<ImagePlus size={13} />} onClick={() => { close(); onPickPhoto(); }}>{chapter.imageUrl ? 'Change photo' : 'Add a photo'}</MenuItem>}
                {editable && <MenuItem icon={<Trash2 size={13} />} danger onClick={() => { close(); onRemove(); }}>Remove chapter</MenuItem>}
              </>
            )}
          </Menu>}
        </div>

        <div className="flex flex-wrap items-center gap-x-2 gap-y-1 mt-1.5 pl-8 text-[12px] font-semibold text-muted">
          {startDraft !== null ? (
            <input autoFocus value={startDraft} onChange={(e) => setStartDraft(e.target.value)} onBlur={commitStart} aria-label="Starts at"
              onKeyDown={(e) => { if (e.key === 'Enter') commitStart(); if (e.key === 'Escape') setStartDraft(null); }}
              className="input py-0.5 px-1.5 !w-[72px] font-mono text-center text-[12px]" />
          ) : (
            <button type="button" disabled={!editable} onClick={() => setStartDraft(formatDuration(chapter.startSec))}
              title={editable ? 'Change the start' : undefined} className="font-mono rounded px-1 hover:bg-[color:var(--bg)] disabled:hover:bg-transparent">
              {formatDuration(chapter.startSec)}
            </button>
          )}
          <span className="font-mono">– {end !== null ? formatDuration(end) : 'end'}</span>
          {hasAudio && (
            <button type="button" onClick={onPlay} className="flex items-center gap-1 rounded px-1.5 py-0.5 hover:bg-[color:var(--bg)]" style={{ color: 'var(--accent)' }}>
              <Play size={11} /> Play
            </button>
          )}
        </div>

        {editable && suggestion && (
          <button type="button" onClick={() => onTitle(suggestion)}
            className="ml-8 mt-1.5 rounded-full px-2.5 py-1 text-[11.5px] font-bold max-w-full truncate" style={{ background: tint(12), color: 'var(--accent)' }}>
            Use “{suggestion}”
          </button>
        )}
        {error && <p className="ml-8 mt-1.5 text-[12px] font-semibold" style={{ color: RED }}>{error}</p>}
      </div>
    </div>
  );
}

function LineRow({
  index, start, text, original, active, match, query, editable, editing, canSplit, onSplit, onPlay, onStartEdit, onChange, onEndEdit,
}: {
  index: number;
  start: number;
  text: string;
  original: string;
  active: boolean;
  match: boolean;
  query: string;
  editable: boolean;
  editing: boolean;
  canSplit: boolean;
  onSplit: () => void;
  onPlay: () => void;
  onStartEdit: () => void;
  onChange: (v: string) => void;
  onEndEdit: (revert: boolean) => void;
}) {
  const edited = text !== original;
  return (
    <div data-line={index} className="group relative flex items-start gap-2 rounded-lg px-1.5 py-1"
      style={{ background: active ? tint(14) : match ? 'color-mix(in srgb, var(--sun, #f6b81e) 14%, transparent)' : 'transparent' }}>
      {canSplit && (
        <button type="button" onClick={onSplit}
          className="absolute left-[60px] -top-2.5 z-[1] rounded-full px-2 py-0.5 text-[11px] font-bold flex items-center gap-1 opacity-0 group-hover:opacity-100 focus:opacity-100 transition-opacity shadow-sm"
          style={{ background: 'var(--accent)', color: '#fff' }}>
          <Scissors size={11} /> Start a new chapter here
        </button>
      )}
      <button type="button" onClick={onPlay} title="Play from here"
        className="font-mono text-[11.5px] font-bold w-[52px] text-right shrink-0 pt-[3px] hover:underline"
        style={{ color: active ? 'var(--accent)' : 'var(--muted)' }}>
        {formatDuration(start)}
      </button>
      {editing ? (
        <textarea autoFocus value={text} rows={Math.max(1, Math.ceil(text.length / 90))} maxLength={2000} aria-label={`Line at ${formatDuration(start)}`}
          onChange={(e) => onChange(e.target.value)} onBlur={() => onEndEdit(false)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); onEndEdit(false); }
            if (e.key === 'Escape') { e.preventDefault(); onEndEdit(true); }
          }}
          className="input py-1 px-2 resize-none text-[13px] leading-snug flex-1" />
      ) : (
        <p onClick={editable ? onStartEdit : undefined} title={editable ? 'Click to correct' : undefined}
          className={`flex-1 text-[13.5px] leading-snug py-[2px] rounded ${editable ? 'cursor-text hover:bg-[color:var(--bg)]' : ''}`}
          style={{ fontWeight: active ? 600 : 400 }}>
          {highlight(text, query)}
          {edited && <span className="inline-block w-1.5 h-1.5 rounded-full ml-1.5 align-middle" style={{ background: 'var(--accent)' }} title="Corrected" />}
        </p>
      )}
    </div>
  );
}

/** A ⋯ button with a small dropdown; items get `close`. */
function Menu({ label, children }: { label: string; children: (close: () => void) => ReactNode }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', onDoc);
    return () => document.removeEventListener('mousedown', onDoc);
  }, [open]);
  return (
    <div className="relative shrink-0" ref={ref}>
      <SmallBtn label={label} onClick={() => setOpen((o) => !o)}><MoreHorizontal size={16} /></SmallBtn>
      {open && (
        <div className="absolute right-0 top-full mt-1 z-20 min-w-[230px] rounded-xl py-1"
          style={{ background: 'var(--surface)', border: '1px solid var(--line)', boxShadow: '0 20px 40px -20px rgba(0,0,0,0.45)' }}>
          {children(() => setOpen(false))}
        </div>
      )}
    </div>
  );
}

function MenuItem({ icon, danger, onClick, children }: { icon: ReactNode; danger?: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button type="button" onClick={onClick} className="w-full flex items-center gap-2.5 px-3 py-2 text-left text-[12.5px] font-semibold hover:bg-[color:var(--bg)]"
      style={{ color: danger ? RED : 'var(--ink)' }}>
      {icon} {children}
    </button>
  );
}

function SmallBtn({ label, onClick, disabled, children }: { label: string; onClick: () => void; disabled?: boolean; children: ReactNode }) {
  return (
    <button type="button" title={label} aria-label={label} onClick={onClick} disabled={disabled}
      className="w-7 h-7 rounded-md flex items-center justify-center text-ink-soft hover:bg-[color:var(--surface)] disabled:opacity-30">
      {children}
    </button>
  );
}
