import type { Api } from './api';
import type { Media } from './types';

/** What role a piece of media plays for its owner. */
export type MediaRole = 'cover' | 'gallery' | 'promo' | 'doc' | 'avatar' | 'review' | 'contrib' | 'menu_item';

export interface UploadTarget {
  ownerType: 'vendor' | 'user' | 'place' | 'review';
  ownerId: string;
  role: MediaRole;
}

interface PresignResponse {
  uploadUrl: string;
  key: string;
  publicUrl: string;
  kind: string;
}

/** Client-side guardrails mirroring the server's allowlist/caps, so users
 * get an instant error instead of a round trip. Kept in sync with
 * internal/media/media.go. */
export const MEDIA_LIMITS = {
  image: { types: ['image/jpeg', 'image/png', 'image/webp'], max: 8 << 20 },
  video: { types: ['video/mp4', 'video/webm', 'video/quicktime'], max: 100 << 20 },
  doc: { types: ['application/pdf', 'image/jpeg', 'image/png'], max: 15 << 20 },
} as const;

/**
 * uploadMedia runs the full presigned flow: ask the API to sign a PUT,
 * upload the bytes straight to R2 (no auth header — R2 rejects headers not
 * in the signature), then confirm so the API records and associates the
 * object. Returns the created Media row.
 */
export async function uploadMedia(api: Api, file: File, target: UploadTarget): Promise<Media> {
  const presign = await api.post<PresignResponse>('/media/presign', {
    ownerType: target.ownerType,
    ownerId: target.ownerId,
    role: target.role,
    contentType: file.type,
    size: file.size,
  });

  const put = await fetch(presign.uploadUrl, {
    method: 'PUT',
    headers: { 'Content-Type': file.type },
    body: file,
  });
  if (!put.ok) throw new Error(`Upload failed (${put.status})`);

  return api.post<Media>('/media/confirm', {
    key: presign.key,
    ownerType: target.ownerType,
    ownerId: target.ownerId,
    role: target.role,
  });
}

/** Validate a file against the limits for a kind before uploading. Returns
 * an error message, or null if the file is acceptable. */
export function validateFile(file: File, kind: keyof typeof MEDIA_LIMITS): string | null {
  const limit = MEDIA_LIMITS[kind];
  if (!(limit.types as readonly string[]).includes(file.type)) {
    return `Unsupported file type (${file.type || 'unknown'}).`;
  }
  if (file.size > limit.max) {
    return `File is too large (max ${Math.round(limit.max / (1 << 20))} MB).`;
  }
  return null;
}
