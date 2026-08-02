// Marker-icon widgets (places, traveler avatars, location dot, callout banners),
// rasterized to bitmaps for google_maps_flutter via core/widgets/marker_bitmap.dart.

import 'package:flutter/material.dart';
import '../theme/palette.dart';

/// Spot Popup Banner Callout — renders spots as a sleek small banner
/// (icon/logo circle on the left + spot name text) with indicators for
/// deals, user favorites, and trip calendar stops.
class SpotBannerMarker extends StatelessWidget {
  final String name;
  final Color color;
  final IconData icon;
  final bool hasDeal;
  final bool isFavorite;
  final bool inTripCalendar;
  final bool matchesInterest;
  final bool emphasized;

  const SpotBannerMarker({
    super.key,
    required this.name,
    required this.color,
    required this.icon,
    this.hasDeal = false,
    this.isFavorite = false,
    this.inTripCalendar = false,
    this.matchesInterest = false,
    this.emphasized = true,
  });

  @override
  Widget build(BuildContext context) {
    final isHighPriority = matchesInterest || isFavorite || inTripCalendar;
    final opacity = emphasized || isHighPriority ? 1.0 : 0.65;
    final borderColor = inTripCalendar
        ? SpottoColors.accent
        : (isFavorite
            ? Colors.pinkAccent
            : (isHighPriority ? color : Colors.white));

    return AnimatedOpacity(
      duration: const Duration(milliseconds: 200),
      opacity: opacity,
      child: Container(
        padding: const EdgeInsets.fromLTRB(4, 4, 10, 4),
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(20),
          border:
              Border.all(color: borderColor, width: isHighPriority ? 2.0 : 1.5),
          boxShadow: const [
            BoxShadow(
              color: Color(0x2E000000),
              blurRadius: 8,
              offset: Offset(0, 3),
            ),
          ],
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            // Icon/logo circle
            Stack(
              alignment: Alignment.center,
              clipBehavior: Clip.none,
              children: [
                Container(
                  width: 28,
                  height: 28,
                  decoration: BoxDecoration(
                    color: color,
                    shape: BoxShape.circle,
                  ),
                  child: Icon(icon, color: Colors.white, size: 15),
                ),
                if (hasDeal)
                  Positioned(
                    top: -2,
                    right: -2,
                    child: Container(
                      width: 10,
                      height: 10,
                      decoration: BoxDecoration(
                        color: const Color(0xFFF6B81E),
                        shape: BoxShape.circle,
                        border: Border.all(color: Colors.white, width: 1.5),
                      ),
                    ),
                  ),
              ],
            ),
            const SizedBox(width: 6),
            // Spot Name text
            Flexible(
              child: Text(
                name,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  fontSize: 11.5,
                  fontWeight:
                      isHighPriority ? FontWeight.w800 : FontWeight.w600,
                  color: SpottoColors.ink,
                ),
              ),
            ),
            if (inTripCalendar) ...[
              const SizedBox(width: 4),
              const Icon(Icons.calendar_today_rounded,
                  size: 11, color: SpottoColors.accent),
            ] else if (isFavorite) ...[
              const SizedBox(width: 4),
              const Icon(Icons.favorite, size: 11, color: Colors.pinkAccent),
            ],
          ],
        ),
      ),
    );
  }
}

class PlacePinMarker extends StatelessWidget {
  final Color color;
  final IconData icon;
  final bool hasDeal;
  final bool emphasized;

  const PlacePinMarker({
    super.key,
    required this.color,
    required this.icon,
    required this.hasDeal,
    required this.emphasized,
  });

  @override
  Widget build(BuildContext context) {
    final size = emphasized ? 42.0 : 36.0;
    return AnimatedOpacity(
      duration: const Duration(milliseconds: 200),
      opacity: emphasized ? 1.0 : 0.55,
      child: Stack(
        alignment: Alignment.center,
        clipBehavior: Clip.none,
        children: [
          Container(
            width: size,
            height: size,
            decoration: BoxDecoration(
              color: color,
              shape: BoxShape.circle,
              border: Border.all(color: Colors.white, width: 2.5),
              boxShadow: const [
                BoxShadow(
                  color: Color(0x33000000),
                  blurRadius: 6,
                  offset: Offset(0, 3),
                ),
              ],
            ),
            child: Icon(icon, color: Colors.white, size: size * 0.48),
          ),
          if (hasDeal)
            Positioned(
              top: -1,
              right: -1,
              child: Container(
                width: 13,
                height: 13,
                decoration: BoxDecoration(
                  color: const Color(0xFFF6B81E),
                  shape: BoxShape.circle,
                  border: Border.all(color: Colors.white, width: 2),
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class AvatarPinMarker extends StatelessWidget {
  final String initial;
  final Color color;

  const AvatarPinMarker({
    super.key,
    required this.initial,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 36,
      height: 36,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: color,
        shape: BoxShape.circle,
        border: Border.all(color: Colors.white, width: 2.5),
        boxShadow: const [
          BoxShadow(
            color: Color(0x33000000),
            blurRadius: 6,
            offset: Offset(0, 3),
          ),
        ],
      ),
      child: Text(
        initial,
        style: const TextStyle(
          color: Colors.white,
          fontWeight: FontWeight.w800,
          fontSize: 14,
        ),
      ),
    );
  }
}

class CurrentLocationDot extends StatelessWidget {
  const CurrentLocationDot({super.key});

  @override
  Widget build(BuildContext context) {
    return Stack(
      alignment: Alignment.center,
      children: [
        Container(
          width: 26,
          height: 26,
          decoration: BoxDecoration(
            color: SpottoColors.accent.withValues(alpha: 0.25),
            shape: BoxShape.circle,
          ),
        ),
        Container(
          width: 14,
          height: 14,
          decoration: BoxDecoration(
            color: SpottoColors.accent,
            shape: BoxShape.circle,
            border: Border.all(color: Colors.white, width: 2.5),
            boxShadow: const [
              BoxShadow(
                color: Color(0x44000000),
                blurRadius: 4,
                offset: Offset(0, 2),
              ),
            ],
          ),
        ),
      ],
    );
  }
}

class GooglePlacePinMarker extends StatelessWidget {
  final String name;
  final String category;
  final VoidCallback onTap;

  const GooglePlacePinMarker({
    super.key,
    required this.name,
    required this.category,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: const Color(0xFF4285F4), width: 1.5),
          boxShadow: const [
            BoxShadow(
                color: Color(0x22000000), blurRadius: 6, offset: Offset(0, 2)),
          ],
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 18,
              height: 18,
              decoration: const BoxDecoration(
                color: Color(0xFF4285F4),
                shape: BoxShape.circle,
              ),
              child: const Icon(Icons.place_rounded,
                  color: Colors.white, size: 11),
            ),
            const SizedBox(width: 5),
            Flexible(
              child: Text(
                name,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                    fontSize: 10.5,
                    fontWeight: FontWeight.w700,
                    color: SpottoColors.ink),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
