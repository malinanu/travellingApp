// Canned local-guide engine — stands in for the prototype's live Claude
// call: keyword matching over the real places, aware of weather and style.

import 'dart:math' as math;

import 'places.dart';
import '../models/models.dart';

const guideSuggestions = [
  'Where can I work today with a sea view?',
  'Best surf spot for a total beginner?',
  "Where's the move for sunset tonight?",
  'Any deals on right now?',
];

const guideSuggestionsRainy = [
  'Where can I work today with a sea view?',
  "What's good to do in the rain?",
  'Any indoor workshops nearby?',
  'Any deals on right now?',
];

String guideGreeting(String styleLabel, String weather) => weather == 'rainy'
    ? "Hey! It's raining over Weligama right now 🌧️ — I've bumped indoor spots to "
        "the top. Ask me where to go and I'll keep you dry (and caffeinated)."
    : "Hey! I'm your Spotto guide for Weligama 🌴 I've set things up for a "
        '${styleLabel.toLowerCase()} — ask me where to work, surf, eat or catch '
        "the sunset and I'll point you to the right spot.";

Place? guideFindPlace(String text) {
  final low = text.toLowerCase();
  for (final p in places) {
    if (low.contains(p.name.toLowerCase()) ||
        low.contains('${p.name.split(' ').first.toLowerCase()} ')) {
      return p;
    }
  }
  return null;
}

bool _any(String q, List<String> words) => words.any(q.contains);

/// Reply after a short "typing" delay, like the real call.
Future<String> guideReply(String question, {required String weather, String? style}) async {
  final delay = 700 + math.Random().nextInt(500);
  await Future.delayed(Duration(milliseconds: delay));
  return _reply(question.toLowerCase(), weather, style);
}

String _reply(String q, String weather, String? style) {
  final rainy = weather == 'rainy';

  if (_any(q, ['work', 'wifi', 'wi-fi', 'laptop', 'cowork', 'focus', 'desk'])) {
    return 'Driftwood Coffee & Co-Work is your spot — fastest fibre in the bay '
        '(9.2/10, 88 Mbps), a proper co-work loft and sea-view desks on the upper '
        "deck. There's a free flat white with any day pass right now too.";
  }
  if (_any(q, ['surf', 'wave', 'board', 'lesson', 'beginner'])) {
    return rainy
        ? "I'd skip the surf while this shower passes — the break gets messy. If "
            'you want to stay busy, Clay & Co Pottery Studio is dry, hands-on and '
            'five minutes away.'
        : 'Weligama Bay Surf School, no contest — the gentle beach-break here is '
            "exactly why beginners come to Weligama. They're doing 20% off group "
            'lessons today; dawn patrol is the calmest water.';
  }
  if (_any(q, ['sunset', 'drink', 'cocktail', 'bar', 'night', 'evening', 'dj'])) {
    return rainy
        ? 'Hangtime Rooftop is the sunset spot, but the roof is open to the sky — '
            'wait for the rain to ease. If it clears by 5, happy hour runs 5–7 PM '
            'with 2-for-1 arrack cocktails.'
        : "Hangtime Rooftop & Sunset Bar — palm-fringed roof facing due west, it's "
            'where locals point when the sky turns gold. Get there for happy hour, '
            '5–7 PM, 2-for-1 arrack cocktails.';
  }
  if (_any(q, ['rain', 'indoor', 'wet', 'monsoon', 'inside'])) {
    return 'Two great dry plans: Clay & Co Pottery Studio does a drop-in '
        'wheel-throwing taster for Rs 1,500, and Sea Salt Batik Atelier lets you '
        'wax your own batik under the canopy. Both are proper local-artisan spots.';
  }
  if (_any(q, ['deal', 'offer', 'discount', 'cheap', 'free'])) {
    final active = places.where((p) => p.deal.active).toList();
    final lines = active.map((p) => '${p.name}: ${p.deal.label}').join('. ');
    return 'Live right now — $lines. Claim in the app and show the code at the counter.';
  }
  if (_any(q, ['eat', 'food', 'lunch', 'dinner', 'breakfast', 'hungry', 'menu'])) {
    return 'Driftwood does a great coconut roti & curd bowl for breakfast, and '
        "Hangtime's hot butter cuttlefish is the rooftop classic for the evening. "
        'Both menus are in the app.';
  }
  if (_any(q, ['pottery', 'clay', 'craft', 'workshop', 'batik', 'artisan', 'quest'])) {
    return 'Check in at Clay & Co Pottery Studio and Sea Salt Batik Atelier — '
        "they're both working artisan studios, and checking in at two completes "
        "today's quest for a free Ceylon tea.";
  }
  if (_any(q, ['coffee', 'flat white', 'espresso', 'cafe', 'café'])) {
    return 'Driftwood Coffee & Co-Work pulls the best single-origin pour-over in '
        "the bay (Rs 750), and there's a free flat white with any day pass today.";
  }
  if (_any(q, ['beach', 'swim', 'sea', 'sand'])) {
    return rainy
        ? 'The beach is moody in the rain — beautiful for a walk, not a swim. Give '
            'it an hour; monsoon showers here pass quickly.'
        : 'Weligama Beach Point is right on the bay — calm, sandy and walkable. '
            'For the classic palm-on-a-hill photo, tuk-tuk down to Coconut Tree '
            'Hill around 5 PM.';
  }
  if (_any(q, ['hello', 'hi ', 'hey', 'morning'])) {
    return 'Hey hey! What are you in the mood for — work, surf, food or something '
        'creative? I know this bay inside out.';
  }

  // friendly fallback (the prototype's offline line)
  return "Good question — I'd start at Driftwood Coffee & Co-Work: the bay's "
      'fastest Wi-Fi (9.2/10), great coffee, and everything else is a short walk '
      'from there. Ask me about surf, sunset or deals for more.';
}
