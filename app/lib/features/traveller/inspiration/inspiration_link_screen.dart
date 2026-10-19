// One saved reel: where its places are on a map, a card per place (save for
// later / add to trip), and any names the reel mentioned that couldn't be
// pinned on a map.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../core/api/api_client.dart';
import '../../../core/theme/palette.dart';
import '../../../data/models/inspiration.dart';
import '../../../data/repositories/inspiration_repository.dart';
import 'board_widgets.dart';
import 'inspiration_board_screen.dart' show platformIcon;

class InspirationLinkScreen extends ConsumerStatefulWidget {
  final String linkId;
  const InspirationLinkScreen({super.key, required this.linkId});

  @override
  ConsumerState<InspirationLinkScreen> createState() => _InspirationLinkScreenState();
}

class _InspirationLinkScreenState extends ConsumerState<InspirationLinkScreen> {
  final _scroll = ScrollController();
  final _cardKeys = <int, GlobalKey>{};
  bool _savingAll = false;

  @override
  void initState() {
    super.initState();
    // Opened from a notification, the link may not be in the list yet.
    WidgetsBinding.instance.addPostFrameCallback((_) => ref.read(boardLinksProvider.notifier).fetch(widget.linkId));
  }

  @override
  void dispose() {
    _scroll.dispose();
    super.dispose();
  }

  Future<void> _saveAll(BoardLink link) async {
    setState(() => _savingAll = true);
    final notifier = ref.read(savedPlacesProvider.notifier);
    var added = 0;
    try {
      for (final p in link.confirmedPlaces) {
        if (notifier.forGeoPlace(p.geoPlace!.id) != null) continue;
        await notifier.saveGeoPlace(p.geoPlace!.id, sourceLinkId: link.id);
        added++;
      }
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
          content: Text(added == 0 ? 'All of these are already saved.' : 'Saved $added ${added == 1 ? 'place' : 'places'} for later.'),
        ));
      }
    } catch (e) {
      if (mounted) ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
    } finally {
      if (mounted) setState(() => _savingAll = false);
    }
  }

  void _scrollToCard(int index) {
    final ctx = _cardKeys[index]?.currentContext;
    if (ctx != null) Scrollable.ensureVisible(ctx, duration: const Duration(milliseconds: 350), alignment: 0.1);
  }

  @override
  Widget build(BuildContext context) {
    final link = ref.watch(boardLinksProvider).where((l) => l.id == widget.linkId).firstOrNull;
    if (link == null) {
      return const Scaffold(backgroundColor: SpottoColors.sand, body: Center(child: CircularProgressIndicator()));
    }
    final confirmed = link.confirmedPlaces;
    final unconfirmed = link.places.where((p) => !p.confirmed).toList();
    final refs = [for (final p in confirmed) BoardPlaceRef.fromBoardPlace(p)];

    return Scaffold(
      backgroundColor: SpottoColors.sand,
      appBar: AppBar(
        backgroundColor: SpottoColors.sand,
        surfaceTintColor: Colors.transparent,
        title: Text(link.region.isNotEmpty ? link.region : 'Places in this reel',
            style: const TextStyle(fontWeight: FontWeight.w800)),
        actions: [
          IconButton(
            tooltip: 'Open the original',
            icon: Icon(platformIcon(link.platform)),
            onPressed: () => launchUrl(Uri.parse(link.url), mode: LaunchMode.externalApplication),
          ),
        ],
      ),
      body: link.working
          ? const _Watching()
          : ListView(
              controller: _scroll,
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 32),
              children: [
                _Header(link: link),
                if (link.status == 'partial' && link.error.isNotEmpty) ...[
                  const SizedBox(height: 10),
                  _Notice(text: link.error),
                ],
                if (refs.isNotEmpty) ...[
                  const SizedBox(height: 14),
                  BoardPlacesMap(places: refs, height: 230, onTapPlace: _scrollToCard),
                  const SizedBox(height: 14),
                  Row(
                    children: [
                      Expanded(
                        child: Text('${refs.length} ${refs.length == 1 ? 'place' : 'places'} on the map',
                            style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w800, color: SpottoColors.ink)),
                      ),
                      TextButton.icon(
                        onPressed: _savingAll ? null : () => _saveAll(link),
                        icon: _savingAll
                            ? const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2))
                            : const Icon(Icons.bookmarks_rounded, size: 18),
                        label: const Text('Save all', style: TextStyle(fontWeight: FontWeight.w800)),
                      ),
                    ],
                  ),
                  const SizedBox(height: 6),
                  for (var i = 0; i < confirmed.length; i++) ...[
                    BoardPlaceCard(
                      key: _cardKeys.putIfAbsent(i, GlobalKey.new),
                      place: refs[i],
                      number: i + 1,
                      detail: confirmed[i].evidenceLabel,
                      sourceLinkId: link.id,
                    ),
                    const SizedBox(height: 10),
                  ],
                ],
                if (unconfirmed.isNotEmpty) ...[
                  const SizedBox(height: 12),
                  Text(confirmed.isEmpty ? 'Places mentioned' : "Mentioned, but we couldn't pin these",
                      style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w800, color: SpottoColors.inkSoft)),
                  const SizedBox(height: 6),
                  for (final p in unconfirmed)
                    ListTile(
                      contentPadding: EdgeInsets.zero,
                      leading: const Icon(Icons.help_outline_rounded, color: SpottoColors.muted),
                      title: Text(p.name, style: const TextStyle(fontWeight: FontWeight.w700)),
                      subtitle: Text([p.where, p.evidenceLabel].where((s) => s.isNotEmpty).join(' · ')),
                      trailing: TextButton(
                        onPressed: () => launchUrl(
                          Uri.https('www.google.com', '/maps/search/', {
                            'api': '1',
                            'query': [p.name, p.where, p.country].where((s) => s.isNotEmpty).join(', '),
                          }),
                          mode: LaunchMode.externalApplication,
                        ),
                        child: const Text('Search Maps'),
                      ),
                    ),
                ],
                if (link.places.isEmpty && !link.failed)
                  const Padding(
                    padding: EdgeInsets.symmetric(vertical: 40),
                    child: Text("We watched this reel but couldn't spot any named places.",
                        textAlign: TextAlign.center, style: TextStyle(color: SpottoColors.muted)),
                  ),
                if (link.failed) ...[const SizedBox(height: 16), _Notice(text: link.error)],
              ],
            ),
    );
  }
}

class _Watching extends StatelessWidget {
  const _Watching();

  @override
  Widget build(BuildContext context) {
    return const Center(
      child: Padding(
        padding: EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            CircularProgressIndicator(),
            SizedBox(height: 16),
            Text('Watching the reel…', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w800)),
            SizedBox(height: 6),
            Text('This usually takes under a minute. We\'ll send a notification when it\'s ready.',
                textAlign: TextAlign.center, style: TextStyle(color: SpottoColors.muted)),
          ],
        ),
      ),
    );
  }
}

class _Header extends StatelessWidget {
  final BoardLink link;
  const _Header({required this.link});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (link.author.isNotEmpty)
          Text('From ${link.author}', style: const TextStyle(color: SpottoColors.muted, fontWeight: FontWeight.w600)),
        if (link.caption.isNotEmpty) ...[
          const SizedBox(height: 4),
          Text(link.caption, maxLines: 3, overflow: TextOverflow.ellipsis,
              style: const TextStyle(color: SpottoColors.inkSoft, height: 1.35)),
        ],
      ],
    );
  }
}

class _Notice extends StatelessWidget {
  final String text;
  const _Notice({required this.text});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(color: const Color(0x1FF6B81E), borderRadius: BorderRadius.circular(12)),
      child: Row(
        children: [
          const Icon(Icons.info_outline_rounded, size: 18, color: Color(0xFF8A6410)),
          const SizedBox(width: 8),
          Expanded(child: Text(text, style: const TextStyle(color: Color(0xFF8A6410), fontWeight: FontWeight.w600))),
        ],
      ),
    );
  }
}
