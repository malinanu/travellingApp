// Growth tools — port of GrowthTools.jsx: flash sale push, stamp card,
// behind-the-scenes clips, sponsored boost. All session-only.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/mock/vendor_seed.dart';
import '../widgets/vendor_common.dart';

class VendorGrowthScreen extends ConsumerStatefulWidget {
  const VendorGrowthScreen({super.key});

  @override
  ConsumerState<VendorGrowthScreen> createState() => _VendorGrowthScreenState();
}

class _VendorGrowthScreenState extends ConsumerState<VendorGrowthScreen> {
  int _flashPct = 20;
  String _flashState = 'idle'; // idle | sending | sent
  bool _uploading = false;
  double _uploadProgress = 0;
  String _boostTier = 'category';
  bool _boostOn = false;

  Future<void> _sendFlash() async {
    setState(() => _flashState = 'sending');
    await Future.delayed(const Duration(milliseconds: 1400));
    if (!mounted) return;
    setState(() => _flashState = 'sent');
  }

  Future<void> _upload() async {
    setState(() {
      _uploading = true;
      _uploadProgress = 0;
    });
    for (var i = 1; i <= 20; i++) {
      await Future.delayed(const Duration(milliseconds: 90));
      if (!mounted) return;
      setState(() => _uploadProgress = i / 20);
    }
    if (!mounted) return;
    setState(() => _uploading = false);
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Clip uploaded — travellers will see it soon')));
    }
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;

    return ListView(
      padding: const EdgeInsets.fromLTRB(20, 8, 20, 24),
      children: [
        _flashSaleCard(text),
        const SizedBox(height: 16),
        _stampCard(text),
        const SizedBox(height: 16),
        _btsCard(text),
        const SizedBox(height: 16),
        _boostCard(text),
      ],
    );
  }

  Widget _sectionHeader(TextTheme text, IconData icon, String title, String sub) => Row(
        children: [
          Container(
            width: 40,
            height: 40,
            decoration: BoxDecoration(
              color: SpottoColors.accentTint,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Icon(icon, size: 20, color: SpottoColors.accent),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: text.titleMedium?.copyWith(fontSize: 16)),
                Text(sub, style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
              ],
            ),
          ),
        ],
      );

  Widget _flashSaleCard(TextTheme text) {
    final sent = _flashState == 'sent';
    return VendorCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _sectionHeader(text, Icons.bolt, 'Flash sale push',
              '${flash.inRange} travellers within ${flash.radiusKm} km right now'),
          const SizedBox(height: 12),
          Row(
            children: [
              _audiencePill('${flash.nomads} nomads'),
              const SizedBox(width: 8),
              _audiencePill('${flash.surfers} surfers'),
              const Spacer(),
              Text('last push ${flash.lastSentMins ~/ 60} hr ago',
                  style: text.labelSmall?.copyWith(color: SpottoColors.muted)),
            ],
          ),
          const SizedBox(height: 14),
          Row(
            children: [
              for (final pct in const [10, 20, 30]) ...[
                Expanded(
                  child: PressScale(
                    scale: 0.97,
                    onTap: () => setState(() => _flashPct = pct),
                    child: Container(
                      padding: const EdgeInsets.symmetric(vertical: 10),
                      alignment: Alignment.center,
                      decoration: BoxDecoration(
                        color: _flashPct == pct ? SpottoColors.accent : Colors.white,
                        borderRadius: BorderRadius.circular(12),
                        border: Border.all(
                            color: _flashPct == pct
                                ? SpottoColors.accent
                                : SpottoColors.line,
                            width: 1.5),
                      ),
                      child: Text('$pct% off',
                          style: TextStyle(
                              fontSize: 13,
                              fontWeight: FontWeight.w800,
                              color: _flashPct == pct
                                  ? Colors.white
                                  : SpottoColors.ink)),
                    ),
                  ),
                ),
                if (pct != 30) const SizedBox(width: 8),
              ],
            ],
          ),
          const SizedBox(height: 12),
          SizedBox(
            width: double.infinity,
            child: FilledButton.icon(
              style: sent
                  ? FilledButton.styleFrom(backgroundColor: vendorGreen)
                  : null,
              onPressed: _flashState == 'idle' ? _sendFlash : null,
              icon: _flashState == 'sending'
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(
                          strokeWidth: 2.5, color: Colors.white))
                  : Icon(sent ? Icons.check : Icons.send, size: 17),
              label: Text(sent
                  ? 'Sent to ${flash.inRange} travellers'
                  : _flashState == 'sending'
                      ? 'Sending…'
                      : 'Send $_flashPct% flash sale'),
            ),
          ),
        ],
      ),
    );
  }

  Widget _audiencePill(String label) => Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
        decoration: BoxDecoration(
          color: SpottoColors.sand,
          borderRadius: BorderRadius.circular(999),
        ),
        child: Text(label,
            style: const TextStyle(
                fontSize: 11.5, fontWeight: FontWeight.w800, color: SpottoColors.ink)),
      );

  Widget _stampCard(TextTheme text) {
    return VendorCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _sectionHeader(text, Icons.loyalty_outlined, 'Stamp card',
              '${stampCard.totalHolders} holders · ${stampCard.redeemedThisMonth} rewards this month'),
          const SizedBox(height: 12),
          Container(
            padding: const EdgeInsets.all(14),
            decoration: BoxDecoration(
              color: SpottoColors.sand,
              borderRadius: BorderRadius.circular(16),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Buy ${stampCard.stampsNeeded}, get "${stampCard.reward}"',
                    style: text.titleSmall),
                const SizedBox(height: 10),
                Row(
                  children: [
                    for (var i = 0; i < stampCard.stampsNeeded; i++) ...[
                      if (i > 0) const SizedBox(width: 6),
                      Container(
                        width: 28,
                        height: 28,
                        decoration: BoxDecoration(
                          color: i < 4 ? SpottoColors.accent : Colors.white,
                          shape: BoxShape.circle,
                          border: Border.all(
                              color: i < 4 ? SpottoColors.accent : SpottoColors.line,
                              width: 1.5),
                        ),
                        child: i < 4
                            ? const Icon(Icons.check, size: 15, color: Colors.white)
                            : null,
                      ),
                    ],
                  ],
                ),
              ],
            ),
          ),
          const SizedBox(height: 12),
          for (final b in stampCard.buckets)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Row(
                children: [
                  SizedBox(
                    width: 110,
                    child: Text(b.label,
                        style:
                            text.bodySmall?.copyWith(color: SpottoColors.inkSoft)),
                  ),
                  Expanded(
                    child: ClipRRect(
                      borderRadius: BorderRadius.circular(999),
                      child: Container(
                        height: 8,
                        color: SpottoColors.ink.withValues(alpha: 0.08),
                        child: FractionallySizedBox(
                          alignment: Alignment.centerLeft,
                          widthFactor: b.count / stampCard.totalHolders,
                          child: Container(color: SpottoColors.accent),
                        ),
                      ),
                    ),
                  ),
                  const SizedBox(width: 10),
                  SizedBox(
                    width: 28,
                    child: Text('${b.count}',
                        textAlign: TextAlign.end, style: text.labelMedium),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }

  Widget _btsCard(TextTheme text) {
    return VendorCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _sectionHeader(text, Icons.videocam_outlined, 'Behind the scenes',
              'Short clips shown on your traveller profile'),
          const SizedBox(height: 12),
          SizedBox(
            height: 110,
            child: ListView(
              scrollDirection: Axis.horizontal,
              children: [
                for (final c in btsClips)
                  Container(
                    width: 150,
                    margin: const EdgeInsets.only(right: 10),
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(
                      gradient: LinearGradient(
                        begin: Alignment.topLeft,
                        end: Alignment.bottomRight,
                        colors: [Color(c.gradStart), Color(c.gradEnd)],
                      ),
                      borderRadius: BorderRadius.circular(16),
                    ),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Icon(spottoIcon(c.icon), size: 18, color: Colors.white),
                        const Spacer(),
                        Text(c.title,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(
                                fontSize: 12,
                                fontWeight: FontWeight.w800,
                                color: Colors.white)),
                        const SizedBox(height: 3),
                        Text('${c.secs}s · ${c.views} views',
                            style: const TextStyle(
                                fontSize: 10.5,
                                fontWeight: FontWeight.w600,
                                color: Colors.white70)),
                      ],
                    ),
                  ),
              ],
            ),
          ),
          const SizedBox(height: 12),
          if (_uploading) ...[
            ClipRRect(
              borderRadius: BorderRadius.circular(999),
              child: LinearProgressIndicator(
                value: _uploadProgress,
                minHeight: 8,
                backgroundColor: SpottoColors.ink.withValues(alpha: 0.08),
                color: SpottoColors.accent,
              ),
            ),
            const SizedBox(height: 8),
            Text('Uploading… ${(_uploadProgress * 100).round()}%',
                style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
          ] else
            SizedBox(
              width: double.infinity,
              child: OutlinedButton.icon(
                onPressed: _upload,
                icon: const Icon(Icons.upload_outlined, size: 17),
                label: const Text('Upload a new clip'),
              ),
            ),
        ],
      ),
    );
  }

  Widget _boostCard(TextTheme text) {
    return VendorCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _sectionHeader(text, Icons.rocket_launch_outlined, 'Sponsored boost',
              'Promote "${boostEvent.name}" · ${boostEvent.when}'),
          const SizedBox(height: 12),
          for (final t in boostTiers)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: PressScale(
                scale: 0.98,
                onTap: () => setState(() => _boostTier = t.id),
                child: Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: _boostTier == t.id
                        ? SpottoColors.accentTint
                        : Colors.white,
                    borderRadius: BorderRadius.circular(14),
                    border: Border.all(
                        color: _boostTier == t.id
                            ? SpottoColors.accent
                            : SpottoColors.line,
                        width: 1.5),
                  ),
                  child: Row(
                    children: [
                      Icon(spottoIcon(t.icon),
                          size: 18,
                          color: _boostTier == t.id
                              ? SpottoColors.accent
                              : SpottoColors.muted),
                      const SizedBox(width: 10),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Row(
                              children: [
                                Text(t.label, style: text.titleSmall),
                                if (t.best) ...[
                                  const SizedBox(width: 6),
                                  Container(
                                    padding: const EdgeInsets.symmetric(
                                        horizontal: 6, vertical: 2),
                                    decoration: BoxDecoration(
                                      color: SpottoColors.sun,
                                      borderRadius: BorderRadius.circular(6),
                                    ),
                                    child: const Text('BEST VALUE',
                                        style: TextStyle(
                                            fontSize: 8.5,
                                            fontWeight: FontWeight.w800,
                                            color: SpottoColors.ink)),
                                  ),
                                ],
                              ],
                            ),
                            Text('${t.reach} · ${t.days} days',
                                style: text.bodySmall
                                    ?.copyWith(color: SpottoColors.muted)),
                          ],
                        ),
                      ),
                      Text('\$${t.price}', style: text.titleMedium),
                    ],
                  ),
                ),
              ),
            ),
          const SizedBox(height: 4),
          SizedBox(
            width: double.infinity,
            child: FilledButton.icon(
              style: _boostOn
                  ? FilledButton.styleFrom(backgroundColor: vendorGreen)
                  : null,
              onPressed: () => setState(() => _boostOn = !_boostOn),
              icon: Icon(_boostOn ? Icons.check : Icons.rocket_launch, size: 17),
              label: Text(_boostOn ? 'Boost active — tap to stop' : 'Start boost'),
            ),
          ),
        ],
      ),
    );
  }
}
