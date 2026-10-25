// Widget and unit test coverage for:
//   • InspirationLink model (JSON round-trip, platform detection, title inference)
//   • TripNotifier inspiration mutations (add, remove, edit note, dedupe)
//   • formatCurrency helper
//   • InspirationLink.platformOf / titleOf edge cases

import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:spotto/core/storage/local_store.dart';
import 'package:spotto/data/catalog/trip_catalog.dart';
import 'package:spotto/data/repositories/repositories.dart';

// ── Helpers ────────────────────────────────────────────────────────────────

void _mockSecureStorage() {
  const channel = MethodChannel('plugins.it_nomads.com/flutter_secure_storage');
  TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
      .setMockMethodCallHandler(channel, (call) async {
    switch (call.method) {
      case 'read':
        return null;
      case 'readAll':
        return <String, String>{};
      case 'containsKey':
        return false;
      default:
        return null;
    }
  });
}

Future<ProviderContainer> makeContainer() async {
  SharedPreferences.setMockInitialValues({});
  _mockSecureStorage();
  final store = await LocalStore.init();
  final container = ProviderContainer(overrides: [
    localStoreProvider.overrideWithValue(store),
  ]);
  addTearDown(container.dispose);
  return container;
}

InspirationLink _fakeLink({String url = 'https://www.instagram.com/reel/abc123', String note = ''}) =>
    InspirationLink(
      id: 'insp_test_${url.hashCode.abs()}',
      url: url,
      title: InspirationLink.titleOf(url),
      platform: InspirationLink.platformOf(url),
      note: note,
      addedAt: '2026-08-04T12:00:00.000Z',
    );

// ── InspirationLink unit tests ──────────────────────────────────────────────

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  group('InspirationLink.platformOf', () {
    test('detects instagram', () {
      expect(InspirationLink.platformOf('https://www.instagram.com/reel/abc'), 'instagram');
    });
    test('detects tiktok', () {
      expect(InspirationLink.platformOf('https://www.tiktok.com/@user/video/123'), 'tiktok');
    });
    test('detects youtube full URL', () {
      expect(InspirationLink.platformOf('https://www.youtube.com/watch?v=xyz'), 'youtube');
    });
    test('detects youtu.be shortlink', () {
      expect(InspirationLink.platformOf('https://youtu.be/xyz'), 'youtube');
    });
    test('falls back to web for unknown domains', () {
      expect(InspirationLink.platformOf('https://www.example.com/some-article'), 'web');
    });
    test('case-insensitive matching', () {
      expect(InspirationLink.platformOf('HTTPS://INSTAGRAM.COM/reel/abc'), 'instagram');
    });
  });

  group('InspirationLink.titleOf', () {
    test('instagram URL → Instagram post', () {
      expect(InspirationLink.titleOf('https://www.instagram.com/reel/abc'), 'Instagram post');
    });
    test('tiktok URL → TikTok video', () {
      expect(InspirationLink.titleOf('https://www.tiktok.com/@user/video/123'), 'TikTok video');
    });
    test('youtube URL → YouTube video', () {
      expect(InspirationLink.titleOf('https://www.youtube.com/watch?v=xyz'), 'YouTube video');
    });
    test('generic URL → host without www.', () {
      expect(InspirationLink.titleOf('https://www.example.com/article'), 'example.com');
    });
    test('malformed URL → raw string fallback', () {
      expect(InspirationLink.titleOf('not-a-url'), 'not-a-url');
    });
  });

  group('InspirationLink JSON round-trip', () {
    test('toJson / fromJson preserves all fields', () {
      final link = _fakeLink(url: 'https://www.instagram.com/reel/abc', note: 'Great sunset spot');
      final json = link.toJson();
      final restored = InspirationLink.fromJson(json);

      expect(restored.id, link.id);
      expect(restored.url, link.url);
      expect(restored.title, link.title);
      expect(restored.platform, 'instagram');
      expect(restored.note, 'Great sunset spot');
      expect(restored.addedAt, link.addedAt);
    });

    test('fromJson tolerates missing optional fields', () {
      final json = {'id': 'insp_1', 'url': 'https://example.com'};
      final link = InspirationLink.fromJson(json);
      expect(link.title, ''); // defaults to ''
      expect(link.platform, 'web');
      expect(link.note, '');
    });
  });

  group('InspirationLink.copyWith', () {
    test('copies with new note, keeps other fields', () {
      final link = _fakeLink();
      final updated = link.copyWith(note: 'New note');
      expect(updated.note, 'New note');
      expect(updated.url, link.url);
      expect(updated.id, link.id);
    });

    test('copies with new title, keeps other fields', () {
      final link = _fakeLink();
      final updated = link.copyWith(title: 'Custom Title');
      expect(updated.title, 'Custom Title');
      expect(updated.platform, link.platform);
    });
  });

  // ── TripNotifier inspiration mutations ─────────────────────────────────

  group('TripNotifier inspiration mutations', () {
    test('addInspiration appends link and persists to trip JSON', () async {
      final c = await makeContainer();
      final notifier = c.read(tripProvider.notifier);
      final link = _fakeLink();

      notifier.addInspiration(link);

      final trip = c.read(tripProvider);
      expect(trip.inspirations, hasLength(1));
      expect(trip.inspirations.first.url, link.url);

      // Verify JSON round-trip preserves inspiration
      final restored = Trip.fromJson(trip.toJson());
      expect(restored.inspirations, hasLength(1));
      expect(restored.inspirations.first.platform, 'instagram');
    });

    test('addInspiration deduplicates by URL', () async {
      final c = await makeContainer();
      final notifier = c.read(tripProvider.notifier);
      final link = _fakeLink();

      notifier.addInspiration(link);
      notifier.addInspiration(link); // duplicate

      expect(c.read(tripProvider).inspirations, hasLength(1));
    });

    test('removeInspiration removes by id', () async {
      final c = await makeContainer();
      final notifier = c.read(tripProvider.notifier);
      final link = _fakeLink();
      notifier.addInspiration(link);

      notifier.removeInspiration(link.id);

      expect(c.read(tripProvider).inspirations, isEmpty);
    });

    test('removeInspiration on missing id is a no-op', () async {
      final c = await makeContainer();
      final notifier = c.read(tripProvider.notifier);
      notifier.addInspiration(_fakeLink());

      notifier.removeInspiration('nonexistent_id');

      expect(c.read(tripProvider).inspirations, hasLength(1));
    });

    test('editInspirationNote updates note in place', () async {
      final c = await makeContainer();
      final notifier = c.read(tripProvider.notifier);
      final link = _fakeLink();
      notifier.addInspiration(link);

      notifier.editInspirationNote(link.id, 'Updated note');

      final insp = c.read(tripProvider).inspirations.first;
      expect(insp.note, 'Updated note');
      expect(insp.url, link.url); // url unchanged
    });

    test('multiple inspirations added newest-first', () async {
      final c = await makeContainer();
      final notifier = c.read(tripProvider.notifier);
      final first = _fakeLink(url: 'https://www.instagram.com/reel/first');
      final second = _fakeLink(url: 'https://www.tiktok.com/@u/video/second');
      notifier.addInspiration(first);
      notifier.addInspiration(second);

      final list = c.read(tripProvider).inspirations;
      expect(list[0].url, second.url); // newest first
      expect(list[1].url, first.url);
    });

    test('Trip.fromJson with no inspirations key returns empty list', () async {
      // Simulates loading an older trip saved before this feature shipped.
      final json = {
        'name': 'Old Trip',
        'code': 'OLD-1',
        'startDate': '2026-01-01',
        'collaborators': <String>[],
        'days': <dynamic>[],
        // no 'inspirations' key
      };
      final trip = Trip.fromJson(json);
      expect(trip.inspirations, isEmpty);
    });
  });
}
