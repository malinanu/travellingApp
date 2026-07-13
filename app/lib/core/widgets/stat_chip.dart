import 'package:flutter/material.dart';

import '../theme/palette.dart';

/// Small labelled stat (wifi score, hours, rating) used in cards and sheets.
class StatChip extends StatelessWidget {
  final IconData icon;
  final String value;
  final String label;
  final Color? iconColor;

  const StatChip({super.key, required this.icon, required this.value, required this.label, this.iconColor});

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      decoration: BoxDecoration(
        color: SpottoColors.sand,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: SpottoColors.line, width: 1.5),
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(icon, size: 18, color: iconColor ?? SpottoColors.accent),
          const SizedBox(height: 4),
          Text(value, style: text.labelLarge),
          Text(label, style: text.labelSmall),
        ],
      ),
    );
  }
}
