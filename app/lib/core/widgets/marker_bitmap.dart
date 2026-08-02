// Rasterizes Flutter widgets into BitmapDescriptors for use as custom
// google_maps_flutter markers (which render native bitmaps, not live widget
// trees), with an in-memory cache so identical markers are never re-rendered.

import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/scheduler.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';

class MarkerBitmapCache {
  MarkerBitmapCache._();
  static final MarkerBitmapCache instance = MarkerBitmapCache._();

  final Map<String, BitmapDescriptor> _cache = {};
  final Set<String> _pending = {};

  BitmapDescriptor? get(String key) => _cache[key];

  /// Renders [widget] off-screen at logical [size] and stores the resulting
  /// bitmap under [key]. Calls [onReady] once the bitmap lands in the cache.
  /// No-ops if [key] is already cached or a render for it is already in
  /// flight — safe to call on every build.
  void render({
    required BuildContext context,
    required String key,
    required Widget widget,
    required Size size,
    required VoidCallback onReady,
    double pixelRatio = 3.0,
  }) {
    if (_cache.containsKey(key) || _pending.contains(key)) return;
    _pending.add(key);

    // Overlay.insert() marks the Overlay dirty, which isn't allowed while
    // our own widget is still mid-build — defer the insert (and capture)
    // to after this frame finishes.
    SchedulerBinding.instance.addPostFrameCallback((_) async {
      if (!context.mounted) {
        _pending.remove(key);
        return;
      }
      final repaintKey = GlobalKey();
      late OverlayEntry entry;
      entry = OverlayEntry(
        builder: (_) => Positioned(
          left: -9999,
          top: -9999,
          width: size.width,
          height: size.height,
          child: RepaintBoundary(
            key: repaintKey,
            child: Material(type: MaterialType.transparency, child: widget),
          ),
        ),
      );
      Overlay.of(context).insert(entry);

      try {
        await SchedulerBinding.instance.endOfFrame;
        final boundary = repaintKey.currentContext?.findRenderObject()
            as RenderRepaintBoundary?;
        if (boundary == null) return;
        final image = await boundary.toImage(pixelRatio: pixelRatio);
        final byteData = await image.toByteData(format: ui.ImageByteFormat.png);
        if (byteData == null) return;
        _cache[key] = BitmapDescriptor.bytes(
          byteData.buffer.asUint8List(),
          width: size.width,
          height: size.height,
        );
        onReady();
      } finally {
        entry.remove();
        _pending.remove(key);
      }
    });
  }
}
