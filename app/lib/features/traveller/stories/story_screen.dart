// One story: its cover, then by kind — the audio player, chapters and transcript, a
// guide's places, an itinerary's days (which the traveller can add to their
// trips), or just the text.

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../core/api/api_client.dart';
import '../../../core/theme/palette.dart';
import '../../../data/catalog/itinerary_import.dart';
import '../../../data/catalog/trip_catalog.dart';
import '../../../data/models/content.dart';
import '../../../data/repositories/content_repository.dart';
import '../../../data/repositories/repositories.dart';
import 'story_player.dart';
import 'story_widgets.dart';

class StoryScreen extends ConsumerWidget {
  const StoryScreen({super.key, required this.storyId});
  final String storyId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final story = ref.watch(storyProvider(storyId));
    return Scaffold(
      backgroundColor: SpottoColors.sand,
      body: switch (story) {
        AsyncData(:final value) => _StoryBody(item: value),
        AsyncError(:final error) => _StoryError(
            gone: error is DioException && error.response?.statusCode == 404,
            message: ApiClient.errorMessage(error),
            onRetry: () => ref.invalidate(storyProvider(storyId)),
          ),
        _ => const Center(child: CircularProgressIndicator()),
      },
    );
  }
}

class _StoryBody extends ConsumerWidget {
  const _StoryBody({required this.item});
  final ContentItem item;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final meta = storyMetaLine(item);
    final guidePlaces = [
      for (final id in item.placeIds)
        if (item.place(id) != null) item.place(id)!,
    ];

    return CustomScrollView(slivers: [
      SliverAppBar(
        pinned: true,
        expandedHeight: 230,
        backgroundColor: SpottoColors.sand,
        surfaceTintColor: Colors.transparent,
        leading: Padding(
          padding: const EdgeInsets.all(8),
          child: CircleAvatar(
            backgroundColor: Colors.white,
            child: IconButton(
              tooltip: 'Back',
              icon: const Icon(Icons.arrow_back_rounded, color: SpottoColors.ink, size: 20),
              onPressed: () => context.canPop() ? context.pop() : context.go('/t/stories'),
            ),
          ),
        ),
        flexibleSpace: FlexibleSpaceBar(
          background: StoryCover(item: item, iconSize: 56),
        ),
      ),
      SliverPadding(
        padding: const EdgeInsets.fromLTRB(18, 18, 18, 110),
        sliver: SliverList.list(children: [
          Align(alignment: Alignment.centerLeft, child: StoryKindBadge(item: item)),
          const SizedBox(height: 10),
          Text(item.title, style: text.headlineSmall),
          if (meta.isNotEmpty) ...[
            const SizedBox(height: 6),
            Text(meta, style: text.labelMedium?.copyWith(color: SpottoColors.muted)),
          ],
          if (item.summary.isNotEmpty) ...[
            const SizedBox(height: 12),
            Text(item.summary, style: text.bodyLarge?.copyWith(color: SpottoColors.inkSoft, height: 1.45)),
          ],
          const SizedBox(height: 18),
          if (item.kind == ContentKind.audio && item.audioUrl.isNotEmpty) ...[
            StoryPlayerCard(item: item),
            const SizedBox(height: 22),
            if (item.transcript case final transcript?) ...[
              StoryTranscript(item: item, transcript: transcript),
              const SizedBox(height: 22),
            ],
          ],
          if (item.kind == ContentKind.itinerary && item.stopCount > 0) ...[
            FilledButton.icon(
              style: FilledButton.styleFrom(
                backgroundColor: SpottoColors.accent,
                padding: const EdgeInsets.symmetric(vertical: 14),
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
              ),
              onPressed: () => showAddItinerarySheet(context, ref, item),
              icon: const Icon(Icons.add_road_rounded),
              label: const Text('Add to my trips'),
            ),
            const SizedBox(height: 22),
          ],
          if (item.body.isNotEmpty) ...[
            StoryMarkdown(data: item.body),
            const SizedBox(height: 22),
          ],
          if (item.kind == ContentKind.guide && guidePlaces.isNotEmpty) ...[
            Text('Places in this guide', style: text.titleMedium),
            const SizedBox(height: 10),
            for (var i = 0; i < guidePlaces.length; i++) ...[
              StoryPlaceTile(place: guidePlaces[i], index: i + 1),
              const SizedBox(height: 10),
            ],
            const SizedBox(height: 12),
          ],
          if (item.kind == ContentKind.itinerary)
            for (var i = 0; i < item.days.length; i++) ...[
              _ItineraryDayCard(item: item, day: item.days[i], number: i + 1),
              const SizedBox(height: 12),
            ],
          if (item.tags.isNotEmpty) ...[
            const SizedBox(height: 6),
            Wrap(spacing: 8, runSpacing: 8, children: [
              for (final t in item.tags)
                Chip(
                  label: Text('#$t'),
                  visualDensity: VisualDensity.compact,
                  backgroundColor: Colors.white,
                  side: const BorderSide(color: SpottoColors.line),
                ),
            ]),
          ],
        ]),
      ),
    ]);
  }
}

class _ItineraryDayCard extends ConsumerWidget {
  const _ItineraryDayCard({required this.item, required this.day, required this.number});
  final ContentItem item;
  final ItineraryDay day;
  final int number;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    return Container(
      padding: const EdgeInsets.fromLTRB(16, 14, 16, 8),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: SpottoColors.line, width: 1.5),
      ),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text('DAY $number', style: text.labelSmall?.copyWith(color: SpottoColors.muted, letterSpacing: 1.1)),
        if (day.title.isNotEmpty) Text(day.title, style: text.titleMedium),
        const SizedBox(height: 8),
        for (final stop in day.stops)
          InkWell(
            borderRadius: BorderRadius.circular(10),
            onTap: stop.placeId.isEmpty ? null : () => openStoryPlace(context, ref, stop.placeId),
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 8),
              child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
                SizedBox(
                  width: 72,
                  child: Text(stop.minutes == null ? '' : tripFmtTime(stop.minutes!),
                      style: text.labelMedium?.copyWith(color: SpottoColors.accent)),
                ),
                Expanded(
                  child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                    Text(stop.name, style: text.titleSmall),
                    if (stop.note.isNotEmpty)
                      Text(stop.note, style: text.bodySmall?.copyWith(color: SpottoColors.inkSoft)),
                  ]),
                ),
                if (stop.placeId.isNotEmpty) const Icon(Icons.chevron_right_rounded, color: SpottoColors.muted),
              ]),
            ),
          ),
      ]),
    );
  }
}

/// Asks for a name and start date, then makes the itinerary the active trip.
Future<void> showAddItinerarySheet(BuildContext context, WidgetRef ref, ContentItem item) async {
  final added = await showModalBottomSheet<String>(
    context: context,
    isScrollControlled: true,
    backgroundColor: Colors.white,
    shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(20))),
    builder: (context) => _AddItinerarySheet(item: item),
  );
  if (added == null || !context.mounted) return;
  ScaffoldMessenger.of(context).showSnackBar(SnackBar(
    content: Text('“$added” is now your active trip.'),
    action: SnackBarAction(label: 'Open', onPressed: () => context.go('/t/trip')),
  ));
}

class _AddItinerarySheet extends ConsumerStatefulWidget {
  const _AddItinerarySheet({required this.item});
  final ContentItem item;

  @override
  ConsumerState<_AddItinerarySheet> createState() => _AddItinerarySheetState();
}

class _AddItinerarySheetState extends ConsumerState<_AddItinerarySheet> {
  late final _name = TextEditingController(text: widget.item.title);
  var _start = DateTime.now();

  @override
  void dispose() {
    _name.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final item = widget.item;
    return Padding(
      padding: EdgeInsets.fromLTRB(20, 18, 20, 20 + MediaQuery.of(context).viewInsets.bottom),
      child: Column(mainAxisSize: MainAxisSize.min, crossAxisAlignment: CrossAxisAlignment.stretch, children: [
        Text('Add to my trips', style: text.titleLarge),
        const SizedBox(height: 4),
        Text(
          '${item.days.length} ${item.days.length == 1 ? 'day' : 'days'} · ${item.stopCount} '
          '${item.stopCount == 1 ? 'stop' : 'stops'}. You can change anything afterwards in the Trip tab.',
          style: text.bodyMedium?.copyWith(color: SpottoColors.muted),
        ),
        const SizedBox(height: 16),
        TextField(
          controller: _name,
          textCapitalization: TextCapitalization.sentences,
          decoration: const InputDecoration(labelText: 'Trip name', border: OutlineInputBorder()),
        ),
        const SizedBox(height: 12),
        OutlinedButton.icon(
          style: OutlinedButton.styleFrom(padding: const EdgeInsets.symmetric(vertical: 14)),
          icon: const Icon(Icons.calendar_today_rounded, size: 18),
          label: Text('Starts ${DateFormat('EEE d MMM y').format(_start)}'),
          onPressed: () async {
            final now = DateTime.now();
            final picked = await showDatePicker(
              context: context,
              initialDate: _start,
              firstDate: DateTime(now.year, now.month, now.day),
              lastDate: now.add(const Duration(days: 730)),
            );
            if (picked != null) setState(() => _start = picked);
          },
        ),
        const SizedBox(height: 16),
        FilledButton(
          style: FilledButton.styleFrom(
            backgroundColor: SpottoColors.accent,
            padding: const EdgeInsets.symmetric(vertical: 14),
          ),
          onPressed: () {
            final trip =
                tripFromItinerary(item, name: _name.text, startDate: DateFormat('yyyy-MM-dd').format(_start));
            ref.read(tripProvider.notifier).update(trip);
            Navigator.of(context).pop(trip.name);
          },
          child: const Text('Add trip'),
        ),
      ]),
    );
  }
}

class _StoryError extends StatelessWidget {
  const _StoryError({required this.gone, required this.message, required this.onRetry});
  final bool gone;
  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return SafeArea(
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        IconButton(
          tooltip: 'Back',
          icon: const Icon(Icons.arrow_back_rounded),
          onPressed: () => context.canPop() ? context.pop() : context.go('/t/stories'),
        ),
        Expanded(
          child: Center(
            child: Padding(
              padding: const EdgeInsets.all(32),
              child: Column(mainAxisSize: MainAxisSize.min, children: [
                Icon(gone ? Icons.auto_stories_outlined : Icons.cloud_off_rounded, size: 44, color: SpottoColors.muted),
                const SizedBox(height: 12),
                Text(gone ? 'This story isn’t available any more.' : "This story couldn't load.",
                    style: text.titleMedium, textAlign: TextAlign.center),
                const SizedBox(height: 6),
                Text(gone ? 'It may have been taken down.' : message,
                    style: text.bodyMedium?.copyWith(color: SpottoColors.muted), textAlign: TextAlign.center),
                if (!gone) TextButton(onPressed: onRetry, child: const Text('Try again')),
              ]),
            ),
          ),
        ),
      ]),
    );
  }
}
