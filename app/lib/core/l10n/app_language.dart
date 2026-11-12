/// Supported languages for the Spotto traveller app.
enum AppLanguage {
  en(
    code: 'en',
    nativeName: 'English',
    englishName: 'English',
    flag: '🇬🇧',
  ),
  nl(
    code: 'nl',
    nativeName: 'Nederlands',
    englishName: 'Dutch',
    flag: '🇳🇱',
  ),
  de(
    code: 'de',
    nativeName: 'Deutsch',
    englishName: 'German',
    flag: '🇩🇪',
  ),
  fr(
    code: 'fr',
    nativeName: 'Français',
    englishName: 'French',
    flag: '🇫🇷',
  ),
  ru(
    code: 'ru',
    nativeName: 'Русский',
    englishName: 'Russian',
    flag: '🇷🇺',
  ),
  si(
    code: 'si',
    nativeName: 'සිංහල',
    englishName: 'Sinhala',
    flag: '🇱🇰',
  );

  final String code;
  final String nativeName;
  final String englishName;
  final String flag;

  const AppLanguage({
    required this.code,
    required this.nativeName,
    required this.englishName,
    required this.flag,
  });

  /// Resolves an [AppLanguage] from an ISO code (e.g. 'en', 'nl', 'de', 'fr', 'ru', 'si').
  /// Defaults to [AppLanguage.en] if unknown or null.
  static AppLanguage fromCode(String? code) {
    if (code == null) return AppLanguage.en;
    final normalized = code.trim().toLowerCase().split(RegExp('[-_]')).first;
    for (final lang in AppLanguage.values) {
      if (lang.code == normalized) return lang;
    }
    return AppLanguage.en;
  }
}
