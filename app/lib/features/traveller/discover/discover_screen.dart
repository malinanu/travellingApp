// Discover tab: stylised Weligama Bay map with pins, weather bar and
// time-of-day filter — port of MapView.jsx + SatelliteMap.jsx (M1 scope).

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/mock/places.dart' as data;
import '../../../data/repositories/repositories.dart';
import '../group/group_sheet.dart';
import '../place_detail/business_drawer.dart';
import '../trip/widgets/trip_common.dart';

class DiscoverScreen extends ConsumerWidget {
  const DiscoverScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final time = ref.watch(timeFilterProvider);
    final weather = data.weatherKinds[ref.watch(weatherProvider)]!;
    final text = Theme.of(context).textTheme;
    final groupMode = ref.watch(groupModeProvider);
    final group = ref.watch(groupProvider);
    final members = group.memberIds
        .map((id) => data.travelers.where((t) => t.id == id).firstOrNull)
        .nonNulls
        .toList();

    return Stack(
      fit: StackFit.expand,
      children: [
        // base map
        SvgPicture.asset('assets/map/weligama_bay.svg', fit: BoxFit.cover),

        // time-of-day lighting overlay (700ms like the prototype)
        AnimatedContainer(
          duration: const Duration(milliseconds: 700),
          decoration: BoxDecoration(gradient: _lighting(time)),
        ),

        // pins
        ...ref.watch(placesProvider).map((p) {
          final emphasized = p.bestTimes.contains(time);
          return Align(
            alignment: FractionalOffset(p.x / 100, p.y / 100),
            child: AnimatedOpacity(
              duration: const Duration(milliseconds: 400),
              opacity: emphasized ? 1 : 0.45,
              child: PressScale(
                onTap: () => showBusinessDrawer(context, p),
                child: _Pin(
                  color: SpottoColors.pinTones[p.color] ?? SpottoColors.accent,
                  icon: spottoIcon(p.icon),
                  hasDeal: p.deal.active,
                  emphasized: emphasized,
                ),
              ),
            ),
          );
        }),

        // group member overlay
        if (groupMode)
          ...members.where((m) => m.x > 0 || m.y > 0).map(
                (m) => Align(
                  alignment: FractionalOffset(m.x / 100, m.y / 100),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      AvatarBubble(initial: m.initial, color: Color(m.color), size: 34),
                      Container(
                        padding:
                            const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                        decoration: BoxDecoration(
                          color: Colors.white,
                          borderRadius: BorderRadius.circular(8),
                          boxShadow: const [
                            BoxShadow(
                                color: SpottoColors.inkShadow,
                                blurRadius: 6,
                                offset: Offset(0, 2))
                          ],
                        ),
                        child: Text(m.name,
                            style: const TextStyle(
                                fontSize: 10,
                                fontWeight: FontWeight.w800,
                                color: SpottoColors.ink)),
                      ),
                    ],
                  ),
                ),
              ),

        // top: weather bar
        SafeArea(
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
            child: Column(
              children: [
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(18),
                    boxShadow: const [
                      BoxShadow(color: SpottoColors.inkShadow, blurRadius: 12, offset: Offset(0, 4)),
                    ],
                  ),
                  child: Row(
                    children: [
                      Icon(spottoIcon(weather.icon), color: SpottoColors.sun),
                      const SizedBox(width: 10),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text('${weather.label} · ${weather.temp}', style: text.labelLarge),
                            Text(weather.tip, style: text.bodySmall),
                          ],
                        ),
                      ),
                      // demo affordance: tap to cycle weather
                      IconButton(
                        icon: const Icon(Icons.refresh, size: 18, color: SpottoColors.muted),
                        onPressed: () {
                          final order = ['sunny', 'cloudy', 'rainy'];
                          final cur = ref.read(weatherProvider);
                          ref.read(weatherProvider.notifier).state =
                              order[(order.indexOf(cur) + 1) % order.length];
                        },
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 8),
                Row(
                  children: [
                    // group mode toggle (long-press opens the sheet directly)
                    PressScale(
                      scale: 0.95,
                      onTap: () {
                        final on = ref.read(groupModeProvider);
                        if (!on) {
                          ref.read(groupModeProvider.notifier).state = true;
                          showGroupSheet(context);
                        } else {
                          ref.read(groupModeProvider.notifier).state = false;
                        }
                      },
                      child: Container(
                        height: 40,
                        padding: const EdgeInsets.symmetric(horizontal: 10),
                        decoration: BoxDecoration(
                          color: groupMode ? SpottoColors.accent : Colors.white,
                          borderRadius: BorderRadius.circular(16),
                          boxShadow: const [
                            BoxShadow(
                                color: SpottoColors.inkShadow,
                                blurRadius: 10,
                                offset: Offset(0, 3)),
                          ],
                        ),
                        child: Row(
                          children: [
                            Icon(Icons.group_outlined,
                                size: 17,
                                color:
                                    groupMode ? Colors.white : SpottoColors.accent),
                            const SizedBox(width: 6),
                            Text(
                              groupMode
                                  ? '${group.name} · ${members.length + 1}'
                                  : 'Group',
                              style: TextStyle(
                                  fontSize: 12.5,
                                  fontWeight: FontWeight.w800,
                                  color: groupMode ? Colors.white : SpottoColors.ink),
                            ),
                          ],
                        ),
                      ),
                    ),
                    if (groupMode) ...[
                      const SizedBox(width: 8),
                      PressScale(
                        scale: 0.95,
                        onTap: () => showGroupSheet(context),
                        child: Container(
                          width: 40,
                          height: 40,
                          decoration: BoxDecoration(
                            color: Colors.white,
                            borderRadius: BorderRadius.circular(16),
                            boxShadow: const [
                              BoxShadow(
                                  color: SpottoColors.inkShadow,
                                  blurRadius: 10,
                                  offset: Offset(0, 3)),
                            ],
                          ),
                          child: const Icon(Icons.person_add_outlined,
                              size: 17, color: SpottoColors.accent),
                        ),
                      ),
                    ],
                    const Spacer(),
                    PressScale(
                      scale: 0.95,
                      onTap: () => context.push('/t/profile'),
                      child: Container(
                        width: 40,
                        height: 40,
                        alignment: Alignment.center,
                        decoration: BoxDecoration(
                          gradient: const LinearGradient(
                            begin: Alignment.topLeft,
                            end: Alignment.bottomRight,
                            colors: [SpottoColors.accent, SpottoColors.accent2],
                          ),
                          shape: BoxShape.circle,
                          border: Border.all(color: Colors.white, width: 2.5),
                          boxShadow: const [
                            BoxShadow(
                                color: SpottoColors.inkShadow,
                                blurRadius: 10,
                                offset: Offset(0, 3)),
                          ],
                        ),
                        child: Text(data.me.initial,
                            style: const TextStyle(
                                fontSize: 13,
                                fontWeight: FontWeight.w800,
                                color: Colors.white)),
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),

        // bottom: daily quest chip + time filter pills (above the nav bar)
        Positioned(
          left: 16,
          right: 16,
          bottom: 16,
          child: Column(
            children: [
              _QuestChip(checkIns: ref.watch(checkInsProvider).length),
              const SizedBox(height: 8),
              Container(
            padding: const EdgeInsets.all(6),
            decoration: BoxDecoration(
              color: Colors.white,
              borderRadius: BorderRadius.circular(999),
              boxShadow: const [
                BoxShadow(color: SpottoColors.inkShadow, blurRadius: 12, offset: Offset(0, 4)),
              ],
            ),
            child: Row(
              children: [
                for (final t in data.timeSlots)
                  Expanded(
                    child: PressScale(
                      onTap: () => ref.read(timeFilterProvider.notifier).state = t.id,
                      child: AnimatedContainer(
                        duration: const Duration(milliseconds: 200),
                        padding: const EdgeInsets.symmetric(vertical: 9),
                        decoration: BoxDecoration(
                          color: time == t.id ? SpottoColors.accent : Colors.transparent,
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: Column(
                          children: [
                            Icon(spottoIcon(t.icon),
                                size: 18,
                                color: time == t.id ? Colors.white : SpottoColors.muted),
                            Text(t.label,
                                style: text.labelSmall?.copyWith(
                                    color: time == t.id ? Colors.white : SpottoColors.muted)),
                          ],
                        ),
                      ),
                    ),
                  ),
              ],
            ),
              ),
            ],
          ),
        ),
      ],
    );
  }

  LinearGradient _lighting(String time) => switch (time) {
        'morning' => LinearGradient(
            begin: Alignment.topRight,
            end: Alignment.bottomLeft,
            colors: [
              const Color(0xFFFFE0A6).withValues(alpha: 0.35),
              const Color(0xFFFFEECD).withValues(alpha: 0.08),
              Colors.transparent,
            ],
          ),
        'evening' => LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [
              const Color(0xFFFF9646).withValues(alpha: 0.42),
              const Color(0xFFFF5E78).withValues(alpha: 0.22),
              const Color(0xFF46285A).withValues(alpha: 0.34),
              const Color(0xFF231637).withValues(alpha: 0.45),
            ],
          ),
        _ => LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [
              Colors.white.withValues(alpha: 0.16),
              Colors.transparent,
            ],
          ),
      };
}

/// Daily quest progress: check in at [goal] artisan spots for a free tea.
class _QuestChip extends StatelessWidget {
  final int checkIns;
  const _QuestChip({required this.checkIns});

  static const goal = 2;

  @override
  Widget build(BuildContext context) {
    final done = checkIns.clamp(0, goal);
    final complete = done >= goal;

    return PressScale(
      scale: 0.98,
      onTap: () => ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text(complete
            ? 'Quest complete! Show this at any partner café for your free Ceylon tea.'
            : 'Check in at Clay & Co Pottery Studio and Sea Salt Batik Atelier to finish today\'s quest.'),
      )),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        decoration: BoxDecoration(
          color: complete ? SpottoColors.ink : Colors.white,
          borderRadius: BorderRadius.circular(18),
          boxShadow: const [
            BoxShadow(color: SpottoColors.inkShadow, blurRadius: 12, offset: Offset(0, 4)),
          ],
        ),
        child: Row(
          children: [
            Icon(complete ? Icons.emoji_events : Icons.explore,
                size: 18, color: SpottoColors.sun),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    complete
                        ? 'Free Ceylon Tea unlocked!'
                        : 'Daily quest · visit $goal artisan spots',
                    style: TextStyle(
                        fontSize: 12.5,
                        fontWeight: FontWeight.w800,
                        color: complete ? Colors.white : SpottoColors.ink),
                  ),
                  const SizedBox(height: 4),
                  ClipRRect(
                    borderRadius: BorderRadius.circular(999),
                    child: Container(
                      height: 5,
                      color: complete
                          ? Colors.white.withValues(alpha: 0.25)
                          : SpottoColors.ink.withValues(alpha: 0.08),
                      child: AnimatedFractionallySizedBox(
                        duration: const Duration(milliseconds: 400),
                        alignment: Alignment.centerLeft,
                        widthFactor: done / goal,
                        child: Container(color: SpottoColors.sun),
                      ),
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 10),
            Text('$done/$goal',
                style: TextStyle(
                    fontSize: 12.5,
                    fontWeight: FontWeight.w800,
                    color: complete ? SpottoColors.sun : SpottoColors.muted)),
          ],
        ),
      ),
    );
  }
}

class _Pin extends StatelessWidget {
  final Color color;
  final IconData icon;
  final bool hasDeal;
  final bool emphasized;

  const _Pin({required this.color, required this.icon, required this.hasDeal, required this.emphasized});

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 52,
      height: 58,
      child: Stack(
        alignment: Alignment.topCenter,
        children: [
          AnimatedContainer(
            duration: const Duration(milliseconds: 300),
            width: emphasized ? 44 : 38,
            height: emphasized ? 44 : 38,
            decoration: BoxDecoration(
              color: color,
              shape: BoxShape.circle,
              border: Border.all(color: Colors.white, width: 3),
              boxShadow: const [
                BoxShadow(color: SpottoColors.inkShadow, blurRadius: 8, offset: Offset(0, 3)),
              ],
            ),
            child: Icon(icon, color: Colors.white, size: emphasized ? 20 : 17),
          ),
          if (hasDeal)
            Positioned(
              right: 2,
              top: 0,
              child: Container(
                width: 14,
                height: 14,
                decoration: BoxDecoration(
                  color: SpottoColors.sun,
                  shape: BoxShape.circle,
                  border: Border.all(color: Colors.white, width: 2),
                ),
              ),
            ),
        ],
      ),
    );
  }
}
