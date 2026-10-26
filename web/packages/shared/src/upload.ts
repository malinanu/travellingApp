import type { Api } from './api';
import type { Media } from './types';

/** What role a piece of media plays for its owner. */
export type MediaRole = 'cover' | 'gallery' | 'promo' | 'doc' | 'br_cert' | 'avatar' | 'review' | 'contrib' | 'menu_item' | 'audio' | 'inline' | 'chapter';

export interface UploadTarget {
  ownerType: 'vendor' | 'user' | 'place' | 'review' | 'content';
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
  audio: { types: ['audio/mpeg', 'audio/mp3', 'audio/mp4', 'audio/x-m4a', 'audio/aac'], max: 200 << 20 },
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

  let put: Response;
  try {
    put = await fetch(presign.uploadUrl, {
      method: 'PUT',
      headers: { 'Content-Type': file.type },
      body: file,
    });
  } catch {
    // fetch() throws its own bare "TypeError: Failed to fetch" — with no
    // HTTP status at all — when the request never gets a response, which
    // for this direct-to-R2 PUT almost always means R2's bucket CORS
    // policy doesn't allow this origin (a Cloudflare dashboard setting,
    // not something wrong with the file or a transient blip most of the
    // time). Rethrow something a vendor/admin can actually act on instead
    // of that cryptic browser string, naming the site that has to be in the
    // bucket's AllowedOrigins (each web app uploads from its own origin).
    const origin = typeof window === 'undefined' ? 'this site' : window.location.origin;
    throw new Error(
      `Couldn't reach file storage. Try again in a moment; if it keeps failing, the storage isn't allowing uploads from ${origin} yet (Cloudflare R2 → bucket → Settings → CORS policy). It's not your file.`,
    );
  }
  if (!put.ok) {
    // R2 presigned PUT URLs expire 10 minutes after they're issued (see
    // handleMediaPresign). A big file on a slow connection can genuinely
    // outlive that window and get a real 403 from R2 — surface something
    // actionable instead of a bare status code, same reasoning as the
    // network-failure case above.
    if (put.status === 403) {
      throw new Error("Upload took too long and the link expired — try again.");
    }
    throw new Error(`Upload failed (${put.status})`);
  }

  return api.post<Media>('/media/confirm', {
    key: presign.key,
    ownerType: target.ownerType,
    ownerId: target.ownerId,
    role: target.role,
  });
}

/**
 * Shrinks a photo in the browser before upload: at most maxPx on its long
 * side, re-encoded as JPEG. A phone's 8 MB photo comes out around 300 KB,
 * which matters for travellers loading it on mobile data. Returns the file
 * as it is when it's already small, isn't a decodable image, or wouldn't
 * get smaller (the server checks every upload either way).
 */
export async function shrinkImage(file: File, maxPx = 1600, quality = 0.82): Promise<File> {
  if (typeof document === 'undefined' || !file.type.startsWith('image/')) return file;
  let bitmap: ImageBitmap;
  try {
    bitmap = await createImageBitmap(file);
  } catch {
    return file;
  }
  const scale = Math.min(1, maxPx / Math.max(bitmap.width, bitmap.height));
  if (scale === 1 && file.size <= 600 << 10) {
    bitmap.close();
    return file;
  }
  const canvas = document.createElement('canvas');
  canvas.width = Math.round(bitmap.width * scale);
  canvas.height = Math.round(bitmap.height * scale);
  const ctx = canvas.getContext('2d');
  if (!ctx) {
    bitmap.close();
    return file;
  }
  ctx.fillStyle = '#fff'; // JPEG has no transparency
  ctx.fillRect(0, 0, canvas.width, canvas.height);
  ctx.drawImage(bitmap, 0, 0, canvas.width, canvas.height);
  bitmap.close();
  const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, 'image/jpeg', quality));
  if (!blob || blob.size >= file.size) return file;
  return new File([blob], file.name.replace(/\.[^.]+$/, '') + '.jpg', { type: 'image/jpeg' });
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
