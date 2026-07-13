// Shared bits for the trip planner: traveller resolution + avatar bubble.

import 'package:flutter/material.dart';

import '../../../../data/mock/places.dart' as data;
import '../../../../data/models/models.dart';

/// Resolve a traveller id → Traveler; 'me'/null → [data.me].
Traveler? tripWho(String? id) {
  if (id == null || id == 'me') return data.me;
  for (final t in data.travelers) {
    if (t.id == id) return t;
  }
  return null;
}

/// Round initials avatar matching the prototype's Bubble.
class AvatarBubble extends StatelessWidget {
  final String initial;
  final Color color;
  final double size;
  final double borderWidth;

  const AvatarBubble({
    super.key,
    required this.initial,
    required this.color,
    this.size = 40,
    this.borderWidth = 2.5,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      width: size,
      height: size,
      alignment: Alignment.center,
      decoration: BoxDecoration(
        color: color,
        shape: BoxShape.circle,
        border: borderWidth > 0 ? Border.all(color: Colors.white, width: borderWidth) : null,
      ),
      child: Text(
        initial,
        style: TextStyle(
          color: Colors.white,
          fontWeight: FontWeight.w800,
          fontSize: size * (initial.length > 1 ? 0.34 : 0.42),
        ),
      ),
    );
  }
}
