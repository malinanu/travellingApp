'use client';

// Verification-document uploads for the vendor Business Profile. Docs
// (PDF or image scans) go to R2 with role "doc"; uploading any flips the
// vendor's docs flag, which the admin approval queue surfaces.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { FileText, Loader2, ShieldCheck, Trash2, Upload } from 'lucide-react';
import { useRef, useState } from 'react';

import { uploadMedia, validateFile, type Media } from '@spotto/shared';
import { api } from '../lib/api';

export default function VendorDocs({ vendorId }: { vendorId: string }) {
  const qc = useQueryClient();
  const { data: media = [] } = useQuery<Media[]>({
    queryKey: ['vendor-media'],
    queryFn: () => api.get<Media[]>('/vendor/media'),
  });
  const docs = media.filter((m) => m.role === 'doc');

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
      await uploadMedia(api, file, { ownerType: 'vendor', ownerId: vendorId, role: 'doc' });
      qc.invalidateQueries({ queryKey: ['vendor-media'] });
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Upload failed — try again.');
    } finally {
      setBusy(false);
    }
  };

  const remove = useMutation({
    mutationFn: (id: string) => api.del(`/media/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['vendor-media'] }),
  });

  return (
    <div className="card p-6">
      <div className="flex items-center gap-2 mb-1">
        <span className="w-9 h-9 rounded-xl bg-accent-tint text-accent flex items-center justify-center">
          <ShieldCheck size={18} />
        </span>
        <div>
          <p className="font-extrabold text-[15px]">Verification documents</p>
          <p className="text-[12.5px] text-muted font-semibold">
            Business registration, licence, or ID — helps us verify your listing
          </p>
        </div>
      </div>

      {error && <p className="text-[13px] font-bold text-coral mt-3">{error}</p>}

      <div className="mt-3 space-y-2">
        {docs.map((d) => (
          <div key={d.id} className="flex items-center gap-3 rounded-xl px-3 py-2.5"
            style={{ background: 'var(--sand, #faf7f2)', border: '1px solid var(--line, #eee)' }}>
            <FileText size={18} className="text-accent shrink-0" />
            <a href={d.url} target="_blank" rel="noreferrer"
              className="flex-1 text-[13px] font-bold truncate hover:underline">
              {d.contentType.includes('pdf') ? 'Document (PDF)' : 'Document (image)'}
            </a>
            <button onClick={() => remove.mutate(d.id)} aria-label="Delete document"
              className="rounded-lg p-1.5 text-coral hover:bg-[#fde4d8] transition-colors">
              <Trash2 size={15} />
            </button>
          </div>
        ))}
      </div>

      <button type="button" disabled={busy} onClick={() => ref.current?.click()}
        className="mt-3 w-full rounded-xl border-2 border-dashed border-line flex items-center justify-center gap-2 py-3 text-muted hover:opacity-80 transition-opacity disabled:opacity-50">
        {busy ? <Loader2 size={18} className="animate-spin" /> : <Upload size={18} />}
        <span className="text-[13px] font-bold">{busy ? 'Uploading…' : 'Upload document (PDF or image)'}</span>
        <input ref={ref} type="file" accept="application/pdf,image/*" className="hidden"
          onChange={(e) => { upload(e.target.files?.[0]); e.target.value = ''; }} />
      </button>
    </div>
  );
}
