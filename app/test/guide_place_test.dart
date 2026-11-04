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
}

