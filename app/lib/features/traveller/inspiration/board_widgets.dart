// Shared pieces of the Inspiration Board: place cards, the place map, the
// trip-day picker and the save / add-to-trip actions.

import 'dart:math' as math;

import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../core/api/api_client.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/models/inspiration.dart';
import '../../../data/models/models.dart' show Place;
import '../../../data/repositories/inspiration_repository.dart';
import '../../../data/repositories/repositories.dart';
import '../place_detail/business_drawer.dart';

/// A place shown on the board: from a reel, or saved for later.
class BoardPlaceRef {
  final String name;
  final String kind;
  final String subtitle;
  final double? lat;
  final double? lng;
  final GeoPlace? geoPlace;
  final String? placeId;
  const BoardPlaceRef({
    required this.name,
    required this.kind,
    this.subtitle = '',
    this.lat,
    this.lng,
    this.geoPlace,
    this.placeId,
  });

  factory BoardPlaceRef.fromBoardPlace(BoardPlace p) => BoardPlaceRef(
        name: p.geoPlace?.name ?? p.name,
        kind: p.kind,
        subtitle: p.geoPlace?.address.isNotEmpty == true ? p.geoPlace!.address : p.where,
        lat: p.geoPlace?.lat,
        lng: p.geoPlace?.lng,
        geoPlace: p.geoPlace,
        placeId: p.placeId,
      );

  factory BoardPlaceRef.fromSaved(SavedPlace s) => BoardPlaceRef(
        name: s.name,
        kind: s.kind,
        subtitle: s.geoPlace?.address ?? '',
        lat: s.lat,
        lng: s.lng,
        geoPlace: s.geoPlace,
        placeId: s.placeId,
      );
}

const kindColors = {
  'eat': Color(0xFFC2683F),
  'drink': Color(0xFF7C5CFF),
  'stay': SpottoColors.deepTeal,
  'see': SpottoColors.accent,
  'do': SpottoColors.accent2,
  'shop': Color(0xFFB2472F),
};

Color kindColor(String kind) => kindColors[kind] ?? SpottoColors.accent;

/// Opens a place: the Spotto drawer for catalog places, Google Maps otherwise.
Future<void> openBoardPlace(BuildContext context, WidgetRef ref, BoardPlaceRef p) async {
  if (p.placeId != null) {
    try {
      final Place place = ref.read(placesProvider).where((x) => x.id == p.placeId).firstOrNull ??
          await ref.read(placeDetailsProvider(p.placeId!).future);
      if (context.mounted) showBusinessDrawer(context, place);
      return;
    } catch (_) {
      // Fall through to Maps.
    }
  }
  final maps = p.geoPlace?.mapsUrl ?? '';
  final uri = maps.isNotEmpty
      ? Uri.parse(maps)
      : Uri.https('www.google.com', '/maps/search/', {'api': '1', 'query': p.name});
  await launchUrl(uri, mode: LaunchMode.externalApplication);
}

/// Asks which trip day to add to. Returns the day index, or null.
Future<int?> pickTripDay(BuildContext context, WidgetRef ref, String placeName) {
  return showModalBottomSheet<int>(
    context: context,
    backgroundColor: Colors.white,
    shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(24))),
    builder: (sheet) {
      final trip = ref.read(tripProvider);
      return SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(20, 18, 20, 12),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('Add to which day?', style: Theme.of(sheet).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w800)),
              const SizedBox(height: 2),
              Text(placeName, style: const TextStyle(color: SpottoColors.muted, fontSize: 13)),
              const SizedBox(height: 12),
              ConstrainedBox(
                constraints: BoxConstraints(maxHeight: MediaQuery.of(sheet).size.height * 0.45),
                child: ListView(
                  shrinkWrap: true,
                  children: [
                    for (var i = 0; i < trip.days.length; i++)
                      ListTile(
                        contentPadding: EdgeInsets.zero,
                        leading: CircleAvatar(
                          backgroundColor: SpottoColors.accentTint,
                          child: Text('${i + 1}', style: const TextStyle(fontWeight: FontWeight.w800, color: SpottoColors.accent)),
                        ),
                        title: Text('Day ${i + 1}', style: const TextStyle(fontWeight: FontWeight.w700)),
                        subtitle: Text(_stopCount(trip.days[i].stops.length)),
                        onTap: () => Navigator.pop(sheet, i),
                      ),
                    ListTile(
                      contentPadding: EdgeInsets.zero,
                      leading: const CircleAvatar(
                        backgroundColor: SpottoColors.sand,
                        child: Icon(Icons.add_rounded, color: SpottoColors.accent),
                      ),
                      title: const Text('New day', style: TextStyle(fontWeight: FontWeight.w700)),
                      onTap: () {
                        ref.read(tripProvider.notifier).addDay();
                        Navigator.pop(sheet, ref.read(tripProvider).days.length - 1);
                      },
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      );
    },
  );
}

String _stopCount(int n) => n == 0 ? 'No stops yet' : (n == 1 ? '1 stop' : '$n stops');

/// Day picker, then adds the place to the trip with its real location.
Future<void> addBoardPlaceToTrip(BuildContext context, WidgetRef ref, BoardPlaceRef p) async {
  final day = await pickTripDay(context, ref, p.name);
  if (day == null || !context.mounted) return;
  final notifier = ref.read(tripProvider.notifier);
  final catalog = p.placeId == null ? null : ref.read(placesProvider).where((x) => x.id == p.placeId).firstOrNull;
  if (catalog != null) {
    notifier.addPlaceStop(day, catalog);
  } else if (p.lat != null && p.lng != null) {
    notifier.addGeoStop(day, name: p.name, lat: p.lat!, lng: p.lng!, type: kindLabel(p.kind), icon: kindIcon(p.kind));
  } else {
    notifier.addCustomStop(day, p.name);
  }
  ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('Added ${p.name} to Day ${day + 1}.')));
}

/// Whether a place is in "save for later", and a toggle for it.
bool isBoardPlaceSaved(WidgetRef ref, BoardPlaceRef p) {
  final saved = ref.watch(savedPlacesProvider);
  if (p.geoPlace != null && saved.any((s) => s.geoPlaceId == p.geoPlace!.id)) return true;
  return p.placeId != null && saved.any((s) => s.placeId == p.placeId);
}

Future<void> toggleBoardPlaceSaved(BuildContext context, WidgetRef ref, BoardPlaceRef p, {String? sourceLinkId}) async {
  final notifier = ref.read(savedPlacesProvider.notifier);
  final existing = (p.geoPlace != null ? notifier.forGeoPlace(p.geoPlace!.id) : null) ??
      (p.placeId != null ? notifier.forCatalogPlace(p.placeId!) : null);
  try {
    if (existing != null) {
      await notifier.remove(existing.id);
    } else if (p.geoPlace != null) {
      await notifier.saveGeoPlace(p.geoPlace!.id, sourceLinkId: sourceLinkId);
    } else if (p.placeId != null) {
      await notifier.saveCatalogPlace(p.placeId!, sourceLinkId: sourceLinkId);
    }
  } catch (e) {
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
    }
  }
}

/// Place photo from the API's Google photo proxy, or a kind icon.
class BoardPlacePhoto extends StatelessWidget {
  final String? photoUrl;
  final String kind;
  final double size;
  const BoardPlacePhoto({super.key, this.photoUrl, required this.kind, this.size = 64});

  @override
  Widget build(BuildContext context) {
    final color = kindColor(kind);
    final fallback = Container(
      width: size,
      height: size,
      color: color.withValues(alpha: 0.12),
      child: Icon(spottoIcon(kindIcon(kind)), color: color, size: size * 0.38),
    );
    return ClipRRect(
      borderRadius: BorderRadius.circular(14),
      child: photoUrl == null
          ? fallback
          : CachedNetworkImage(
              imageUrl: resolveMediaUrl(photoUrl!),
              width: size,
              height: size,
              fit: BoxFit.cover,
              placeholder: (_, _) => fallback,
              errorWidget: (_, _, _) => fallback,
            ),
    );
  }
}

class KindChip extends StatelessWidget {
  final String kind;
  const KindChip(this.kind, {super.key});

  @override
  Widget build(BuildContext context) {
    final color = kindColor(kind);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(color: color.withValues(alpha: 0.12), borderRadius: BorderRadius.circular(8)),
      child: Text(kindLabel(kind), style: TextStyle(fontSize: 11, fontWeight: FontWeight.w800, color: color)),
    );
  }
}

/// A place card with save and add-to-trip actions. [number] matches the map pin.
class BoardPlaceCard extends ConsumerWidget {
  final BoardPlaceRef place;
  final int? number;
  final String? detail;
  final String? sourceLinkId;
  const BoardPlaceCard({super.key, required this.place, this.number, this.detail, this.sourceLinkId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final saved = isBoardPlaceSaved(ref, place);
    final rating = place.geoPlace?.rating;
    return PressScale(
      scale: 0.98,
      onTap: () => openBoardPlace(context, ref, place),
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(18),
          border: Border.all(color: SpottoColors.line),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Stack(
                  clipBehavior: Clip.none,
                  children: [
                    BoardPlacePhoto(photoUrl: place.geoPlace?.photoUrl, kind: place.kind),
                    if (number != null)
                      Positioned(
                        left: -6,
                        top: -6,
                        child: Container(
                          width: 22,
                          height: 22,
                          alignment: Alignment.center,
                          decoration: BoxDecoration(
                            color: SpottoColors.ink,
                            shape: BoxShape.circle,
                            border: Border.all(color: Colors.white, width: 2),
                          ),
                          child: Text('$number',
                              style: const TextStyle(fontSize: 10, fontWeight: FontWeight.w900, color: Colors.white)),
                        ),
                      ),
                  ],
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(place.name,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w800, color: SpottoColors.ink)),
                      const SizedBox(height: 4),
                      Wrap(
                        spacing: 6,
                        runSpacing: 4,
                        crossAxisAlignment: WrapCrossAlignment.center,
                        children: [
                          KindChip(place.kind),
                          if (rating != null)
                            Row(mainAxisSize: MainAxisSize.min, children: [
                              const Icon(Icons.star_rounded, size: 14, color: SpottoColors.sun),
                              Text(' ${rating.toStringAsFixed(1)}',
                                  style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w700)),
                              if (place.geoPlace!.ratingCount > 0)
                                Text(' (${place.geoPlace!.ratingCount})',
                                    style: const TextStyle(fontSize: 11, color: SpottoColors.muted)),
                            ]),
                        ],
                      ),
                      if (place.subtitle.isNotEmpty) ...[
                        const SizedBox(height: 4),
                        Text(place.subtitle,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(fontSize: 12, color: SpottoColors.muted)),
                      ],
                      if (detail != null) ...[
                        const SizedBox(height: 2),
                        Text(detail!, style: const TextStyle(fontSize: 11.5, color: SpottoColors.inkSoft)),
                      ],
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 10),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton.icon(
                    onPressed: () => toggleBoardPlaceSaved(context, ref, place, sourceLinkId: sourceLinkId),
                    icon: Icon(saved ? Icons.bookmark_rounded : Icons.bookmark_border_rounded, size: 18),
                    label: Text(saved ? 'Saved' : 'Save for later'),
                    style: OutlinedButton.styleFrom(
                      foregroundColor: SpottoColors.accent,
                      side: BorderSide(color: saved ? SpottoColors.accent : SpottoColors.line),
                      backgroundColor: saved ? SpottoColors.accentTint : null,
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: FilledButton.icon(
                    onPressed: () => addBoardPlaceToTrip(context, ref, place),
                    icon: const Icon(Icons.event_available_rounded, size: 18),
                    label: const Text('Add to trip'),
                    style: FilledButton.styleFrom(
                      backgroundColor: SpottoColors.sun,
                      foregroundColor: SpottoColors.ink,
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                    ),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

/// Map of numbered places, framed to fit them all.
class BoardPlacesMap extends StatefulWidget {
  final List<BoardPlaceRef> places;
  final double height;
  final void Function(int index)? onTapPlace;
  const BoardPlacesMap({super.key, required this.places, this.height = 220, this.onTapPlace});

  @override
  State<BoardPlacesMap> createState() => _BoardPlacesMapState();
}

class _BoardPlacesMapState extends State<BoardPlacesMap> {
  GoogleMapController? _map;

  List<(int, BoardPlaceRef)> get _located => [
        for (var i = 0; i < widget.places.length; i++)
          if (widget.places[i].lat != null && widget.places[i].lng != null) (i, widget.places[i]),
      ];

  void _fit() {
    final pts = _located.map((e) => LatLng(e.$2.lat!, e.$2.lng!)).toList();
    if (_map == null || pts.isEmpty) return;
    if (pts.length == 1) {
      _map!.moveCamera(CameraUpdate.newLatLngZoom(pts.first, 14));
      return;
    }
    var sw = pts.first, ne = pts.first;
    for (final p in pts) {
      sw = LatLng(math.min(sw.latitude, p.latitude), math.min(sw.longitude, p.longitude));
      ne = LatLng(math.max(ne.latitude, p.latitude), math.max(ne.longitude, p.longitude));
    }
    _map!.moveCamera(CameraUpdate.newLatLngBounds(LatLngBounds(southwest: sw, northeast: ne), 40));
  }

  @override
  void didUpdateWidget(BoardPlacesMap old) {
    super.didUpdateWidget(old);
    if (old.places.length != widget.places.length) WidgetsBinding.instance.addPostFrameCallback((_) => _fit());
  }

  @override
  Widget build(BuildContext context) {
    final located = _located;
    if (located.isEmpty) return const SizedBox.shrink();
    final first = located.first.$2;
    return ClipRRect(
      borderRadius: BorderRadius.circular(20),
      child: SizedBox(
        height: widget.height,
        child: GoogleMap(
          initialCameraPosition: CameraPosition(target: LatLng(first.lat!, first.lng!), zoom: 11),
          markers: {
            for (final (i, p) in located)
              Marker(
                markerId: MarkerId('$i'),
                position: LatLng(p.lat!, p.lng!),
                infoWindow: InfoWindow(title: '${i + 1}. ${p.name}', snippet: kindLabel(p.kind)),
                onTap: widget.onTapPlace == null ? null : () => widget.onTapPlace!(i),
              ),
          },
          zoomControlsEnabled: false,
          mapToolbarEnabled: false,
          myLocationButtonEnabled: false,
          compassEnabled: false,
          onMapCreated: (c) {
            _map = c;
            Future.delayed(const Duration(milliseconds: 250), _fit);
          },
        ),
      ),
    );
  }
}
