// Stories: parsing the /content API, "near you" ordering, turning an
// itinerary into a trip, the Stories tab, adding an itinerary from a story
// to the traveller's trips, following an audio guide's transcript, and its
// photo chapters (where in the guide the listener is).

import 'dart:ui' show Tristate;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:spotto/core/audio/story_audio.dart';
import 'package:spotto/core/storage/local_store.dart';
import 'package:spotto/data/catalog/itinerary_import.dart';
import 'package:spotto/data/catalog/trip_catalog.dart';
import 'package:spotto/data/models/content.dart';
import 'package:spotto/data/repositories/content_repository.dart';
import 'package:spotto/data/repositories/repositories.dart';
import 'package:spotto/features/traveller/stories/stories_screen.dart';
import 'package:spotto/features/traveller/stories/story_player.dart';
import 'package:spotto/features/traveller/stories/story_screen.dart';

/// A live itinerary as GET /content/{id} returns it.
const _itineraryJson = {
  'id': 'cnt_1',
  'type': 'itinerary',
  'slug': 'two-days-kandy-1',
  'title': 'Two days of culture',
  'summary': 'Kandy to Sigiriya.',
  'body': '## Before you go\nBring a sarong for the temples.',
  'coverUrl': '',
  'audioUrl': '',
  'audioSeconds': 0,
  'chapters': [],
  'placeIds': [],
  'days': [
    {
      'title': 'Kandy',
      'stops': [
        {'placeId': 'plc_tooth', 'name': 'Temple of the Tooth', 'time': '09:00'},
        {'name': 'Lunch by the lake'},
      ],
    },
    {
      'title': 'Sigiriya',
      'stops': [
        {'name': 'Pidurangala sunset', 'lat': 7.966, 'lng': 80.761, 'time': '17:30', 'note': 'Bring water'},
        {'name': 'Night drive back', 'time': '23:50'},
      ],
    },
  ],
  'tags': ['culture'],
  'regionId': 'kandy',
  'regionName': 'Kandy',
  'regionLat': 7.2906,
  'regionLng': 80.6337,
  'language': 'en',
  'authorName': 'Spotto Admin',
  'status': 'published',
  'publishedAt': '2026-10-26T06:00:00Z',
  'views': 3,
  'plays': 0,
  'places': [
    {'id': 'plc_tooth', 'name': 'Temple of the Tooth', 'category': 'Temple', 'lat': 7.2936, 'lng': 80.6413},
  ],
};

ContentItem _post(String id, {String? region, double? lat, double? lng, ContentKind kind = ContentKind.blog}) =>
    ContentItem(id: id, kind: kind, title: 'Post $id', regionName: region ?? '', regionLat: lat, regionLng: lng);

void _mockSecureStorage() {
  const channel = MethodChannel('plugins.it_nomads.com/flutter_secure_storage');
  TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.setMockMethodCallHandler(channel, (call) async {
    switch (call.method) {
      case 'readAll':
        return <String, String>{};
      case 'containsKey':
        return false;
      default:
        return null;
    }
  });
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  group('ContentItem.fromJson', () {
    test('reads a full itinerary', () {
      final item = ContentItem.fromJson(_itineraryJson);
      expect(item.kind, ContentKind.itinerary);
      expect(item.days.length, 2);
      expect(item.stopCount, 4);
      expect(item.days[1].stops.first.lat, 7.966);
      expect(item.days[1].stops.first.note, 'Bring water');
      expect(item.place('plc_tooth')?.category, 'Temple');
      expect(item.publishedAt, DateTime.utc(2026, 10, 26, 6));
      expect(item.regionLat, 7.2906);
    });

    test('tolerates a list entry with missing fields and an unknown type', () {
      final item = ContentItem.fromJson({'id': 'x', 'type': 'podcast', 'title': 'Hi', 'chapters': null, 'days': 'bad'});
      expect(item.kind, ContentKind.blog);
      expect(item.chapters, isEmpty);
      expect(item.days, isEmpty);
      expect(item.body, '');
      expect(item.regionLat, isNull);
    });

    test('reads an audio post with chapters', () {
      final item = ContentItem.fromJson({
        'id': 'a',
        'type': 'audio',
        'title': 'Fort walk',
        'audioUrl': '/media/raw/content/a/m.mp3?token=t',
        'audioSeconds': 200,
        'chapters': [
          {'title': 'Lighthouse', 'startSec': 0},
          {'title': 'Flag Rock', 'startSec': 60},
        ],
      });
      expect(item.kind, ContentKind.audio);
      expect(item.chapters.map((c) => c.startSec), [0, 60]);
      expect(formatListenLength(item.audioSeconds), '3 min');
      expect(item.transcript, isNull);
    });

    test("reads an audio post's transcript, skipping empty lines", () {
      final item = ContentItem.fromJson({
        'id': 'a',
        'type': 'audio',
        'title': 'Fort walk',
        'transcript': {
          'language': 'en',
          'durationSec': 12.5,
          'segments': [
            {'start': 0.0, 'end': 4.2, 'text': ' Welcome to Galle Fort. '},
            {'start': 4.2, 'end': 6.0, 'text': ''},
            {'start': 6.0, 'end': 9.5, 'text': 'The lighthouse is ahead.'},
            'not a line',
          ],
        },
      });
      final t = item.transcript!;
      expect(t.language, 'en');
      expect(t.lines.map((l) => l.text), ['Welcome to Galle Fort.', 'The lighthouse is ahead.']);
      expect(t.lines.last.start, 6.0);
    });

    test('a transcript with no words, or of the wrong shape, is left out', () {
      ContentTranscript? read(Object? t) => ContentItem.fromJson({'id': 'a', 'type': 'audio', 'transcript': t}).transcript;
      expect(read({'language': 'en', 'segments': []}), isNull);
      expect(read({'segments': [{'start': 0, 'end': 1, 'text': '  '}]}), isNull);
      expect(read('nope'), isNull);
      expect(read({'segments': 'nope'}), isNull);
    });
  });

  test('stop times parse only as HH:mm', () {
    expect(const ItineraryStop(name: 'a', time: '09:30').minutes, 570);
    expect(const ItineraryStop(name: 'a', time: '7:05').minutes, 425);
    expect(const ItineraryStop(name: 'a', time: '24:00').minutes, isNull);
    expect(const ItineraryStop(name: 'a', time: 'noon').minutes, isNull);
    expect(const ItineraryStop(name: 'a').minutes, isNull);
  });

  test('splitNearby puts nearby towns first, nearest first, and keeps order otherwise', () {
    final items = [
      _post('jaffna', lat: 9.66, lng: 80.03),
      _post('galle1', lat: 6.05, lng: 80.22),
      _post('island'),
      _post('unawatuna', lat: 6.01, lng: 80.25),
      _post('galle2', lat: 6.05, lng: 80.22),
    ];
    final split = splitNearby(items, lat: 6.03, lng: 80.24);
    expect(split.near.map((i) => i.id), ['unawatuna', 'galle1', 'galle2']);
    expect(split.rest.map((i) => i.id), ['jaffna', 'island']);
  });

  test('listen lengths and clock times read naturally', () {
    expect(formatListenLength(0), '');
    expect(formatListenLength(20), '1 min');
    expect(formatListenLength(754), '13 min');
    expect(formatListenLength(3900), '1 h 05 min');
    expect(formatClock(const Duration(seconds: 754)), '12:34');
    expect(formatClock(const Duration(seconds: 3723)), '1:02:03');
  });

  test('currentChapter follows the playback position', () {
    const chapters = [ContentChapter(title: 'a', startSec: 10), ContentChapter(title: 'b', startSec: 60)];
    expect(currentChapter(chapters, const Duration(seconds: 5)), -1);
    expect(currentChapter(chapters, const Duration(seconds: 10)), 0);
    expect(currentChapter(chapters, const Duration(seconds: 61)), 1);
  });

  group('photo chapters', () {
    final guide = ContentItem.fromJson({
      'id': 'sig',
      'type': 'audio',
      'title': 'Sigiriya rock',
      'coverUrl': '/media/raw/content/sig/cover.jpg?token=c',
      'audioSeconds': 600,
      'chapters': [
        {'title': 'Water gardens', 'startSec': 0, 'imageUrl': '/media/raw/content/sig/a.jpg?token=a'},
        {'title': 'Mirror wall', 'startSec': 120},
        {'title': "Lion's Paw", 'startSec': 360, 'imageUrl': '/media/raw/content/sig/c.jpg?token=c'},
      ],
    });

    test('reads chapter photos, ends and fallbacks', () {
      expect(guide.hasChapterPhotos, isTrue);
      expect(guide.chapters.first.imageUrl, '/media/raw/content/sig/a.jpg?token=a');
      expect(guide.chapters[1].imageUrl, '');
      expect([for (var i = 0; i < 3; i++) guide.chapterEndSec(i)], [120, 360, 600]);
      expect(guide.chapterImage(1), guide.coverUrl, reason: 'no photo: the cover');
      expect(guide.chapterImage(-1), guide.coverUrl);
      expect(guide.chapterAt(0), 0);
      expect(guide.chapterAt(119.9), 0);
      expect(guide.chapterAt(120), 1);
      expect(guide.chapterAt(9999), 2);
      expect(ContentItem.fromJson({'id': 'x', 'type': 'audio', 'chapters': [{'title': 'a', 'startSec': 0}]}).hasChapterPhotos, isFalse);
    });

    test('the lock screen follows the chapter being heard', () {
      final start = storyNowShowing(guide, Duration.zero);
      expect(start.chapter, 0);
      expect(start.art, '/media/raw/content/sig/a.jpg?token=a');
      expect(start.subtitle, 'Water gardens');
      final wall = storyNowShowing(guide, const Duration(minutes: 3));
      expect(wall.chapter, 1);
      expect(wall.art, guide.coverUrl);
      expect(wall.subtitle, 'Mirror wall');
      expect(storyNowShowing(guide, const Duration(minutes: 7)).art, '/media/raw/content/sig/c.jpg?token=c');
      final none = storyNowShowing(const ContentItem(id: 'n', kind: ContentKind.audio, title: 'n', coverUrl: 'cov'), Duration.zero);
      expect((none.chapter, none.art, none.subtitle), (-1, 'cov', ''));
    });

    // Without photos, so no network images load in the test.
    final plain = ContentItem.fromJson({
      'id': 'p',
      'type': 'audio',
      'title': 'Galle Fort',
      'audioSeconds': 300,
      'chapters': [
        {'title': 'Lighthouse', 'startSec': 0},
        {'title': 'Flag Rock', 'startSec': 120},
        {'title': 'Dutch Church', 'startSec': 200},
      ],
    });

    testWidgets('the chapter photo says where in the guide it is', (tester) async {
      await tester.pumpWidget(MaterialApp(home: Scaffold(body: StoryChapterHero(item: plain, index: 1))));
      expect(find.text('Chapter 2 of 3 · 2:00–3:20'), findsOneWidget);
      expect(find.text('Flag Rock'), findsOneWidget);
    });

    testWidgets('the photo strip marks the chapter playing and jumps', (tester) async {
      final tapped = <int>[];
      await tester.pumpWidget(MaterialApp(
        home: Scaffold(body: StoryChapterStrip(item: plain, current: 2, onTap: tapped.add)),
      ));
      expect(find.text('Lighthouse'), findsOneWidget);
      expect(find.text('0:00–2:00'), findsOneWidget);
      expect(find.byIcon(Icons.graphic_eq_rounded), findsOneWidget, reason: 'one card marked as playing');
      expect(tester.getSemantics(find.bySemanticsLabel(RegExp(r'^Chapter 3: Dutch Church'))).flagsCollection.isSelected, Tristate.isTrue);
      await tester.tap(find.text('Flag Rock'));
      expect(tapped, [1]);
    });

    testWidgets('transcript lines sit under their chapter', (tester) async {
      final transcript = ContentTranscript(lines: [
        for (final (s, t) in [(0.0, 'The lighthouse.'), (60.0, 'Built in 1939.'), (125.0, 'Flag Rock ahead.')])
          TranscriptLine(start: s, end: s + 5, text: t),
      ]);
      await tester.pumpWidget(MaterialApp(
        home: Scaffold(
          body: StoryTranscriptView(
            transcript: transcript,
            chapters: plain.chapters,
            current: -1,
            open: true,
            onToggle: () {},
            onTapLine: (_) {},
          ),
        ),
      ));
      expect(find.text('1 · Lighthouse'), findsOneWidget);
      expect(find.text('2 · Flag Rock'), findsOneWidget);
      expect(find.text('3 · Dutch Church'), findsNothing, reason: 'no lines in it');
      double y(String s) => tester.getTopLeft(find.text(s)).dy;
      expect(y('1 · Lighthouse') < y('The lighthouse.') && y('Built in 1939.') < y('2 · Flag Rock'), isTrue);
      expect(y('2 · Flag Rock') < y('Flag Rock ahead.'), isTrue);
    });
  });

  test('currentLine follows the playback position', () {
    const lines = [
      TranscriptLine(start: 1.5, end: 4, text: 'a'),
      TranscriptLine(start: 4, end: 9, text: 'b'),
      TranscriptLine(start: 12.25, end: 15, text: 'c'),
    ];
    expect(currentLine(lines, Duration.zero), -1);
    expect(currentLine(lines, const Duration(milliseconds: 1500)), 0);
    expect(currentLine(lines, const Duration(seconds: 4)), 1);
    expect(currentLine(lines, const Duration(seconds: 11)), 1, reason: 'a pause keeps the last line');
    expect(currentLine(lines, const Duration(seconds: 20)), 2);
    expect(currentLine(const [], const Duration(seconds: 3)), -1);
  });

  test('a folded transcript shows the lines around the one being heard', () {
    expect(transcriptWindow(20, -1), (0, kTranscriptPeek));
    expect(transcriptWindow(20, 0), (0, kTranscriptPeek));
    expect(transcriptWindow(20, 8), (7, 7 + kTranscriptPeek));
    expect(transcriptWindow(20, 19), (20 - kTranscriptPeek, 20));
    expect(transcriptWindow(3, 2), (0, 3));
  });

  testWidgets('the transcript highlights the line being heard and opens in full', (tester) async {
    final transcript = ContentTranscript(language: 'en', lines: [
      for (var i = 0; i < 10; i++) TranscriptLine(start: i * 10.0, end: i * 10.0 + 9, text: 'Line $i'),
    ]);
    final tapped = <int>[];
    var open = false;
    Future<void> show({required int current}) => tester.pumpWidget(MaterialApp(
          home: Scaffold(
            body: SingleChildScrollView(
              child: StatefulBuilder(
                builder: (context, setState) => StoryTranscriptView(
                  transcript: transcript,
                  current: current,
                  open: open,
                  onToggle: () => setState(() => open = !open),
                  onTapLine: tapped.add,
                ),
              ),
            ),
          ),
        ));

    await show(current: -1);
    expect(find.text('Line 0'), findsOneWidget);
    expect(find.text('Line 6'), findsNothing);
    expect(find.text('Show full transcript (10 lines)'), findsOneWidget);

    // Folded, it follows along: from the line before the one being heard.
    await show(current: 5);
    expect(find.text('Line 3'), findsNothing);
    expect(find.text('Line 4'), findsOneWidget);
    expect(find.text('Line 9'), findsOneWidget);
    expect(find.text('0:50'), findsOneWidget);
    expect(find.text('…'), findsOneWidget);

    await tester.tap(find.text('Line 7'));
    expect(tapped, [7]);

    await tester.tap(find.text('Show full transcript (10 lines)'));
    await tester.pump();
    expect(find.text('Line 0'), findsOneWidget);
    expect(find.text('Line 9'), findsOneWidget);
    expect(find.text('Show less'), findsOneWidget);
  });

  testWidgets('an audio story shows its transcript under the player', (tester) async {
    final container = ProviderContainer(overrides: [
      storyProvider.overrideWith((ref, id) async => ContentItem.fromJson({
            'id': 'cnt_a',
            'type': 'audio',
            'title': 'Fort walk',
            'audioUrl': '/media/raw/content/cnt_a/m.mp3?token=t',
            'audioSeconds': 30,
            'transcript': {
              'language': 'en',
              'segments': [
                {'start': 0.0, 'end': 4.0, 'text': 'Welcome to Galle Fort.'},
                {'start': 4.0, 'end': 9.0, 'text': 'The lighthouse is ahead.'},
              ],
            },
          })),
    ]);
    addTearDown(container.dispose);
    await tester.pumpWidget(UncontrolledProviderScope(
      container: container,
      child: const MaterialApp(home: StoryScreen(storyId: 'cnt_a')),
    ));
    await tester.pumpAndSettle();

    await tester.scrollUntilVisible(find.text('The lighthouse is ahead.'), 200, scrollable: find.byType(Scrollable).first);
    expect(find.text('Transcript'), findsOneWidget);
    expect(find.text('Welcome to Galle Fort.'), findsOneWidget);
    expect(find.textContaining('Show full transcript'), findsNothing);
  });

  group('tripFromItinerary', () {
    test('keeps days, stops, times and places, with coordinates for every stop', () {
      final item = ContentItem.fromJson(_itineraryJson);
      final trip = tripFromItinerary(item, name: '  ', startDate: '2026-11-01');
      expect(trip.name, 'Two days of culture');
      expect(trip.startDate, '2026-11-01');
      expect(trip.code, startsWith('TRP-'));
      expect(trip.days.length, 2);

      final kandy = trip.days[0].stops;
      expect(kandy[0].placeId, 'plc_tooth');
      expect(kandy[0].time, 9 * 60);
      expect(kandy[0].custom?.name, 'Temple of the Tooth');
      expect(kandy[0].custom?.type, 'Temple');
      expect((kandy[0].custom?.lat, kandy[0].custom?.lng), (7.2936, 80.6413)); // the linked place
      expect(kandy[1].time, 9 * 60 + 90); // no time: 90 minutes after the last
      expect((kandy[1].custom?.lat, kandy[1].custom?.lng), (7.2906, 80.6337)); // the post's town

      final sigiriya = trip.days[1].stops;
      expect(sigiriya[0].time, 17 * 60 + 30);
      expect((sigiriya[0].custom?.lat, sigiriya[0].custom?.lng), (7.966, 80.761)); // its own
      expect(sigiriya[1].time, 23 * 60 + 30); // clamped like the trip planner
      // Every stop resolves for the trip screen, loaded catalog or not.
      expect(trip.days.expand((d) => d.stops).every((s) => tripPlace(s) != null), isTrue);
    });

    test('an itinerary without days still makes a usable trip', () {
      final trip = tripFromItinerary(const ContentItem(id: 'e', kind: ContentKind.itinerary, title: ''),
          name: 'My plan', startDate: '2026-11-01');
      expect(trip.name, 'My plan');
      expect(trip.days.length, 1);
      expect(trip.days.single.stops, isEmpty);
    });
  });

  testWidgets('the Stories tab lists nearby towns first and filters by kind', (tester) async {
    final container = ProviderContainer(overrides: [
      storiesFeedProvider.overrideWith((ref) async => [
            _post('jaffna', region: 'Jaffna', lat: 9.66, lng: 80.03),
            _post('galle', region: 'Galle', lat: 6.05, lng: 80.22, kind: ContentKind.audio),
          ]),
      devicePositionProvider.overrideWith((ref) async => ((lat: 6.03, lng: 80.24), DeviceLocationResult.granted)),
    ]);
    addTearDown(container.dispose);
    await tester.pumpWidget(UncontrolledProviderScope(
      container: container,
      child: const MaterialApp(home: Scaffold(body: StoriesScreen())),
    ));
    await tester.pumpAndSettle();

    expect(find.text('NEAR YOU'), findsOneWidget);
    expect(find.text('MORE FROM SRI LANKA'), findsOneWidget);
    final near = tester.getTopLeft(find.text('Post galle')).dy;
    final far = tester.getTopLeft(find.text('Post jaffna')).dy;
    expect(near, lessThan(far));

    await tester.tap(find.text('Itineraries'));
    await tester.pump();
    expect(container.read(storiesKindProvider), ContentKind.itinerary);
  });

  testWidgets('an itinerary story can be added as the active trip', (tester) async {
    SharedPreferences.setMockInitialValues({});
    _mockSecureStorage();
    final store = await LocalStore.init();
    final container = ProviderContainer(overrides: [
      localStoreProvider.overrideWithValue(store),
      storyProvider.overrideWith((ref, id) async => ContentItem.fromJson(_itineraryJson)),
    ]);
    addTearDown(container.dispose);
    await tester.pumpWidget(UncontrolledProviderScope(
      container: container,
      child: const MaterialApp(home: StoryScreen(storyId: 'cnt_1')),
    ));
    await tester.pumpAndSettle();

    expect(find.text('Two days of culture'), findsOneWidget);
    expect(find.text('Temple of the Tooth'), findsOneWidget);
    await tester.scrollUntilVisible(find.text('Bring water'), 200, scrollable: find.byType(Scrollable).first);
    expect(find.text('Bring water'), findsOneWidget);
    await tester.scrollUntilVisible(find.text('Add to my trips'), -200, scrollable: find.byType(Scrollable).first);

    await tester.tap(find.text('Add to my trips'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'Culture weekend');
    await tester.tap(find.text('Add trip'));
    await tester.pumpAndSettle();

    final trip = container.read(tripProvider);
    expect(trip.name, 'Culture weekend');
    expect(trip.days.length, 2);
    expect(trip.days.first.stops.first.custom?.name, 'Temple of the Tooth');
    expect(container.read(savedTripsProvider).first.code, trip.code);
    expect(find.text('“Culture weekend” is now your active trip.'), findsOneWidget);
  });
}
