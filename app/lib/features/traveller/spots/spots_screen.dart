// Spots tab: curated business discovery feed with modern multi-section
// carousels, category discovery, personalized travel recommendations,
// and quick filtering.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';

import '../../../core/api/api_client.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/category_icons.dart';
import '../../../core/widgets/illustration.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/progressive_image.dart';
import '../../../core/widgets/stagger_in.dart';
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';
import '../../../data/vendor_categories.dart';
import '../place_detail/business_drawer.dart';

// Nearest-section tuning
const _kNearRadiusKm = 10.0;

// Category filter chips in top scroll row
const _categoryFilters = [
  ('food', 'Food & Drink'),
  ('cafe', 'Cafes'),
  ('landmark', 'Landmarks'),
  ('temple', 'Temples'),
  ('wildlife', 'Wildlife & Parks'),
  ('beach', 'Beaches & Surf'),
  ('cowork', 'Coworking'),
];

// Circular category avatar items
class _CategoryCircleItem {
  final String key;
  final String label;
  final IconData icon;
  final Color tintColor;
  final Color iconColor;

  const _CategoryCircleItem({
    required this.key,
    required this.label,
    required this.icon,
    required this.tintColor,
    required this.iconColor,
  });
}

const _circleCategories = [
  _CategoryCircleItem(
    key: 'food',
    label: 'Food & Dining',
    icon: Icons.restaurant_rounded,
    tintColor: Color(0xFFFDEEE7),
    iconColor: Color(0xFFF0682F),
  ),
  _CategoryCircleItem(
    key: 'cafe',
    label: 'Cafes & Coffee',
    icon: Icons.local_cafe_rounded,
    tintColor: Color(0xFFF7EFE8),
    iconColor: Color(0xFFC2683F),
  ),
  _CategoryCircleItem(
    key: 'landmark',
    label: 'Landmarks',
    icon: Icons.tour_rounded,
    tintColor: Color(0xFFE2F3F5),
    iconColor: Color(0xFF0B6F86),
  ),
  _CategoryCircleItem(
    key: 'temple',
    label: 'Temples & Culture',
    icon: Icons.temple_buddhist_rounded,
    tintColor: Color(0xFFFEF5D9),
    iconColor: Color(0xFFEAA200),
  ),
  _CategoryCircleItem(
    key: 'wildlife',
    label: 'Wildlife & Nature',
    icon: Icons.park_rounded,
    tintColor: Color(0xFFE8F5E9),
    iconColor: Color(0xFF2E7D32),
  ),
  _CategoryCircleItem(
    key: 'beach',
    label: 'Beaches & Surf',
    icon: Icons.waves_rounded,
    tintColor: Color(0xFFE1F5FE),
    iconColor: Color(0xFF0288D1),
  ),
  _CategoryCircleItem(
    key: 'cowork',
    label: 'Nomad & Work',
    icon: Icons.laptop_chromebook_rounded,
    tintColor: Color(0xFFECEFF1),
    iconColor: Color(0xFF0B4A54),
  ),
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
    return places.where((p) {
      if (_regionId != null && p.regionId != _regionId) return false;
      if (_category != null && !_matchesCategory(p, _category!)) return false;
      if (q.isEmpty) return true;
      return p.name.toLowerCase().contains(q) ||
          p.type.toLowerCase().contains(q) ||
          p.region.toLowerCase().contains(q) ||
          p.tags.any((t) => t.toLowerCase().contains(q));
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
    final userPrefs = ref.watch(prefsProvider);

    final filtered = _applyFilters(places);
    final sorted = locationStatus == PlacesLocationStatus.sorted &&
        filtered.any((p) => p.distanceKm != null) &&
        filtered.length >= 4;

    final regionName = _regionId == null
        ? null
        : places.firstWhere((p) => p.regionId == _regionId, orElse: () => places.first).region;

    final isSearchingOrFiltered = _query.isNotEmpty || _category != null;

    final rows = <Widget>[];
    var rowIndex = 0;
    Widget staggered(Widget child) => StaggerIn(index: rowIndex++, child: child);

    // ── Header Title & Subtitle ─────────────────────────────────────────────
    rows.add(Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          staggered(Text(
            regionName != null ? 'Spots in $regionName' : 'Curated Spots',
            style: text.headlineSmall?.copyWith(fontWeight: FontWeight.w800, color: SpottoColors.ink),
          )),
          const SizedBox(height: 4),
          staggered(Text(
            sorted && filtered.first.distanceKm != null
                ? '${filtered.length} curated places · nearest ${_fmtKm(filtered.first.distanceKm!)} away'
                : '${filtered.length} curated places across Sri Lanka',
            style: text.bodyMedium?.copyWith(color: SpottoColors.muted),
          )),
        ],
      ),
    ));

    // ── Search Field ────────────────────────────────────────────────────────
    rows.add(const SizedBox(height: 14));
    rows.add(Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: staggered(_SearchField(
        controller: _searchController,
        onChanged: (v) => setState(() => _query = v),
      )),
    ));

    // ── Filter Chips Row ────────────────────────────────────────────────────
    rows.add(const SizedBox(height: 10));
    rows.add(staggered(_FilterChipRow(
      category: _category,
      regionLabel: regionName,
      onCategoryTap: (id) => setState(() => _category = _category == id ? null : id),
      onRegionTap: () => _pickRegion(places),
    )));

    // ── Location Hint Banner (if denied) ────────────────────────────────────
    if (locationStatus == PlacesLocationStatus.deniedForever ||
        locationStatus == PlacesLocationStatus.unavailable) {
      rows.add(const SizedBox(height: 10));
      rows.add(Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16),
        child: staggered(_LocationHintBanner(
          status: locationStatus,
          onTap: () async {
            if (locationStatus == PlacesLocationStatus.deniedForever) {
              await Geolocator.openAppSettings();
            } else {
              await ref.read(placesProvider.notifier).refresh();
            }
          },
        )),
      ));
    }

    rows.add(const SizedBox(height: 16));

    // ── Content: Active Search/Filter Grid vs. Multi-Section Discovery Feed ──
    if (isSearchingOrFiltered) {
      // Result summary bar
      rows.add(Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16),
        child: Row(
          children: [
            Text(
              '${filtered.length} spot${filtered.length == 1 ? '' : 's'} found',
              style: text.titleSmall?.copyWith(fontWeight: FontWeight.w700, color: SpottoColors.ink),
            ),
            if (_category != null) ...[
              const SizedBox(width: 8),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                decoration: BoxDecoration(
                  color: SpottoColors.accentTint,
                  borderRadius: BorderRadius.circular(999),
                ),
                child: Text(
                  _circleCategories.firstWhere((c) => c.key == _category, orElse: () => _circleCategories.first).label,
                  style: text.labelSmall?.copyWith(color: SpottoColors.accent, fontWeight: FontWeight.w600, fontSize: 11),
                ),
              ),
            ],
            const Spacer(),
            TextButton(
              onPressed: () {
                _searchController.clear();
                setState(() {
                  _query = '';
                  _category = null;
                });
              },
              child: const Text('Reset', style: TextStyle(color: SpottoColors.accent, fontWeight: FontWeight.w600)),
            ),
          ],
        ),
      ));
      rows.add(const SizedBox(height: 10));

      if (filtered.isEmpty) {
        rows.add(Padding(
          padding: const EdgeInsets.symmetric(vertical: 40, horizontal: 24),
          child: Column(
            children: [
              Icon(Icons.search_off_rounded, size: 48, color: SpottoColors.muted.withValues(alpha: 0.5)),
              const SizedBox(height: 12),
              Text(
                'No spots match your filters.',
                textAlign: TextAlign.center,
                style: text.titleSmall?.copyWith(fontWeight: FontWeight.w600, color: SpottoColors.ink),
              ),
              const SizedBox(height: 4),
              Text(
                'Try adjusting your search query, clearing the category, or selecting another region.',
                textAlign: TextAlign.center,
                style: text.bodySmall?.copyWith(color: SpottoColors.muted),
              ),
            ],
          ),
        ));
      } else {
        rows.add(Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: GridView.builder(
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            itemCount: filtered.length,
            gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
              crossAxisCount: 2,
              mainAxisSpacing: 12,
              crossAxisSpacing: 12,
              childAspectRatio: 0.91,
            ),
            itemBuilder: (context, i) => _SpotTileCard(place: filtered[i]),
          ),
        ));
      }
    } else {
      // Discovery Mode: Multi-section carousels & categorized discovery feed

      // 1. Nearest to you
      final nearPlaces = sorted
          ? filtered.takeWhile((p) => (p.distanceKm ?? double.infinity) <= _kNearRadiusKm).take(8).toList()
          : <Place>[];

      if (nearPlaces.length >= 2) {
        final maxKm = nearPlaces.map((p) => p.distanceKm!).reduce((a, b) => a > b ? a : b);
        rows.add(staggered(_SpotSectionHeader(
          title: 'Nearest to you',
          pill: 'within ${_fmtKm(maxKm)}',
          leadIcon: Icons.near_me_rounded,
          onActionTap: () => setState(() => _category = 'food'),
        )));
        rows.add(const SizedBox(height: 12));
        rows.add(staggered(_QuadTileCarousel(places: nearPlaces)));
        rows.add(const SizedBox(height: 24));
      }

      // 2. Explore Categories (Circle Avatar Row)
      rows.add(staggered(const _SpotSectionHeader(
        title: 'Explore Categories',
        pill: 'Curated',
      )));
      rows.add(const SizedBox(height: 12));
      rows.add(staggered(_CategoryCircleCarousel(
        selectedCategory: _category,
        places: filtered,
        onSelectCategory: (catKey) => setState(() => _category = _category == catKey ? null : catKey),
      )));
      rows.add(const SizedBox(height: 24));

      // 3. Hot Deals & Perks
      final dealPlaces = filtered.where((p) => p.deal.active).take(8).toList();
      if (dealPlaces.length >= 2) {
        rows.add(staggered(_SpotSectionHeader(
          title: 'Hot Deals & Perks',
          pill: '${dealPlaces.length} perks',
          leadIcon: Icons.local_offer_rounded,
        )));
        rows.add(const SizedBox(height: 12));
        rows.add(staggered(_QuadTileCarousel(places: dealPlaces)));
        rows.add(const SizedBox(height: 24));
      }

      // 4. Personalized / Matched to Travel Style
      final matchedPlaces = _matchedPlacesForUser(filtered, userPrefs);
      if (matchedPlaces.length >= 2) {
        rows.add(staggered(const _SpotSectionHeader(
          title: 'Matched to Your Style',
          pill: 'For you',
          leadIcon: Icons.auto_awesome_rounded,
        )));
        rows.add(const SizedBox(height: 12));
        rows.add(staggered(_QuadTileCarousel(places: matchedPlaces)));
        rows.add(const SizedBox(height: 24));
      }

      // 5. Top Rated Gems
      final topRatedPlaces = (List<Place>.from(filtered)..sort((a, b) => b.rating.compareTo(a.rating)))
          .where((p) => p.rating >= 4.0)
          .take(8)
          .toList();
      if (topRatedPlaces.length >= 2) {
        rows.add(staggered(const _SpotSectionHeader(
          title: 'Top Rated Gems',
          pill: '★ 4.0+',
          leadIcon: Icons.star_rounded,
        )));
        rows.add(const SizedBox(height: 12));
        rows.add(staggered(_QuadTileCarousel(places: topRatedPlaces)));
        rows.add(const SizedBox(height: 24));
      }

      // 6. Nomad & Wi-Fi Friendly
      final nomadPlaces = filtered.where((p) => p.wifi.score >= 8 || p.tags.contains('cowork')).take(8).toList();
      if (nomadPlaces.length >= 2) {
        rows.add(staggered(const _SpotSectionHeader(
          title: 'Nomad & Wi-Fi Friendly',
          pill: 'High speed',
          leadIcon: Icons.wifi_rounded,
        )));
        rows.add(const SizedBox(height: 12));
        rows.add(staggered(_QuadTileCarousel(places: nomadPlaces)));
        rows.add(const SizedBox(height: 24));
      }

      // 7. More Spots to Explore (continuous 2-column grid)
      final seenIds = {
        ...nearPlaces.map((p) => p.id),
        ...dealPlaces.map((p) => p.id),
        ...matchedPlaces.map((p) => p.id),
        ...topRatedPlaces.map((p) => p.id),
        ...nomadPlaces.map((p) => p.id),
      };
      final remainingPlaces = filtered.where((p) => !seenIds.contains(p.id)).toList();
      final allToBrowse = remainingPlaces.isNotEmpty ? remainingPlaces : filtered;

      rows.add(staggered(_SpotSectionHeader(
        title: regionName != null ? 'All Spots in $regionName' : 'More Spots to Explore',
        pill: '${allToBrowse.length} spots',
      )));
      rows.add(const SizedBox(height: 12));
      rows.add(Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16),
        child: GridView.builder(
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          itemCount: allToBrowse.length,
          gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
            crossAxisCount: 2,
            mainAxisSpacing: 12,
            crossAxisSpacing: 12,
            childAspectRatio: 0.91,
          ),
          itemBuilder: (context, i) => _SpotTileCard(place: allToBrowse[i]),
        ),
      ));
    }

    return SafeArea(
      child: RefreshIndicator(
        onRefresh: () => ref.read(placesProvider.notifier).refresh(),
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.fromLTRB(0, 12, 0, 90),
          children: rows,
        ),
      ),
    );
  }
}

// ── Search & Filter Widgets ─────────────────────────────────────────────────

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
        prefixIcon: const Icon(Icons.search, size: 20, color: SpottoColors.muted),
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
      height: 36,
      child: ListView(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 16),
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
          boxShadow: [
            BoxShadow(
              color: SpottoColors.inkShadow.withValues(alpha: selected ? 0.15 : 0.04),
              blurRadius: 4,
              offset: const Offset(0, 1),
            ),
          ],
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(label,
                style: text.labelMedium?.copyWith(
                  color: selected ? Colors.white : SpottoColors.ink,
                  fontWeight: selected ? FontWeight.w700 : FontWeight.w500,
                )),
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

// ── Multi-Section Headers & Carousels ───────────────────────────────────────

class _SpotSectionHeader extends StatelessWidget {
  const _SpotSectionHeader({
    required this.title,
    this.pill,
    this.leadIcon,
    this.onActionTap,
  });

  final String title;
  final String? pill;
  final IconData? leadIcon;
  final VoidCallback? onActionTap;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.center,
        children: [
          if (leadIcon != null) ...[
            Icon(leadIcon, size: 18, color: SpottoColors.accent),
            const SizedBox(width: 6),
          ],
          Text(
            title,
            style: text.titleMedium?.copyWith(
              fontWeight: FontWeight.w800,
              fontSize: 18,
              color: SpottoColors.ink,
            ),
          ),
          if (pill != null) ...[
            const SizedBox(width: 8),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3.5),
              decoration: BoxDecoration(
                color: SpottoColors.accentTint,
                borderRadius: BorderRadius.circular(999),
              ),
              child: Text(
                pill!,
                style: text.labelSmall?.copyWith(
                  fontSize: 11,
                  fontWeight: FontWeight.w700,
                  color: SpottoColors.accent,
                ),
              ),
            ),
          ],
          const Spacer(),
          if (onActionTap != null)
            PressScale(
              onTap: onActionTap,
              child: Container(
                width: 32,
                height: 32,
                alignment: Alignment.center,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: Colors.white,
                  border: Border.all(color: SpottoColors.line, width: 1.2),
                  boxShadow: [
                    BoxShadow(
                      color: SpottoColors.inkShadow.withValues(alpha: 0.05),
                      blurRadius: 4,
                      offset: const Offset(0, 1),
                    ),
                  ],
                ),
                child: const Icon(
                  Icons.arrow_forward_rounded,
                  size: 16,
                  color: SpottoColors.ink,
                ),
              ),
            ),
        ],
      ),
    );
  }
}

/// 4-Tile quadrant unit with horizontal scroll.
/// Computes column width so exactly 2 columns (4 tiles total) fit on screen.
class _QuadTileCarousel extends StatelessWidget {
  const _QuadTileCarousel({required this.places});

  final List<Place> places;

  @override
  Widget build(BuildContext context) {
    final sw = MediaQuery.sizeOf(context).width;
    // Exactly 2 columns visible on screen: padding is 16 on each side (32 total), gap is 12.
    final colWidth = ((sw - 32 - 12) / 2).clamp(160.0, 220.0);
    final colCount = (places.length / 2).ceil();

    return SizedBox(
      height: 374,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        physics: const BouncingScrollPhysics(),
        padding: const EdgeInsets.symmetric(horizontal: 16),
        itemCount: colCount,
        separatorBuilder: (_, __) => const SizedBox(width: 12),
        itemBuilder: (context, colIndex) {
          final topIndex = colIndex * 2;
          final bottomIndex = topIndex + 1;
          final topPlace = places[topIndex];
          final bottomPlace = bottomIndex < places.length ? places[bottomIndex] : null;

          return SizedBox(
            width: colWidth,
            child: Column(
              children: [
                _SpotTileCard(place: topPlace, width: colWidth),
                const SizedBox(height: 12),
                if (bottomPlace != null)
                  _SpotTileCard(place: bottomPlace, width: colWidth)
                else
                  const Spacer(),
              ],
            ),
          );
        },
      ),
    );
  }
}

/// Circular category carousel with icon avatars and live spot counts
class _CategoryCircleCarousel extends StatelessWidget {
  const _CategoryCircleCarousel({
    required this.selectedCategory,
    required this.places,
    required this.onSelectCategory,
  });

  final String? selectedCategory;
  final List<Place> places;
  final ValueChanged<String> onSelectCategory;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;

    return SizedBox(
      height: 106,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        physics: const BouncingScrollPhysics(),
        padding: const EdgeInsets.symmetric(horizontal: 16),
        itemCount: _circleCategories.length,
        separatorBuilder: (_, __) => const SizedBox(width: 16),
        itemBuilder: (context, i) {
          final cat = _circleCategories[i];
          final isSelected = selectedCategory == cat.key;
          final count = places.where((p) => _matchesCategory(p, cat.key)).length;

          return PressScale(
            onTap: () => onSelectCategory(cat.key),
            child: SizedBox(
              width: 72,
              child: Column(
                children: [
                  AnimatedContainer(
                    duration: const Duration(milliseconds: 200),
                    width: 60,
                    height: 60,
                    decoration: BoxDecoration(
                      color: cat.tintColor,
                      shape: BoxShape.circle,
                      border: Border.all(
                        color: isSelected ? SpottoColors.accent : SpottoColors.line,
                        width: isSelected ? 2.5 : 1.2,
                      ),
                      boxShadow: [
                        BoxShadow(
                          color: (isSelected ? SpottoColors.accent : SpottoColors.inkShadow)
                              .withValues(alpha: isSelected ? 0.25 : 0.06),
                          blurRadius: isSelected ? 8 : 4,
                          offset: const Offset(0, 2),
                        ),
                      ],
                    ),
                    child: Icon(cat.icon, size: 28, color: cat.iconColor),
                  ),
                  const SizedBox(height: 6),
                  Text(
                    cat.label,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    textAlign: TextAlign.center,
                    style: text.labelSmall?.copyWith(
                      fontWeight: isSelected ? FontWeight.w800 : FontWeight.w600,
                      fontSize: 11,
                      color: isSelected ? SpottoColors.accent : SpottoColors.ink,
                    ),
                  ),
                  const SizedBox(height: 1),
                  Text(
                    '$count spot${count == 1 ? '' : 's'}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    textAlign: TextAlign.center,
                    style: text.bodySmall?.copyWith(
                      fontSize: 10,
                      color: SpottoColors.muted,
                    ),
                  ),
                ],
              ),
            ),
          );
        },
      ),
    );
  }
}

/// Compact Modern Spot Tile Card
class _SpotTileCard extends StatelessWidget {
  const _SpotTileCard({
    required this.place,
    this.width,
  });

  final Place place;
  final double? width;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;

    return PressScale(
      onTap: () => showBusinessDrawer(context, place),
      child: Container(
        width: width,
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: SpottoColors.line, width: 1.2),
          boxShadow: [
            BoxShadow(
              color: SpottoColors.inkShadow.withValues(alpha: 0.05),
              blurRadius: 8,
              offset: const Offset(0, 3),
            ),
          ],
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            // ── Cover Image with Badge Overlays ──────────────────────────────
            ClipRRect(
              borderRadius: const BorderRadius.vertical(top: Radius.circular(15)),
              child: SizedBox(
                width: double.infinity,
                height: 104,
                child: Stack(
                  fit: StackFit.expand,
                  children: [
                    if (place.cover.isNotEmpty)
                      ProgressiveImage(
                        url: resolveMediaUrl(place.coverThumb.isNotEmpty ? place.coverThumb : place.cover),
                        microThumb: place.coverMicro,
                        fit: BoxFit.cover,
                        errorBuilder: (_) => Illustration(kind: place.photo),
                      )
                    else
                      Illustration(kind: place.photo),

                    // Deal badge top-left
                    if (place.deal.active)
                      Positioned(
                        left: 8,
                        top: 8,
                        child: Container(
                          padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 3.5),
                          decoration: BoxDecoration(
                            color: const Color(0xFFD32F2F),
                            borderRadius: BorderRadius.circular(6),
                            boxShadow: const [
                              BoxShadow(color: Colors.black26, blurRadius: 4, offset: Offset(0, 1)),
                            ],
                          ),
                          child: Text(
                            place.deal.tag ?? 'Deal',
                            style: const TextStyle(
                              color: Colors.white,
                              fontWeight: FontWeight.w700,
                              fontSize: 10,
                              letterSpacing: 0.2,
                            ),
                          ),
                        ),
                      )
                    else if (place.distanceKm != null)
                      Positioned(
                        left: 8,
                        top: 8,
                        child: Container(
                          padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 3),
                          decoration: BoxDecoration(
                            color: Colors.black.withValues(alpha: 0.65),
                            borderRadius: BorderRadius.circular(6),
                          ),
                          child: Text(
                            _fmtKm(place.distanceKm!),
                            style: const TextStyle(
                              color: Colors.white,
                              fontWeight: FontWeight.w600,
                              fontSize: 10,
                            ),
                          ),
                        ),
                      ),

                    // Category micro-icon top-right
                    Positioned(
                      right: 8,
                      top: 8,
                      child: Container(
                        width: 26,
                        height: 26,
                        alignment: Alignment.center,
                        decoration: BoxDecoration(
                          color: Colors.white.withValues(alpha: 0.94),
                          shape: BoxShape.circle,
                          boxShadow: const [
                            BoxShadow(color: Colors.black12, blurRadius: 4, offset: Offset(0, 1)),
                          ],
                        ),
                        child: Icon(
                          categoryIcon(place.categoryId),
                          size: 14,
                          color: SpottoColors.accent,
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),

            // ── Information Text Area ────────────────────────────────────────
            Padding(
              padding: const EdgeInsets.fromLTRB(10, 8, 10, 9),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(
                    place.name,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: text.titleSmall?.copyWith(
                      fontWeight: FontWeight.w700,
                      fontSize: 13.5,
                      color: SpottoColors.ink,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Row(
                    children: [
                      const Icon(Icons.star_rounded, size: 14, color: SpottoColors.sun),
                      const SizedBox(width: 3),
                      Text(
                        place.rating > 0 ? place.rating.toStringAsFixed(1) : 'New',
                        style: text.labelSmall?.copyWith(
                          fontWeight: FontWeight.w700,
                          fontSize: 11.5,
                          color: SpottoColors.ink,
                        ),
                      ),
                      Expanded(
                        child: Text(
                          ' · ${_categoryLabel(place)}',
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: text.bodySmall?.copyWith(
                            fontSize: 11.5,
                            color: SpottoColors.muted,
                          ),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 2),
                  Row(
                    children: [
                      if (place.deal.active) ...[
                        const Icon(Icons.local_offer_outlined, size: 12, color: SpottoColors.accent),
                        const SizedBox(width: 3),
                        Expanded(
                          child: Text(
                            'Perk available',
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: text.labelSmall?.copyWith(
                              fontSize: 11,
                              color: SpottoColors.accent,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                        ),
                      ] else ...[
                        Expanded(
                          child: Text(
                            _formatStatusAndDistance(place),
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: text.bodySmall?.copyWith(
                              fontSize: 11,
                              color: SpottoColors.muted,
                            ),
                          ),
                        ),
                      ],
                    ],
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ── Location Hint Banner ────────────────────────────────────────────────────

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

// ── Helpers ─────────────────────────────────────────────────────────────────

String _fmtKm(double km) {
  if (km < 1) return '${(((km * 1000) / 10).round() * 10)} m';
  if (km < 10) return '${km.toStringAsFixed(1)} km';
  return '${km.round()} km';
}

bool _matchesCategory(Place p, String categoryKey) {
  final t = p.type.toLowerCase();
  final cid = p.categoryId.toLowerCase();
  switch (categoryKey) {
    case 'food':
      return {'food', 'cafe', 'restaurants'}.contains(t) || {'restaurants', 'coffee'}.contains(cid);
    case 'cafe':
      return t == 'cafe' ||
          cid == 'coffee' ||
          p.tags.any((tag) => tag.toLowerCase() == 'cafe' || tag.toLowerCase() == 'coffee');
    case 'landmark':
      return {'landmark', 'attraction'}.contains(t) || cid == 'things_to_do';
    case 'temple':
      return t == 'temple';
    case 'wildlife':
      return {'wildlife', 'park'}.contains(t);
    case 'beach':
      return p.seaView || p.tags.any((tag) => tag.toLowerCase() == 'beach' || tag.toLowerCase() == 'surf');
    case 'cowork':
      return p.wifi.score >= 8 || p.tags.any((tag) => tag.toLowerCase() == 'cowork' || tag.toLowerCase() == 'coworking');
    default:
      return t == categoryKey || cid == categoryKey;
  }
}

String _categoryLabel(Place place) {
  if (place.type.isNotEmpty) {
    return place.type[0].toUpperCase() + place.type.substring(1);
  }
  final cat = categoryById(place.categoryId);
  if (cat != null && cat.label.isNotEmpty) return cat.label;
  return 'Spot';
}

String _formatStatusAndDistance(Place place) {
  final parts = <String>[];
  if (place.distanceKm != null) {
    parts.add(_fmtKm(place.distanceKm!));
  } else if (place.distance.isNotEmpty) {
    parts.add(place.distance);
  }
  if (place.openNow) {
    parts.add('Open');
  } else if (place.wifi.score >= 8) {
    parts.add('Wi-Fi ${place.wifi.score}');
  } else if (place.region.isNotEmpty) {
    parts.add(place.region);
  }
  return parts.isEmpty ? 'Curated spot' : parts.join(' · ');
}

List<Place> _matchedPlacesForUser(List<Place> places, UserPrefs? prefs) {
  if (prefs == null || (prefs.styles.isEmpty && prefs.interests.isEmpty)) {
    return const [];
  }
  final targetStyles = prefs.styles.map((s) => s.toLowerCase()).toSet();
  final targetInterests = prefs.interests.map((i) => i.toLowerCase()).toSet();

  return places.where((p) {
    for (final s in p.styles) {
      if (targetStyles.contains(s.toLowerCase())) return true;
    }
    for (final t in p.tags) {
      final tl = t.toLowerCase();
      if (targetInterests.contains(tl) || targetStyles.contains(tl)) return true;
    }
    if (targetInterests.contains(p.type.toLowerCase())) return true;
    return false;
  }).take(8).toList();
}
