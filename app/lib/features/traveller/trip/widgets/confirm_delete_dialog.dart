// Owner confirmation before a stop is removed for everyone.

import 'package:flutter/material.dart';

import '../../../../core/theme/palette.dart';
import '../../../../data/models/models.dart';

/// Returns true when the owner confirms the removal.
Future<bool> showConfirmDeleteDialog(
    BuildContext context, String stopName, Traveler? addedBy) async {
  const coral = Color(0xFFC2666F);
  final credit = addedBy != null && addedBy.id != 'me' ? addedBy : null;
  final text = Theme.of(context).textTheme;

  final confirmed = await showDialog<bool>(
    context: context,
    builder: (context) => Dialog(
      insetPadding: const EdgeInsets.symmetric(horizontal: 28),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(24)),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              width: 48,
              height: 48,
              decoration: BoxDecoration(
                color: coral.withValues(alpha: 0.14),
                borderRadius: BorderRadius.circular(16),
              ),
              child: const Icon(Icons.gpp_maybe_outlined, size: 24, color: coral),
            ),
            const SizedBox(height: 12),
            Text('Remove “$stopName”?', style: text.titleMedium?.copyWith(fontSize: 18)),
            const SizedBox(height: 6),
            Text(
              '${credit != null ? 'This stop was added by ${credit.name}. ' : ''}'
              "As the trip owner, you need to confirm before it's removed for everyone.",
              style: text.bodyMedium?.copyWith(color: SpottoColors.muted),
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                Expanded(
                  child: FilledButton(
                    style: FilledButton.styleFrom(
                      backgroundColor: SpottoColors.sand,
                      foregroundColor: SpottoColors.ink,
                    ),
                    onPressed: () => Navigator.of(context).pop(false),
                    child: const Text('Keep it'),
                  ),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: FilledButton(
                    style: FilledButton.styleFrom(backgroundColor: coral),
                    onPressed: () => Navigator.of(context).pop(true),
                    child: const Text('Remove'),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    ),
  );
  return confirmed ?? false;
}
