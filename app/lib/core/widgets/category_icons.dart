// Category icon per real Place.type value — used for both the Spots
// screen's card badge and the Discover map's pins, so the same place
// reads the same icon everywhere.

import 'package:flutter/material.dart';

IconData categoryIcon(String type) {
  switch (type) {
    case 'cafe':
      return Icons.local_cafe;
    case 'food':
      return Icons.restaurant;
    case 'landmark':
      return Icons.account_balance;
    case 'temple':
      return Icons.temple_buddhist;
    case 'wildlife':
      return Icons.pets;
    case 'park':
      return Icons.park;
    case 'attraction':
      return Icons.attractions;
    default:
      return Icons.place;
  }
}
