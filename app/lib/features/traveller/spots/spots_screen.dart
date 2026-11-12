// Spots tab: curated business discovery feed with modern multi-section
// carousels, category discovery, personalized travel recommendations,
// and 120fps virtualized sliver scrolling.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';

import '../../../core/api/api_client.dart';
import '../../../core/l10n/l10n_provider.dart';
import '../../../core/theme/palette.dart';
import '../../../core/utils/vibe_matching.dart';
import '../../../core/widgets/category_icons.dart';
import '../../../core/widgets/illustration.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/progressive_image.dart';
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';
import '../../../data/vendor_categories.dart';
import '../place_detail/business_drawer.dart';

// Nearest-section tuning
const _kNearRadiusKm = 10.0;

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
    key: 'vibe',
    label: 'My Vibe',
    icon: Icons.auto_awesome_rounded,
    tintColor: Color(0xFFE0F2F1),
    iconColor: Color(0xFF00796B),
  ),
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

  List<Place> _applyFilters(List<Place> places, [UserPrefs? prefs]) {
    final q = _query.trim().toLowerCase();
    final res = places.where((p) {
      if (_regionId != null && p.regionId != _regionId) return false;
      if (_category != null) {
        if (_category == 'vibe') {
          if (prefs != null && (prefs.styles.isNotEmpty || prefs.interests.isNotEmpty)) {
            final vm = calculateVibeMatch(place: p, prefs: prefs);
            if (vm.percentage < 65) return false;
          }
        } else if (!_matchesCategory(p, _category!)) {
          return false;
        }
      }
      if (q.isEmpty) return true;
      return p.name.toLowerCase().contains(q) ||
          p.type.toLowerCase().contains(q) ||
          p.region.toLowerCase().contains(q) ||
          p.tags.any((t) => t.toLowerCase().contains(q));
    }).toList();
    if (_category == 'vibe' && prefs != null && (prefs.styles.isNotEmpty || prefs.interests.isNotEmpty)) {
      res.sort((a, b) {
        final sa = calculateVibeMatch(place: a, prefs: prefs).percentage;
        final sb = calculateVibeMatch(place: b, prefs: prefs).percentage;
        return sb.compareTo(sa);
      });
    }
    return res;
  }

  Future<void> _pickRegion(List<Place> allPlaces) async {
    final options = <String, String>{};
    for (final p in allPlaces) {
      final rid = p.regionId.trim();
      final rname = p.region.trim();
      if (rid.isNotEmpty && rname.isNotEmpty) {
        options[rid] = rname;
      }
    }
    final sorted = options.entries.toList()..sort((a, b) => a.value.compareTo(b.value));
    final picked = await showModalBottomSheet<String?>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      clipBehavior: Clip.antiAlias,
      backgroundColor: Colors.white,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(20))),
      builder: (context) => SafeArea(
        top: false,
        child: Container(
          constraints: BoxConstraints(
            maxHeight: MediaQuery.sizeOf(context).height * 0.75,
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Container(
                width: 36,
                height: 4,
                margin: const EdgeInsets.only(top: 10, bottom: 8),
                decoration: BoxDecoration(
                  color: SpottoColors.line,
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
              Padding(
                padding: const EdgeInsets.fromLTRB(20, 2, 8, 8),
                child: Row(
                  children: [
                    Text(
                      'Select Region',
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.w700,
                        color: SpottoColors.ink,
                      ),
                    ),
                    const Spacer(),
                    if (_regionId != null)
                      TextButton(
                        onPressed: () => Navigator.pop(context, ''),
                        child: const Text('Reset', style: TextStyle(color: SpottoColors.accent, fontWeight: FontWeight.w600)),
                      ),
                    IconButton(
                      icon: const Icon(Icons.close_rounded, color: SpottoColors.muted),
                      tooltip: 'Close',
                      onPressed: () => Navigator.pop(context),
                    ),
                  ],
                ),
              ),
              const Divider(height: 1, color: SpottoColors.line),
              Flexible(
                child: ListView(
                  shrinkWrap: true,
                  physics: const BouncingScrollPhysics(),
                  padding: const EdgeInsets.symmetric(vertical: 8),
                  children: [
                    ListTile(
                      leading: const Icon(Icons.public_rounded, color: SpottoColors.accent),
                      title: Text(ref.read(l10nProvider).allRegions, style: const TextStyle(fontWeight: FontWeight.w600)),
                      trailing: _regionId == null ? const Icon(Icons.check, color: SpottoColors.accent) : null,
                      onTap: () => Navigator.pop(context, ''),
                    ),
                    for (final e in sorted)
                      ListTile(
                        leading: const Icon(Icons.place_outlined, color: SpottoColors.muted),
                        title: Text(e.value),
                        trailing: _regionId == e.key ? const Icon(Icons.check, color: SpottoColors.accent) : null,
                        onTap: () => Navigator.pop(context, e.key),
                      ),
                  ],
                ),
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
    final l10n = ref.watch(l10nProvider);
    final places = ref.watch(placesProvider);
    final locationStatus = ref.watch(placesLocationStatusProvider);
    final userPrefs = ref.watch(prefsProvider);

    final filtered = _applyFilters(places, userPrefs);
    final sorted = locationStatus == PlacesLocationStatus.sorted &&
        filtered.any((p) => p.distanceKm != null) &&
        filtered.length >= 4;

    final regionName = _regionId == null || places.isEmpty
        ? null
        : places.where((p) => p.regionId == _regionId).firstOrNull?.region;

    final isSearchingOrFiltered = _query.isNotEmpty || _category != null;

    final slivers = <Widget>[];

    // ── Header: Title & Subtitle ────────────────────────────────────────────
    slivers.add(SliverToBoxAdapter(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              regionName != null ? 'Spots in $regionName' : 'Curated Spots',
              style: text.headlineSmall?.copyWith(fontWeight: FontWeight.w800, color: SpottoColors.ink),
            ),
            const SizedBox(height: 4),
            Text(
              sorted && filtered.first.distanceKm != null
                  ? '${filtered.length} curated places · nearest ${_fmtKm(filtered.first.distanceKm!)} away'
                  : '${filtered.length} curated places across Sri Lanka',
              style: text.bodyMedium?.copyWith(color: SpottoColors.muted),
            ),
          ],
        ),
      ),
    ));

    // ── Search Field ────────────────────────────────────────────────────────
    slivers.add(SliverToBoxAdapter(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 14, 16, 14),
        child: _SearchField(
          controller: _searchController,
          onChanged: (v) => setState(() => _query = v),
        ),
      ),
    ));

    // ── Location Hint Banner (if denied) ────────────────────────────────────
    if (locationStatus == PlacesLocationStatus.deniedForever ||
        locationStatus == PlacesLocationStatus.unavailable) {
      slivers.add(SliverToBoxAdapter(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(16, 0, 16, 14),
          child: _LocationHintBanner(
            status: locationStatus,
            onTap: () async {
              if (locationStatus == PlacesLocationStatus.deniedForever) {
                await Geolocator.openAppSettings();
              } else {
                await ref.read(placesProvider.notifier).refresh();
              }
            },
          ),
        ),
      ));
    }

    // ── Content: Active Search/Filter Grid vs. 120fps Discovery Feed ─────────
    if (isSearchingOrFiltered) {
      // 1. Result Summary Header
      slivers.add(SliverToBoxAdapter(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
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
        ),
      ));

      if (filtered.isEmpty) {
        slivers.add(SliverToBoxAdapter(
          child: Padding(
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
          ),
        ));
      } else {
        // True Virtualized 120fps Grid
        slivers.add(SliverPadding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          sliver: SliverGrid(
            gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
              crossAxisCount: 2,
              mainAxisSpacing: 12,
              crossAxisSpacing: 12,
              childAspectRatio: 0.91,
            ),
            delegate: SliverChildBuilderDelegate(
              (context, index) => _SpotTileCard(place: filtered[index]),
              childCount: filtered.length,
            ),
          ),
        ));
      }
    } else {
      // ── Discovery Mode: Fully Virtualized Multi-Section Feed ───────────────

      // 1. Nearest to you
      final nearPlaces = sorted
          ? filtered.takeWhile((p) => (p.distanceKm ?? double.infinity) <= _kNearRadiusKm).take(8).toList()
          : <Place>[];

      if (nearPlaces.length >= 2) {
        final maxKm = nearPlaces.map((p) => p.distanceKm!).reduce((a, b) => a > b ? a : b);
        slivers.add(SliverToBoxAdapter(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _SpotSectionHeader(
                title: 'Nearest to you',
                pill: 'within ${_fmtKm(maxKm)}',
                leadIcon: Icons.near_me_rounded,
                onActionTap: () => setState(() => _category = 'food'),
              ),
              const SizedBox(height: 12),
              RepaintBoundary(child: _QuadTileCarousel(places: nearPlaces)),
              const SizedBox(height: 24),
            ],
          ),
        ));
      }

      // 2. Explore Categories with "All Regions" Item at the Last Position
      slivers.add(SliverToBoxAdapter(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const _SpotSectionHeader(
              title: 'Explore Categories',
              pill: 'Curated',
            ),
            const SizedBox(height: 12),
            RepaintBoundary(
              child: _CategoryCircleCarousel(
                selectedCategory: _category,
                selectedRegionName: regionName,
                places: filtered,
                totalPlacesCount: places.length,
                userPrefs: userPrefs,
                onSelectCategory: (catKey) => setState(() => _category = _category == catKey ? null : catKey),
                onPickRegion: () => _pickRegion(places),
              ),
            ),
            const SizedBox(height: 24),
          ],
        ),
      ));

      // 3. Featured Partners / Sponsored Ads
      final sponsoredPlaces = filtered.where((p) => p.isSponsored).take(8).toList();
      if (sponsoredPlaces.length >= 2) {
        slivers.add(SliverToBoxAdapter(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _SpotSectionHeader(
                title: 'Featured Partners',
                pill: 'Sponsored',
                leadIcon: Icons.auto_awesome_rounded,
              ),
              const SizedBox(height: 12),
              RepaintBoundary(child: _QuadTileCarousel(places: sponsoredPlaces)),
              const SizedBox(height: 24),
            ],
          ),
        ));
      }

      // 4. Hot Deals & Perks
      final dealPlaces = filtered.where((p) => p.deal.active).take(8).toList();
      if (dealPlaces.length >= 2) {
        slivers.add(SliverToBoxAdapter(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _SpotSectionHeader(
                title: 'Hot Deals & Perks',
                pill: '${dealPlaces.length} perks',
                leadIcon: Icons.local_offer_rounded,
              ),
              const SizedBox(height: 12),
              RepaintBoundary(child: _QuadTileCarousel(places: dealPlaces)),
              const SizedBox(height: 24),
            ],
          ),
        ));
      }

      // 5. Matched to Your Travel Style
      final matchedPlaces = _matchedPlacesForUser(filtered, userPrefs);
      if (matchedPlaces.length >= 2) {
        slivers.add(SliverToBoxAdapter(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const _SpotSectionHeader(
                title: 'Matched to Your Vibe',
                pill: 'Top fit for your style',
                leadIcon: Icons.auto_awesome_rounded,
              ),
              const SizedBox(height: 12),
              RepaintBoundary(child: _QuadTileCarousel(places: matchedPlaces)),
              const SizedBox(height: 24),
            ],
          ),
        ));
      }

      // 6. Top Rated Gems
      final topRatedPlaces = (List<Place>.from(filtered)..sort((a, b) => b.rating.compareTo(a.rating)))
          .where((p) => p.rating >= 4.0)
          .take(8)
          .toList();
      if (topRatedPlaces.length >= 2) {
        slivers.add(SliverToBoxAdapter(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const _SpotSectionHeader(
                title: 'Top Rated Gems',
                pill: '★ 4.0+',
                leadIcon: Icons.star_rounded,
              ),
              const SizedBox(height: 12),
              RepaintBoundary(child: _QuadTileCarousel(places: topRatedPlaces)),
              const SizedBox(height: 24),
            ],
          ),
        ));
      }

      // 7. Nomad & Wi-Fi Friendly
      final nomadPlaces = filtered.where((p) => p.wifi.score >= 8 || p.tags.contains('cowork')).take(8).toList();
      if (nomadPlaces.length >= 2) {
        slivers.add(SliverToBoxAdapter(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const _SpotSectionHeader(
                title: 'Nomad & Wi-Fi Friendly',
                pill: 'High speed',
                leadIcon: Icons.wifi_rounded,
              ),
              const SizedBox(height: 12),
              RepaintBoundary(child: _QuadTileCarousel(places: nomadPlaces)),
              const SizedBox(height: 24),
            ],
          ),
        ));
      }

      // 8. Remaining Spots: 120fps Virtualized Continuous SliverGrid
      final seenIds = {
        ...nearPlaces.map((p) => p.id),
        ...sponsoredPlaces.map((p) => p.id),
        ...dealPlaces.map((p) => p.id),
        ...matchedPlaces.map((p) => p.id),
        ...topRatedPlaces.map((p) => p.id),
        ...nomadPlaces.map((p) => p.id),
      };
      final remainingPlaces = filtered.where((p) => !seenIds.contains(p.id)).toList();
      final allToBrowse = remainingPlaces.isNotEmpty ? remainingPlaces : filtered;

      slivers.add(SliverToBoxAdapter(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _SpotSectionHeader(
              title: regionName != null ? 'All Spots in $regionName' : 'More Spots to Explore',
              pill: '${allToBrowse.length} spots',
            ),
            const SizedBox(height: 12),
          ],
        ),
      ));

      // Virtualized Lazy SliverGrid — only builds ~6 cards visible on screen
      slivers.add(SliverPadding(
        padding: const EdgeInsets.symmetric(horizontal: 16),
        sliver: SliverGrid(
          gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
            crossAxisCount: 2,
            mainAxisSpacing: 12,
            crossAxisSpacing: 12,
            childAspectRatio: 0.91,
          ),
          delegate: SliverChildBuilderDelegate(
            (context, index) => _SpotTileCard(place: allToBrowse[index]),
            childCount: allToBrowse.length,
          ),
        ),
      ));
    }

    // Bottom buffer for smooth tab-bar clearance
    slivers.add(const SliverToBoxAdapter(child: SizedBox(height: 90)));

    return SafeArea(
      child: RefreshIndicator(
        onRefresh: () => ref.read(placesProvider.notifier).refresh(),
        child: CustomScrollView(
          physics: const BouncingScrollPhysics(parent: AlwaysScrollableScrollPhysics()),
          slivers: slivers,
        ),
      ),
    );
  }
}

// ── Search & Filter Widgets ─────────────────────────────────────────────────

class _SearchField extends ConsumerWidget {
  const _SearchField({required this.controller, required this.onChanged});

  final TextEditingController controller;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = ref.watch(l10nProvider);
    return TextField(
      controller: controller,
      onChanged: onChanged,
      decoration: InputDecoration(
        hintText: l10n.searchPlaceholder,
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

// ── Multi-Section Headers & Carousels ───────────────────────────────────────

class _SpotSectionHeader extends ConsumerWidget {
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
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final l10n = ref.watch(l10nProvider);

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
    final colWidth = ((sw - 32 - 12) / 2).clamp(160.0, 220.0);
    final colCount = (places.length / 2).ceil();

    return SizedBox(
      height: 374,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        physics: const BouncingScrollPhysics(),
        padding: const EdgeInsets.symmetric(horizontal: 16),
        itemCount: colCount,
        separatorBuilder: (_, _) => const SizedBox(width: 12),
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

/// Circular category carousel featuring the 7 curated category avatars
/// with real-time spot counts, followed by "All Regions" at the last position.
class _CategoryCircleCarousel extends ConsumerWidget {
  const _CategoryCircleCarousel({
    required this.selectedCategory,
    required this.selectedRegionName,
    required this.places,
    required this.totalPlacesCount,
    this.userPrefs,
    required this.onSelectCategory,
    required this.onPickRegion,
  });

  final String? selectedCategory;
  final String? selectedRegionName;
  final List<Place> places;
  final int totalPlacesCount;
  final UserPrefs? userPrefs;
  final ValueChanged<String> onSelectCategory;
  final VoidCallback onPickRegion;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final l10n = ref.watch(l10nProvider);

    // Total items: 7 categories + 1 (Regions item at last position) = 8 items
    return SizedBox(
      height: 106,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        physics: const BouncingScrollPhysics(),
        padding: const EdgeInsets.symmetric(horizontal: 16),
        itemCount: _circleCategories.length + 1,
        separatorBuilder: (_, _) => const SizedBox(width: 16),
        itemBuilder: (context, i) {
          // ── Last Item: All Regions / Selected Region ──────────────────────
          if (i == _circleCategories.length) {
            final isRegionActive = selectedRegionName != null;
            return PressScale(
              onTap: onPickRegion,
              child: SizedBox(
                width: 72,
                child: Column(
                  children: [
                    AnimatedContainer(
                      duration: const Duration(milliseconds: 200),
                      width: 60,
                      height: 60,
                      decoration: BoxDecoration(
                        color: isRegionActive ? SpottoColors.accent : SpottoColors.accentTint,
                        shape: BoxShape.circle,
                        border: Border.all(
                          color: SpottoColors.accent,
                          width: isRegionActive ? 2.5 : 1.5,
                        ),
                        boxShadow: [
                          BoxShadow(
                            color: SpottoColors.accent.withValues(alpha: isRegionActive ? 0.3 : 0.08),
                            blurRadius: isRegionActive ? 8 : 4,
                            offset: const Offset(0, 2),
                          ),
                        ],
                      ),
                      child: Icon(
                        Icons.location_on_rounded,
                        size: 28,
                        color: isRegionActive ? Colors.white : SpottoColors.accent,
                      ),
                    ),
                    const SizedBox(height: 6),
                    Text(
                      selectedRegionName ?? l10n.allRegions,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      textAlign: TextAlign.center,
                      style: text.labelSmall?.copyWith(
                        fontWeight: isRegionActive ? FontWeight.w800 : FontWeight.w700,
                        fontSize: 11,
                        color: isRegionActive ? SpottoColors.accent : SpottoColors.ink,
                      ),
                    ),
                    const SizedBox(height: 1),
                    Text(
                      isRegionActive ? '${places.length} spots' : 'Filter region',
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
          }

          // ── Items 0..N-1: Curated Category Circles ─────────────────────────
          final cat = _circleCategories[i];
          final isSelected = selectedCategory == cat.key;
          final int count;
          if (cat.key == 'vibe') {
            count = (userPrefs != null && (userPrefs!.styles.isNotEmpty || userPrefs!.interests.isNotEmpty))
                ? places.where((p) => calculateVibeMatch(place: p, prefs: userPrefs).percentage >= 65).length
                : places.length;
          } else {
            count = places.where((p) => _matchesCategory(p, cat.key)).length;
          }

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

/// Compact Modern Spot Tile Card with support for Sponsored Ads, Deals, and Distance
class _SpotTileCard extends ConsumerWidget {
  const _SpotTileCard({
    required this.place,
    this.width,
  });

  final Place place;
  final double? width;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final userPrefs = ref.watch(prefsProvider);
    final hasPrefs = userPrefs != null && (userPrefs.styles.isNotEmpty || userPrefs.interests.isNotEmpty);
    final vibeMatch = hasPrefs ? calculateVibeMatch(place: place, prefs: userPrefs) : null;
    final showVibeBadge = vibeMatch != null && vibeMatch.percentage >= 70;

    return PressScale(
      onTap: () => showBusinessDrawer(context, place),
      child: Container(
        width: width,
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(
            color: place.isSponsored ? const Color(0xFFFFB300) : SpottoColors.line,
            width: place.isSponsored ? 1.5 : 1.2,
          ),
          boxShadow: [
            BoxShadow(
              color: place.isSponsored
                  ? const Color(0x33FFB300)
                  : SpottoColors.inkShadow.withValues(alpha: 0.05),
              blurRadius: place.isSponsored ? 10 : 8,
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

                    // Top-Left: Sponsored Badge OR Deal Badge OR Distance Pill
                    if (place.isSponsored)
                      Positioned(
                        left: 8,
                        top: 8,
                        child: Container(
                          padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 3.5),
                          decoration: BoxDecoration(
                            color: const Color(0xFF103D3A),
                            borderRadius: BorderRadius.circular(6),
                            border: Border.all(color: const Color(0xFFFFB300), width: 1.2),
                            boxShadow: const [
                              BoxShadow(color: Colors.black38, blurRadius: 4, offset: Offset(0, 1)),
                            ],
                          ),
                          child: const Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Icon(Icons.auto_awesome, size: 9, color: Color(0xFFFFB300)),
                              SizedBox(width: 3),
                              Text(
                                'SPONSORED',
                                style: TextStyle(
                                  color: Colors.white,
                                  fontWeight: FontWeight.w800,
                                  fontSize: 8.5,
                                  letterSpacing: 0.3,
                                ),
                              ),
                            ],
                          ),
                        ),
                      )
                    else if (place.deal.active)
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

                    // Bottom-Left: Vibe Suitability Badge
                    if (showVibeBadge)
                      Positioned(
                        left: 8,
                        bottom: 8,
                        child: Container(
                          padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 3),
                          decoration: BoxDecoration(
                            color: const Color(0xE60B4A54),
                            borderRadius: BorderRadius.circular(6),
                            border: Border.all(color: Colors.white.withValues(alpha: 0.25), width: 0.8),
                            boxShadow: const [
                              BoxShadow(color: Colors.black26, blurRadius: 4, offset: Offset(0, 1)),
                            ],
                          ),
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              const Icon(Icons.bolt_rounded, size: 11, color: Color(0xFF4EE0B5)),
                              const SizedBox(width: 2),
                              Text(
                                '${vibeMatch.percentage}% Vibe',
                                style: const TextStyle(
                                  color: Colors.white,
                                  fontWeight: FontWeight.w700,
                                  fontSize: 10,
                                  letterSpacing: 0.1,
                                ),
                              ),
                            ],
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
                      if (place.isSponsored) ...[
                        const Icon(Icons.verified_rounded, size: 12, color: Color(0xFFFFB300)),
                        const SizedBox(width: 3),
                        Expanded(
                          child: Text(
                            'Featured Partner',
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: text.labelSmall?.copyWith(
                              fontSize: 11,
                              color: SpottoColors.ink,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                        ),
                      ] else if (place.deal.active) ...[
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

class _LocationHintBanner extends ConsumerWidget {
  const _LocationHintBanner({required this.status, required this.onTap});

  final PlacesLocationStatus status;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final l10n = ref.watch(l10nProvider);
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
  final scored = <({Place place, int percentage})>[];
  for (final p in places) {
    final match = calculateVibeMatch(place: p, prefs: prefs);
    if (match.percentage >= 60) {
      scored.add((place: p, percentage: match.percentage));
    }
  }
  scored.sort((a, b) => b.percentage.compareTo(a.percentage));
  return scored.take(8).map((e) => e.place).toList();

}
