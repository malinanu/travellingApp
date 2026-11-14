// Time-of-day logic for the Discover map: whether a place is open during the
// Morning / Afternoon / Evening slot the traveller picked, and whether it's a
// music spot. Pure functions over what the server sends for a place.

import '../../data/models/models.dart';

/// Minutes-from-midnight windows for each Discover time slot.
const placeTimeSlots = <String, (int, int)>{
  'morning': (6 * 60, 12 * 60),
  'afternoon': (12 * 60, 17 * 60),
  'evening': (17 * 60, 24 * 60),
};

/// Today's opening window in minutes from midnight. [close] is past 1440
/// when the place stays open over midnight.
typedef PlaceHours = ({int open, int close});

final _clock = RegExp(r'(\d{1,2})(?::(\d{2}))?\s*([AaPp]\.?[Mm]\.?)?');

int? _minutes(RegExpMatch m) {
  var h = int.parse(m.group(1)!);
  final min = int.tryParse(m.group(2) ?? '0') ?? 0;
  final meridiem = (m.group(3) ?? '').replaceAll('.', '').toLowerCase();
  if (min > 59) return null;
  if (meridiem.isEmpty) {
    if (h > 24) return null;
  } else {
    if (h < 1 || h > 12) return null;
    if (meridiem == 'pm' && h != 12) h += 12;
    if (meridiem == 'am' && h == 12) h = 0;
  }
  return h * 60 + min;
}

/// Parses a place's hours label ("9:00 AM – 5:00 PM", "18:00-02:00",
/// "Open 24 hours"). Null when the label has no usable times.
PlaceHours? parsePlaceHours(String label) {
  final text = label.trim();
  if (text.isEmpty) return null;
  final lower = text.toLowerCase();
  if (lower.contains('24 hours') || lower.contains('24/7')) return (open: 0, close: 24 * 60);
  final times = _clock.allMatches(text).map(_minutes).toList();
  if (times.length < 2 || times[0] == null || times[1] == null) return null;
  final open = times[0]!;
  var close = times[1]!;
  if (close <= open) close += 24 * 60; // through midnight
  return (open: open, close: close);
}

/// Whether a place with this hours label is open at some point in [slot].
/// Null when the label doesn't say (unknown hours).
bool? placeOpenIn(String hoursLabel, String slot) {
  final window = placeTimeSlots[slot];
  if (window == null) return null;
  if (hoursLabel.trim().toLowerCase().startsWith('closed')) return false;
  final hours = parsePlaceHours(hoursLabel);
  if (hours == null) return null;
  bool overlaps(int from, int to) => hours.open < to && from < hours.close;
  // The second check catches a place that opened yesterday evening and is
  // still open after midnight.
  return overlaps(window.$1, window.$2) || overlaps(window.$1 + 24 * 60, window.$2 + 24 * 60);
}

/// Whether the Discover map shows [p] for [slot]. A place with known hours
/// shows only when it's open in that slot; one whose hours we don't know
/// stays on the map rather than vanishing.
bool placeShownAt(Place p, String slot) => placeOpenIn(p.hours, slot) ?? true;

const _musicCategories = {'live-music', 'nightlife'};
final _musicWord = RegExp(r'\b(live music|music|dj|band|karaoke|jazz|concert)\b', caseSensitive: false);

/// Whether [p] is a music spot: a live-music / nightlife venue, or tagged so.
bool placeHasMusic(Place p) =>
    _musicCategories.contains(p.categoryId) ||
    _musicCategories.contains(p.type) ||
    p.tags.any(_musicWord.hasMatch);
