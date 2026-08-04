// Persisted theme-mode preference (system / light / dark).
// Stored in SharedPreferences so it survives app restarts.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../storage/local_store.dart';

const _kThemeModeKey = 'theme_mode'; // 'system' | 'light' | 'dark'

/// Reads / writes the user's ThemeMode preference.
class ThemeModeNotifier extends Notifier<ThemeMode> {
  @override
  ThemeMode build() {
    // Read persisted value synchronously from the already-initialised store.
    final raw = ref.read(localStoreProvider).prefs.getString(_kThemeModeKey);
    return _fromString(raw);
  }

  void setMode(ThemeMode mode) {
    state = mode;
    ref.read(localStoreProvider).prefs.setString(_kThemeModeKey, _toString(mode));
  }

  void toggle() {
    switch (state) {
      case ThemeMode.system:
        setMode(ThemeMode.light);
      case ThemeMode.light:
        setMode(ThemeMode.dark);
      case ThemeMode.dark:
        setMode(ThemeMode.system);
    }
  }
}

final themeModeProvider = NotifierProvider<ThemeModeNotifier, ThemeMode>(
  ThemeModeNotifier.new,
);

ThemeMode _fromString(String? raw) {
  switch (raw) {
    case 'light':
      return ThemeMode.light;
    case 'dark':
      return ThemeMode.dark;
    default:
      return ThemeMode.system;
  }
}

String _toString(ThemeMode mode) {
  switch (mode) {
    case ThemeMode.light:
      return 'light';
    case ThemeMode.dark:
      return 'dark';
    case ThemeMode.system:
      return 'system';
  }
}
