// Labels, icons and colours for travel content types and review statuses,
// shared by the Content list and editor.

import { BookOpen, CalendarRange, Headphones, Lightbulb, Map, type LucideIcon } from 'lucide-react';

import type { ContentStatus, ContentType } from '@spotto/shared';

export const CONTENT_TYPES: { type: ContentType; label: string; icon: LucideIcon; color: string; hint: string }[] = [
  { type: 'audio', label: 'Audio guide', icon: Headphones, color: '#7c3aed', hint: 'An audio guide or audiobook: an MP3 or M4A file, with chapters.' },
  { type: 'blog', label: 'Blog post', icon: BookOpen, color: '#0b6f86', hint: 'A story or article. Also shown on the website blog.' },
  { type: 'guide', label: 'Travel guide', icon: Map, color: '#059669', hint: 'A long guide to a town or theme, linking Spotto places.' },
  { type: 'itinerary', label: 'Itinerary', icon: CalendarRange, color: '#d97706', hint: 'A day-by-day plan travellers can add to their trip.' },
  { type: 'tip', label: 'Tip', icon: Lightbulb, color: '#db2777', hint: 'A short tip or piece of news.' },
];

export function contentTypeMeta(type: ContentType) {
  return CONTENT_TYPES.find((t) => t.type === type) ?? CONTENT_TYPES[1];
}

export const STATUS_META: Record<ContentStatus, { label: string; color: string }> = {
  draft: { label: 'Draft', color: '#6c8582' },
  in_review: { label: 'Waiting for review', color: '#c98a00' },
  published: { label: 'Live', color: '#059669' },
  rejected: { label: 'Sent back', color: '#e11d48' },
  archived: { label: 'Taken down', color: '#6c8582' },
};

export const LANGUAGES: { code: string; label: string }[] = [
  { code: 'en', label: 'English' },
  { code: 'si', label: 'Sinhala' },
  { code: 'ta', label: 'Tamil' },
  { code: 'de', label: 'German' },
  { code: 'fr', label: 'French' },
  { code: 'ru', label: 'Russian' },
  { code: 'zh', label: 'Chinese' },
  { code: 'hi', label: 'Hindi' },
];

/** "en" → "English"; any code the browser knows, else the code itself. */
export function languageName(code: string): string {
  const known = LANGUAGES.find((l) => l.code === code);
  if (known) return known.label;
  try {
    return new Intl.DisplayNames(['en'], { type: 'language' }).of(code) ?? code;
  } catch {
    return code;
  }
}

/** 754 → "12:34", 3723 → "1:02:03". */
export function formatDuration(sec: number): string {
  const s = Math.max(0, Math.round(sec));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const ss = String(s % 60).padStart(2, '0');
  return h ? `${h}:${String(m).padStart(2, '0')}:${ss}` : `${m}:${ss}`;
}

/** "12:34" or "1:02:03" → seconds, or null when it isn't a time. */
export function parseDuration(text: string): number | null {
  const parts = text.trim().split(':');
  if (parts.length < 1 || parts.length > 3 || parts.some((p) => !/^\d+$/.test(p))) return null;
  return parts.reduce((acc, p) => acc * 60 + Number(p), 0);
}
