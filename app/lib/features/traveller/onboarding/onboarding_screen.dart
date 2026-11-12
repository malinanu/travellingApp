// Onboarding: welcome → travel style → interests → privacy & permissions. Saves UserPrefs.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/l10n/l10n_provider.dart';
import '../../../core/l10n/language_selector_sheet.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/pill.dart';
import '../../../core/widgets/spotto_card.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/catalog/app_options.dart' as data;
import '../../../data/models/models.dart';
import '../../../data/repositories/nearby_alerts_repository.dart';
import '../../../data/repositories/repositories.dart';
import '../inspiration/nearby_alerts_ui.dart';

class OnboardingScreen extends ConsumerStatefulWidget {
  const OnboardingScreen({super.key});

  @override
  ConsumerState<OnboardingScreen> createState() => _OnboardingScreenState();
}

class _OnboardingScreenState extends ConsumerState<OnboardingScreen> {
  int step = 0;
  final Set<String> styles = {};
  final Set<String> picked = {};

  bool _shareLocation = false;
  bool _nearbyAlerts = false;
  bool _visibleToGroup = true;
  bool _countVibeVotes = true;
  bool _usageAnalytics = true;
  bool _personalisedOffers = false;
  bool _finishing = false;

  Future<void> _finish() async {
    if (_finishing) return;
    setState(() => _finishing = true);
    try {
      ref.read(prefsProvider.notifier).save(
            UserPrefs(styles: styles.toList(), interests: picked.toList()),
          );
      await ref.read(apiProvider).dio.put('/me/privacy', data: {
        'shareLocation': _shareLocation,
        'usageAnalytics': _usageAnalytics,
        'visibleToGroup': _visibleToGroup,
        'countVibeVotes': _countVibeVotes,
        'personalisedOffers': _personalisedOffers,
      });
      await ref.read(meProvider.notifier).refresh();
    } catch (_) {
      // Preferences are saved locally even if offline.
    }
    if (mounted) {
      context.go('/t');
    }
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final l10n = ref.watch(l10nProvider);
    final currentLang = ref.watch(appLanguageProvider);

    Widget toggleRow(
      String title,
      String sub,
      IconData icon, {
      required bool value,
      required ValueChanged<bool> onChanged,
    }) {
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 8),
        child: Row(
          children: [
            Container(
              width: 36,
              height: 36,
              decoration: BoxDecoration(
                color: SpottoColors.accentTint,
                borderRadius: BorderRadius.circular(10),
              ),
              child: Icon(icon, size: 20, color: SpottoColors.accent),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(title, style: text.titleSmall),
                  Text(sub, style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                ],
              ),
            ),
            Switch(
              value: value,
              activeTrackColor: SpottoColors.accent,
              onChanged: onChanged,
            ),
          ],
        ),
      );
    }

    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: switch (step) {
            0 => Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Spacer(),
                   Align(
                     alignment: Alignment.topRight,
                     child: InkWell(
                       onTap: () => showLanguageSelector(context, ref),
                       borderRadius: BorderRadius.circular(20),
                       child: Container(
                         padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                         decoration: BoxDecoration(
                           color: SpottoColors.accentTint,
                           borderRadius: BorderRadius.circular(20),
                         ),
                         child: Row(
                           mainAxisSize: MainAxisSize.min,
                           children: [
                             Text(currentLang.flag, style: const TextStyle(fontSize: 16)),
                             const SizedBox(width: 6),
                             Text(
                               currentLang.nativeName,
                               style: const TextStyle(color: SpottoColors.accent, fontSize: 13, fontWeight: FontWeight.w700),
                             ),
                             const Icon(Icons.arrow_drop_down, color: SpottoColors.accent, size: 18),
                           ],
                         ),
                       ),
                     ),
                   ),
                   const Spacer(),
                  Text('Ayubowan 👋', style: text.displaySmall),
                  const SizedBox(height: 8),
                  Text(
                    'Let’s tune Spotto to how you travel. A few quick questions.',
                    style: text.bodyLarge,
                  ),
                  const Spacer(),
                  SizedBox(
                    width: double.infinity,
                    child: FilledButton(
                      onPressed: () => setState(() => step = 1),
                      child: Text(l10n.getStarted),
                    ),
                  ),
                ],
              ),
            1 => Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('How do you travel?', style: text.headlineSmall),
                  const SizedBox(height: 4),
                  Text('Pick as many as fit.', style: text.bodyMedium),
                  const SizedBox(height: 12),
                  Expanded(
                    child: ListView(
                      children: [
                        for (final s in data.travelStyles) ...[
                          SpottoCard(
                            onTap: () => setState(() {
                              styles.contains(s.id) ? styles.remove(s.id) : styles.add(s.id);
                            }),
                            color: styles.contains(s.id) ? SpottoColors.accentTint : Colors.white,
                            child: Row(
                              children: [
                                Icon(spottoIcon(s.icon), color: SpottoColors.accent),
                                const SizedBox(width: 14),
                                Expanded(
                                  child: Column(
                                    crossAxisAlignment: CrossAxisAlignment.start,
                                    children: [
                                      Text(s.label, style: text.titleMedium),
                                      Text(s.blurb, style: text.bodySmall),
                                    ],
                                  ),
                                ),
                                if (styles.contains(s.id))
                                  const Icon(Icons.check_circle, color: SpottoColors.accent),
                              ],
                            ),
                          ),
                          const SizedBox(height: 12),
                        ],
                      ],
                    ),
                  ),
                  SizedBox(
                    width: double.infinity,
                    child: FilledButton(
                      onPressed: styles.isEmpty ? null : () => setState(() => step = 2),
                      child: Text(l10n.next),
                    ),
                  ),
                ],
              ),
            2 => Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('What are you into?', style: text.headlineSmall),
                  const SizedBox(height: 6),
                  Text('Pick a few — we’ll surface matching spots.', style: text.bodyMedium),
                  const SizedBox(height: 16),
                  Wrap(
                    spacing: 10,
                    runSpacing: 10,
                    children: [
                      for (final i in data.interests)
                        Pill(
                          label: i.label,
                          icon: spottoIcon(i.icon),
                          selected: picked.contains(i.id),
                          onTap: () => setState(() {
                            picked.contains(i.id) ? picked.remove(i.id) : picked.add(i.id);
                          }),
                        ),
                    ],
                  ),
                  const Spacer(),
                  SizedBox(
                    width: double.infinity,
                    child: FilledButton(
                      onPressed: picked.isEmpty ? null : () => setState(() => step = 3),
                      child: Text(l10n.next),
                    ),
                  ),
                ],
              ),
            _ => Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('Privacy & Permissions', style: text.headlineSmall),
                  const SizedBox(height: 6),
                  Text(
                    'Control how your data is shared. You can change these anytime in Profile.',
                    style: text.bodyMedium?.copyWith(color: SpottoColors.muted),
                  ),
                  const SizedBox(height: 16),
                  Expanded(
                    child: ListView(
                      children: [
                        toggleRow(
                          'Share live location',
                          'Other opted-in travellers nearby can see you on the map',
                          Icons.place_outlined,
                          value: _shareLocation,
                          onChanged: (v) => setState(() => _shareLocation = v),
                        ),
                        const Divider(height: 1),
                        toggleRow(
                          'Nearby saved-place alerts',
                          "A heads-up when you're close to a place you saved",
                          Icons.notifications_active_outlined,
                          value: _nearbyAlerts,
                          onChanged: (v) async {
                            if (v) {
                              await enableNearbyAlertsFlow(context, ref);
                              if (mounted) {
                                setState(() => _nearbyAlerts = ref.read(nearbyAlertsProvider).enabled);
                              }
                            } else {
                              ref.read(nearbyAlertsProvider.notifier).disable();
                              setState(() => _nearbyAlerts = false);
                            }
                          },
                        ),
                        const Divider(height: 1),
                        toggleRow(
                          'Visible to group',
                          'Appear in the group member list',
                          Icons.group_outlined,
                          value: _visibleToGroup,
                          onChanged: (v) => setState(() => _visibleToGroup = v),
                        ),
                        const Divider(height: 1),
                        toggleRow(
                          'Count my vibe votes',
                          'Include your votes in public vibe stats',
                          Icons.monitor_heart_outlined,
                          value: _countVibeVotes,
                          onChanged: (v) => setState(() => _countVibeVotes = v),
                        ),
                        const Divider(height: 1),
                        toggleRow(
                          'Usage analytics',
                          'Anonymous stats, including your rough area (about 1 km) when you use the map',
                          Icons.insights_outlined,
                          value: _usageAnalytics,
                          onChanged: (v) => setState(() => _usageAnalytics = v),
                        ),
                        const Divider(height: 1),
                        toggleRow(
                          'Personalised offers',
                          'Deals matched to your travel style',
                          Icons.campaign_outlined,
                          value: _personalisedOffers,
                          onChanged: (v) => setState(() => _personalisedOffers = v),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 12),
                  SizedBox(
                    width: double.infinity,
                    child: FilledButton(
                      onPressed: _finishing ? null : _finish,
                      child: _finishing
                          ? const SizedBox(
                              width: 18,
                              height: 18,
                              child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                            )
                          : Text(l10n.getStarted),
                    ),
                  ),
                ],
              ),
          },
        ),
      ),
    );
  }
}
