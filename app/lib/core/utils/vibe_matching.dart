import 'package:flutter/material.dart';

import '../../data/models/models.dart';
import '../../data/vendor_categories.dart';

class VibeMatchResult {
  final int percentage;
  final String headline;
  final String badgeLabel;
  final String description;
  final bool isEstimated;
  final Color color;
  final IconData icon;

  const VibeMatchResult({
    required this.percentage,
    required this.headline,
    required this.badgeLabel,
    required this.description,
    required this.isEstimated,
    required this.color,
    required this.icon,
  });
}

/// Computes the Vibe Suitability Match between a place's crowdsourced vibe
/// and the traveller's saved travel styles + interests.
VibeMatchResult calculateVibeMatch({
  required Place place,
  required UserPrefs? prefs,
}) {
  final styles = prefs?.styles ?? const [];
  final interests = prefs?.interests ?? const [];

  // Default targets if user hasn't configured a profile yet
  double prodSum = 0, loudSum = 0, crowdSum = 0;
  double prodWeight = 0, loudWeight = 0, crowdWeight = 0;

  void add(double p, double l, double c, double w) {
    prodSum += p * w;
    loudSum += l * w;
    crowdSum += c * w;
    prodWeight += w;
    loudWeight += w;
    crowdWeight += w;
  }

  String primaryStyleLabel = 'your vibe';

  for (final s in styles) {
    switch (s.toLowerCase().trim()) {
      case 'nomad':
      case 'digital nomad':
        add(4.5, 1.8, 2.5, 1.5);
        primaryStyleLabel = 'Digital Nomad';
      case 'surf':
      case 'surf traveler':
        add(2.0, 3.0, 3.0, 1.0);
        if (primaryStyleLabel == 'your vibe') primaryStyleLabel = 'Surf Traveler';
      case 'luxury':
        add(3.0, 1.5, 1.8, 1.2);
        if (primaryStyleLabel == 'your vibe') primaryStyleLabel = 'Luxury Traveler';
      case 'backpacker':
        add(2.5, 3.8, 4.0, 1.2);
        if (primaryStyleLabel == 'your vibe') primaryStyleLabel = 'Backpacker';
    }
  }

  for (final i in interests) {
    switch (i.toLowerCase().trim()) {
      case 'quiet':
      case 'quiet zone':
        loudSum += 1.0 * 1.5;
        loudWeight += 1.5;
      case 'coworking':
        prodSum += 5.0 * 1.5;
        prodWeight += 1.5;
      case 'wifi':
      case 'fast wi-fi':
        prodSum += 4.5 * 1.2;
        prodWeight += 1.2;
      case 'nightlife':
      case 'live music':
        loudSum += 4.5 * 1.5;
        crowdSum += 4.5 * 1.5;
        loudWeight += 1.5;
        crowdWeight += 1.5;
    }
  }

  final double targetProd = prodWeight > 0 ? (prodSum / prodWeight) : 3.0;
  final double targetLoud = loudWeight > 0 ? (loudSum / loudWeight) : 2.5;
  final double targetCrowd = crowdWeight > 0 ? (crowdSum / crowdWeight) : 2.5;

  final dims = relevantVibeDimensions(place.categoryId);
  final isEstimated = place.vibe.sampleSize <= 0;

  // Actual readings or category priors
  double actualProd = place.vibe.productive;
  double actualLoud = place.vibe.loud;
  double actualCrowd = place.vibe.crowded;

  if (isEstimated) {
    switch (place.categoryId.toLowerCase()) {
      case 'coffee':
      case 'coworking':
        actualProd = 4.2;
        actualLoud = 2.0;
        actualCrowd = 2.5;
      case 'restaurants':
        actualProd = 2.5;
        actualLoud = 3.0;
        actualCrowd = 3.2;
      case 'nightlife':
      case 'bars':
        actualProd = 1.5;
        actualLoud = 4.2;
        actualCrowd = 4.2;
      case 'surf':
      case 'beaches':
        actualProd = 2.0;
        actualLoud = 2.8;
        actualCrowd = 2.8;
      case 'hotels':
      case 'stays':
        actualProd = 3.2;
        actualLoud = 1.8;
        actualCrowd = 2.0;
      default:
        actualProd = 2.5;
        actualLoud = 2.5;
        actualCrowd = 2.5;
    }
  }

  double totalScore = 0;
  int evaluated = 0;

  if (dims.contains('productive')) {
    totalScore += 1.0 - (targetProd - actualProd).abs() / 4.0;
    evaluated++;
  }
  if (dims.contains('loud')) {
    totalScore += 1.0 - (targetLoud - actualLoud).abs() / 4.0;
    evaluated++;
  }
  if (dims.contains('crowded')) {
    totalScore += 1.0 - (targetCrowd - actualCrowd).abs() / 4.0;
    evaluated++;
  }

  double score = evaluated > 0 ? (totalScore / evaluated) : 0.75;
  if (isEstimated) {
    score = 0.70 + (score - 0.5) * 0.4;
  }

  final int percentage = (score * 100).round().clamp(10, 99);

  final Color color;
  final String headline;
  final String badgeLabel;
  final String description;
  final IconData icon;

  if (percentage >= 82) {
    color = const Color(0xFF0B6F86); // Teal
    icon = Icons.auto_awesome;
    headline = '$percentage% Vibe Match';
    badgeLabel = 'Ideal for $primaryStyleLabel';
    description = isEstimated
        ? 'Great expected fit based on category & amenities. Be the first to rate!'
        : 'Rated by ${place.vibe.sampleSize} travellers as a great match for $primaryStyleLabel.';
  } else if (percentage >= 65) {
    color = const Color(0xFFD97706); // Amber
    icon = Icons.check_circle_outline;
    headline = '$percentage% Good Fit';
    badgeLabel = 'Comfortable for $primaryStyleLabel';
    description = isEstimated
        ? 'Solid expected match for a casual visit.'
        : 'Consistent atmosphere suitable for a balanced visit.';
  } else {
    color = const Color(0xFF6B7280); // Neutral slate
    icon = Icons.info_outline;
    headline = '$percentage% Alternative Vibe';
    badgeLabel = 'Different from $primaryStyleLabel';
    description = 'Has a different energy level than your usual preferences.';
  }

  return VibeMatchResult(
    percentage: percentage,
    headline: headline,
    badgeLabel: badgeLabel,
    description: description,
    isEstimated: isEstimated,
    color: color,
    icon: icon,
  );
}

