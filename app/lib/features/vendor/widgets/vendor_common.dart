// Shared vendor-portal building blocks: cards, tones, status colors.

import 'package:flutter/material.dart';

import '../../../core/theme/palette.dart';

const vendorGreen = Color(0xFF16A34A);
const vendorGreenBg = Color(0xFFE7F6EC);
const vendorRed = Color(0xFFDC4A2D);
const vendorRedBg = Color(0xFFFDE8E4);

/// KPI tone map from the prototype.
const kpiTones = {
  'teal': (fg: Color(0xFF0B6F86), bg: Color(0xFFD6EFEC)),
  'coral': (fg: Color(0xFFF0682F), bg: Color(0xFFFFE5D6)),
  'sun': (fg: Color(0xFFC98A00), bg: Color(0xFFFBEEC4)),
};

/// Travel-style dot colors used in redemption feeds.
Color styleDot(String style) => switch (style) {
      'Digital Nomad' => const Color(0xFF0B6F86),
      'Surf Traveler' => const Color(0xFFF0682F),
      'Luxury' => const Color(0xFFC98A00),
      'Backpacker' => const Color(0xFF5A5F9E),
      _ => SpottoColors.muted,
    };

/// Offer status chip colors.
({Color fg, Color bg, String label}) offerStatus(String status) => switch (status) {
      'live' => (fg: vendorGreen, bg: vendorGreenBg, label: 'Live'),
      'ending' => (fg: vendorRed, bg: vendorRedBg, label: 'Ending soon'),
      'draft' => (fg: SpottoColors.muted, bg: Color(0x14103D3A), label: 'Draft'),
      _ => (fg: SpottoColors.muted, bg: Color(0x14103D3A), label: 'Ended'),
    };

/// Flat white card with the vendor portal's hairline border.
class VendorCard extends StatelessWidget {
  final Widget child;
  final EdgeInsetsGeometry padding;
  const VendorCard({super.key, required this.child, this.padding = const EdgeInsets.all(18)});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: double.infinity,
      padding: padding,
      decoration: BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: SpottoColors.line),
        boxShadow: const [
          BoxShadow(color: Color(0x0A103D3A), blurRadius: 2, offset: Offset(0, 1))
        ],
      ),
      child: child,
    );
  }
}

class StatusChip extends StatelessWidget {
  final String status;
  const StatusChip({super.key, required this.status});

  @override
  Widget build(BuildContext context) {
    final s = offerStatus(status);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(color: s.bg, borderRadius: BorderRadius.circular(999)),
      child: Text(s.label.toUpperCase(),
          style: TextStyle(
              fontSize: 10, fontWeight: FontWeight.w800, letterSpacing: 0.4, color: s.fg)),
    );
  }
}
