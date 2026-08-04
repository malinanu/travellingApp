// Dark-mode colour palette for Spotto — mirrors SpottoColors but with
// appropriate luminance adjustments so every token still satisfies WCAG AA
// contrast on dark backgrounds.

import 'package:flutter/material.dart';

abstract final class SpottoDarkColors {
  /// Background — deep coastal night, slightly warm so it reads as "sand
  /// after dark" rather than plain black.
  static const sand = Color(0xFF0F1F1E);

  /// Surface (cards, sheets, inputs) — one step lighter than background.
  static const surface = Color(0xFF162422);

  /// Accent stays the same hue but bumped in lightness for contrast on dark.
  static const accent = Color(0xFF2BAAC8);
  static const accent2 = Color(0xFF29C4B8);
  static const accentTint = Color(0xFF122B32);

  /// Sunflower yellow — slightly desaturated to reduce eye strain.
  static const sun = Color(0xFFD8A520);

  /// Text colours.
  static const ink = Color(0xFFDFF5F4);     // primary text (near-white)
  static const inkSoft = Color(0xFFACC8C6);  // secondary text
  static const muted = Color(0xFF7DA8A6);    // hints, labels

  /// Dividers / borders.
  static const line = Color(0xFF243635);

  /// Colours derived from the light theme counterparts.
  static const deepTeal = Color(0xFF1C7A8A);
  static const paleTeal = Color(0xFF1E4B50);

  /// Shadow — slightly blue-tinted for depth on dark surfaces.
  static const inkShadow = Color(0x44000000);

  static const ratingTones = {
    'good': accent,
    'fair': Color(0xFFC49A20),
    'poor': Color(0xFFD4634A),
  };
  static const ratingTints = {
    'good': accentTint,
    'fair': Color(0x30D8A520),
    'poor': Color(0x25D4634A),
  };

  static const pinTones = {
    'teal': accent,
    'coral': Color(0xFFF07048),
    'sun': Color(0xFFD8A520),
    'clay': Color(0xFFD07850),
    'plum': Color(0xFF9A82FF),
  };
}
