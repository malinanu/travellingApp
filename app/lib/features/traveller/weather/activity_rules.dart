// Pure Dart activity-suitability engine for the Weather screen — no
// Flutter import, unit-testable on its own. Resolves which "category" a
// traveller cares about from their onboarding prefs, then scores each
// forecast hour for that category from the fields Tomorrow.io actually
// gives us (no swell/tide data, so surf is wind+rain only — see
// categoryFootnote).

import 'package:intl/intl.dart';

import '../../../data/models/models.dart';

enum ActivityCategory { surf, hike, beach, general }

enum Suitability { good, fair, poor }

class HourRating {
  final DateTime time;
  final Suitability level;
  final int score;
  final String reason; // empty when good
  const HourRating({required this.time, required this.level, required this.score, required this.reason});
}

// Interests are multi-select and more specific than the single onboarding
// style, so they win; fixed order makes the winner deterministic
// regardless of pill-tap order.
const _interestPriority = ['surf', 'hiking', 'beach'];

ActivityCategory resolveCategory(UserPrefs? prefs) {
  if (prefs != null) {
    for (final id in _interestPriority) {
      if (prefs.interests.contains(id)) return _categoryForInterest(id);
    }
    if (prefs.styles.contains('surf')) return ActivityCategory.surf;
  }
  return ActivityCategory.general;
}

ActivityCategory _categoryForInterest(String id) {
  switch (id) {
    case 'surf':
      return ActivityCategory.surf;
    case 'hiking':
      return ActivityCategory.hike;
    case 'beach':
      return ActivityCategory.beach;
    default:
      return ActivityCategory.general;
  }
}

String categoryLabel(ActivityCategory c) {
  switch (c) {
    case ActivityCategory.surf:
      return 'Surf window';
    case ActivityCategory.hike:
      return 'Hiking conditions';
    case ActivityCategory.beach:
      return 'Beach conditions';
    case ActivityCategory.general:
      return 'Outdoor conditions';
  }
}

/// Icon name from the app's existing spotto_icons vocabulary.
String categoryIconName(ActivityCategory c) {
  switch (c) {
    case ActivityCategory.surf:
      return 'Waves';
    case ActivityCategory.hike:
      return 'Mountain';
    case ActivityCategory.beach:
      return 'Palmtree';
    case ActivityCategory.general:
      return 'Compass';
  }
}

String? categoryFootnote(ActivityCategory c) =>
    c == ActivityCategory.surf ? 'Wind and rain only — no swell data yet.' : null;

/// Additive-penalty scoring: 100 minus the sum of triggered penalties,
/// banded into good/fair/poor. The reason reported is whichever single
/// penalty contributed most, so the "why" falls out of the scoring instead
/// of needing a separate explanation table.
HourRating rateHour(ActivityCategory category, HourlyPoint h) {
  final reasons = <String, int>{};
  var penalty = 0;
  void add(String reason, int amount) {
    if (amount <= 0) return;
    penalty += amount;
    reasons[reason] = (reasons[reason] ?? 0) + amount;
  }

  switch (category) {
    case ActivityCategory.surf:
      if (h.windSpeed > 28) {
        add('too windy, bumpy faces', 60);
      } else if (h.windSpeed > 18) {
        add('breezy, choppier faces', 35);
      } else if (h.windSpeed > 10) {
        add('light wind', 15);
      }
      if (h.precipProbability > 60) {
        add('showers', 25);
      } else if (h.precipProbability > 30) {
        add('showers', 10);
      }
      if (h.temp < 22) add('chilly water', 10);
      if (h.uvIndex >= 9) add('strong sun', 5);
      break;

    case ActivityCategory.hike:
      if (h.windSpeed > 35) add('exposed and gusty', 25);
      if (h.precipProbability > 70) {
        add('wet trail', 65);
      } else if (h.precipProbability > 45) {
        add('wet trail', 40);
      } else if (h.precipProbability > 20) {
        add('showers possible', 20);
      }
      if (h.temp > 34) {
        add('dangerous heat', 55);
      } else if (h.temp > 31) {
        add('very hot', 35);
      } else if (h.temp > 27) {
        add('warm', 20);
      } else if (h.temp < 10) {
        add('cold', 30);
      }
      if (h.uvIndex > 10) {
        add('extreme UV', 30);
      } else if (h.uvIndex >= 8) {
        add('high UV', 20);
      } else if (h.uvIndex >= 6) {
        add('moderate UV', 10);
      }
      // Cloud cover is relief, not a penalty, on a hot day.
      if (h.cloudCover >= 50 && h.temp > 29) penalty = (penalty - 10).clamp(0, 1000);
      break;

    case ActivityCategory.beach:
      if (h.windSpeed > 30) add('wind-blown sand', 20);
      if (h.precipProbability > 55) {
        add('showers likely', 55);
      } else if (h.precipProbability > 25) {
        add('showers possible', 25);
      }
      if (h.temp < 21) {
        add('too cool to swim', 45);
      } else if (h.temp < 25) {
        add('cool', 25);
      } else if (h.temp < 28) {
        add('mild', 10);
      }
      if (h.uvIndex > 10) {
        add('extreme UV — shade up', 25);
      } else if (h.uvIndex >= 8) {
        add('high UV', 15);
      }
      if (h.cloudCover >= 80) add('flat grey light', 15);
      break;

    case ActivityCategory.general:
      if (h.windSpeed > 35) add('windy', 15);
      if (h.precipProbability > 55) {
        add('showers likely', 50);
      } else if (h.precipProbability > 25) {
        add('showers possible', 25);
      }
      if (h.temp > 34) {
        add('very hot', 40);
      } else if (h.temp > 31) {
        add('hot', 20);
      } else if (h.temp < 15) {
        add('cool', 25);
      }
      if (h.uvIndex >= 10) add('extreme UV', 12);
      break;
  }

  final score = (100 - penalty).clamp(0, 100);
  final level = score >= 70 ? Suitability.good : (score >= 40 ? Suitability.fair : Suitability.poor);
  var reason = '';
  if (reasons.isNotEmpty) {
    reason = reasons.entries.reduce((a, b) => a.value >= b.value ? a : b).key;
  }
  return HourRating(time: h.time, level: level, score: score, reason: reason);
}

List<HourRating> rateHours(ActivityCategory category, List<HourlyPoint> hourly) =>
    [for (final h in hourly) rateHour(category, h)];

/// Longest run of good hours in the next [windowHours] (falling back to the
/// longest run of at-least-fair), formatted e.g. '6–9 AM'. Null when
/// nothing in the window clears even the fair bar.
String? bestWindow(List<HourRating> ratings, {int windowHours = 12}) {
  final window = ratings.take(windowHours).toList();
  if (window.isEmpty) return null;

  List<HourRating>? longestRun(bool Function(Suitability) ok) {
    List<HourRating>? best;
    var current = <HourRating>[];
    for (final r in window) {
      if (ok(r.level)) {
        current = [...current, r];
        if (best == null || current.length > best.length) best = current;
      } else {
        current = [];
      }
    }
    return best;
  }

  final goodRun = longestRun((l) => l == Suitability.good);
  final run = (goodRun != null && goodRun.length >= 2) ? goodRun : longestRun((l) => l != Suitability.poor);
  if (run == null || run.isEmpty) return null;

  final fmt = DateFormat('h a');
  final start = fmt.format(run.first.time);
  final end = fmt.format(run.last.time.add(const Duration(hours: 1)));
  return '$start–$end';
}

/// One-sentence hero headline, voiced per category.
String headline(ActivityCategory category, List<HourRating> ratings) {
  if (ratings.isEmpty) return 'Check back soon for local conditions.';
  final now = ratings.first;
  final window = bestWindow(ratings);
  if (now.level == Suitability.good) {
    return window != null ? 'Good now, holding through $window.' : 'Good conditions right now.';
  }
  if (window != null) return 'Poor right now — best window $window.';
  final reason = now.reason.isEmpty ? 'unsettled conditions' : now.reason;
  return 'Rough all day — $reason.';
}
