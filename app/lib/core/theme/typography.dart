// Fredoka for display/headings, Nunito for body — matching the prototype.

import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

import 'palette.dart';

TextTheme spottoTextTheme() {
  final fredoka = GoogleFonts.fredokaTextTheme();
  final nunito = GoogleFonts.nunitoTextTheme();
  const ink = SpottoColors.ink;

  return TextTheme(
    displayLarge: fredoka.displayLarge!.copyWith(color: ink, fontWeight: FontWeight.w600),
    displayMedium: fredoka.displayMedium!.copyWith(color: ink, fontWeight: FontWeight.w600),
    displaySmall: fredoka.displaySmall!.copyWith(color: ink, fontWeight: FontWeight.w600),
    headlineLarge: fredoka.headlineLarge!.copyWith(color: ink, fontWeight: FontWeight.w600),
    headlineMedium: fredoka.headlineMedium!.copyWith(color: ink, fontWeight: FontWeight.w600),
    headlineSmall: fredoka.headlineSmall!.copyWith(color: ink, fontWeight: FontWeight.w600),
    titleLarge: fredoka.titleLarge!.copyWith(color: ink, fontWeight: FontWeight.w600),
    titleMedium: fredoka.titleMedium!.copyWith(color: ink, fontWeight: FontWeight.w600),
    titleSmall: fredoka.titleSmall!.copyWith(color: ink, fontWeight: FontWeight.w600),
    bodyLarge: nunito.bodyLarge!.copyWith(color: ink, fontWeight: FontWeight.w600),
    bodyMedium: nunito.bodyMedium!.copyWith(color: SpottoColors.inkSoft, fontWeight: FontWeight.w600),
    bodySmall: nunito.bodySmall!.copyWith(color: SpottoColors.muted, fontWeight: FontWeight.w600),
    labelLarge: nunito.labelLarge!.copyWith(color: ink, fontWeight: FontWeight.w800),
    labelMedium: nunito.labelMedium!.copyWith(color: SpottoColors.inkSoft, fontWeight: FontWeight.w800),
    labelSmall: nunito.labelSmall!.copyWith(color: SpottoColors.muted, fontWeight: FontWeight.w700),
  );
}
