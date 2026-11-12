import '../app_strings.dart';

class FrStrings implements AppStrings {
  const FrStrings();

  // ── Common ──────────────────────────────────────────────────────────────
  @override
  String get save => 'Enregistrer';
  @override
  String get saved => 'Enregistré';
  @override
  String get cancel => 'Annuler';
  @override
  String get next => 'Suivant';
  @override
  String get back => 'Retour';
  @override
  String get close => 'Fermer';
  @override
  String get done => 'Terminé';
  @override
  String get continueText => 'Continuer';
  @override
  String get getStarted => 'C\'est parti !';
  @override
  String get dismiss => 'Ignorer';
  @override
  String get retry => 'Réessayer';
  @override
  String get search => 'Rechercher';
  @override
  String get loading => 'Chargement...';
  @override
  String get error => 'Oups, une erreur est survenue';
  @override
  String get somethingWentWrong => 'Un petit souci est survenu';
  @override
  String get delete => 'Supprimer';
  @override
  String get edit => 'Modifier';
  @override
  String get viewAll => 'Tout voir';

  // ── Language Selection ──────────────────────────────────────────────────
  @override
  String get language => 'Langue';
  @override
  String get chooseLanguageTitle => 'Choisissez votre langue';
  @override
  String get chooseLanguageSubtitle => 'Sélectionnez votre langue préférée pour Spotto';
  @override
  String get changeLanguage => 'Changer de langue';

  // ── Navigation ──────────────────────────────────────────────────────────
  @override
  String get navDiscover => 'Découvrir';
  @override
  String get navSpots => 'Lieux';
  @override
  String get navStories => 'Histoires';
  @override
  String get navTrip => 'Voyage';

  // ── Discover ────────────────────────────────────────────────────────────
  @override
  String get goodMorning => 'Bonjour';
  @override
  String get goodAfternoon => 'Bon après-midi';
  @override
  String get goodEvening => 'Bonsoir';
  @override
  String get searchPlaceholder => 'Rechercher un lieu, un café, une plage...';
  @override
  String get recommendedForYou => 'Spécialement pour vous';
  @override
  String get currentVibe => 'Ambiance du lieu';
  @override
  String get dailyQuest => 'Défi du jour';
  @override
  String get checkIn => 'Check-in';
  @override
  String get checkedIn => 'Enregistré sur place';

  // ── Categories & Spots ──────────────────────────────────────────────────
  @override
  String get categoryCafes => 'Cafés';
  @override
  String get categoryStays => 'Hébergements';
  @override
  String get categorySights => 'Incontournables';
  @override
  String get categorySurf => 'Surf';
  @override
  String get categoryCulture => 'Culture';
  @override
  String get categoryNightlife => 'Vie nocturne';
  @override
  String get openNow => 'Ouvert';
  @override
  String get closed => 'Fermé';
  @override
  String distanceAway(String distance) => 'À $distance';
  @override
  String get filters => 'Filtres';
  @override
  String get allRegions => 'Toutes les régions';

  // ── Stories & Audio ─────────────────────────────────────────────────────
  @override
  String get audioGuides => 'Guides audio';
  @override
  String get listen => 'Écouter';
  @override
  String get playing => 'En lecture';
  @override
  String get paused => 'En pause';
  @override
  String get readStory => 'Lire l\'histoire';

  // ── Trip ────────────────────────────────────────────────────────────────
  @override
  String get myTrip => 'Mon Voyage';
  @override
  String get planATrip => 'Créer un voyage';
  @override
  String dayNumber(int number) => 'Jour $number';
  @override
  String get addStop => 'Ajouter une étape';
  @override
  String get directions => 'Itinéraire';

  // ── Local Guide ─────────────────────────────────────────────────────────
  @override
  String get localGuide => 'Guide Local';
  @override
  String tunedFor(String style) => 'Adapté à votre style $style';
  @override
  String get askYourLocalFriend => 'Posez une question à votre guide...';
  @override
  String get giveMeMoreOptions => 'Donne-moi d\'autres idées';
  @override
  String get planATripWithThese => 'Créer un voyage avec ces lieux';
  @override
  String get clearChat => 'Effacer l\'échange';
  @override
  String get chatCleared => 'Historique effacé';
  @override
  String get summarize => 'Résumé';

  // ── Offers & Deals ──────────────────────────────────────────────────────
  @override
  String get activeOffer => 'Bon plan disponible';
  @override
  String get claimOffer => 'Profiter de l\'offre';
  @override
  String get claimed => 'Offre activée';
  @override
  String get showAtCounter => 'Présentez à la caisse';
  @override
  String get hideQrCode => 'Masquer le QR code';
  @override
  String get tapToShowQr => 'Afficher le QR code';
  @override
  String get markAsUsed => 'Déjà utilisé';
  @override
  String get offerUsedPerk => 'Offre utilisée ! Bon shopping.';
  @override
  String get rateAgainIn15Minutes => 'Vous pourrez à nouveau noter ce lieu dans 15 minutes';

  // ── Profile & Settings ──────────────────────────────────────────────────
  @override
  String get profile => 'Profil';
  @override
  String get travelStyle => 'Votre style de voyage';
  @override
  String get interests => 'Ce que vous aimez';
  @override
  String get notifications => 'Notifications';
  @override
  String get account => 'Compte';
  @override
  String get myBookings => 'Mes réservations';
  @override
  String get myBookingsSubtitle => 'Réservations passées et à venir';
  @override
  String get passwordAndSecurity => 'Mot de passe et sécurité';
  @override
  String get passwordAndSecuritySubtitle => 'Changer le mot de passe, 2FA';
  @override
  String get savedPayment => 'Moyens de paiement';
  @override
  String get privacy => 'Confidentialité';
  @override
  String get privacySubtitle => 'Partage de votre position et données';
  @override
  String get helpAndSupport => 'Aide et contact';
  @override
  String get logOut => 'Se déconnecter';

  // ── Auth & Onboarding ───────────────────────────────────────────────────
  @override
  String get welcomeTitle => 'Bienvenue sur Spotto';
  @override
  String get welcomeSubtitle => 'Vivez le Sri Lanka comme un local';
  @override
  String get signIn => 'Connexion';
  @override
  String get createAccount => 'Créer un compte';
  @override
  String get emailAddress => 'E-mail';
  @override
  String get password => 'Mot de passe';
  @override
  String get fullName => 'Nom complet';
  @override
  String get continueWithEmail => 'Continuer par e-mail';
  @override
  String get tellUsYourTravelStyle => 'Comment aimez-vous voyager ?';
}
