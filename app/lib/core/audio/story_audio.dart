// Plays Stories audio (audio guides and audiobooks) with just_audio, through
// audio_service so it keeps playing with the screen off or the app in the
// background, with lock-screen and notification controls.
//
// One post plays at a time. main() starts the service (initStoryAudio) and
// hands the handler to the app through audioHandlerProvider.

import 'dart:async';

import 'package:audio_service/audio_service.dart';
import 'package:audio_session/audio_session.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:just_audio/just_audio.dart';

import '../../data/models/content.dart';
import '../../data/repositories/content_repository.dart';
import '../api/api_client.dart';

/// How far the skip buttons (and the lock-screen ones) jump.
const kStorySkip = Duration(seconds: 15);

/// Playback speeds the player offers.
const kStorySpeeds = [0.75, 1.0, 1.25, 1.5, 2.0];

/// What the lock screen and notification show while [item] is at [pos]: the
/// playing chapter (-1 before the first, or with no chapters), its photo
/// (else the cover; '' for neither) and its title ('' outside a chapter).
({int chapter, String art, String subtitle}) storyNowShowing(ContentItem item, Duration pos) {
  final i = item.chapterAt(pos.inMilliseconds / 1000);
  return (chapter: i, art: item.chapterImage(i), subtitle: i >= 0 ? item.chapters[i].title : '');
}

class StoryAudioHandler extends BaseAudioHandler {
  StoryAudioHandler({AudioPlayer? player}) : _player = player ?? AudioPlayer() {
    _player.playbackEventStream.listen(
      (_) => _broadcast(),
      onError: (Object e, StackTrace st) => debugPrint('story audio: playback error: $e'),
    );
    _player.playingStream.listen((_) => _broadcast());
    _player.speedStream.listen((_) => _broadcast());
    _player.durationStream.listen((d) {
      final item = mediaItem.value;
      if (d != null && item != null && item.duration != d) mediaItem.add(item.copyWith(duration: d));
    });
    _player.playingStream.listen((playing) {
      final story = _story;
      if (playing && story != null && _counted.add(story.id)) onListen?.call(story.id);
    });
    _player.positionStream.listen(_followChapter);
  }

  /// Shows the playing chapter's photo and title on the lock screen and in
  /// the notification, so a listener can see where in the guide they are.
  void _followChapter(Duration pos) {
    final story = _story;
    final item = mediaItem.value;
    if (story == null || item == null || story.chapters.isEmpty) return;
    final now = storyNowShowing(story, pos);
    if (now.chapter == _shownChapter) return;
    _shownChapter = now.chapter;
    mediaItem.add(item.copyWith(
      artUri: now.art.isEmpty ? null : Uri.parse(resolveMediaUrl(now.art)),
      artist: now.subtitle.isNotEmpty ? now.subtitle : _artist(story),
      displaySubtitle: now.subtitle.isNotEmpty ? now.subtitle : null,
    ));
  }

  static String _artist(ContentItem item) => item.authorName.isNotEmpty ? item.authorName : 'Spotto';

  final AudioPlayer _player;
  ContentItem? _story;
  int _shownChapter = -1;
  bool _sessionReady = false;
  final _counted = <String>{};

  /// Called the first time each post starts playing in this session.
  void Function(String storyId)? onListen;

  /// The post that's loaded, or null when nothing is.
  ContentItem? get story => _story;

  Stream<Duration> get positionStream => _player.positionStream;
  Duration get position => _player.position;
  Duration? get duration => _player.duration;
  double get speed => _player.speed;

  /// Plays [item]'s audio, from [start] when given. Resumes when it's
  /// already the loaded post (the media link may be newer, e.g. after the
  /// old one expired, so a fresh one is loaded when it changed).
  Future<void> playStory(ContentItem item, {Duration? start}) async {
    if (item.audioUrl.isEmpty) throw StateError('This post has no audio.');
    if (!_sessionReady) {
      try {
        final session = await AudioSession.instance;
        await session.configure(const AudioSessionConfiguration.speech());
      } catch (e) {
        debugPrint('story audio: audio session setup failed: $e');
      }
      _sessionReady = true;
    }
    final same = _story?.id == item.id && _story?.audioUrl == item.audioUrl &&
        _player.processingState != ProcessingState.idle;
    if (!same) {
      final resumeAt = start ?? (_story?.id == item.id ? _player.position : null);
      _story = item;
      final now = storyNowShowing(item, resumeAt ?? Duration.zero);
      _shownChapter = now.chapter;
      mediaItem.add(MediaItem(
        id: item.id,
        title: item.title,
        artist: now.subtitle.isNotEmpty ? now.subtitle : _artist(item),
        album: item.regionName.isNotEmpty ? item.regionName : 'Spotto Stories',
        displaySubtitle: now.subtitle.isNotEmpty ? now.subtitle : null,
        duration: item.audioSeconds > 0 ? Duration(seconds: item.audioSeconds) : null,
        artUri: now.art.isEmpty ? null : Uri.parse(resolveMediaUrl(now.art)),
      ));
      await _player.setUrl(resolveMediaUrl(item.audioUrl), initialPosition: resumeAt);
    } else if (start != null) {
      await _player.seek(start);
    }
    unawaited(_player.play());
  }

  /// Jumps by [by] (negative to go back), within the recording.
  Future<void> skip(Duration by) async {
    var to = _player.position + by;
    final d = _player.duration;
    if (to < Duration.zero) to = Duration.zero;
    if (d != null && to > d) to = d;
    await _player.seek(to);
  }

  @override
  Future<void> play() => _player.play();

  @override
  Future<void> pause() => _player.pause();

  @override
  Future<void> seek(Duration position) => _player.seek(position);

  @override
  Future<void> fastForward() => skip(kStorySkip);

  @override
  Future<void> rewind() => skip(-kStorySkip);

  @override
  Future<void> setSpeed(double speed) => _player.setSpeed(speed);

  /// Stops and unloads the post (the mini-player goes away).
  @override
  Future<void> stop() async {
    await _player.stop();
    _story = null;
    _shownChapter = -1;
    mediaItem.add(null);
    await super.stop();
  }

  void _broadcast() {
    final playing = _player.playing;
    playbackState.add(PlaybackState(
      controls: [
        MediaControl.rewind,
        if (playing) MediaControl.pause else MediaControl.play,
        MediaControl.fastForward,
        MediaControl.stop,
      ],
      systemActions: const {MediaAction.seek, MediaAction.seekForward, MediaAction.seekBackward},
      androidCompactActionIndices: const [0, 1, 2],
      processingState: switch (_player.processingState) {
        ProcessingState.idle => AudioProcessingState.idle,
        ProcessingState.loading => AudioProcessingState.loading,
        ProcessingState.buffering => AudioProcessingState.buffering,
        ProcessingState.ready => AudioProcessingState.ready,
        ProcessingState.completed => AudioProcessingState.completed,
      },
      playing: playing,
      updatePosition: _player.position,
      bufferedPosition: _player.bufferedPosition,
      speed: _player.speed,
    ));
  }
}

/// Starts the background audio service. Never fails: without it (or if it
/// doesn't start in time) the handler still plays while the app is open,
/// just without lock-screen controls.
Future<StoryAudioHandler> initStoryAudio() async {
  try {
    return await AudioService.init(
      builder: StoryAudioHandler.new,
      config: const AudioServiceConfig(
        androidNotificationChannelId: 'com.spotto.spotto.stories',
        androidNotificationChannelName: 'Stories audio',
        androidNotificationChannelDescription: 'Controls for audio guides you are listening to',
        androidNotificationOngoing: true,
        androidStopForegroundOnPause: true,
        fastForwardInterval: kStorySkip,
        rewindInterval: kStorySkip,
      ),
    ).timeout(const Duration(seconds: 5));
  } catch (e) {
    debugPrint('story audio: background service unavailable, playing in the app only: $e');
    return StoryAudioHandler();
  }
}

/// The app's audio handler; main() overrides it with the one bound to the
/// background service, and has it count listens (countStoryListens).
final audioHandlerProvider = Provider<StoryAudioHandler>((ref) => StoryAudioHandler());

/// Reports each post's first listen in this session to the API.
void countStoryListens(StoryAudioHandler handler, ApiClient api) {
  handler.onListen = (id) => countStoryPlay(api, id);
}

/// What's loaded in the player (null = nothing).
final nowPlayingProvider = StreamProvider<MediaItem?>((ref) => ref.watch(audioHandlerProvider).mediaItem);

final playbackStateProvider = StreamProvider<PlaybackState>((ref) => ref.watch(audioHandlerProvider).playbackState);
