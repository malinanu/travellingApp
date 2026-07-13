// Business profile editor — port of ProfileEditor.jsx: type, Wi-Fi,
// amenities, best times. Type changes drive the Menu entry's visibility.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/mock/vendor_seed.dart';
import '../../../data/repositories/repositories.dart';
import '../widgets/vendor_common.dart';

class VendorProfileEditorScreen extends ConsumerWidget {
  const VendorProfileEditorScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final business = ref.watch(vendorAuthProvider);
    final profile = ref.watch(vendorProfileProvider);
    final profileNotifier = ref.read(vendorProfileProvider.notifier);

    return Scaffold(
      appBar: AppBar(
          title: const Text('Business profile'), backgroundColor: SpottoColors.sand),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(20, 8, 20, 24),
        children: [
          // business type
          VendorCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Business type', style: text.titleMedium?.copyWith(fontSize: 16)),
                Text('Food types unlock the Menu tab',
                    style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                const SizedBox(height: 12),
                GridView.count(
                  shrinkWrap: true,
                  physics: const NeverScrollableScrollPhysics(),
                  crossAxisCount: 2,
                  mainAxisSpacing: 10,
                  crossAxisSpacing: 10,
                  childAspectRatio: 2.8,
                  children: [
                    for (final t in businessTypes)
                      PressScale(
                        scale: 0.97,
                        onTap: () {
                          final b = business;
                          if (b != null) {
                            ref
                                .read(vendorAuthProvider.notifier)
                                .updateBusiness(
                                    b.copyWith(typeId: t.id, type: t.label));
                          }
                        },
                        child: Container(
                          padding: const EdgeInsets.symmetric(horizontal: 12),
                          decoration: BoxDecoration(
                            color: business?.typeId == t.id
                                ? SpottoColors.accent
                                : SpottoColors.sand,
                            borderRadius: BorderRadius.circular(14),
                          ),
                          child: Row(
                            children: [
                              Icon(spottoIcon(t.icon),
                                  size: 18,
                                  color: business?.typeId == t.id
                                      ? Colors.white
                                      : SpottoColors.ink),
                              const SizedBox(width: 8),
                              Expanded(
                                child: Text(t.label,
                                    maxLines: 1,
                                    overflow: TextOverflow.ellipsis,
                                    style: TextStyle(
                                        fontSize: 12.5,
                                        fontWeight: FontWeight.w800,
                                        color: business?.typeId == t.id
                                            ? Colors.white
                                            : SpottoColors.ink)),
                              ),
                              if (t.food)
                                Icon(Icons.restaurant_menu,
                                    size: 13,
                                    color: business?.typeId == t.id
                                        ? Colors.white70
                                        : SpottoColors.muted),
                            ],
                          ),
                        ),
                      ),
                  ],
                ),
              ],
            ),
          ),
          const SizedBox(height: 16),

          // wifi
          VendorCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text('Wi-Fi', style: text.titleMedium?.copyWith(fontSize: 16)),
                          Text(
                              profile.wifiEnabled
                                  ? '${profile.wifiScore}/10 · ${profile.wifiSpeed} Mbps'
                                  : 'Hidden from travellers',
                              style: text.bodySmall
                                  ?.copyWith(color: SpottoColors.muted)),
                        ],
                      ),
                    ),
                    Switch(
                      value: profile.wifiEnabled,
                      activeTrackColor: SpottoColors.accent,
                      onChanged: (v) =>
                          profileNotifier.set(profile.copyWith(wifiEnabled: v)),
                    ),
                  ],
                ),
                if (profile.wifiEnabled) ...[
                  const SizedBox(height: 6),
                  Row(
                    children: [
                      const Icon(Icons.wifi, size: 17, color: SpottoColors.accent),
                      Expanded(
                        child: Slider(
                          value: profile.wifiSpeed.toDouble(),
                          min: 5,
                          max: 150,
                          activeColor: SpottoColors.accent,
                          onChanged: (v) => profileNotifier.set(profile.copyWith(
                              wifiSpeed: v.round(),
                              wifiScore: (v / 150 * 10 * 10).round() / 10)),
                        ),
                      ),
                      SizedBox(
                        width: 66,
                        child: Text('${profile.wifiSpeed} Mbps',
                            textAlign: TextAlign.end, style: text.labelMedium),
                      ),
                    ],
                  ),
                ],
              ],
            ),
          ),
          const SizedBox(height: 16),

          // best times + sea view
          VendorCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Best times to visit',
                    style: text.titleMedium?.copyWith(fontSize: 16)),
                const SizedBox(height: 10),
                Row(
                  children: [
                    for (final (i, slot) in const [
                      (id: 'morning', label: 'Morning'),
                      (id: 'afternoon', label: 'Afternoon'),
                      (id: 'evening', label: 'Evening'),
                    ].indexed) ...[
                      if (i > 0) const SizedBox(width: 8),
                      Expanded(
                        child: PressScale(
                          scale: 0.97,
                          onTap: () {
                            final times = [...profile.bestTimes];
                            times.contains(slot.id)
                                ? times.remove(slot.id)
                                : times.add(slot.id);
                            profileNotifier
                                .set(profile.copyWith(bestTimes: times));
                          },
                          child: Container(
                            padding: const EdgeInsets.symmetric(vertical: 10),
                            alignment: Alignment.center,
                            decoration: BoxDecoration(
                              color: profile.bestTimes.contains(slot.id)
                                  ? SpottoColors.accent
                                  : Colors.white,
                              borderRadius: BorderRadius.circular(12),
                              border: Border.all(
                                  color: profile.bestTimes.contains(slot.id)
                                      ? SpottoColors.accent
                                      : SpottoColors.line,
                                  width: 1.5),
                            ),
                            child: Text(slot.label,
                                style: TextStyle(
                                    fontSize: 12.5,
                                    fontWeight: FontWeight.w800,
                                    color: profile.bestTimes.contains(slot.id)
                                        ? Colors.white
                                        : SpottoColors.ink)),
                          ),
                        ),
                      ),
                    ],
                  ],
                ),
                const SizedBox(height: 14),
                Row(
                  children: [
                    Expanded(
                      child: Text('Sea view seating', style: text.titleSmall),
                    ),
                    Switch(
                      value: profile.seaView,
                      activeTrackColor: SpottoColors.accent,
                      onChanged: (v) =>
                          profileNotifier.set(profile.copyWith(seaView: v)),
                    ),
                  ],
                ),
                Row(
                  children: [
                    const Icon(Icons.schedule, size: 16, color: SpottoColors.muted),
                    const SizedBox(width: 8),
                    Text('Open ${profile.opens} – ${profile.closes}',
                        style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                  ],
                ),
              ],
            ),
          ),
          const SizedBox(height: 16),

          // amenities
          VendorCard(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Amenities', style: text.titleMedium?.copyWith(fontSize: 16)),
                Text('Shown as tags on your traveller listing',
                    style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                const SizedBox(height: 12),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    for (final a in const [
                      'Power outlets', 'Standing desks', 'Air-con loft',
                      'Specialty coffee', 'Quiet zone', 'Sea-view deck',
                      'Board storage', 'Live music', 'Vegan options',
                    ])
                      PressScale(
                        scale: 0.95,
                        onTap: () {
                          final list = [...profile.amenities];
                          list.contains(a) ? list.remove(a) : list.add(a);
                          profileNotifier.set(profile.copyWith(amenities: list));
                        },
                        child: Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 12, vertical: 7),
                          decoration: BoxDecoration(
                            color: profile.amenities.contains(a)
                                ? SpottoColors.accent
                                : SpottoColors.sand,
                            borderRadius: BorderRadius.circular(999),
                          ),
                          child: Text(a,
                              style: TextStyle(
                                  fontSize: 12.5,
                                  fontWeight: FontWeight.w700,
                                  color: profile.amenities.contains(a)
                                      ? Colors.white
                                      : SpottoColors.ink)),
                        ),
                      ),
                  ],
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
