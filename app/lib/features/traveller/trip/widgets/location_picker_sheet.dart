// Location picker bottom sheet for Trip Planner days:
// allows choosing or searching Sri Lankan travel destinations and regions.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/theme/palette.dart';
import '../../../../core/widgets/press_scale.dart';
import '../../../../core/widgets/sheet_scaffold.dart';
import '../../../../data/repositories/repositories.dart';

typedef DestinationChoice = ({String name, String regionId, String tag});

const List<DestinationChoice> kPopularDestinations = [
  (name: 'Ella', regionId: 'ella', tag: 'Hill Country · Hiking & Vibe'),
  (name: 'Galle', regionId: 'galle', tag: 'South Coast · Fort & Heritage'),
  (name: 'Mirissa', regionId: 'mirissa', tag: 'South Coast · Beaches & Whales'),
  (name: 'Weligama', regionId: 'weligama', tag: 'South Coast · Surf & Cafés'),
  (name: 'Ahangama', regionId: 'ahangama', tag: 'South Coast · Surf & Nightlife'),
  (name: 'Hiriketiya', regionId: 'hiriketiya', tag: 'South Coast · Horseshoe Bay'),
  (name: 'Colombo', regionId: 'colombo', tag: 'City · Food, Arts & History'),
  (name: 'Kandy', regionId: 'kandy', tag: 'Hill Country · Lakes & Culture'),
  (name: 'Sigiriya', regionId: 'sigiriya', tag: 'Cultural Triangle · Ancient Citadel'),
  (name: 'Nuwara Eliya', regionId: 'nuwara-eliya', tag: 'Hill Country · Tea Estates'),
  (name: 'Arugam Bay', regionId: 'arugam-bay', tag: 'East Coast · World-class Waves'),
  (name: 'Trincomalee', regionId: 'trincomalee', tag: 'East Coast · Beaches & Diving'),
  (name: 'Dambulla', regionId: 'dambulla', tag: 'Cultural Triangle · Cave Temples'),
  (name: 'Negombo', regionId: 'negombo', tag: 'West Coast · Lagoons & Coastal'),
];

Future<DestinationChoice?> showLocationPickerSheet(
  BuildContext context, {
  required String? currentLocation,
}) {
  return showModalBottomSheet<DestinationChoice>(
    context: context,
    isScrollControlled: true,
    backgroundColor: Colors.transparent,
    builder: (_) => _LocationPickerSheet(currentLocation: currentLocation),
  );
}

class _LocationPickerSheet extends ConsumerStatefulWidget {
  final String? currentLocation;
  const _LocationPickerSheet({required this.currentLocation});

  @override
  ConsumerState<_LocationPickerSheet> createState() => _LocationPickerSheetState();
}

class _LocationPickerSheetState extends ConsumerState<_LocationPickerSheet> {
  final _searchController = TextEditingController();
  String _query = '';

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final places = ref.watch(placesProvider);

    // Collect dynamic regions from loaded places
    final dynamicMap = <String, String>{};
    for (final p in places) {
      final rid = p.regionId.trim();
      final rname = p.region.trim();
      if (rid.isNotEmpty && rname.isNotEmpty) {
        dynamicMap[rid] = rname;
      }
    }

    final allDestinations = <DestinationChoice>[...kPopularDestinations];
    for (final entry in dynamicMap.entries) {
      if (!allDestinations.any((d) => d.regionId.toLowerCase() == entry.key.toLowerCase())) {
        allDestinations.add((name: entry.value, regionId: entry.key, tag: 'Discovered Spot'));
      }
    }

    final query = _query.trim().toLowerCase();
    final filtered = query.isEmpty
        ? allDestinations
        : allDestinations
            .where((d) => d.name.toLowerCase().contains(query) || d.tag.toLowerCase().contains(query))
            .toList();

    return SheetScaffold(
      padding: const EdgeInsets.fromLTRB(20, 6, 20, 24),
      child: SizedBox(
        height: MediaQuery.of(context).size.height * 0.72,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('Travelling Location', style: text.headlineSmall?.copyWith(fontSize: 19)),
                      const SizedBox(height: 2),
                      Text(
                        'Set where you are staying or exploring on this day.',
                        style: text.bodySmall?.copyWith(color: SpottoColors.muted),
                      ),
                    ],
                  ),
                ),
                if (widget.currentLocation != null)
                  TextButton.icon(
                    onPressed: () => Navigator.of(context).pop((name: '', regionId: '', tag: '')),
                    icon: const Icon(Icons.clear_rounded, size: 14, color: Color(0xFFB3261E)),
                    label: const Text('Clear', style: TextStyle(color: Color(0xFFB3261E), fontSize: 12)),
                  ),
              ],
            ),
            const SizedBox(height: 14),
            TextField(
              controller: _searchController,
              autofocus: false,
              onChanged: (val) => setState(() => _query = val),
              decoration: InputDecoration(
                hintText: 'Search city or region (e.g. Ella, Mirissa)…',
                prefixIcon: const Icon(Icons.search, size: 18, color: SpottoColors.muted),
                suffixIcon: _query.isNotEmpty
                    ? IconButton(
                        icon: const Icon(Icons.close, size: 16, color: SpottoColors.muted),
                        onPressed: () {
                          _searchController.clear();
                          setState(() => _query = '');
                        },
                      )
                    : null,
              ),
            ),
            const SizedBox(height: 14),
            if (query.isEmpty) ...[
              Text(
                'POPULAR DESTINATIONS',
                style: text.labelSmall?.copyWith(
                  color: SpottoColors.muted,
                  letterSpacing: 0.8,
                  fontWeight: FontWeight.w700,
                ),
              ),
              const SizedBox(height: 8),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  for (final dest in kPopularDestinations.take(8)) ...[
                    _chip(dest),
                  ],
                ],
              ),
              const SizedBox(height: 14),
              Text(
                'ALL DESTINATIONS',
                style: text.labelSmall?.copyWith(
                  color: SpottoColors.muted,
                  letterSpacing: 0.8,
                  fontWeight: FontWeight.w700,
                ),
              ),
              const SizedBox(height: 8),
            ],
            Expanded(
              child: filtered.isEmpty
                  ? Center(
                      child: Column(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          const Icon(Icons.location_off_outlined, size: 36, color: SpottoColors.muted),
                          const SizedBox(height: 8),
                          Text('No towns found matching "$_query"',
                              style: text.bodyMedium?.copyWith(color: SpottoColors.muted)),
                        ],
                      ),
                    )
                  : ListView.separated(
                      itemCount: filtered.length,
                      separatorBuilder: (_, _) => const Divider(height: 1, color: SpottoColors.line),
                      itemBuilder: (context, index) {
                        final dest = filtered[index];
                        final isSelected = widget.currentLocation != null &&
                            (widget.currentLocation!.toLowerCase() == dest.name.toLowerCase() ||
                                widget.currentLocation!.toLowerCase() == dest.regionId.toLowerCase());

                        return ListTile(
                          contentPadding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
                          leading: Container(
                            width: 36,
                            height: 36,
                            decoration: BoxDecoration(
                              color: isSelected ? SpottoColors.accent : SpottoColors.accentTint,
                              borderRadius: BorderRadius.circular(10),
                            ),
                            child: Icon(
                              Icons.location_on_rounded,
                              size: 18,
                              color: isSelected ? Colors.white : SpottoColors.accent,
                            ),
                          ),
                          title: Text(
                            dest.name,
                            style: TextStyle(
                              fontSize: 14.5,
                              fontWeight: isSelected ? FontWeight.w800 : FontWeight.w600,
                              color: isSelected ? SpottoColors.accent : SpottoColors.ink,
                            ),
                          ),
                          subtitle: Text(
                            dest.tag,
                            style: text.bodySmall?.copyWith(color: SpottoColors.muted),
                          ),
                          trailing: isSelected
                              ? const Icon(Icons.check_circle_rounded, color: SpottoColors.accent, size: 20)
                              : const Icon(Icons.chevron_right_rounded, color: SpottoColors.muted, size: 18),
                          onTap: () => Navigator.of(context).pop(dest),
                        );
                      },
                    ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _chip(DestinationChoice dest) {
    final isSelected = widget.currentLocation != null &&
        (widget.currentLocation!.toLowerCase() == dest.name.toLowerCase() ||
            widget.currentLocation!.toLowerCase() == dest.regionId.toLowerCase());

    return PressScale(
      scale: 0.94,
      onTap: () => Navigator.of(context).pop(dest),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        decoration: BoxDecoration(
          color: isSelected ? SpottoColors.accent : Colors.white,
          borderRadius: BorderRadius.circular(20),
          border: Border.all(
            color: isSelected ? SpottoColors.accent : SpottoColors.line,
            width: 1.2,
          ),
          boxShadow: isSelected
              ? [
                  BoxShadow(
                    color: SpottoColors.accent.withValues(alpha: 0.2),
                    blurRadius: 6,
                    offset: const Offset(0, 2),
                  )
                ]
              : null,
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.place_rounded,
              size: 13,
              color: isSelected ? Colors.white : SpottoColors.accent,
            ),
            const SizedBox(width: 4),
            Text(
              dest.name,
              style: TextStyle(
                fontSize: 12,
                fontWeight: isSelected ? FontWeight.w800 : FontWeight.w600,
                color: isSelected ? Colors.white : SpottoColors.ink,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
