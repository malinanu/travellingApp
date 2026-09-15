// Social-inspiration import sheet — lets travellers paste Instagram, TikTok,
// YouTube, or any website link into their trip for reference while planning.

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../../core/theme/palette.dart';
import '../../../../core/widgets/press_scale.dart';
import '../../../../core/widgets/spotto_icons.dart';
import '../../../../data/mock/trip_catalog.dart';
import '../../../../data/repositories/repositories.dart';

/// Opens the inspiration board as a modal bottom sheet.
Future<void> showInspirationSheet(BuildContext context) {
  return showModalBottomSheet(
    context: context,
    isScrollControlled: true,
    backgroundColor: Colors.transparent,
    builder: (_) => const _InspirationSheet(),
  );
}

class _InspirationSheet extends ConsumerStatefulWidget {
  const _InspirationSheet();

  @override
  ConsumerState<_InspirationSheet> createState() => _InspirationSheetState();
}

class _InspirationSheetState extends ConsumerState<_InspirationSheet> {
  final _urlCtrl = TextEditingController();
  final _noteCtrl = TextEditingController();
  bool _adding = false;
  String? _err;

  @override
  void dispose() {
    _urlCtrl.dispose();
    _noteCtrl.dispose();
    super.dispose();
  }

  void _add() {
    final raw = _urlCtrl.text.trim();
    if (raw.isEmpty) return;
    // Prepend https:// if the user forgot
    final url = raw.startsWith('http://') || raw.startsWith('https://') ? raw : 'https://$raw';
    final uri = Uri.tryParse(url);
    if (uri == null || !uri.hasScheme) {
      setState(() => _err = 'Invalid URL — please check and try again.');
      return;
    }
    final link = InspirationLink(
      id: 'insp_${DateTime.now().millisecondsSinceEpoch}',
      url: url,
      title: InspirationLink.titleOf(url),
      platform: InspirationLink.platformOf(url),
      note: _noteCtrl.text.trim(),
      addedAt: DateTime.now().toIso8601String(),
    );
    ref.read(tripProvider.notifier).addInspiration(link);
    _urlCtrl.clear();
    _noteCtrl.clear();
    setState(() {
      _adding = false;
      _err = null;
    });
  }

  @override
  Widget build(BuildContext context) {
    final trip = ref.watch(tripProvider);
    final links = trip.inspirations;
    final text = Theme.of(context).textTheme;

    return DraggableScrollableSheet(
      initialChildSize: 0.7,
      minChildSize: 0.4,
      maxChildSize: 0.92,
      expand: false,
      builder: (_, scrollCtrl) => Container(
        decoration: const BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
        ),
        child: Column(
          children: [
            // Handle
            Padding(
              padding: const EdgeInsets.only(top: 10),
              child: Container(
                width: 40,
                height: 4,
                decoration: BoxDecoration(
                  color: SpottoColors.muted.withValues(alpha: 0.3),
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
            ),
            // Header
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 14, 12, 0),
              child: Row(
                children: [
                  Container(
                    width: 38,
                    height: 38,
                    decoration: const BoxDecoration(
                      color: SpottoColors.accentTint,
                      shape: BoxShape.circle,
                    ),
                    child: const Icon(Icons.bookmark_add_rounded, size: 20, color: SpottoColors.accent),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text('Inspiration Board', style: text.titleMedium?.copyWith(fontWeight: FontWeight.w800)),
                        Text('Save reels, posts & sites into your trip',
                            style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                      ],
                    ),
                  ),
                  IconButton(
                    onPressed: () => Navigator.pop(context),
                    icon: const Icon(Icons.close, size: 20),
                    color: SpottoColors.muted,
                  ),
                ],
              ),
            ),
            const SizedBox(height: 12),
            // Add-link panel
            AnimatedSize(
              duration: const Duration(milliseconds: 240),
              curve: Curves.easeInOut,
              child: _adding ? _AddLinkPanel(
                urlCtrl: _urlCtrl,
                noteCtrl: _noteCtrl,
                err: _err,
                onSave: _add,
                onCancel: () => setState(() { _adding = false; _err = null; }),
              ) : Padding(
                padding: const EdgeInsets.symmetric(horizontal: 20),
                child: PressScale(
                  scale: 0.97,
                  onTap: () => setState(() => _adding = true),
                  child: Container(
                    padding: const EdgeInsets.symmetric(vertical: 13),
                    decoration: BoxDecoration(
                      borderRadius: BorderRadius.circular(18),
                      border: Border.all(color: SpottoColors.accent, width: 2),
                    ),
                    child: const Row(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Icon(Icons.add_link_rounded, size: 18, color: SpottoColors.accent),
                        SizedBox(width: 8),
                        Text('Add a link',
                            style: TextStyle(
                                fontSize: 14,
                                fontWeight: FontWeight.w800,
                                color: SpottoColors.accent)),
                      ],
                    ),
                  ),
                ),
              ),
            ),
            const SizedBox(height: 8),
            const Divider(height: 1, indent: 20, endIndent: 20),
            // Links list
            Expanded(
              child: links.isEmpty
                  ? Center(
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Container(
                            width: 60,
                            height: 60,
                            decoration: const BoxDecoration(
                              color: SpottoColors.accentTint,
                              shape: BoxShape.circle,
                            ),
                            child: const Icon(Icons.bookmark_border_rounded,
                                size: 28, color: SpottoColors.accent),
                          ),
                          const SizedBox(height: 14),
                          Text('No inspiration saved yet',
                              style: text.titleMedium?.copyWith(fontSize: 16)),
                          const SizedBox(height: 5),
                          Text('Paste any Instagram, TikTok, YouTube\nor website link to save it here.',
                              textAlign: TextAlign.center,
                              style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                        ],
                      ),
                    )
                  : ListView.separated(
                      controller: scrollCtrl,
                      padding: const EdgeInsets.fromLTRB(20, 12, 20, 32),
                      itemCount: links.length,
                      separatorBuilder: (_, _) => const SizedBox(height: 10),
                      itemBuilder: (_, i) => _InspirationCard(
                        link: links[i],
                        onDelete: () =>
                            ref.read(tripProvider.notifier).removeInspiration(links[i].id),
                        onEditNote: (note) =>
                            ref.read(tripProvider.notifier).editInspirationNote(links[i].id, note),
                      ),
                    ),
            ),
          ],
        ),
      ),
    );
  }
}

// ─────── add link panel ───────────────────────────────────────────────────────

class _AddLinkPanel extends StatelessWidget {
  final TextEditingController urlCtrl;
  final TextEditingController noteCtrl;
  final String? err;
  final VoidCallback onSave;
  final VoidCallback onCancel;
  const _AddLinkPanel({
    required this.urlCtrl,
    required this.noteCtrl,
    required this.err,
    required this.onSave,
    required this.onCancel,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      margin: const EdgeInsets.symmetric(horizontal: 20),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: SpottoColors.accentTint.withValues(alpha: 0.35),
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: SpottoColors.accent.withValues(alpha: 0.25)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.link_rounded, size: 15, color: SpottoColors.accent),
              const SizedBox(width: 6),
              Text('Paste a link',
                  style: const TextStyle(
                      fontSize: 13, fontWeight: FontWeight.w800, color: SpottoColors.accent)),
              const Spacer(),
              GestureDetector(
                onTap: onCancel,
                child: const Icon(Icons.close, size: 16, color: SpottoColors.muted),
              ),
            ],
          ),
          const SizedBox(height: 10),
          TextField(
            controller: urlCtrl,
            autofocus: true,
            keyboardType: TextInputType.url,
            style: const TextStyle(fontSize: 13.5),
            decoration: InputDecoration(
              hintText: 'https://www.instagram.com/reel/...',
              hintStyle: TextStyle(color: SpottoColors.muted.withValues(alpha: 0.7), fontSize: 13),
              filled: true,
              fillColor: Colors.white,
              contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(12),
                borderSide: BorderSide.none,
              ),
              suffixIcon: IconButton(
                icon: const Icon(Icons.content_paste_rounded, size: 16, color: SpottoColors.muted),
                tooltip: 'Paste from clipboard',
                onPressed: () async {
                  final data = await Clipboard.getData(Clipboard.kTextPlain);
                  if (data?.text != null) urlCtrl.text = data!.text!;
                },
              ),
            ),
            onSubmitted: (_) => onSave(),
          ),
          if (err != null) ...[
            const SizedBox(height: 6),
            Text(err!,
                style: const TextStyle(fontSize: 11.5, color: Colors.red, fontWeight: FontWeight.w600)),
          ],
          const SizedBox(height: 8),
          TextField(
            controller: noteCtrl,
            style: const TextStyle(fontSize: 13.5),
            decoration: InputDecoration(
              hintText: 'Note (optional)',
              hintStyle: TextStyle(color: SpottoColors.muted.withValues(alpha: 0.7), fontSize: 13),
              filled: true,
              fillColor: Colors.white,
              contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(12),
                borderSide: BorderSide.none,
              ),
            ),
          ),
          const SizedBox(height: 12),
          SizedBox(
            width: double.infinity,
            child: FilledButton.icon(
              onPressed: onSave,
              icon: const Icon(Icons.bookmark_add_rounded, size: 16),
              label: const Text('Save to trip'),
              style: FilledButton.styleFrom(
                backgroundColor: SpottoColors.accent,
                padding: const EdgeInsets.symmetric(vertical: 12),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

// ─────── single inspiration card ─────────────────────────────────────────────

class _InspirationCard extends ConsumerWidget {
  final InspirationLink link;
  final VoidCallback onDelete;
  final void Function(String note) onEditNote;
  const _InspirationCard({required this.link, required this.onDelete, required this.onEditNote});

  IconData _platformIcon() {
    switch (link.platform) {
      case 'instagram':
        return Icons.camera_alt_rounded;
      case 'tiktok':
        return Icons.music_video_rounded;
      case 'youtube':
        return Icons.play_circle_rounded;
      default:
        return Icons.language_rounded;
    }
  }

  Color _platformColor() {
    switch (link.platform) {
      case 'instagram':
        return const Color(0xFFE1306C);
      case 'tiktok':
        return const Color(0xFF010101);
      case 'youtube':
        return const Color(0xFFFF0000);
      default:
        return SpottoColors.accent;
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final color = _platformColor();

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(16),
        boxShadow: const [
          BoxShadow(color: Color(0x12103D3A), blurRadius: 10, offset: Offset(0, 2)),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 34,
                height: 34,
                decoration: BoxDecoration(
                  color: color.withValues(alpha: 0.12),
                  shape: BoxShape.circle,
                ),
                child: Icon(_platformIcon(), size: 17, color: color),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(link.title,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                            fontSize: 13.5, fontWeight: FontWeight.w700)),
                    Text(link.url,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: text.bodySmall?.copyWith(color: SpottoColors.muted, fontSize: 11)),
                  ],
                ),
              ),
              const SizedBox(width: 6),
              PressScale(
                scale: 0.9,
                onTap: () => launchUrl(Uri.parse(link.url), mode: LaunchMode.externalApplication),
                child: Container(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
                  decoration: BoxDecoration(
                    color: color.withValues(alpha: 0.1),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Text('Open',
                      style: TextStyle(fontSize: 12, fontWeight: FontWeight.w800, color: color)),
                ),
              ),
              const SizedBox(width: 6),
              GestureDetector(
                onTap: onDelete,
                child: const Icon(Icons.delete_outline_rounded, size: 18, color: SpottoColors.muted),
              ),
            ],
          ),
          if (link.platform == 'instagram') ...[
            const SizedBox(height: 8),
            _ExtractPlacesButton(url: link.url),
          ],
          if (link.note.isNotEmpty) ...[
            const SizedBox(height: 8),
            GestureDetector(
              onTap: () async {
                final ctrl = TextEditingController(text: link.note);
                final result = await showDialog<String>(
                  context: context,
                  builder: (_) => AlertDialog(
                    title: const Text('Edit note', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w800)),
                    content: TextField(
                      controller: ctrl,
                      decoration: const InputDecoration(hintText: 'Add a note...'),
                      autofocus: true,
                    ),
                    actions: [
                      TextButton(onPressed: () => Navigator.pop(context), child: const Text('Cancel')),
                      FilledButton(
                        onPressed: () => Navigator.pop(context, ctrl.text.trim()),
                        child: const Text('Save'),
                      ),
                    ],
                  ),
                );
                if (result != null) onEditNote(result);
              },
              child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                decoration: BoxDecoration(
                  color: const Color(0xFFF5F5F5),
                  borderRadius: BorderRadius.circular(10),
                ),
                child: Row(
                  children: [
                    const Icon(Icons.notes_rounded, size: 13, color: SpottoColors.muted),
                    const SizedBox(width: 5),
                    Expanded(
                      child: Text(link.note,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(fontSize: 12, color: SpottoColors.muted)),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

// ─────── extract places ───────────────────────────────────────────────────

class _ExtractPlacesButton extends ConsumerStatefulWidget {
  final String url;
  const _ExtractPlacesButton({required this.url});

  @override
  ConsumerState<_ExtractPlacesButton> createState() => _ExtractPlacesButtonState();
}

class _ExtractPlacesButtonState extends ConsumerState<_ExtractPlacesButton> {
  bool _loading = false;

  Future<void> _extract() async {
    setState(() => _loading = true);
    try {
      final places = await extractPlacesFromLink(ref, widget.url);
      if (!mounted) return;
      if (places.isEmpty) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text("Couldn't find any places mentioned in that link.")),
        );
      } else {
        showModalBottomSheet(
          context: context,
          isScrollControlled: true,
          backgroundColor: Colors.transparent,
          builder: (_) => _ExtractedPlacesSheet(places: places),
        );
      }
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text("Couldn't read that link — it may be private or unavailable.")),
      );
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return PressScale(
      scale: 0.97,
      onTap: _loading ? null : _extract,
      child: Container(
        padding: const EdgeInsets.symmetric(vertical: 9),
        decoration: BoxDecoration(
          color: SpottoColors.accentTint,
          borderRadius: BorderRadius.circular(12),
        ),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            if (_loading)
              const SizedBox(
                width: 13,
                height: 13,
                child: CircularProgressIndicator(strokeWidth: 2, color: SpottoColors.accent),
              )
            else
              const Icon(Icons.auto_awesome_rounded, size: 14, color: SpottoColors.accent),
            const SizedBox(width: 6),
            Text(_loading ? 'Reading link…' : 'Extract places',
                style: const TextStyle(fontSize: 12.5, fontWeight: FontWeight.w800, color: SpottoColors.accent)),
          ],
        ),
      ),
    );
  }
}

class _ExtractedPlacesSheet extends ConsumerWidget {
  final List<ExtractedPlace> places;
  const _ExtractedPlacesSheet({required this.places});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    return DraggableScrollableSheet(
      initialChildSize: 0.55,
      minChildSize: 0.3,
      maxChildSize: 0.85,
      expand: false,
      builder: (_, scrollCtrl) => Container(
        decoration: const BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
        ),
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.only(top: 10),
              child: Container(
                width: 40,
                height: 4,
                decoration: BoxDecoration(
                  color: SpottoColors.muted.withValues(alpha: 0.3),
                  borderRadius: BorderRadius.circular(2),
                ),
              ),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 14, 20, 4),
              child: Align(
                alignment: Alignment.centerLeft,
                child: Text('Places found in this link',
                    style: text.titleMedium?.copyWith(fontWeight: FontWeight.w800)),
              ),
            ),
            Expanded(
              child: ListView.separated(
                controller: scrollCtrl,
                padding: const EdgeInsets.fromLTRB(20, 8, 20, 32),
                itemCount: places.length,
                separatorBuilder: (_, _) => const SizedBox(height: 10),
                itemBuilder: (_, i) => _ExtractedPlaceRow(place: places[i]),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _ExtractedPlaceRow extends ConsumerWidget {
  final ExtractedPlace place;
  const _ExtractedPlaceRow({required this.place});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final favorites = place.matched ? ref.watch(favoritesProvider) : const <String>{};
    final isFavorite = place.placeId != null && favorites.contains(place.placeId);

    void addToTrip() {
      final trip = ref.read(tripProvider);
      if (trip.days.isEmpty) return;
      final catalogPlace = place.matched
          ? ref.read(placesProvider).where((p) => p.id == place.placeId).firstOrNull
          : null;
      if (catalogPlace != null) {
        ref.read(tripProvider.notifier).addPlaceStop(0, catalogPlace);
      } else {
        ref.read(tripProvider.notifier).addCustomStop(0, place.name);
      }
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Added "${place.name}" to Day 1 — move it to another day from the Trip tab.')),
      );
    }

    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: SpottoColors.line),
      ),
      child: Row(
        children: [
          Container(
            width: 38,
            height: 38,
            decoration: const BoxDecoration(
              color: SpottoColors.accentTint,
              shape: BoxShape.circle,
            ),
            child: Icon(
              place.category != null ? spottoIcon(place.category!) : Icons.place_outlined,
              size: 18,
              color: SpottoColors.accent,
            ),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(place.name,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(fontSize: 13.5, fontWeight: FontWeight.w700)),
                Text(place.matched ? 'In Spotto' : "Not in Spotto's catalog yet",
                    style: const TextStyle(fontSize: 11, color: SpottoColors.muted)),
              ],
            ),
          ),
          if (place.matched)
            PressScale(
              scale: 0.9,
              onTap: () => ref.read(favoritesProvider.notifier).toggle(place.placeId!),
              child: Icon(isFavorite ? Icons.favorite_rounded : Icons.favorite_border_rounded,
                  size: 20, color: isFavorite ? Colors.pinkAccent : SpottoColors.muted),
            ),
          const SizedBox(width: 8),
          PressScale(
            scale: 0.92,
            onTap: addToTrip,
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 7),
              decoration: BoxDecoration(
                color: SpottoColors.accent,
                borderRadius: BorderRadius.circular(10),
              ),
              child: const Text('Add to trip',
                  style: TextStyle(fontSize: 11.5, fontWeight: FontWeight.w800, color: Colors.white)),
            ),
          ),
        ],
      ),
    );
  }
}
