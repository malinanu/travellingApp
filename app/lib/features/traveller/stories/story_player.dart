// The audio player on an audio story, and the mini-player that stays above
// the tabs while something is loaded (see TravellerShell).

import 'package:audio_service/audio_service.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/audio/story_audio.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/progressive_image.dart';
import '../../../data/models/content.dart';
import 'story_widgets.dart';

/// Starts [item] (from [start] when given), reporting a failure in a snackbar.
Future<void> playStoryAudio(BuildContext context, WidgetRef ref, ContentItem item, {Duration? start}) async {
  try {
    await ref.read(audioHandlerProvider).playStory(item, start: start);
  } catch (e) {
    if (context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text("Couldn't play this audio. ${e is StateError ? e.message : 'Check your connection and try again.'}")),
      );
    }
  }
}

bool _busy(PlaybackState? s) =>
    s != null && (s.processingState == AudioProcessingState.loading || s.processingState == AudioProcessingState.buffering);

/// Index of the chapter playing at [pos], or -1 before the first.
int currentChapter(List<ContentChapter> chapters, Duration pos) {
  var at = -1;
  for (var i = 0; i < chapters.length; i++) {
    if (pos.inSeconds >= chapters[i].startSec) at = i;
  }
  return at;
}

class StoryPlayerCard extends ConsumerStatefulWidget {
  const StoryPlayerCard({super.key, required this.item});
  final ContentItem item;

  @override
  ConsumerState<StoryPlayerCard> createState() => _StoryPlayerCardState();
}

class _StoryPlayerCardState extends ConsumerState<StoryPlayerCard> {
  double? _dragging; // seconds, while the slider is held

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final item = widget.item;
    final handler = ref.watch(audioHandlerProvider);
    final nowPlaying = ref.watch(nowPlayingProvider).value;
    final state = ref.watch(playbackStateProvider).value;
    final loaded = nowPlaying?.id == item.id;
    final playing = loaded && (state?.playing ?? false) && state?.processingState != AudioProcessingState.completed;

    Future<void> toggle() async {
      if (!loaded) return playStoryAudio(context, ref, item);
      if (playing) return handler.pause();
      if (state?.processingState == AudioProcessingState.completed) await handler.seek(Duration.zero);
      if (!context.mounted) return;
      return playStoryAudio(context, ref, item);
    }

    return Container(
      padding: const EdgeInsets.fromLTRB(16, 14, 16, 12),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: SpottoColors.line, width: 1.5),
      ),
      child: StreamBuilder<Duration>(
        stream: loaded ? handler.positionStream : const Stream.empty(),
        builder: (context, snap) {
          final total = (loaded ? handler.duration : null) ??
              (item.audioSeconds > 0 ? Duration(seconds: item.audioSeconds) : Duration.zero);
          final pos = loaded ? (snap.data ?? handler.position) : Duration.zero;
          final maxSec = total.inMilliseconds / 1000;
          final value = (_dragging ?? pos.inMilliseconds / 1000).clamp(0.0, maxSec > 0 ? maxSec : 0.0);
          final chapterIx = loaded ? currentChapter(item.chapters, pos) : -1;

          return Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
            SliderTheme(
              data: SliderTheme.of(context).copyWith(trackHeight: 4, overlayShape: SliderComponentShape.noOverlay),
              child: Slider(
                value: value.toDouble(),
                max: maxSec > 0 ? maxSec : 1,
                activeColor: SpottoColors.accent,
                inactiveColor: SpottoColors.accentTint,
                onChanged: maxSec > 0 ? (v) => setState(() => _dragging = v) : null,
                onChangeEnd: (v) async {
                  setState(() => _dragging = null);
                  final to = Duration(milliseconds: (v * 1000).round());
                  if (loaded) {
                    await handler.seek(to);
                  } else {
                    await playStoryAudio(context, ref, item, start: to);
                  }
                },
              ),
            ),
            const SizedBox(height: 6),
            Row(children: [
              Text(formatClock(Duration(milliseconds: (value * 1000).round())), style: text.labelSmall),
              const Spacer(),
              Text(total > Duration.zero ? formatClock(total) : '--:--', style: text.labelSmall),
            ]),
            const SizedBox(height: 4),
            Row(mainAxisAlignment: MainAxisAlignment.center, children: [
              _SpeedButton(enabled: loaded, speed: state?.speed ?? 1.0, onPick: handler.setSpeed),
              const Spacer(),
              IconButton(
                tooltip: 'Back 15 seconds',
                iconSize: 30,
                onPressed: loaded ? () => handler.skip(-kStorySkip) : null,
                icon: const Icon(Icons.replay_10_rounded),
              ),
              const SizedBox(width: 8),
              SizedBox(
                width: 64,
                height: 64,
                child: FilledButton(
                  style: FilledButton.styleFrom(
                    shape: const CircleBorder(),
                    padding: EdgeInsets.zero,
                    backgroundColor: SpottoColors.accent,
                  ),
                  onPressed: toggle,
                  child: loaded && _busy(state)
                      ? const SizedBox(
                          width: 24, height: 24, child: CircularProgressIndicator(strokeWidth: 2.5, color: Colors.white))
                      : Icon(playing ? Icons.pause_rounded : Icons.play_arrow_rounded, size: 34, semanticLabel: playing ? 'Pause' : 'Play'),
                ),
              ),
              const SizedBox(width: 8),
              IconButton(
                tooltip: 'Forward 15 seconds',
                iconSize: 30,
                onPressed: loaded ? () => handler.skip(kStorySkip) : null,
                icon: const Icon(Icons.forward_10_rounded),
              ),
              const Spacer(),
              const SizedBox(width: 48),
            ]),
            if (item.chapters.isNotEmpty) ...[
              const Divider(height: 24),
              Text('Chapters', style: text.titleSmall),
              const SizedBox(height: 4),
              for (var i = 0; i < item.chapters.length; i++)
                InkWell(
                  borderRadius: BorderRadius.circular(10),
                  onTap: () {
                    final to = Duration(seconds: item.chapters[i].startSec);
                    if (loaded) {
                      handler.seek(to);
                      if (!playing) playStoryAudio(context, ref, item);
                    } else {
                      playStoryAudio(context, ref, item, start: to);
                    }
                  },
                  child: Padding(
                    padding: const EdgeInsets.symmetric(vertical: 9, horizontal: 4),
                    child: Row(children: [
                      Icon(
                        i == chapterIx ? Icons.graphic_eq_rounded : Icons.play_circle_outline_rounded,
                        size: 18,
                        color: i == chapterIx ? SpottoColors.accent : SpottoColors.muted,
                      ),
                      const SizedBox(width: 10),
                      Expanded(
                        child: Text(item.chapters[i].title,
                            style: text.bodyMedium?.copyWith(
                              fontWeight: i == chapterIx ? FontWeight.w700 : FontWeight.w500,
                              color: i == chapterIx ? SpottoColors.accent : SpottoColors.ink,
                            )),
                      ),
                      Text(formatClock(Duration(seconds: item.chapters[i].startSec)),
                          style: text.labelSmall?.copyWith(color: SpottoColors.muted)),
                    ]),
                  ),
                ),
            ],
          ]);
        },
      ),
    );
  }
}

class _SpeedButton extends StatelessWidget {
  const _SpeedButton({required this.enabled, required this.speed, required this.onPick});
  final bool enabled;
  final double speed;
  final Future<void> Function(double) onPick;

  static String _label(double s) => '${s.toStringAsFixed(s == s.roundToDouble() ? 0 : 2).replaceFirst(RegExp(r'0$'), '')}×';

  @override
  Widget build(BuildContext context) {
    return PopupMenuButton<double>(
      enabled: enabled,
      tooltip: 'Playback speed',
      initialValue: speed,
      onSelected: onPick,
      itemBuilder: (context) => [
        for (final s in kStorySpeeds) PopupMenuItem(value: s, child: Text(_label(s))),
      ],
      child: Container(
        width: 48,
        padding: const EdgeInsets.symmetric(vertical: 6),
        decoration: BoxDecoration(
          color: enabled ? SpottoColors.accentTint : SpottoColors.line,
          borderRadius: BorderRadius.circular(999),
        ),
        alignment: Alignment.center,
        child: Text(_label(speed),
            style: Theme.of(context).textTheme.labelMedium?.copyWith(
                  color: enabled ? SpottoColors.accent : SpottoColors.muted,
                )),
      ),
    );
  }
}

/// Shown above the tabs while a story's audio is loaded, except on that
/// story's own page ([location] is the current route).
class StoryMiniPlayer extends ConsumerWidget {
  const StoryMiniPlayer({super.key, required this.location});
  final String location;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final item = ref.watch(nowPlayingProvider).value;
    if (item == null || location == '/t/stories/${Uri.encodeComponent(item.id)}') return const SizedBox.shrink();
    final handler = ref.watch(audioHandlerProvider);
    final state = ref.watch(playbackStateProvider).value;
    final playing = (state?.playing ?? false) && state?.processingState != AudioProcessingState.completed;
    final text = Theme.of(context).textTheme;
    final story = handler.story;

    return Material(
      color: Colors.white,
      child: InkWell(
        onTap: () => context.go('/t/stories/${Uri.encodeComponent(item.id)}'),
        child: Container(
          decoration: const BoxDecoration(border: Border(top: BorderSide(color: SpottoColors.line))),
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            StreamBuilder<Duration>(
              stream: handler.positionStream,
              builder: (context, snap) {
                final total = handler.duration ?? item.duration;
                final pos = snap.data ?? handler.position;
                final v = total == null || total.inMilliseconds == 0 ? 0.0 : pos.inMilliseconds / total.inMilliseconds;
                return LinearProgressIndicator(
                  value: v.clamp(0.0, 1.0),
                  minHeight: 2,
                  backgroundColor: SpottoColors.accentTint,
                  color: SpottoColors.accent,
                );
              },
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(12, 8, 4, 8),
              child: Row(children: [
                SizedBox(
                  width: 40,
                  height: 40,
                  child: story != null
                      ? StoryCover(item: story, height: 40, borderRadius: BorderRadius.circular(8), iconSize: 20)
                      : item.artUri != null
                          ? ProgressiveImage(url: item.artUri.toString(), borderRadius: BorderRadius.circular(8))
                          : const Icon(Icons.headphones_rounded, color: SpottoColors.accent),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                    Text(item.title, style: text.titleSmall, maxLines: 1, overflow: TextOverflow.ellipsis),
                    Text(
                      _busy(state) ? 'Loading…' : (playing ? 'Playing' : 'Paused'),
                      style: text.labelSmall?.copyWith(color: SpottoColors.muted),
                    ),
                  ]),
                ),
                IconButton(
                  tooltip: playing ? 'Pause' : 'Play',
                  onPressed: () async {
                    if (playing) return handler.pause();
                    if (state?.processingState == AudioProcessingState.completed) await handler.seek(Duration.zero);
                    await handler.play();
                  },
                  icon: Icon(playing ? Icons.pause_rounded : Icons.play_arrow_rounded, color: SpottoColors.accent),
                ),
                IconButton(
                  tooltip: 'Close player',
                  onPressed: handler.stop,
                  icon: const Icon(Icons.close_rounded, color: SpottoColors.muted),
                ),
              ]),
            ),
          ]),
        ),
      ),
    );
  }
}
