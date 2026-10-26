// Stories tab: audio guides, blog posts, travel guides, itineraries and tips
// published in the Command Center, filtered by kind and search, with posts
// about towns near the traveller first.

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/api/api_client.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/pill.dart';
import '../../../core/widgets/stagger_in.dart';
import '../../../data/models/content.dart';
import '../../../data/repositories/content_repository.dart';
import '../../../data/repositories/repositories.dart';
import 'story_widgets.dart';

class StoriesScreen extends ConsumerStatefulWidget {
  const StoriesScreen({super.key});

  @override
  ConsumerState<StoriesScreen> createState() => _StoriesScreenState();
}

class _StoriesScreenState extends ConsumerState<StoriesScreen> {
  late final _search = TextEditingController(text: ref.read(storiesQueryProvider));
  Timer? _debounce;

  @override
  void dispose() {
    _debounce?.cancel();
    _search.dispose();
    super.dispose();
  }

  void _onSearch(String v) {
    setState(() {}); // shows/hides the clear button
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 350), () => ref.read(storiesQueryProvider.notifier).state = v.trim());
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final kind = ref.watch(storiesKindProvider);
    final query = ref.watch(storiesQueryProvider);
    final feed = ref.watch(storiesFeedProvider);
    final position = ref.watch(devicePositionProvider).value;

    final rows = <Widget>[];
    var rowIndex = 0;
    Widget staggered(Widget child) => StaggerIn(index: rowIndex++, child: child);

    rows.add(staggered(Text('Stories', style: text.headlineSmall)));
    rows.add(const SizedBox(height: 4));
    rows.add(staggered(Text('Audio guides, travel guides, itineraries and tips from people who know Sri Lanka.',
        style: text.bodyMedium)));
    rows.add(const SizedBox(height: 14));
    rows.add(staggered(TextField(
      controller: _search,
      onChanged: _onSearch,
      textInputAction: TextInputAction.search,
      decoration: InputDecoration(
        hintText: 'Search stories',
        prefixIcon: const Icon(Icons.search, size: 18, color: SpottoColors.muted),
        suffixIcon: _search.text.isEmpty
            ? null
            : IconButton(
                icon: const Icon(Icons.close, size: 18, color: SpottoColors.muted),
                onPressed: () {
                  _search.clear();
                  _onSearch('');
                },
              ),
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
      ),
    )));
    rows.add(const SizedBox(height: 10));
    rows.add(staggered(SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: Row(children: [
        Pill(label: 'All', selected: kind == null, onTap: () => ref.read(storiesKindProvider.notifier).state = null),
        for (final k in ContentKind.values) ...[
          const SizedBox(width: 8),
          Pill(
            label: k.plural,
            icon: kindIcon(k),
            selected: kind == k,
            onTap: () => ref.read(storiesKindProvider.notifier).state = kind == k ? null : k,
          ),
        ],
      ]),
    )));
    rows.add(const SizedBox(height: 18));

    void open(ContentItem item) => context.push('/t/stories/${Uri.encodeComponent(item.id)}');
    void addCards(List<ContentItem> items) {
      for (final item in items) {
        rows.add(staggered(StoryCard(key: ValueKey(item.id), item: item, onTap: () => open(item))));
        rows.add(const SizedBox(height: 14));
      }
    }

    switch (feed) {
      case AsyncData(:final value) when value.isEmpty:
        rows.add(_Message(
          icon: Icons.auto_stories_outlined,
          title: query.isNotEmpty || kind != null ? 'No stories match.' : 'No stories yet.',
          body: query.isNotEmpty || kind != null
              ? 'Try another search or kind of story.'
              : 'Guides, audio tours and tips will appear here as they are published.',
        ));
      case AsyncData(:final value):
        final pos = position;
        final split = pos != null && pos.$2 == DeviceLocationResult.granted && query.isEmpty
            ? splitNearby(value, lat: pos.$1.lat, lng: pos.$1.lng)
            : null;
        if (split != null && split.near.isNotEmpty) {
          rows.add(_SectionLabel('Near you'));
          addCards(split.near);
          if (split.rest.isNotEmpty) rows.add(_SectionLabel('More from Sri Lanka'));
          addCards(split.rest);
        } else {
          addCards(value);
        }
      case AsyncError(:final error):
        rows.add(_Message(
          icon: Icons.cloud_off_rounded,
          title: "Stories couldn't load.",
          body: ApiClient.errorMessage(error),
          action: TextButton(onPressed: () => ref.invalidate(storiesFeedProvider), child: const Text('Try again')),
        ));
      default:
        rows.add(const Padding(
          padding: EdgeInsets.symmetric(vertical: 40),
          child: Center(child: CircularProgressIndicator()),
        ));
    }

    return SafeArea(
      child: RefreshIndicator(
        onRefresh: () => ref.refresh(storiesFeedProvider.future),
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 90),
          children: rows,
        ),
      ),
    );
  }
}

class _SectionLabel extends StatelessWidget {
  const _SectionLabel(this.label);
  final String label;

  @override
  Widget build(BuildContext context) => Padding(
        padding: const EdgeInsets.only(bottom: 10, top: 2),
        child: Text(label.toUpperCase(),
            style: Theme.of(context).textTheme.labelSmall?.copyWith(color: SpottoColors.muted, letterSpacing: 1.1)),
      );
}

class _Message extends StatelessWidget {
  const _Message({required this.icon, required this.title, required this.body, this.action});
  final IconData icon;
  final String title;
  final String body;
  final Widget? action;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 32, horizontal: 12),
      child: Column(children: [
        Icon(icon, size: 40, color: SpottoColors.muted),
        const SizedBox(height: 10),
        Text(title, style: text.titleMedium, textAlign: TextAlign.center),
        const SizedBox(height: 4),
        Text(body, style: text.bodyMedium?.copyWith(color: SpottoColors.muted), textAlign: TextAlign.center),
        if (action != null) ...[const SizedBox(height: 8), action!],
      ]),
    );
  }
}
