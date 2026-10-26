// Pieces shared by the Stories tab and a single story: the card, the cover,
// the Markdown body and a linked-place row.

import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../core/api/api_client.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/progressive_image.dart';
import '../../../core/widgets/spotto_card.dart';
import '../../../data/models/content.dart';
import '../../../data/repositories/repositories.dart';
import '../place_detail/business_drawer.dart';

Color kindColor(ContentKind kind) => switch (kind) {
      ContentKind.audio => const Color(0xFF7C5CFF),
      ContentKind.blog => SpottoColors.accent,
      ContentKind.guide => SpottoColors.accent2,
      ContentKind.itinerary => const Color(0xFFC2683F),
      ContentKind.tip => const Color(0xFF8A6410),
    };

IconData kindIcon(ContentKind kind) => switch (kind) {
      ContentKind.audio => Icons.headphones_rounded,
      ContentKind.blog => Icons.menu_book_rounded,
      ContentKind.guide => Icons.map_outlined,
      ContentKind.itinerary => Icons.event_note_rounded,
      ContentKind.tip => Icons.lightbulb_outline_rounded,
    };

/// "Audio guide · 12 min", "Itinerary · 2 days", "Blog".
String storyKindLine(ContentItem item) {
  final extra = switch (item.kind) {
    ContentKind.audio => formatListenLength(item.audioSeconds),
    ContentKind.itinerary when item.days.isNotEmpty => '${item.days.length} ${item.days.length == 1 ? 'day' : 'days'}',
    _ => '',
  };
  return extra.isEmpty ? item.kind.label : '${item.kind.label} · $extra';
}

/// "Nimali Perera · 26 Oct 2026 · Galle" (whatever is known).
String storyMetaLine(ContentItem item) => [
      if (item.authorName.isNotEmpty) item.authorName,
      if (item.publishedAt != null) DateFormat('d MMM y').format(item.publishedAt!.toLocal()),
      if (item.regionName.isNotEmpty) item.regionName,
    ].join(' · ');

/// The post's cover photo, or its kind's colour and icon when it has none.
class StoryCover extends StatelessWidget {
  const StoryCover({super.key, required this.item, this.height, this.borderRadius, this.iconSize = 34});

  final ContentItem item;
  final double? height;
  final BorderRadius? borderRadius;
  final double iconSize;

  @override
  Widget build(BuildContext context) {
    final color = kindColor(item.kind);
    Widget placeholder(BuildContext context) => Container(
          height: height,
          decoration: BoxDecoration(color: color.withValues(alpha: 0.12), borderRadius: borderRadius),
          alignment: Alignment.center,
          child: Icon(kindIcon(item.kind), size: iconSize, color: color.withValues(alpha: 0.7)),
        );
    if (item.coverUrl.isEmpty) return placeholder(context);
    return ProgressiveImage(
      url: resolveMediaUrl(item.coverUrl),
      height: height,
      width: double.infinity,
      borderRadius: borderRadius,
      errorBuilder: placeholder,
    );
  }
}

class StoryKindBadge extends StatelessWidget {
  const StoryKindBadge({super.key, required this.item, this.onImage = false});

  final ContentItem item;
  final bool onImage;

  @override
  Widget build(BuildContext context) {
    final color = kindColor(item.kind);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: onImage ? Colors.white : color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(mainAxisSize: MainAxisSize.min, children: [
        Icon(kindIcon(item.kind), size: 14, color: color),
        const SizedBox(width: 5),
        Text(storyKindLine(item),
            style: Theme.of(context).textTheme.labelSmall?.copyWith(color: color, fontWeight: FontWeight.w700)),
      ]),
    );
  }
}

/// A post in the Stories list.
class StoryCard extends StatelessWidget {
  const StoryCard({super.key, required this.item, required this.onTap});

  final ContentItem item;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final meta = storyMetaLine(item);
    return SpottoCard(
      onTap: onTap,
      padding: EdgeInsets.zero,
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Stack(children: [
          StoryCover(
            item: item,
            height: 150,
            borderRadius: const BorderRadius.vertical(top: Radius.circular(19)),
          ),
          Positioned(left: 12, top: 12, child: StoryKindBadge(item: item, onImage: true)),
        ]),
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 14),
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text(item.title, style: text.titleMedium, maxLines: 2, overflow: TextOverflow.ellipsis),
            if (item.summary.isNotEmpty) ...[
              const SizedBox(height: 4),
              Text(item.summary,
                  style: text.bodyMedium?.copyWith(color: SpottoColors.inkSoft),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis),
            ],
            if (meta.isNotEmpty) ...[
              const SizedBox(height: 8),
              Text(meta, style: text.labelSmall?.copyWith(color: SpottoColors.muted)),
            ],
          ]),
        ),
      ]),
    );
  }
}

/// A post's Markdown body. Links open in the browser (web and mail links
/// only); photos load through the API like every other upload.
class StoryMarkdown extends StatelessWidget {
  const StoryMarkdown({super.key, required this.data});

  final String data;

  static Future<void> _openLink(String? href) async {
    final uri = href == null ? null : Uri.tryParse(href);
    if (uri == null || !{'http', 'https', 'mailto'}.contains(uri.scheme)) return;
    await launchUrl(uri, mode: LaunchMode.externalApplication);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final body = theme.textTheme.bodyLarge?.copyWith(height: 1.55, color: SpottoColors.ink);
    return MarkdownBody(
      data: data,
      onTapLink: (text, href, title) => _openLink(href),
      imageBuilder: (uri, title, alt) {
        final url = resolveMediaUrl(uri.toString());
        if (!url.startsWith('https://') && !url.startsWith('http://')) return const SizedBox.shrink();
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 6),
          child: ClipRRect(
            borderRadius: BorderRadius.circular(14),
            child: Image(
              image: CachedNetworkImageProvider(url, cacheKey: mediaCacheKey(url)),
              width: double.infinity,
              fit: BoxFit.cover,
              semanticLabel: alt,
              errorBuilder: (_, _, _) => const SizedBox.shrink(),
            ),
          ),
        );
      },
      styleSheet: MarkdownStyleSheet.fromTheme(theme).copyWith(
        p: body,
        listBullet: body,
        h1: theme.textTheme.titleLarge,
        h2: theme.textTheme.titleLarge,
        h3: theme.textTheme.titleMedium,
        h4: theme.textTheme.titleSmall,
        a: body?.copyWith(color: SpottoColors.accent, decoration: TextDecoration.underline),
        blockquote: body?.copyWith(fontStyle: FontStyle.italic, color: SpottoColors.inkSoft),
        blockquoteDecoration: const BoxDecoration(
          border: Border(left: BorderSide(color: SpottoColors.accent, width: 3)),
        ),
        blockquotePadding: const EdgeInsets.only(left: 14, top: 2, bottom: 2),
        code: theme.textTheme.bodyMedium?.copyWith(fontFamily: 'monospace', backgroundColor: SpottoColors.accentTint),
        blockSpacing: 14,
      ),
    );
  }
}

/// Opens a linked Spotto place's drawer: straight from the loaded places,
/// or after fetching it.
Future<void> openStoryPlace(BuildContext context, WidgetRef ref, String placeId) async {
  final cached = ref.read(placesProvider).where((p) => p.id == placeId).firstOrNull;
  if (cached != null) {
    await showBusinessDrawer(context, cached);
    return;
  }
  try {
    final place = await ref.read(placeDetailsProvider(placeId).future);
    if (context.mounted) await showBusinessDrawer(context, place);
  } catch (e) {
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
    }
  }
}

/// A linked place: photo, name and kind, tapping opens it.
class StoryPlaceTile extends ConsumerWidget {
  const StoryPlaceTile({super.key, required this.place, this.index});

  final ContentPlace place;

  /// Shown as a number badge when the places are an ordered route.
  final int? index;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    return SpottoCard(
      onTap: () => openStoryPlace(context, ref, place.id),
      padding: const EdgeInsets.all(10),
      radius: 16,
      shadow: false,
      child: Row(children: [
        SizedBox(
          width: 52,
          height: 52,
          child: place.cover.isEmpty
              ? Container(
                  decoration: BoxDecoration(color: SpottoColors.accentTint, borderRadius: BorderRadius.circular(12)),
                  alignment: Alignment.center,
                  child: index != null
                      ? Text('$index', style: text.titleMedium?.copyWith(color: SpottoColors.accent))
                      : const Icon(Icons.place_outlined, color: SpottoColors.accent),
                )
              : ProgressiveImage(url: resolveMediaUrl(place.cover), borderRadius: BorderRadius.circular(12)),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text(place.name, style: text.titleSmall, maxLines: 1, overflow: TextOverflow.ellipsis),
            if (place.category.isNotEmpty)
              Text(place.category, style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
          ]),
        ),
        const Icon(Icons.chevron_right_rounded, color: SpottoColors.muted),
      ]),
    );
  }
}
