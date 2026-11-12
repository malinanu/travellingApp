import '../app_language.dart';
import '../app_strings.dart';
import 'de.dart';
import 'en.dart';
import 'fr.dart';
import 'nl.dart';
import 'ru.dart';
import 'si.dart';

export 'de.dart';
export 'en.dart';
export 'fr.dart';
export 'nl.dart';
export 'ru.dart';
export 'si.dart';

/// Registry mapping supported languages to their [AppStrings] translation instances.
const Map<AppLanguage, AppStrings> translations = {
  AppLanguage.en: EnStrings(),
  AppLanguage.nl: NlStrings(),
  AppLanguage.de: DeStrings(),
  AppLanguage.fr: FrStrings(),
  AppLanguage.ru: RuStrings(),
  AppLanguage.si: SiStrings(),
};

/// Returns the [AppStrings] instance for the provided [AppLanguage].
/// Defaults to English if an unmapped language is passed.
AppStrings lookupStrings(AppLanguage language) {
  return translations[language] ?? const EnStrings();
}
