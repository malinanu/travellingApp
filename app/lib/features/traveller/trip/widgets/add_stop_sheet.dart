// Add-stop bottom sheet: search local catalog, Google Places search, style suggestions, custom stop.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/api/google_places_service.dart';
import '../../../../core/theme/palette.dart';
import '../../../../core/widgets/press_scale.dart';
import '../../../../core/widgets/sheet_scaffold.dart';
import '../../../../core/widgets/spotto_icons.dart';
import '../../../../data/catalog/trip_catalog.dart';
import '../../../../data/repositories/repositories.dart';

Future<void> showAddStopSheet(BuildContext context, WidgetRef ref, int dayIx) {
  return showModalBottomSheet(
    context: context,
    isScrollControlled: true,
    builder: (_) => _AddStopSheet(dayIx: dayIx),
  );
}

class _AddStopSheet extends ConsumerStatefulWidget {
  final int dayIx;
  const _AddStopSheet({required this.dayIx});

  @override
  ConsumerState<_AddStopSheet> createState() => _AddStopSheetState();
}

class _AddStopSheetState extends ConsumerState<_AddStopSheet> {
  final _controller = TextEditingController();
  String _query = '';
  List<GooglePlacesSearchResult> _googleResults = [];
  bool _isSearchingGoogle = false;

  void _onQueryChanged(String val) async {
    setState(() => _query = val);
    final query = val.trim();
    if (query.length >= 2) {
      setState(() => _isSearchingGoogle = true);
      final placesService = ref.read(googlePlacesServiceProvider);
      final results = await placesService.search(query);
      if (mounted && _query.trim() == query) {
        setState(() {
          _googleResults = results;
          _isSearchingGoogle = false;
        });
      }
    } else {
      setState(() {
        _googleResults = [];
        _isSearchingGoogle = false;
      });
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final trip = ref.watch(tripProvider);
    final styleIds = ref.watch(prefsProvider)?.styles ?? const [];
    final usedIds = trip.days[widget.dayIx].stops.map((s) => s.placeId).toSet();

    final query = _query.trim().toLowerCase();
    final results = query.isEmpty
        ? <TripCatalogItem>[]
        : tripCatalog
            .where((p) => '${p.name} ${p.type}'.toLowerCase().contains(query))
            .toList();
    final suggested = tripCatalog
        .where((p) => p.styles.any(styleIds.contains) && !usedIds.contains(p.id))
        .take(5)
        .toList();
    final catalogList = query.isNotEmpty ? results : suggested;

    return SheetScaffold(
      padding: const EdgeInsets.fromLTRB(20, 4, 20, 24),
      child: SizedBox(
        height: MediaQuery.of(context).size.height * 0.75,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Add a stop', style: text.headlineSmall?.copyWith(fontSize: 19)),
            const SizedBox(height: 12),
            TextField(
              controller: _controller,
              autofocus: true,
              onChanged: _onQueryChanged,
              decoration: InputDecoration(
                hintText: 'Search places on Google Maps or Spotto…',
                prefixIcon: const Icon(Icons.search, size: 18, color: SpottoColors.muted),
                suffixIcon: _query.isNotEmpty
                    ? IconButton(
                        icon: const Icon(Icons.close, size: 16, color: SpottoColors.muted),
                        onPressed: () {
                          _controller.clear();
                          setState(() {
                            _query = '';
                            _googleResults = [];
                          });
                        },
                      )
                    : null,
              ),
            ),
            const SizedBox(height: 12),
            if (query.isEmpty && suggested.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(bottom: 8, left: 4),
                child: Row(
                  children: [
                    const Icon(Icons.auto_awesome, size: 13, color: SpottoColors.accent),
                    const SizedBox(width: 5),
                    Text('SUGGESTED FOR YOUR STYLE',
                        style: text.labelSmall
                            ?.copyWith(color: SpottoColors.muted, letterSpacing: 0.6)),
                  ],
                ),
              ),
            Expanded(
              child: ListView(
                children: [
                  // Spotto Catalog Places
                  for (final p in catalogList) _catalogRow(p, usedIds.contains(p.id)),

                  // Google Maps Search Results Section
                  if (query.isNotEmpty) ...[
                    if (_googleResults.isNotEmpty) ...[
                      const Padding(
                        padding: EdgeInsets.fromLTRB(4, 14, 4, 8),
                        child: Row(
                          children: [
                            Icon(Icons.place_rounded, size: 14, color: SpottoColors.accent),
                            SizedBox(width: 5),
                            Text('GOOGLE MAPS PLACES',
                                style: TextStyle(
                                    fontSize: 10.5,
                                    fontWeight: FontWeight.bold,
                                    color: SpottoColors.accent,
                                    letterSpacing: 0.7)),
                          ],
                        ),
                      ),
                      for (final gres in _googleResults) _googlePlaceRow(gres),
                    ],
                    if (_isSearchingGoogle)
                      const Padding(
                        padding: EdgeInsets.all(16),
                        child: Center(
                          child: SizedBox(
                            width: 18,
                            height: 18,
                            child: CircularProgressIndicator(strokeWidth: 2.5),
                          ),
                        ),
                      ),
                    _customRow(),
                  ],

                  if (query.isNotEmpty && catalogList.isEmpty && _googleResults.isEmpty && !_isSearchingGoogle)
                    Padding(
                      padding: const EdgeInsets.only(top: 8),
                      child: Text('No exact matches — add it as a custom stop above.',
                          textAlign: TextAlign.center,
                          style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                    ),
                  if (query.isEmpty && suggested.isEmpty)
                    Padding(
                      padding: const EdgeInsets.only(top: 8),
                      child: Text('Type to search any Google Maps place, hotel, café, beach…',
                          textAlign: TextAlign.center,
                          style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _catalogRow(TripCatalogItem p, bool used) {
    final text = Theme.of(context).textTheme;
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Opacity(
        opacity: used ? 0.45 : 1,
        child: PressScale(
          onTap: used
              ? null
              : () {
                  ref.read(tripProvider.notifier).addCatalogStop(widget.dayIx, p);
                  Navigator.of(context).pop();
                },
          child: Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Colors.white,
              borderRadius: BorderRadius.circular(16),
              border: Border.all(color: SpottoColors.line, width: 1.5),
            ),
            child: Row(
              children: [
                Container(
                  width: 40,
                  height: 40,
                  decoration: BoxDecoration(
                    color: SpottoColors.accentTint,
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Icon(spottoIcon(p.icon), size: 19, color: SpottoColors.accent),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(p.name,
                          maxLines: 1, overflow: TextOverflow.ellipsis, style: text.titleSmall),
                      Text('${p.type} · ${tripFmtRs(p.cost)}',
                          style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                    ],
                  ),
                ),
                used
                    ? Text('Added',
                        style: text.labelSmall?.copyWith(color: SpottoColors.muted))
                    : const Icon(Icons.add, size: 18, color: SpottoColors.accent),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _googlePlaceRow(GooglePlacesSearchResult gres) {
    final text = Theme.of(context).textTheme;
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: PressScale(
        onTap: () {
          ref.read(tripProvider.notifier).addPlaceStop(widget.dayIx, gres.toPlace());
          Navigator.of(context).pop();
        },
        child: Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: SpottoColors.accent.withValues(alpha: 0.3), width: 1.5),
          ),
          child: Row(
            children: [
              Container(
                width: 40,
                height: 40,
                decoration: BoxDecoration(
                  color: SpottoColors.accent.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: const Icon(Icons.place_rounded, size: 20, color: SpottoColors.accent),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Expanded(
                          child: Text(gres.name,
                              maxLines: 1, overflow: TextOverflow.ellipsis, style: text.titleSmall),
                        ),
                        Container(
                          padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                          decoration: BoxDecoration(
                            color: SpottoColors.accent.withValues(alpha: 0.1),
                            borderRadius: BorderRadius.circular(6),
                          ),
                          child: const Text('Google',
                              style: TextStyle(fontSize: 9.5, color: SpottoColors.accent, fontWeight: FontWeight.bold)),
                        ),
                      ],
                    ),
                    const SizedBox(height: 2),
                    Text(gres.address,
                        maxLines: 1, overflow: TextOverflow.ellipsis, style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                  ],
                ),
              ),
              const SizedBox(width: 8),
              const Icon(Icons.add_circle_outline, size: 20, color: SpottoColors.accent),
            ],
          ),
        ),
      ),
    );
  }

  Widget _customRow() {
    final text = Theme.of(context).textTheme;
    final name = _query.trim();
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: PressScale(
        onTap: () {
          ref.read(tripProvider.notifier).addCustomStop(widget.dayIx, name);
          Navigator.of(context).pop();
        },
        child: Container(
          padding: const EdgeInsets.all(12),
          decoration: BoxDecoration(
            color: SpottoColors.accentTint,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: SpottoColors.accent, width: 1.5),
          ),
          child: Row(
            children: [
              Container(
                width: 40,
                height: 40,
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: const Icon(Icons.edit_location_alt, size: 19, color: SpottoColors.accent),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('Add “$name”',
                        maxLines: 1, overflow: TextOverflow.ellipsis, style: text.titleSmall),
                    Text('Custom stop',
                        style: text.bodySmall?.copyWith(color: SpottoColors.inkSoft)),
                  ],
                ),
              ),
              const Icon(Icons.add, size: 18, color: SpottoColors.accent),
            ],
          ),
        ),
      ),
    );
  }
}
