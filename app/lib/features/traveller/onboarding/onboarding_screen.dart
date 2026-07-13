// Onboarding: welcome → travel style → interests. Saves UserPrefs.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/pill.dart';
import '../../../core/widgets/spotto_card.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/mock/places.dart' as data;
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';

class OnboardingScreen extends ConsumerStatefulWidget {
  const OnboardingScreen({super.key});

  @override
  ConsumerState<OnboardingScreen> createState() => _OnboardingScreenState();
}

class _OnboardingScreenState extends ConsumerState<OnboardingScreen> {
  int step = 0;
  String? style;
  final Set<String> picked = {};

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: switch (step) {
            0 => Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Spacer(),
                  Text('Ayubowan 👋', style: text.displaySmall),
                  const SizedBox(height: 8),
                  Text(
                    'Let’s tune Spotto to how you travel. Two quick questions.',
                    style: text.bodyLarge,
                  ),
                  const Spacer(),
                  SizedBox(
                    width: double.infinity,
                    child: FilledButton(
                      onPressed: () => setState(() => step = 1),
                      child: const Text('Let’s go'),
                    ),
                  ),
                ],
              ),
            1 => Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('How do you travel?', style: text.headlineSmall),
                  const SizedBox(height: 16),
                  Expanded(
                    child: ListView(
                      children: [
                        for (final s in data.travelStyles) ...[
                          SpottoCard(
                            onTap: () => setState(() => style = s.id),
                            color: style == s.id ? SpottoColors.accentTint : Colors.white,
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
                                if (style == s.id)
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
                      onPressed: style == null ? null : () => setState(() => step = 2),
                      child: const Text('Next'),
                    ),
                  ),
                ],
              ),
            _ => Column(
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
                      onPressed: picked.isEmpty
                          ? null
                          : () {
                              ref.read(prefsProvider.notifier).save(
                                  UserPrefs(style: style!, interests: picked.toList()));
                              context.go('/t');
                            },
                      child: const Text('Start exploring'),
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
