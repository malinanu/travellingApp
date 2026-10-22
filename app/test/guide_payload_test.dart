// Local Guide trip-planner cards: parsing the server's payloads, the
// message sent when the question card is answered, and adding a plan's
// stops to a trip (new or current) on the right days.

import 'package:flutter_test/flutter_test.dart';
import 'package:spotto/data/mock/trip_catalog.dart';
import 'package:spotto/features/traveller/guide/guide_payload.dart';

Map<String, dynamic> _place(String ref, String name, {String? placeId}) => {
      'ref': ref,
      'placeId': ?placeId,
      if (placeId == null) 'geoPlaceId': ref.substring(2),
      'name': name,
      'kind': 'see',
      'rating': 4.7,
      'ratingCount': 30000,
      'lat': 7.957,
      'lng': 80.76,
      'photoUrl': '/geo-places/${ref.substring(2)}/photo',
    };

final _planJson = {
  'type': 'plan',
  'plan': {
    'id': 'gplan_1',
    'title': '2 nights in Sigiriya',
    'startDate': '2026-11-12',
    'nights': 2,
    'stay': 'Hotel Sigiriya',
    'days': [
      {
        'title': 'Fortress',
        'date': '2026-11-12',
        'stops': [
          {'time': '07:00', 'durationMin': 180, 'note': 'Go early.', 'place': _place('g:gp_1', 'Sigiriya Rock')},
          {'time': '13:00', 'note': 'Lunch.', 'place': _place('p:cafe1', 'Rastha Cafe', placeId: 'cafe1')},
        ],
      },
      {
        'date': '2026-11-13',
        'stops': [
          {'time': '05:30', 'note': 'Sunrise.', 'place': _place('g:gp_2', 'Pidurangala Rock')},
        ],
      },
    ],
    'tips': ['Carry water'],
  },
};

void main() {
  test('parses each card type and ignores unknown ones', () {
    final plan = GuidePayload.tryParse(_planJson)!;
    expect(plan.type, 'plan');
    expect(plan.plan!.days.length, 2);
    expect(plan.plan!.days.first.stops.first.place.asGeoPlace!.photoUrl, '/geo-places/gp_1/photo');
    expect(plan.plan!.days.first.stops[1].place.asGeoPlace, isNull); // catalog place

    final q = GuidePayload.tryParse({
      'type': 'questions',
      'questions': {
        'questions': [
          {'id': 'dates', 'text': 'When are you going?', 'kind': 'dates'},
          {'id': 'hotel', 'text': 'Booked a hotel?', 'kind': 'text', 'options': ['Not yet']},
        ],
      },
    })!;
    expect(q.questions.map((e) => e.kind), ['dates', 'text']);

    expect(GuidePayload.tryParse({'type': 'mystery'}), isNull);
    expect(GuidePayload.tryParse(null), isNull);
    expect(GuidePayload.tryParse({'type': 'plan', 'plan': 'broken'}), isNull);
  });

  test('answers become one message, skipping unanswered questions', () {
    const qs = [
      GuideQuestion(id: 'nights', text: 'How many nights?', kind: 'single'),
      GuideQuestion(id: 'hotel', text: 'Booked a hotel?', kind: 'text'),
      GuideQuestion(id: 'likes', text: 'What do you enjoy?', kind: 'multi'),
    ];
    expect(
      composeGuideAnswers(qs, {
        'nights': ['2 nights'],
        'hotel': [' '],
        'likes': ['Hiking', 'Food'],
      }),
      'How many nights? 2 nights\nWhat do you enjoy? Hiking, Food',
    );
  });

  test('parseClock', () {
    expect(parseClock('07:30'), 450);
    expect(parseClock('7:05'), 425);
    expect(parseClock('24:00'), isNull);
    expect(parseClock(''), isNull);
  });

  test('a new trip gets every stop on its plan day, timed and located', () {
    final action = GuideCalendarAction.fromPlan(GuidePayload.tryParse(_planJson)!.plan!);
    final trip = guideNewTrip(action, '2026-11-12');
    expect(trip.name, '2 nights in Sigiriya');
    expect(trip.days.length, 2);
    expect(trip.days[0].stops.map((s) => s.custom!.name), ['Sigiriya Rock', 'Rastha Cafe']);
    expect(trip.days[0].stops.first.time, 7 * 60);
    expect(trip.days[0].stops[1].placeId, 'cafe1');
    expect(trip.days[1].stops.single.custom!.lat, 7.957);
  });

  test('adding to the current trip matches dates, grows the trip, skips duplicates', () {
    const current = Trip(
      name: 'Sri Lanka',
      code: 'TRP-ABC',
      startDate: '2026-11-10',
      collaborators: [],
      days: [TripDay(id: 'd1', stops: []), TripDay(id: 'd2', stops: [])],
    );
    final action = GuideCalendarAction.fromPlan(GuidePayload.tryParse(_planJson)!.plan!);
    final once = guideAddStops(current, action);
    // The plan starts 12 Nov = day 3 of a trip starting 10 Nov.
    expect(once.days.length, 4);
    expect(once.days[2].stops.length, 2);
    expect(once.days[3].stops.single.custom!.name, 'Pidurangala Rock');
    expect(guideAddStops(once, action).days[2].stops.length, 2);
  });

  test('one place from the plan becomes a one-stop action on its day', () {
    final plan = GuidePayload.tryParse(_planJson)!.plan!;
    final only = GuideCalendarAction.fromPlan(plan, only: [
      GuideCalendarStop(dayIndex: 1, time: '05:30', place: plan.days[1].stops.single.place),
    ]);
    expect(only.stops.length, 1);
    expect(only.dayCount, 2);
  });
}
