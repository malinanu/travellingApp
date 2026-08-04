import 'package:flutter/material.dart';

import 'dark_palette.dart';
import 'palette.dart';
import 'typography.dart';

ThemeData spottoTheme() {
  final textTheme = spottoTextTheme();
  return ThemeData(
    useMaterial3: true,
    brightness: Brightness.light,
    scaffoldBackgroundColor: SpottoColors.sand,
    colorScheme: ColorScheme.fromSeed(
      seedColor: SpottoColors.accent,
      brightness: Brightness.light,
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
      showDragHandle: false,
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

/// Dark variant of the Spotto theme — used when ThemeMode.dark is active.
ThemeData spottoThemeDark() {
  final textTheme = spottoTextTheme(dark: true);
  return ThemeData(
    useMaterial3: true,
    brightness: Brightness.dark,
    scaffoldBackgroundColor: SpottoDarkColors.sand,
    colorScheme: ColorScheme.fromSeed(
      seedColor: SpottoDarkColors.accent,
      brightness: Brightness.dark,
      primary: SpottoDarkColors.accent,
      secondary: SpottoDarkColors.accent2,
      surface: SpottoDarkColors.surface,
      onSurface: SpottoDarkColors.ink,
      outline: SpottoDarkColors.line,
    ),
    textTheme: textTheme,
    cardTheme: const CardThemeData(
      color: SpottoDarkColors.surface,
      elevation: 0,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.all(Radius.circular(20))),
      margin: EdgeInsets.zero,
    ),
    bottomSheetTheme: const BottomSheetThemeData(
      backgroundColor: SpottoDarkColors.sand,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(28)),
      ),
      showDragHandle: false,
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: SpottoDarkColors.accent,
        foregroundColor: SpottoDarkColors.sand,
        shape: const StadiumBorder(),
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
        textStyle: textTheme.labelLarge?.copyWith(fontSize: 16),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: SpottoDarkColors.accent,
        side: const BorderSide(color: SpottoDarkColors.line, width: 1.5),
        shape: const StadiumBorder(),
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 14),
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: SpottoDarkColors.surface,
      contentPadding: const EdgeInsets.symmetric(horizontal: 18, vertical: 15),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(16),
        borderSide: const BorderSide(color: SpottoDarkColors.line, width: 1.5),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(16),
        borderSide: const BorderSide(color: SpottoDarkColors.line, width: 1.5),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(16),
        borderSide: const BorderSide(color: SpottoDarkColors.accent, width: 2),
      ),
      hintStyle: textTheme.bodyMedium?.copyWith(color: SpottoDarkColors.muted),
    ),
    dividerTheme: const DividerThemeData(color: SpottoDarkColors.line, thickness: 1),
    chipTheme: ChipThemeData(
      backgroundColor: SpottoDarkColors.accentTint,
      labelStyle: textTheme.labelMedium?.copyWith(color: SpottoDarkColors.accent),
      shape: const StadiumBorder(),
      side: BorderSide.none,
    ),
  );
}
