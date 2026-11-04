// Local Guide chat — port of AIGuide.jsx, backed by the real
// POST /guide/chat (see server/internal/http/guide.go) instead of a
// canned keyword-matcher. Conversations persist server-side, so history
// is fetched on open via GET /guide/conversations/latest. Trip planning
// replies carry cards (questions, a visual plan, add-to-calendar) — see
// guide_cards.dart.

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:geolocator/geolocator.dart';
import 'package:go_router/go_router.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../core/api/api_client.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../data/catalog/app_options.dart' as data;
import '../../../data/repositories/repositories.dart';
import 'guide_cards.dart';
import 'guide_payload.dart';
import 'guide_place.dart';

const _guideSuggestions = [
  'Where can I work today with a sea view?',
  'Help me plan a trip',
  "Where's the move for sunset tonight?",
  'Any deals on right now?',
];

const _guideSuggestionsRainy = [
  'Where can I work today with a sea view?',
  "What's good to do in the rain?",
  'Help me plan a trip',
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
  final GuidePayload? payload;
  final List<String> suggestions;
  const _Msg(
    this.mine,
    this.content, {
    this.placeIds = const [],
    this.cards = const [],
    this.payload,
    this.suggestions = const [],
  });

  String get text => content;
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
  List<String> _activeSuggestions = const [];
  // Trip name each calendar card's places were added to, by message index.
  final _addedTo = <int, String>{};

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
        final lastAssistant = msgs.reversed.where((m) => m['role'] == 'assistant').firstOrNull;
        List<String> activeSuggestions = const [];
        if (body['suggestions'] is List && (body['suggestions'] as List).isNotEmpty) {
          activeSuggestions = (body['suggestions'] as List).cast<String>();
        } else if (lastAssistant != null) {
          final p = GuidePayload.tryParse(lastAssistant['payload']);
          final pIds = (lastAssistant['placeIds'] as List? ?? const []).cast<String>();
          if (p?.plan != null) {
            activeSuggestions = const [
              '🔄 Give me more options',
              '🏖️ Beach vibe stops',
              '☕ Cafes & chill spots',
              '📅 Add to trip calendar',
            ];
          } else if (pIds.isNotEmpty) {
            activeSuggestions = const [
              '🔄 Give me more options',
              '📅 Plan a trip with these',
              '☕ Cafes & chill',
              '🏖️ Beach vibe',
              '📍 Closer spots',
            ];
          } else {
            activeSuggestions = const [
              '📅 Plan a trip here',
              '☕ Best cafes nearby',
              '🏖️ Fun things to do',
              '🔄 More recommendations',
            ];
          }
        }
        setState(() {
          _conversationId = body['conversationId'] as String?;
          _activeSuggestions = activeSuggestions;
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
                payload: GuidePayload.tryParse(m['payload']),
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
      final rainy = ref.read(weatherConditionProvider) == 'rainy';
      final initialSuggestions = rainy ? _guideSuggestionsRainy : _guideSuggestions;
      setState(() {
        _messages.add(_Msg(false, _guideGreeting(_styleLabel, ref.read(weatherConditionProvider))));
        _activeSuggestions = initialSuggestions;
      });
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
    final lowerQ = q.toLowerCase();
    if (lowerQ == 'clear' ||
        lowerQ == 'clear chat' ||
        lowerQ == 'clear the chat' ||
        lowerQ == 'clear chat history' ||
        lowerQ == 'clear history' ||
        lowerQ == 'reset chat' ||
        lowerQ == 'reset conversation' ||
        lowerQ.startsWith('clear chat') ||
        lowerQ.startsWith('clear the chat')) {
      _input.clear();
      await _confirmClearChat();
      return;
    }
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
    // to the centre of Sri Lanka, and the guide must not recommend around that.
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
        // with tool calls; the server caps the whole reply at 90s (a trip
        // plan searches Google Places several times).
        options: Options(receiveTimeout: const Duration(seconds: 110)),
      );
      final body = (res.data as Map).cast<String, dynamic>();
      if (!mounted) return;
      final payload = GuidePayload.tryParse(body['payload']);
      final cards = _parseCards(body['places']);
      final placeIds = (body['placeIds'] as List? ?? const []).cast<String>();
      final serverSuggestions = (body['suggestions'] as List? ?? const []).cast<String>();
      final suggestions = serverSuggestions.isNotEmpty
          ? serverSuggestions
          : (payload?.plan != null
              ? const [
                  '🔄 Give me more options',
                  '🏖️ Beach vibe stops',
                  '☕ Cafes & chill spots',
                  '📅 Add to trip calendar',
                ]
              : (cards.isNotEmpty || placeIds.isNotEmpty
                  ? const [
                      '🔄 Give me more options',
                      '📅 Plan a trip with these',
                      '☕ Cafes & chill',
                      '🏖️ Beach vibe',
                      '📍 Closer spots',
                    ]
                  : const [
                      '📅 Plan a trip here',
                      '☕ Best cafes nearby',
                      '🏖️ Fun things to do',
                      '🔄 More recommendations',
                    ]));

      setState(() {
        _conversationId = body['conversationId'] as String?;
        _messages.add(_Msg(
          false,
          body['reply'] as String? ?? '',
          placeIds: placeIds,
          cards: cards,
          payload: payload,
          suggestions: suggestions,
        ));
        _activeSuggestions = suggestions;
        _busy = false;
      });
      // "Add it to a new trip" / "…to my current trip": the traveller already
      // said where, so add straight away instead of asking again.
      final cal = payload?.calendar;
      if (cal != null && cal.mode != 'ask') {
        _scrollDown();
        await _addToCalendar(cal, cal.mode, _messages.length - 1);
      }
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

  Future<void> _confirmClearChat() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        backgroundColor: Theme.of(ctx).colorScheme.surface,
        title: const Text('Clear chat history?', style: TextStyle(fontWeight: FontWeight.w800)),
        content: const Text('This will delete your current conversation with Local Guide and start fresh.'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            style: FilledButton.styleFrom(backgroundColor: Theme.of(ctx).colorScheme.error),
            child: const Text('Clear'),
          ),
        ],
      ),
    );
    if (confirmed == true) {
      await _clearChat();
    }
  }

  Future<void> _clearChat() async {
    try {
      await ref.read(apiProvider).dio.post('/guide/conversations/clear', data: {
        if (_conversationId != null) 'conversationId': _conversationId,
      });
    } catch (_) {}
    if (!mounted) return;
    final rainy = ref.read(weatherConditionProvider) == 'rainy';
    final initialSuggestions = rainy ? _guideSuggestionsRainy : _guideSuggestions;
    setState(() {
      _conversationId = null;
      _messages.clear();
      _messages.add(_Msg(false, _guideGreeting(_styleLabel, ref.read(weatherConditionProvider))));
      _activeSuggestions = initialSuggestions;
    });
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
      content: Text('Chat history cleared'),
      duration: Duration(seconds: 2),
    ));
  }

  Future<void> _summarizeChat() async {
    final localStore = ref.read(localStoreProvider);
    final savedSummaries = (localStore.getJsonList('spotto_saved_chat_summaries') ?? []).cast<Map>();
    if (_messages.length <= 1) {
      if (savedSummaries.isNotEmpty) {
        _showSavedSummariesSheet(savedSummaries);
        return;
      }
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
        content: Text('Chat with Local Guide first to generate a summary!'),
      ));
      return;
    }
    showDialog(
      context: context,
      barrierDismissible: false,
      builder: (_) => const Center(child: CircularProgressIndicator(color: SpottoColors.accent)),
    );
    try {
      final res = await ref.read(apiProvider).dio.post('/guide/conversations/summarize');
      if (mounted) Navigator.pop(context);
      if (!mounted) return;
      final body = (res.data as Map).cast<String, dynamic>();
      final title = body['title'] as String? ?? 'Trip Summary & Recommendations';
      final summary = body['summary'] as String? ?? '';
      final places = _parseCards(body['places']);

      showModalBottomSheet(
        context: context,
        isScrollControlled: true,
        backgroundColor: Theme.of(context).colorScheme.surface,
        shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(24))),
        builder: (sheet) => DraggableScrollableSheet(
          initialChildSize: 0.75,
          minChildSize: 0.4,
          maxChildSize: 0.95,
          expand: false,
          builder: (_, scroll) => Padding(
            padding: const EdgeInsets.fromLTRB(20, 16, 20, 20),
            child: ListView(
              controller: scroll,
              children: [
                Center(
                  child: Container(
                    width: 40,
                    height: 4,
                    decoration: BoxDecoration(color: SpottoColors.line, borderRadius: BorderRadius.circular(2)),
                  ),
                ),
                const SizedBox(height: 16),
                Row(
                  children: [
                    Container(
                      padding: const EdgeInsets.all(8),
                      decoration: BoxDecoration(color: SpottoColors.accentTint, borderRadius: BorderRadius.circular(12)),
                      child: const Icon(Icons.auto_stories, color: SpottoColors.accent, size: 22),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Text(title, style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w800)),
                    ),
                    if (savedSummaries.isNotEmpty)
                      IconButton(
                        tooltip: 'View all saved summaries',
                        icon: const Icon(Icons.folder_special_outlined, color: SpottoColors.accent),
                        onPressed: () {
                          Navigator.pop(sheet);
                          _showSavedSummariesSheet(savedSummaries);
                        },
                      ),
                  ],
                ),
                const SizedBox(height: 16),
                Container(
                  padding: const EdgeInsets.all(16),
                  decoration: BoxDecoration(
                    color: Theme.of(context).scaffoldBackgroundColor,
                    borderRadius: BorderRadius.circular(16),
                    border: Border.all(color: SpottoColors.line),
                  ),
                  child: SelectableText(
                    summary,
                    style: const TextStyle(fontSize: 14, height: 1.55, fontWeight: FontWeight.w500),
                  ),
                ),
                if (places.isNotEmpty) ...[
                  const SizedBox(height: 16),
                  const Text('Recommended Places', style: TextStyle(fontSize: 15, fontWeight: FontWeight.w800)),
                  const SizedBox(height: 8),
                  for (final p in places)
                    Padding(
                      padding: const EdgeInsets.only(bottom: 6),
                      child: GuidePlaceCard(place: p),
                    ),
                ],
                const SizedBox(height: 20),
                Row(
                  children: [
                    Expanded(
                      child: OutlinedButton.icon(
                        onPressed: () {
                          Clipboard.setData(ClipboardData(text: '$title\n\n$summary'));
                          ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Summary copied to clipboard!')));
                        },
                        icon: const Icon(Icons.copy_rounded, size: 18),
                        label: const Text('Copy'),
                        style: OutlinedButton.styleFrom(padding: const EdgeInsets.symmetric(vertical: 12)),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: FilledButton.icon(
                        onPressed: () async {
                          final currentList = (localStore.getJsonList('spotto_saved_chat_summaries') ?? []).toList();
                          currentList.insert(0, {
                            'title': title,
                            'summary': summary,
                            'savedAt': DateTime.now().toIso8601String(),
                            'places': [for (final p in places) p.toJson()],
                          });
                          await localStore.setJsonList('spotto_saved_chat_summaries', currentList);
                          if (!mounted) return;
                          if (sheet.mounted) Navigator.pop(sheet);
                          ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
                            content: Text('Summary saved for later use!'),
                            duration: Duration(seconds: 2),
                          ));
                        },
                        icon: const Icon(Icons.bookmark_added_rounded, size: 18),
                        label: const Text('Save for Later'),
                        style: FilledButton.styleFrom(
                          backgroundColor: SpottoColors.accent,
                          padding: const EdgeInsets.symmetric(vertical: 12),
                        ),
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      );
    } catch (e) {
      if (mounted) Navigator.pop(context);
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
    }
  }

  void _showSavedSummariesSheet(List<dynamic> list) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(24))),
      builder: (sheet) => DraggableScrollableSheet(
        initialChildSize: 0.75,
        minChildSize: 0.4,
        maxChildSize: 0.95,
        expand: false,
        builder: (_, scroll) => Padding(
          padding: const EdgeInsets.fromLTRB(20, 16, 20, 20),
          child: ListView(
            controller: scroll,
            children: [
              Center(
                child: Container(
                  width: 40,
                  height: 4,
                  decoration: BoxDecoration(color: SpottoColors.line, borderRadius: BorderRadius.circular(2)),
                ),
              ),
              const SizedBox(height: 16),
              const Row(
                children: [
                  Icon(Icons.folder_special, color: SpottoColors.accent, size: 24),
                  SizedBox(width: 10),
                  Text('Saved Trip Summaries', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w800)),
                ],
              ),
              const SizedBox(height: 16),
              for (var idx = 0; idx < list.length; idx++) ...[
                Builder(builder: (ctx) {
                  final item = (list[idx] as Map).cast<String, dynamic>();
                  final itemTitle = item['title'] as String? ?? 'Trip Summary';
                  final itemSummary = item['summary'] as String? ?? '';
                  final itemPlaces = _parseCards(item['places']);
                  return Container(
                    margin: const EdgeInsets.only(bottom: 14),
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(
                      color: Theme.of(context).scaffoldBackgroundColor,
                      borderRadius: BorderRadius.circular(16),
                      border: Border.all(color: SpottoColors.line),
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          children: [
                            Expanded(
                              child: Text(itemTitle, style: const TextStyle(fontSize: 15, fontWeight: FontWeight.w800)),
                            ),
                            IconButton(
                              icon: const Icon(Icons.copy_rounded, size: 18),
                              tooltip: 'Copy',
                              onPressed: () {
                                Clipboard.setData(ClipboardData(text: '$itemTitle\n\n$itemSummary'));
                                ScaffoldMessenger.of(context).showSnackBar(const SnackBar(content: Text('Summary copied to clipboard!')));
                              },
                            ),
                            IconButton(
                              icon: Icon(Icons.delete_outline, size: 18, color: Theme.of(context).colorScheme.error),
                              tooltip: 'Delete',
                              onPressed: () async {
                                final currentList = (ref.read(localStoreProvider).getJsonList('spotto_saved_chat_summaries') ?? []).toList();
                                if (idx < currentList.length) {
                                  currentList.removeAt(idx);
                                  await ref.read(localStoreProvider).setJsonList('spotto_saved_chat_summaries', currentList);
                                  if (sheet.mounted) Navigator.pop(sheet);
                                  if (!mounted) return;
                                  if (currentList.isNotEmpty) {
                                    _showSavedSummariesSheet(currentList);
                                  }
                                }
                              },
                            ),
                          ],
                        ),
                        const SizedBox(height: 8),
                        SelectableText(
                          itemSummary,
                          style: const TextStyle(fontSize: 13, height: 1.5),
                        ),
                        if (itemPlaces.isNotEmpty) ...[
                          const SizedBox(height: 10),
                          Wrap(
                            spacing: 6,
                            runSpacing: 6,
                            children: [
                              for (final p in itemPlaces)
                                Chip(
                                  label: Text(p.name, style: const TextStyle(fontSize: 11.5)),
                                  visualDensity: VisualDensity.compact,
                                ),
                            ],
                          ),
                        ],
                      ],
                    ),
                  );
                }),
              ],
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _addToCalendar(GuideCalendarAction action, String mode, int? messageIndex) async {
    final name = await applyGuideCalendar(context, ref, action, mode);
    if (name == null || !mounted) return;
    if (messageIndex != null) setState(() => _addedTo[messageIndex] = name);
    final n = action.stops.length;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text('Added ${n == 1 ? action.stops.first.place.name : '$n places'} to $name.'),
      action: SnackBarAction(label: 'Open', onPressed: () => context.go('/t/trip')),
    ));
  }

  Widget _buildBubble(BuildContext context, String content, {required bool isMine}) {
    return Align(
      alignment: isMine ? Alignment.centerRight : Alignment.centerLeft,
      child: Container(
        constraints: BoxConstraints(maxWidth: MediaQuery.of(context).size.width * 0.84),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        decoration: BoxDecoration(
          color: isMine ? SpottoColors.accent : Theme.of(context).colorScheme.surface,
          borderRadius: BorderRadius.only(
            topLeft: const Radius.circular(20),
            topRight: const Radius.circular(20),
            bottomLeft: Radius.circular(isMine ? 20 : 6),
            bottomRight: Radius.circular(isMine ? 6 : 20),
          ),
          boxShadow: const [
            BoxShadow(
              color: Color(0x1F103D3A),
              blurRadius: 12,
              offset: Offset(0, 2),
            ),
          ],
        ),
        child: isMine
            ? Text(
                content,
                style: const TextStyle(
                  fontSize: 14.5,
                  height: 1.5,
                  fontWeight: FontWeight.w500,
                  color: Colors.white,
                ),
              )
            : MarkdownBody(
                data: content,
                selectable: true,
                onTapLink: (text, href, title) {
                  if (href != null && href.isNotEmpty) {
                    launchUrl(Uri.parse(href), mode: LaunchMode.externalApplication);
                  }
                },
                styleSheet: MarkdownStyleSheet.fromTheme(Theme.of(context)).copyWith(
                  p: TextStyle(
                    fontSize: 14.5,
                    height: 1.5,
                    fontWeight: FontWeight.w500,
                    color: Theme.of(context).colorScheme.onSurface,
                  ),
                  strong: TextStyle(
                    fontSize: 14.5,
                    height: 1.5,
                    fontWeight: FontWeight.w800,
                    color: Theme.of(context).colorScheme.onSurface,
                  ),
                  em: TextStyle(
                    fontSize: 14.5,
                    height: 1.5,
                    fontStyle: FontStyle.italic,
                    color: Theme.of(context).colorScheme.onSurface,
                  ),
                  listBullet: const TextStyle(
                    fontSize: 14.5,
                    height: 1.5,
                    fontWeight: FontWeight.w700,
                    color: SpottoColors.accent,
                  ),
                  blockSpacing: 8.0,
                ),
              ),
      ),
    );
  }

  Widget? _payloadCard(int i, GuidePayload p) => switch (p.type) {
        'questions' => GuideQuestionsCard(
            questions: p.questions,
            answered: i < _messages.length - 1 || _busy,
            onSubmit: _send,
          ),
        'plan' => GuidePlanCard(plan: p.plan!, onAdd: (a) => _addToCalendar(a, 'ask', null)),
        'calendar' => GuideCalendarCard(
            action: p.calendar!,
            addedTo: _addedTo[i],
            onChoose: (mode) => _addToCalendar(p.calendar!, mode, i),
          ),
        _ => null,
      };

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final places = ref.watch(placesProvider);
    final userPrefs = ref.watch(prefsProvider);
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
                  IconButton(
                    tooltip: 'Summarize & save for later',
                    icon: const Icon(Icons.bookmark_add_outlined, size: 21, color: SpottoColors.accent),
                    onPressed: _summarizeChat,
                  ),
                  IconButton(
                    tooltip: 'Clear chat history',
                    icon: Icon(Icons.delete_outline_rounded, size: 21, color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.6)),
                    onPressed: _confirmClearChat,
                  ),
                  const SizedBox(width: 4),
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
                  final msgPlaces = m.mine || m.payload != null
                      ? const <GuidePlace>[]
                      : guideCardsFor(m.cards, m.placeIds, places, m.content, userPrefs);
                  final card = m.payload == null ? null : _payloadCard(i, m.payload!);
                  if (m.mine) {
                    return Padding(
                      padding: const EdgeInsets.only(bottom: 12),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.end,
                        children: [
                          _buildBubble(context, m.content, isMine: true),
                        ],
                      ),
                    );
                  }

                  // Assistant turn with recommended places: separate into intro bubble, unified place cards, outro bubble
                  if (msgPlaces.isNotEmpty) {
                    final turn = parsePlaceTurnContent(m.content, msgPlaces);
                    final hasIntro = turn.intro.trim().isNotEmpty;
                    final hasOutro = turn.outro.trim().isNotEmpty;

                    // Fallback: if no intro, outro, or place descriptions were parsed, show full content
                    final fallbackToFull = !hasIntro && !hasOutro && turn.placeDescriptions.isEmpty;

                    return Padding(
                      padding: const EdgeInsets.only(bottom: 12),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          if (fallbackToFull)
                            _buildBubble(context, m.content, isMine: false)
                          else if (hasIntro)
                            _buildBubble(context, turn.intro, isMine: false),
                          for (final place in msgPlaces)
                            Padding(
                              padding: const EdgeInsets.only(top: 8),
                              child: GuidePlaceCard(
                                place: place,
                                description: place.description.isNotEmpty
                                    ? place.description
                                    : (turn.placeDescriptions[place.id] ?? turn.placeDescriptions[place.name]),
                              ),
                            ),
                          if (!fallbackToFull && hasOutro)
                            Padding(
                              padding: const EdgeInsets.only(top: 8),
                              child: _buildBubble(context, turn.outro, isMine: false),
                            ),
                          if (card != null)
                            Padding(
                              padding: const EdgeInsets.only(top: 8),
                              child: card,
                            ),
                        ],
                      ),
                    );
                  }

                  // Normal assistant turn (Q&A, itinerary, questions)
                  return Padding(
                    padding: const EdgeInsets.only(bottom: 12),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        _buildBubble(context, m.content, isMine: false),
                        if (card != null)
                          Padding(
                            padding: const EdgeInsets.only(top: 8),
                            child: card,
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
            // suggestion chips throughout conversation
            if (!_busy && _activeSuggestions.isNotEmpty)
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
                child: SingleChildScrollView(
                  scrollDirection: Axis.horizontal,
                  physics: const BouncingScrollPhysics(),
                  child: Row(
                    children: [
                      for (final s in _activeSuggestions)
                        Padding(
                          padding: const EdgeInsets.only(right: 8),
                          child: PressScale(
                            scale: 0.95,
                            onTap: () => _send(s),
                            child: Container(
                              padding: const EdgeInsets.symmetric(
                                  horizontal: 13, vertical: 8),
                              decoration: BoxDecoration(
                                color: Theme.of(context).colorScheme.surface,
                                borderRadius: BorderRadius.circular(999),
                                border: Border.all(color: SpottoColors.line, width: 1.5),
                                boxShadow: const [
                                  BoxShadow(color: Color(0x14103D3A), blurRadius: 8, offset: Offset(0, 2)),
                                ],
                              ),
                              child: Text(
                                s,
                                style: TextStyle(
                                  fontSize: 12.5,
                                  fontWeight: FontWeight.w700,
                                  color: Theme.of(context).colorScheme.onSurface,
                                ),
                              ),
                            ),
                          ),
                        ),
                    ],
                  ),
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
