import 'package:flutter/material.dart';

/// Lightweight staggered entrance for list rows — a fade + 10px rise whose
/// delay is encoded into the animation curve (via Interval) rather than a
/// Future.delayed timer, so it stays a stateless, controller-free widget.
/// Shared across screens (originally private to spots_screen.dart) so the
/// whole app has one entrance signature.
class StaggerIn extends StatelessWidget {
  const StaggerIn({super.key, required this.index, required this.child, this.axis = Axis.vertical});

  final int index;
  final Widget child;
  final Axis axis;

  @override
  Widget build(BuildContext context) {
    if (MediaQuery.maybeDisableAnimationsOf(context) ?? false) return child;
    final delayMs = index.clamp(0, 6) * 45;
    final totalMs = 320 + delayMs;
    return TweenAnimationBuilder<double>(
      tween: Tween(begin: 0, end: 1),
      duration: Duration(milliseconds: totalMs),
      curve: Interval(delayMs / totalMs, 1.0, curve: Curves.easeOutCubic),
      child: child,
      builder: (context, t, child) {
        if (t == 1) return child!;
        final rise = (1 - t) * 10;
        final offset = axis == Axis.vertical ? Offset(0, rise) : Offset(rise, 0);
        return Opacity(
          opacity: t,
          child: Transform.translate(offset: offset, child: child),
        );
      },
    );
  }
}
