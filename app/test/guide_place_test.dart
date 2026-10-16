// Local Guide place cards: the server's summaries become cards with a
// "0.8 km · $$ · ★ 4.6" line and a directions link; a reply that only
// carries ids (older servers, older history) falls back to cached places.

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
    expect(card.metaLine, r'0.8 km · $$ · ★ 4.6');
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
}
