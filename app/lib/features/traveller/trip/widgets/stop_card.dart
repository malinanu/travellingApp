// StopCard + LegRow — one itinerary entry and the travel leg between two.

import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../../core/theme/palette.dart';
import '../../../../core/widgets/press_scale.dart';
import '../../../../core/widgets/spotto_icons.dart';
import '../../../../data/mock/trip_catalog.dart';
import '../../../../data/models/models.dart';

const tripCoral = Color(0xFFC2666F); // remove-action tone from the prototype

class StopCard extends StatelessWidget {
  final TripStop stop;
  final TripCatalogItem place;
  final bool first;
  final bool last;
  final Traveler? by;
  final ValueChanged<int> onTime;
  final VoidCallback onUp;
  final VoidCallback onDown;
  final VoidCallback onRemove;
  final VoidCallback? onOpen; // null when not a Spotto place
  final bool builderMode;

  const StopCard({
    super.key,
    required this.stop,
    required this.place,
    required this.first,
    required this.last,
    required this.by,
    required this.onTime,
    required this.onUp,
    required this.onDown,
    required this.onRemove,
    this.onOpen,
    this.builderMode = false,
  });

  Future<void> _pickTime(BuildContext context) async {
    final picked = await showTimePicker(
      context: context,
      initialTime: TimeOfDay(hour: stop.time ~/ 60, minute: stop.time % 60),
    );
    if (picked != null) onTime(picked.hour * 60 + picked.minute);
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final credit = by != null && by!.id != 'me' ? by : null;

    Widget arrow(IconData icon, bool disabled, VoidCallback onTap) => PressScale(
          scale: 0.9,
          onTap: disabled ? null : onTap,
          child: Container(
            width: 28,
            height: 28,
            decoration: BoxDecoration(
              color: SpottoColors.sand,
              borderRadius: BorderRadius.circular(8),
            ),
            child: Icon(icon,
                size: 16,
                color: disabled ? SpottoColors.ink.withValues(alpha: 0.18) : SpottoColors.inkSoft),
          ),
        );

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(24),
        boxShadow: const [BoxShadow(color: Color(0x40103D3A), blurRadius: 14, offset: Offset(0, 2), spreadRadius: -8)],
      ),
      child: Row(
        children: [
          PressScale(
            scale: 0.95,
            onTap: onOpen,
            child: Container(
              width: 48,
              height: 48,
              decoration: BoxDecoration(
                color: SpottoColors.accentTint,
                borderRadius: BorderRadius.circular(16),
              ),
              child: Icon(spottoIcon(place.icon), size: 22, color: SpottoColors.accent),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(place.name,
                    maxLines: 1, overflow: TextOverflow.ellipsis, style: text.titleSmall),
                Text('${place.type} · ${tripFmtRs(place.cost)}',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                const SizedBox(height: 6),
                Row(
                  children: [
                    GestureDetector(
                      onTap: () => _pickTime(context),
                      child: Container(
                        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                        decoration: BoxDecoration(
                          color: SpottoColors.accentTint,
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            const Icon(Icons.schedule, size: 12, color: SpottoColors.accent),
                            const SizedBox(width: 4),
                            Text(tripFmtTime(stop.time),
                                style: const TextStyle(
                                    fontSize: 12,
                                    fontWeight: FontWeight.w800,
                                    color: SpottoColors.accent)),
                          ],
                        ),
                      ),
                    ),
                    if (credit != null) ...[
                      const SizedBox(width: 8),
                      Flexible(
                        child: Container(
                          padding: const EdgeInsets.fromLTRB(2, 2, 8, 2),
                          decoration: BoxDecoration(
                            color: SpottoColors.sand,
                            borderRadius: BorderRadius.circular(999),
                          ),
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Container(
                                width: 16,
                                height: 16,
                                alignment: Alignment.center,
                                decoration: BoxDecoration(
                                    color: Color(credit.color), shape: BoxShape.circle),
                                child: Text(credit.initial,
                                    style: const TextStyle(
                                        color: Colors.white,
                                        fontSize: 9,
                                        fontWeight: FontWeight.w800)),
                              ),
                              const SizedBox(width: 4),
                              Flexible(
                                child: Text(credit.name,
                                    maxLines: 1,
                                    overflow: TextOverflow.ellipsis,
                                    style: const TextStyle(
                                        fontSize: 10.5,
                                        fontWeight: FontWeight.w700,
                                        color: SpottoColors.muted)),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ],
                  ],
                ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          if (builderMode)
            const SizedBox(
              width: 28,
              height: 48,
              child: Icon(Icons.drag_indicator, size: 22, color: SpottoColors.muted),
            )
          else
            Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                arrow(Icons.keyboard_arrow_up, first, onUp),
                const SizedBox(height: 2),
                arrow(Icons.keyboard_arrow_down, last, onDown),
              ],
            ),
          const SizedBox(width: 4),
          PressScale(
            scale: 0.9,
            onTap: onRemove,
            child: const SizedBox(
              width: 28,
              height: 28,
              child: Icon(Icons.close, size: 15, color: tripCoral),
            ),
          ),
        ],
      ),
    );
  }
}

class LegRow extends StatelessWidget {
  final Leg leg;
  const LegRow({super.key, required this.leg});

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 44,
      child: Padding(
        padding: const EdgeInsets.only(left: 26),
        child: Row(
          children: [
            // dashed connector
            SizedBox(
              width: 2,
              child: CustomPaint(painter: _DashedLinePainter(), size: const Size(2, 44)),
            ),
            const SizedBox(width: 12),
            Icon(spottoIcon(leg.icon), size: 15, color: SpottoColors.muted),
            const SizedBox(width: 8),
            Text('${leg.mins} min ${leg.mode.toLowerCase()} · ${leg.dist}',
                style: const TextStyle(
                    fontSize: 12, fontWeight: FontWeight.w700, color: SpottoColors.muted)),
            const Spacer(),
            PressScale(
              scale: 0.95,
              onTap: () =>
                  launchUrl(Uri.parse(leg.gmaps), mode: LaunchMode.externalApplication),
              child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(999),
                  border: Border.all(color: SpottoColors.line, width: 1.5),
                ),
                child: const Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(Icons.near_me, size: 12, color: SpottoColors.accent),
                    SizedBox(width: 4),
                    Text('Directions',
                        style: TextStyle(
                            fontSize: 11.5,
                            fontWeight: FontWeight.w800,
                            color: SpottoColors.accent)),
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

class _DashedLinePainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = SpottoColors.ink.withValues(alpha: 0.18)
      ..strokeWidth = 2;
    const dash = 4.0, gap = 4.0;
    var y = 0.0;
    while (y < size.height) {
      canvas.drawLine(Offset(1, y), Offset(1, (y + dash).clamp(0, size.height)), paint);
      y += dash + gap;
    }
  }

  @override
  bool shouldRepaint(_DashedLinePainter oldDelegate) => false;
}
