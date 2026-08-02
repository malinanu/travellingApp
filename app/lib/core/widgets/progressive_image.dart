import 'dart:convert';
import 'dart:typed_data';
import 'dart:ui';

import 'package:flutter/material.dart';

import '../theme/palette.dart';

/// A "blur-up" progressive image: shows [microThumb] (a tiny inline base64
/// placeholder, e.g. `data:image/jpeg;base64,...`) instantly with a blur
/// filter, then cross-fades in the real network image at [url] once it
/// loads — never a blank flash between the two.
///
/// Never requests more than the one network image passed in. Callers pick
/// which resolution tier is appropriate for how prominently the image is
/// shown: a card/list tile should pass a ~320px thumbUrl (never the full
/// original — there's usually no viewer to justify the extra bytes for
/// something shown that small); only a genuine full-screen photo viewer
/// should ever pass the real full-resolution url. [url] should already be
/// resolved (see resolveMediaUrl) — this widget doesn't do that itself,
/// matching how every existing Image.network call site in this app works.
class ProgressiveImage extends StatefulWidget {
  final String url;
  final String microThumb;
  final double? width;
  final double? height;
  final BoxFit fit;
  final BorderRadius? borderRadius;
  final WidgetBuilder? errorBuilder;

  const ProgressiveImage({
    super.key,
    required this.url,
    this.microThumb = '',
    this.width,
    this.height,
    this.fit = BoxFit.cover,
    this.borderRadius,
    this.errorBuilder,
  });

  @override
  State<ProgressiveImage> createState() => _ProgressiveImageState();
}

class _ProgressiveImageState extends State<ProgressiveImage> {
  bool _loaded = false;
  bool _errored = false;

  Uint8List? _decodeMicro() {
    if (widget.microThumb.isEmpty) return null;
    try {
      final comma = widget.microThumb.indexOf(',');
      final b64 = comma == -1 ? widget.microThumb : widget.microThumb.substring(comma + 1);
      return base64Decode(b64);
    } catch (_) {
      // Malformed placeholder data shouldn't ever take down the real
      // image below it — just skip the blur layer.
      return null;
    }
  }

  @override
  Widget build(BuildContext context) {
    if (widget.url.isEmpty) {
      return widget.errorBuilder?.call(context) ?? const SizedBox.shrink();
    }
    final micro = _decodeMicro();

    Widget content = Stack(
      fit: StackFit.expand,
      children: [
        // Solid fallback while there's no blur placeholder yet (the
        // background worker hasn't reached this image, or generation
        // failed) — otherwise that window is a blank/transparent gap
        // instead of the sand-colored loading state every one of these
        // call sites used to show via Image.network's old loadingBuilder.
        if (micro == null) const ColoredBox(color: SpottoColors.sand),
        if (micro != null)
          ImageFiltered(
            imageFilter: ImageFilter.blur(sigmaX: 8, sigmaY: 8, tileMode: TileMode.decal),
            child: Image.memory(micro, fit: widget.fit, gaplessPlayback: true),
          ),
        if (!_errored)
          AnimatedOpacity(
            opacity: _loaded ? 1 : 0,
            duration: const Duration(milliseconds: 200),
            child: Image.network(
              widget.url,
              fit: widget.fit,
              gaplessPlayback: true,
              frameBuilder: (context, child, frame, wasSynchronouslyLoaded) {
                // wasSynchronouslyLoaded is true whenever Flutter's own
                // ImageCache already has this image (revisiting a screen,
                // scrolling back up a list) — that used to skip this
                // entirely, which left _loaded stuck at false and the
                // AnimatedOpacity permanently invisible for any cached
                // image. Schedule the same post-frame setState either way;
                // addPostFrameCallback is what makes mutating state safe
                // during frameBuilder (which runs mid-build) regardless of
                // whether the load was sync or async.
                if (frame != null && !_loaded) {
                  WidgetsBinding.instance.addPostFrameCallback((_) {
                    if (mounted) setState(() => _loaded = true);
                  });
                }
                return child;
              },
              errorBuilder: (context, error, stackTrace) {
                WidgetsBinding.instance.addPostFrameCallback((_) {
                  if (mounted) setState(() => _errored = true);
                });
                return const SizedBox.shrink();
              },
            ),
          ),
        if (_errored) widget.errorBuilder?.call(context) ?? const SizedBox.shrink(),
      ],
    );

    if (widget.borderRadius != null) {
      content = ClipRRect(borderRadius: widget.borderRadius!, child: content);
    }
    return SizedBox(width: widget.width, height: widget.height, child: content);
  }
}
