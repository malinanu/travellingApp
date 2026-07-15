'use client';

// Validate — manual SPOTTO-code entry with the QRValidator.jsx result states:
// valid / already redeemed / wrong vendor / unknown. Port of the Validate
// mockup — a camera-viewfinder visual plus "Simulate scan" (this is a
// desktop web portal with no real device camera integration, matching the
// mockup's own "simulate" wording) alongside the manual entry form.

import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  CheckCircle2, CircleSlash, QrCode, RotateCcw, ScanLine, ShieldCheck, XCircle,
} from 'lucide-react';
import { useState } from 'react';

import { timeAgo, type ValidateResponse } from '@spotto/shared';
import Shell from '../../components/shell';
import { api } from '../../lib/api';

const DEMO_CODES = ['SPOTTO-DRIF', 'SPOTTO-X4K9', 'SPOTTO-88Q2'];

export default function ValidatePage() {
  const qc = useQueryClient();
  const [code, setCode] = useState('SPOTTO-');
  const [result, setResult] = useState<ValidateResponse | null>(null);

  const validate = useMutation({
    mutationFn: (c: string) => api.post<ValidateResponse>('/vendor/validate', { code: c }),
    onSuccess: (r) => {
      setResult(r);
      if (r.result === 'valid') qc.invalidateQueries({ queryKey: ['redemptions'] });
    },
  });

  const reset = () => {
    setResult(null);
    setCode('SPOTTO-');
  };

  const submitCode = (c: string) => {
    if (c.trim().length > 7) validate.mutate(c.trim());
  };

  return (
    <Shell>
      <div className="mb-5">
        <h1 className="text-2xl font-extrabold">Validate a redemption</h1>
        <p className="text-muted font-semibold mt-0.5">
          Scan the traveler&rsquo;s Spotto code, or type it in. Confirm in one tap.
        </p>
      </div>

      <div className="grid lg:grid-cols-[1fr_340px] gap-5 items-start">
        <div>
          <div className="rounded-2xl overflow-hidden aspect-[4/3] flex items-center justify-center relative"
            style={{ background: 'linear-gradient(160deg, #0b1f22 0%, #0e2b2e 100%)' }}>
            <div className="relative w-40 h-40 flex items-center justify-center">
              <span className="absolute top-0 left-0 w-8 h-8 border-t-2 border-l-2 border-accent rounded-tl-xl" />
              <span className="absolute top-0 right-0 w-8 h-8 border-t-2 border-r-2 border-accent rounded-tr-xl" />
              <span className="absolute bottom-0 left-0 w-8 h-8 border-b-2 border-l-2 border-accent rounded-bl-xl" />
              <span className="absolute bottom-0 right-0 w-8 h-8 border-b-2 border-r-2 border-accent rounded-br-xl" />
              <div className="text-center text-white/70">
                <QrCode size={40} className="mx-auto mb-2" />
                <p className="text-[12.5px] font-semibold">Point at traveler&rsquo;s code</p>
              </div>
            </div>
            <span className="absolute bottom-4 rounded-full bg-black/40 text-white text-[12px] font-bold px-3 py-1.5 flex items-center gap-1.5">
              <span className="w-1.5 h-1.5 rounded-full bg-[#3ec97a]" /> Camera ready
            </span>
          </div>
          <button
            className="btn-accent w-full py-3 mt-3 flex items-center justify-center gap-2"
            disabled={validate.isPending}
            onClick={() => submitCode(DEMO_CODES[Math.floor(Math.random() * DEMO_CODES.length)])}
          >
            <ScanLine size={16} /> Simulate scan
          </button>
        </div>

        <div className="space-y-4">
          {!result ? (
            <>
              <div className="card p-5">
                <h3 className="font-extrabold text-[15px] mb-1">Enter code manually</h3>
                <p className="text-[13px] text-muted font-semibold mb-3">If the camera won&rsquo;t focus.</p>
                <form
                  className="flex gap-2"
                  onSubmit={(e) => {
                    e.preventDefault();
                    submitCode(code);
                  }}
                >
                  <input
                    className="input flex-1 uppercase tracking-wide font-extrabold"
                    placeholder="SPOTTO-XXXX"
                    value={code}
                    onChange={(e) => setCode(e.target.value.toUpperCase())}
                    spellCheck={false}
                  />
                  <button className="btn-accent px-5" disabled={validate.isPending}>
                    {validate.isPending ? '…' : 'Check'}
                  </button>
                </form>
                {validate.isError && (
                  <p className="text-coral font-bold text-[13px] mt-3">
                    {validate.error instanceof Error ? validate.error.message : 'Failed'}
                  </p>
                )}
                <p className="text-[12px] text-muted font-semibold mt-4 mb-2">Try a demo code:</p>
                <div className="flex flex-wrap gap-2">
                  {DEMO_CODES.map((c) => (
                    <button key={c} onClick={() => setCode(c)}
                      className="rounded-lg bg-sand px-2.5 py-1 text-[12px] font-extrabold font-mono hover:bg-accent-tint transition-colors">
                      {c}
                    </button>
                  ))}
                </div>
              </div>

              <div className="rounded-2xl bg-accent-tint p-4 flex gap-3">
                <ShieldCheck size={18} className="text-accent shrink-0 mt-0.5" />
                <div>
                  <p className="font-extrabold text-[13.5px] text-accent">Fraud-safe</p>
                  <p className="text-[12.5px] font-semibold text-ink/80 mt-0.5">
                    Each code can only be redeemed once. Spotto auto-expires it the moment you confirm.
                  </p>
                </div>
              </div>
            </>
          ) : (
            <ResultCard r={result} code={code} onReset={reset} />
          )}
        </div>
      </div>
    </Shell>
  );
}

function ResultCard({ r, code, onReset }: { r: ValidateResponse; code: string; onReset: () => void }) {
  const base = 'rounded-2xl p-6 text-left card';
  return (
    <div>
      {r.result === 'valid' && r.claim && (
        <div className={`${base} bg-accent-tint border border-accent/30`}>
          <p className="flex items-center gap-2 text-accent font-extrabold text-lg">
            <CheckCircle2 size={22} /> Valid — redeemed just now
          </p>
          <p className="mt-3 font-extrabold text-[15px]">
            {r.claim.travellerName || 'Traveller'}
            {r.claim.travellerCountry ? ` · ${r.claim.travellerCountry}` : ''}
          </p>
          {r.claim.travellerStyle && (
            <p className="text-[13px] text-muted font-bold capitalize">{r.claim.travellerStyle}</p>
          )}
          <p className="mt-2 text-[14px] font-semibold">{r.claim.offerLabel}</p>
          <p className="text-[12.5px] text-muted font-bold">{r.claim.code}</p>
        </div>
      )}
      {r.result === 'already_redeemed' && (
        <div className={`${base} bg-[#fdf0cd] border border-sun/50`}>
          <p className="flex items-center gap-2 text-[#a97b00] font-extrabold text-lg">
            <CircleSlash size={22} /> Already redeemed
          </p>
          <p className="mt-2 text-[14px] font-semibold">
            {code} was used {timeAgo(r.redeemedAt ?? r.claim?.redeemedAt)}. Each code works once.
          </p>
        </div>
      )}
      {r.result === 'wrong_vendor' && (
        <div className={`${base} bg-[#fde4d8] border border-coral/40`}>
          <p className="flex items-center gap-2 text-coral font-extrabold text-lg">
            <XCircle size={22} /> Not your code
          </p>
          <p className="mt-2 text-[14px] font-semibold">
            This code belongs to another business&rsquo;s offer.
          </p>
        </div>
      )}
      {r.result === 'unknown' && (
        <div className={`${base} bg-sand border border-line`}>
          <p className="flex items-center gap-2 font-extrabold text-lg">
            <XCircle size={22} /> Invalid code
          </p>
          <p className="mt-2 text-[14px] font-semibold text-muted">
            {code} doesn&rsquo;t match any claimed deal. Check for typos.
          </p>
        </div>
      )}
      <button onClick={onReset} className="btn-accent w-full py-3 mt-4 flex items-center justify-center gap-2">
        <RotateCcw size={16} /> Validate another
      </button>
    </div>
  );
}
