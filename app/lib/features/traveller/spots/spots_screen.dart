// Spots tab: scrollable business cards — port of SpotsList.jsx.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';

import '../../../core/api/api_client.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/category_icons.dart';
import '../../../core/widgets/illustration.dart';
import '../../../core/widgets/progressive_image.dart';
import '../../../core/widgets/spotto_card.dart';
import '../../../core/widgets/stagger_in.dart';
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';
import '../place_detail/business_drawer.dart';

// Nearest-section tuning — see _sectionise below for the full rationale.
const _kNearRadiusKm = 10.0;
const _kMaxNear = 8;
const _kMinNear = 3;
const _kMinTail = 3;

// Category filter chips, bucketed from the real Place.type values in the
// imported dataset (landmark/food/temple/wildlife/park/cafe/attraction —
// see server/sri_lanka_spots.json). id, label, matching types.
const _categoryFilters = [
  ('landmark', 'Landmarks', {'landmark', 'attraction'}),
  ('food', 'Food & Drink', {'food', 'cafe'}),
  ('temple', 'Temples', {'temple'}),
  ('wildlife', 'Wildlife & Parks', {'wildlife', 'park'}),
];

class SpotsScreen extends ConsumerStatefulWidget {
  const SpotsScreen({super.key});

  @override
  ConsumerState<SpotsScreen> createState() => _SpotsScreenState();
}

class _SpotsScreenState extends ConsumerState<SpotsScreen> {
  final _searchController = TextEditingController();
  String _query = '';
  String? _category;
  String? _regionId;

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  List<Place> _applyFilters(List<Place> places) {
    final q = _query.trim().toLowerCase();
    final categoryTypes =
        _category == null ? null : _categoryFilters.firstWhere((c) => c.$1 == _category).$3;
    return places.where((p) {
      if (_regionId != null && p.regionId != _regionId) return false;
      if (categoryTypes != null && !categoryTypes.contains(p.type)) return false;
      if (q.isEmpty) return true;
      return p.name.toLowerCase().contains(q) ||
          p.type.toLowerCase().contains(q) ||
          p.region.toLowerCase().contains(q);
    }).toList();
  }

  Future<void> _pickRegion(List<Place> allPlaces) async {
    final options = <String, String>{};
    for (final p in allPlaces) {
      if (p.regionId.isNotEmpty) options[p.regionId] = p.region;
    }
    final sorted = options.entries.toList()..sort((a, b) => a.value.compareTo(b.value));
    final picked = await showModalBottomSheet<String?>(
      context: context,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(20))),
      builder: (context) => SafeArea(
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 8),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              ListTile(
                title: const Text('All regions'),
                trailing: _regionId == null ? const Icon(Icons.check, color: SpottoColors.accent) : null,
                onTap: () => Navigator.pop(context, ''),
              ),
              for (final e in sorted)
                ListTile(
                  title: Text(e.value),
                  trailing: _regionId == e.key ? const Icon(Icons.check, color: SpottoColors.accent) : null,
                  onTap: () => Navigator.pop(context, e.key),
                ),
            ],
          ),
        ),
      ),
    );
    if (picked != null) setState(() => _regionId = picked.isEmpty ? null : picked);
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final places = ref.watch(placesProvider);
    final locationStatus = ref.watch(placesLocationStatusProvider);
    final filtered = _applyFilters(places);
    final sorted = locationStatus == PlacesLocationStatus.sorted &&
        filtered.any((p) => p.distanceKm != null) &&
        filtered.length >= 4;
    final sections = _sectionise(filtered, sorted);
    final regionName = _regionId == null
        ? null
        : places.firstWhere((p) => p.regionId == _regionId, orElse: () => places.first).region;

    final rows = <Widget>[];
    var rowIndex = 0;
    Widget staggered(Widget child) => StaggerIn(index: rowIndex++, child: child);

    rows.add(staggered(Text(regionName != null ? 'Spots in $regionName' : 'Nearby Spots', style: text.headlineSmall)));
    rows.add(const SizedBox(height: 4));
    rows.add(staggered(Text(
      sorted && filtered.first.distanceKm != null
          ? '${filtered.length} curated places · nearest ${_fmtKm(filtered.first.distanceKm!)} away'
          : '${filtered.length} curated places nearby',
      style: text.bodyMedium,
    )));
    rows.add(const SizedBox(height: 14));
    rows.add(staggered(_SearchField(
      controller: _searchController,
      onChanged: (v) => setState(() => _query = v),
    )));
    rows.add(const SizedBox(height: 10));
    rows.add(staggered(_FilterChipRow(
      category: _category,
      regionLabel: regionName,
      onCategoryTap: (id) => setState(() => _category = _category == id ? null : id),
      onRegionTap: () => _pickRegion(places),
    )));
    if (locationStatus == PlacesLocationStatus.deniedForever ||
        locationStatus == PlacesLocationStatus.unavailable) {
      rows.add(const SizedBox(height: 10));
      rows.add(staggered(_LocationHintBanner(
        status: locationStatus,
        onTap: () async {
          if (locationStatus == PlacesLocationStatus.deniedForever) {
            await Geolocator.openAppSettings();
          } else {
            await ref.read(placesProvider.notifier).refresh();
          }
        },
      )));
    }
    rows.add(const SizedBox(height: 16));

    if (filtered.isEmpty) {
      rows.add(staggered(Padding(
        padding: const EdgeInsets.symmetric(vertical: 24),
        child: Text('No spots match your filters.',
            textAlign: TextAlign.center, style: text.bodyMedium?.copyWith(color: SpottoColors.muted)),
      )));
    }

    for (final section in sections) {
      if (section.label != null) {
        rows.add(staggered(_SectionHeader(section: section)));
        rows.add(const SizedBox(height: 12));
      }
      for (final p in section.places) {
        rows.add(staggered(_SpotCard(
          key: ValueKey(p.id),
          place: p,
          distanceLabel: p.distanceKm != null ? _fmtKm(p.distanceKm!) : null,
        )));
        rows.add(const SizedBox(height: 14));
      }
    }

    return SafeArea(
      child: RefreshIndicator(
        onRefresh: () => ref.read(placesProvider.notifier).refresh(),
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 90),
          children: rows,
        ),
      ),
    );
  }
}

class _SearchField extends StatelessWidget {
  const _SearchField({required this.controller, required this.onChanged});

  final TextEditingController controller;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    return TextField(
      controller: controller,
      onChanged: onChanged,
      decoration: InputDecoration(
        hintText: 'Search spots — name, type, region…',
        prefixIcon: const Icon(Icons.search, size: 18, color: SpottoColors.muted),
        suffixIcon: controller.text.isNotEmpty
            ? IconButton(
                icon: const Icon(Icons.close, size: 18, color: SpottoColors.muted),
                onPressed: () {
                  controller.clear();
                  onChanged('');
                },
              )
            : null,
        filled: true,
        fillColor: Colors.white,
        contentPadding: const EdgeInsets.symmetric(vertical: 12),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(14),
          borderSide: const BorderSide(color: SpottoColors.line, width: 1.5),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(14),
          borderSide: const BorderSide(color: SpottoColors.line, width: 1.5),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(14),
          borderSide: const BorderSide(color: SpottoColors.accent, width: 1.5),
        ),
      ),
    );
  }
}

class _FilterChipRow extends StatelessWidget {
  const _FilterChipRow({
    required this.category,
    required this.regionLabel,
    required this.onCategoryTap,
    required this.onRegionTap,
  });

  final String? category;
  final String? regionLabel;
  final ValueChanged<String> onCategoryTap;
  final VoidCallback onRegionTap;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 34,
      child: ListView(
        scrollDirection: Axis.horizontal,
        children: [
          for (final c in _categoryFilters) ...[
            _FilterChip(label: c.$2, selected: category == c.$1, onTap: () => onCategoryTap(c.$1)),
            const SizedBox(width: 8),
          ],
          _FilterChip(
            label: regionLabel ?? 'All regions',
            selected: regionLabel != null,
            trailingIcon: Icons.expand_more,
            onTap: onRegionTap,
          ),
        ],
      ),
    );
  }
}

class _FilterChip extends StatelessWidget {
  const _FilterChip({required this.label, required this.selected, required this.onTap, this.trailingIcon});

  final String label;
  final bool selected;
  final VoidCallback onTap;
  final IconData? trailingIcon;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return InkWell(
      borderRadius: BorderRadius.circular(999),
      onTap: onTap,
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 150),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
        decoration: BoxDecoration(
          color: selected ? SpottoColors.accent : Colors.white,
          borderRadius: BorderRadius.circular(999),
          border: selected ? null : Border.all(color: SpottoColors.line, width: 1.5),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(label,
                style: text.labelMedium?.copyWith(color: selected ? Colors.white : SpottoColors.ink)),
            if (trailingIcon != null) ...[
              const SizedBox(width: 2),
              Icon(trailingIcon, size: 16, color: selected ? Colors.white : SpottoColors.muted),
            ],
          ],
        ),
      ),
    );
  }
}

class _Section {
  final String? label;
  final String? pill;
  final bool primary;
  final List<Place> places;
  const _Section({this.label, this.pill, this.primary = false, required this.places});
}

/// Splits an already server-sorted (nearest-first, unknowns last) places
/// list into a "Nearest to you" group and a "Further afield" group. Uses
/// takeWhile (not where) because that ordering guarantee is what makes this
/// O(n) instead of a full re-sort/re-filter.
List<_Section> _sectionise(List<Place> places, bool locationSorted) {
  if (!locationSorted) return [_Section(places: places)];

  final located = places.where((p) => p.distanceKm != null).toList();
  if (located.isEmpty || places.length < 4) {
    return [_Section(places: places)];
  }

  var near = places.takeWhile((p) => (p.distanceKm ?? double.infinity) <= _kNearRadiusKm).length;
  if (near == 0) near = _kMinNear.clamp(0, located.length);
  if (near > _kMaxNear) near = _kMaxNear;
  if (places.length - near < _kMinTail) near = places.length;

  if (near >= places.length) {
    return [_Section(places: places)];
  }

  final nearPlaces = places.take(near).toList();
  final farPlaces = places.skip(near).toList();
  final farLocated = farPlaces.where((p) => p.distanceKm != null).toList();

  return [
    _Section(
      label: 'Nearest to you',
      pill: 'within ${_fmtKm(nearPlaces.map((p) => p.distanceKm!).reduce((a, b) => a > b ? a : b))}',
      primary: true,
      places: nearPlaces,
    ),
    _Section(
      label: 'Further afield',
      pill: farLocated.isEmpty
          ? null
          : 'from ${_fmtKm(farLocated.map((p) => p.distanceKm!).reduce((a, b) => a < b ? a : b))}',
      places: farPlaces,
    ),
  ];
}

String _fmtKm(double km) {
  if (km < 1) return '${(((km * 1000) / 10).round() * 10)} m';
  if (km < 10) return '${km.toStringAsFixed(1)} km';
  return '${km.round()} km';
}

class _SectionHeader extends StatelessWidget {
  const _SectionHeader({required this.section});

  final _Section section;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        if (section.primary) ...[
          const Icon(Icons.near_me_rounded, size: 15, color: SpottoColors.accent),
          const SizedBox(width: 8),
        ],
        Text(section.label!, style: text.titleMedium?.copyWith(fontSize: 15)),
        if (section.pill != null) ...[
          const SizedBox(width: 8),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 4),
            decoration: BoxDecoration(
              color: section.primary ? SpottoColors.accentTint : Colors.transparent,
              borderRadius: BorderRadius.circular(999),
              border: section.primary ? null : Border.all(color: SpottoColors.line, width: 1.5),
            ),
            child: Text(
              section.pill!,
              style: text.labelMedium?.copyWith(
                fontSize: 11,
                color: section.primary ? SpottoColors.accent : SpottoColors.muted,
              ),
            ),
          ),
        ],
        const SizedBox(width: 10),
        Expanded(child: Container(height: 1.5, color: SpottoColors.line)),
      ],
    );
  }
}

class _SpotCard extends StatelessWidget {
  const _SpotCard({super.key, required this.place, required this.distanceLabel});

  final Place place;
  final String? distanceLabel;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return SpottoCard(
      padding: EdgeInsets.zero,
      onTap: () => showBusinessDrawer(context, place),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(18),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            SizedBox(
              height: 130,
              width: double.infinity,
              child: Stack(
                fit: StackFit.expand,
                children: [
                  // Real uploaded cover photo when the vendor has one, else
                  // fall back to the illustration — see business_drawer.dart
                  // for the same pattern. A card this size never needs more
                  // than the thumbnail tier — there's no viewer for covers.
                  if (place.cover.isNotEmpty)
                    ProgressiveImage(
                      url: resolveMediaUrl(place.coverThumb.isNotEmpty ? place.coverThumb : place.cover),
                      microThumb: place.coverMicro,
                      fit: BoxFit.cover,
                      errorBuilder: (_) => Illustration(kind: place.photo),
                    )
                  else
                    Illustration(kind: place.photo),
                  // Bottom scrim so the white name/subtitle stays readable
                  // over a photographic cover.
                  if (place.cover.isNotEmpty)
                    const DecoratedBox(
                      decoration: BoxDecoration(
                        gradient: LinearGradient(
                          begin: Alignment.topCenter,
                          end: Alignment.bottomCenter,
                          colors: [Colors.transparent, Colors.black54],
                        ),
                      ),
                    ),
                  if (place.deal.active)
                    Positioned(
                      left: 12,
                      top: 12,
                      child: Container(
                        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
                        decoration: BoxDecoration(
                          color: SpottoColors.sun,
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: Text(place.deal.tag ?? 'Deal',
                            style: text.labelSmall?.copyWith(color: SpottoColors.ink)),
                      ),
                    ),
                  Positioned(
                    right: 12,
                    top: 12,
                    child: Container(
                      width: 30,
                      height: 30,
                      alignment: Alignment.center,
                      decoration: BoxDecoration(
                        color: Colors.white.withValues(alpha: 0.92),
                        shape: BoxShape.circle,
                        boxShadow: const [
                          BoxShadow(color: SpottoColors.inkShadow, blurRadius: 6, offset: Offset(0, 2)),
                        ],
                      ),
                      child: Icon(categoryIcon(place.categoryId), size: 16, color: SpottoColors.accent),
                    ),
                  ),
                  Positioned(
                    left: 14,
                    right: 12,
                    bottom: 10,
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.end,
                      children: [
                        Expanded(
                          child: Text(
                            place.name,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: text.titleMedium?.copyWith(color: Colors.white),
                          ),
                        ),
                        if (distanceLabel != null) ...[
                          const SizedBox(width: 10),
                          Container(
                            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                            decoration: BoxDecoration(
                              color: Colors.white.withValues(alpha: 0.92),
                              borderRadius: BorderRadius.circular(999),
                            ),
                            child: Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                const Icon(Icons.near_me_rounded, size: 11, color: SpottoColors.accent),
                                const SizedBox(width: 4),
                                Text(distanceLabel!,
                                    style: text.labelSmall
                                        ?.copyWith(fontSize: 11, height: 1.1, color: SpottoColors.ink)),
                              ],
                            ),
                          ),
                        ],
                      ],
                    ),
                  ),
                ],
              ),
            ),
            Padding(
              padding: const EdgeInsets.all(12),
              child: Row(
                children: [
                  const Icon(Icons.star, size: 16, color: SpottoColors.sun),
                  const SizedBox(width: 4),
                  Text(place.rating.toStringAsFixed(1), style: text.labelLarge),
                  Text(
                    distanceLabel != null || place.distance.isEmpty
                        ? '  ·  ${place.type}'
                        : '  ·  ${place.type}  ·  ${place.distance}',
                    style: text.bodySmall,
                  ),
                  const Spacer(),
                  if (place.wifi.score >= 8) ...[
                    const Icon(Icons.wifi, size: 15, color: SpottoColors.accent),
                    const SizedBox(width: 3),
                    Text('${place.wifi.score}', style: text.labelMedium),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// Tappable hint shown when spots couldn't be sorted by device location —
/// makes the silent /places fallback (repositories.dart's _currentPosition)
/// visible and actionable instead of just looking unsorted.
class _LocationHintBanner extends StatelessWidget {
  const _LocationHintBanner({required this.status, required this.onTap});

  final PlacesLocationStatus status;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final label = status == PlacesLocationStatus.deniedForever
        ? 'Location is off — open Settings to see nearest spots first'
        : 'Enable location to see nearest spots first';
    return InkWell(
      borderRadius: BorderRadius.circular(12),
      onTap: onTap,
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          color: SpottoColors.sun.withValues(alpha: 0.15),
          borderRadius: BorderRadius.circular(12),
        ),
        child: Row(
          children: [
            const Icon(Icons.location_on_outlined, size: 18, color: SpottoColors.ink),
            const SizedBox(width: 8),
            Expanded(child: Text(label, style: text.bodySmall)),
          ],
        ),
      ),
    );
  }
}

