'use client';

// Small pieces shared by the admin booking and money pages.

import type { ReactNode } from 'react';

export function KpiTile({ label, value, hint }: { label: string; value: ReactNode; hint?: ReactNode }) {
  return (
    <div className="panel p-4">
      <p className="text-[11.5px] font-bold uppercase tracking-wide text-muted">{label}</p>
      <div className="mt-1 text-[22px] font-bold text-ink tabular-nums">{value}</div>
      {hint && <p className="mt-0.5 text-[12px] font-medium text-muted">{hint}</p>}
    </div>
  );
}

export function Tabs<T extends string>({ value, onChange, options }: { value: T; onChange: (v: T) => void; options: { id: T; label: string }[] }) {
  return (
    <div className="flex flex-wrap gap-1.5">
      {options.map((o) => (
        <button key={o.id} onClick={() => onChange(o.id)} className="btn"
          style={{
            background: value === o.id ? 'var(--accent)' : 'var(--surface)',
            color: value === o.id ? '#fff' : 'var(--ink)',
            borderColor: value === o.id ? 'var(--accent)' : 'var(--line)',
          }}>
          {o.label}
        </button>
      ))}
    </div>
  );
}

export function PageHeader({ title, sub, actions }: { title: string; sub?: ReactNode; actions?: ReactNode }) {
  return (
    <div className="flex flex-wrap items-end justify-between gap-3 mb-4">
      <div>
        <h2 className="text-[20px] font-bold text-ink">{title}</h2>
        {sub && <p className="text-[13px] text-muted font-medium max-w-3xl">{sub}</p>}
      </div>
      {actions && <div className="flex flex-wrap gap-2">{actions}</div>}
    </div>
  );
}

/** "YYYY-MM-DD" in Sri Lanka, offset by n days. */
export function colomboDay(offsetDays = 0): string {
  return new Date(Date.now() + 330 * 60_000 + offsetDays * 86_400_000).toISOString().slice(0, 10);
}
