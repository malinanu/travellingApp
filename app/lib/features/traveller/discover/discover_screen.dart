// Discover tab: real Google Map of Weligama Bay with highlighted place
// markers, weather bar and time-of-day filter — port of MapView.jsx +
// SatelliteMap.jsx, upgraded from the illustrated SVG to google_maps_flutter.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/map_marker.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/mock/places.dart' as data;
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';
import '../group/group_sheet.dart';
import '../place_detail/business_drawer.dart';
import '../weather/activity_rules.dart';

/// Weligama Bay — the map's synchronous construction-time default (a
/// GoogleMap needs some initial target immediately, before any async
/// location lookup can complete); overridden by an animateCamera to the
/// user's real device position as soon as it resolves (see
/// _maybeCenterOnDevice), same fallback coordinates as
/// devicePositionProvider/kFallbackLat/kFallbackLng in repositories.dart.
const _weligamaBay = LatLng(kFallbackLat, kFallbackLng);

/// Real user's initial for the "you" avatar bubble, falling back to '?'
/// before /me has loaded.
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
  final _pinCache = PinBitmapCache();
  Set<Marker> _markers = {};
  String? _builtForTime;
  bool _builtForGroupMode = false;

  // The map builds immediately at the Weligama fallback (markers must never
  // wait on an async location lookup — bestEffortPosition() isn't cached
  // across calls, so a fresh lookup can take 10+ seconds or stall entirely,
  // and the map/markers used to stay visible throughout that even when
  // recentering hadn't happened yet). Once devicePositionProvider resolves,
  // glide the camera to the real position exactly once, so it doesn't fight
  // the user's own panning (the myLocationButton covers manual recenter
  // after that).
  GoogleMapController? _mapController;
  LatLngAt? _resolvedPosition;
  bool _centeredOnDevice = false;

  void _maybeCenterOnDevice() {
    if (_centeredOnDevice || _mapController == null || _resolvedPosition == null) return;
    _centeredOnDevice = true;
    _mapController!.animateCamera(
      CameraUpdate.newLatLngZoom(LatLng(_resolvedPosition!.lat, _resolvedPosition!.lng), 16),
    );
  }

  final _searchController = TextEditingController();
  String _searchQuery = '';

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  List<Place> _searchResults(List<Place> places) {
    final q = _searchQuery.trim().toLowerCase();
    if (q.isEmpty) return const [];
    return places
        .where((p) =>
            p.name.toLowerCase().contains(q) ||
            p.type.toLowerCase().contains(q) ||
            p.region.toLowerCase().contains(q))
        .take(6)
        .toList();
  }

  void _selectSearchResult(Place p) {
    FocusScope.of(context).unfocus();
    setState(() {
      _searchQuery = '';
      _searchController.clear();
    });
    if (p.lat != 0 || p.lng != 0) {
      _mapController?.animateCamera(CameraUpdate.newLatLngZoom(LatLng(p.lat, p.lng), 17));
    }
    showBusinessDrawer(context, p);
  }

  Future<void> _rebuildMarkers({
    required List<Place> places,
    required String time,
    required bool groupMode,
    required List<Traveler> members,
    required BuildContext context,
  }) async {
    final next = <Marker>{};
    for (final p in places) {
      if (p.lat == 0 && p.lng == 0) continue; // no real coordinates yet
      final emphasized = p.bestTimes.contains(time);
      final icon = await _pinCache.place(
        key: p.id,
        color: SpottoColors.pinTones[p.color] ?? SpottoColors.accent,
        icon: spottoIcon(p.icon),
        hasDeal: p.deal.active,
        emphasized: emphasized,
      );
      next.add(Marker(
        markerId: MarkerId('place_${p.id}'),
        position: LatLng(p.lat, p.lng),
        icon: icon,
        anchor: const Offset(0.5, 0.5),
        alpha: emphasized ? 1.0 : 0.45,
        zIndexInt: emphasized ? 1 : 0,
        consumeTapEvents: true,
        onTap: () => showBusinessDrawer(context, p),
      ));
    }

    if (groupMode) {
      for (final m in members) {
        if (m.lat == 0 && m.lng == 0) continue;
        final icon = await _pinCache.avatar(
          key: 'traveler_${m.id}',
          initial: m.initial,
          color: Color(m.color),
        );
        next.add(Marker(
          markerId: MarkerId('traveler_${m.id}'),
          position: LatLng(m.lat, m.lng),
          icon: icon,
          anchor: const Offset(0.5, 0.5),
          infoWindow: InfoWindow(title: m.name, snippet: m.sub),
        ));
      }
    }

    if (!mounted) return;
    setState(() => _markers = next);
  }

  @override
  Widget build(BuildContext context) {
    final time = ref.watch(timeFilterProvider);
    final weatherAsync = ref.watch(currentWeatherProvider);
    final category = resolveCategory(ref.watch(prefsProvider));
    final text = Theme.of(context).textTheme;
    final groupMode = ref.watch(groupModeProvider);
    final group = ref.watch(groupProvider);
    final members = group.members.map(Traveler.fromGroupMember).toList();
    final places = ref.watch(placesProvider);
    // Kicks off periodic location reporting + nearby-travellers fetch while
    // this screen is open and the user has opted in (see PrivacyPage) — not
    // used for markers directly (no exact position for non-group members),
    // but keeps group_sheet's "invite nearby travellers" list fresh.
    ref.watch(nearbyTravellersProvider);
    ref.listen<AsyncValue<LatLngAt>>(devicePositionProvider, (_, next) {
      _resolvedPosition = next.valueOrNull;
      _maybeCenterOnDevice();
      // The lookup may have just prompted for (and secured) location
      // permission — re-check so myLocationEnabled below can flip on
      // reactively instead of staying stuck at whatever it read on first
      // build.
      if (next.hasValue) ref.invalidate(locationPermissionProvider);
    });
    final hasLocationPermission = ref.watch(locationPermissionProvider).valueOrNull ?? false;

    if (_builtForTime != time || _builtForGroupMode != groupMode) {
      _builtForTime = time;
      _builtForGroupMode = groupMode;
      _rebuildMarkers(
        places: places,
        time: time,
        groupMode: groupMode,
        members: members,
        context: context,
      );
    }

    return Stack(
      fit: StackFit.expand,
      children: [
        // base map — built immediately at the Weligama fallback so markers
        // are never gated on an async location lookup; animateCamera glides
        // to the real position once devicePositionProvider resolves (see
        // _maybeCenterOnDevice).
        GoogleMap(
          initialCameraPosition: const CameraPosition(target: _weligamaBay, zoom: 16),
          markers: _markers,
          myLocationEnabled: hasLocationPermission,
          myLocationButtonEnabled: hasLocationPermission,
          zoomControlsEnabled: false,
          mapToolbarEnabled: false,
          compassEnabled: false,
          onMapCreated: (controller) {
            _mapController = controller;
            _maybeCenterOnDevice();
          },
        ),

        // time-of-day lighting overlay (700ms like the prototype)
        IgnorePointer(
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 700),
            decoration: BoxDecoration(gradient: _lighting(time)),
          ),
        ),

        // top: weather bar
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
                    onChanged: (v) => setState(() => _searchQuery = v),
                    decoration: InputDecoration(
                      hintText: 'Search spots on the map…',
                      prefixIcon: const Icon(Icons.search, size: 18, color: SpottoColors.muted),
                      suffixIcon: _searchQuery.isNotEmpty
                          ? IconButton(
                              icon: const Icon(Icons.close, size: 18, color: SpottoColors.muted),
                              onPressed: () => setState(() {
                                _searchQuery = '';
                                _searchController.clear();
                              }),
                            )
                          : null,
                      border: InputBorder.none,
                      contentPadding: const EdgeInsets.symmetric(vertical: 12, horizontal: 4),
                    ),
                  ),
                ),
                if (_searchResults(places).isNotEmpty)
                  Container(
                    margin: const EdgeInsets.only(top: 6),
                    constraints: const BoxConstraints(maxHeight: 260),
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
                        for (final p in _searchResults(places))
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

