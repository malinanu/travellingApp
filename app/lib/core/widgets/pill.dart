import 'package:flutter/material.dart';

import '../theme/palette.dart';
import 'press_scale.dart';

/// Tag / filter pill. Selected = filled accent, unselected = pale tint.
class Pill extends StatelessWidget {
  final String label;
  final IconData? icon;
  final bool selected;
  final VoidCallback? onTap;

  const Pill({super.key, required this.label, this.icon, this.selected = false, this.onTap});

  @override
  Widget build(BuildContext context) {
    final fg = selected ? Colors.white : SpottoColors.accent;
    final pill = AnimatedContainer(
      duration: const Duration(milliseconds: 160),
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
      decoration: BoxDecoration(
        color: selected ? SpottoColors.accent : SpottoColors.accentTint,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) ...[Icon(icon, size: 16, color: fg), const SizedBox(width: 6)],
          Text(
            label,
            style: Theme.of(context).textTheme.labelMedium?.copyWith(color: fg),
          ),
        ],
      ),
    );
    if (onTap == null) return pill;
    return PressScale(onTap: onTap, child: pill);
  }
}
