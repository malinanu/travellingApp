import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:spotto/data/models/models.dart';
import 'package:spotto/features/traveller/guide/guide_place.dart';

void main() {
  test('parses a server summary into a card', () {
    final card = GuidePlace.fromJson({
      'id': 'plc_1',
      'name': 'Sea Breeze',
      'category': 'Restaurant',
      'priceLevel': r'$$',
      'rating': 4.6,
      'distanceKm': 0.8,
      'lat': 6.03,
      'lng': 80.21,
      'coverThumb': '/media/raw/a.jpg?token=x',
    });
    expect(card.metaLine, r'800 m · $$ · ★ 4.6');
    expect(card.directionsUrl.toString(), 'https://www.google.com/maps/dir/?api=1&destination=6.03,80.21');
  });

  test('meta line skips unknowns and shows the region without a distance', () {
    const card = GuidePlace(id: 'p', name: 'X', region: 'Galle');
    expect(card.metaLine, 'Galle');
    expect(card.directionsUrl, isNull);
  });

  test('server summaries win; ids fall back to cached places, capped', () {
    const summary = GuidePlace(id: 's', name: 'From server');
    expect(guideCardsFor([summary], ['other'], const []).single.name, 'From server');

    final cached = [
      for (var i = 0; i < 6; i++)
        Place.fromJson({'id': 'p$i', 'name': 'Place $i', 'lat': 6.0, 'lng': 80.0}),
    ];
    final cards = guideCardsFor(const [], ['p1', 'missing', 'p0', 'p2', 'p3', 'p4'], cached);
    expect(cards.map((c) => c.id), ['p1', 'p0', 'p2', 'p3']);
  });

  testWidgets('GuidePlaceCard renders name, badges, and action buttons', (tester) async {
    const place = GuidePlace(
      id: 'plc_1',
      name: 'Coconut Tree Hill',
      category: 'Attraction',
      priceLevel: r'$$',
      rating: 4.8,
      distanceKm: 1.2,
      lat: 5.9431,
      lng: 80.4608,
    );

    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(
          home: Scaffold(
            body: GuidePlaceCard(place: place),
          ),
        ),
      ),
    );

    expect(find.text('Coconut Tree Hill'), findsOneWidget);
    expect(find.text('Attraction'), findsOneWidget);
    expect(find.text('4.8'), findsOneWidget);
    expect(find.text('1.2km'), findsOneWidget);
    expect(find.text(r'$$'), findsOneWidget);
    expect(find.text('Trip'), findsOneWidget);
    expect(find.byIcon(Icons.directions_outlined), findsOneWidget);
  });

  testWidgets('MarkdownBody renders formatted text without raw markdown asterisks', (tester) async {
    const rawText = '* **Tuana Mirissa**: Great beach cafe.\n* **Secret Beach**: Cozy cove.';

    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(
          body: MarkdownBody(
            data: rawText,
          ),
        ),
      ),
    );

    expect(find.textContaining('Tuana Mirissa'), findsOneWidget);
    expect(find.textContaining('**Tuana Mirissa**'), findsNothing);
    expect(find.textContaining('Secret Beach'), findsOneWidget);
    expect(find.textContaining('**Secret Beach**'), findsNothing);
  });

  test('parses and serializes description field', () {
    final place = GuidePlace.fromJson({
      'id': 'p1',
      'name': 'Doctor\'s House',
      'description': 'A historic Dutch colonial property with woodfired pizza.',
    });
    expect(place.description, 'A historic Dutch colonial property with woodfired pizza.');
    expect(place.toJson()['description'], 'A historic Dutch colonial property with woodfired pizza.');
  });

  test('extracts place description and backfills guideCardsFor', () {
    const reply = '''Here are some spots:
- **Doctor's House**: Historic property with great pizza and garden vibe.
- **Dots Bay House** (0.8 km): Beachfront surf lodge with acoustic sessions.''';

    final cards = guideCardsFor(
      [const GuidePlace(id: 'p1', name: 'Doctor\'s House')],
      const [],
      const [],
      reply,
    );
    expect(cards.first.description, 'Historic property with great pizza and garden vibe.');
  });

  test('parsePlaceTurnContent cleanly separates intro, place descriptions, and outro', () {
    const rawText = '''Since it's rainy out today, here are some great cozy spots to check out:

- **Doctor's House**: A historic Dutch colonial property with woodfired pizza and craft beer.
- **Dots Bay House**: Beachfront surf hangout with healthy smoothie bowls.

Would you prefer something right on the beach or a quieter spot inland?''';

    final places = [
      const GuidePlace(id: 'p1', name: 'Doctor\'s House'),
      const GuidePlace(id: 'p2', name: 'Dots Bay House'),
    ];

    final turn = parsePlaceTurnContent(rawText, places);
    expect(turn.intro, 'Since it\'s rainy out today, here are some great cozy spots to check out:');
    expect(turn.placeDescriptions['Doctor\'s House'], 'A historic Dutch colonial property with woodfired pizza and craft beer.');
    expect(turn.placeDescriptions['Dots Bay House'], 'Beachfront surf hangout with healthy smoothie bowls.');
    expect(turn.outro, 'Would you prefer something right on the beach or a quieter spot inland?');
  });

  test('parsePlaceTurnContent preserves outro statement without question mark and without blank line', () {
    const rawText = '''Here are a couple recommendations:
- **Doctor's House**: Woodfired pizza and beach garden.
- **Dots Bay House**: Surf lodge right on the beach.
Hope you have a wonderful time in Mirissa!''';

    final places = [
      const GuidePlace(id: 'p1', name: 'Doctor\'s House'),
      const GuidePlace(id: 'p2', name: 'Dots Bay House'),
    ];

    final turn = parsePlaceTurnContent(rawText, places);
    expect(turn.intro, 'Here are a couple recommendations:');
    expect(turn.placeDescriptions['Doctor\'s House'], 'Woodfired pizza and beach garden.');
    expect(turn.placeDescriptions['Dots Bay House'], 'Surf lodge right on the beach.');
    expect(turn.outro, 'Hope you have a wonderful time in Mirissa!');
  });

  test('extractPlaceDescriptionFromText handles curly apostrophes and markdown links', () {
    const text = '''Recommendations:
- **Doctor’s House**: Vibrant beachfront bar.
- [Sea Breeze](https://example.com): Cozy terrace.''';

    expect(extractPlaceDescriptionFromText(text, 'Doctor\'s House'), 'Vibrant beachfront bar.');
    expect(extractPlaceDescriptionFromText(text, 'Sea Breeze'), 'Cozy terrace.');
  });

  testWidgets('GuidePlaceCard renders description and location region badge inside unified card', (tester) async {
    const place = GuidePlace(
      id: 'plc_1',
      name: 'Doctor\'s House',
      category: 'Restaurant',
      region: 'Mirissa',
      description: 'A **historic** Dutch colonial property with woodfired pizzas.',
      rating: 4.8,
    );

    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(
          home: Scaffold(
            body: GuidePlaceCard(place: place),
          ),
        ),
      ),
    );

    expect(find.text('Doctor\'s House'), findsOneWidget);
    // Verifies location region badge is displayed
    expect(find.text('Mirissa'), findsOneWidget);
    expect(find.byIcon(Icons.place_outlined), findsOneWidget);
    // Verifies inline markdown bold is cleaned up cleanly
    expect(find.text('A historic Dutch colonial property with woodfired pizzas.'), findsOneWidget);
    expect(find.byIcon(Icons.format_quote_rounded), findsOneWidget);
  });

  testWidgets('GuidePlaceCard renders vibe suitability badge when present', (tester) async {
    const place = GuidePlace(
      id: 'plc_vibe',
      name: 'Nomad Oasis',
      category: 'Cafe',
      vibeMatchPercent: 92,
      vibeBadge: '92% Vibe',
    );

    await tester.pumpWidget(
      const ProviderScope(
        child: MaterialApp(
          home: Scaffold(
            body: GuidePlaceCard(place: place),
          ),
        ),
      ),
    );

    expect(find.text('92% Vibe'), findsOneWidget);
    expect(find.byIcon(Icons.bolt_rounded), findsOneWidget);
  });

  test('GuidePlace parses vibeMatchPercent and vibeBadge from JSON', () {
    final card = GuidePlace.fromJson({
      'id': 'plc_1',
      'name': 'Nomad Hub',
      'vibeMatchPercent': 88,
      'vibeBadge': '88% Vibe',
    });
    expect(card.vibeMatchPercent, 88);
    expect(card.vibeBadge, '88% Vibe');
    expect(card.toJson()['vibeMatchPercent'], 88);
    expect(card.toJson()['vibeBadge'], '88% Vibe');
  });
}


