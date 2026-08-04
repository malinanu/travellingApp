import 'package:flutter/material.dart';

enum AppLanguage { en, si }

class AppStrings {
  final AppLanguage language;

  const AppStrings(this.language);

  static AppStrings of(BuildContext context) {
    return const AppStrings(AppLanguage.en);
  }

  String get appName => 'Spotto';
  String get discover => language == AppLanguage.si ? 'සොයාබලන්න' : 'Discover';
  String get spots => language == AppLanguage.si ? 'ස්ථාන' : 'Spots';
  String get offers => language == AppLanguage.si ? 'දීමනා' : 'Offers';
  String get bookings => language == AppLanguage.si ? 'වෙන්කිරීම්' : 'Bookings';
  String get guide => language == AppLanguage.si ? 'මගපෙන්වීම' : 'Guide';
  String get profile => language == AppLanguage.si ? 'ගිණුම' : 'Profile';
  String get searchPlaceholder => language == AppLanguage.si ? 'ස්ථාන, ආහාර, හෝ ක්‍රියාකාරකම් සොයන්න...' : 'Search places, food, surf spots...';
  String get claimOffer => language == AppLanguage.si ? 'දීමනාව ලබාගන්න' : 'Claim Offer';
  String get redeemOffer => language == AppLanguage.si ? 'භාවිත කරන්න' : 'Redeem Offer';
}
