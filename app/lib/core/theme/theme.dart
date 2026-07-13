import 'package:flutter/material.dart';

import 'palette.dart';
import 'typography.dart';

ThemeData spottoTheme() {
  final textTheme = spottoTextTheme();
  return ThemeData(
    useMaterial3: true,
    scaffoldBackgroundColor: SpottoColors.sand,
    colorScheme: ColorScheme.fromSeed(
      seedColor: SpottoColors.accent,
      primary: SpottoColors.accent,
      secondary: SpottoColors.accent2,
      surface: Colors.white,
      onSurface: SpottoColors.ink,
      outline: SpottoColors.line,
    ),
    textTheme: textTheme,
    cardTheme: const CardThemeData(
      color: Colors.white,
      elevation: 0,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(20))),
      margin: EdgeInsets.zero,
    ),
    bottomSheetTheme: const BottomSheetThemeData(
      backgroundColor: SpottoColors.sand,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(28)),
      ),
      showDragHandle: false, // SheetScaffold draws its own grab handle
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: SpottoColors.accent,
        foregroundColor: Colors.white,
        shape: const StadiumBorder(),
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
        textStyle: textTheme.labelLarge?.copyWith(fontSize: 16),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: SpottoColors.accent,
        side: const BorderSide(color: SpottoColors.line, width: 1.5),
        shape: const StadiumBorder(),
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: Colors.white,
      contentPadding: const EdgeInsets.symmetric(horizontal: 18, vertical: 15),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(16),
        borderSide: const BorderSide(color: SpottoColors.line, width: 1.5),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(16),
        borderSide: const BorderSide(color: SpottoColors.line, width: 1.5),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(16),
        borderSide: const BorderSide(color: SpottoColors.accent, width: 2),
      ),
      hintStyle: textTheme.bodyMedium?.copyWith(color: SpottoColors.muted),
    ),
    dividerTheme: const DividerThemeData(color: SpottoColors.line, thickness: 1),
    chipTheme: ChipThemeData(
      backgroundColor: SpottoColors.accentTint,
      labelStyle: textTheme.labelMedium?.copyWith(color: SpottoColors.accent),
      shape: const StadiumBorder(),
      side: BorderSide.none,
    ),
  );
}
