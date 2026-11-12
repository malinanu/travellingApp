import '../app_strings.dart';

class DeStrings implements AppStrings {
  const DeStrings();

  // ── Common ──────────────────────────────────────────────────────────────
  @override
  String get save => 'Speichern';
  @override
  String get saved => 'Gespeichert';
  @override
  String get cancel => 'Abbrechen';
  @override
  String get next => 'Weiter';
  @override
  String get back => 'Zurück';
  @override
  String get close => 'Schließen';
  @override
  String get done => 'Fertig';
  @override
  String get continueText => 'Weiter';
  @override
  String get getStarted => 'Los geht\'s!';
  @override
  String get dismiss => 'Schließen';
  @override
  String get retry => 'Nochmal versuchen';
  @override
  String get search => 'Suchen';
  @override
  String get loading => 'Gleich da...';
  @override
  String get error => 'Fehler aufgetreten';
  @override
  String get somethingWentWrong => 'Da ist leider etwas schiefgelaufen';
  @override
  String get delete => 'Löschen';
  @override
  String get edit => 'Bearbeiten';
  @override
  String get viewAll => 'Alle anzeigen';

  // ── Language Selection ──────────────────────────────────────────────────
  @override
  String get language => 'Sprache';
  @override
  String get chooseLanguageTitle => 'Wähle deine Sprache';
  @override
  String get chooseLanguageSubtitle => 'In welcher Sprache möchtest du Spotto nutzen?';
  @override
  String get changeLanguage => 'Sprache ändern';

  // ── Navigation ──────────────────────────────────────────────────────────
  @override
  String get navDiscover => 'Entdecken';
  @override
  String get navSpots => 'Orte';
  @override
  String get navStories => 'Stories';
  @override
  String get navTrip => 'Reise';

  // ── Discover ────────────────────────────────────────────────────────────
  @override
  String get goodMorning => 'Guten Morgen';
  @override
  String get goodAfternoon => 'Schönen Nachmittag';
  @override
  String get goodEvening => 'Guten Abend';
  @override
  String get searchPlaceholder => 'Orte, Cafés, Strände finden...';
  @override
  String get recommendedForYou => 'Für dich ausgewählt';
  @override
  String get currentVibe => 'Vibe vor Ort';
  @override
  String get dailyQuest => 'Tages-Challenge';
  @override
  String get checkIn => 'Check-in';
  @override
  String get checkedIn => 'Eingecheckt';

  // ── Categories & Spots ──────────────────────────────────────────────────
  @override
  String get categoryCafes => 'Cafés';
  @override
  String get categoryStays => 'Unterkünfte';
  @override
  String get categorySights => 'Highlights';
  @override
  String get categorySurf => 'Surfen';
  @override
  String get categoryCulture => 'Kultur';
  @override
  String get categoryNightlife => 'Nachtleben';
  @override
  String get openNow => 'Jetzt geöffnet';
  @override
  String get closed => 'Geschlossen';
  @override
  String distanceAway(String distance) => '$distance entfernt';
  @override
  String get filters => 'Filter';
  @override
  String get allRegions => 'Alle Regionen';

  // ── Stories & Audio ─────────────────────────────────────────────────────
  @override
  String get audioGuides => 'Audio-Guides';
  @override
  String get listen => 'Anhören';
  @override
  String get playing => 'Wird abgespielt';
  @override
  String get paused => 'Pausiert';
  @override
  String get readStory => 'Story lesen';

  // ── Trip ────────────────────────────────────────────────────────────────
  @override
  String get myTrip => 'Meine Reise';
  @override
  String get planATrip => 'Trip planen';
  @override
  String dayNumber(int number) => 'Tag $number';
  @override
  String get addStop => 'Stopp hinzufügen';
  @override
  String get directions => 'Route anzeigen';

  // ── Local Guide ─────────────────────────────────────────────────────────
  @override
  String get localGuide => 'Local Guide';
  @override
  String tunedFor(String style) => 'Passend zu deinem Style ($style)';
  @override
  String get askYourLocalFriend => 'Frag deinen Local Guide...';
  @override
  String get giveMeMoreOptions => 'Zeig mir andere Ideen';
  @override
  String get planATripWithThese => 'Trip mit diesen Orten planen';
  @override
  String get clearChat => 'Chat leeren';
  @override
  String get chatCleared => 'Chatverlauf gelöscht';
  @override
  String get summarize => 'Zusammenfassung';

  // ── Offers & Deals ──────────────────────────────────────────────────────
  @override
  String get activeOffer => 'Dein Angebot';
  @override
  String get claimOffer => 'Angebot sichern';
  @override
  String get claimed => 'Gesichert';
  @override
  String get showAtCounter => 'An der Kasse vorzeigen';
  @override
  String get hideQrCode => 'QR-Code verbergen';
  @override
  String get tapToShowQr => 'QR-Code anzeigen';
  @override
  String get markAsUsed => 'Eingelöst';
  @override
  String get offerUsedPerk => 'Angebot eingelöst! Viel Spaß damit.';
  @override
  String get rateAgainIn15Minutes => 'Du kannst diesen Ort in 15 Minuten wieder bewerten';

  // ── Profile & Settings ──────────────────────────────────────────────────
  @override
  String get profile => 'Profil';
  @override
  String get travelStyle => 'Dein Reisestil';
  @override
  String get interests => 'Deine Interessen';
  @override
  String get notifications => 'Benachrichtigungen';
  @override
  String get account => 'Konto';
  @override
  String get myBookings => 'Meine Buchungen';
  @override
  String get myBookingsSubtitle => 'Kommende & vergangene Buchungen';
  @override
  String get passwordAndSecurity => 'Passwort & Sicherheit';
  @override
  String get passwordAndSecuritySubtitle => 'Passwort ändern, 2FA';
  @override
  String get savedPayment => 'Zahlungsmethoden';
  @override
  String get privacy => 'Datenschutz';
  @override
  String get privacySubtitle => 'Standort & geteilte Daten';
  @override
  String get helpAndSupport => 'Hilfe & Kontakt';
  @override
  String get logOut => 'Abmelden';

  // ── Auth & Onboarding ───────────────────────────────────────────────────
  @override
  String get welcomeTitle => 'Willkommen bei Spotto';
  @override
  String get welcomeSubtitle => 'Erlebe Sri Lanka wie die Locals';
  @override
  String get signIn => 'Anmelden';
  @override
  String get createAccount => 'Konto erstellen';
  @override
  String get emailAddress => 'E-Mail-Adresse';
  @override
  String get password => 'Passwort';
  @override
  String get fullName => 'Vollständiger Name';
  @override
  String get continueWithEmail => 'Weiter mit E-Mail';
  @override
  String get tellUsYourTravelStyle => 'Wie reist du am liebsten?';
}
