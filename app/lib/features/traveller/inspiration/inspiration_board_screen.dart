// Inspiration Board: save a travel reel (paste or Share → Spotto), the
// server watches it and lists its places, and the traveller saves the ones
// they like. Two tabs: the saved links, and everything saved for later.

import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/api/api_client.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../data/models/inspiration.dart';
import '../../../data/repositories/inspiration_repository.dart';
import 'board_widgets.dart';
import 'nearby_alerts_ui.dart';

class InspirationBoardScreen extends ConsumerStatefulWidget {
  /// A link to add straight away (from Share → Spotto).
  final String? sharedUrl;

  /// Open on the Saved tab.
  final bool savedTab;
  const InspirationBoardScreen({super.key, this.sharedUrl, this.savedTab = false});

  @override
  ConsumerState<InspirationBoardScreen> createState() => _InspirationBoardScreenState();
}

class _InspirationBoardScreenState extends ConsumerState<InspirationBoardScreen> with SingleTickerProviderStateMixin {
  late final TabController _tabs = TabController(length: 2, vsync: this, initialIndex: widget.savedTab ? 1 : 0);
  final _url = TextEditingController();
  bool _adding = false;
  String? _error;
  String? _clipboardLink;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      ref.read(boardLinksProvider.notifier).refresh();
      ref.read(savedPlacesProvider.notifier).refresh();
      if (widget.sharedUrl != null) {
        _add(widget.sharedUrl!);
      } else {
        _checkClipboard();
      }
    });
  }

  @override
  void dispose() {
    _tabs.dispose();
    _url.dispose();
    super.dispose();
  }

  /// Offers a reel link the traveller just copied, so pasting is one tap.
  /// Android only: on iOS, reading the clipboard asks "Allow Paste?" every
  /// time, so iOS users use the paste button instead.
  Future<void> _checkClipboard() async {
    if (defaultTargetPlatform != TargetPlatform.android) return;
    final text = (await Clipboard.getData(Clipboard.kTextPlain))?.text ?? '';
    final link = findSupportedLink(text);
    if (link == null || !mounted) return;
    final known = ref.read(boardLinksProvider).any((l) => l.url == link);
    if (!known) setState(() => _clipboardLink = link);
  }

  Future<void> _add(String raw) async {
    final link = findSupportedLink(raw) ?? raw.trim();
    if (link.isEmpty || _adding) return;
    setState(() {
      _adding = true;
      _error = null;
      _clipboardLink = null;
    });
    try {
      final added = await ref.read(boardLinksProvider.notifier).add(link);
      _url.clear();
      if (!mounted) return;
      _tabs.animateTo(0);
      if (!added.working) context.push('/t/inspiration/${added.id}');
    } catch (e) {
      if (mounted) setState(() => _error = ApiClient.errorMessage(e));
    } finally {
      if (mounted) setState(() => _adding = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final links = ref.watch(boardLinksProvider);
    final saved = ref.watch(savedPlacesProvider);
    return Scaffold(
      backgroundColor: SpottoColors.sand,
      appBar: AppBar(
        backgroundColor: SpottoColors.sand,
        surfaceTintColor: Colors.transparent,
        title: const Text('Inspiration Board', style: TextStyle(fontWeight: FontWeight.w800)),
        bottom: TabBar(
          controller: _tabs,
          labelColor: SpottoColors.ink,
          unselectedLabelColor: SpottoColors.muted,
          indicatorColor: SpottoColors.sun,
          indicatorWeight: 3,
          labelStyle: const TextStyle(fontWeight: FontWeight.w800),
          tabs: [
            Tab(text: links.isEmpty ? 'Reels' : 'Reels · ${links.length}'),
            Tab(text: saved.isEmpty ? 'Saved' : 'Saved · ${saved.length}'),
          ],
        ),
      ),
      body: TabBarView(
        controller: _tabs,
        children: [
          RefreshIndicator(
            onRefresh: () => ref.read(boardLinksProvider.notifier).refresh(),
            child: ListView(
              padding: const EdgeInsets.fromLTRB(16, 16, 16, 32),
              children: [
                _AddBar(controller: _url, busy: _adding, error: _error, onAdd: () => _add(_url.text)),
                if (_clipboardLink != null) ...[
                  const SizedBox(height: 10),
                  _ClipboardOffer(link: _clipboardLink!, onAdd: () => _add(_clipboardLink!)),
                ],
                const SizedBox(height: 16),
                if (links.isEmpty)
                  const _EmptyLinks()
                else
                  for (final l in links) ...[
                    _LinkCard(link: l),
                    const SizedBox(height: 12),
                  ],
              ],
            ),
          ),
          const SavedPlacesTab(),
        ],
      ),
    );
  }
}

class _AddBar extends StatelessWidget {
  final TextEditingController controller;
  final bool busy;
  final String? error;
  final VoidCallback onAdd;
  const _AddBar({required this.controller, required this.busy, required this.error, required this.onAdd});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(20),
        boxShadow: const [BoxShadow(color: Color(0x12103D3A), blurRadius: 12, offset: Offset(0, 3))],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('Save a travel reel', style: TextStyle(fontSize: 15, fontWeight: FontWeight.w800)),
          const SizedBox(height: 2),
          const Text("Paste an Instagram, TikTok or YouTube link. We'll watch it and list the places in it.",
              style: TextStyle(fontSize: 12.5, color: SpottoColors.muted)),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: controller,
                  keyboardType: TextInputType.url,
                  textInputAction: TextInputAction.go,
                  onSubmitted: (_) => onAdd(),
                  decoration: InputDecoration(
                    hintText: 'https://www.instagram.com/reel/…',
                    isDense: true,
                    filled: true,
                    fillColor: SpottoColors.sand,
                    contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 12),
                    border: OutlineInputBorder(borderRadius: BorderRadius.circular(12), borderSide: BorderSide.none),
                    suffixIcon: IconButton(
                      tooltip: 'Paste',
                      icon: const Icon(Icons.content_paste_rounded, size: 18, color: SpottoColors.muted),
                      onPressed: () async {
                        final t = (await Clipboard.getData(Clipboard.kTextPlain))?.text;
                        if (t != null) controller.text = findSupportedLink(t) ?? t.trim();
                      },
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 8),
              FilledButton(
                onPressed: busy ? null : onAdd,
                style: FilledButton.styleFrom(
                  backgroundColor: SpottoColors.sun,
                  foregroundColor: SpottoColors.ink,
                  padding: const EdgeInsets.symmetric(horizontal: 18, vertical: 14),
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                ),
                child: busy
                    ? const SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2))
                    : const Text('Add', style: TextStyle(fontWeight: FontWeight.w800)),
              ),
            ],
          ),
          if (error != null) ...[
            const SizedBox(height: 8),
            Text(error!, style: const TextStyle(fontSize: 12.5, color: Color(0xFFB2472F), fontWeight: FontWeight.w600)),
          ],
        ],
      ),
    );
  }
}

class _ClipboardOffer extends StatelessWidget {
  final String link;
  final VoidCallback onAdd;
  const _ClipboardOffer({required this.link, required this.onAdd});

  @override
  Widget build(BuildContext context) {
    return Material(
      color: SpottoColors.accentTint,
      borderRadius: BorderRadius.circular(14),
      child: InkWell(
        borderRadius: BorderRadius.circular(14),
        onTap: onAdd,
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Row(
            children: [
              const Icon(Icons.content_paste_go_rounded, color: SpottoColors.accent),
              const SizedBox(width: 10),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text('Add the link you copied?', style: TextStyle(fontWeight: FontWeight.w800)),
                    Text(link, maxLines: 1, overflow: TextOverflow.ellipsis,
                        style: const TextStyle(fontSize: 12, color: SpottoColors.inkSoft)),
                  ],
                ),
              ),
              const Text('Add', style: TextStyle(fontWeight: FontWeight.w800, color: SpottoColors.accent)),
            ],
          ),
        ),
      ),
    );
  }
}

class _EmptyLinks extends StatelessWidget {
  const _EmptyLinks();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.symmetric(vertical: 32, horizontal: 12),
      child: Column(
        children: [
          Icon(Icons.video_library_outlined, size: 44, color: SpottoColors.accent),
          SizedBox(height: 12),
          Text('No reels yet', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w800)),
          SizedBox(height: 6),
          Text(
            'Found a "places to visit" reel? Tap Share in Instagram or TikTok and pick Spotto, or paste the link above.',
            textAlign: TextAlign.center,
            style: TextStyle(color: SpottoColors.muted, height: 1.4),
          ),
        ],
      ),
    );
  }
}

IconData platformIcon(String platform) => switch (platform) {
      'instagram' => Icons.camera_alt_rounded,
      'tiktok' => Icons.music_note_rounded,
      'youtube' => Icons.play_circle_fill_rounded,
      _ => Icons.link_rounded,
    };

class _LinkCard extends ConsumerWidget {
  final BoardLink link;
  const _LinkCard({required this.link});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final found = link.places.length;
    final (statusText, statusColor) = switch (link.status) {
      'pending' || 'processing' => ('Watching the reel…', SpottoColors.accent),
      'failed' => (link.error.isNotEmpty ? link.error : "Couldn't read this link", const Color(0xFFB2472F)),
      _ when found == 0 => ('No named places found', SpottoColors.muted),
      _ => (found == 1 ? '1 place' : '$found places', SpottoColors.ink),
    };
    return PressScale(
      scale: 0.98,
      onTap: link.working ? null : () => context.push('/t/inspiration/${link.id}'),
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(18),
          border: Border.all(color: SpottoColors.line),
        ),
        child: Row(
          children: [
            ClipRRect(
              borderRadius: BorderRadius.circular(12),
              child: SizedBox(
                width: 64,
                height: 84,
                child: link.thumbnailUrl.isEmpty
                    ? Container(color: SpottoColors.accentTint, child: Icon(platformIcon(link.platform), color: SpottoColors.accent))
                    : CachedNetworkImage(
                        imageUrl: link.thumbnailUrl,
                        fit: BoxFit.cover,
                        errorWidget: (_, _, _) =>
                            Container(color: SpottoColors.accentTint, child: Icon(platformIcon(link.platform), color: SpottoColors.accent)),
                      ),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(children: [
                    Icon(platformIcon(link.platform), size: 14, color: SpottoColors.muted),
                    const SizedBox(width: 4),
                    Expanded(
                      child: Text(link.author.isNotEmpty ? link.author : link.platform,
                          maxLines: 1, overflow: TextOverflow.ellipsis,
                          style: const TextStyle(fontSize: 12, color: SpottoColors.muted, fontWeight: FontWeight.w600)),
                    ),
                  ]),
                  const SizedBox(height: 4),
                  Text(
                    link.region.isNotEmpty ? link.region : (link.caption.isNotEmpty ? link.caption : link.url),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w800, color: SpottoColors.ink),
                  ),
                  const SizedBox(height: 6),
                  Row(children: [
                    if (link.working) ...[
                      const SizedBox(width: 12, height: 12, child: CircularProgressIndicator(strokeWidth: 2)),
                      const SizedBox(width: 6),
                    ],
                    Expanded(
                      child: Text(statusText,
                          maxLines: 2, overflow: TextOverflow.ellipsis,
                          style: TextStyle(fontSize: 12.5, fontWeight: FontWeight.w700, color: statusColor)),
                    ),
                  ]),
                  if (link.canRetry)
                    Align(
                      alignment: Alignment.centerLeft,
                      child: TextButton(
                        style: TextButton.styleFrom(padding: EdgeInsets.zero, minimumSize: const Size(0, 32)),
                        onPressed: () async {
                          try {
                            await ref.read(boardLinksProvider.notifier).retry(link.id);
                          } catch (e) {
                            if (context.mounted) {
                              ScaffoldMessenger.of(context)
                                  .showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
                            }
                          }
                        },
                        child: const Text('Retry', style: TextStyle(fontWeight: FontWeight.w800)),
                      ),
                    ),
                ],
              ),
            ),
            PopupMenuButton<String>(
              icon: const Icon(Icons.more_vert_rounded, color: SpottoColors.muted),
              onSelected: (_) async {
                try {
                  await ref.read(boardLinksProvider.notifier).remove(link.id);
                } catch (e) {
                  if (context.mounted) {
                    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
                  }
                }
              },
              itemBuilder: (_) => const [PopupMenuItem(value: 'remove', child: Text('Remove'))],
            ),
          ],
        ),
      ),
    );
  }
}

/// Everything saved for later, grouped by city, with a map.
class SavedPlacesTab extends ConsumerWidget {
  const SavedPlacesTab({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final saved = ref.watch(savedPlacesProvider);
    if (saved.isEmpty) {
      return const Center(
        child: Padding(
          padding: EdgeInsets.all(32),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(Icons.bookmark_border_rounded, size: 44, color: SpottoColors.accent),
              SizedBox(height: 12),
              Text('Nothing saved yet', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w800)),
              SizedBox(height: 6),
              Text('Save places from a reel, or bookmark any spot on the map, and they show up here.',
                  textAlign: TextAlign.center, style: TextStyle(color: SpottoColors.muted, height: 1.4)),
            ],
          ),
        ),
      );
    }
    final groups = <String, List<SavedPlace>>{};
    for (final s in saved) {
      groups.putIfAbsent(s.city.isNotEmpty ? s.city : (s.place != null ? 'Spotto spots' : 'Other'), () => []).add(s);
    }
    final refs = [for (final s in saved) BoardPlaceRef.fromSaved(s)];
    return RefreshIndicator(
      onRefresh: () => ref.read(savedPlacesProvider.notifier).refresh(),
      child: ListView(
        padding: const EdgeInsets.fromLTRB(16, 16, 16, 32),
        children: [
          const NearbyAlertsCard(),
          const SizedBox(height: 14),
          BoardPlacesMap(places: refs, height: 200),
          for (final entry in groups.entries) ...[
            Padding(
              padding: const EdgeInsets.fromLTRB(4, 20, 4, 8),
              child: Text('${entry.key} · ${entry.value.length}',
                  style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w800, color: SpottoColors.ink)),
            ),
            for (final s in entry.value) ...[
              BoardPlaceCard(place: BoardPlaceRef.fromSaved(s)),
              const SizedBox(height: 10),
            ],
          ],
        ],
      ),
    );
  }
}
