// Sunny Postcard palette — ported from the prototype's CSS variables.

import 'package:flutter/material.dart';

abstract final class SpottoColors {
  static const sand = Color(0xFFFBF6ED); // background
  static const accent = Color(0xFF0B6F86); // teal
  static const accent2 = Color(0xFF0E8C8A);
  static const accentTint = Color(0xFFDDEDEF);
  static const sun = Color(0xFFF6B81E); // deals / ratings / primary action
  static const ink = Color(0xFF103D3A);
  static const inkSoft = Color(0xFF3A5A57);
  static const muted = Color(0xFF6C8582);
  static const line = Color(0xFFE7E1D4);
  static const deepTeal = Color(0xFF0B4A54);
  static const clay = Color(0xFF0A5F6E);
  static const paleTeal = Color(0xFFBDE4E0);

  /// Soft ink shadow used across cards and sheets.
  static const inkShadow = Color(0x2E103D3A); // rgba(16,61,58,0.18)

  /// Pin tone colors keyed by Place.color.
  static const pinTones = {
    'teal': accent,
    'coral': Color(0xFFF0682F),
    'sun': Color(0xFFEAA200),
    'clay': Color(0xFFC2683F),
    'plum': Color(0xFF7C5CFF),
  };

  /// Illustration base tone per kind (ILLO_TONE in the prototype).
  static const illoTones = {
    'cowork': Color(0xFF0B6F86),
    'surf': Color(0xFF0E8C8A),
    'sunset': Color(0xFF0B4A54),
    'clay': Color(0xFF0A5F6E),
    'batik': Color(0xFF0B6F86),
  };
}
