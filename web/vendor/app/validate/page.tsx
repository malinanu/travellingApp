'use client';

// Validate — manual SPOTTO-code entry with the QRValidator.jsx result states:
// valid / already redeemed / wrong vendor / unknown.

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { CheckCircle2, CircleSlash, QrCode, RotateCcw, XCircle } from 'lucide-react';
import { useState } from 'react';

import { timeAgo, type ValidateResponse } from '@spotto/shared';
import Shell from '../../components/shell';
import { api } from '../../lib/api';

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

  return (
    <Shell>
      <div className="max-w-xl mx-auto">
        <div className="card p-8 text-center">
          <span className="mx-auto w-14 h-14 rounded-2xl bg-accent-tint text-accent flex items-center justify-center mb-4">
            <QrCode size={28} />
          </span>
          <h2 className="text-xl font-extrabold mb-1">Validate a deal code</h2>
          <p className="text-muted font-semibold text-[14px] mb-6">
            Ask the traveller for the code on their claim screen.
          </p>

          {!result && (
            <form
              onSubmit={(e) => {
                e.preventDefault();
                if (code.trim().length > 7) validate.mutate(code.trim());
              }}
            >
              <input
                className="input text-center text-[22px] tracking-[0.2em] font-extrabold uppercase mb-4"
                value={code}
                onChange={(e) => setCode(e.target.value.toUpperCase())}
                spellCheck={false}
                autoFocus
              />
              <button className="btn-accent w-full py-3" disabled={validate.isPending}>
                {validate.isPending ? 'Checking…' : 'Validate'}
              </button>
              {validate.isError && (
                <p className="text-coral font-bold text-[13px] mt-3">
                  {validate.error instanceof Error ? validate.error.message : 'Failed'}
                </p>
              )}
            </form>
          )}

          {result && <ResultCard r={result} code={code} onReset={reset} />}
        </div>
      </div>
    </Shell>
  );
}

function ResultCard({ r, code, onReset }: { r: ValidateResponse; code: string; onReset: () => void }) {
  const base = 'rounded-2xl p-6 text-left';
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
