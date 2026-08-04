// Weather screen: real conditions from Tomorrow.io or Open-Meteo (via GET /weather),
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
import '../../../core/api/google_places_service.dart';
import '../../../core/theme/mood.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/spotto_card.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../core/widgets/stagger_in.dart';
import '../../../data/mock/sri_lanka_locations.dart';
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
    final batchAsync = ref.watch(batchedWeatherDeckProvider);

    return Scaffold(
      backgroundColor: SpottoColors.sand,
      body: Stack(
        children: [
          PageView.builder(
            controller: _pageController,
            itemCount: pageCount,
            onPageChanged: (i) => setState(() => _page = i),
            itemBuilder: (context, index) {
              final String id = index == 0 ? 'current' : pins[index - 1].id;
              final String name = index == 0 ? 'My Location' : pins[index - 1].name;
              final bool isPinned = index > 0;
              final VoidCallback onRetry = () => ref.invalidate(batchedWeatherDeckProvider);
              final VoidCallback onTap = index == 0 ? _addPin : () => _removePin(id);

              final AsyncValue<WeatherSnapshot> weatherValue = batchAsync.whenData((batchMap) {
                final snap = batchMap[id];
                if (snap != null) return snap;
                throw Exception('Weather data unavailable for $name');
              });

              return _WeatherPage(
                weather: weatherValue,
                locationName: name,
                isPinned: isPinned,
                onRetry: onRetry,
                onLocationTap: onTap,
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
      data: (s) => _WeatherPageContent(
        snap: s,
        locationName: locationName,
        isPinned: isPinned,
        onLocationTap: onLocationTap,
      ),
      loading: () => const Scaffold(
        backgroundColor: SpottoColors.sand,
        body: Center(child: CircularProgressIndicator(color: SpottoColors.accent)),
      ),
      error: (err, stack) => _WeatherErrorState(
        locationName: locationName,
        error: err,
        onRetry: onRetry,
      ),
    );
  }
}

class _WeatherPageContent extends ConsumerStatefulWidget {
  const _WeatherPageContent({
    required this.snap,
    required this.locationName,
    required this.isPinned,
    required this.onLocationTap,
  });

  final WeatherSnapshot snap;
  final String locationName;
  final bool isPinned;
  final VoidCallback onLocationTap;

  @override
  ConsumerState<_WeatherPageContent> createState() => _WeatherPageContentState();
}

class _WeatherPageContentState extends ConsumerState<_WeatherPageContent> {
  final _scrollController = ScrollController();
  double _scrollY = 0;

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _scrollController.removeListener(_onScroll);
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {

    if (_scrollController.hasClients) {
      setState(() => _scrollY = _scrollController.offset);
    }
  }

  @override
  Widget build(BuildContext context) {
    final snap = widget.snap;
    final prefs = ref.watch(prefsProvider);
    final mainStyle = (prefs?.styles ?? const ['nomad']).first;
    final mood = moodFor(snap.now.condition, snap.now.cloudCoverPct, DateTime.now());
    final text = Theme.of(context).textTheme;

    return Stack(
      children: [
        // Background Mood Gradient
        Positioned.fill(
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 600),
            decoration: BoxDecoration(gradient: mood.gradient),
          ),
        ),

        CustomScrollView(
          controller: _scrollController,
          slivers: [
            // Hero App Bar with parallax & Location title
            SliverAppBar(
              expandedHeight: 280,
              pinned: true,
              backgroundColor: Colors.transparent,
              elevation: 0,
              leading: IconButton(
                icon: Container(
                  padding: const EdgeInsets.all(6),
                  decoration: const BoxDecoration(color: Colors.white24, shape: BoxShape.circle),
                  child: const Icon(Icons.arrow_back, color: Colors.white, size: 20),
                ),
                onPressed: () => Navigator.maybePop(context),
              ),
              title: Opacity(
                opacity: (_scrollY / 180).clamp(0.0, 1.0),
                child: Text(
                  widget.locationName,
                  style: text.titleMedium?.copyWith(color: Colors.white, fontWeight: FontWeight.bold),
                ),
              ),
              actions: [
                Padding(
                  padding: const EdgeInsets.only(right: 12),
                  child: IconButton(
                    icon: Container(
                      padding: const EdgeInsets.all(6),
                      decoration: const BoxDecoration(color: Colors.white24, shape: BoxShape.circle),
                      child: Icon(
                        widget.isPinned ? Icons.bookmark_remove_rounded : Icons.bookmark_add_rounded,
                        color: Colors.white,
                        size: 20,
                      ),
                    ),
                    onPressed: widget.onLocationTap,
                  ),
                ),
              ],
              flexibleSpace: FlexibleSpaceBar(
                background: SafeArea(
                  child: Padding(
                    padding: const EdgeInsets.fromLTRB(24, 48, 24, 16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          children: [
                            const Icon(Icons.location_on, color: Colors.white70, size: 16),
                            const SizedBox(width: 4),
                            Text(
                              widget.locationName,
                              style: text.titleMedium?.copyWith(
                                color: Colors.white,
                                fontWeight: FontWeight.bold,
                                letterSpacing: 0.5,
                              ),
                            ),
                          ],
                        ),
                        const Spacer(),
                        Row(
                          crossAxisAlignment: CrossAxisAlignment.end,
                          children: [
                            Text(
                              '${snap.now.Temp.round()}°',
                              style: text.displayLarge?.copyWith(
                                fontSize: 64,
                                color: Colors.white,
                                fontWeight: FontWeight.bold,
                                height: 1,
                              ),
                            ),
                            const SizedBox(width: 16),
                            Padding(
                              padding: const EdgeInsets.only(bottom: 8),
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text(
                                    snap.now.ConditionLabel,
                                    style: text.titleMedium?.copyWith(
                                      color: Colors.white,
                                      fontWeight: FontWeight.bold,
                                    ),
                                  ),
                                  Text(
                                    'H: ${snap.High.round()}°  L: ${snap.Low.round()}°',
                                    style: text.bodySmall?.copyWith(color: Colors.white70),
                                  ),
                                ],
                              ),
                            ),
                          ],
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),

            // Main Content Body
            SliverToBoxAdapter(
              child: Container(
                decoration: const BoxDecoration(
                  color: SpottoColors.sand,
                  borderRadius: BorderRadius.vertical(top: Radius.circular(28)),
                ),
                padding: const EdgeInsets.fromLTRB(20, 24, 20, 48),
                child: StaggerIn(
                  children: [
                    // 1. Activity suitability rule engine
                    _ActivityRuleCard(snap: snap, style: mainStyle),
                    const SizedBox(height: 20),

                    // 2. Hourly forecast timeline
                    _HourlyForecastCard(hourly: snap.Hourly),
                    const SizedBox(height: 20),

                    // 3. Stats 6-cell grid
                    _StatsGrid(snap: snap),
                  ],
                ),
              ),
            ),
          ],
        ),
      ],
    );
  }
}

class _ActivityRuleCard extends StatelessWidget {
  const _ActivityRuleCard({required this.snap, required this.style});

  final WeatherSnapshot snap;
  final String style;

  @override
  Widget build(BuildContext context) {
    final rule = evaluateActivity(style, snap);
    final text = Theme.of(context).textTheme;

    return SpottoCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: rule.tone.withValues(alpha: 0.12),
                  shape: BoxShape.circle,
                ),
                child: Icon(spottoIcon(rule.icon), size: 20, color: rule.tone),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(rule.title, style: text.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                    Text(rule.scoreText, style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          Text(
            rule.tip,
            style: text.bodyMedium?.copyWith(color: SpottoColors.inkSoft),
          ),
        ],
      ),
    );
  }
}

class _HourlyForecastCard extends StatelessWidget {
  const _HourlyForecastCard({required this.hourly});

  final List<HourlyPoint> hourly;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;

    return SpottoCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('24-Hour Forecast', style: text.titleSmall?.copyWith(fontWeight: FontWeight.bold)),
          const SizedBox(height: 16),
          SizedBox(
            height: 90,
            child: ListView.separated(
              scrollDirection: Axis.horizontal,
              itemCount: hourly.length,
              separatorBuilder: (_, __) => const SizedBox(width: 16),
              itemBuilder: (context, i) {
                final item = hourly[i];
                final timeStr = DateFormat('h a').format(item.Time);
                return Column(
                  children: [
                    Text(timeStr, style: text.labelSmall?.copyWith(color: SpottoColors.muted)),
                    const SizedBox(height: 6),
                    Icon(spottoIcon(item.Icon), size: 20, color: SpottoColors.accent),
                    const SizedBox(height: 6),
                    Text('${item.Temp.round()}°', style: text.titleSmall?.copyWith(fontWeight: FontWeight.bold)),
                  ],
                );
              },
            ),
          ),
        ],
      ),
    );
  }
}

class _StatsGrid extends StatelessWidget {
  const _StatsGrid({required this.snap});

  final WeatherSnapshot snap;

  @override
  Widget build(BuildContext context) {
    final n = snap.now;
    final items = [
      (label: 'FEELS LIKE', val: '${n.FeelsLike.round()}°', icon: Icons.thermostat),
      (label: 'HUMIDITY', val: '${n.Humidity}%', icon: Icons.water_drop_outlined),
      (label: 'WIND', val: '${n.WindSpeedKmh.round()} km/h', icon: Icons.air),
      (label: 'UV INDEX', val: n.UVIndex.toStringAsFixed(1), icon: Icons.wb_sunny_outlined),
      (label: 'PRECIPITATION', val: '${n.PrecipProbability}%', icon: Icons.umbrella_outlined),
      (label: 'PRESSURE', val: '${n.PressureHpa.round()} hPa', icon: Icons.compress),
    ];

    return GridView.builder(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: 2,
        mainAxisSpacing: 12,
        crossAxisSpacing: 12,
        childAspectRatio: 2.2,
      ),
      itemCount: items.length,
      itemBuilder: (context, i) {
        final it = items[i];
        return SpottoCard(
          padding: const EdgeInsets.all(12),
          child: Row(
            children: [
              Icon(it.icon, size: 20, color: SpottoColors.accent),
              const SizedBox(width: 10),
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Text(it.label, style: const TextStyle(fontSize: 9.5, fontWeight: FontWeight.bold, color: SpottoColors.muted)),
                  const SizedBox(height: 2),
                  Text(it.val, style: const TextStyle(fontSize: 15, fontWeight: FontWeight.bold, color: SpottoColors.ink)),
                ],
              ),
            ],
          ),
        );
      },
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
      children: List.generate(
        count,
        (i) => AnimatedContainer(
          duration: const Duration(milliseconds: 200),
          margin: const EdgeInsets.symmetric(horizontal: 4),
          width: i == index ? 20 : 8,
          height: 8,
          decoration: BoxDecoration(
            color: i == index ? SpottoColors.accent : SpottoColors.muted.withValues(alpha: 0.3),
            borderRadius: BorderRadius.circular(4),
          ),
        ),
      ),
    );
  }
}

class _WeatherErrorState extends StatelessWidget {
  const _WeatherErrorState({
    required this.locationName,
    required this.error,
    required this.onRetry,
  });

  final String locationName;
  final Object error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;

    return Scaffold(
      backgroundColor: SpottoColors.sand,
      appBar: AppBar(
        backgroundColor: Colors.transparent,
        elevation: 0,
        title: Text(locationName),
      ),
      body: Center(child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const Icon(Icons.cloud_off, size: 48, color: SpottoColors.muted),
            const SizedBox(height: 16),
            Text('Weather data unavailable', style: text.titleMedium),
            const SizedBox(height: 8),
            Text('Check your connection and try again.', style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: const Text('Try again')),
          ],
        ),
      )),
    );
  }
}

/// Search sheet for pinning a new weather location — searches curated Sri Lanka
/// locations, local places regions, and live OpenStreetMap Nominatim for ANY
/// town, beach, or spot in Sri Lanka. Also includes a prominent "Use Current Location (GPS)"
/// button at the top.
class _LocationSearchSheet extends ConsumerStatefulWidget {
  const _LocationSearchSheet({required this.places, required this.excludeIds});

  final List<Place> places;
  final Set<String> excludeIds;

  @override
  ConsumerState<_LocationSearchSheet> createState() => _LocationSearchSheetState();
}

class _LocationSearchSheetState extends ConsumerState<_LocationSearchSheet> {
  final _controller = TextEditingController();
  String _query = '';
  List<PinnedLocation> _osmResults = [];
  bool _isSearchingOSM = false;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  List<PinnedLocation> get _localLocations {
    final seen = <String>{};
    final out = <PinnedLocation>[];

    // 1. Curated Sri Lanka locations
    for (final loc in kSriLankaWeatherLocations) {
      if (seen.add(loc.id) && !widget.excludeIds.contains(loc.id)) {
        out.add(loc);
      }
    }

    // 2. Additional regions from loaded places
    for (final p in widget.places) {
      if (p.regionId.isNotEmpty && seen.add(p.regionId) && !widget.excludeIds.contains(p.regionId)) {
        out.add(PinnedLocation(id: p.regionId, name: p.region, lat: p.lat, lng: p.lng));
      }
    }
    out.sort((a, b) => a.name.compareTo(b.name));
    return out;
  }

  void _onQueryChanged(String v) {
    setState(() => _query = v);
    final q = v.trim();
    if (q.length >= 2) {
      _fetchOSM(q);
    } else {
      setState(() {
        _osmResults = [];
        _isSearchingOSM = false;
      });
    }
  }

  Future<void> _fetchOSM(String query) async {
    setState(() => _isSearchingOSM = true);
    try {
      final searchService = ref.read(placesSearchServiceProvider);
      final gResults = await searchService.search(query);
      if (!mounted) return;
      final mapped = gResults.map((r) => PinnedLocation(
        id: r.id,
        name: r.address.isNotEmpty ? '${r.name}, ${r.address}' : r.name,
        lat: r.lat,
        lng: r.lng,
      )).toList();
      setState(() {
        _osmResults = mapped;
        _isSearchingOSM = false;
      });
    } catch (_) {
      if (mounted) setState(() => _isSearchingOSM = false);
    }
  }

  Future<void> _useCurrentLocationGPS() async {
    final messenger = ScaffoldMessenger.of(context);
    final (pos, _) = await bestEffortPosition();
    if (!mounted) return;
    if (pos == null) {
      messenger.showSnackBar(
        const SnackBar(content: Text('Location is off or denied — enable GPS in settings.')),
      );
      return;
    }
    final gpsPin = PinnedLocation(
      id: 'gps_${DateTime.now().millisecondsSinceEpoch}',
      name: 'Current Location (GPS)',
      lat: double.parse(pos.latitude.toStringAsFixed(3)),
      lng: double.parse(pos.longitude.toStringAsFixed(3)),
    );
    Navigator.pop(context, gpsPin);
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final q = _query.trim().toLowerCase();
    final local = _localLocations;
    final filteredLocal = q.isEmpty
        ? local
        : local.where((r) => r.name.toLowerCase().contains(q)).toList();

    return SafeArea(
      child: Padding(
        padding: EdgeInsets.only(bottom: MediaQuery.viewInsetsOf(context).bottom),
        child: Container(
          constraints: BoxConstraints(maxHeight: MediaQuery.of(context).size.height * 0.75),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
                child: TextField(
                  controller: _controller,
                  autofocus: true,
                  onChanged: _onQueryChanged,
                  decoration: InputDecoration(
                    hintText: 'Search any location in Sri Lanka…',
                    prefixIcon: const Icon(Icons.search, size: 18, color: SpottoColors.muted),
                    suffixIcon: _isSearchingOSM
                        ? const Padding(
                            padding: EdgeInsets.all(12),
                            child: SizedBox(
                              width: 14,
                              height: 14,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            ),
                          )
                        : null,
                  ),
                ),
              ),

              // Use My Current Location GPS Option
              ListTile(
                leading: Container(
                  padding: const EdgeInsets.all(6),
                  decoration: const BoxDecoration(
                    color: SpottoColors.accentTint,
                    shape: BoxShape.circle,
                  ),
                  child: const Icon(Icons.my_location_rounded, size: 18, color: SpottoColors.accent),
                ),
                title: const Text('Use My Current Location (GPS)',
                    style: TextStyle(fontWeight: FontWeight.bold, color: SpottoColors.accent)),
                subtitle: const Text('Detect exact coordinates from device GPS',
                    style: TextStyle(fontSize: 11, color: SpottoColors.muted)),
                onTap: _useCurrentLocationGPS,
              ),
              const Divider(height: 1),

              Flexible(
                child: (filteredLocal.isEmpty && _osmResults.isEmpty && !_isSearchingOSM)
                    ? Padding(
                        padding: const EdgeInsets.all(24),
                        child: Text(
                          'No Sri Lanka locations matching "$_query"',
                          style: text.bodyMedium?.copyWith(color: SpottoColors.muted),
                        ),
                      )
                    : ListView(
                        shrinkWrap: true,
                        children: [
                          if (filteredLocal.isNotEmpty) ...[
                            const Padding(
                              padding: EdgeInsets.fromLTRB(16, 10, 16, 4),
                              child: Text('SRI LANKA REGIONS & CITIES',
                                  style: TextStyle(
                                      fontSize: 10,
                                      fontWeight: FontWeight.bold,
                                      color: SpottoColors.muted,
                                      letterSpacing: 0.8)),
                            ),
                            for (final r in filteredLocal)
                              ListTile(
                                leading: const Icon(Icons.place_outlined, color: SpottoColors.accent),
                                title: Text(r.name),
                                onTap: () => Navigator.pop(context, r),
                              ),
                          ],
                          if (_osmResults.isNotEmpty) ...[
                            const Padding(
                              padding: EdgeInsets.fromLTRB(16, 10, 16, 4),
                              child: Text('MAP LOCATIONS (SEARCH RESULTS)',
                                  style: TextStyle(
                                      fontSize: 10,
                                      fontWeight: FontWeight.bold,
                                      color: SpottoColors.accent,
                                      letterSpacing: 0.8)),
                            ),
                            for (final r in _osmResults)
                              ListTile(
                                leading: const Icon(Icons.location_on_outlined, color: SpottoColors.sun),
                                title: Text(r.name, maxLines: 1, overflow: TextOverflow.ellipsis),
                                onTap: () => Navigator.pop(context, r),
                              ),
                          ],
                        ],
                      ),
              ),
              const SizedBox(height: 8),
            ],
          ),
        ),
      ),
    );
  }
}
