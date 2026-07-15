// Trip share / collaborators sheet + the shareable invitation dialog.

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/palette.dart';
import '../../../../core/widgets/press_scale.dart';
import '../../../../core/widgets/sheet_scaffold.dart';
import '../../../../data/mock/places.dart' as data;
import '../../../../data/mock/trip_catalog.dart';
import '../../../../data/repositories/repositories.dart';
import 'trip_common.dart';

/// Deterministic avatar colors for group members, who don't have one stored
/// server-side (unlike the mock traveler list, which hardcoded a color).
const _memberColors = [0xFFF0682F, 0xFF0369A1, 0xFF7C5CFF, 0xFF0B6F86, 0xFF16A34A];

Future<void> showTripShareSheet(BuildContext context) {
  return showModalBottomSheet(
    context: context,
    isScrollControlled: true,
    builder: (_) => const _TripShareSheet(),
  );
}

class _TripShareSheet extends ConsumerStatefulWidget {
  const _TripShareSheet();

  @override
  ConsumerState<_TripShareSheet> createState() => _TripShareSheetState();
}

class _TripShareSheetState extends ConsumerState<_TripShareSheet> {
  bool _copied = false;
  final Set<String> _sentIds = {};

  String get _link => 'spotto.app/t/${ref.read(tripProvider).code}';

  void _copy() {
    Clipboard.setData(ClipboardData(text: 'https://$_link'));
    setState(() => _copied = true);
    Future.delayed(const Duration(milliseconds: 1700), () {
      if (mounted) setState(() => _copied = false);
    });
  }

  Future<void> _showInvitation(TripContact? contact) async {
    final sent = await showTripInvitation(context, contact, _link);
    if (sent && contact != null) setState(() => _sentIds.add(contact.id));
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final trip = ref.watch(tripProvider);
    final collabs = trip.collaborators.map(tripWho).nonNulls.toList();
    final canAdd = data.travelers.where((t) => !trip.collaborators.contains(t.id)).toList();
    // Real group members (from GET /me/group), excluding anyone already a
    // trip collaborator — replaces the old hardcoded phone-contacts list.
    final groupContacts = [
      for (final m in ref.watch(groupProvider).members)
        if (!trip.collaborators.contains(m.id))
          TripContact(
            id: m.id,
            name: m.name,
            sub: m.email,
            initial: m.name.isNotEmpty ? m.name[0].toUpperCase() : '?',
            color: _memberColors[m.id.hashCode.abs() % _memberColors.length],
          ),
    ];

    return SheetScaffold(
      padding: const EdgeInsets.fromLTRB(20, 4, 20, 28),
      child: SizedBox(
        height: MediaQuery.of(context).size.height * 0.72,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('PLAN TOGETHER',
                style: text.labelSmall?.copyWith(color: SpottoColors.accent, letterSpacing: 1)),
            Text(trip.name,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: text.headlineSmall?.copyWith(fontSize: 21)),
            const SizedBox(height: 12),
            Expanded(
              child: ListView(
                children: [
                  // invite-link card
                  Container(
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(
                      gradient: const LinearGradient(
                        begin: Alignment.topLeft,
                        end: Alignment.bottomRight,
                        colors: [SpottoColors.accent, SpottoColors.accent2],
                      ),
                      borderRadius: BorderRadius.circular(24),
                      boxShadow: const [
                        BoxShadow(color: Color(0x660B6F86), blurRadius: 32, offset: Offset(0, 16), spreadRadius: -16)
                      ],
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Row(
                          children: [
                            Icon(Icons.link, size: 14, color: Colors.white70),
                            SizedBox(width: 6),
                            Text('Trip invite link',
                                style: TextStyle(
                                    fontSize: 12,
                                    fontWeight: FontWeight.w800,
                                    color: Colors.white70)),
                          ],
                        ),
                        const SizedBox(height: 2),
                        Text(_link,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(
                                fontSize: 15,
                                fontWeight: FontWeight.w800,
                                color: Colors.white)),
                        const SizedBox(height: 12),
                        Row(
                          children: [
                            Expanded(
                              child: PressScale(
                                scale: 0.97,
                                onTap: _copy,
                                child: Container(
                                  padding: const EdgeInsets.symmetric(vertical: 10),
                                  decoration: BoxDecoration(
                                    color: Colors.white,
                                    borderRadius: BorderRadius.circular(16),
                                  ),
                                  child: Row(
                                    mainAxisAlignment: MainAxisAlignment.center,
                                    children: [
                                      Icon(_copied ? Icons.check : Icons.copy,
                                          size: 16, color: SpottoColors.accent),
                                      const SizedBox(width: 6),
                                      Text(_copied ? 'Copied!' : 'Copy link',
                                          style: const TextStyle(
                                              fontSize: 14,
                                              fontWeight: FontWeight.w800,
                                              color: SpottoColors.accent)),
                                    ],
                                  ),
                                ),
                              ),
                            ),
                            const SizedBox(width: 8),
                            Expanded(
                              child: PressScale(
                                scale: 0.97,
                                onTap: () => _showInvitation(null),
                                child: Container(
                                  padding: const EdgeInsets.symmetric(vertical: 10),
                                  decoration: BoxDecoration(
                                    color: Colors.white.withValues(alpha: 0.18),
                                    borderRadius: BorderRadius.circular(16),
                                    border: Border.all(
                                        color: Colors.white.withValues(alpha: 0.4),
                                        width: 1.5),
                                  ),
                                  child: const Row(
                                    mainAxisAlignment: MainAxisAlignment.center,
                                    children: [
                                      Icon(Icons.share_outlined, size: 16, color: Colors.white),
                                      SizedBox(width: 6),
                                      Text('Share',
                                          style: TextStyle(
                                              fontSize: 14,
                                              fontWeight: FontWeight.w800,
                                              color: Colors.white)),
                                    ],
                                  ),
                                ),
                              ),
                            ),
                          ],
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 16),
                  _sectionLabel(text, 'Can edit this trip · ${collabs.length + 1}'),
                  _memberRow(
                    text,
                    initial: data.me.initial,
                    color: Color(data.me.color),
                    title: 'You',
                    sub: 'Trip owner · approves deletions',
                    trailing: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                      decoration: BoxDecoration(
                        color: SpottoColors.accentTint,
                        borderRadius: BorderRadius.circular(6),
                      ),
                      child: Text('OWNER',
                          style: text.labelSmall
                              ?.copyWith(fontSize: 9.5, color: SpottoColors.accent)),
                    ),
                  ),
                  for (final c in collabs)
                    _memberRow(
                      text,
                      initial: c.initial,
                      color: Color(c.color),
                      title: c.name,
                      sub: 'Editor · can add & update stops',
                      trailing: PressScale(
                        scale: 0.9,
                        onTap: () =>
                            ref.read(tripProvider.notifier).removeCollaborator(c.id),
                        child: Container(
                          width: 36,
                          height: 36,
                          decoration: BoxDecoration(
                            color: SpottoColors.sand,
                            borderRadius: BorderRadius.circular(12),
                          ),
                          child: const Icon(Icons.person_remove_outlined,
                              size: 17, color: Color(0xFFC2666F)),
                        ),
                      ),
                    ),
                  if (canAdd.isNotEmpty) ...[
                    const SizedBox(height: 8),
                    _sectionLabel(text, 'Invite travellers', icon: Icons.person_add_outlined),
                    for (final t in canAdd)
                      _memberRow(
                        text,
                        initial: t.initial,
                        color: Color(t.color),
                        title: t.name,
                        sub: t.sub,
                        faded: true,
                        trailing: _inviteButton('Invite', Icons.add, () {
                          ref.read(tripProvider.notifier).addCollaborator(t.id);
                        }),
                      ),
                  ],
                  if (groupContacts.isNotEmpty) ...[
                    const SizedBox(height: 8),
                    _sectionLabel(text, 'Your group', icon: Icons.groups_outlined),
                    Padding(
                      padding: const EdgeInsets.only(left: 4, bottom: 8),
                      child: Text('From your ${ref.watch(groupProvider).name} group.',
                          style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                    ),
                    for (final c in groupContacts)
                      _memberRow(
                        text,
                        initial: c.initial,
                        color: Color(c.color),
                        title: c.name,
                        sub: c.sub,
                        trailing: _sentIds.contains(c.id)
                            ? _inviteButton('Invited', Icons.check, null)
                            : _inviteButton('Invite', Icons.send, () => _showInvitation(c)),
                      ),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _sectionLabel(TextTheme text, String label, {IconData? icon}) => Padding(
        padding: const EdgeInsets.only(top: 8, bottom: 8, left: 4),
        child: Row(
          children: [
            if (icon != null) ...[
              Icon(icon, size: 13, color: SpottoColors.accent),
              const SizedBox(width: 5),
            ],
            Text(label.toUpperCase(),
                style: text.labelSmall?.copyWith(color: SpottoColors.muted, letterSpacing: 0.6)),
          ],
        ),
      );

  Widget _memberRow(
    TextTheme text, {
    required String initial,
    required Color color,
    required String title,
    required String sub,
    required Widget trailing,
    bool faded = false,
  }) =>
      Padding(
        padding: const EdgeInsets.only(bottom: 8),
        child: Container(
          padding: const EdgeInsets.all(10),
          decoration: BoxDecoration(
            color: faded ? Colors.white.withValues(alpha: 0.6) : Colors.white,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: SpottoColors.line, width: 1.5),
          ),
          child: Row(
            children: [
              AvatarBubble(initial: initial, color: color, size: 40, borderWidth: 0),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(title,
                        maxLines: 1, overflow: TextOverflow.ellipsis, style: text.titleSmall),
                    Text(sub,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                  ],
                ),
              ),
              trailing,
            ],
          ),
        ),
      );

  Widget _inviteButton(String label, IconData icon, VoidCallback? onTap) => PressScale(
        scale: 0.95,
        onTap: onTap,
        child: Container(
          height: 36,
          padding: const EdgeInsets.symmetric(horizontal: 12),
          decoration: BoxDecoration(
            color: onTap == null ? SpottoColors.sand : SpottoColors.accent,
            borderRadius: BorderRadius.circular(12),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon, size: 15, color: onTap == null ? SpottoColors.muted : Colors.white),
              const SizedBox(width: 5),
              Text(label,
                  style: TextStyle(
                      fontSize: 12.5,
                      fontWeight: FontWeight.w800,
                      color: onTap == null ? SpottoColors.muted : Colors.white)),
            ],
          ),
        ),
      );
}

/// Centered invitation card. Returns true when "sent".
Future<bool> showTripInvitation(BuildContext context, TripContact? to, String link) async {
  final sent = await showDialog<bool>(
    context: context,
    builder: (_) => _TripInvitationDialog(to: to, link: link),
  );
  return sent ?? false;
}

class _TripInvitationDialog extends ConsumerWidget {
  final TripContact? to;
  final String link;
  const _TripInvitationDialog({required this.to, required this.link});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final text = Theme.of(context).textTheme;
    final trip = ref.watch(tripProvider);
    final first = tripDayDate(trip.startDate, 0);
    final last = tripDayDate(trip.startDate, trip.days.length - 1);
    final range = trip.days.length == 1
        ? DateFormat('EEEE, MMMM d, yyyy').format(first)
        : '${DateFormat('MMM d').format(first)} – ${DateFormat('MMM d, yyyy').format(last)}';

    return Dialog(
      insetPadding: const EdgeInsets.symmetric(horizontal: 24),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(28)),
      clipBehavior: Clip.antiAlias,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          // header
          Container(
            width: double.infinity,
            padding: const EdgeInsets.fromLTRB(24, 28, 24, 24),
            decoration: const BoxDecoration(
              gradient: LinearGradient(
                begin: Alignment.topLeft,
                end: Alignment.bottomRight,
                colors: [SpottoColors.accent, SpottoColors.accent2],
              ),
            ),
            child: Column(
              children: [
                Container(
                  width: 48,
                  height: 48,
                  decoration: BoxDecoration(
                    color: Colors.white.withValues(alpha: 0.22),
                    borderRadius: BorderRadius.circular(16),
                  ),
                  child: const Icon(Icons.flight_takeoff, size: 24, color: Colors.white),
                ),
                const SizedBox(height: 12),
                const Text("YOU'RE INVITED",
                    style: TextStyle(
                        fontSize: 11,
                        fontWeight: FontWeight.w800,
                        letterSpacing: 2,
                        color: Colors.white70)),
                const SizedBox(height: 4),
                Text(trip.name,
                    textAlign: TextAlign.center,
                    style: text.headlineSmall?.copyWith(color: Colors.white, fontSize: 24)),
                const SizedBox(height: 10),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                  decoration: BoxDecoration(
                    color: Colors.white.withValues(alpha: 0.2),
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const Icon(Icons.calendar_month, size: 13, color: Colors.white),
                      const SizedBox(width: 5),
                      Text(
                          '$range · ${trip.days.length} day${trip.days.length > 1 ? 's' : ''}',
                          style: const TextStyle(
                              fontSize: 12.5,
                              fontWeight: FontWeight.w800,
                              color: Colors.white)),
                    ],
                  ),
                ),
                const SizedBox(height: 12),
                const Text('Hosted by you · Weligama, Sri Lanka',
                    style: TextStyle(
                        fontSize: 12.5, fontWeight: FontWeight.w700, color: Colors.white)),
              ],
            ),
          ),
          // body
          Padding(
            padding: const EdgeInsets.all(20),
            child: Column(
              children: [
                if (to != null)
                  Container(
                    margin: const EdgeInsets.only(bottom: 12),
                    padding: const EdgeInsets.all(10),
                    decoration: BoxDecoration(
                      color: SpottoColors.sand,
                      borderRadius: BorderRadius.circular(16),
                    ),
                    child: Row(
                      children: [
                        AvatarBubble(
                            initial: to!.initial,
                            color: Color(to!.color),
                            size: 36,
                            borderWidth: 0),
                        const SizedBox(width: 10),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text('SENDING TO',
                                  style: text.labelSmall?.copyWith(
                                      fontSize: 10, color: SpottoColors.muted)),
                              Text('${to!.name} · ${to!.sub}',
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                  style: text.titleSmall),
                            ],
                          ),
                        ),
                      ],
                    ),
                  ),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                  decoration: BoxDecoration(
                    borderRadius: BorderRadius.circular(16),
                    border: Border.all(color: SpottoColors.line, width: 1.5),
                  ),
                  child: Row(
                    children: [
                      const Icon(Icons.link, size: 15, color: SpottoColors.accent),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text(link,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: text.bodySmall
                                ?.copyWith(fontWeight: FontWeight.w700)),
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 12),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton.icon(
                    onPressed: () => Navigator.of(context).pop(true),
                    icon: Icon(to != null ? Icons.chat_bubble_outline : Icons.send, size: 17),
                    label: Text(
                        to != null ? 'Send to ${to!.name.split(' ').first}' : 'Send invitation'),
                  ),
                ),
                const SizedBox(height: 8),
                TextButton(
                  onPressed: () {
                    Clipboard.setData(ClipboardData(
                        text:
                            'You\'re invited to "${trip.name}" on Spotto — $range. Join: https://$link'));
                    ScaffoldMessenger.of(context)
                        .showSnackBar(const SnackBar(content: Text('Invitation copied')));
                  },
                  child: const Text('Copy invitation text'),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
