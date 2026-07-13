// Travel-group sheet — port of GroupSheet.jsx: members, invite link + faux
// QR, add nearby travellers, remove, leave.

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/sheet_scaffold.dart';
import '../../../data/mock/places.dart' as data;
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';
import '../trip/widgets/trip_common.dart';

const _green = Color(0xFF3EC97A);
const _coral = Color(0xFFC2666F);

Future<void> showGroupSheet(BuildContext context) {
  return showModalBottomSheet(
    context: context,
    isScrollControlled: true,
    builder: (_) => const _GroupSheet(),
  );
}

class _GroupSheet extends ConsumerStatefulWidget {
  const _GroupSheet();

  @override
  ConsumerState<_GroupSheet> createState() => _GroupSheetState();
}

class _GroupSheetState extends ConsumerState<_GroupSheet> {
  bool _copied = false;
  bool _showQR = false;
  String? _justAdded;

  void _copy(String link) {
    Clipboard.setData(ClipboardData(text: 'https://$link'));
    setState(() => _copied = true);
    Future.delayed(const Duration(milliseconds: 1800), () {
      if (mounted) setState(() => _copied = false);
    });
  }

  void _add(String id) {
    ref.read(groupProvider.notifier).addMember(id);
    setState(() => _justAdded = id);
    Future.delayed(const Duration(milliseconds: 1200), () {
      if (mounted && _justAdded == id) setState(() => _justAdded = null);
    });
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final group = ref.watch(groupProvider);
    final members = group.memberIds
        .map((id) => data.travelers.where((t) => t.id == id).firstOrNull)
        .nonNulls
        .toList();
    final suggestions =
        data.travelers.where((t) => !group.memberIds.contains(t.id)).toList();
    final link = 'spotto.app/g/${group.code}';

    return SheetScaffold(
      padding: const EdgeInsets.fromLTRB(20, 4, 20, 28),
      child: SizedBox(
        height: MediaQuery.of(context).size.height * 0.72,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('TRAVEL GROUP',
                          style: text.labelSmall
                              ?.copyWith(color: SpottoColors.accent, letterSpacing: 1)),
                      Text(group.name,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: text.headlineSmall?.copyWith(fontSize: 21)),
                    ],
                  ),
                ),
                SizedBox(
                  height: 30,
                  width: 30.0 + (members.take(4).length + 1 - 1) * 19.0,
                  child: Stack(
                    children: [
                      for (final (i, m) in members.take(4).indexed)
                        Positioned(
                          left: i * 19.0,
                          child: AvatarBubble(
                              initial: m.initial, color: Color(m.color), size: 30),
                        ),
                      Positioned(
                        left: members.take(4).length * 19.0,
                        child: Container(
                          width: 30,
                          height: 30,
                          alignment: Alignment.center,
                          decoration: BoxDecoration(
                            color: Colors.white,
                            shape: BoxShape.circle,
                            border: Border.all(color: SpottoColors.sand, width: 2.5),
                          ),
                          child: Text('${members.length + 1}',
                              style: const TextStyle(
                                  fontSize: 11.5,
                                  fontWeight: FontWeight.w800,
                                  color: SpottoColors.ink)),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            Expanded(
              child: ListView(
                children: [
                  _inviteCard(text, link),
                  const SizedBox(height: 16),
                  Padding(
                    padding: const EdgeInsets.only(left: 4, bottom: 8),
                    child: Row(
                      children: [
                        Text('IN YOUR GROUP · ${members.length + 1}',
                            style: text.labelSmall
                                ?.copyWith(color: SpottoColors.muted, letterSpacing: 0.6)),
                        const Spacer(),
                        Container(
                            width: 7,
                            height: 7,
                            decoration:
                                const BoxDecoration(color: _green, shape: BoxShape.circle)),
                        const SizedBox(width: 4),
                        Text('Live', style: text.labelSmall?.copyWith(color: _green)),
                      ],
                    ),
                  ),
                  _memberRow(text, data.me, host: true),
                  for (final m in members) _memberRow(text, m),
                  if (suggestions.isNotEmpty) ...[
                    Padding(
                      padding: const EdgeInsets.only(top: 12, bottom: 8, left: 4),
                      child: Row(
                        children: [
                          const Icon(Icons.person_add_outlined,
                              size: 13, color: SpottoColors.accent),
                          const SizedBox(width: 5),
                          Text('ADD TRAVELLERS NEARBY',
                              style: text.labelSmall?.copyWith(
                                  color: SpottoColors.muted, letterSpacing: 0.6)),
                        ],
                      ),
                    ),
                    for (final s in suggestions) _suggestRow(text, s),
                  ],
                  const SizedBox(height: 12),
                  PressScale(
                    scale: 0.98,
                    onTap: () {
                      ref.read(groupProvider.notifier).leave();
                      Navigator.of(context).pop();
                    },
                    child: Container(
                      padding: const EdgeInsets.symmetric(vertical: 12),
                      decoration: BoxDecoration(
                        borderRadius: BorderRadius.circular(16),
                        border: Border.all(
                            color: _coral.withValues(alpha: 0.35), width: 1.5),
                      ),
                      child: const Row(
                        mainAxisAlignment: MainAxisAlignment.center,
                        children: [
                          Icon(Icons.logout, size: 15, color: _coral),
                          SizedBox(width: 6),
                          Text('Leave group',
                              style: TextStyle(
                                  fontSize: 13.5,
                                  fontWeight: FontWeight.w800,
                                  color: _coral)),
                        ],
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _inviteCard(TextTheme text, String link) => Container(
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
            Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      const Row(
                        children: [
                          Icon(Icons.link, size: 14, color: Colors.white70),
                          SizedBox(width: 6),
                          Text('Invite link',
                              style: TextStyle(
                                  fontSize: 12,
                                  fontWeight: FontWeight.w800,
                                  color: Colors.white70)),
                        ],
                      ),
                      Text(link,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                              fontSize: 15,
                              fontWeight: FontWeight.w800,
                              color: Colors.white)),
                    ],
                  ),
                ),
                PressScale(
                  scale: 0.95,
                  onTap: () => setState(() => _showQR = !_showQR),
                  child: Container(
                    width: 44,
                    height: 44,
                    decoration: BoxDecoration(
                      color: Colors.white.withValues(alpha: 0.2),
                      borderRadius: BorderRadius.circular(16),
                    ),
                    child: const Icon(Icons.qr_code_2, size: 20, color: Colors.white),
                  ),
                ),
              ],
            ),
            if (_showQR) ...[
              const SizedBox(height: 12),
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(16),
                ),
                child: Row(
                  children: [
                    Container(
                      padding: const EdgeInsets.all(6),
                      decoration: BoxDecoration(
                        borderRadius: BorderRadius.circular(12),
                        border: Border.all(color: SpottoColors.line, width: 1.5),
                      ),
                      child: const CustomPaint(
                          size: Size(66, 66), painter: MiniQrPainter()),
                    ),
                    const SizedBox(width: 14),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text('Scan to join', style: text.titleSmall),
                          Text(
                              "Point a camera here — they'll drop straight into ${ref.read(groupProvider).name}.",
                              style:
                                  text.bodySmall?.copyWith(color: SpottoColors.muted)),
                        ],
                      ),
                    ),
                  ],
                ),
              ),
            ],
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: PressScale(
                    scale: 0.97,
                    onTap: () => _copy(link),
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
                  child: Container(
                    padding: const EdgeInsets.symmetric(vertical: 10),
                    decoration: BoxDecoration(
                      color: Colors.white.withValues(alpha: 0.18),
                      borderRadius: BorderRadius.circular(16),
                      border:
                          Border.all(color: Colors.white.withValues(alpha: 0.4), width: 1.5),
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
              ],
            ),
          ],
        ),
      );

  Widget _memberRow(TextTheme text, Traveler t, {bool host = false}) => Padding(
        padding: const EdgeInsets.only(bottom: 8),
        child: Container(
          padding: const EdgeInsets.all(10),
          decoration: BoxDecoration(
            color: Colors.white,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: SpottoColors.line, width: 1.5),
          ),
          child: Row(
            children: [
              _liveAvatar(t),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Flexible(
                          child: Text(t.name,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: text.titleSmall),
                        ),
                        if (host) ...[
                          const SizedBox(width: 6),
                          Container(
                            padding:
                                const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                            decoration: BoxDecoration(
                              color: SpottoColors.accentTint,
                              borderRadius: BorderRadius.circular(6),
                            ),
                            child: Text('HOST',
                                style: text.labelSmall?.copyWith(
                                    fontSize: 9.5, color: SpottoColors.accent)),
                          ),
                        ],
                      ],
                    ),
                    Row(
                      children: [
                        const Icon(Icons.place, size: 12, color: SpottoColors.accent),
                        const SizedBox(width: 3),
                        Flexible(
                          child: Text('near ${t.at}',
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style:
                                  text.bodySmall?.copyWith(color: SpottoColors.muted)),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
              if (host)
                Text('You', style: text.labelMedium?.copyWith(color: SpottoColors.muted))
              else
                PressScale(
                  scale: 0.9,
                  onTap: () => ref.read(groupProvider.notifier).removeMember(t.id),
                  child: Container(
                    width: 36,
                    height: 36,
                    decoration: BoxDecoration(
                      color: SpottoColors.sand,
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: const Icon(Icons.person_remove_outlined,
                        size: 17, color: _coral),
                  ),
                ),
            ],
          ),
        ),
      );

  Widget _suggestRow(TextTheme text, Traveler t) {
    final justAdded = _justAdded == t.id;
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Container(
        padding: const EdgeInsets.all(10),
        decoration: BoxDecoration(
          color: Colors.white.withValues(alpha: 0.6),
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: SpottoColors.line, width: 1.5),
        ),
        child: Row(
          children: [
            AvatarBubble(initial: t.initial, color: Color(t.color), size: 44),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(t.name,
                      maxLines: 1, overflow: TextOverflow.ellipsis, style: text.titleSmall),
                  Text(t.sub,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                ],
              ),
            ),
            PressScale(
              scale: 0.95,
              onTap: () => _add(t.id),
              child: Container(
                height: 36,
                padding: const EdgeInsets.symmetric(horizontal: 12),
                decoration: BoxDecoration(
                  color: justAdded ? _green : SpottoColors.accent,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(justAdded ? Icons.check : Icons.add, size: 16, color: Colors.white),
                    const SizedBox(width: 5),
                    Text(justAdded ? 'Added' : 'Add',
                        style: const TextStyle(
                            fontSize: 12.5,
                            fontWeight: FontWeight.w800,
                            color: Colors.white)),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _liveAvatar(Traveler t) => SizedBox(
        width: 46,
        height: 46,
        child: Stack(
          children: [
            AvatarBubble(initial: t.initial, color: Color(t.color), size: 44),
            Positioned(
              right: 0,
              bottom: 0,
              child: Container(
                width: 13,
                height: 13,
                decoration: BoxDecoration(
                  color: _green,
                  shape: BoxShape.circle,
                  border: Border.all(color: Colors.white, width: 2.5),
                ),
              ),
            ),
          ],
        ),
      );
}

/// Deterministic faux-QR (decorative only) — port of MiniQR.
class MiniQrPainter extends CustomPainter {
  const MiniQrPainter();

  static const _n = 9;
  static const _seed = [0x1f4, 0x2a7, 0x35b, 0x1c9, 0x2e6, 0x3b1, 0x14d, 0x2f2, 0x39a];

  bool _inFinder(int r, int c) =>
      (r < 3 && c < 3) || (r < 3 && c > _n - 4) || (r > _n - 4 && c < 3);

  @override
  void paint(Canvas canvas, Size size) {
    final cell = size.width / _n;
    final ink = Paint()..color = SpottoColors.ink;
    final white = Paint()..color = Colors.white;

    for (var r = 0; r < _n; r++) {
      for (var c = 0; c < _n; c++) {
        if (!_inFinder(r, c) && ((_seed[r] >> c) & 1) == 1) {
          canvas.drawRRect(
            RRect.fromRectAndRadius(
              Rect.fromLTWH(c * cell + cell * 0.1, r * cell + cell * 0.1, cell * 0.8, cell * 0.8),
              Radius.circular(cell * 0.2),
            ),
            ink,
          );
        }
      }
    }

    void finder(int fr, int fc) {
      canvas.drawRRect(
        RRect.fromRectAndRadius(
          Rect.fromLTWH(fc * cell, fr * cell, cell * 3, cell * 3),
          Radius.circular(cell * 0.5),
        ),
        ink,
      );
      canvas.drawRRect(
        RRect.fromRectAndRadius(
          Rect.fromLTWH((fc + 0.6) * cell, (fr + 0.6) * cell, cell * 1.8, cell * 1.8),
          Radius.circular(cell * 0.35),
        ),
        white,
      );
      canvas.drawRRect(
        RRect.fromRectAndRadius(
          Rect.fromLTWH((fc + 1) * cell, (fr + 1) * cell, cell, cell),
          Radius.circular(cell * 0.25),
        ),
        ink,
      );
    }

    finder(0, 0);
    finder(0, _n - 3);
    finder(_n - 3, 0);
  }

  @override
  bool shouldRepaint(MiniQrPainter oldDelegate) => false;
}
