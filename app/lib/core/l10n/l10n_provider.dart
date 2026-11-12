import 'dart:ui' as ui;
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../data/repositories/repositories.dart';
import '../storage/local_store.dart';
import 'app_language.dart';
import 'app_strings.dart';
import 'translations/translations.dart';

class AppLanguageNotifier extends Notifier<AppLanguage> {
  @override
  AppLanguage build() {
    final store = ref.read(localStoreProvider);
    final savedCode = store.getString(StoreKeys.appLanguage);
    if (savedCode != null && savedCode.isNotEmpty) {
      return AppLanguage.fromCode(savedCode);
    }
    // Default to device locale if recognized, else English.
    final deviceLocale = ui.PlatformDispatcher.instance.locale.languageCode;
    return AppLanguage.fromCode(deviceLocale);
  }

  Future<void> setLanguage(AppLanguage language) async {
    state = language;
    final store = ref.read(localStoreProvider);
    await store.setString(StoreKeys.appLanguage, language.code);
    await store.setBool(StoreKeys.hasSelectedLanguage, true);
    ref.read(hasSelectedLanguageProvider.notifier).markSelectedSync();
  }
}

final appLanguageProvider = NotifierProvider<AppLanguageNotifier, AppLanguage>(
  AppLanguageNotifier.new,
);

class HasSelectedLanguageNotifier extends Notifier<bool> {
  @override
  bool build() {
    final store = ref.read(localStoreProvider);
    if (store.getBool(StoreKeys.hasSelectedLanguage)) return true;
    if (store.getString(StoreKeys.appLanguage) != null) return true;
    // Returning users who already have an active auth session shouldn't be blocked.
    if (store.getToken() != null) return true;
    return false;
  }

  void markSelectedSync() {
    state = true;
  }

  Future<void> markSelected() async {
    state = true;
    final store = ref.read(localStoreProvider);
    await store.setBool(StoreKeys.hasSelectedLanguage, true);
  }
}

final hasSelectedLanguageProvider = NotifierProvider<HasSelectedLanguageNotifier, bool>(
  HasSelectedLanguageNotifier.new,
);

/// Active [AppStrings] translation dictionary for the current language.
final l10nProvider = Provider<AppStrings>((ref) {
  final currentLang = ref.watch(appLanguageProvider);
  return lookupStrings(currentLang);
});

/// Convenient extension on [WidgetRef] to access current strings without verbose syntax.
extension WidgetRefL10n on WidgetRef {
  AppStrings get l10n => watch(l10nProvider);
}
