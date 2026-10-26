// A small Markdown renderer for travel content (post bodies). It covers what
// the editor's help lists — headings, paragraphs, **bold**, *italic*,
// `code`, [links](url), ![images](url), - and 1. lists, > quotes and ---
// — and nothing else. All text is escaped before any tag is added and raw
// HTML is never passed through, so the result is safe to set as innerHTML.
// Links and images only keep http(s), mailto and app-relative (/…) URLs.
// website/site.js carries a plain-JS copy of this for the public blog;
// keep the two in step.

const escapeHtml = (s: string) =>
  s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;');

/** url is already HTML-escaped. Returns the attribute value, or null. */
function safeUrl(url: string, resolve: (u: string) => string): string | null {
  if (/^(https?:\/\/|mailto:)/i.test(url)) return url;
  if (url.startsWith('/') && !url.startsWith('//')) return resolve(url);
  return null;
}

function inline(text: string, resolve: (u: string) => string): string {
  return text
    .split(/(`[^`]+`)/)
    .map((part) => {
      if (/^`[^`]+`$/.test(part)) return `<code>${escapeHtml(part.slice(1, -1))}</code>`;
      let h = escapeHtml(part);
      h = h.replace(/!\[([^\]]*)\]\(([^)\s]+)\)/g, (m, alt: string, url: string) => {
        const src = safeUrl(url, resolve);
        return src ? `<img src="${src}" alt="${alt}" loading="lazy">` : alt;
      });
      h = h.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, (m, label: string, url: string) => {
        const href = safeUrl(url, resolve);
        return href ? `<a href="${href}" target="_blank" rel="noopener noreferrer">${label}</a>` : label;
      });
      h = h.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
      h = h.replace(/(^|[^*])\*([^*\s][^*]*)\*/g, '$1<em>$2</em>');
      h = h.replace(/(^|[\s(])_([^_\s][^_]*)_(?=[\s).,!?:;]|$)/g, '$1<em>$2</em>');
      return h;
    })
    .join('');
}

/** Renders Markdown to safe HTML. resolve turns an app-relative media path
 * (/media/raw/…) into a full URL. */
export function renderMarkdown(md: string, resolve: (u: string) => string = (u) => u): string {
  const lines = md.replace(/\r\n?/g, '\n').split('\n');
  const out: string[] = [];
  let para: string[] = [];
  let list: { tag: 'ul' | 'ol'; items: string[] } | null = null;
  let quote: string[] = [];

  const flushPara = () => {
    if (para.length) out.push(`<p>${inline(para.join(' '), resolve)}</p>`);
    para = [];
  };
  const flushList = () => {
    if (list) out.push(`<${list.tag}>${list.items.map((i) => `<li>${inline(i, resolve)}</li>`).join('')}</${list.tag}>`);
    list = null;
  };
  const flushQuote = () => {
    if (quote.length) out.push(`<blockquote>${inline(quote.join(' '), resolve)}</blockquote>`);
    quote = [];
  };
  const flushAll = () => {
    flushPara();
    flushList();
    flushQuote();
  };

  for (const raw of lines) {
    const line = raw.trimEnd();
    const t = line.trim();
    let m: RegExpMatchArray | null;
    if (!t) {
      flushAll();
    } else if ((m = t.match(/^(#{1,6})\s+(.*)$/))) {
      flushAll();
      const level = Math.min(Math.max(m[1].length, 2), 4);
      out.push(`<h${level}>${inline(m[2], resolve)}</h${level}>`);
    } else if (/^(-{3,}|\*{3,}|_{3,})$/.test(t)) {
      flushAll();
      out.push('<hr>');
    } else if ((m = t.match(/^>\s?(.*)$/))) {
      flushPara();
      flushList();
      quote.push(m[1]);
    } else if ((m = t.match(/^[-*+]\s+(.*)$/)) || (m = t.match(/^\d+[.)]\s+(.*)$/))) {
      flushPara();
      flushQuote();
      const tag = /^\d/.test(t) ? 'ol' : 'ul';
      if (!list || list.tag !== tag) {
        flushList();
        list = { tag, items: [] };
      }
      list.items.push(m[1]);
    } else if (list && /^\s{2,}/.test(raw)) {
      // An indented line continues the list item above it.
      list.items[list.items.length - 1] += ' ' + t;
    } else {
      flushList();
      flushQuote();
      para.push(t);
    }
  }
  flushAll();
  return out.join('\n');
}
