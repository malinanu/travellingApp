// Opt-in for "you're near a saved place" alerts: an explanation before any
// permission prompt, and a card on the Saved tab showing whether they're on.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';

import '../../../core/theme/palette.dart';
import '../../../data/repositories/nearby_alerts_repository.dart';

/// Explains the feature, then asks for permissions and turns alerts on.
Future<void> enableNearbyAlertsFlow(BuildContext context, WidgetRef ref) async {
  final go = await showDialog<bool>(
    context: context,
    builder: (ctx) => AlertDialog(
      title: const Text('Heads-up near saved places', style: TextStyle(fontWeight: FontWeight.w800)),
      content: const Text(
        "We'll let you know when you're close to a place you saved, even when Spotto is closed.\n\n"
        'To do that, your phone needs location set to "Allow all the time". Your phone does the '
        'checking on its own: your location isn\'t sent to Spotto. At most 3 alerts a day, and '
        'never twice in a day for the same place.',
      ),
      actions: [
        TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Not now')),
        FilledButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('Turn on')),
      ],
    ),
  );
  if (go != true || !context.mounted) return;
  final result = await ref.read(nearbyAlertsProvider.notifier).enable();
  if (!context.mounted) return;
  final message = switch (result) {
    NearbyEnableResult.enabled => "Done. We'll give you a heads-up near your saved places.",
    NearbyEnableResult.openedSettings =>
      'In Settings, open Location and choose "Allow all the time", then come back to Spotto.',
    NearbyEnableResult.locationDenied => 'Location is turned off for Spotto, so we can\'t tell when you\'re nearby.',
    NearbyEnableResult.notificationsDenied => 'Notifications are turned off for Spotto, so we can\'t alert you.',
  };
  ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(message)));
}

/// Card on the Saved tab: an offer to turn alerts on, or their status.
class NearbyAlertsCard extends ConsumerWidget {
  const NearbyAlertsCard({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final s = ref.watch(nearbyAlertsProvider);
    final (title, body, action, onAction) = switch (s) {
      NearbyAlertsState(enabled: false) => (
          'Get a heads-up nearby',
          "We'll tell you when you're close to one of these places, even with the app closed.",
          'Turn on',
          () => enableNearbyAlertsFlow(context, ref),
        ),
      NearbyAlertsState(issue: NearbyAlertsIssue.needsAlwaysLocation) => (
          'One more step for nearby alerts',
          'Set location to "Allow all the time" so alerts work while Spotto is closed.',
          'Open settings',
          () => Geolocator.openAppSettings(),
        ),
      NearbyAlertsState(issue: NearbyAlertsIssue.locationDenied) => (
          'Nearby alerts are paused',
          'Location is turned off for Spotto.',
          'Open settings',
          () => Geolocator.openAppSettings(),
        ),
      _ => (
          'Nearby alerts are on',
          s.watching == 1 ? 'Watching 1 place near you.' : 'Watching ${s.watching} places near you.',
          null,
          null,
        ),
    };
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: s.enabled && s.issue == null ? SpottoColors.accentTint : const Color(0x1FF6B81E),
        borderRadius: BorderRadius.circular(16),
      ),
      child: Row(
        children: [
          Icon(s.enabled && s.issue == null ? Icons.notifications_active_rounded : Icons.near_me_rounded,
              color: SpottoColors.accent),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: const TextStyle(fontWeight: FontWeight.w800, color: SpottoColors.ink)),
                const SizedBox(height: 2),
                Text(body, style: const TextStyle(fontSize: 12.5, color: SpottoColors.inkSoft)),
              ],
            ),
          ),
          if (action != null) ...[
            const SizedBox(width: 8),
            FilledButton(
              onPressed: onAction,
              style: FilledButton.styleFrom(backgroundColor: SpottoColors.sun, foregroundColor: SpottoColors.ink),
              child: Text(action, style: const TextStyle(fontWeight: FontWeight.w800)),
            ),
          ],
        ],
      ),
    );
  }
}
