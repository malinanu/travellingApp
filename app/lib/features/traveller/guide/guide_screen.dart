// Local Guide chat — port of AIGuide.jsx with the canned guide engine.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/mock/guide_engine.dart';
import '../../../data/mock/places.dart' as data;
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';
import '../place_detail/business_drawer.dart';

class _Msg {
  final bool mine;
  final String content;
  const _Msg(this.mine, this.content);
}

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

  String get _styleLabel {
    final style = ref.read(prefsProvider)?.styles.firstOrNull;
    return data.travelStyles.where((s) => s.id == style).firstOrNull?.label ?? 'Traveler';
  }

  @override
  void initState() {
    super.initState();
    _messages.add(_Msg(false, guideGreeting(_styleLabel, ref.read(weatherConditionProvider))));
  }

  @override
  void dispose() {
    _input.dispose();
    _scroll.dispose();
    super.dispose();
  }

  /// guideFindPlace matches against the hardcoded demo dataset, whose ids
  /// (e.g. 'driftwood') don't exist in the live places table — resolve by
  /// name against the real, API-backed list so we never hand a phantom
  /// place into the business drawer (posting a review against it 404s).
  Place? _liveGuidePlace(Place? mock) {
    if (mock == null) return null;
    return ref.read(placesProvider).where((p) => p.name == mock.name).firstOrNull;
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

    final reply = await guideReply(q,
        weather: ref.read(weatherConditionProvider), style: ref.read(prefsProvider)?.styles.firstOrNull);
    if (!mounted) return;
    setState(() {
      _messages.add(_Msg(false, reply));
      _busy = false;
    });
    _scrollDown();
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final rainy = ref.watch(weatherConditionProvider) == 'rainy';
    final suggestions = rainy ? guideSuggestionsRainy : guideSuggestions;

    return Scaffold(
      backgroundColor: SpottoColors.sand,
      body: SafeArea(
        child: Column(
          children: [
            // header
            Container(
              padding: const EdgeInsets.fromLTRB(20, 10, 16, 12),
              decoration: const BoxDecoration(
                color: Colors.white,
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
                      decoration: const BoxDecoration(
                          color: SpottoColors.sand, shape: BoxShape.circle),
                      child: const Icon(Icons.close, size: 19, color: SpottoColors.ink),
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
                  final place = m.mine ? null : _liveGuidePlace(guideFindPlace(m.content));
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
                              color: m.mine ? SpottoColors.accent : Colors.white,
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
                                color: m.mine ? Colors.white : SpottoColors.ink,
                              ),
                            ),
                          ),
                        ),
                        if (place != null)
                          Padding(
                            padding: const EdgeInsets.only(top: 6),
                            child: PressScale(
                              scale: 0.98,
                              onTap: () => showBusinessDrawer(context, place),
                              child: Container(
                                padding: const EdgeInsets.all(10),
                                decoration: BoxDecoration(
                                  color: Colors.white,
                                  borderRadius: BorderRadius.circular(16),
                                  boxShadow: const [
                                    BoxShadow(
                                        color: Color(0x2E103D3A),
                                        blurRadius: 12,
                                        offset: Offset(0, 2))
                                  ],
                                ),
                                child: Row(
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
                                    Container(
                                      width: 32,
                                      height: 32,
                                      decoration: BoxDecoration(
                                        color: SpottoColors.accentTint,
                                        borderRadius: BorderRadius.circular(10),
                                      ),
                                      child: Icon(spottoIcon(place.icon),
                                          size: 16, color: SpottoColors.accent),
                                    ),
                                    const SizedBox(width: 8),
                                    Column(
                                      crossAxisAlignment: CrossAxisAlignment.start,
                                      children: [
                                        Text(place.name,
                                            style: const TextStyle(
                                                fontSize: 12.5,
                                                fontWeight: FontWeight.w800,
                                                color: SpottoColors.ink)),
                                        const Text('View details →',
                                            style: TextStyle(
                                                fontSize: 11,
                                                fontWeight: FontWeight.w700,
                                                color: SpottoColors.accent)),
                                      ],
                                    ),
                                  ],
                                ),
                              ),
                            ),
                          ),
                      ],
                    ),
                  );
                },
              ),
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
                            color: Colors.white,
                            borderRadius: BorderRadius.circular(999),
                            border: Border.all(color: SpottoColors.line, width: 1.5),
                          ),
                          child: Text(s,
                              style: const TextStyle(
                                  fontSize: 12.5,
                                  fontWeight: FontWeight.w700,
                                  color: SpottoColors.ink)),
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
                  color: Colors.white,
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
                        style: const TextStyle(
                            fontSize: 14.5,
                            fontWeight: FontWeight.w500,
                            color: SpottoColors.ink),
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
        decoration: const BoxDecoration(
          color: Colors.white,
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
