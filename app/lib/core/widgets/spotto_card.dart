import 'package:flutter/material.dart';

import '../theme/palette.dart';
import 'press_scale.dart';

class SpottoCard extends StatelessWidget {
  final Widget child;
  final VoidCallback? onTap;
  final EdgeInsetsGeometry padding;
  final double radius;
  final Color? color;
  final bool shadow;

  const SpottoCard({
    super.key,
    required this.child,
    this.onTap,
    this.padding = const EdgeInsets.all(16),
    this.radius = 20,
    this.color,
    this.shadow = true,
  });

  @override
  Widget build(BuildContext context) {
    final cardColor = color ?? Theme.of(context).cardTheme.color ?? Theme.of(context).colorScheme.surface;
    final card = Container(
      padding: padding,
      decoration: BoxDecoration(
        color: cardColor,
        borderRadius: BorderRadius.circular(radius),
        border: Border.all(color: Theme.of(context).colorScheme.outline, width: 1.5),
        boxShadow: shadow
            ? const [BoxShadow(color: Color(0x14103D3A), blurRadius: 14, offset: Offset(0, 6))]
            : null,
      ),
      child: child,
    );
    if (onTap == null) return card;
    return PressScale(onTap: onTap, child: card);
  }
}
