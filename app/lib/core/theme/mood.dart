// Weather-screen hero mood — a gradient built entirely from the existing
// SpottoColors palette (no new color system, no literal sky-blue clone of
// a reference weather app). Resolution order is: precipitation beats
// cloud cover beats clock, since actual weather should always win over
// what time it happens to be.

import 'package:flutter/material.dart';

import '../../data/models/models.dart';
import 'palette.dart';

enum SpottoMood { dawn, day, golden, night, overcast, rain }

SpottoMood moodFor(WeatherSnapshot snapshot, {DateTime? at}) {
  final now = at ?? DateTime.now();
  final n = snapshot.now;

  if (n.precipProbability >= 50 || (snapshot.hourly.isNotEmpty && snapshot.hourly.first.precipMm > 0.1)) {
    return SpottoMood.rain;
  }
  if (n.cloudCover >= 65) return SpottoMood.overcast;

  final sunrise = snapshot.sunrise;
  final sunset = snapshot.sunset;
  if (now.isBefore(sunrise.add(const Duration(hours: 1))) && now.isAfter(sunrise.subtract(const Duration(hours: 1)))) {
    return SpottoMood.dawn;
  }
  if (now.isAfter(sunset.subtract(const Duration(hours: 1))) && now.isBefore(sunset.add(const Duration(minutes: 30)))) {
    return SpottoMood.golden;
  }
  if (now.isAfter(sunset) || now.isBefore(sunrise)) return SpottoMood.night;
  return SpottoMood.day;
}

/// The hero's opaque background gradient — top to bottom.
LinearGradient heroGradient(SpottoMood mood) {
  switch (mood) {
    case SpottoMood.day:
      return const LinearGradient(
        begin: Alignment.topCenter,
        end: Alignment.bottomCenter,
        colors: [SpottoColors.deepTeal, SpottoColors.accent, SpottoColors.accent2],
        stops: [0, 0.55, 1],
      );
    case SpottoMood.dawn:
      return const LinearGradient(
        begin: Alignment.topCenter,
        end: Alignment.bottomCenter,
        colors: [SpottoColors.deepTeal, SpottoColors.clay, SpottoColors.sun],
        stops: [0, 0.5, 1],
      );
    case SpottoMood.golden:
      return const LinearGradient(
        begin: Alignment.topCenter,
        end: Alignment.bottomCenter,
        colors: [SpottoColors.ink, SpottoColors.deepTeal, SpottoColors.clay, SpottoColors.sun],
        stops: [0, 0.38, 0.72, 1],
      );
    case SpottoMood.night:
      return const LinearGradient(
        begin: Alignment.topCenter,
        end: Alignment.bottomCenter,
        colors: [SpottoColors.ink, SpottoColors.deepTeal],
      );
    case SpottoMood.overcast:
      return const LinearGradient(
        begin: Alignment.topCenter,
        end: Alignment.bottomCenter,
        colors: [SpottoColors.inkSoft, SpottoColors.muted, SpottoColors.paleTeal],
        stops: [0, 0.6, 1],
      );
    case SpottoMood.rain:
      return const LinearGradient(
        begin: Alignment.topCenter,
        end: Alignment.bottomCenter,
        colors: [SpottoColors.ink, SpottoColors.deepTeal, SpottoColors.inkSoft],
        stops: [0, 0.5, 1],
      );
  }
}

/// A subtle radial accent over the hero gradient — a sun bloom for clear
/// day, faint moonlight for night, none for the flatter moods. Returns
/// null when the mood doesn't call for one.
RadialGradient? heroAccent(SpottoMood mood) {
  switch (mood) {
    case SpottoMood.day:
      return RadialGradient(
        center: const Alignment(0.72, -0.5),
        radius: 0.9,
        colors: [SpottoColors.sun.withValues(alpha: 0.14), Colors.transparent],
      );
    case SpottoMood.night:
      return RadialGradient(
        center: const Alignment(0.7, -0.6),
        radius: 0.9,
        colors: [SpottoColors.paleTeal.withValues(alpha: 0.08), Colors.transparent],
      );
    case SpottoMood.dawn:
    case SpottoMood.golden:
    case SpottoMood.overcast:
    case SpottoMood.rain:
      return null;
  }
}
