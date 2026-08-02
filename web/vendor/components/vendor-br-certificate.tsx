'use client';

// Business Registration certificate upload + number confirmation. The doc
// (PDF or image, media role "br_cert" — single-value, re-upload replaces)
// goes to R2 like any other upload; a background scheduler (see server's
// br_cert_scheduler.go) then reads the printed certificate number via
// Gemini and pre-fills it here, and verifies it against Sri Lanka's DRC
// eROC registry once a number is set. The vendor can always see/edit the
// number before or after that — this card polls while either step is
// still in flight and stops once resolved.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, CheckCircle2, Clock3, FileText, Loader2, ShieldCheck, Trash2, Upload, XCircle } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';

import { resolveMediaUrl, uploadMedia, useAutoSave, validateFile, type Media, type Vendor } from '@spotto/shared';
import { api } from '../lib/api';

export default function VendorBRCertificate({ vendorId }: { vendorId: string }) {
  const qc = useQueryClient();

  const { data: media = [] } = useQuery<Media[]>({
    queryKey: ['vendor-media'],
    queryFn: () => api.get<Media[]>('/vendor/media'),
  });
  const doc = media.find((m) => m.role === 'br_cert');

  // Own query on the shared 'business' cache key rather than the page-level
  // useBusiness() hook, so this card can poll on its own schedule while the
  // OCR/verify pipeline is in flight without affecting other consumers.
  const { data: business } = useQuery<Vendor>({
    queryKey: ['business'],
    queryFn: () => api.get<Vendor>('/vendor/business'),
    refetchInterval: (query) => {
      const v = query.state.data;
      const inFlight = v?.brCertOcrStatus === 'pending' || v?.brCertVerifyStatus === 'pending';
      return inFlight ? 4000 : false;
    },
  });

  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const ref = useRef<HTMLInputElement>(null);

  const upload = async (file: File | undefined) => {
    if (!file) return;
    const invalid = validateFile(file, 'doc');
    if (invalid) { setError(invalid); return; }
    setError(null);
    setBusy(true);
    try {
      await uploadMedia(api, file, { ownerType: 'vendor', ownerId: vendorId, role: 'br_cert' });
      qc.invalidateQueries({ queryKey: ['vendor-media'] });
      qc.invalidateQueries({ queryKey: ['business'] });
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Upload failed — try again.');
    } finally {
      setBusy(false);
    }
  };

  const remove = useMutation({
    mutationFn: (id: string) => api.del(`/media/${id}`),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['vendor-media'] });
      qc.invalidateQueries({ queryKey: ['business'] });
    },
  });

  const [certNumber, setCertNumber] = useState('');
  const [dirty, setDirty] = useState(false);
  useEffect(() => {
    if (business && !dirty) setCertNumber(business.brCertNumber);
  }, [business, dirty]);

  const saveCertNumber = useMutation({
    mutationFn: () => api.patch<Vendor>('/vendor/business/br-cert-number', { certNumber }),
    onSuccess: () => {
      setDirty(false);
      qc.invalidateQueries({ queryKey: ['business'] });
    },
  });
  // See useAutoSave's own doc comment for why scheduling from an effect
  // (rather than synchronously after setCertNumber) avoids a stale-state
  // save race — same pattern as the identity/profile fields on this page.
  const certAutoSave = useAutoSave({
    deps: [certNumber],
    dirty,
    onSave: () => saveCertNumber.mutate(),
  });

  return (
    <div className="card p-6">
      <div className="flex items-center gap-2 mb-1">
        <span className="w-9 h-9 rounded-xl bg-accent-tint text-accent flex items-center justify-center">
          <ShieldCheck size={18} />
        </span>
        <div>
          <p className="font-extrabold text-[15px]">Business Registration certificate</p>
          <p className="text-[12.5px] text-muted font-semibold">
            Upload your BR certificate — we&apos;ll read the certificate number and verify it with the Registrar of Companies
          </p>
        </div>
      </div>

      {error && <p className="text-[13px] font-bold text-coral mt-3">{error}</p>}

      {doc && (
        <div className="mt-3 flex items-center gap-3 rounded-xl px-3 py-2.5"
          style={{ background: 'var(--sand, #faf7f2)', border: '1px solid var(--line, #eee)' }}>
          <FileText size={18} className="text-accent shrink-0" />
          <a href={resolveMediaUrl(doc.url)} target="_blank" rel="noreferrer"
            className="flex-1 text-[13px] font-bold truncate hover:underline">
            {doc.contentType.includes('pdf') ? 'Certificate (PDF)' : 'Certificate (image)'}
          </a>
          <button onClick={() => remove.mutate(doc.id)} aria-label="Delete certificate"
            className="rounded-lg p-1.5 text-coral hover:bg-[#fde4d8] transition-colors">
            <Trash2 size={15} />
          </button>
        </div>
      )}

      <button type="button" disabled={busy} onClick={() => ref.current?.click()}
        className="mt-3 w-full rounded-xl border-2 border-dashed border-line flex items-center justify-center gap-2 py-3 text-muted hover:opacity-80 transition-opacity disabled:opacity-50">
        {busy ? <Loader2 size={18} className="animate-spin" /> : <Upload size={18} />}
        <span className="text-[13px] font-bold">{busy ? 'Uploading…' : doc ? 'Replace certificate' : 'Upload certificate (PDF or image)'}</span>
        <input ref={ref} type="file" accept="application/pdf,image/*" className="hidden"
          onChange={(e) => { upload(e.target.files?.[0]); e.target.value = ''; }} />
      </button>

      <label className="block mt-4">
        <span className="block text-[12.5px] font-extrabold mb-1.5 text-muted">Certificate number</span>
        <input
          className="input"
          placeholder="e.g. PV00334782"
          value={certNumber}
          onChange={(e) => { setCertNumber(e.target.value.toUpperCase()); setDirty(true); }}
          onBlur={certAutoSave.flush}
        />
      </label>

      {doc && (
        <StatusBadge
          ocrStatus={business?.brCertOcrStatus ?? ''}
          verifyStatus={business?.brCertVerifyStatus ?? ''}
          companyName={business?.brCertCompanyName}
          message={business?.brCertMessage}
        />
      )}
    </div>
  );
}

function StatusBadge({ ocrStatus, verifyStatus, companyName, message }: {
  ocrStatus: string;
  verifyStatus: string;
  companyName?: string;
  message?: string;
}) {
  const base = 'mt-3 flex items-start gap-2 rounded-xl px-3 py-2.5 text-[12.5px] font-bold';

  if (ocrStatus === 'pending') {
    return (
      <div className={base} style={{ background: 'var(--sand, #faf7f2)' }}>
        <Loader2 size={16} className="animate-spin shrink-0 mt-0.5" />
        <span>Reading certificate…</span>
      </div>
    );
  }
  if (verifyStatus === 'pending') {
    return (
      <div className={base} style={{ background: 'var(--sand, #faf7f2)' }}>
        <Clock3 size={16} className="shrink-0 mt-0.5" />
        <span>Pending verification</span>
      </div>
    );
  }
  if (verifyStatus === 'verified') {
    return (
      <div className={base} style={{ background: '#e9f7ef', color: '#1a7a42' }}>
        <CheckCircle2 size={16} className="shrink-0 mt-0.5" />
        <span>Verified{companyName ? ` — matches "${companyName}"` : ''}</span>
      </div>
    );
  }
  if (verifyStatus === 'invalid') {
    return (
      <div className={base + ' text-coral'} style={{ background: '#fde4d8' }}>
        <XCircle size={16} className="shrink-0 mt-0.5" />
        <span>Verification failed{message ? ` — ${message}` : ''}</span>
      </div>
    );
  }
  if (verifyStatus === 'manual_review') {
    return (
      <div className={base} style={{ background: '#fff4e0', color: '#8a5a00' }}>
        <AlertTriangle size={16} className="shrink-0 mt-0.5" />
        <span>Couldn&apos;t verify automatically — our team will review this manually.</span>
      </div>
    );
  }
  return null;
}
