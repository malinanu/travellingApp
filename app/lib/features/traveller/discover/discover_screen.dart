// Discover tab: interactive map with spot popup banners, user-interest & trip
// calendar prioritization, Google Places search, and auto-centering on launch.

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_compass/flutter_compass.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';
import 'package:go_router/go_router.dart';
import 'package:latlong2/latlong.dart' as ll;

import '../../../core/api/google_places_service.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/category_icons.dart';
import '../../../core/widgets/map_marker.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/mock/places.dart' as data;
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';
import '../group/group_sheet.dart';
import '../place_detail/business_drawer.dart';
import '../weather/activity_rules.dart';

enum _LocationMode { idle, following, heading }

/// Default initial fallback map center (Weligama Bay) used before device GPS resolves.
const _kDefaultMapCenter = ll.LatLng(kFallbackLat, kFallbackLng);

String _myInitial(WidgetRef ref) {
  final name = ref.watch(meProvider)?['name'] as String?;
  return name != null && name.isNotEmpty ? name[0].toUpperCase() : '?';
}

class DiscoverScreen extends ConsumerStatefulWidget {
  const DiscoverScreen({super.key});

  @override
  ConsumerState<DiscoverScreen> createState() => _DiscoverScreenState();
}

class _DiscoverScreenState extends ConsumerState<DiscoverScreen> {
  final MapController _mapController = MapController();

  LatLngAt? _resolvedPosition;
  bool _centeredOnDevice = false;

  @override
  void initState() {
    super.initState();
    // Auto-center on user's current location on app launch as soon as GPS resolves
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _autoCenterOnUserLocation();
    });
  }

  Future<void> _autoCenterOnUserLocation() async {
    final (pos, _) = await bestEffortPosition();
    if (!mounted || pos == null) return;
    final userPos = ll.LatLng(pos.latitude, pos.longitude);
    setState(() {
      _lastKnownLatLng = userPos;
      _centeredOnDevice = true;
      _locationMode = _LocationMode.following;
    });
    _mapController.move(userPos, 16.0);
    _startPositionStream();
  }

  void _maybeCenterOnDevice() {
    if (_resolvedPosition == null) return;
    final userPos = ll.LatLng(_resolvedPosition!.lat, _resolvedPosition!.lng);
    if (!_centeredOnDevice) {
      _centeredOnDevice = true;
      _mapController.move(userPos, 16.0);
    }
  }

  _LocationMode _locationMode = _LocationMode.idle;
  StreamSubscription<Position>? _positionSub;
  StreamSubscription<CompassEvent>? _compassSub;
  ll.LatLng? _lastKnownLatLng;
  double _currentBearing = 0;
  DateTime _lastHeadingApply = DateTime.fromMillisecondsSinceEpoch(0);

  void _updateCamera() {
    if (_lastKnownLatLng == null) return;
    final bearing = _locationMode == _LocationMode.heading ? _currentBearing : 0.0;
    _mapController.moveAndRotate(_lastKnownLatLng!, 16, bearing);
  }

  void _startPositionStream() {
    _positionSub?.cancel();
    _positionSub = Geolocator.getPositionStream(
      locationSettings: const LocationSettings(
        accuracy: LocationAccuracy.high,
        distanceFilter: 5,
      ),
    ).listen((pos) {
      _lastKnownLatLng = ll.LatLng(pos.latitude, pos.longitude);
      if (_locationMode != _LocationMode.idle) _updateCamera();
    });
  }

  void _startCompassStream() {
    _compassSub?.cancel();
    _compassSub = FlutterCompass.events?.listen((event) {
      final heading = event.heading;
      if (heading == null) return;
      final now = DateTime.now();
      if (now.difference(_lastHeadingApply) < const Duration(milliseconds: 150) &&
          (heading - _currentBearing).abs() < 3) {
        return;
      }
      _lastHeadingApply = now;
      _currentBearing = heading;
      if (_locationMode == _LocationMode.heading) _updateCamera();
    });
  }

  Future<void> _onLocationButtonTap() async {
    if (_locationMode == _LocationMode.idle) {
      final (pos, result) = await bestEffortPosition();
      if (!mounted) return;
      if (pos == null) {
        if (result == DeviceLocationResult.deniedForever) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
              content: Text('Location is off — enable it in Settings to center the map on you.'),
            ),
          );
        }
        return;
      }
      ref.invalidate(locationPermissionProvider);
      _lastKnownLatLng = ll.LatLng(pos.latitude, pos.longitude);
      setState(() => _locationMode = _LocationMode.following);
      _updateCamera();
      _startPositionStream();
    } else if (_locationMode == _LocationMode.following) {
      setState(() => _locationMode = _LocationMode.heading);
      _updateCamera();
      _startCompassStream();
    } else {
      _compassSub?.cancel();
      _compassSub = null;
      setState(() => _locationMode = _LocationMode.following);
      _updateCamera();
    }
  }

  void _onCameraMoveStarted() {
    _positionSub?.cancel();
    _positionSub = null;
    _compassSub?.cancel();
    _compassSub = null;
    if (_locationMode != _LocationMode.idle && mounted) {
      setState(() => _locationMode = _LocationMode.idle);
    }
  }

  final _searchController = TextEditingController();
  String _searchQuery = '';
  List<GooglePlacesSearchResult> _googleResults = [];
  bool _isSearchingGoogle = false;
  String _selectedCategory = 'All';
  double _currentZoom = 15.0;
  String? _selectedPlaceId;

  List<Place> _filterPlaces(List<Place> places, Set<String> favoriteIds) {
    switch (_selectedCategory) {
      case 'Cafés':
        return places.where((p) =>
          p.type.toLowerCase().contains('caf') ||
          p.type.toLowerCase().contains('coffee') ||
          p.icon == 'Coffee' ||
          p.name.toLowerCase().contains('cafe')
        ).toList();
      case 'Food':
        return places.where((p) =>
          p.type.toLowerCase().contains('food') ||
          p.type.toLowerCase().contains('restaurant') ||
          p.icon == 'Utensils'
        ).toList();
      case 'Beaches':
        return places.where((p) =>
          p.type.toLowerCase().contains('beach') ||
          p.type.toLowerCase().contains('surf') ||
          p.icon == 'Waves' ||
          p.name.toLowerCase().contains('beach')
        ).toList();
      case 'Deals':
        return places.where((p) => p.deal.active).toList();
      case 'Favorites':
        return places.where((p) => favoriteIds.contains(p.id)).toList();
      default:
        return places;
    }
  }

  void _onSearchQueryChanged(String query) async {
    setState(() => _searchQuery = query);
    if (query.trim().length >= 2) {
      setState(() => _isSearchingGoogle = true);
      final placesService = ref.read(googlePlacesServiceProvider);
      final results = await placesService.search(
        query,
        lat: _lastKnownLatLng?.latitude ?? kFallbackLat,
        lng: _lastKnownLatLng?.longitude ?? kFallbackLng,
      );
      if (mounted && _searchQuery == query) {
        setState(() {
          _googleResults = results;
          _isSearchingGoogle = false;
        });
      }
    } else {
      setState(() => _googleResults = []);
    }
  }

  @override
  void dispose() {
    _searchController.dispose();
    _positionSub?.cancel();
    _compassSub?.cancel();
    _mapController.dispose();
    super.dispose();
  }

  List<Place> _searchResults(List<Place> localPlaces) {
    final q = _searchQuery.trim().toLowerCase();
    if (q.isEmpty) return const [];
    return localPlaces
        .where((p) =>
            p.name.toLowerCase().contains(q) ||
            p.type.toLowerCase().contains(q) ||
            p.region.toLowerCase().contains(q))
        .take(5)
        .toList();
  }

  void _selectSearchResult(Place p) {
    FocusScope.of(context).unfocus();
    setState(() {
      _searchQuery = '';
      _searchController.clear();
      _googleResults = [];
    });
    if (p.lat != 0 || p.lng != 0) {
      _mapController.move(ll.LatLng(p.lat, p.lng), 16.5);
    }
    showBusinessDrawer(context, p);
  }

  List<Marker> _buildMapMarkers({
    required List<Place> places,
    required String time,
    required bool groupMode,
    required List<Traveler> members,
    required Set<String> favoriteIds,
    required Set<String> tripCalendarPlaceNames,
    required List<String> userInterests,
  }) {
    final markers = <Marker>[];

    final filteredPlaces = _filterPlaces(places, favoriteIds);

    // 1. Places markers as Small Banner Callouts or Compact Pins based on zoom & selection
    for (final p in filteredPlaces) {
      if (p.lat == 0 && p.lng == 0) continue;
      final emphasized = p.bestTimes.contains(time);
      final isFavorite = favoriteIds.contains(p.id);
      final inTripCalendar = tripCalendarPlaceNames.contains(p.id) || tripCalendarPlaceNames.contains(p.name.toLowerCase());
      final matchesInterest = userInterests.any((style) => p.styles.contains(style));

      final color = SpottoColors.pinTones[p.color] ?? SpottoColors.accent;
      final icon = categoryIcon(p.type);

      final isSelected = _selectedPlaceId == p.id;
      final showBanner = isSelected || (_currentZoom >= 14.5 && (p.deal.active || inTripCalendar || isFavorite));

      markers.add(
        Marker(
          point: ll.LatLng(p.lat, p.lng),
          width: showBanner ? 155 : 38,
          height: showBanner ? 42 : 38,
          alignment: Alignment.center,
          child: showBanner
              ? SpotBannerMarker(
                  name: p.name,
                  color: color,
                  icon: icon,
                  hasDeal: p.deal.active,
                  isFavorite: isFavorite,
                  inTripCalendar: inTripCalendar,
                  matchesInterest: matchesInterest,
                  emphasized: emphasized,
                  onTap: () {
                    setState(() => _selectedPlaceId = p.id);
                    showBusinessDrawer(context, p);
                  },
                )
              : PlacePinMarker(
                  color: color,
                  icon: icon,
                  hasDeal: p.deal.active,
                  emphasized: emphasized || isFavorite || matchesInterest,
                  onTap: () {
                    setState(() => _selectedPlaceId = p.id);
                    showBusinessDrawer(context, p);
                  },
                ),
        ),
      );
    }

    // 2. Group members markers
    if (groupMode) {
      for (final m in members) {
        if (m.lat == 0 && m.lng == 0) continue;
        markers.add(
          Marker(
            point: ll.LatLng(m.lat, m.lng),
            width: 40,
            height: 40,
            alignment: Alignment.center,
            child: GestureDetector(
              onTap: () {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text('${m.name} · ${m.sub}')),
                );
              },
              child: AvatarPinMarker(
                initial: m.initial,
                color: Color(m.color),
              ),
            ),
          ),
        );
      }
    }

    // 3. User location dot marker
    if (_lastKnownLatLng != null) {
      markers.add(
        Marker(
          point: _lastKnownLatLng!,
          width: 28,
          height: 28,
          alignment: Alignment.center,
          child: const CurrentLocationDot(),
        ),
      );
    }

    return markers;
  }

  @override
  Widget build(BuildContext context) {
    final time = ref.watch(timeFilterProvider);
    final weatherAsync = ref.watch(currentWeatherProvider);
    final prefs = ref.watch(prefsProvider);
    final category = resolveCategory(prefs);
    final text = Theme.of(context).textTheme;
    final groupMode = ref.watch(groupModeProvider);
    final group = ref.watch(groupProvider);
    final members = group.members.map(Traveler.fromGroupMember).toList();
    final places = ref.watch(placesProvider);
    final favorites = ref.watch(favoritesProvider);
    final trip = ref.watch(tripProvider);

    final favoriteIds = favorites;
    final Set<String> tripCalendarPlaceNames = trip.days
        .expand((d) => d.stops)
        .map((s) => s.placeId.isNotEmpty ? s.placeId : (s.custom?.name.toLowerCase() ?? ''))
        .where((id) => id.isNotEmpty)
        .toSet();
    final userInterests = prefs?.styles ?? const <String>[];

    ref.watch(nearbyTravellersProvider);
    ref.listen<AsyncValue<LatLngAt>>(devicePositionProvider, (_, next) {
      _resolvedPosition = next.valueOrNull;
      if (_resolvedPosition != null && _lastKnownLatLng == null) {
        _lastKnownLatLng = ll.LatLng(_resolvedPosition!.lat, _resolvedPosition!.lng);
      }
      _maybeCenterOnDevice();
      if (next.hasValue) ref.invalidate(locationPermissionProvider);
    });

    final markers = _buildMapMarkers(
      places: places,
      time: time,
      groupMode: groupMode,
      members: members,
      favoriteIds: favoriteIds,
      tripCalendarPlaceNames: tripCalendarPlaceNames,
      userInterests: userInterests,
    );

    final localResults = _searchResults(places);
    final showSearchResults = localResults.isNotEmpty || _googleResults.isNotEmpty || _isSearchingGoogle;

    return Stack(
      fit: StackFit.expand,
      children: [
        // Interactive flutter_map tile engine
        FlutterMap(
          mapController: _mapController,
          options: MapOptions(
            initialCenter: _kDefaultMapCenter,
            initialZoom: 15,
            maxZoom: 19,
            minZoom: 3,
            onPositionChanged: (position, hasGesture) {
              if (hasGesture) _onCameraMoveStarted();
              if ((position.zoom - _currentZoom).abs() > 0.3) {
                setState(() => _currentZoom = position.zoom);
              }
            },
          ),
          children: [
            TileLayer(
              urlTemplate: 'https://a.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}@2x.png',
              userAgentPackageName: 'com.spotto.spotto',
              tileProvider: NetworkTileProvider(),
            ),
            MarkerLayer(markers: markers),
          ],
        ),

        // time-of-day lighting overlay
        IgnorePointer(
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 700),
            decoration: BoxDecoration(gradient: _lighting(time)),
          ),
        ),

        // top: weather bar & Google Places search bar
        SafeArea(
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
            child: Column(
              children: [
                PressScale(
                  scale: 0.98,
                  onTap: () => context.push('/t/weather'),
                  child: Container(
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
                        Icon(spottoIcon(weatherAsync.valueOrNull?.now.icon ?? 'Sun'), color: SpottoColors.sun),
                        const SizedBox(width: 10),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                weatherAsync.when(
                                  data: (s) => '${s.now.conditionLabel} · ${s.now.temp.round()}°',
                                  loading: () => 'Getting conditions…',
                                  error: (_, __) => 'Weather unavailable',
                                ),
                                style: text.labelLarge,
                              ),
                              Text(
                                weatherAsync.valueOrNull != null
                                    ? headline(category, rateHours(category, weatherAsync.value!.hourly))
                                    : 'Tap for the full forecast',
                                style: text.bodySmall,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                              ),
                            ],
                          ),
                        ),
                        const Icon(Icons.chevron_right, size: 18, color: SpottoColors.muted),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 8),
                Container(
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(16),
                    boxShadow: const [
                      BoxShadow(color: SpottoColors.inkShadow, blurRadius: 12, offset: Offset(0, 4)),
                    ],
                  ),
                  child: TextField(
                    controller: _searchController,
                    onChanged: _onSearchQueryChanged,
                    decoration: InputDecoration(
                      hintText: 'Search spots or Google Maps…',
                      prefixIcon: const Icon(Icons.search, size: 18, color: SpottoColors.muted),
                      suffixIcon: _searchQuery.isNotEmpty
                          ? IconButton(
                              icon: const Icon(Icons.close, size: 18, color: SpottoColors.muted),
                              onPressed: () => setState(() {
                                _searchQuery = '';
                                _searchController.clear();
                                _googleResults = [];
                              }),
                            )
                          : null,
                      border: InputBorder.none,
                      contentPadding: const EdgeInsets.symmetric(vertical: 12, horizontal: 4),
                    ),
                  ),
                ),
                const SizedBox(height: 8),
                // Category Filter Bar
                SingleChildScrollView(
                  scrollDirection: Axis.horizontal,
                  child: Row(
                    children: [
                      _categoryChip('All', Icons.apps_rounded),
                      _categoryChip('Cafés', Icons.coffee_rounded),
                      _categoryChip('Food', Icons.restaurant_rounded),
                      _categoryChip('Beaches', Icons.beach_access_rounded),
                      _categoryChip('Deals', Icons.local_offer_rounded),
                      _categoryChip('Favorites', Icons.favorite_rounded),
                    ],
                  ),
                ),

                // Live search results dropdown (Local Spots + Google Maps results)
                if (showSearchResults)
                  Container(
                    margin: const EdgeInsets.only(top: 6),
                    constraints: const BoxConstraints(maxHeight: 300),
                    decoration: BoxDecoration(
                      color: Colors.white,
                      borderRadius: BorderRadius.circular(16),
                      boxShadow: const [
                        BoxShadow(color: SpottoColors.inkShadow, blurRadius: 12, offset: Offset(0, 4)),
                      ],
                    ),
                    child: ListView(
                      shrinkWrap: true,
                      padding: EdgeInsets.zero,
                      children: [
                        if (localResults.isNotEmpty) ...[
                          const Padding(
                            padding: EdgeInsets.fromLTRB(16, 10, 16, 4),
                            child: Text('SPOTTO SPOTS', style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: SpottoColors.muted, letterSpacing: 0.8)),
                          ),
                          for (final p in localResults)
                            ListTile(
                              dense: true,
                              leading: Icon(spottoIcon(p.icon), color: SpottoColors.pinTones[p.color] ?? SpottoColors.accent),
                              title: Text(p.name, maxLines: 1, overflow: TextOverflow.ellipsis),
                              subtitle: Text(
                                [p.type, if (p.region.isNotEmpty) p.region].join(' · '),
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                              ),
                              onTap: () => _selectSearchResult(p),
                            ),
                        ],
                        if (_googleResults.isNotEmpty) ...[
                          const Padding(
                            padding: EdgeInsets.fromLTRB(16, 10, 16, 4),
                            child: Text('GOOGLE MAPS PLACES', style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: SpottoColors.accent, letterSpacing: 0.8)),
                          ),
                          for (final gres in _googleResults)
                            ListTile(
                              dense: true,
                              leading: const Icon(Icons.place_rounded, color: SpottoColors.accent),
                              title: Text(gres.name, maxLines: 1, overflow: TextOverflow.ellipsis),
                              subtitle: Text(gres.address, maxLines: 1, overflow: TextOverflow.ellipsis),
                              trailing: Container(
                                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                                decoration: BoxDecoration(color: SpottoColors.accent.withValues(alpha: 0.1), borderRadius: BorderRadius.circular(6)),
                                child: const Text('Google', style: TextStyle(fontSize: 10, color: SpottoColors.accent, fontWeight: FontWeight.bold)),
                              ),
                              onTap: () => _selectSearchResult(gres.toPlace()),
                            ),
                        ],
                        if (_isSearchingGoogle)
                          const Padding(
                            padding: EdgeInsets.all(12),
                            child: Center(child: SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2))),
                          ),
                      ],
                    ),
                  ),

                const SizedBox(height: 8),
                Row(
                  children: [
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
                        child: Text(_myInitial(ref),
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

        // Location / compass button
        Positioned(
          right: 16,
          bottom: 150,
          child: _LocationCompassButton(mode: _locationMode, onTap: _onLocationButtonTap),
        ),

        // bottom: daily quest chip + time filter pills
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

  Widget _categoryChip(String label, IconData icon) {
    final isSelected = _selectedCategory == label;
    return Padding(
      padding: const EdgeInsets.only(right: 6),
      child: PressScale(
        scale: 0.95,
        onTap: () => setState(() => _selectedCategory = label),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 11, vertical: 6),
          decoration: BoxDecoration(
            color: isSelected ? SpottoColors.accent : Colors.white,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(
              color: isSelected ? SpottoColors.accent : SpottoColors.line,
              width: 1.2,
            ),
            boxShadow: const [
              BoxShadow(color: Color(0x18103D3A), blurRadius: 6, offset: Offset(0, 2)),
            ],
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon, size: 13, color: isSelected ? Colors.white : SpottoColors.accent),
              const SizedBox(width: 5),
              Text(
                label,
                style: TextStyle(
                  fontSize: 11.5,
                  fontWeight: isSelected ? FontWeight.w800 : FontWeight.w600,
                  color: isSelected ? Colors.white : SpottoColors.ink,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _LocationCompassButton extends StatelessWidget {
  const _LocationCompassButton({required this.mode, required this.onTap});

  final _LocationMode mode;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final active = mode != _LocationMode.idle;
    final icon = switch (mode) {
      _LocationMode.idle => Icons.near_me_outlined,
      _LocationMode.following => Icons.my_location,
      _LocationMode.heading => Icons.explore,
    };
    return PressScale(
      scale: 0.9,
      onTap: onTap,
      child: Container(
        width: 46,
        height: 46,
        alignment: Alignment.center,
        decoration: BoxDecoration(
          color: active ? SpottoColors.accent : Colors.white,
          shape: BoxShape.circle,
          boxShadow: const [
            BoxShadow(color: SpottoColors.inkShadow, blurRadius: 12, offset: Offset(0, 4)),
          ],
        ),
        child: Icon(icon, size: 22, color: active ? Colors.white : SpottoColors.ink),
      ),
    );
  }
}

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
