'use client';

// Real photo + promo-video management for the vendor Business Profile,
// replacing the old decorative PHOTO_SLOTS placeholder. Uploads go straight
// to Cloudflare R2 via the shared presigned-upload helper; a cover image, a
// gallery, and one promo video are all mirrored to the traveller listing by
// the server's syncVendorPlace.

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Film, ImagePlus, Loader2, Star, Trash2 } from 'lucide-react';
import { useRef, useState } from 'react';

import { resolveMediaUrl, uploadMedia, validateFile, type Media, type MediaRole } from '@spotto/shared';
import { api } from '../lib/api';

export default function VendorPhotos({ vendorId }: { vendorId: string }) {
  const qc = useQueryClient();
  const { data: media = [] } = useQuery<Media[]>({
    queryKey: ['vendor-media'],
    queryFn: () => api.get<Media[]>('/vendor/media'),
  });

  const cover = media.find((m) => m.role === 'cover');
  const gallery = media.filter((m) => m.role === 'gallery');
  const promo = media.find((m) => m.role === 'promo');

  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<MediaRole | null>(null);

  const upload = async (file: File | undefined, role: MediaRole, kind: 'image' | 'video') => {
    if (!file) return;
    const invalid = validateFile(file, kind);
    if (invalid) {
      setError(invalid);
      return;
    }
    setError(null);
    setBusy(role);
    try {
      await uploadMedia(api, file, { ownerType: 'vendor', ownerId: vendorId, role });
      qc.invalidateQueries({ queryKey: ['vendor-media'] });
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Upload failed — try again.');
    } finally {
      setBusy(null);
    }
  };

  const remove = useMutation({
    mutationFn: (id: string) => api.del(`/media/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['vendor-media'] }),
  });

  return (
    <div className="card p-6">
      <div className="flex items-center gap-2 mb-4">
        <span className="w-9 h-9 rounded-xl bg-accent-tint text-accent flex items-center justify-center">
          <ImagePlus size={18} />
        </span>
        <div>
          <p className="font-extrabold text-[15px]">Photos & video</p>
          <p className="text-[12.5px] text-muted font-semibold">Show off your space — travellers see these on your listing</p>
        </div>
      </div>

      {error && <p className="text-[13px] font-bold text-coral mb-3">{error}</p>}

      {/* Cover */}
      <p className="text-[11.5px] font-extrabold uppercase text-muted mb-1.5">Cover photo</p>
      <UploadTile
        busy={busy === 'cover'}
        media={cover}
        onPick={(f) => upload(f, 'cover', 'image')}
        onRemove={cover ? () => remove.mutate(cover.id) : undefined}
        accept="image/*"
        badge="Cover"
        aspect="aspect-[16/9]"
      />

      {/* Gallery */}
      <p className="text-[11.5px] font-extrabold uppercase text-muted mt-5 mb-1.5">Gallery</p>
      <div className="grid grid-cols-3 gap-2.5">
        {gallery.map((m) => (
          <div key={m.id} className="relative rounded-xl overflow-hidden aspect-[4/3] group">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src={resolveMediaUrl(m.url)} alt="" className="w-full h-full object-cover" />
            <button
              onClick={() => remove.mutate(m.id)}
              className="absolute right-1.5 top-1.5 rounded-full p-1 bg-black/50 text-white opacity-0 group-hover:opacity-100 transition-opacity"
              aria-label="Delete photo">
              <Trash2 size={13} />
            </button>
          </div>
        ))}
        <FilePicker accept="image/*" busy={busy === 'gallery'} onPick={(f) => upload(f, 'gallery', 'image')} label="Add photo" />
      </div>

      {/* Promo video */}
      <p className="text-[11.5px] font-extrabold uppercase text-muted mt-5 mb-1.5">Promo video</p>
      {promo ? (
        <div className="relative rounded-xl overflow-hidden aspect-video">
          <video src={resolveMediaUrl(promo.url)} controls className="w-full h-full object-cover" />
          <button
            onClick={() => remove.mutate(promo.id)}
            className="absolute right-1.5 top-1.5 rounded-full p-1 bg-black/50 text-white"
            aria-label="Delete video">
            <Trash2 size={14} />
          </button>
        </div>
      ) : (
        <FilePicker accept="video/*" busy={busy === 'promo'} onPick={(f) => upload(f, 'promo', 'video')}
          label="Add promo video" icon={<Film size={22} />} aspect="aspect-video" />
      )}
    </div>
  );
}

function UploadTile({
  media, busy, onPick, onRemove, accept, badge, aspect,
}: {
  media?: Media; busy: boolean; onPick: (f?: File) => void; onRemove?: () => void;
  accept: string; badge?: string; aspect: string;
}) {
  if (media) {
    return (
      <div className={`relative rounded-xl overflow-hidden ${aspect} group`}>
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img src={resolveMediaUrl(media.url)} alt="" className="w-full h-full object-cover" />
        {badge && (
          <div className="absolute left-2 top-2 rounded-full px-2 py-0.5 text-[10px] font-extrabold flex items-center gap-1 bg-[var(--sun)] text-[#5a3b00]">
            <Star size={10} className="fill-current" /> {badge}
          </div>
        )}
        {onRemove && (
          <button onClick={onRemove}
            className="absolute right-2 top-2 rounded-full p-1.5 bg-black/50 text-white opacity-0 group-hover:opacity-100 transition-opacity"
            aria-label="Remove">
            <Trash2 size={14} />
          </button>
        )}
      </div>
    );
  }
  return <FilePicker accept={accept} busy={busy} onPick={onPick} label="Upload" aspect={aspect} />;
}

function FilePicker({
  accept, busy, onPick, label, icon, aspect = 'aspect-[4/3]',
}: {
  accept: string; busy: boolean; onPick: (f?: File) => void; label: string; icon?: React.ReactNode; aspect?: string;
}) {
  const ref = useRef<HTMLInputElement>(null);
  return (
    <button type="button" disabled={busy} onClick={() => ref.current?.click()}
      className={`${aspect} rounded-xl border-2 border-dashed border-line flex flex-col items-center justify-center gap-1 text-muted hover:opacity-80 transition-opacity disabled:opacity-50`}>
      {busy ? <Loader2 size={22} className="animate-spin" /> : (icon ?? <ImagePlus size={22} />)}
      <span className="text-[12px] font-bold">{busy ? 'Uploading…' : label}</span>
      <input ref={ref} type="file" accept={accept} className="hidden"
        onChange={(e) => { onPick(e.target.files?.[0]); e.target.value = ''; }} />
    </button>
  );
}
