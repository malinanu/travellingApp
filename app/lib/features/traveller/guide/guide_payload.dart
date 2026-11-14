// Local Guide cards — mirror the payloads in server/internal/http/guide_planner.go.
// A reply can carry one: trip questions to answer, a visual trip plan, or
// places to add to the Trip calendar. Also the pure logic behind them:
// composing the answers message and merging stops into a trip.

import '../../../data/catalog/trip_catalog.dart';
import '../../../data/models/inspiration.dart';

double? _d(Object? v) => (v as num?)?.toDouble();
int? _tripDay(Object? v) {
  final n = (v as num?)?.toInt() ?? 0;
  return n >= 1 ? n : null;
}
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
  String get simpleKind => guideSimpleKind(kind, name);

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

  /// 1-based day of the traveller's existing trip this plan starts on, when
  /// they named one ("plan my 2nd day").
  final int? tripDay;
  const GuidePlan({
    required this.id,
    required this.title,
    this.area = '',
    this.startDate,
    this.nights = 0,
    this.stay = '',
    this.days = const [],
    this.tips = const [],
    this.tripDay,
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
        tripDay: _tripDay(j['tripDay']),
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

  /// Hints for which day of the current trip the stops belong on: the
  /// 1-based trip day the traveller named, the plan's area, and its day
  /// titles ("Day 2 in Sigiriya").
  final int? tripDay;
  final String area;
  final List<String> dayTitles;
  const GuideCalendarAction({
    this.planId,
    this.mode = 'ask',
    required this.title,
    this.startDate,
    required this.dayCount,
    required this.stops,
    this.tripDay,
    this.area = '',
    this.dayTitles = const [],
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
      tripDay: _tripDay(j['tripDay']),
      area: _s(j['area']),
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
      tripDay: plan.tripDay,
      area: plan.area,
      dayTitles: [for (final d in plan.days) d.title],
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
/// already on that day isn't added twice. [startDay] (0-based) puts the
/// action's first day on that trip day instead, whatever the dates say.
Trip guideAddStops(Trip trip, GuideCalendarAction action, {String? planStart, int? startDay}) {
  final offset = startDay != null
      ? startDay - guideFirstDayIndex(action)
      : (guideDateOffset(trip, planStart ?? action.startDate) ?? 0);
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

/// The lowest plan day the action's stops use (0 for a whole plan).
int guideFirstDayIndex(GuideCalendarAction action) {
  var first = action.stops.isEmpty ? 0 : action.stops.first.dayIndex;
  for (final s in action.stops) {
    if (s.dayIndex < first) first = s.dayIndex;
  }
  return first;
}

/// How many plan days the action's stops spread over.
int guideDaySpan(GuideCalendarAction action) {
  var last = 0;
  for (final s in action.stops) {
    if (s.dayIndex > last) last = s.dayIndex;
  }
  return last - guideFirstDayIndex(action) + 1;
}

/// Days from the trip's start to [planStart], when both are dated and the
/// plan doesn't start before the trip. Null when dates can't place it.
int? guideDateOffset(Trip trip, String? planStart) {
  final tripStart = _date(trip.startDate);
  final start = _date(planStart);
  if (tripStart == null || start == null) return null;
  final days = DateTime(start.year, start.month, start.day)
      .difference(DateTime(tripStart.year, tripStart.month, tripStart.day))
      .inDays;
  return days < 0 ? null : days;
}

final _dayNumber = RegExp(r'\bday\s*(\d{1,2})\b', caseSensitive: false);

/// The trip day (0-based) [action]'s first day most likely belongs on, or
/// null when nothing points to one. In order: matching dates, the trip day
/// the traveller named, "Day N" in the plan's titles, then a trip day whose
/// location is the plan's area.
int? guideSuggestDay(Trip trip, GuideCalendarAction action) {
  final first = guideFirstDayIndex(action);
  final byDate = guideDateOffset(trip, action.startDate);
  if (byDate != null && byDate + first <= trip.days.length) return byDate + first;
  if (action.tripDay != null) return action.tripDay! - 1 + first;

  final titles = [
    if (first < action.dayTitles.length) action.dayTitles[first],
    action.title,
  ];
  for (final t in titles) {
    final m = _dayNumber.firstMatch(t);
    if (m != null) {
      final n = int.parse(m.group(1)!);
      if (n >= 1 && n <= trip.days.length + 1) return n - 1;
    }
  }

  final haystack = '${action.area} ${titles.join(' ')}'.toLowerCase();
  for (var i = 0; i < trip.days.length; i++) {
    final loc = (trip.days[i].location ?? '').trim().toLowerCase();
    // Whole words: "Ella" must not match inside "Wellawaya".
    if (loc.length >= 3 && RegExp('(^|[^a-z])${RegExp.escape(loc)}(\$|[^a-z])').hasMatch(haystack)) return i;
  }
  // The guide placed these on a later day itself ("put it on day 2").
  return first > 0 ? first : null;
}

const _kindWords = {
  'eat': ['restaurant', 'cafe', 'café', 'coffee', 'bakery', 'food', 'kitchen', 'diner', 'eatery', 'bistro', 'grill'],
  'drink': ['bar', 'pub', 'lounge', 'brewery', 'nightclub', 'cocktail', 'wine'],
  'stay': ['hotel', 'villa', 'resort', 'hostel', 'lodge', 'guesthouse', 'guest house', 'homestay', 'lodging'],
  'shop': ['shop', 'store', 'market', 'mall', 'boutique', 'supermarket'],
  'do': ['surf', 'safari', 'tour', 'hike', 'hiking', 'trek', 'diving', 'snorkel', 'spa', 'yoga', 'class', 'activity'],
};

/// Spotto's simple kind for a guide place. [kind] may already be one, or a
/// category label ("Restaurant", "landmark"); the name breaks ties.
String guideSimpleKind(String kind, [String name = '']) {
  if (placeKinds.contains(kind)) return kind;
  for (final text in [kind.toLowerCase(), name.toLowerCase()]) {
    if (text.isEmpty) continue;
    final words = text.split(RegExp(r'[^a-zà-ÿ]+'));
    for (final e in _kindWords.entries) {
      for (final w in e.value) {
        if (w.contains(' ') ? text.contains(w) : words.contains(w)) return e.key;
      }
    }
  }
  return 'see';
}
