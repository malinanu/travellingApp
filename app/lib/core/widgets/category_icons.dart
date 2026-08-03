// Category icon per place's canonical categoryId — used for both the Spots
// screen's card badge and the Discover map's pins, so the same place reads
// the same icon everywhere. Resolves through the vendor-category taxonomy
// (data/vendor_categories.dart) + spottoIcon's Lucide-name→Material mapping.
//
// Previously kyed off Place.type against a fixed set of generic POI kinds
// (cafe/food/landmark/temple/wildlife/park/attraction) that never matched
// real vendor category strings — every vendor place silently fell back to
// the default pin. categoryId fixes that by matching on the same canonical
// id the taxonomy itself uses.

import 'package:flutter/material.dart';

import '../../data/vendor_categories.dart';
import 'spotto_icons.dart';

IconData categoryIcon(String categoryId) {
  final category = categoryById(categoryId);
  if (category == null) return Icons.place;
  return spottoIcon(category.icon);
}
