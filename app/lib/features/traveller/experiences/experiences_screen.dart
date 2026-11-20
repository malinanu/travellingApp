// Experiences & events: tours, classes, trips and nights out run by local
// hosts and venues, bookable in the app (see GET /experiences). Each card
// opens the host's place sheet, where the booking form lives.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../core/api/api_client.dart';
import '../../../core/l10n/l10n_provider.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';
import '../../../data/vendor_categories.dart';
import '../place_detail/business_drawer.dart';
import '../profile/profile_screen.dart' show circleButton;

class ExperiencesScreen extends ConsumerStatefulWidget {
  const ExperiencesScreen({super.key});

  @override
  ConsumerState<ExperiencesScreen> createState() => _ExperiencesScreenState();
}

class _ExperiencesScreenState extends ConsumerState<ExperiencesScreen> {
  String _category = '';
  bool _opening = false;

  @override
  Widget build(BuildContext context) {
    final l10n = ref.watch(l10nProvider);
    final text = Theme.of(context).textTheme;
    final position = ref.watch(devicePositionProvider).value?.$1;
    final categories = [
      for (final c in ref.watch(categoriesProvider))
        if (c.bookingType == BookingType.experience || c.bookingType == BookingType.event) c,
    ];
    final listings = ref.watch(experiencesProvider((category: _category, lat: position?.lat, lng: position?.lng)));

    return Scaffold(
      backgroundColor: SpottoColors.sand,
      body: SafeArea(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 4),
              child: Row(
                children: [
                  circleButton(Icons.chevron_left, () => context.pop()),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(l10n.experiencesTitle, style: text.titleMedium?.copyWith(fontSize: 17)),
                        Text(l10n.experiencesSubtitle, style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                      ],
                    ),
                  ),
                ],
              ),
            ),
            SizedBox(
              height: 46,
              child: ListView(
                scrollDirection: Axis.horizontal,
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
                children: [
                  _CategoryChip(
                    label: l10n.allCategoriesFilter,
                    icon: Icons.auto_awesome_outlined,
                    selected: _category.isEmpty,
                    onTap: () => setState(() => _category = ''),
                  ),
                  for (final c in categories)
                    _CategoryChip(
                      label: c.label,
                      icon: spottoIcon(c.icon),
                      selected: _category == c.id,
                      onTap: () => setState(() => _category = c.id),
                    ),
                ],
              ),
            ),
            Expanded(
              child: listings.when(
                data: (items) => items.isEmpty
                    ? Center(
                        child: Padding(
                          padding: const EdgeInsets.all(32),
                          child: Text(
                            l10n.experiencesEmpty,
                            textAlign: TextAlign.center,
                            style: text.bodyMedium?.copyWith(color: SpottoColors.muted),
                          ),
                        ),
                      )
                    : ListView.separated(
                        padding: const EdgeInsets.fromLTRB(16, 8, 16, 32),
                        itemCount: items.length,
                        separatorBuilder: (_, _) => const SizedBox(height: 10),
                        itemBuilder: (context, i) => _ExperienceCard(
                          listing: items[i],
                          onTap: _opening ? null : () => _open(items[i]),
                        ),
                      ),
                loading: () => const Center(child: CircularProgressIndicator()),
                error: (e, _) => Center(
                  child: Padding(
                    padding: const EdgeInsets.all(32),
                    child: Text(ApiClient.errorMessage(e), textAlign: TextAlign.center),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<void> _open(ExperienceListing listing) async {
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _opening = true);
    try {
      final place = await ref.read(placeDetailsProvider(listing.placeId).future);
      if (mounted) await showBusinessDrawer(context, place);
    } catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
    } finally {
      if (mounted) setState(() => _opening = false);
    }
  }
}

class _CategoryChip extends StatelessWidget {
  final String label;
  final IconData icon;
  final bool selected;
  final VoidCallback onTap;
  const _CategoryChip({required this.label, required this.icon, required this.selected, required this.onTap});

  @override
  Widget build(BuildContext context) => Padding(
        padding: const EdgeInsets.only(right: 6),
        child: ChoiceChip(
          avatar: Icon(icon, size: 16, color: selected ? Colors.white : SpottoColors.ink),
          label: Text(label, style: TextStyle(fontWeight: FontWeight.w700, color: selected ? Colors.white : SpottoColors.ink)),
          selected: selected,
          selectedColor: SpottoColors.accent,
          showCheckmark: false,
          onSelected: (_) => onTap(),
        ),
      );
}

class _ExperienceCard extends ConsumerWidget {
  final ExperienceListing listing;
  final VoidCallback? onTap;
  const _ExperienceCard({required this.listing, required this.onTap});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = ref.watch(l10nProvider);
    final s = listing.service;
    final next = listing.nextOccurrence;
    final meta = [
      listing.vendorName,
      if (listing.town.isNotEmpty) listing.town,
      if (listing.distanceKm != null) '${listing.distanceKm!.toStringAsFixed(1)} km',
    ].join(' · ');
    return PressScale(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.all(14),
        decoration: BoxDecoration(color: Colors.white, borderRadius: BorderRadius.circular(16)),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(s.name, style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w800)),
                ),
                if (listing.vendorKind == 'host')
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                    decoration: BoxDecoration(
                      color: SpottoColors.accent.withValues(alpha: 0.12),
                      borderRadius: BorderRadius.circular(999),
                    ),
                    child: Text(
                      l10n.localHostBadge,
                      style: const TextStyle(fontSize: 10.5, fontWeight: FontWeight.w800, color: SpottoColors.accent),
                    ),
                  ),
              ],
            ),
            const SizedBox(height: 2),
            Text(meta, style: const TextStyle(fontSize: 12.5, fontWeight: FontWeight.w600, color: SpottoColors.muted)),
            if (s.description.isNotEmpty) ...[
              const SizedBox(height: 6),
              Text(s.description, maxLines: 2, overflow: TextOverflow.ellipsis, style: const TextStyle(fontSize: 13)),
            ],
            const SizedBox(height: 8),
            Row(
              children: [
                Text(
                  servicePriceLine(l10n, s),
                  style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w800, color: SpottoColors.ink),
                ),
                const Spacer(),
                if (next != null)
                  Text(
                    l10n.nextDateLabel(DateFormat('EEE d MMM · HH:mm').format(next.start)),
                    style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w700, color: SpottoColors.accent),
                  ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
