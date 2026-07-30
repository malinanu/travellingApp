// Weather screen: real conditions from Tomorrow.io (via GET /weather),
// themed to the traveller's onboarding category — surf/hike/beach/general.
// "Essentials" scope: hero, hourly+precipitation timeline, one
// activity-suitability card, and a 6-cell stat grid. See activity_rules.dart
// for the scoring engine and core/theme/mood.dart for the hero gradient.

import 'dart:math' as math;
import 'dart:ui' as ui;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import '../../../core/api/api_client.dart';
import '../../../core/theme/mood.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/spotto_card.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../core/widgets/stagger_in.dart';
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';
import 'activity_rules.dart';

class WeatherScreen extends ConsumerStatefulWidget {
  const WeatherScreen({super.key});

  @override
  ConsumerState<WeatherScreen> createState() => _WeatherScreenState();
}

class _WeatherScreenState extends ConsumerState<WeatherScreen> {
  final _pageController = PageController();
  int _page = 0;

  @override
  void dispose() {
    _pageController.dispose();
    super.dispose();
  }

  Future<void> _addPin() async {
    final pins = ref.read(weatherPinsProvider);
    if (pins.length >= kMaxWeatherPins) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('You can pin up to 3 locations — remove one to add another.')),
      );
      return;
    }
    final picked = await showModalBottomSheet<PinnedLocation>(
      context: context,
      backgroundColor: Colors.white,
      isScrollControlled: true,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(20))),
      builder: (context) => _LocationSearchSheet(
        places: ref.read(placesProvider),
        excludeIds: pins.map((p) => p.id).toSet(),
      ),
    );
    if (picked == null || !mounted) return;
    ref.read(weatherPinsProvider.notifier).add(picked);
    final target = ref.read(weatherPinsProvider).length; // index of the page just added
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_pageController.hasClients) {
        _pageController.animateToPage(target,
            duration: const Duration(milliseconds: 320), curve: Curves.easeOutCubic);
      }
    });
  }

  void _removePin(String id) {
    ref.read(weatherPinsProvider.notifier).remove(id);
    final maxPage = ref.read(weatherPinsProvider).length;
    if (_page > maxPage) {
      setState(() => _page = maxPage);
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (_pageController.hasClients) _pageController.jumpToPage(maxPage);
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final pins = ref.watch(weatherPinsProvider);
    final pageCount = 1 + pins.length;

    return Scaffold(
      backgroundColor: SpottoColors.sand,
      body: Stack(
        children: [
          PageView.builder(
            controller: _pageController,
            itemCount: pageCount,
            onPageChanged: (i) => setState(() => _page = i),
            itemBuilder: (context, index) {
              if (index == 0) {
                return _WeatherPage(
                  weather: ref.watch(currentWeatherProvider),
                  locationName: 'Nearby',
                  isPinned: false,
                  onRetry: () => ref.invalidate(currentWeatherProvider),
                  onLocationTap: _addPin,
                );
              }
              final pin = pins[index - 1];
              final at = (lat: pin.lat, lng: pin.lng);
              return _WeatherPage(
                weather: ref.watch(weatherSnapshotProvider(at)),
                locationName: pin.name,
                isPinned: true,
                onRetry: () => ref.invalidate(weatherSnapshotProvider(at)),
                onLocationTap: () => _removePin(pin.id),
              );
            },
          ),
          if (pageCount > 1)
            Positioned(
              left: 0,
              right: 0,
              bottom: 0,
              child: SafeArea(
                child: Padding(
                  padding: const EdgeInsets.only(bottom: 10),
                  child: _PageDots(count: pageCount, index: _page),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

/// One page of the swipeable weather deck — "My Location" or a pinned spot.
class _WeatherPage extends StatelessWidget {
  const _WeatherPage({
    required this.weather,
    required this.locationName,
    required this.isPinned,
    required this.onRetry,
    required this.onLocationTap,
  });

  final AsyncValue<WeatherSnapshot> weather;
  final String locationName;
  final bool isPinned;
  final VoidCallback onRetry;
  final VoidCallback onLocationTap;

  @override
  Widget build(BuildContext context) {
    return weather.when(
      loading: () => const _WeatherLoading(),
      error: (e, _) => _WeatherError(message: ApiClient.errorMessage(e), onRetry: onRetry),
      data: (snapshot) => _WeatherBody(
        snapshot: snapshot,
        locationName: locationName,
        isPinned: isPinned,
        onRefresh: () async => onRetry(),
        onLocationTap: onLocationTap,
      ),
    );
  }
}

class _PageDots extends StatelessWidget {
  const _PageDots({required this.count, required this.index});

  final int count;
  final int index;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        for (var i = 0; i < count; i++)
          AnimatedContainer(
            duration: const Duration(milliseconds: 200),
            margin: const EdgeInsets.symmetric(horizontal: 3),
            width: i == index ? 18 : 6,
            height: 6,
            decoration: BoxDecoration(
              color: i == index ? SpottoColors.accent : SpottoColors.line,
              borderRadius: BorderRadius.circular(999),
            ),
          ),
      ],
    );
  }
}

class _WeatherBody extends ConsumerWidget {
  const _WeatherBody({
    required this.snapshot,
    required this.locationName,
    required this.isPinned,
    required this.onRefresh,
    required this.onLocationTap,
  });

  final WeatherSnapshot snapshot;
  final String locationName;
  final bool isPinned;
  final Future<void> Function() onRefresh;
  final VoidCallback onLocationTap;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final prefs = ref.watch(prefsProvider);
    final category = resolveCategory(prefs);
    final ratings = rateHours(category, snapshot.hourly);
    final mood = moodFor(snapshot);
    final heroHeight = math.min(340.0, MediaQuery.sizeOf(context).height * 0.42);

    return RefreshIndicator(
      color: SpottoColors.accent,
      backgroundColor: Colors.white,
      onRefresh: onRefresh,
      child: CustomScrollView(
        slivers: [
          SliverAppBar(
            pinned: true,
            expandedHeight: heroHeight,
            backgroundColor: heroGradient(mood).colors.first,
            // Material 3 tints the app bar background with the theme's
            // primary color (SpottoColors.accent) as it scrolls under
            // content by default — without turning that off, the bar's
            // color visibly drifts away from the mood gradient it's meant
            // to match once you scroll.
            surfaceTintColor: Colors.transparent,
            elevation: 0,
            scrolledUnderElevation: 0,
            systemOverlayStyle: SystemUiOverlayStyle.light,
            iconTheme: const IconThemeData(color: Colors.white),
            titleSpacing: 0,
            title: Text('${snapshot.now.temp.round()}°',
                style: Theme.of(context).textTheme.titleMedium?.copyWith(color: Colors.white)),
            flexibleSpace: FlexibleSpaceBar(
              titlePadding: const EdgeInsetsDirectional.fromSTEB(56, 0, 16, 14),
              background: _Hero(
                snapshot: snapshot,
                mood: mood,
                category: category,
                ratings: ratings,
                locationName: locationName,
                isPinned: isPinned,
                onLocationTap: onLocationTap,
              ),
            ),
          ),
          SliverPadding(
            padding: const EdgeInsets.fromLTRB(16, 18, 16, 0),
            sliver: SliverList(
              delegate: SliverChildListDelegate([
                StaggerIn(index: 0, child: _TimelineCard(snapshot: snapshot)),
                const SizedBox(height: 14),
                StaggerIn(index: 1, child: _ActivityCard(category: category, ratings: ratings)),
                const SizedBox(height: 18),
                Text('Conditions', style: Theme.of(context).textTheme.titleMedium),
                const SizedBox(height: 12),
              ]),
            ),
          ),
          SliverPadding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            sliver: SliverGrid.count(
              crossAxisCount: 2,
              mainAxisSpacing: 12,
              crossAxisSpacing: 12,
              childAspectRatio: 1.28,
              children: [
                for (final (i, cell) in _statCells(snapshot).indexed) StaggerIn(index: i, child: cell),
              ],
            ),
          ),
          SliverToBoxAdapter(
            child: Padding(
              padding: const EdgeInsets.fromLTRB(16, 18, 16, 32),
              child: Center(
                child: Text(
                  'Updated ${_relativeTime(snapshot.fetchedAt)} · Tomorrow.io',
                  style: Theme.of(context).textTheme.labelSmall?.copyWith(color: SpottoColors.muted),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  List<Widget> _statCells(WeatherSnapshot s) {
    final n = s.now;
    final windDir = _compassLabel(n.windDirection);
    return [
      _StatCard(
        icon: Icons.wb_sunny_outlined,
        label: 'UV INDEX',
        value: n.uvIndex.round().toString(),
        unit: 'of 11',
        desc: _uvDesc(n.uvIndex),
      ),
      _StatCard(
        icon: Icons.air,
        label: 'WIND',
        value: n.windSpeed.round().toString(),
        unit: 'km/h',
        desc: 'From the $windDir',
        accessory: SizedBox(width: 34, height: 34, child: CustomPaint(painter: _CompassPainter(n.windDirection))),
      ),
      _StatCard(
        icon: Icons.water_drop_outlined,
        label: 'HUMIDITY',
        value: n.humidity.toString(),
        unit: '%',
        desc: n.feelsLike - n.temp >= 2 ? 'Muggy — feels warmer' : 'Comfortable levels',
      ),
      _StatCard(
        icon: Icons.speed_outlined,
        label: 'PRESSURE',
        value: n.pressure.round().toString(),
        unit: 'hPa',
        desc: 'Normal range',
      ),
      _StatCard(
        icon: Icons.visibility_outlined,
        label: 'VISIBILITY',
        value: n.visibility.round().toString(),
        unit: 'km',
        desc: n.visibility >= 10 ? 'Clear to the horizon' : 'Slightly hazy',
      ),
      _SunCard(sunrise: s.sunrise, sunset: s.sunset),
    ];
  }
}

String _uvDesc(double uv) {
  if (uv >= 11) return 'Extreme — avoid midday sun';
  if (uv >= 8) return 'Very high — seek shade 11–3';
  if (uv >= 6) return 'High — wear sunscreen';
  if (uv >= 3) return 'Moderate exposure risk';
  return 'Low rest of day';
}

String _compassLabel(int deg) {
  const dirs = ['north', 'northeast', 'east', 'southeast', 'south', 'southwest', 'west', 'northwest'];
  return dirs[((deg % 360) / 45).round() % 8];
}

String _relativeTime(DateTime t) {
  final mins = DateTime.now().difference(t).inMinutes;
  if (mins < 1) return 'just now';
  if (mins < 60) return '$mins min ago';
  final hours = (mins / 60).floor();
  return '$hours h ago';
}

class _Hero extends StatelessWidget {
  const _Hero({
    required this.snapshot,
    required this.mood,
    required this.category,
    required this.ratings,
    required this.locationName,
    required this.isPinned,
    required this.onLocationTap,
  });

  final WeatherSnapshot snapshot;
  final SpottoMood mood;
  final ActivityCategory category;
  final List<HourRating> ratings;
  final String locationName;
  final bool isPinned;
  final VoidCallback onLocationTap;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final accent = heroAccent(mood);
    return AnimatedContainer(
      duration: const Duration(milliseconds: 700),
      decoration: BoxDecoration(gradient: heroGradient(mood)),
      child: Stack(
        fit: StackFit.expand,
        children: [
          if (accent != null) DecoratedBox(decoration: BoxDecoration(gradient: accent)),
          Positioned(
            right: -18,
            bottom: -14,
            child: Transform.rotate(
              angle: -0.08,
              child: Icon(spottoIcon(categoryIconName(category)), size: 112, color: Colors.white.withValues(alpha: 0.13)),
            ),
          ),
          SafeArea(
            child: Padding(
              padding: const EdgeInsets.fromLTRB(20, 8, 20, 18),
              child: Column(
                mainAxisAlignment: MainAxisAlignment.end,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Material(
                    color: Colors.transparent,
                    child: InkWell(
                      borderRadius: BorderRadius.circular(999),
                      onTap: onLocationTap,
                      child: Container(
                        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                        decoration: BoxDecoration(
                          color: Colors.white.withValues(alpha: 0.14),
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            const Icon(Icons.place, size: 13, color: Colors.white),
                            const SizedBox(width: 5),
                            Text(locationName, style: text.labelMedium?.copyWith(color: Colors.white)),
                            const SizedBox(width: 5),
                            Icon(isPinned ? Icons.close : Icons.add, size: 14, color: Colors.white),
                          ],
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(height: 14),
                  Row(
                    children: [
                      Icon(spottoIcon(snapshot.now.icon), size: 22, color: SpottoColors.sun),
                      const SizedBox(width: 8),
                      Text(snapshot.now.conditionLabel, style: text.titleMedium?.copyWith(color: Colors.white)),
                    ],
                  ),
                  const SizedBox(height: 2),
                  Text('${snapshot.now.temp.round()}°',
                      style: text.displayLarge?.copyWith(
                        color: Colors.white,
                        fontSize: 88,
                        height: 0.98,
                        letterSpacing: -2.5,
                      )),
                  const SizedBox(height: 6),
                  Text(
                    'H ${snapshot.high.round()}°  ·  L ${snapshot.low.round()}°  ·  Feels ${snapshot.now.feelsLike.round()}°',
                    style: text.labelMedium?.copyWith(color: SpottoColors.paleTeal),
                  ),
                  const SizedBox(height: 12),
                  Text(
                    headline(category, ratings),
                    style: text.bodyMedium?.copyWith(color: Colors.white.withValues(alpha: 0.78)),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _TimelineCard extends StatelessWidget {
  const _TimelineCard({required this.snapshot});

  final WeatherSnapshot snapshot;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final hourly = snapshot.hourly.take(24).toList();
    final totalMm = hourly.fold<double>(0, (sum, h) => sum + h.precipMm);
    final maxMm = hourly.fold<double>(1, (m, h) => h.precipMm > m ? h.precipMm : m);

    return SpottoCard(
      padding: const EdgeInsets.fromLTRB(0, 14, 0, 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Row(
              children: [
                Text('Next 24 hours', style: text.titleMedium),
                const Spacer(),
                if (totalMm > 0.1) Text('${totalMm.toStringAsFixed(1)} mm total', style: text.labelSmall),
              ],
            ),
          ),
          const SizedBox(height: 12),
          if (hourly.isEmpty)
            const Padding(padding: EdgeInsets.symmetric(horizontal: 16), child: Text('No forecast available.'))
          else
            SizedBox(
              height: 156,
              child: SingleChildScrollView(
                scrollDirection: Axis.horizontal,
                padding: const EdgeInsets.only(left: 16, right: 16),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    for (final (i, h) in hourly.indexed)
                      StaggerIn(
                        index: i,
                        axis: Axis.horizontal,
                        child: _HourColumn(hour: h, isNow: i == 0, maxMm: maxMm),
                      ),
                  ],
                ),
              ),
            ),
          if (totalMm <= 0.1 && hourly.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(left: 16, top: 8),
              child: Text('No rain expected in the next 24 hours.',
                  style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
            ),
        ],
      ),
    );
  }
}

class _HourColumn extends StatelessWidget {
  const _HourColumn({required this.hour, required this.isNow, required this.maxMm});

  final HourlyPoint hour;
  final bool isNow;
  final double maxMm;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final barH = 44 * (hour.precipMm / maxMm).clamp(0.0, 1.0);
    final block = Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(isNow ? 'Now' : DateFormat('h a').format(hour.time), style: text.labelSmall),
        const SizedBox(height: 6),
        Icon(spottoIcon(hour.icon), size: 20, color: hour.condition == 'sunny' ? SpottoColors.sun : SpottoColors.accent),
        const SizedBox(height: 6),
        Text('${hour.temp.round()}°', style: text.labelLarge),
        const SizedBox(height: 4),
        SizedBox(
          height: 14,
          child: hour.precipProbability >= 20
              ? Text('${hour.precipProbability}%', style: text.labelSmall?.copyWith(color: SpottoColors.accent))
              : null,
        ),
        const SizedBox(height: 8),
        SizedBox(
          height: 44,
          child: Stack(
            alignment: Alignment.bottomCenter,
            children: [
              Container(width: 3, height: 44, color: SpottoColors.accentTint),
              Align(
                alignment: Alignment.bottomCenter,
                child: Container(
                  width: 14,
                  height: barH,
                  decoration: BoxDecoration(
                    color: SpottoColors.accent.withValues(alpha: 0.85),
                    borderRadius: const BorderRadius.vertical(top: Radius.circular(6)),
                  ),
                ),
              ),
            ],
          ),
        ),
      ],
    );
    return Container(
      width: 56,
      padding: const EdgeInsets.symmetric(vertical: 6),
      decoration: isNow
          ? BoxDecoration(color: SpottoColors.accentTint, borderRadius: BorderRadius.circular(14))
          : null,
      child: block,
    );
  }
}

class _ActivityCard extends StatefulWidget {
  const _ActivityCard({required this.category, required this.ratings});

  final ActivityCategory category;
  final List<HourRating> ratings;

  @override
  State<_ActivityCard> createState() => _ActivityCardState();
}

class _ActivityCardState extends State<_ActivityCard> {
  int? _selected;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final ratings = widget.ratings;
    if (ratings.isEmpty) return const SizedBox.shrink();
    final ribbon = ratings.take(12).toList();
    final activeIndex = _selected ?? 0;
    final active = ribbon[activeIndex.clamp(0, ribbon.length - 1)];
    final window = bestWindow(ratings);
    final footnote = categoryFootnote(widget.category);

    return SpottoCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 34,
                height: 34,
                decoration: BoxDecoration(color: SpottoColors.accentTint, borderRadius: BorderRadius.circular(12)),
                child: Icon(spottoIcon(categoryIconName(widget.category)), size: 18, color: SpottoColors.accent),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(categoryLabel(widget.category), style: text.titleMedium),
                    AnimatedSwitcher(
                      duration: const Duration(milliseconds: 180),
                      child: Text(
                        _selected == null
                            ? (window != null ? 'Best $window' : 'Checking the next few hours')
                            : '${DateFormat('h a').format(active.time)} · ${_levelLabel(active.level)}',
                        key: ValueKey('${_selected ?? -1}'),
                        style: text.bodySmall,
                      ),
                    ),
                  ],
                ),
              ),
              _RatingBadge(level: active.level),
            ],
          ),
          const SizedBox(height: 14),
          Row(
            children: [
              for (final (i, r) in ribbon.indexed)
                Expanded(
                  child: GestureDetector(
                    behavior: HitTestBehavior.opaque,
                    onTap: () => setState(() => _selected = i),
                    child: _RibbonSlot(rating: r, isNow: i == 0, showLabel: i % 3 == 0),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 10),
          Text(
            active.reason.isEmpty
                ? 'Good conditions.'
                : '${_levelLabel(active.level)}${activeIndex == 0 ? '' : ' at ${DateFormat('h a').format(active.time)}'} — ${active.reason}.',
            style: text.bodySmall,
          ),
          if (footnote != null) ...[
            const SizedBox(height: 4),
            Text(footnote, style: text.bodySmall?.copyWith(color: SpottoColors.muted, fontStyle: FontStyle.italic)),
          ],
        ],
      ),
    );
  }

  String _levelLabel(Suitability l) {
    switch (l) {
      case Suitability.good:
        return 'Good';
      case Suitability.fair:
        return 'Fair';
      case Suitability.poor:
        return 'Poor';
    }
  }
}

class _RatingBadge extends StatelessWidget {
  const _RatingBadge({required this.level});

  final Suitability level;

  @override
  Widget build(BuildContext context) {
    final key = level.name;
    final tone = SpottoColors.ratingTones[key]!;
    final tint = SpottoColors.ratingTints[key]!;
    final label = key[0].toUpperCase() + key.substring(1);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(color: tint, borderRadius: BorderRadius.circular(999)),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(width: 6, height: 6, decoration: BoxDecoration(color: tone, shape: BoxShape.circle)),
          const SizedBox(width: 6),
          Text(label, style: Theme.of(context).textTheme.labelMedium?.copyWith(color: tone)),
        ],
      ),
    );
  }
}

class _RibbonSlot extends StatelessWidget {
  const _RibbonSlot({required this.rating, required this.isNow, required this.showLabel});

  final HourRating rating;
  final bool isNow;
  final bool showLabel;

  @override
  Widget build(BuildContext context) {
    final tone = SpottoColors.ratingTones[rating.level.name]!;
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 6,
          height: isNow ? 28 : 22,
          decoration: BoxDecoration(color: tone, borderRadius: BorderRadius.circular(999)),
        ),
        const SizedBox(height: 4),
        SizedBox(
          height: 14,
          child: showLabel
              ? Text(DateFormat('h a').format(rating.time), style: Theme.of(context).textTheme.labelSmall)
              : null,
        ),
      ],
    );
  }
}

class _StatCard extends StatelessWidget {
  const _StatCard({
    required this.icon,
    required this.label,
    required this.value,
    required this.unit,
    required this.desc,
    this.accessory,
  });

  final IconData icon;
  final String label;
  final String value;
  final String unit;
  final String desc;
  final Widget? accessory;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return SpottoCard(
      padding: const EdgeInsets.all(14),
      child: Stack(
        children: [
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Icon(icon, size: 15, color: SpottoColors.muted),
                  const SizedBox(width: 6),
                  Text(label, style: text.labelSmall?.copyWith(color: SpottoColors.muted)),
                ],
              ),
              const Spacer(),
              Row(
                crossAxisAlignment: CrossAxisAlignment.baseline,
                textBaseline: TextBaseline.alphabetic,
                children: [
                  Text(value, style: text.headlineSmall),
                  const SizedBox(width: 4),
                  Text(unit, style: text.labelMedium?.copyWith(color: SpottoColors.muted)),
                ],
              ),
              const SizedBox(height: 4),
              Text(desc, style: text.bodySmall, maxLines: 2, overflow: TextOverflow.ellipsis),
            ],
          ),
          if (accessory != null) Positioned(top: 0, right: 0, child: accessory!),
        ],
      ),
    );
  }
}

class _SunCard extends StatelessWidget {
  const _SunCard({required this.sunrise, required this.sunset});

  final DateTime sunrise;
  final DateTime sunset;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final daylight = sunset.difference(sunrise);
    return SpottoCard(
      padding: const EdgeInsets.all(14),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.wb_twilight, size: 15, color: SpottoColors.muted),
              const SizedBox(width: 6),
              Text('SUN', style: text.labelSmall?.copyWith(color: SpottoColors.muted)),
            ],
          ),
          const Spacer(),
          Row(
            children: [
              const Icon(Icons.arrow_upward, size: 13, color: SpottoColors.sun),
              const SizedBox(width: 4),
              Text(DateFormat('H:mm').format(sunrise), style: text.titleMedium),
            ],
          ),
          const SizedBox(height: 2),
          Row(
            children: [
              const Icon(Icons.arrow_downward, size: 13, color: SpottoColors.sun),
              const SizedBox(width: 4),
              Text(DateFormat('H:mm').format(sunset), style: text.titleMedium),
            ],
          ),
          const SizedBox(height: 4),
          Text('${daylight.inHours}h ${daylight.inMinutes % 60}m of daylight', style: text.bodySmall),
        ],
      ),
    );
  }
}

/// Small instrument-style wind-direction dial — matches the app's existing
/// hand-rolled CustomPainter precedent (e.g. group_sheet.dart's
/// MiniQrPainter) rather than a plain rotated Material icon.
class _CompassPainter extends CustomPainter {
  const _CompassPainter(this.degrees);

  final int degrees;

  @override
  void paint(Canvas canvas, Size size) {
    final center = size.center(Offset.zero);
    final radius = size.width / 2 - 1;

    final ring = Paint()
      ..color = SpottoColors.line
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1.5;
    canvas.drawCircle(center, radius, ring);

    for (var i = 0; i < 4; i++) {
      final angle = i * math.pi / 2;
      final outer = center + Offset(math.sin(angle), -math.cos(angle)) * radius;
      final inner = center + Offset(math.sin(angle), -math.cos(angle)) * (radius - 3);
      canvas.drawLine(inner, outer, ring);
    }

    canvas.save();
    canvas.translate(center.dx, center.dy);
    canvas.rotate(degrees * math.pi / 180);
    final needle = Paint()..color = SpottoColors.accent;
    final path = Path()
      ..moveTo(0, -radius + 4)
      ..lineTo(3.5, 4)
      ..lineTo(0, 0)
      ..lineTo(-3.5, 4)
      ..close();
    canvas.drawPath(path, needle);
    canvas.restore();

    final tp = TextPainter(
      text: const TextSpan(text: 'N', style: TextStyle(fontSize: 7, color: SpottoColors.muted)),
      textDirection: ui.TextDirection.ltr,
    )..layout();
    tp.paint(canvas, center + Offset(-tp.width / 2, -radius - 2));
  }

  @override
  bool shouldRepaint(_CompassPainter oldDelegate) => oldDelegate.degrees != degrees;
}

class _WeatherLoading extends StatelessWidget {
  const _WeatherLoading();

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(gradient: heroGradient(SpottoMood.day)),
      child: const Center(child: CircularProgressIndicator(color: Colors.white)),
    );
  }
}

class _WeatherError extends StatelessWidget {
  const _WeatherError({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.cloud_off_outlined, size: 40, color: SpottoColors.muted),
            const SizedBox(height: 12),
            Text(message, textAlign: TextAlign.center, style: Theme.of(context).textTheme.bodyMedium),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: const Text('Try again')),
          ],
        ),
      ),
    );
  }
}

/// Search sheet for pinning a new weather location — searches the distinct
/// regions present in the already-loaded places list (same derivation as
/// the Spots screen's region filter), not individual spots: weather is
/// regional, and every place within a region already rounds to the same
/// server-side cache cell, so any one of them is an equally accurate query
/// point.
class _LocationSearchSheet extends StatefulWidget {
  const _LocationSearchSheet({required this.places, required this.excludeIds});

  final List<Place> places;
  final Set<String> excludeIds;

  @override
  State<_LocationSearchSheet> createState() => _LocationSearchSheetState();
}

class _LocationSearchSheetState extends State<_LocationSearchSheet> {
  final _controller = TextEditingController();
  String _query = '';

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  List<PinnedLocation> get _regions {
    final seen = <String>{};
    final out = <PinnedLocation>[];
    for (final p in widget.places) {
      if (p.regionId.isEmpty || !seen.add(p.regionId) || widget.excludeIds.contains(p.regionId)) continue;
      out.add(PinnedLocation(id: p.regionId, name: p.region, lat: p.lat, lng: p.lng));
    }
    out.sort((a, b) => a.name.compareTo(b.name));
    return out;
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final q = _query.trim().toLowerCase();
    final regions = _regions;
    final results = q.isEmpty ? regions : regions.where((r) => r.name.toLowerCase().contains(q)).toList();
    return SafeArea(
      child: Padding(
        padding: EdgeInsets.only(bottom: MediaQuery.viewInsetsOf(context).bottom),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
              child: TextField(
                controller: _controller,
                autofocus: true,
                onChanged: (v) => setState(() => _query = v),
                decoration: const InputDecoration(
                  hintText: 'Search a region — Arugam Bay, Ella…',
                  prefixIcon: Icon(Icons.search, size: 18, color: SpottoColors.muted),
                ),
              ),
            ),
            Flexible(
              child: results.isEmpty
                  ? Padding(
                      padding: const EdgeInsets.all(24),
                      child: Text(
                        regions.isEmpty ? 'No other regions to pin yet.' : 'No matching region.',
                        style: text.bodyMedium?.copyWith(color: SpottoColors.muted),
                      ),
                    )
                  : ListView(
                      shrinkWrap: true,
                      children: [
                        for (final r in results)
                          ListTile(
                            leading: const Icon(Icons.place_outlined, color: SpottoColors.accent),
                            title: Text(r.name),
                            onTap: () => Navigator.pop(context, r),
                          ),
                      ],
                    ),
            ),
            const SizedBox(height: 8),
          ],
        ),
      ),
    );
  }
}
