// Shared frame for the public legal pages (/terms, /privacy) linked from the
// registration wizard's agreement checkbox. Deliberately outside <Shell>, so
// they open without an account.

import Link from 'next/link';

import { SpottoMark } from './shell';

export interface LegalSection {
  heading: string;
  body: React.ReactNode;
}

export default function LegalPage({ title, updated, intro, sections }: {
  title: string;
  updated: string;
  intro: React.ReactNode;
  sections: LegalSection[];
}) {
  return (
    <div className="min-h-screen bg-sand">
      <header className="border-b border-line bg-white">
        <div className="max-w-[760px] mx-auto px-4 sm:px-6 h-16 flex items-center justify-between gap-3">
          <Link href="/login" className="flex items-center gap-2.5">
            <SpottoMark size={26} />
            <span className="leading-none">
              <span className="block font-display font-extrabold text-[17px] tracking-tight text-ink">Spotto</span>
              <span className="block text-[9.5px] font-extrabold uppercase tracking-[0.2em] mt-0.5 text-muted">Vendor Portal</span>
            </span>
          </Link>
          <Link href="/login" className="text-[13.5px] font-extrabold text-accent whitespace-nowrap">Back to sign in</Link>
        </div>
      </header>
      <main className="max-w-[760px] mx-auto px-4 sm:px-6 py-10">
        <h1 className="font-display font-extrabold leading-tight text-[30px] text-ink">{title}</h1>
        <p className="text-[13px] font-bold text-muted mt-1.5">Last updated {updated}</p>
        <div className="text-[15px] font-semibold leading-relaxed text-ink-soft mt-6">{intro}</div>
        <ol className="mt-8 space-y-7">
          {sections.map((s, i) => (
            <li key={s.heading}>
              <h2 className="font-display font-extrabold text-[19px] text-ink">{i + 1}. {s.heading}</h2>
              <div className="text-[14.5px] font-semibold leading-relaxed text-ink-soft mt-2 space-y-2.5 [&_ul]:list-disc [&_ul]:pl-5 [&_ul]:space-y-1.5 [&_a]:text-accent [&_a]:underline">
                {s.body}
              </div>
            </li>
          ))}
        </ol>
      </main>
    </div>
  );
}
