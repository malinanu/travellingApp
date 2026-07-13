// Launch screen: pick Traveller or Vendor. Remembers the last role.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../core/theme/palette.dart';
import '../core/widgets/illustration.dart';
import '../core/widgets/spotto_card.dart';
import '../data/repositories/repositories.dart';

class RoleGateScreen extends ConsumerWidget {
  const RoleGateScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    return Scaffold(
      body: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Spacer(),
              Text('Spotto', style: text.displayMedium?.copyWith(color: SpottoColors.accent)),
              const SizedBox(height: 4),
              Text('Your Weligama travel companion', style: text.bodyLarge),
              const SizedBox(height: 32),
              _RoleCard(
                title: 'I’m travelling',
                subtitle: 'Discover spots, plan trips, claim deals',
                kind: 'surf',
                onTap: () {
                  ref.read(roleProvider.notifier).set('traveller');
                  context.go('/t');
                },
              ),
              const SizedBox(height: 16),
              _RoleCard(
                title: 'I run a business',
                subtitle: 'Offers, redemptions and growth tools',
                kind: 'cowork',
                onTap: () {
                  ref.read(roleProvider.notifier).set('vendor');
                  context.go('/v');
                },
              ),
              const Spacer(flex: 2),
            ],
          ),
        ),
      ),
    );
  }
}

class _RoleCard extends StatelessWidget {
  final String title;
  final String subtitle;
  final String kind;
  final VoidCallback onTap;

  const _RoleCard({required this.title, required this.subtitle, required this.kind, required this.onTap});

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return SpottoCard(
      onTap: onTap,
      padding: EdgeInsets.zero,
      child: ClipRRect(
        borderRadius: BorderRadius.circular(18),
        child: SizedBox(
          height: 140,
          child: Stack(
            fit: StackFit.expand,
            children: [
              Illustration(kind: kind),
              Positioned(
                left: 16,
                right: 16,
                bottom: 14,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(title, style: text.titleLarge?.copyWith(color: Colors.white)),
                    Text(subtitle,
                        style: text.bodyMedium?.copyWith(color: Colors.white.withValues(alpha: 0.85))),
                  ],
                ),
              ),
              const Positioned(
                right: 14,
                bottom: 16,
                child: Icon(Icons.arrow_forward_rounded, color: Colors.white),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
