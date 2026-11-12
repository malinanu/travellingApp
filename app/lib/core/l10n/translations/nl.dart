import '../app_strings.dart';

class NlStrings implements AppStrings {
  const NlStrings();

  // ── Common ──────────────────────────────────────────────────────────────
  @override
  String get save => 'Opslaan';
  @override
  String get saved => 'Opgeslagen';
  @override
  String get cancel => 'Annuleren';
  @override
  String get next => 'Volgende';
  @override
  String get back => 'Terug';
  @override
  String get close => 'Sluiten';
  @override
  String get done => 'Klaar';
  @override
  String get continueText => 'Doorgaan';
  @override
  String get getStarted => 'Aan de slag!';
  @override
  String get dismiss => 'Sluiten';
  @override
  String get retry => 'Opnieuw proberen';
  @override
  String get search => 'Zoeken';
  @override
  String get loading => 'Even geduld...';
  @override
  String get error => 'Er ging iets mis';
  @override
  String get somethingWentWrong => 'Oeps, er is een foutje opgetreden';
  @override
  String get delete => 'Verwijderen';
  @override
  String get edit => 'Aanpassen';
  @override
  String get viewAll => 'Alles bekijken';

  // ── Language Selection ──────────────────────────────────────────────────
  @override
  String get language => 'Taal';
  @override
  String get chooseLanguageTitle => 'Kies je taal';
  @override
  String get chooseLanguageSubtitle => 'Selecteer je voorkeurstaal voor Spotto';
  @override
  String get changeLanguage => 'Taal wijzigen';

  // ── Navigation ──────────────────────────────────────────────────────────
  @override
  String get navDiscover => 'Ontdek';
  @override
  String get navSpots => 'Plekken';
  @override
  String get navStories => 'Verhalen';
  @override
  String get navTrip => 'Reis';

  // ── Discover ────────────────────────────────────────────────────────────
  @override
  String get goodMorning => 'Goedemorgen';
  @override
  String get goodAfternoon => 'Fijne middag';
  @override
  String get goodEvening => 'Goedenavond';
  @override
  String get searchPlaceholder => 'Zoek plekken, cafés, stranden...';
  @override
  String get recommendedForYou => 'Speciaal voor jou';
  @override
  String get currentVibe => 'Vibe op locatie';
  @override
  String get dailyQuest => 'Dag-challenge';
  @override
  String get checkIn => 'Check-in';
  @override
  String get checkedIn => 'Ingecheckt';

  // ── Categories & Spots ──────────────────────────────────────────────────
  @override
  String get categoryCafes => 'Cafés';
  @override
  String get categoryStays => 'Verblijven';
  @override
  String get categorySights => 'Highlights';
  @override
  String get categorySurf => 'Surfen';
  @override
  String get categoryCulture => 'Cultuur';
  @override
  String get categoryNightlife => 'Nachtleven';
  @override
  String get openNow => 'Nu open';
  @override
  String get closed => 'Gesloten';
  @override
  String distanceAway(String distance) => 'Op $distance';
  @override
  String get filters => 'Filters';
  @override
  String get allRegions => 'Alle regio\'s';

  // ── Stories & Audio ─────────────────────────────────────────────────────
  @override
  String get audioGuides => 'Audiogidsen';
  @override
  String get listen => 'Luisteren';
  @override
  String get playing => 'Speelt af';
  @override
  String get paused => 'Gepauzeerd';
  @override
  String get readStory => 'Lees verhaal';

  // ── Trip ────────────────────────────────────────────────────────────────
  @override
  String get myTrip => 'Mijn Reis';
  @override
  String get planATrip => 'Trip plannen';
  @override
  String dayNumber(int number) => 'Dag $number';
  @override
  String get addStop => 'Stop toevoegen';
  @override
  String get directions => 'Route';

  // ── Local Guide ─────────────────────────────────────────────────────────
  @override
  String get localGuide => 'Local Guide';
  @override
  String tunedFor(String style) => 'Afgestemd op jouw stijl ($style)';
  @override
  String get askYourLocalFriend => 'Vraag het je Local Guide...';
  @override
  String get giveMeMoreOptions => 'Laat meer opties zien';
  @override
  String get planATripWithThese => 'Plan een trip met deze plekken';
  @override
  String get clearChat => 'Chat wissen';
  @override
  String get chatCleared => 'Geschiedenis gewist';
  @override
  String get summarize => 'Samenvatting';

  // ── Offers & Deals ──────────────────────────────────────────────────────
  @override
  String get activeOffer => 'Jouw deal';
  @override
  String get claimOffer => 'Pak deze deal';
  @override
  String get claimed => 'Gepakt';
  @override
  String get showAtCounter => 'Laat zien aan de kassa';
  @override
  String get hideQrCode => 'QR-code verbergen';
  @override
  String get tapToShowQr => 'Toon QR-code';
  @override
  String get markAsUsed => 'Gebruikt';
  @override
  String get offerUsedPerk => 'Deal gebruikt! Veel plezier ermee.';
  @override
  String get rateAgainIn15Minutes => 'Je kunt deze plek over 15 minuten weer beoordelen';

  // ── Profile & Settings ──────────────────────────────────────────────────
  @override
  String get profile => 'Profiel';
  @override
  String get travelStyle => 'Jouw reisstijl';
  @override
  String get interests => 'Jouw interesses';
  @override
  String get notifications => 'Meldingen';
  @override
  String get account => 'Account';
  @override
  String get myBookings => 'Mijn Boekingen';
  @override
  String get myBookingsSubtitle => 'Komende & eerdere boekingen';
  @override
  String get passwordAndSecurity => 'Wachtwoord & beveiliging';
  @override
  String get passwordAndSecuritySubtitle => 'Wachtwoord wijzigen, 2FA';
  @override
  String get savedPayment => 'Betaalmethoden';
  @override
  String get privacy => 'Privacy';
  @override
  String get privacySubtitle => 'Locatie & gedeelde gegevens';
  @override
  String get helpAndSupport => 'Hulp & contact';
  @override
  String get logOut => 'Uitloggen';

  // ── Auth & Onboarding ───────────────────────────────────────────────────
  @override
  String get welcomeTitle => 'Welkom bij Spotto';
  @override
  String get welcomeSubtitle => 'Ervaar Sri Lanka als een local';
  @override
  String get signIn => 'Inloggen';
  @override
  String get createAccount => 'Account aanmaken';
  @override
  String get emailAddress => 'E-mailadres';
  @override
  String get password => 'Wachtwoord';
  @override
  String get fullName => 'Volledige naam';
  @override
  String get continueWithEmail => 'Verder met e-mail';
  @override
  String get tellUsYourTravelStyle => 'Hoe reis jij het liefst?';
}
