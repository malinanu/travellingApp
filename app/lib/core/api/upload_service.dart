// Client side of the presigned R2 upload flow, mirroring the web
// uploadMedia() helper: ask the API to sign a PUT, upload the bytes straight
// to R2 (a bare Dio with no auth interceptor — R2 rejects headers not in the
// signature), then confirm so the API records and associates the object.

import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../data/repositories/repositories.dart';
import 'api_client.dart';

class UploadService {
  final ApiClient api;
  UploadService(this.api);

  /// Uploads [bytes] and returns the created media row (includes 'url').
  Future<Map<String, dynamic>> upload({
    required Uint8List bytes,
    required String contentType,
    required String ownerType,
    required String ownerId,
    required String role,
  }) async {
    final presign = await api.dio.post('/media/presign', data: {
      'ownerType': ownerType,
      'ownerId': ownerId,
      'role': role,
      'contentType': contentType,
      'size': bytes.length,
    });
    final p = (presign.data as Map).cast<String, dynamic>();
    final uploadUrl = p['uploadUrl'] as String;
    final key = p['key'] as String;

    // Straight to R2 — a fresh Dio so the app's bearer-token interceptor and
    // base URL don't touch the presigned request. Content-Type must match
    // exactly what was signed.
    final put = await Dio().put<void>(
      uploadUrl,
      data: bytes,
      options: Options(
        headers: {
          'Content-Type': contentType,
          Headers.contentLengthHeader: bytes.length,
        },
      ),
    );
    final status = put.statusCode ?? 0;
    if (status < 200 || status >= 300) {
      throw Exception('Upload failed ($status)');
    }

    final confirm = await api.dio.post('/media/confirm', data: {
      'key': key,
      'ownerType': ownerType,
      'ownerId': ownerId,
      'role': role,
    });
    return (confirm.data as Map).cast<String, dynamic>();
  }

  /// Deletes a previously uploaded media row (R2 object + DB row) — the
  /// server only allows this for the uploader themselves (or an admin), see
  /// handleMediaDelete's ownership check in server/internal/http/media.go.
  Future<void> delete(String mediaId) async {
    await api.dio.delete('/media/$mediaId');
  }
}

final uploadServiceProvider = Provider<UploadService>((ref) {
  return UploadService(ref.watch(apiProvider));
});

/// Best-effort MIME type from a picked file's name/mime, restricted to what
/// the server accepts. Returns null for an unsupported type.
String? mimeForUpload(String? mimeType, String name) {
  final byMime = _allowed[mimeType?.toLowerCase()];
  if (byMime != null) return byMime;
  final lower = name.toLowerCase();
  for (final entry in _byExt.entries) {
    if (lower.endsWith(entry.key)) return entry.value;
  }
  return null;
}

const _allowed = {
  'image/jpeg': 'image/jpeg',
  'image/png': 'image/png',
  'image/webp': 'image/webp',
  'video/mp4': 'video/mp4',
  'video/quicktime': 'video/quicktime',
};

const _byExt = {
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.png': 'image/png',
  '.webp': 'image/webp',
  '.mp4': 'video/mp4',
  '.mov': 'video/quicktime',
};
