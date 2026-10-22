// Local Guide cards — mirror the payloads in server/internal/http/guide_planner.go.
// A reply can carry one: trip questions to answer, a visual trip plan, or
// places to add to the Trip calendar. Also the pure logic behind them:
// composing the answers message and merging stops into a trip.

import '../../../data/mock/trip_catalog.dart';
import '../../../data/models/inspiration.dart';

double? _d(Object? v) => (v as num?)?.toDouble();
String _s(Object? v) => v as String? ?? '';
List<Map<String, dynamic>> _maps(Object? v) => [
      for (final e in (v as List? ?? const []))
        if (e is Map) e.cast<String, dynamic>(),
    ];

class GuideQuestion {
  final String id;
  final String text;

  /// single | multi | text | dates
  final String kind;
  final List<String> options;
  const GuideQuestion({required this.id, required this.text, required this.kind, this.options = const []});

  factory GuideQuestion.fromJson(Map<String, dynamic> j) => GuideQuestion(
        id: _s(j['id']),
        text: _s(j['text']),
        kind: _s(j['kind']).isEmpty ? 'text' : _s(j['kind']),
        options: (j['options'] as List? ?? const []).whereType<String>().toList(),
      );
}

class GuidePlanPlace {
  final String ref;
  final String? placeId; // Spotto catalog place
  final String? geoPlaceId; // place found on Google
  final String name;
  final String kind;
  final String address;
  final double? rating;
  final int ratingCount;
  final double lat;
  final double lng;
  final String? photoUrl;
  final String mapsUrl;
  const GuidePlanPlace({
    required this.ref,
    this.placeId,
    this.geoPlaceId,
    required this.name,
    this.kind = '',
    this.address = '',
    this.rating,
    this.ratingCount = 0,
    required this.lat,
    required this.lng,
    this.photoUrl,
    this.mapsUrl = '',
  });

  factory GuidePlanPlace.fromJson(Map<String, dynamic> j) => GuidePlanPlace(
        ref: _s(j['ref']),
        placeId: (j['placeId'] as String?)?.isNotEmpty == true ? j['placeId'] as String : null,
        geoPlaceId: (j['geoPlaceId'] as String?)?.isNotEmpty == true ? j['geoPlaceId'] as String : null,
        name: _s(j['name']),
        kind: _s(j['kind']),
        address: _s(j['address']),
        rating: _d(j['rating']),
        ratingCount: (j['ratingCount'] as num?)?.toInt() ?? 0,
        lat: _d(j['lat']) ?? 0,
        lng: _d(j['lng']) ?? 0,
        photoUrl: (j['photoUrl'] as String?)?.isNotEmpty == true ? j['photoUrl'] as String : null,
        mapsUrl: _s(j['mapsUrl']),
      );

  /// Spotto's simple kind (eat/see/…) — catalog places send a category label.
  String get simpleKind => placeKinds.contains(kind) ? kind : 'see';

  /// As a Google-found place, for the shared board widgets (photo, Maps link).
  GeoPlace? get asGeoPlace => geoPlaceId == null
      ? null
      : GeoPlace(
          id: geoPlaceId!,
          name: name,
          lat: lat,
          lng: lng,
          address: address,
          kind: simpleKind,
          rating: rating,
          ratingCount: ratingCount,
          mapsUrl: mapsUrl,
          photoUrl: photoUrl,
        );
}

class GuidePlanStop {
  final String time; // "HH:mm", may be empty
  final int durationMin;
  final String note;
  final GuidePlanPlace place;
  const GuidePlanStop({this.time = '', this.durationMin = 0, this.note = '', required this.place});

  factory GuidePlanStop.fromJson(Map<String, dynamic> j) => GuidePlanStop(
        time: _s(j['time']),
        durationMin: (j['durationMin'] as num?)?.toInt() ?? 0,
        note: _s(j['note']),
        place: GuidePlanPlace.fromJson((j['place'] as Map? ?? const {}).cast<String, dynamic>()),
      );
}

class GuidePlanDay {
  final String title;
  final String? date;
  final List<GuidePlanStop> stops;
  const GuidePlanDay({this.title = '', this.date, this.stops = const []});

  factory GuidePlanDay.fromJson(Map<String, dynamic> j) => GuidePlanDay(
        title: _s(j['title']),
        date: (j['date'] as String?)?.isNotEmpty == true ? j['date'] as String : null,
        stops: [for (final s in _maps(j['stops'])) GuidePlanStop.fromJson(s)],
      );
}

class GuidePlan {
  final String id;
  final String title;
  final String area;
  final String? startDate;
  final int nights;
  final String stay;
  final List<GuidePlanDay> days;
  final List<String> tips;
  const GuidePlan({
    required this.id,
    required this.title,
    this.area = '',
    this.startDate,
    this.nights = 0,
    this.stay = '',
    this.days = const [],
    this.tips = const [],
  });

  factory GuidePlan.fromJson(Map<String, dynamic> j) => GuidePlan(
        id: _s(j['id']),
        title: _s(j['title']),
        area: _s(j['area']),
        startDate: (j['startDate'] as String?)?.isNotEmpty == true ? j['startDate'] as String : null,
        nights: (j['nights'] as num?)?.toInt() ?? 0,
        stay: _s(j['stay']),
        days: [for (final d in _maps(j['days'])) GuidePlanDay.fromJson(d)],
        tips: (j['tips'] as List? ?? const []).whereType<String>().toList(),
      );
}

class GuideCalendarStop {
  final int dayIndex;
  final String time;
  final GuidePlanPlace place;
  const GuideCalendarStop({required this.dayIndex, this.time = '', required this.place});

  factory GuideCalendarStop.fromJson(Map<String, dynamic> j) => GuideCalendarStop(
        dayIndex: (j['dayIndex'] as num?)?.toInt() ?? 0,
        time: _s(j['time']),
        place: GuidePlanPlace.fromJson((j['place'] as Map? ?? const {}).cast<String, dynamic>()),
      );
}

/// Places to add to the Trip calendar. [mode] is new | current | ask.
class GuideCalendarAction {
  final String? planId;
  final String mode;
  final String title;
  final String? startDate;
  final int dayCount;
  final List<GuideCalendarStop> stops;
  const GuideCalendarAction({
    this.planId,
    this.mode = 'ask',
    required this.title,
    this.startDate,
    required this.dayCount,
    required this.stops,
  });

  factory GuideCalendarAction.fromJson(Map<String, dynamic> j) {
    final stops = [for (final s in _maps(j['stops'])) GuideCalendarStop.fromJson(s)];
    return GuideCalendarAction(
      planId: j['planId'] as String?,
      mode: _s(j['mode']).isEmpty ? 'ask' : _s(j['mode']),
      title: _s(j['title']).isEmpty ? 'My trip' : _s(j['title']),
      startDate: (j['startDate'] as String?)?.isNotEmpty == true ? j['startDate'] as String : null,
      dayCount: _dayCount((j['dayCount'] as num?)?.toInt() ?? 0, stops),
      stops: stops,
    );
  }

  /// The whole plan, or only [only] of its stops (the "+" on one stop).
  factory GuideCalendarAction.fromPlan(GuidePlan plan, {List<GuideCalendarStop>? only}) {
    final stops = only ??
        [
          for (var i = 0; i < plan.days.length; i++)
            for (final s in plan.days[i].stops) GuideCalendarStop(dayIndex: i, time: s.time, place: s.place),
        ];
    return GuideCalendarAction(
      planId: plan.id,
      title: plan.title.isEmpty ? 'My trip' : plan.title,
      startDate: plan.startDate,
      dayCount: _dayCount(only == null ? plan.days.length : 0, stops),
      stops: stops,
    );
  }

  static int _dayCount(int given, List<GuideCalendarStop> stops) {
    var n = given;
    for (final s in stops) {
      if (s.dayIndex + 1 > n) n = s.dayIndex + 1;
    }
    return n < 1 ? 1 : n;
  }
}

/// The card carried by a guide reply.
class GuidePayload {
  final String type; // questions | plan | calendar
  final List<GuideQuestion> questions;
  final GuidePlan? plan;
  final GuideCalendarAction? calendar;
  const GuidePayload({required this.type, this.questions = const [], this.plan, this.calendar});

  static GuidePayload? tryParse(Object? raw) {
    if (raw is! Map) return null;
    final j = raw.cast<String, dynamic>();
    try {
      switch (j['type']) {
        case 'questions':
          final qs = [
            for (final q in _maps((j['questions'] as Map?)?['questions'])) GuideQuestion.fromJson(q),
          ].where((q) => q.text.isNotEmpty).toList();
          return qs.isEmpty ? null : GuidePayload(type: 'questions', questions: qs);
        case 'plan':
          final plan = GuidePlan.fromJson((j['plan'] as Map).cast<String, dynamic>());
          return plan.days.isEmpty ? null : GuidePayload(type: 'plan', plan: plan);
        case 'calendar':
          final cal = GuideCalendarAction.fromJson((j['calendar'] as Map).cast<String, dynamic>());
          return cal.stops.isEmpty ? null : GuidePayload(type: 'calendar', calendar: cal);
      }
    } catch (_) {
      // A card this build can't read: the reply's text still shows.
    }
    return null;
  }
}

/// The message sent when the traveller answers a question card: one
/// "question answer" line per answered question.
String composeGuideAnswers(List<GuideQuestion> questions, Map<String, List<String>> answers) {
  final lines = <String>[];
  for (final q in questions) {
    final a = (answers[q.id] ?? const []).map((s) => s.trim()).where((s) => s.isNotEmpty).toList();
    if (a.isNotEmpty) lines.add('${q.text} ${a.join(', ')}');
  }
  return lines.join('\n');
}

/// "07:30" → 450 minutes, or null.
int? parseClock(String hhmm) {
  final m = RegExp(r'^(\d{1,2}):(\d{2})$').firstMatch(hhmm.trim());
  if (m == null) return null;
  final h = int.parse(m.group(1)!), min = int.parse(m.group(2)!);
  if (h > 23 || min > 59) return null;
  return h * 60 + min;
}

DateTime? _date(String? iso) => iso == null ? null : DateTime.tryParse(iso);

/// A new trip holding [action]'s stops, starting on [startDate].
Trip guideNewTrip(GuideCalendarAction action, String startDate) => guideAddStops(
      Trip(
        name: action.title,
        code: newTripCode(),
        startDate: startDate,
        collaborators: const [],
        days: [for (var i = 0; i < action.dayCount; i++) TripDay(id: tripNewId('d'), stops: const [])],
      ),
      action,
      planStart: startDate,
    );

/// [trip] with [action]'s stops added. Stops land on the trip day with the
/// same date when both have dates (so a plan for 12 Nov goes on 12 Nov),
/// otherwise on the plan's own day number; the trip grows to fit. A place
/// already on that day isn't added twice.
Trip guideAddStops(Trip trip, GuideCalendarAction action, {String? planStart}) {
  final tripStart = _date(trip.startDate);
  final start = _date(planStart ?? action.startDate);
  final offset = (tripStart != null && start != null)
      ? DateTime(start.year, start.month, start.day).difference(DateTime(tripStart.year, tripStart.month, tripStart.day)).inDays
      : 0;
  final days = [...trip.days];
  for (final s in action.stops) {
    final dayIndex = (s.dayIndex + offset).clamp(0, 365);
    while (days.length <= dayIndex) {
      days.add(TripDay(id: tripNewId('d'), stops: const []));
    }
    final existing = days[dayIndex].stops;
    final name = s.place.name.toLowerCase();
    if (existing.any((e) => (e.custom?.name.toLowerCase() ?? '') == name ||
        (s.place.placeId != null && e.placeId == s.place.placeId))) {
      continue;
    }
    final time = parseClock(s.time) ?? (existing.isEmpty ? 9 * 60 : (existing.last.time + 90).clamp(0, 23 * 60 + 30));
    final stop = TripStop(
      id: tripNewId('s'),
      placeId: s.place.placeId ?? '',
      time: time,
      by: 'me',
      custom: CustomStopInfo(
        name: s.place.name,
        type: kindLabel(s.place.simpleKind),
        icon: kindIcon(s.place.simpleKind),
        lat: s.place.lat,
        lng: s.place.lng,
      ),
    );
    days[dayIndex] = days[dayIndex].copyWith(stops: [...existing, stop]..sort((a, b) => a.time.compareTo(b.time)));
  }
  return trip.copyWith(days: days);
}
