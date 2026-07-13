// Offers — port of OfferCreator.jsx: create/edit form (bottom sheet on
// mobile), filter chips and offer lifecycle management.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/charts.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/sheet_scaffold.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/mock/vendor_seed.dart';
import '../../../data/repositories/repositories.dart';
import '../widgets/vendor_common.dart';

class VendorOffersScreen extends ConsumerStatefulWidget {
  const VendorOffersScreen({super.key});

  @override
  ConsumerState<VendorOffersScreen> createState() => _VendorOffersScreenState();
}

class _VendorOffersScreenState extends ConsumerState<VendorOffersScreen> {
  String _filter = 'all';
  String? _confirmDeleteId;

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final offers = ref.watch(offersProvider);
    final filtered = _filter == 'all'
        ? offers
        : offers.where((o) => o.status == _filter).toList();

    return ListView(
      padding: const EdgeInsets.fromLTRB(20, 8, 20, 24),
      children: [
        SizedBox(
          width: double.infinity,
          child: FilledButton.icon(
            onPressed: () => showOfferEditor(context),
            icon: const Icon(Icons.add, size: 18),
            label: const Text('Create a new offer'),
          ),
        ),
        const SizedBox(height: 16),
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: Row(
            children: [
              for (final f in const [
                (id: 'all', label: 'All'),
                (id: 'live', label: 'Live'),
                (id: 'ending', label: 'Ending'),
                (id: 'draft', label: 'Drafts'),
                (id: 'ended', label: 'Ended'),
              ]) ...[
                PressScale(
                  scale: 0.95,
                  onTap: () => setState(() => _filter = f.id),
                  child: Container(
                    margin: const EdgeInsets.only(right: 8),
                    padding:
                        const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
                    decoration: BoxDecoration(
                      color: _filter == f.id ? SpottoColors.accent : Colors.white,
                      borderRadius: BorderRadius.circular(999),
                      border: Border.all(
                          color: _filter == f.id
                              ? SpottoColors.accent
                              : SpottoColors.line,
                          width: 1.5),
                    ),
                    child: Text(f.label,
                        style: TextStyle(
                            fontSize: 13,
                            fontWeight: FontWeight.w800,
                            color: _filter == f.id
                                ? Colors.white
                                : SpottoColors.ink)),
                  ),
                ),
              ],
            ],
          ),
        ),
        const SizedBox(height: 14),
        if (filtered.isEmpty)
          Padding(
            padding: const EdgeInsets.only(top: 32),
            child: Text('No offers here yet.',
                textAlign: TextAlign.center,
                style: text.bodyMedium?.copyWith(color: SpottoColors.muted)),
          ),
        for (final o in filtered) ...[
          _offerCard(o),
          const SizedBox(height: 12),
        ],
      ],
    );
  }

  Widget _offerCard(Offer o) {
    final text = Theme.of(context).textTheme;
    final notifier = ref.read(offersProvider.notifier);
    final confirming = _confirmDeleteId == o.id;

    return VendorCard(
      padding: const EdgeInsets.all(14),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(o.title,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: text.titleSmall?.copyWith(fontSize: 14.5)),
              ),
              const SizedBox(width: 8),
              StatusChip(status: o.status),
            ],
          ),
          const SizedBox(height: 6),
          Text('${o.type} · ${o.value} · ${o.start} → ${o.end}',
              style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
          const SizedBox(height: 10),
          Row(
            children: [
              Donut(
                value: o.claimed,
                max: o.limit,
                size: 40,
                stroke: 6,
                color: o.status == 'ending'
                    ? const Color(0xFFF0682F)
                    : SpottoColors.accent,
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Text('${o.claimed}/${o.limit} claimed · ${o.redeemed} redeemed',
                    style: text.bodySmall?.copyWith(color: SpottoColors.inkSoft)),
              ),
            ],
          ),
          const SizedBox(height: 10),
          Row(
            children: [
              _smallAction('Edit', Icons.edit_outlined,
                  () => showOfferEditor(context, offer: o)),
              const SizedBox(width: 8),
              if (o.status == 'live' || o.status == 'ending')
                _smallAction(
                    'End now', Icons.stop_circle_outlined,
                    () => notifier.setStatus(o.id, 'ended'))
              else if (o.status == 'draft' || o.status == 'ended')
                _smallAction('Go live', Icons.play_circle_outline,
                    () => notifier.setStatus(o.id, 'live')),
              const Spacer(),
              PressScale(
                scale: 0.95,
                onTap: () {
                  if (confirming) {
                    notifier.remove(o.id);
                    setState(() => _confirmDeleteId = null);
                  } else {
                    setState(() => _confirmDeleteId = o.id);
                    Future.delayed(const Duration(seconds: 2), () {
                      if (mounted && _confirmDeleteId == o.id) {
                        setState(() => _confirmDeleteId = null);
                      }
                    });
                  }
                },
                child: Container(
                  padding:
                      const EdgeInsets.symmetric(horizontal: 12, vertical: 7),
                  decoration: BoxDecoration(
                    color: confirming ? vendorRed : vendorRedBg,
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Text(confirming ? 'Sure?' : 'Delete',
                      style: TextStyle(
                          fontSize: 12,
                          fontWeight: FontWeight.w800,
                          color: confirming ? Colors.white : vendorRed)),
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _smallAction(String label, IconData icon, VoidCallback onTap) => PressScale(
        scale: 0.95,
        onTap: onTap,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 7),
          decoration: BoxDecoration(
            color: SpottoColors.sand,
            borderRadius: BorderRadius.circular(10),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon, size: 14, color: SpottoColors.ink),
              const SizedBox(width: 5),
              Text(label,
                  style: const TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w800,
                      color: SpottoColors.ink)),
            ],
          ),
        ),
      );
}

/// Create/edit offer sheet. Pass [offer] to edit.
Future<void> showOfferEditor(BuildContext context, {Offer? offer}) {
  return showModalBottomSheet(
    context: context,
    isScrollControlled: true,
    builder: (_) => _OfferEditor(offer: offer),
  );
}

class _OfferEditor extends ConsumerStatefulWidget {
  final Offer? offer;
  const _OfferEditor({this.offer});

  @override
  ConsumerState<_OfferEditor> createState() => _OfferEditorState();
}

class _OfferEditorState extends ConsumerState<_OfferEditor> {
  late final TextEditingController _title =
      TextEditingController(text: widget.offer?.title ?? '');
  late final TextEditingController _value =
      TextEditingController(text: widget.offer?.value ?? '');
  late String _typeId = switch (widget.offer?.type) {
    'Percentage' => 'pct',
    'Amount' => 'amount',
    'Freebie' => 'freebie',
    'Bundle' => 'bundle',
    _ => 'pct',
  };
  late String _duration = 'today';
  late bool _unlimited = false;
  late int _limit = widget.offer?.limit ?? 50;

  static const _durations = [
    (id: 'today', label: 'Today', start: 'Today 07:00', end: 'Today 22:00'),
    (id: 'weekend', label: 'This weekend', start: 'Sat', end: 'Sun'),
    (id: 'week', label: 'All week', start: 'Mon', end: 'Sun'),
  ];

  @override
  void dispose() {
    _title.dispose();
    _value.dispose();
    super.dispose();
  }

  void _save(String status) {
    final d = _durations.firstWhere((x) => x.id == _duration);
    final type = discountTypes.firstWhere((t) => t.id == _typeId);
    final typeLabel = switch (type.id) {
      'pct' => 'Percentage',
      'amount' => 'Amount',
      'freebie' => 'Freebie',
      _ => 'Bundle',
    };
    final notifier = ref.read(offersProvider.notifier);
    final existing = widget.offer;
    if (existing != null) {
      notifier.replace(existing.copyWith(
        title: _title.text.trim(),
        type: typeLabel,
        value: _value.text.trim(),
        status: status == 'draft' ? 'draft' : existing.status,
        limit: _unlimited ? 9999 : _limit,
      ));
    } else {
      notifier.add(Offer(
        id: 'o${DateTime.now().millisecondsSinceEpoch}',
        title: _title.text.trim().isEmpty ? 'Untitled offer' : _title.text.trim(),
        type: typeLabel,
        value: _value.text.trim().isEmpty ? type.hint : _value.text.trim(),
        status: status,
        start: status == 'draft' ? '—' : d.start,
        end: status == 'draft' ? '—' : d.end,
        claimed: 0,
        limit: _unlimited ? 9999 : _limit,
        redeemed: 0,
      ));
    }
    Navigator.of(context).pop();
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;

    return SheetScaffold(
      padding:
          EdgeInsets.fromLTRB(20, 4, 20, 24 + MediaQuery.of(context).viewInsets.bottom),
      child: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(widget.offer == null ? 'Create an offer' : 'Edit offer',
                style: text.headlineSmall?.copyWith(fontSize: 19)),
            const SizedBox(height: 16),
            Text('Discount type',
                style: text.labelMedium?.copyWith(color: SpottoColors.inkSoft)),
            const SizedBox(height: 8),
            GridView.count(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              crossAxisCount: 2,
              mainAxisSpacing: 8,
              crossAxisSpacing: 8,
              childAspectRatio: 3.4,
              children: [
                for (final t in discountTypes)
                  PressScale(
                    scale: 0.97,
                    onTap: () => setState(() => _typeId = t.id),
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10),
                      decoration: BoxDecoration(
                        color: _typeId == t.id ? SpottoColors.accent : Colors.white,
                        borderRadius: BorderRadius.circular(12),
                        border: Border.all(
                            color: _typeId == t.id
                                ? SpottoColors.accent
                                : SpottoColors.line,
                            width: 1.5),
                      ),
                      child: Row(
                        children: [
                          Icon(spottoIcon(t.icon),
                              size: 16,
                              color:
                                  _typeId == t.id ? Colors.white : SpottoColors.accent),
                          const SizedBox(width: 8),
                          Expanded(
                            child: Column(
                              mainAxisAlignment: MainAxisAlignment.center,
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(t.label,
                                    maxLines: 1,
                                    overflow: TextOverflow.ellipsis,
                                    style: TextStyle(
                                        fontSize: 12,
                                        fontWeight: FontWeight.w800,
                                        color: _typeId == t.id
                                            ? Colors.white
                                            : SpottoColors.ink)),
                                Text(t.hint,
                                    maxLines: 1,
                                    overflow: TextOverflow.ellipsis,
                                    style: TextStyle(
                                        fontSize: 10,
                                        fontWeight: FontWeight.w600,
                                        color: _typeId == t.id
                                            ? Colors.white70
                                            : SpottoColors.muted)),
                              ],
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
              ],
            ),
            const SizedBox(height: 14),
            Text('Offer title',
                style: text.labelMedium?.copyWith(color: SpottoColors.inkSoft)),
            const SizedBox(height: 6),
            TextField(
              controller: _title,
              onChanged: (_) => setState(() {}),
              decoration:
                  const InputDecoration(hintText: 'e.g. Free flat white with any day pass'),
            ),
            const SizedBox(height: 12),
            Text('Value shown to travellers',
                style: text.labelMedium?.copyWith(color: SpottoColors.inkSoft)),
            const SizedBox(height: 6),
            TextField(
              controller: _value,
              onChanged: (_) => setState(() {}),
              decoration: InputDecoration(
                  hintText: discountTypes.firstWhere((t) => t.id == _typeId).hint),
            ),
            const SizedBox(height: 14),
            Text('Duration',
                style: text.labelMedium?.copyWith(color: SpottoColors.inkSoft)),
            const SizedBox(height: 8),
            Row(
              children: [
                for (final (i, d) in _durations.indexed) ...[
                  if (i > 0) const SizedBox(width: 8),
                  Expanded(
                    child: PressScale(
                      scale: 0.97,
                      onTap: () => setState(() => _duration = d.id),
                      child: Container(
                        padding: const EdgeInsets.symmetric(vertical: 10),
                        alignment: Alignment.center,
                        decoration: BoxDecoration(
                          color:
                              _duration == d.id ? SpottoColors.accent : Colors.white,
                          borderRadius: BorderRadius.circular(12),
                          border: Border.all(
                              color: _duration == d.id
                                  ? SpottoColors.accent
                                  : SpottoColors.line,
                              width: 1.5),
                        ),
                        child: Text(d.label,
                            style: TextStyle(
                                fontSize: 12,
                                fontWeight: FontWeight.w800,
                                color: _duration == d.id
                                    ? Colors.white
                                    : SpottoColors.ink)),
                      ),
                    ),
                  ),
                ],
              ],
            ),
            const SizedBox(height: 14),
            Row(
              children: [
                Expanded(
                  child: Text('Claim limit',
                      style:
                          text.labelMedium?.copyWith(color: SpottoColors.inkSoft)),
                ),
                Text('Unlimited',
                    style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                Switch(
                  value: _unlimited,
                  activeTrackColor: SpottoColors.accent,
                  onChanged: (v) => setState(() => _unlimited = v),
                ),
              ],
            ),
            if (!_unlimited)
              Row(
                children: [
                  Expanded(
                    child: Slider(
                      value: _limit.toDouble(),
                      min: 10,
                      max: 500,
                      divisions: 49,
                      activeColor: SpottoColors.accent,
                      onChanged: (v) => setState(() => _limit = v.round()),
                    ),
                  ),
                  SizedBox(
                    width: 44,
                    child: Text('$_limit',
                        textAlign: TextAlign.end, style: text.labelLarge),
                  ),
                ],
              ),
            const SizedBox(height: 10),

            // traveller preview
            Container(
              padding: const EdgeInsets.all(14),
              decoration: BoxDecoration(
                color: const Color(0xFFFDF3D7),
                borderRadius: BorderRadius.circular(16),
                border: Border.all(color: const Color(0xFFF2DCA0), width: 1.5),
              ),
              child: Row(
                children: [
                  Container(
                    padding:
                        const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                    decoration: BoxDecoration(
                      color: SpottoColors.sun,
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Text(
                        _value.text.trim().isEmpty ? 'Deal' : _value.text.trim(),
                        style: const TextStyle(
                            fontSize: 11,
                            fontWeight: FontWeight.w800,
                            color: SpottoColors.ink)),
                  ),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Text(
                        _title.text.trim().isEmpty
                            ? 'Your offer title appears here'
                            : _title.text.trim(),
                        style: text.bodyMedium),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: () => _save('draft'),
                    child: const Text('Save draft'),
                  ),
                ),
                const SizedBox(width: 10),
                Expanded(
                  flex: 2,
                  child: FilledButton(
                    onPressed: () => _save('live'),
                    child: Text(widget.offer == null ? 'Publish offer' : 'Save changes'),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
