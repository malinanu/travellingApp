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
  String get markAsUsedTitle => 'Als gebruikt markeren?';
  @override
  String get markAsUsedConfirm => 'Ja, markeer als gebruikt';
  @override
  String get dismissOfferTitle => 'Deal verwijderen?';
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

  // ── Bookings, payments & experiences ─────────────────────────────────────
  @override
  String get bookHeading => 'Boeken';
  @override
  String get partySizeLabel => 'Aantal personen';
  @override
  String get guestsLabel => 'Gasten';
  @override
  String get quantityLabel => 'Aantal';
  @override
  String get checkInLabel => 'Inchecken';
  @override
  String get checkOutLabel => 'Uitchecken';
  @override
  String get selectDateLabel => 'Kiezen';
  @override
  String get noAvailabilityThisDay => 'Geen plek op deze dag';
  @override
  String get couldNotLoadAvailability => 'Beschikbaarheid laden mislukt';
  @override
  String get noUpcomingDates => 'Geen komende data';
  @override
  String spotsLeft(int n) => 'nog $n';
  @override
  String get bookFree => 'Gratis boeken';
  @override
  String bookAndPay(String amount) => 'Boek & betaal $amount';
  @override
  String get payWithLabel => 'Betalen met';
  @override
  String chargedInCurrency(String currency) => 'Afgerekend in $currency tegen de koers van vandaag. Je bank kan eigen kosten rekenen.';
  @override
  String chargedAs(String amount) => 'Je betaalt $amount';
  @override
  String get bookingInProgress => 'Bezig met boeken…';
  @override
  String get priceFree => 'Gratis';
  @override
  String pricePerPerson(String amount) => '$amount per persoon';
  @override
  String pricePerBooking(String amount) => '$amount per boeking';
  @override
  String pricePerNight(String amount) => '$amount per nacht';
  @override
  String get totalLabel => 'Totaal';
  @override
  String freeCancellationUntil(int hours) => 'Gratis annuleren tot $hours uur vooraf';
  @override
  String lateCancelRefund(int pct) => 'Daarna $pct% terug';
  @override
  String get lateCancelNoRefund => 'Daarna geen terugbetaling';
  @override
  String get meetingPointLabel => 'Verzamelpunt';
  @override
  String get bookedShowAtCheckIn => 'Geboekt: laat dit zien bij aankomst';
  @override
  String partyOf(int n) => '$n personen';
  @override
  String get payTitle => 'Rond je betaling af';
  @override
  String payHeldUntil(String time) => 'Je plek is vastgehouden tot $time';
  @override
  String get paySecurely => 'Veilig betalen';
  @override
  String get payWaiting => 'Wachten tot je betaling binnen is…';
  @override
  String get payExpired => 'De reservering verliep voordat je betaling binnen was. Boek opnieuw voor een nieuw tijdstip.';
  @override
  String get payNotYet => 'We hebben de betaling nog niet ontvangen.';
  @override
  String get payAgain => 'Opnieuw betalen';
  @override
  String get payCardNote => 'Je betaalt op de pagina van de betaalprovider. Spotto ziet je kaartgegevens nooit.';
  @override
  String get couldNotOpenPayment => 'Betaalpagina openen mislukt';
  @override
  String get upcomingSection => 'Gepland';
  @override
  String get pastSection => 'Afgelopen';
  @override
  String get noBookingsYet => 'Nog geen boekingen';
  @override
  String get noBookingsHint => 'Boek een tafel, surfles, kamer of tour van een local via de pagina van een plek.';
  @override
  String get completePayment => 'Betaling afronden';
  @override
  String get cancelBookingAction => 'Boeking annuleren';
  @override
  String get cancellingBooking => 'Annuleren…';
  @override
  String get cancelBookingTitle => 'Deze boeking annuleren?';
  @override
  String refundFullBack(String amount) => 'Je krijgt $amount terug.';
  @override
  String refundPartBack(String refund, String paid) => 'Je krijgt $refund van $paid terug.';
  @override
  String get refundNone => 'De terugbetaaltermijn is verstreken, je krijgt niets terug.';
  @override
  String get keepBooking => 'Behouden';
  @override
  String get confirmCancelBooking => 'Ja, annuleren';
  @override
  String cancelledWithRefund(String amount) => 'Geannuleerd. $amount komt naar je terug.';
  @override
  String get bookingCancelled => 'Boeking geannuleerd.';
  @override
  String refundedAmount(String amount) => '$amount terugbetaald';
  @override
  String paidAmount(String amount) => '$amount betaald';
  @override
  String get statusAwaitingPayment => 'Wacht op betaling';
  @override
  String get statusConfirmed => 'Bevestigd';
  @override
  String get statusCancelled => 'Geannuleerd';
  @override
  String get statusCompleted => 'Voltooid';
  @override
  String get statusNoShow => 'Niet verschenen';
  @override
  String get statusExpired => 'Verlopen';
  @override
  String get experiencesTitle => 'Ervaringen & evenementen';
  @override
  String get experiencesSubtitle => 'Tours, lessen en avonden uit, georganiseerd door locals';
  @override
  String get experiencesEmpty => 'Hier valt nog niets te boeken. Kom snel terug.';
  @override
  String get allCategoriesFilter => 'Alles';
  @override
  String nextDateLabel(String date) => 'Volgende: $date';
  @override
  String get localHostBadge => 'Local host';
  @override
  String get experiencesEntrySubtitle => 'Surfen, walvissen spotten, koken, kroegentochten…';
  @override
  String get payAtCheckoutTitle => 'Je betaalt bij het afrekenen';
  @override
  String get payAtCheckoutBody => 'Boek je iets betaalds, dan betaal je op de beveiligde pagina van de betaalprovider. Spotto bewaart geen kaarten.';
}
