// Local Guide chat — port of AIGuide.jsx, backed by the real
// POST /guide/chat (see server/internal/http/guide.go) instead of a
// canned keyword-matcher. Conversations persist server-side, so history
// is fetched on open via GET /guide/conversations/latest.

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';
import 'package:go_router/go_router.dart';

import '../../../core/api/api_client.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../data/mock/places.dart' as data;
import '../../../data/repositories/repositories.dart';
import 'guide_place.dart';

const _guideSuggestions = [
  'Where can I work today with a sea view?',
  'Best surf spot for a total beginner?',
  "Where's the move for sunset tonight?",
  'Any deals on right now?',
];

const _guideSuggestionsRainy = [
  'Where can I work today with a sea view?',
  "What's good to do in the rain?",
  'Any indoor workshops nearby?',
  'Any deals on right now?',
];

String _guideGreeting(String styleLabel, String weather) => weather == 'rainy'
    ? "Hey! It's raining right now 🌧️ — I've bumped indoor spots to "
        "the top. Ask me where to go and I'll keep you dry (and caffeinated)."
    : "Hey! I'm your Spotto local guide 🌴 I've set things up for a "
        '${styleLabel.toLowerCase()} — ask me where to work, surf, eat or catch '
        "the sunset and I'll point you to the right spot.";

class _Msg {
  final bool mine;
  final String content;
  final List<String> placeIds;
  // Card summaries from the server; empty for replies from older servers,
  // which fall back to placeIds (see guideCardsFor).
  final List<GuidePlace> cards;
  const _Msg(this.mine, this.content, {this.placeIds = const [], this.cards = const []});
}

List<GuidePlace> _parseCards(Object? raw) => [
  for (final p in (raw as List? ?? const []))
    if (p is Map) GuidePlace.fromJson(p.cast<String, dynamic>()),
];

class GuideScreen extends ConsumerStatefulWidget {
  const GuideScreen({super.key});

  @override
  ConsumerState<GuideScreen> createState() => _GuideScreenState();
}

class _GuideScreenState extends ConsumerState<GuideScreen> {
  final _messages = <_Msg>[];
  final _input = TextEditingController();
  final _scroll = ScrollController();
  bool _busy = false;
  String? _conversationId;

  String get _styleLabel {
    final style = ref.read(prefsProvider)?.styles.firstOrNull;
    return data.travelStyles.where((s) => s.id == style).firstOrNull?.label ?? 'Traveler';
  }

  @override
  void initState() {
    super.initState();
    _loadHistory();
  }

  /// Resumes the traveller's most recent conversation, if any — falls back
  /// to the canned weather-aware greeting for a genuinely new chat, or if
  /// the history fetch fails (offline, no prior conversation yet, etc).
  Future<void> _loadHistory() async {
    try {
      final res = await ref.read(apiProvider).dio.get('/guide/conversations/latest');
      final body = (res.data as Map).cast<String, dynamic>();
      final msgs = (body['messages'] as List? ?? const []).cast<Map>();
      // Cards for past replies, keyed by place id.
      final saved = <String, GuidePlace>{
        for (final e in ((body['places'] as Map?) ?? const {}).entries)
          if (e.value is Map) e.key as String: GuidePlace.fromJson((e.value as Map).cast<String, dynamic>()),
      };
      if (msgs.isNotEmpty) {
        if (!mounted) return;
        setState(() {
          _conversationId = body['conversationId'] as String?;
          _messages.addAll([
            for (final m in msgs)
              _Msg(
                m['role'] == 'user',
                m['content'] as String? ?? '',
                placeIds: (m['placeIds'] as List? ?? const []).cast<String>(),
                cards: [
                  for (final id in (m['placeIds'] as List? ?? const []).cast<String>())
                    if (saved[id] != null) saved[id]!,
                ],
              ),
          ]);
        });
        _scrollDown();
        return;
      }
    } catch (_) {
      // No prior conversation yet, or a transient error — either way, a
      // fresh greeting below is the right fallback, not an error state.
    }
    if (mounted) {
      setState(() =>
          _messages.add(_Msg(false, _guideGreeting(_styleLabel, ref.read(weatherConditionProvider)))));
    }
  }

  @override
  void dispose() {
    _input.dispose();
    _scroll.dispose();
    super.dispose();
  }


  void _scrollDown() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scroll.hasClients) {
        _scroll.animateTo(_scroll.position.maxScrollExtent,
            duration: const Duration(milliseconds: 250), curve: Curves.easeOut);
      }
    });
  }

  Future<void> _send([String? preset]) async {
    final q = (preset ?? _input.text).trim();
    if (q.isEmpty || _busy) return;
    setState(() {
      _messages.add(_Msg(true, q));
      _busy = true;
    });
    _input.clear();
    _scrollDown();

    // Await rather than a bare .value read: on a cold start (e.g. the
    // traveller opens Guide before the location permission dialog has been
    // answered — the iOS system prompt in particular can sit for a while),
    // the provider hasn't resolved yet and .value would silently be null,
    // sending this message with no location context at all. bestEffortPosition
    // already caps every branch with its own timeout, so this can't hang.
    final (pos, locationResult) = await ref.read(devicePositionProvider.future);
    // Only a real fix counts: when permission is off the provider falls back
    // to Weligama Bay, and the guide must not recommend around that.
    final locationKnown = locationResult == DeviceLocationResult.granted;
    try {
      final res = await ref.read(apiProvider).dio.post(
        '/guide/chat',
        data: {
          if (_conversationId != null) 'conversationId': _conversationId,
          'message': q,
          'locationKnown': locationKnown,
          if (locationKnown) 'lat': pos.lat,
          if (locationKnown) 'lng': pos.lng,
        },
        // The app-wide 10s receiveTimeout is shorter than one model reply
        // with tool calls; the server caps the whole reply at 60s.
        options: Options(receiveTimeout: const Duration(seconds: 75)),
      );
      final body = (res.data as Map).cast<String, dynamic>();
      if (!mounted) return;
      setState(() {
        _conversationId = body['conversationId'] as String?;
        _messages.add(_Msg(false, body['reply'] as String? ?? '',
            placeIds: (body['placeIds'] as List? ?? const []).cast<String>(), cards: _parseCards(body['places'])));
        _busy = false;
      });
    } catch (e) {
      if (!mounted) return;
      // A rate-limited/errored request still creates its conversation row
      // server-side and returns its id — without picking this up, every
      // failed retry would spawn a brand new empty conversation instead of
      // continuing this one.
      if (e is DioException) {
        final data = e.response?.data;
        if (data is Map && data['conversationId'] is String) {
          _conversationId = data['conversationId'] as String;
        }
      }
      setState(() {
        _messages.add(_Msg(false, ApiClient.errorMessage(e)));
        _busy = false;
      });
    }
    _scrollDown();
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final rainy = ref.watch(weatherConditionProvider) == 'rainy';
    final suggestions = rainy ? _guideSuggestionsRainy : _guideSuggestions;
    final places = ref.watch(placesProvider);
    final locationResult = ref.watch(devicePositionProvider).value?.$2;

    return Scaffold(
      backgroundColor: Theme.of(context).scaffoldBackgroundColor,
      body: SafeArea(
        child: Column(
          children: [
            // header
            Container(
              padding: const EdgeInsets.fromLTRB(20, 10, 16, 12),
              decoration: BoxDecoration(
                color: Theme.of(context).colorScheme.surface,
                boxShadow: [
                  BoxShadow(color: Color(0x40103D3A), blurRadius: 16, offset: Offset(0, 2), spreadRadius: -8)
                ],
              ),
              child: Row(
                children: [
                  Container(
                    width: 44,
                    height: 44,
                    decoration: BoxDecoration(
                      gradient: const LinearGradient(
                        begin: Alignment.topLeft,
                        end: Alignment.bottomRight,
                        colors: [SpottoColors.accent, SpottoColors.accent2],
                      ),
                      borderRadius: BorderRadius.circular(16),
                    ),
                    child: const Icon(Icons.auto_awesome, size: 22, color: Colors.white),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text('Local Guide', style: text.titleMedium?.copyWith(fontSize: 17)),
                        Row(
                          children: [
                            Container(
                              width: 7,
                              height: 7,
                              decoration: const BoxDecoration(
                                  color: Color(0xFF16A34A), shape: BoxShape.circle),
                            ),
                            const SizedBox(width: 5),
                            Text('Tuned for $_styleLabel',
                                style:
                                    text.bodySmall?.copyWith(color: SpottoColors.muted)),
                          ],
                        ),
                      ],
                    ),
                  ),
                  PressScale(
                    scale: 0.9,
                    onTap: () => context.pop(),
                    child: Container(
                      width: 36,
                      height: 36,
                      decoration: BoxDecoration(
                          color: Theme.of(context).scaffoldBackgroundColor, shape: BoxShape.circle),
                      child: Icon(Icons.close, size: 19, color: Theme.of(context).colorScheme.onSurface),
                    ),
                  ),
                ],
              ),
            ),
            // messages
            Expanded(
              child: ListView.builder(
                controller: _scroll,
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
                itemCount: _messages.length + (_busy ? 1 : 0),
                itemBuilder: (context, i) {
                  if (i == _messages.length) return const _TypingBubble();
                  final m = _messages[i];
                  final msgPlaces = m.mine ? const <GuidePlace>[] : guideCardsFor(m.cards, m.placeIds, places);
                  return Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: Column(
                      crossAxisAlignment:
                          m.mine ? CrossAxisAlignment.end : CrossAxisAlignment.start,
                      children: [
                        Align(
                          alignment:
                              m.mine ? Alignment.centerRight : Alignment.centerLeft,
                          child: Container(
                            constraints: BoxConstraints(
                                maxWidth: MediaQuery.of(context).size.width * 0.84),
                            padding: const EdgeInsets.symmetric(
                                horizontal: 14, vertical: 10),
                            decoration: BoxDecoration(
                              color: m.mine ? SpottoColors.accent : Theme.of(context).colorScheme.surface,
                              borderRadius: BorderRadius.only(
                                topLeft: const Radius.circular(20),
                                topRight: const Radius.circular(20),
                                bottomLeft: Radius.circular(m.mine ? 20 : 6),
                                bottomRight: Radius.circular(m.mine ? 6 : 20),
                              ),
                              boxShadow: const [
                                BoxShadow(
                                    color: Color(0x1F103D3A),
                                    blurRadius: 12,
                                    offset: Offset(0, 2))
                              ],
                            ),
                            child: Text(
                              m.content,
                              style: TextStyle(
                                fontSize: 14.5,
                                height: 1.5,
                                fontWeight: FontWeight.w500,
                                color: m.mine ? Colors.white : Theme.of(context).colorScheme.onSurface,
                              ),
                            ),
                          ),
                        ),
                        for (final place in msgPlaces)
                          Padding(
                            padding: const EdgeInsets.only(top: 6),
                            child: GuidePlaceCard(place: place),
                          ),
                      ],
                    ),
                  );
                },
              ),
            ),
            if (locationResult != null && locationResult != DeviceLocationResult.granted)
              _LocationHint(
                deniedForever: locationResult == DeviceLocationResult.deniedForever,
                onTap: () async {
                  if (locationResult == DeviceLocationResult.deniedForever) {
                    await Geolocator.openAppSettings();
                  } else {
                    ref.invalidate(devicePositionProvider);
                  }
                },
              ),
            // suggestion chips (only after just the greeting)
            if (_messages.length <= 1)
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
                child: Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    for (final s in suggestions)
                      PressScale(
                        scale: 0.95,
                        onTap: () => _send(s),
                        child: Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 12, vertical: 8),
                          decoration: BoxDecoration(
                            color: Theme.of(context).colorScheme.surface,
                            borderRadius: BorderRadius.circular(999),
                            border: Border.all(color: SpottoColors.line, width: 1.5),
                          ),
                          child: Text(s,
                              style: TextStyle(
                                  fontSize: 12.5,
                                  fontWeight: FontWeight.w700,
                                  color: Theme.of(context).colorScheme.onSurface)),
                        ),
                      ),
                  ],
                ),
              ),
            // composer
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 16),
              child: Container(
                padding: const EdgeInsets.all(6),
                decoration: BoxDecoration(
                  color: Theme.of(context).colorScheme.surface,
                  borderRadius: BorderRadius.circular(28),
                  boxShadow: const [
                    BoxShadow(color: Color(0x33103D3A), blurRadius: 18, offset: Offset(0, 4), spreadRadius: -8)
                  ],
                ),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.end,
                  children: [
                    Expanded(
                      child: TextField(
                        controller: _input,
                        onChanged: (_) => setState(() {}),
                        onSubmitted: (_) => _send(),
                        minLines: 1,
                        maxLines: 3,
                        textInputAction: TextInputAction.send,
                        decoration: const InputDecoration(
                          hintText: 'Ask your local friend…',
                          filled: false,
                          border: InputBorder.none,
                          enabledBorder: InputBorder.none,
                          focusedBorder: InputBorder.none,
                          contentPadding:
                              EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                        ),
                        style: TextStyle(
                            fontSize: 14.5,
                            fontWeight: FontWeight.w500,
                            color: Theme.of(context).colorScheme.onSurface),
                      ),
                    ),
                    PressScale(
                      scale: 0.9,
                      onTap: _input.text.trim().isEmpty || _busy ? null : _send,
                      child: Container(
                        width: 44,
                        height: 44,
                        decoration: BoxDecoration(
                          color: _input.text.trim().isNotEmpty && !_busy
                              ? SpottoColors.accent
                              : SpottoColors.ink.withValues(alpha: 0.12),
                          shape: BoxShape.circle,
                        ),
                        child: Icon(Icons.send,
                            size: 19,
                            color: _input.text.trim().isNotEmpty && !_busy
                                ? Colors.white
                                : SpottoColors.ink.withValues(alpha: 0.4)),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// Shown while the guide can't see the traveller's location, so its "which
/// town are you in?" makes sense — tapping asks for permission again (or
/// opens settings once it's been permanently denied).
class _LocationHint extends StatelessWidget {
  final bool deniedForever;
  final VoidCallback onTap;
  const _LocationHint({required this.deniedForever, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
      child: PressScale(
        scale: 0.98,
        onTap: onTap,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
          decoration: BoxDecoration(color: SpottoColors.accentTint, borderRadius: BorderRadius.circular(14)),
          child: Row(
            children: [
              const Icon(Icons.location_off_outlined, size: 17, color: SpottoColors.accent),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  deniedForever
                      ? 'Location is off for Spotto. Turn it on in Settings for nearby picks.'
                      : 'Turn on location so I can suggest places near you.',
                  style: const TextStyle(fontSize: 12.5, fontWeight: FontWeight.w700, color: SpottoColors.accent),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _TypingBubble extends StatefulWidget {
  const _TypingBubble();

  @override
  State<_TypingBubble> createState() => _TypingBubbleState();
}

class _TypingBubbleState extends State<_TypingBubble>
    with SingleTickerProviderStateMixin {
  late final AnimationController _c =
      AnimationController(vsync: this, duration: const Duration(milliseconds: 1100))
        ..repeat();

  @override
  void dispose() {
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Align(
      alignment: Alignment.centerLeft,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 14),
        decoration: BoxDecoration(
          color: Theme.of(context).colorScheme.surface,
          borderRadius: BorderRadius.only(
            topLeft: Radius.circular(20),
            topRight: Radius.circular(20),
            bottomLeft: Radius.circular(6),
            bottomRight: Radius.circular(20),
          ),
          boxShadow: [BoxShadow(color: Color(0x1F103D3A), blurRadius: 12, offset: Offset(0, 2))],
        ),
        child: AnimatedBuilder(
          animation: _c,
          builder: (context, _) => Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              for (var i = 0; i < 3; i++) ...[
                if (i > 0) const SizedBox(width: 4),
                Transform.translate(
                  offset: Offset(
                      0,
                      -4 *
                          (0.5 -
                                  ((_c.value + i * 0.15) % 1.0 - 0.5).abs())
                              .clamp(0.0, 0.5) *
                          2),
                  child: Container(
                    width: 7,
                    height: 7,
                    decoration: const BoxDecoration(
                        color: SpottoColors.accent, shape: BoxShape.circle),
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }
}
