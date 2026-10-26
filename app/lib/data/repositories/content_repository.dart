// Stories: live travel posts from the public /content API (written and
// approved in the Command Center — see server/internal/http/content.go).

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_riverpod/legacy.dart';

import '../../core/api/api_client.dart';
import '../models/content.dart';
import 'repositories.dart';

/// Stories tab filters: one kind of post (null = all) and a search.
final storiesKindProvider = StateProvider<ContentKind?>((ref) => null);
final storiesQueryProvider = StateProvider<String>((ref) => '');

/// The newest live posts matching the Stories filters (the API's page
/// limit, 50 — far more than are published today).
final storiesFeedProvider = FutureProvider<List<ContentItem>>((ref) async {
  final kind = ref.watch(storiesKindProvider);
  final q = ref.watch(storiesQueryProvider).trim();
  final res = await ref.read(apiProvider).dio.get('/content', queryParameters: {
    'limit': 50,
    if (kind != null) 'type': kind.id,
    if (q.isNotEmpty) 'q': q,
  });
  final data = res.data;
  final items = data is Map ? data['items'] : null;
  return [
    for (final v in items is List ? items : const [])
      if (v is Map<String, dynamic>) ContentItem.fromJson(v),
  ];
});

/// One live post by id or slug, with its linked places. Not kept once
/// nothing shows it, so reopening a post fetches fresh media links (they
/// expire after a day).
final storyProvider = FutureProvider.autoDispose.family<ContentItem, String>((ref, id) async {
  final res = await ref.read(apiProvider).dio.get('/content/${Uri.encodeComponent(id)}');
  return ContentItem.fromJson(Map<String, dynamic>.from(res.data as Map));
});

/// Counts one listen of an audio post (best effort).
Future<void> countStoryPlay(ApiClient api, String id) async {
  try {
    await api.dio.post('/content/${Uri.encodeComponent(id)}/play');
  } catch (_) {
    // A missed count never matters to the listener.
  }
}
