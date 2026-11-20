'use client';

// Small building blocks both web apps use on their booking and money pages.
// Styled only with the theme tokens both apps define (line, ink, ink-soft,
// muted, accent, surface), so they follow each app's look and dark mode.

import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react';

// ---- money ----

/** "LKR 4,500", or "LKR 4,500.50" when there are cents; minor units in. */
export function formatMinor(minor: number | undefined | null, currency = 'LKR'): string {
  const v = minor ?? 0;
  const negative = v < 0;
  const abs = Math.abs(v);
  const whole = Math.floor(abs / 100).toLocaleString('en-US');
  const cents = abs % 100;
  return `${negative ? '-' : ''}${currency} ${whole}${cents ? `.${String(cents).padStart(2, '0')}` : ''}`;
}

/** LKR amount typed by a person ("4,500.50") to minor units, or null. */
export function parseMajorToMinor(input: string): number | null {
  const clean = input.replace(/[,\s]/g, '').replace(/^LKR/i, '');
  if (!/^\d+(\.\d{1,2})?$/.test(clean)) return null;
  const [w, f = ''] = clean.split('.');
  return Number(w) * 100 + Number(f.padEnd(2, '0'));
}

export function Money({ minor, currency = 'LKR', signed = false, className = '' }: { minor: number | undefined | null; currency?: string; signed?: boolean; className?: string }) {
  const v = minor ?? 0;
  const tone = signed ? (v > 0 ? 'text-emerald-600' : v < 0 ? 'text-rose-600' : '') : '';
  return <span className={`tabular-nums ${tone} ${className}`}>{signed && v > 0 ? '+' : ''}{formatMinor(v, currency)}</span>;
}

// ---- status pill ----

type Tone = 'green' | 'amber' | 'red' | 'blue' | 'grey';
const TONES: Record<Tone, string> = {
  green: 'bg-emerald-500/15 text-emerald-700',
  amber: 'bg-amber-500/20 text-amber-800',
  red: 'bg-rose-500/15 text-rose-700',
  blue: 'bg-sky-500/15 text-sky-700',
  grey: 'bg-slate-500/15 text-slate-600',
};

const STATUS: Record<string, { label: string; tone: Tone }> = {
  // bookings
  awaiting_payment: { label: 'Awaiting payment', tone: 'amber' },
  confirmed: { label: 'Confirmed', tone: 'blue' },
  completed: { label: 'Checked in', tone: 'green' },
  no_show: { label: 'No-show', tone: 'red' },
  cancelled: { label: 'Cancelled', tone: 'grey' },
  expired: { label: 'Expired', tone: 'grey' },
  // payments
  not_required: { label: 'Free', tone: 'grey' },
  pending: { label: 'Unpaid', tone: 'amber' },
  paid: { label: 'Paid', tone: 'green' },
  refunded: { label: 'Refunded', tone: 'grey' },
  partially_refunded: { label: 'Part refunded', tone: 'amber' },
  failed: { label: 'Failed', tone: 'red' },
  // payouts and refunds
  draft: { label: 'Draft', tone: 'grey' },
  approved: { label: 'Approved', tone: 'blue' },
  requested: { label: 'Requested', tone: 'amber' },
  succeeded: { label: 'Refunded', tone: 'green' },
  manual: { label: 'Needs manual refund', tone: 'red' },
  manual_done: { label: 'Refunded by hand', tone: 'green' },
  scheduled: { label: 'Scheduled', tone: 'blue' },
};

export function statusLabel(status: string): string {
  return STATUS[status]?.label ?? status.replace(/_/g, ' ');
}

export function StatusPill({ status, className = '' }: { status: string; className?: string }) {
  const s = STATUS[status] ?? { label: status.replace(/_/g, ' '), tone: 'grey' as Tone };
  return (
    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-extrabold whitespace-nowrap ${TONES[s.tone]} ${className}`}>
      {s.label}
    </span>
  );
}

// ---- empty state ----

export function EmptyState({ icon, title, body, action }: { icon?: ReactNode; title: string; body?: ReactNode; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-center justify-center text-center py-12 px-6">
      {icon && <div className="mb-3 text-muted">{icon}</div>}
      <p className="font-extrabold text-[15px] text-ink">{title}</p>
      {body && <div className="mt-1 max-w-md text-[13px] font-semibold text-muted">{body}</div>}
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}

// ---- overlay pieces ----

function useEscape(open: boolean, onClose: () => void) {
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && onClose();
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, onClose]);
}

export interface ConfirmOptions {
  title: string;
  body?: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  /** danger paints the confirm button red (cancel, delete, no-show). */
  tone?: 'danger' | 'default';
  /** Ask for a typed reason; the promise then resolves to it. */
  reasonLabel?: string;
}

/** A confirm dialog driven by a promise:
 *   const [confirm, dialog] = useConfirm();
 *   if (await confirm({ title: 'Cancel booking?', tone: 'danger' })) ...
 * Render `dialog` once in the page. Resolves false (or null with a
 * reason prompt) when dismissed. */
export function useConfirm(): [(opts: ConfirmOptions) => Promise<boolean | string | null>, ReactNode] {
  const [opts, setOpts] = useState<ConfirmOptions | null>(null);
  const [reason, setReason] = useState('');
  const resolver = useRef<((v: boolean | string | null) => void) | null>(null);

  const confirm = useCallback((o: ConfirmOptions) => {
    setOpts(o);
    setReason('');
    return new Promise<boolean | string | null>((res) => {
      resolver.current = res;
    });
  }, []);
  const close = useCallback((v: boolean | string | null) => {
    resolver.current?.(v);
    resolver.current = null;
    setOpts(null);
  }, []);
  useEscape(!!opts, () => close(opts?.reasonLabel ? null : false));

  const dialog = opts ? (
    <div className="fixed inset-0 z-[80] flex items-center justify-center p-4" role="dialog" aria-modal="true">
      <div className="absolute inset-0 bg-black/40" onClick={() => close(opts.reasonLabel ? null : false)} />
      <div className="relative w-full max-w-md rounded-2xl bg-surface border border-line p-5 shadow-xl">
        <h3 className="font-extrabold text-[16px] text-ink">{opts.title}</h3>
        {opts.body && <div className="mt-2 text-[13.5px] font-semibold text-ink-soft">{opts.body}</div>}
        {opts.reasonLabel && (
          <label className="block mt-3">
            <span className="text-[12px] font-extrabold text-muted">{opts.reasonLabel}</span>
            <textarea
              autoFocus
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              rows={3}
              className="mt-1 w-full rounded-xl border border-line bg-transparent px-3 py-2 text-[13.5px] text-ink outline-none focus:border-accent"
            />
          </label>
        )}
        <div className="mt-5 flex justify-end gap-2">
          <button
            type="button"
            onClick={() => close(opts.reasonLabel ? null : false)}
            className="rounded-xl px-4 py-2 text-[13.5px] font-extrabold text-ink-soft border border-line hover:bg-black/5"
          >
            {opts.cancelLabel ?? 'Keep it'}
          </button>
          <button
            type="button"
            disabled={!!opts.reasonLabel && !reason.trim()}
            onClick={() => close(opts.reasonLabel ? reason.trim() : true)}
            className={`rounded-xl px-4 py-2 text-[13.5px] font-extrabold text-white disabled:opacity-50 ${
              opts.tone === 'danger' ? 'bg-rose-600 hover:bg-rose-700' : 'bg-accent hover:opacity-90'
            }`}
          >
            {opts.confirmLabel ?? 'Confirm'}
          </button>
        </div>
      </div>
    </div>
  ) : null;

  return [confirm, dialog];
}

/** A panel sliding in from the right for a record's details. */
export function Drawer({ open, onClose, title, subtitle, children, footer, width = 460 }: {
  open: boolean;
  onClose: () => void;
  title: ReactNode;
  subtitle?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
  width?: number;
}) {
  useEscape(open, onClose);
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-[70]" role="dialog" aria-modal="true">
      <div className="absolute inset-0 bg-black/30" onClick={onClose} />
      <aside className="absolute right-0 top-0 h-full w-full bg-surface border-l border-line shadow-2xl flex flex-col" style={{ maxWidth: width }}>
        <div className="flex items-start gap-3 px-5 py-4 border-b border-line">
          <div className="min-w-0 flex-1">
            <h3 className="font-extrabold text-[16px] text-ink truncate">{title}</h3>
            {subtitle && <div className="text-[12.5px] font-semibold text-muted mt-0.5">{subtitle}</div>}
          </div>
          <button type="button" onClick={onClose} aria-label="Close" className="rounded-lg px-2 py-1 text-muted hover:bg-black/5 text-[18px] leading-none">
            ×
          </button>
        </div>
        <div className="flex-1 overflow-y-auto px-5 py-4">{children}</div>
        {footer && <div className="px-5 py-3 border-t border-line flex flex-wrap gap-2 justify-end">{footer}</div>}
      </aside>
    </div>
  );
}

/** Label / value rows for a drawer. */
export function DetailList({ rows }: { rows: [ReactNode, ReactNode][] }) {
  return (
    <dl className="divide-y divide-[color:var(--line)]">
      {rows.map(([k, v], i) => (
        <div key={i} className="flex gap-4 py-2 text-[13px]">
          <dt className="w-36 shrink-0 font-extrabold text-muted">{k}</dt>
          <dd className="min-w-0 flex-1 font-semibold text-ink break-words">{v}</dd>
        </div>
      ))}
    </dl>
  );
}

// ---- table ----

export interface Column<T> {
  key: string;
  header: ReactNode;
  render: (row: T) => ReactNode;
  /** Value to sort by; columns without one aren't sortable. */
  sortValue?: (row: T) => string | number;
  align?: 'left' | 'right';
  className?: string;
}

export function DataTable<T>({ columns, rows, rowKey, onRowClick, empty, loading, initialSort }: {
  columns: Column<T>[];
  rows: T[];
  rowKey: (row: T) => string;
  onRowClick?: (row: T) => void;
  empty?: ReactNode;
  loading?: boolean;
  initialSort?: { key: string; dir: 'asc' | 'desc' };
}) {
  const [sort, setSort] = useState(initialSort);
  const sorted = useMemo(() => {
    const col = columns.find((c) => c.key === sort?.key);
    if (!col?.sortValue || !sort) return rows;
    const out = [...rows].sort((a, b) => {
      const x = col.sortValue!(a), y = col.sortValue!(b);
      return x < y ? -1 : x > y ? 1 : 0;
    });
    return sort.dir === 'desc' ? out.reverse() : out;
  }, [rows, columns, sort]);

  if (loading) {
    return (
      <div className="space-y-2 p-4">
        {[0, 1, 2].map((i) => <div key={i} className="h-9 rounded-lg bg-black/5 animate-pulse" />)}
      </div>
    );
  }
  if (!rows.length) return <>{empty ?? <EmptyState title="Nothing here yet" />}</>;

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-[13px]">
        <thead>
          <tr className="border-b border-line">
            {columns.map((c) => (
              <th
                key={c.key}
                className={`px-3 py-2.5 font-extrabold text-[11.5px] uppercase tracking-wide text-muted whitespace-nowrap ${c.align === 'right' ? 'text-right' : 'text-left'} ${c.sortValue ? 'cursor-pointer select-none hover:text-ink' : ''}`}
                onClick={() => c.sortValue && setSort((s) => ({ key: c.key, dir: s?.key === c.key && s.dir === 'asc' ? 'desc' : 'asc' }))}
              >
                {c.header}
                {sort?.key === c.key && <span className="ml-1">{sort.dir === 'asc' ? '↑' : '↓'}</span>}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {sorted.map((row) => (
            <tr
              key={rowKey(row)}
              onClick={onRowClick ? () => onRowClick(row) : undefined}
              className={`border-b border-line last:border-0 ${onRowClick ? 'cursor-pointer hover:bg-black/[0.03]' : ''}`}
            >
              {columns.map((c) => (
                <td key={c.key} className={`px-3 py-2.5 align-middle text-ink ${c.align === 'right' ? 'text-right' : ''} ${c.className ?? ''}`}>
                  {c.render(row)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// ---- dates ----

/** "Tue 18 Nov, 19:30" in Sri Lankan time. */
export function formatLocalDateTime(iso: string | undefined): string {
  if (!iso) return '';
  return new Date(iso).toLocaleString('en-GB', {
    timeZone: 'Asia/Colombo', weekday: 'short', day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit',
  });
}

/** "18 Nov 2026" in Sri Lankan time. */
export function formatLocalDate(iso: string | undefined): string {
  if (!iso) return '';
  return new Date(iso).toLocaleDateString('en-GB', { timeZone: 'Asia/Colombo', day: 'numeric', month: 'short', year: 'numeric' });
}
