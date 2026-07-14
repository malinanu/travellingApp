// Draws the same circular badge look as the old illustrated-map `_Pin`
// widget (colored circle, white border, centered icon glyph, optional
// sun-colored deal badge) directly onto a canvas, for use as a
// google_maps_flutter `BitmapDescriptor`. Markers can't render arbitrary
// widgets, so this is the standard workaround: paint once, cache the bytes.

import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';

/// Renders a place/traveler pin badge to a [BitmapDescriptor].
///
/// [diameter] is the circle's logical size; the image is rendered at
/// [scale]x that for crispness on high-density screens (defaults suit a
/// devicePixelRatio around 3).
Future<BitmapDescriptor> renderPinBitmap({
  required Color color,
  required IconData icon,
  bool hasDeal = false,
  double diameter = 40,
  double scale = 3,
}) async {
  final size = diameter * scale;
  final badgeSize = size * 0.28;
  final canvasSize = size + badgeSize / 2; // room for the corner badge

  final recorder = ui.PictureRecorder();
  final canvas = Canvas(recorder, Rect.fromLTWH(0, 0, canvasSize, canvasSize));
  final center = Offset(size / 2, size / 2);
  final radius = size / 2;

  // Drop shadow.
  canvas.drawCircle(
    center.translate(0, size * 0.04),
    radius,
    Paint()
      ..color = Colors.black.withValues(alpha: 0.18)
      ..maskFilter = ui.MaskFilter.blur(ui.BlurStyle.normal, size * 0.05),
  );

  // White border ring, then the colored fill.
  canvas.drawCircle(center, radius, Paint()..color = Colors.white);
  canvas.drawCircle(center, radius - size * 0.075, Paint()..color = color);

  // Icon glyph, centered.
  final iconSize = size * 0.44;
  final textPainter = TextPainter(textDirection: TextDirection.ltr)
    ..text = TextSpan(
      text: String.fromCharCode(icon.codePoint),
      style: TextStyle(
        fontSize: iconSize,
        fontFamily: icon.fontFamily,
        package: icon.fontPackage,
        color: Colors.white,
      ),
    )
    ..layout();
  textPainter.paint(
    canvas,
    center - Offset(textPainter.width / 2, textPainter.height / 2),
  );

  // Deal badge, top-right corner.
  if (hasDeal) {
    final badgeCenter = Offset(size - badgeSize * 0.3, badgeSize * 0.3);
    canvas.drawCircle(badgeCenter, badgeSize / 2 + 2, Paint()..color = Colors.white);
    canvas.drawCircle(badgeCenter, badgeSize / 2, Paint()..color = const Color(0xFFF6B81E));
  }

  final picture = recorder.endRecording();
  final image = await picture.toImage(canvasSize.round(), canvasSize.round());
  final bytes = await image.toByteData(format: ui.ImageByteFormat.png);
  return BitmapDescriptor.bytes(bytes!.buffer.asUint8List(), imagePixelRatio: scale);
}

/// Renders a traveler avatar bubble (initial on a colored circle, white
/// border) to a [BitmapDescriptor] — same look as [AvatarBubble].
Future<BitmapDescriptor> renderAvatarBitmap({
  required String initial,
  required Color color,
  double diameter = 34,
  double scale = 3,
}) async {
  final size = diameter * scale;
  final recorder = ui.PictureRecorder();
  final canvas = Canvas(recorder, Rect.fromLTWH(0, 0, size, size));
  final center = Offset(size / 2, size / 2);
  final radius = size / 2;

  canvas.drawCircle(
    center.translate(0, size * 0.04),
    radius,
    Paint()
      ..color = Colors.black.withValues(alpha: 0.18)
      ..maskFilter = ui.MaskFilter.blur(ui.BlurStyle.normal, size * 0.05),
  );
  canvas.drawCircle(center, radius, Paint()..color = Colors.white);
  canvas.drawCircle(center, radius - size * 0.075, Paint()..color = color);

  final textPainter = TextPainter(textDirection: TextDirection.ltr)
    ..text = TextSpan(
      text: initial,
      style: TextStyle(
        fontSize: size * 0.38,
        fontWeight: FontWeight.w800,
        color: Colors.white,
      ),
    )
    ..layout();
  textPainter.paint(
    canvas,
    center - Offset(textPainter.width / 2, textPainter.height / 2),
  );

  final picture = recorder.endRecording();
  final image = await picture.toImage(size.round(), size.round());
  final bytes = await image.toByteData(format: ui.ImageByteFormat.png);
  return BitmapDescriptor.bytes(bytes!.buffer.asUint8List(), imagePixelRatio: scale);
}

/// Simple in-memory cache so markers aren't re-rasterized on every rebuild.
class PinBitmapCache {
  final _cache = <String, BitmapDescriptor>{};

  Future<BitmapDescriptor> place({
    required String key,
    required Color color,
    required IconData icon,
    required bool hasDeal,
    required bool emphasized,
  }) async {
    final cacheKey = '${key}_$emphasized';
    final cached = _cache[cacheKey];
    if (cached != null) return cached;
    final bmp = await renderPinBitmap(
      color: color,
      icon: icon,
      hasDeal: hasDeal,
      diameter: emphasized ? 44 : 38,
    );
    _cache[cacheKey] = bmp;
    return bmp;
  }

  Future<BitmapDescriptor> avatar({
    required String key,
    required String initial,
    required Color color,
  }) async {
    final cached = _cache[key];
    if (cached != null) return cached;
    final bmp = await renderAvatarBitmap(initial: initial, color: color);
    _cache[key] = bmp;
    return bmp;
  }
}
