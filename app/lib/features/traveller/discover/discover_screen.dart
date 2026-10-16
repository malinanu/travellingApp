// Discover tab: interactive map with spot popup banners, user-interest & trip
// calendar prioritization, Google Places search, and auto-centering on launch.

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart' show rootBundle;
import 'package:flutter_compass/flutter_compass.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';
import 'package:go_router/go_router.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';

import '../../../core/api/google_places_service.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/category_icons.dart';
import '../../../core/widgets/map_marker.dart';
import '../../../core/widgets/marker_bitmap.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/mock/places.dart' as data;
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';
import '../../../data/vendor_categories.dart';
import '../group/group_sheet.dart';
import '../group/group_ws_provider.dart';
import 'categories_sheet.dart';
import 'quest_sheet.dart';
import '../place_detail/business_drawer.dart';
import '../weather/activity_rules.dart';

enum _LocationMode { idle, following, heading }

/// Default initial fallback map center (Weligama Bay) used before device GPS resolves.
const _kDefaultMapCenter = LatLng(kFallbackLat, kFallbackLng);

const _kPinSlotSize = Size(58, 58);
const _kBannerSlotSize = Size(175, 62);
const _kAvatarSlotSize = Size(48, 48);
const _kLocationDotSlotSize = Size(34, 34);

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
  GoogleMapController? _mapController;
  String? _mapStyle;

  LatLngAt? _resolvedPosition;
  bool _centeredOnDevice = false;

  @override
  void initState() {
    super.initState();
    rootBundle.loadString('assets/map/map_style.json').then((style) {
      if (mounted) setState(() => _mapStyle = style);
    });
    // Auto-center on user's current location on app launch as soon as GPS resolves
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _autoCenterOnUserLocation();
    });
  }

  Future<void> _autoCenterOnUserLocation() async {
    final (pos, _) = await bestEffortPosition();
    if (!mounted || pos == null) return;
    final userPos = LatLng(pos.latitude, pos.longitude);
    setState(() {
      _lastKnownLatLng = userPos;
      _centeredOnDevice = true;
      _locationMode = _LocationMode.following;
    });
    _animateCamera(CameraUpdate.newLatLngZoom(userPos, 16.0));
    _startPositionStream();
  }

  void _maybeCenterOnDevice() {
    if (_resolvedPosition == null) return;
    final userPos = LatLng(_resolvedPosition!.lat, _resolvedPosition!.lng);
    if (!_centeredOnDevice) {
      _centeredOnDevice = true;
      _animateCamera(CameraUpdate.newLatLngZoom(userPos, 16.0));
    }
  }

  _LocationMode _locationMode = _LocationMode.idle;
  StreamSubscription<Position>? _positionSub;
  StreamSubscription<CompassEvent>? _compassSub;
  LatLng? _lastKnownLatLng;
  double _currentBearing = 0;
  DateTime _lastHeadingApply = DateTime.fromMillisecondsSinceEpoch(0);

  // google_maps_flutter's onCameraMoveStarted fires for programmatic camera
  // moves too (unlike flutter_map's hasGesture flag), so this flag lets us
  // tell "we just called animateCamera" apart from an actual user drag —
  // without it, every auto-follow/heading update would immediately cancel
  // itself via _onCameraMoveStarted.
  bool _isProgrammaticCamera = false;

  void _animateCamera(CameraUpdate update) {
    _isProgrammaticCamera = true;
    _mapController?.animateCamera(update);
  }

  void _updateCamera() {
    if (_lastKnownLatLng == null) return;
    final bearing = _locationMode == _LocationMode.heading ? _currentBearing : 0.0;
    _animateCamera(
      CameraUpdate.newCameraPosition(CameraPosition(target: _lastKnownLatLng!, zoom: 16, bearing: bearing)),
    );
  }

  void _startPositionStream() {
    _positionSub?.cancel();
    _positionSub =
        Geolocator.getPositionStream(
          locationSettings: const LocationSettings(accuracy: LocationAccuracy.high, distanceFilter: 5),
        ).listen((pos) {
          _lastKnownLatLng = LatLng(pos.latitude, pos.longitude);
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
            const SnackBar(content: Text('Location is off — enable it in Settings to center the map on you.')),
          );
        }
        return;
      }
      ref.invalidate(locationPermissionProvider);
      _lastKnownLatLng = LatLng(pos.latitude, pos.longitude);
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
  Set<String> _selectedCategories = {'All'};
  double _currentZoom = 15.0;
  String? _selectedPlaceId;

  List<Place> _filterPlaces(List<Place> places, Set<String> favoriteIds) {
    if (_selectedCategories.contains('All')) return places;

    return places.where((p) {
      if (_selectedCategories.contains('Deals') && p.deal.active) return true;
      if (_selectedCategories.contains('Favorites') && favoriteIds.contains(p.id)) return true;

      final cat = categoryById(p.categoryId);
      if (cat != null && _selectedCategories.contains(cat.label)) return true;

      return false;
    }).toList();
  }

  /// Map-pin vibe badge icon for [p] — the category's primary relevant
  /// dimension (see relevantVibeDimensions), or null if there's no real
  /// vibe sample yet (an unrated place shows no badge, never a guess).
  IconData? _vibeBadgeIcon(Place p) {
    if (p.vibe.sampleSize <= 0) return null;
    final dims = relevantVibeDimensions(p.categoryId);
    if (dims.isEmpty) return null;
    for (final v in data.vibes) {
      if (v.id == dims.first) return spottoIcon(v.icon);
    }
    return null;
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
    _mapController?.dispose();
    super.dispose();
  }

  List<Place> _searchResults(List<Place> localPlaces) {
    final q = _searchQuery.trim().toLowerCase();
    if (q.isEmpty) return const [];
    return localPlaces
        .where(
          (p) =>
              p.name.toLowerCase().contains(q) ||
              p.type.toLowerCase().contains(q) ||
              p.region.toLowerCase().contains(q),
        )
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
      _animateCamera(CameraUpdate.newLatLngZoom(LatLng(p.lat, p.lng), 16.5));
    }
    showBusinessDrawer(context, p);
  }

  /// Resolves a cached marker bitmap for [key], kicking off an off-screen
  /// render (via [MarkerBitmapCache]) if it isn't cached yet. Returns a
  /// same-hue placeholder for the first frame or two while the real bitmap
  /// renders, so markers never wait on a blank/missing icon.
  BitmapDescriptor _resolveBitmap({
    required String key,
    required Widget Function() build,
    required Size slotSize,
    required double placeholderHue,
  }) {
    final cached = MarkerBitmapCache.instance.get(key);
    if (cached != null) return cached;
    MarkerBitmapCache.instance.render(
      context: context,
      key: key,
      widget: Center(child: build()),
      size: slotSize,
      onReady: () {
        if (mounted) setState(() {});
      },
    );
    return BitmapDescriptor.defaultMarkerWithHue(placeholderHue);
  }

  Set<Marker> _buildMapMarkers({
    required List<Place> places,
    required String time,
    required bool groupMode,
    required List<Traveler> members,
    required Set<String> favoriteIds,
    required Set<String> tripCalendarPlaceNames,
    required List<String> userInterests,
    required Map<String, LatLng> liveLocations,
  }) {
    final markers = <Marker>{};

    final filteredPlaces = _filterPlaces(places, favoriteIds);

    // 1. Places markers as Small Banner Callouts or Compact Pins based on zoom & selection
    for (final p in filteredPlaces) {
      if (p.lat == 0 && p.lng == 0) continue;
      final emphasized = p.bestTimes.contains(time);
      final isFavorite = favoriteIds.contains(p.id);
      final inTripCalendar =
          tripCalendarPlaceNames.contains(p.id) || tripCalendarPlaceNames.contains(p.name.toLowerCase());
      final matchesInterest = userInterests.any((style) => p.styles.contains(style));

      final color = SpottoColors.pinTones[p.color] ?? SpottoColors.accent;
      var icon = categoryIcon(p.categoryId);
      if (icon == Icons.place && categoryById(p.type) != null) {
        icon = categoryIcon(p.type);
      }
      if (icon == Icons.place && p.icon.isNotEmpty) {
        icon = spottoIcon(p.icon);
      }

      final isSelected = _selectedPlaceId == p.id;
      final showBanner = isSelected || (_currentZoom >= 14.5 && (p.deal.active || inTripCalendar || isFavorite));
      final pinEmphasized = emphasized || isFavorite || matchesInterest;

      final BitmapDescriptor icon_;
      if (showBanner) {
        final key =
            'banner|${p.id}|${color.toARGB32()}|${icon.codePoint}|${p.name}|'
            '${p.deal.active}|$isFavorite|$inTripCalendar|$matchesInterest|$emphasized';
        icon_ = _resolveBitmap(
          key: key,
          slotSize: _kBannerSlotSize,
          placeholderHue: BitmapDescriptor.hueAzure,
          build: () => SpotBannerMarker(
            name: p.name,
            color: color,
            icon: icon,
            hasDeal: p.deal.active,
            isFavorite: isFavorite,
            inTripCalendar: inTripCalendar,
            matchesInterest: matchesInterest,
            emphasized: emphasized,
          ),
        );
      } else {
        final vibeIcon = _vibeBadgeIcon(p);
        final key =
            'pin|${color.toARGB32()}|${icon.codePoint}|${p.deal.active}|'
            '$pinEmphasized|${vibeIcon?.codePoint}';
        icon_ = _resolveBitmap(
          key: key,
          slotSize: _kPinSlotSize,
          placeholderHue: BitmapDescriptor.hueAzure,
          build: () => PlacePinMarker(
            color: color,
            icon: icon,
            hasDeal: p.deal.active,
            emphasized: pinEmphasized,
            vibeIcon: vibeIcon,
          ),
        );
      }

      markers.add(
        Marker(
          markerId: MarkerId('place_${p.id}'),
          position: LatLng(p.lat, p.lng),
          icon: icon_,
          anchor: const Offset(0.5, 0.5),
          onTap: () {
            setState(() => _selectedPlaceId = p.id);
            showBusinessDrawer(context, p);
          },
        ),
      );
    }

    // 2. Group members markers
    if (groupMode) {
      for (final m in members) {
        final livePos = liveLocations[m.id];
        final lat = livePos?.latitude ?? m.lat;
        final lng = livePos?.longitude ?? m.lng;
        if (lat == 0 && lng == 0) continue;

        final key = 'avatar|${m.initial}|${m.color}';
        final icon = _resolveBitmap(
          key: key,
          slotSize: _kAvatarSlotSize,
          placeholderHue: BitmapDescriptor.hueViolet,
          build: () => AvatarPinMarker(initial: m.initial, color: Color(m.color)),
        );
        markers.add(
          Marker(
            markerId: MarkerId('member_${m.id}'),
            position: LatLng(lat, lng),
            icon: icon,
            anchor: const Offset(0.5, 0.5),
            onTap: () {
              ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('${m.name} · ${m.sub}')));
            },
          ),
        );
      }
    }

    // 3. User location dot marker
    if (_lastKnownLatLng != null) {
      const key = 'location_dot';
      final icon = _resolveBitmap(
        key: key,
        slotSize: _kLocationDotSlotSize,
        placeholderHue: BitmapDescriptor.hueBlue,
        build: () => const CurrentLocationDot(),
      );
      markers.add(
        Marker(
          markerId: const MarkerId('current_location'),
          position: _lastKnownLatLng!,
          icon: icon,
          anchor: const Offset(0.5, 0.5),
          zIndexInt: 1,
        ),
      );
    }

    return markers;
  }

  @override
  Widget build(BuildContext context) {
    final time = ref.watch(timeFilterProvider);
    final weatherAsync = ref.watch(currentWeatherProvider);
    // Last saved snapshot while the live one loads (or if it fails).
    final weather = weatherAsync.value ?? ref.watch(cachedWeatherProvider);
    final prefs = ref.watch(prefsProvider);
    final category = resolveCategory(prefs);
    final text = Theme.of(context).textTheme;
    final groupMode = ref.watch(groupModeProvider);
    final group = ref.watch(groupProvider);
    final members = group.members.map(Traveler.fromGroupMember).toList();
    final places = ref.watch(placesProvider);
    final favorites = ref.watch(favoritesProvider);
    final trip = ref.watch(tripProvider);
    final liveLocations = ref.watch(groupLiveLocationsProvider);
    final categories = ref.watch(categoriesProvider);

    final favoriteIds = favorites;
    final Set<String> tripCalendarPlaceNames = trip.days
        .expand((d) => d.stops)
        .map((s) => s.placeId.isNotEmpty ? s.placeId : (s.custom?.name.toLowerCase() ?? ''))
        .where((id) => id.isNotEmpty)
        .toSet();
    final userInterests = prefs?.styles ?? const <String>[];

    ref.watch(nearbyTravellersProvider);
    ref.listen<AsyncValue<(LatLngAt, DeviceLocationResult)>>(devicePositionProvider, (_, next) {
      _resolvedPosition = next.value?.$1;
      if (_resolvedPosition != null && _lastKnownLatLng == null) {
        _lastKnownLatLng = LatLng(_resolvedPosition!.lat, _resolvedPosition!.lng);
      }
      if (_lastKnownLatLng != null) {
        ref.read(groupLiveLocationsProvider.notifier).sendLocation(_lastKnownLatLng!);
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
      liveLocations: liveLocations,
    );

    final localResults = _searchResults(places);
    final showSearchResults = localResults.isNotEmpty || _googleResults.isNotEmpty || _isSearchingGoogle;

    return Stack(
      fit: StackFit.expand,
      children: [
        // Interactive Google Maps engine
        GoogleMap(
          initialCameraPosition: const CameraPosition(target: _kDefaultMapCenter, zoom: 15),
          minMaxZoomPreference: const MinMaxZoomPreference(3, 19),
          style: _mapStyle,
          markers: markers,
          myLocationEnabled: false,
          myLocationButtonEnabled: false,
          zoomControlsEnabled: false,
          mapToolbarEnabled: false,
          compassEnabled: false,
          onMapCreated: (controller) => _mapController = controller,
          onCameraMoveStarted: () {
            if (_isProgrammaticCamera) return;
            _onCameraMoveStarted();
          },
          onCameraMove: (position) {
            if ((position.zoom - _currentZoom).abs() > 0.3) {
              setState(() => _currentZoom = position.zoom);
            }
          },
          onCameraIdle: () => _isProgrammaticCamera = false,
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
                      boxShadow: const [BoxShadow(color: SpottoColors.inkShadow, blurRadius: 12, offset: Offset(0, 4))],
                    ),
                    child: Row(
                      children: [
                        Icon(spottoIcon(weather?.now.icon ?? 'Sun'), color: SpottoColors.sun),
                        const SizedBox(width: 10),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                weather != null
                                    ? '${weather.now.conditionLabel} · ${weather.now.temp.round()}°'
                                    : weatherAsync.hasError
                                    ? 'Weather unavailable'
                                    : 'Getting conditions…',
                                style: text.labelLarge,
                              ),
                              Text(
                                weather != null
                                    ? headline(category, rateHours(category, weather.hourly))
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
                    boxShadow: const [BoxShadow(color: SpottoColors.inkShadow, blurRadius: 12, offset: Offset(0, 4))],
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
                // Category Filter Bar (Google Maps style pills)
                SingleChildScrollView(
                  scrollDirection: Axis.horizontal,
                  child: Row(
                    children: [
                      _categoryChip('All', Icons.apps_rounded),
                      _categoryChip('Deals', Icons.local_offer_rounded),
                      for (final cat in categories.take(9)) _categoryChip(cat.label, spottoIcon(cat.icon)),
                      _categoryChip('Favorites', Icons.favorite_rounded),
                      // "More" button
                      Padding(
                        padding: const EdgeInsets.only(right: 8),
                        child: PressScale(
                          scale: 0.95,
                          onTap: () async {
                            final newSelection = await showCategoriesSheet(
                              context,
                              selectedCategories: _selectedCategories,
                              availableCategories: categories,
                            );
                            if (newSelection != null && mounted) {
                              setState(() => _selectedCategories = newSelection);
                            }
                          },
                          child: Container(
                            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
                            decoration: BoxDecoration(
                              color: Colors.white,
                              borderRadius: BorderRadius.circular(999),
                              border: Border.all(color: SpottoColors.line, width: 1.2),
                              boxShadow: const [
                                BoxShadow(color: Color(0x18103D3A), blurRadius: 6, offset: Offset(0, 2)),
                              ],
                            ),
                            child: const Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                Icon(Icons.more_horiz_rounded, size: 16, color: SpottoColors.ink),
                                SizedBox(width: 6),
                                Text(
                                  'More',
                                  style: TextStyle(fontSize: 13, fontWeight: FontWeight.w600, color: SpottoColors.ink),
                                ),
                              ],
                            ),
                          ),
                        ),
                      ),
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
                      boxShadow: const [BoxShadow(color: SpottoColors.inkShadow, blurRadius: 12, offset: Offset(0, 4))],
                    ),
                    child: ListView(
                      shrinkWrap: true,
                      padding: EdgeInsets.zero,
                      children: [
                        if (localResults.isNotEmpty) ...[
                          const Padding(
                            padding: EdgeInsets.fromLTRB(16, 10, 16, 4),
                            child: Text(
                              'SPOTTO SPOTS',
                              style: TextStyle(
                                fontSize: 10,
                                fontWeight: FontWeight.bold,
                                color: SpottoColors.muted,
                                letterSpacing: 0.8,
                              ),
                            ),
                          ),
                          for (final p in localResults)
                            ListTile(
                              dense: true,
                              leading: Icon(
                                spottoIcon(p.icon),
                                color: SpottoColors.pinTones[p.color] ?? SpottoColors.accent,
                              ),
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
                            child: Text(
                              'GOOGLE MAPS PLACES',
                              style: TextStyle(
                                fontSize: 10,
                                fontWeight: FontWeight.bold,
                                color: SpottoColors.accent,
                                letterSpacing: 0.8,
                              ),
                            ),
                          ),
                          for (final gres in _googleResults)
                            ListTile(
                              dense: true,
                              leading: const Icon(Icons.place_rounded, color: SpottoColors.accent),
                              title: Text(gres.name, maxLines: 1, overflow: TextOverflow.ellipsis),
                              subtitle: Text(gres.address, maxLines: 1, overflow: TextOverflow.ellipsis),
                              trailing: Container(
                                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                                decoration: BoxDecoration(
                                  color: SpottoColors.accent.withValues(alpha: 0.1),
                                  borderRadius: BorderRadius.circular(6),
                                ),
                                child: const Text(
                                  'Google',
                                  style: TextStyle(
                                    fontSize: 10,
                                    color: SpottoColors.accent,
                                    fontWeight: FontWeight.bold,
                                  ),
                                ),
                              ),
                              onTap: () => _selectSearchResult(gres.toPlace()),
                            ),
                        ],
                        if (_isSearchingGoogle)
                          const Padding(
                            padding: EdgeInsets.all(12),
                            child: Center(
                              child: SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2)),
                            ),
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
                            BoxShadow(color: SpottoColors.inkShadow, blurRadius: 10, offset: Offset(0, 3)),
                          ],
                        ),
                        child: Row(
                          children: [
                            Icon(Icons.group_outlined, size: 17, color: groupMode ? Colors.white : SpottoColors.accent),
                            const SizedBox(width: 6),
                            Text(
                              groupMode ? '${group.name} · ${members.length + 1}' : 'Group',
                              style: TextStyle(
                                fontSize: 12.5,
                                fontWeight: FontWeight.w800,
                                color: groupMode ? Colors.white : SpottoColors.ink,
                              ),
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
                              BoxShadow(color: SpottoColors.inkShadow, blurRadius: 10, offset: Offset(0, 3)),
                            ],
                          ),
                          child: const Icon(Icons.person_add_outlined, size: 17, color: SpottoColors.accent),
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
                            BoxShadow(color: SpottoColors.inkShadow, blurRadius: 10, offset: Offset(0, 3)),
                          ],
                        ),
                        child: Text(
                          _myInitial(ref),
                          style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w800, color: Colors.white),
                        ),
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
              const _QuestChip(),
              const SizedBox(height: 8),
              Container(
                padding: const EdgeInsets.all(6),
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(999),
                  boxShadow: const [BoxShadow(color: SpottoColors.inkShadow, blurRadius: 12, offset: Offset(0, 4))],
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
                                Icon(
                                  spottoIcon(t.icon),
                                  size: 18,
                                  color: time == t.id ? Colors.white : SpottoColors.muted,
                                ),
                                Text(
                                  t.label,
                                  style: text.labelSmall?.copyWith(
                                    color: time == t.id ? Colors.white : SpottoColors.muted,
                                  ),
                                ),
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
      colors: [Colors.white.withValues(alpha: 0.16), Colors.transparent],
    ),
  };

  Widget _categoryChip(String label, IconData icon) {
    final isSelected = _selectedCategories.contains(label);
    return Padding(
      padding: const EdgeInsets.only(right: 8),
      child: PressScale(
        scale: 0.95,
        onTap: () {
          setState(() {
            if (label == 'All') {
              _selectedCategories.clear();
              _selectedCategories.add('All');
              return;
            }

            _selectedCategories.remove('All');

            if (_selectedCategories.contains(label)) {
              _selectedCategories.remove(label);
              if (_selectedCategories.isEmpty) {
                _selectedCategories.add('All');
              }
            } else {
              _selectedCategories.add(label);
            }
          });
        },
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
          decoration: BoxDecoration(
            color: isSelected ? SpottoColors.accent : Colors.white,
            borderRadius: BorderRadius.circular(999), // Google Maps style pill
            border: Border.all(color: isSelected ? SpottoColors.accent : SpottoColors.line, width: 1.2),
            boxShadow: const [BoxShadow(color: Color(0x18103D3A), blurRadius: 6, offset: Offset(0, 2))],
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                icon,
                size: 16, // Slightly larger icon to match Maps style
                color: isSelected ? Colors.white : SpottoColors.accent,
              ),
              const SizedBox(width: 6),
              Text(
                label,
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: isSelected ? FontWeight.w700 : FontWeight.w600,
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
          boxShadow: const [BoxShadow(color: SpottoColors.inkShadow, blurRadius: 12, offset: Offset(0, 4))],
        ),
        child: Icon(icon, size: 22, color: active ? Colors.white : SpottoColors.ink),
      ),
    );
  }
}

class _QuestChip extends ConsumerWidget {
  const _QuestChip();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final questAsync = ref.watch(dailyQuestProvider);
    final quest = questAsync.value ?? ref.watch(cachedQuestProvider) ?? kDefaultQuest;

    final done = quest.progress.clamp(0, quest.goal);
    final complete = quest.isComplete;
    final goal = quest.goal;

    return PressScale(
      scale: 0.98,
      onTap: () => showQuestSheet(context),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        decoration: BoxDecoration(
          color: complete ? SpottoColors.ink : Colors.white,
          borderRadius: BorderRadius.circular(18),
          boxShadow: const [BoxShadow(color: SpottoColors.inkShadow, blurRadius: 12, offset: Offset(0, 4))],
        ),
        child: Row(
          children: [
            Icon(complete ? Icons.emoji_events : Icons.explore, size: 18, color: SpottoColors.sun),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    complete ? '${quest.rewardDescription} unlocked!' : 'Daily quest · visit $goal spots',
                    style: TextStyle(
                      fontSize: 12.5,
                      fontWeight: FontWeight.w800,
                      color: complete ? Colors.white : SpottoColors.ink,
                    ),
                  ),
                  const SizedBox(height: 4),
                  ClipRRect(
                    borderRadius: BorderRadius.circular(999),
                    child: Container(
                      height: 5,
                      color: complete ? Colors.white.withValues(alpha: 0.25) : SpottoColors.ink.withValues(alpha: 0.08),
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
            Text(
              '$done/$goal',
              style: TextStyle(
                fontSize: 12.5,
                fontWeight: FontWeight.w800,
                color: complete ? SpottoColors.sun : SpottoColors.muted,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
