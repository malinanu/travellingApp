// Discover time-of-day filter and music spots (core/utils/place_time.dart).

import 'package:flutter_test/flutter_test.dart';
import 'package:spotto/core/utils/place_time.dart';
import 'package:spotto/data/models/models.dart';
import 'package:spotto/data/repositories/repositories.dart' show timeSlotFor;

Place _place({String hours = '', String categoryId = '', List<String> tags = const []}) => Place.fromJson({
      'id': 'p1',
      'name': 'Test place',
      'lat': 6.9,
      'lng': 79.86,
      'hours': hours,
      'categoryId': categoryId,
      'tags': tags,
    });

void main() {
  test('parses the server hours label, 24h times and all-day', () {
    expect(parsePlaceHours('9:00 AM – 5:00 PM'), (open: 9 * 60, close: 17 * 60));
    expect(parsePlaceHours('18:00-02:00'), (open: 18 * 60, close: 26 * 60));
    expect(parsePlaceHours('12:00 PM – 12:00 AM'), (open: 12 * 60, close: 24 * 60));
    expect(parsePlaceHours('Open 24 hours'), (open: 0, close: 24 * 60));
    expect(parsePlaceHours(''), isNull);
    expect(parsePlaceHours('Call for times'), isNull);
  });

  test('a day cafe is open morning and afternoon, not evening', () {
    const hours = '7:00 AM – 4:00 PM';
    expect(placeOpenIn(hours, 'morning'), isTrue);
    expect(placeOpenIn(hours, 'afternoon'), isTrue);
    expect(placeOpenIn(hours, 'evening'), isFalse);
  });

  test('a bar open over midnight is evening only', () {
    const hours = '6:00 PM – 2:00 AM';
    expect(placeOpenIn(hours, 'morning'), isFalse);
    expect(placeOpenIn(hours, 'afternoon'), isFalse);
    expect(placeOpenIn(hours, 'evening'), isTrue);
  });

  test('closed today hides everywhere; unknown hours stay on the map', () {
    expect(placeOpenIn('Closed today — Poya', 'evening'), isFalse);
    expect(placeOpenIn('', 'evening'), isNull);
    expect(placeShownAt(_place(hours: 'Closed today'), 'morning'), isFalse);
    expect(placeShownAt(_place(), 'morning'), isTrue);
    expect(placeShownAt(_place(hours: '7:00 AM – 4:00 PM'), 'evening'), isFalse);
  });

  test('music spots come from category or tags', () {
    expect(placeHasMusic(_place(categoryId: 'live-music')), isTrue);
    expect(placeHasMusic(_place(categoryId: 'nightlife')), isTrue);
    expect(placeHasMusic(_place(tags: ['Sea view', 'Live music'])), isTrue);
    expect(placeHasMusic(_place(tags: ['DJ nights'])), isTrue);
    expect(placeHasMusic(_place(categoryId: 'cafe', tags: ['Wifi'])), isFalse);
  });

  test('the filter starts on the slot it is now', () {
    expect(timeSlotFor(DateTime(2026, 11, 14, 9)), 'morning');
    expect(timeSlotFor(DateTime(2026, 11, 14, 14)), 'afternoon');
    expect(timeSlotFor(DateTime(2026, 11, 14, 21)), 'evening');
    expect(timeSlotFor(DateTime(2026, 11, 14, 1)), 'evening');
  });
}
