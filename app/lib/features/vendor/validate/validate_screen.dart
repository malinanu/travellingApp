// Validate — port of QRValidator.jsx: mock scanner with animated reticle,
// simulate-scan, manual code entry, result card. Validates seeded codes AND
// codes the traveller claimed on this device.

import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../data/mock/vendor_seed.dart';
import '../../../data/repositories/repositories.dart';
import '../widgets/vendor_common.dart';

class VendorValidateScreen extends ConsumerStatefulWidget {
  const VendorValidateScreen({super.key});

  @override
  ConsumerState<VendorValidateScreen> createState() => _VendorValidateScreenState();
}

class _VendorValidateScreenState extends ConsumerState<VendorValidateScreen>
    with SingleTickerProviderStateMixin {
  final _code = TextEditingController();
  ValidationResult? _result;
  bool _scanning = false;
  late final AnimationController _scanAnim =
      AnimationController(vsync: this, duration: const Duration(milliseconds: 1600))
        ..repeat(reverse: true);

  @override
  void dispose() {
    _code.dispose();
    _scanAnim.dispose();
    super.dispose();
  }

  void _validate(String raw) {
    if (raw.trim().isEmpty) return;
    setState(() => _result = ref.read(codeValidatorProvider)(raw));
  }

  Future<void> _simulateScan() async {
    setState(() {
      _scanning = true;
      _result = null;
    });
    await Future.delayed(const Duration(milliseconds: 1200));
    if (!mounted) return;
    final r = math.Random();
    final pool = [...validCodes.keys, ...validCodesLegacy.keys];
    final code =
        r.nextDouble() < 0.8 ? pool[r.nextInt(pool.length)] : 'SPOTTO-BAD${r.nextInt(90) + 10}';
    setState(() {
      _scanning = false;
      _result = ref.read(codeValidatorProvider)(code);
    });
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;

    return ListView(
      padding: const EdgeInsets.fromLTRB(20, 8, 20, 24),
      children: [
        // scanner panel (diegetic dark viewfinder — stays dark by design)
        ClipRRect(
          borderRadius: BorderRadius.circular(24),
          child: Container(
            height: 300,
            decoration: const BoxDecoration(
              gradient: RadialGradient(
                center: Alignment(0, -0.2),
                radius: 1.2,
                colors: [Color(0xFF16453F), Color(0xFF0C2825)],
              ),
            ),
            child: Stack(
              alignment: Alignment.center,
              children: [
                // faux QR while "scanning"
                if (_scanning)
                  const Opacity(
                      opacity: 0.35,
                      child: Icon(Icons.qr_code_2, size: 130, color: Colors.white)),
                if (!_scanning)
                  Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const Icon(Icons.qr_code_scanner,
                          size: 42, color: Colors.white54),
                      const SizedBox(height: 10),
                      Text('Point at a traveller\'s Spotto code',
                          style: text.bodySmall?.copyWith(color: Colors.white70)),
                    ],
                  ),
                // reticle corners
                SizedBox(
                  width: 190,
                  height: 190,
                  child: CustomPaint(painter: _ReticlePainter()),
                ),
                // scan line
                AnimatedBuilder(
                  animation: _scanAnim,
                  builder: (context, _) => Positioned(
                    top: 70 + _scanAnim.value * 160,
                    left: 60,
                    right: 60,
                    child: Container(
                      height: 2.5,
                      decoration: BoxDecoration(
                        color: SpottoColors.sun,
                        borderRadius: BorderRadius.circular(999),
                        boxShadow: const [
                          BoxShadow(color: Color(0x99F6B81E), blurRadius: 12)
                        ],
                      ),
                    ),
                  ),
                ),
                Positioned(
                  bottom: 16,
                  left: 20,
                  right: 20,
                  child: FilledButton.icon(
                    style: FilledButton.styleFrom(
                      backgroundColor: SpottoColors.sun,
                      foregroundColor: SpottoColors.ink,
                    ),
                    onPressed: _scanning ? null : _simulateScan,
                    icon: _scanning
                        ? const SizedBox(
                            width: 16,
                            height: 16,
                            child: CircularProgressIndicator(
                                strokeWidth: 2.5, color: SpottoColors.ink))
                        : const Icon(Icons.center_focus_strong, size: 18),
                    label: Text(_scanning ? 'Scanning…' : 'Simulate scan'),
                  ),
                ),
              ],
            ),
          ),
        ),
        const SizedBox(height: 16),

        // manual entry
        VendorCard(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('Enter code manually',
                  style: text.titleMedium?.copyWith(fontSize: 16)),
              const SizedBox(height: 10),
              Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: _code,
                      textCapitalization: TextCapitalization.characters,
                      onSubmitted: _validate,
                      decoration: const InputDecoration(hintText: 'SPOTTO-XXXX'),
                    ),
                  ),
                  const SizedBox(width: 10),
                  FilledButton(
                    onPressed: () => _validate(_code.text),
                    child: const Text('Check'),
                  ),
                ],
              ),
              const SizedBox(height: 10),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  for (final demo in ['SPOTTO-X4K9', 'SPOTTO-88Q2', 'SPOTTO-DRIF'])
                    PressScale(
                      scale: 0.95,
                      onTap: () {
                        _code.text = demo;
                        _validate(demo);
                      },
                      child: Container(
                        padding: const EdgeInsets.symmetric(
                            horizontal: 10, vertical: 6),
                        decoration: BoxDecoration(
                          color: SpottoColors.sand,
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: Text(demo,
                            style: const TextStyle(
                                fontSize: 12,
                                fontWeight: FontWeight.w800,
                                color: SpottoColors.ink)),
                      ),
                    ),
                ],
              ),
              const SizedBox(height: 6),
              Text('Tip: codes claimed by a traveller on this phone validate here too.',
                  style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
            ],
          ),
        ),
        if (_result != null) ...[
          const SizedBox(height: 16),
          _ResultCard(result: _result!),
        ],
      ],
    );
  }
}

class _ResultCard extends StatelessWidget {
  final ValidationResult result;
  const _ResultCard({required this.result});

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final ok = result.valid;

    return Container(
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: ok ? vendorGreenBg : vendorRedBg,
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: ok ? vendorGreen : vendorRed, width: 1.5),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(ok ? Icons.verified : Icons.cancel_outlined,
                  size: 26, color: ok ? vendorGreen : vendorRed),
              const SizedBox(width: 10),
              Expanded(
                child: Text(ok ? 'Valid code — redeem it!' : 'Code not recognised',
                    style: text.titleMedium
                        ?.copyWith(color: ok ? vendorGreen : vendorRed)),
              ),
            ],
          ),
          const SizedBox(height: 8),
          Text(result.code,
              style: text.headlineSmall?.copyWith(letterSpacing: 2, fontSize: 18)),
          if (ok) ...[
            const SizedBox(height: 8),
            Text('${result.name} · ${result.detail}', style: text.titleSmall),
            const SizedBox(height: 2),
            Text('${result.offer} (${result.value})',
                style: text.bodyMedium?.copyWith(color: SpottoColors.inkSoft)),
          ] else ...[
            const SizedBox(height: 6),
            Text('Ask the traveller to re-open the deal in their Spotto app.',
                style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
          ],
        ],
      ),
    );
  }
}

class _ReticlePainter extends CustomPainter {
  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = Colors.white
      ..strokeWidth = 4
      ..strokeCap = StrokeCap.round
      ..style = PaintingStyle.stroke;
    const len = 28.0;
    final w = size.width, h = size.height;

    void corner(Offset a, Offset b, Offset c) {
      canvas.drawPath(
          Path()
            ..moveTo(a.dx, a.dy)
            ..lineTo(b.dx, b.dy)
            ..lineTo(c.dx, c.dy),
          paint);
    }

    corner(const Offset(0, len), Offset.zero, const Offset(len, 0));
    corner(Offset(w - len, 0), Offset(w, 0), Offset(w, len));
    corner(Offset(w, h - len), Offset(w, h), Offset(w - len, h));
    corner(Offset(len, h), Offset(0, h), Offset(0, h - len));
  }

  @override
  bool shouldRepaint(_ReticlePainter oldDelegate) => false;
}
