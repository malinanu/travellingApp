// Spot detail bottom sheet — port of BusinessDrawer.jsx: illustration
// header, stats, deal claim, vibe voting, artisan check-in, menu, actions.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../core/api/api_client.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/illustration.dart';
import '../../../core/widgets/pill.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/sheet_scaffold.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../core/widgets/stat_chip.dart';
import '../../../data/mock/places.dart' as data;
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';

Future<void> showBusinessDrawer(BuildContext context, Place place) {
  return showModalBottomSheet(
    context: context,
    isScrollControlled: true,
    backgroundColor: Colors.transparent,
    builder: (context) => DraggableScrollableSheet(
      initialChildSize: 0.72,
      maxChildSize: 0.95,
      minChildSize: 0.5,
      expand: false,
      builder: (context, controller) => SheetScaffold(
        child: SingleChildScrollView(
          controller: controller,
          child: BusinessDrawerBody(place: place),
        ),
      ),
    ),
  );
}

class BusinessDrawerBody extends ConsumerWidget {
  final Place place;
  const BusinessDrawerBody({super.key, required this.place});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final claimedCode = ref.watch(claimsProvider)[place.id];

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // illustration header
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 6, 16, 0),
          child: ClipRRect(
            borderRadius: BorderRadius.circular(22),
            child: SizedBox(
              height: 170,
              child: Stack(
                fit: StackFit.expand,
                children: [
                  // Real uploaded cover photo when the vendor has one, else
                  // fall back to the illustration.
                  if (place.cover.isNotEmpty)
                    Image.network(
                      place.cover,
                      fit: BoxFit.cover,
                      errorBuilder: (_, __, ___) => Illustration(kind: place.photo),
                    )
                  else
                    Illustration(kind: place.photo),
                  // Bottom scrim so the white name/subtitle stays readable
                  // over a photographic cover.
                  if (place.cover.isNotEmpty)
                    const DecoratedBox(
                      decoration: BoxDecoration(
                        gradient: LinearGradient(
                          begin: Alignment.topCenter,
                          end: Alignment.bottomCenter,
                          colors: [Colors.transparent, Colors.black54],
                        ),
                      ),
                    ),
                  Positioned(
                    left: 16,
                    bottom: 12,
                    right: 16,
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(place.name, style: text.titleLarge?.copyWith(color: Colors.white)),
                        Text('${place.type} · ${place.distance} · ${place.priceLevel}',
                            style: text.bodyMedium
                                ?.copyWith(color: Colors.white.withValues(alpha: 0.9))),
                      ],
                    ),
                  ),
                  Positioned(
                    right: 12,
                    top: 12,
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                      decoration: BoxDecoration(
                        color: Colors.white,
                        borderRadius: BorderRadius.circular(999),
                      ),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          const Icon(Icons.star, size: 16, color: SpottoColors.sun),
                          const SizedBox(width: 4),
                          Text('${place.rating}', style: text.labelLarge),
                          Text(' (${place.reviews})', style: text.labelSmall),
                        ],
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
        // Uploaded gallery photos + a tappable promo-video thumbnail.
        if (place.gallery.isNotEmpty || place.promoVideo.isNotEmpty)
          SizedBox(
            height: 88,
            child: ListView(
              scrollDirection: Axis.horizontal,
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
              children: [
                if (place.promoVideo.isNotEmpty)
                  Padding(
                    padding: const EdgeInsets.only(right: 8),
                    child: PressScale(
                      onTap: () => launchUrl(Uri.parse(place.promoVideo),
                          mode: LaunchMode.externalApplication),
                      child: Container(
                        width: 118,
                        height: 88,
                        decoration: BoxDecoration(
                          color: Colors.black87,
                          borderRadius: BorderRadius.circular(12),
                        ),
                        child: const Center(
                          child: Icon(Icons.play_circle_fill, color: Colors.white, size: 34),
                        ),
                      ),
                    ),
                  ),
                for (final url in place.gallery)
                  Padding(
                    padding: const EdgeInsets.only(right: 8),
                    child: ClipRRect(
                      borderRadius: BorderRadius.circular(12),
                      child: Image.network(url,
                          width: 118,
                          height: 88,
                          fit: BoxFit.cover,
                          errorBuilder: (_, __, ___) => const SizedBox(width: 118, height: 88)),
                    ),
                  ),
              ],
            ),
          ),
        Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // stat chips
              Row(
                children: [
                  Expanded(
                    child: StatChip(
                        icon: Icons.wifi,
                        value: '${place.wifi.score}',
                        label: '${place.wifi.speed} · ${place.wifi.label}'),
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: StatChip(
                        icon: Icons.schedule,
                        value: place.openNow ? 'Open' : 'Closed',
                        label: place.hours,
                        iconColor: place.openNow ? SpottoColors.accent2 : SpottoColors.muted),
                  ),
                ],
              ),
              const SizedBox(height: 14),

              // deal claim
              if (place.deal.active) ...[
                _DealCard(place: place, claimedCode: claimedCode),
                const SizedBox(height: 14),
              ],

              // vibe check
              _VibeCheck(placeId: place.id),
              const SizedBox(height: 14),

              // artisan quest check-in
              if (place.artisan) ...[
                _CheckInButton(place: place),
                const SizedBox(height: 14),
              ],

              // tags
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [for (final t in place.tags) Pill(label: t)],
              ),
              const SizedBox(height: 14),
              Text(place.blurb, style: text.bodyLarge),

              // menu
              if (place.menu.isNotEmpty) ...[
                const SizedBox(height: 18),
                Text('Menu', style: text.titleMedium),
                const SizedBox(height: 8),
                for (final section in place.menu) ...[
                  Text(section.name,
                      style: text.labelMedium?.copyWith(color: SpottoColors.accent)),
                  const SizedBox(height: 6),
                  for (final item in section.items)
                    Padding(
                      padding: const EdgeInsets.only(bottom: 10),
                      child: Row(
                        children: [
                          ClipRRect(
                            borderRadius: BorderRadius.circular(12),
                            child: item.img.isNotEmpty
                                ? Image.network(item.img,
                                    width: 40,
                                    height: 40,
                                    fit: BoxFit.cover,
                                    errorBuilder: (_, __, ___) => _MenuIconTile(icon: item.icon))
                                : _MenuIconTile(icon: item.icon),
                          ),
                          const SizedBox(width: 12),
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(item.name, style: text.bodyLarge),
                                Text(item.desc, style: text.bodySmall),
                              ],
                            ),
                          ),
                          Text('Rs ${item.price}', style: text.labelLarge),
                        ],
                      ),
                    ),
                  const SizedBox(height: 6),
                ],
              ],

              const SizedBox(height: 18),
              _ReviewsSection(place: place),

              if (!place.contact.isEmpty) ...[
                const SizedBox(height: 18),
                _ContactLinks(contact: place.contact),
              ],

              // bottom actions
              const SizedBox(height: 18),
              Row(
                children: [
                  Expanded(
                    child: FilledButton.icon(
                      style: FilledButton.styleFrom(
                        backgroundColor: SpottoColors.ink,
                        shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(16)),
                      ),
                      onPressed: () => launchUrl(
                        Uri.parse(
                            'https://www.google.com/maps/search/?api=1&query=${Uri.encodeComponent('${place.name} Weligama')}'),
                        mode: LaunchMode.externalApplication,
                      ),
                      icon: const Icon(Icons.near_me, size: 17),
                      label: const Text('Directions'),
                    ),
                  ),
                  const SizedBox(width: 10),
                  PressScale(
                    scale: 0.95,
                    onTap: () => ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(content: Text('Saved to bookmarks'))),
                    child: Container(
                      width: 56,
                      height: 50,
                      decoration: BoxDecoration(
                        color: SpottoColors.sand,
                        borderRadius: BorderRadius.circular(16),
                      ),
                      child: const Icon(Icons.bookmark_outline,
                          size: 19, color: SpottoColors.ink),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 24),
            ],
          ),
        ),
      ],
    );
  }
}

class _VibeCheck extends ConsumerStatefulWidget {
  final String placeId;
  const _VibeCheck({required this.placeId});

  @override
  ConsumerState<_VibeCheck> createState() => _VibeCheckState();
}

class _VibeCheckState extends ConsumerState<_VibeCheck> {
  String? _justVoted;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final votes = ref.watch(vibeVotesProvider)[widget.placeId] ?? const <String, int>{};
    final total = data.vibes.fold(0, (a, v) => a + (votes[v.id] ?? 0));
    final lead = [...data.vibes]
      ..sort((a, b) => (votes[b.id] ?? 0).compareTo(votes[a.id] ?? 0));

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: SpottoColors.sand,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Icon(Icons.monitor_heart_outlined, size: 14, color: SpottoColors.muted),
              const SizedBox(width: 5),
              Text('CURRENT VIBE',
                  style: text.labelSmall
                      ?.copyWith(color: SpottoColors.muted, letterSpacing: 0.6)),
              const Spacer(),
              if (total > 0)
                Row(
                  children: [
                    Icon(spottoIcon(lead.first.icon),
                        size: 13, color: Color(lead.first.color)),
                    const SizedBox(width: 4),
                    Text('Mostly ${lead.first.label.toLowerCase()} · $total votes',
                        style: text.labelSmall?.copyWith(color: Color(lead.first.color))),
                  ],
                ),
            ],
          ),
          const SizedBox(height: 10),
          Row(
            children: [
              for (final (i, v) in data.vibes.indexed) ...[
                if (i > 0) const SizedBox(width: 8),
                Expanded(
                  child: PressScale(
                    scale: 0.95,
                    onTap: () {
                      ref.read(vibeVotesProvider.notifier).vote(widget.placeId, v.id);
                      setState(() => _justVoted = v.id);
                    },
                    child: _vibeOption(text, v, votes, total),
                  ),
                ),
              ],
            ],
          ),
        ],
      ),
    );
  }

  Widget _vibeOption(TextTheme text, VibeTag v, Map<String, int> votes, int total) {
    final count = votes[v.id] ?? 0;
    final pct = total > 0 ? count / total : 0.0;
    final mine = _justVoted == v.id;
    final color = Color(v.color);

    return Container(
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: mine ? color : Colors.white,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: mine ? color : SpottoColors.line, width: 1.5),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(spottoIcon(v.icon), size: 18, color: mine ? Colors.white : color),
          const SizedBox(height: 4),
          Text(v.label,
              style: TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w800,
                  color: mine ? Colors.white : SpottoColors.ink)),
          const SizedBox(height: 6),
          ClipRRect(
            borderRadius: BorderRadius.circular(999),
            child: Container(
              height: 6,
              color: mine
                  ? Colors.white.withValues(alpha: 0.3)
                  : SpottoColors.ink.withValues(alpha: 0.08),
              child: AnimatedFractionallySizedBox(
                duration: const Duration(milliseconds: 500),
                alignment: Alignment.centerLeft,
                widthFactor: pct,
                child: Container(color: mine ? Colors.white : color),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _CheckInButton extends ConsumerWidget {
  final Place place;
  const _CheckInButton({required this.place});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final didCheckIn = ref.watch(checkInsProvider).contains(place.id);

    return PressScale(
      scale: 0.98,
      onTap: didCheckIn
          ? null
          : () => ref.read(checkInsProvider.notifier).checkIn(place.id),
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.symmetric(vertical: 12),
        decoration: BoxDecoration(
          color: didCheckIn ? SpottoColors.accentTint : SpottoColors.sun,
          borderRadius: BorderRadius.circular(16),
        ),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(didCheckIn ? Icons.verified_outlined : Icons.explore,
                size: 17,
                color: didCheckIn ? SpottoColors.accent : const Color(0xFF5A3B00)),
            const SizedBox(width: 6),
            Text(
              didCheckIn
                  ? 'Checked in — quest progress saved'
                  : 'Check in for Daily Quest',
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w800,
                color: didCheckIn ? SpottoColors.accent : const Color(0xFF5A3B00),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _DealCard extends ConsumerStatefulWidget {
  final Place place;
  final String? claimedCode;
  const _DealCard({required this.place, required this.claimedCode});

  @override
  ConsumerState<_DealCard> createState() => _DealCardState();
}

class _DealCardState extends ConsumerState<_DealCard> {
  @override
  void initState() {
    super.initState();
    // Real traveller impression, once per time this card is actually shown
    // (initState runs once per widget instance, not on every rebuild) —
    // lets the vendor portal show a genuine "Views" count per offer.
    final offerId = widget.place.deal.id;
    if (offerId != null) _trackView(offerId);
  }

  Future<void> _trackView(String offerId) async {
    try {
      await ref.read(apiProvider).dio.post('/offers/$offerId/view');
    } catch (_) {
      // Fire-and-forget — a failed view ping shouldn't affect the UI.
    }
  }

  @override
  Widget build(BuildContext context) {
    final place = widget.place;
    final claimedCode = widget.claimedCode;
    final text = Theme.of(context).textTheme;
    final claimed = claimedCode != null;

    return AnimatedContainer(
      duration: const Duration(milliseconds: 250),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: claimed ? SpottoColors.ink : const Color(0xFFFDF3D7),
        borderRadius: BorderRadius.circular(18),
        border: Border.all(
            color: claimed ? SpottoColors.ink : const Color(0xFFF2DCA0), width: 1.5),
      ),
      child: claimed
          ? Row(
              crossAxisAlignment: CrossAxisAlignment.center,
              children: [
                // White backing is required for the QR to actually scan —
                // this card's background is SpottoColors.ink (dark), and a
                // QR rendered directly on a dark background has no contrast.
                Container(
                  padding: const EdgeInsets.all(6),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: QrImageView(data: claimedCode, size: 64, gapless: false),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text('Show at counter',
                          style: text.labelMedium
                              ?.copyWith(color: SpottoColors.paleTeal)),
                      const SizedBox(height: 4),
                      Text(claimedCode,
                          style: text.headlineSmall
                              ?.copyWith(color: Colors.white, letterSpacing: 2)),
                      const SizedBox(height: 4),
                      Text(place.deal.label ?? '',
                          style: text.bodySmall
                              ?.copyWith(color: SpottoColors.paleTeal)),
                    ],
                  ),
                ),
              ],
            )
          : Row(
              children: [
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                  decoration: BoxDecoration(
                    color: SpottoColors.sun,
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Text(place.deal.tag ?? 'Deal',
                      style: text.labelSmall?.copyWith(color: SpottoColors.ink)),
                ),
                const SizedBox(width: 10),
                Expanded(child: Text(place.deal.label ?? '', style: text.bodyMedium)),
                FilledButton(
                  style: FilledButton.styleFrom(
                    backgroundColor: SpottoColors.sun,
                    foregroundColor: SpottoColors.ink,
                    padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
                  ),
                  onPressed: () async {
                    final messenger = ScaffoldMessenger.of(context);
                    try {
                      await ref.read(claimsProvider.notifier).claim(
                            place.id,
                            offerId: place.deal.id,
                            offerLabel: place.deal.label ?? '',
                            offerValue: place.deal.tag ?? '',
                          );
                    } catch (e) {
                      messenger.showSnackBar(
                          SnackBar(content: Text(ApiClient.errorMessage(e))));
                    }
                  },
                  child: const Text('Claim'),
                ),
              ],
            ),
    );
  }
}

/// Tappable phone/website/social row — only the fields the vendor actually
/// filled in are shown, so this renders nothing for the many businesses
/// that haven't set any contact info yet (see !place.contact.isEmpty guard
/// at the call site).
class _ContactLinks extends StatelessWidget {
  final ContactInfo contact;
  const _ContactLinks({required this.contact});

  Future<void> _open(String url) =>
      launchUrl(Uri.parse(url), mode: LaunchMode.externalApplication);

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final chips = <Widget>[];

    void addChip(String? value, IconData icon, String label, String Function(String) toUrl) {
      if (value == null || value.isEmpty) return;
      chips.add(PressScale(
        scale: 0.96,
        onTap: () => _open(toUrl(value)),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
          decoration: BoxDecoration(
            color: SpottoColors.sand,
            borderRadius: BorderRadius.circular(14),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon, size: 16, color: SpottoColors.ink),
              const SizedBox(width: 6),
              Text(label, style: text.labelMedium?.copyWith(color: SpottoColors.ink)),
            ],
          ),
        ),
      ));
    }

    addChip(contact.phone, Icons.call_outlined, 'Call', (v) => 'tel:$v');
    addChip(contact.website, Icons.language, 'Website',
        (v) => v.startsWith('http') ? v : 'https://$v');
    addChip(contact.instagram, Icons.camera_alt_outlined, 'Instagram',
        (v) => v.startsWith('http') ? v : 'https://instagram.com/${v.replaceAll('@', '')}');
    addChip(contact.facebook, Icons.groups_outlined, 'Facebook',
        (v) => v.startsWith('http') ? v : 'https://facebook.com/$v');

    if (chips.isEmpty) return const SizedBox.shrink();
    return Wrap(spacing: 8, runSpacing: 8, children: chips);
  }
}

/// Reviews: existing list (rating + text) plus a form to submit or edit
/// your own. Ratings of 1-2 count against the vendor for the admin map's
/// red-flag threshold — this is the real write path behind that feature.
class _ReviewsSection extends ConsumerStatefulWidget {
  final Place place;
  const _ReviewsSection({required this.place});

  @override
  ConsumerState<_ReviewsSection> createState() => _ReviewsSectionState();
}

class _ReviewsSectionState extends ConsumerState<_ReviewsSection> {
  int _draftRating = 0;
  final _textController = TextEditingController();
  bool _submitting = false;

  @override
  void dispose() {
    _textController.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_draftRating == 0 || _submitting) return;
    setState(() => _submitting = true);
    final messenger = ScaffoldMessenger.of(context);
    try {
      await submitReview(ref, widget.place.id, _draftRating, _textController.text.trim());
      _textController.clear();
      setState(() => _draftRating = 0);
    } catch (e) {
      messenger.showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final reviewsAsync = ref.watch(placeReviewsProvider(widget.place.id));

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Reviews', style: text.titleMedium),
        const SizedBox(height: 10),

        // write/edit form
        Container(
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            color: SpottoColors.sand,
            borderRadius: BorderRadius.circular(16),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  for (var i = 1; i <= 5; i++)
                    PressScale(
                      scale: 0.85,
                      onTap: () => setState(() => _draftRating = i),
                      child: Padding(
                        padding: const EdgeInsets.only(right: 4),
                        child: Icon(
                          i <= _draftRating ? Icons.star : Icons.star_border,
                          color: SpottoColors.sun,
                          size: 26,
                        ),
                      ),
                    ),
                ],
              ),
              const SizedBox(height: 10),
              TextField(
                controller: _textController,
                maxLines: 2,
                decoration: const InputDecoration(
                  hintText: 'Share how it went (optional)',
                  filled: true,
                  fillColor: Colors.white,
                ),
              ),
              const SizedBox(height: 10),
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: _draftRating == 0 || _submitting ? null : _submit,
                  child: Text(_submitting ? 'Posting…' : 'Post review'),
                ),
              ),
            ],
          ),
        ),
        const SizedBox(height: 12),

        reviewsAsync.when(
          data: (reviews) {
            if (reviews.isEmpty) {
              return Text('No reviews yet — be the first.',
                  style: text.bodySmall?.copyWith(color: SpottoColors.muted));
            }
            return Column(
              children: [
                for (final r in reviews) ...[
                  _ReviewTile(review: r),
                  const SizedBox(height: 8),
                ],
              ],
            );
          },
          loading: () => const Padding(
            padding: EdgeInsets.symmetric(vertical: 12),
            child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
          ),
          error: (_, __) => Text('Couldn\'t load reviews.',
              style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
        ),
      ],
    );
  }
}

class _ReviewTile extends StatelessWidget {
  final Review review;
  const _ReviewTile({required this.review});

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: SpottoColors.ink.withValues(alpha: 0.06)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Row(
                children: [
                  for (var i = 1; i <= 5; i++)
                    Icon(
                      i <= review.rating ? Icons.star : Icons.star_border,
                      color: SpottoColors.sun,
                      size: 15,
                    ),
                ],
              ),
              if (review.travellerName != null && review.travellerName!.isNotEmpty) ...[
                const SizedBox(width: 8),
                Text(review.travellerName!,
                    style: text.labelSmall?.copyWith(fontWeight: FontWeight.w800)),
              ],
            ],
          ),
          if (review.text.isNotEmpty) ...[
            const SizedBox(height: 6),
            Text(review.text, style: text.bodySmall),
          ],
        ],
      ),
    );
  }
}

/// Fallback icon tile shown for a menu item that has no uploaded photo.
class _MenuIconTile extends StatelessWidget {
  final String icon;
  const _MenuIconTile({required this.icon});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 40,
      height: 40,
      decoration: BoxDecoration(
        color: SpottoColors.accentTint,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Icon(spottoIcon(icon), size: 20, color: SpottoColors.accent),
    );
  }
}
