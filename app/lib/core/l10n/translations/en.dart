import '../app_strings.dart';

class EnStrings implements AppStrings {
  const EnStrings();

  // ── Common ──────────────────────────────────────────────────────────────
  @override
  String get save => 'Save';
  @override
  String get saved => 'Saved';
  @override
  String get cancel => 'Cancel';
  @override
  String get next => 'Next';
  @override
  String get back => 'Back';
  @override
  String get close => 'Close';
  @override
  String get done => 'Done';
  @override
  String get continueText => 'Continue';
  @override
  String get getStarted => 'Get Started';
  @override
  String get dismiss => 'Dismiss';
  @override
  String get retry => 'Retry';
  @override
  String get search => 'Search';
  @override
  String get loading => 'Loading...';
  @override
  String get error => 'Error';
  @override
  String get somethingWentWrong => 'Something went wrong';
  @override
  String get delete => 'Delete';
  @override
  String get edit => 'Edit';
  @override
  String get viewAll => 'View all';

  // ── Language Selection ──────────────────────────────────────────────────
  @override
  String get language => 'Language';
  @override
  String get chooseLanguageTitle => 'Choose your language';
  @override
  String get chooseLanguageSubtitle => 'Select your preferred language for Spotto';
  @override
  String get changeLanguage => 'Change Language';

  // ── Navigation ──────────────────────────────────────────────────────────
  @override
  String get navDiscover => 'Discover';
  @override
  String get navSpots => 'Spots';
  @override
  String get navStories => 'Stories';
  @override
  String get navTrip => 'Trip';

  // ── Discover ────────────────────────────────────────────────────────────
  @override
  String get goodMorning => 'Good morning';
  @override
  String get goodAfternoon => 'Good afternoon';
  @override
  String get goodEvening => 'Good evening';
  @override
  String get searchPlaceholder => 'Search spots, cafes, beaches...';
  @override
  String get recommendedForYou => 'Recommended for you';
  @override
  String get currentVibe => 'Current Vibe';
  @override
  String get dailyQuest => 'Daily Quest';
  @override
  String get checkIn => 'Check in';
  @override
  String get checkedIn => 'Checked in';

  // ── Categories & Spots ──────────────────────────────────────────────────
  @override
  String get categoryCafes => 'Cafes';
  @override
  String get categoryStays => 'Stays';
  @override
  String get categorySights => 'Sights';
  @override
  String get categorySurf => 'Surf';
  @override
  String get categoryCulture => 'Culture';
  @override
  String get categoryNightlife => 'Nightlife';
  @override
  String get openNow => 'Open now';
  @override
  String get closed => 'Closed';
  @override
  String distanceAway(String distance) => '$distance away';
  @override
  String get filters => 'Filters';
  @override
  String get allRegions => 'All regions';

  // ── Stories & Audio ─────────────────────────────────────────────────────
  @override
  String get audioGuides => 'Audio Guides';
  @override
  String get listen => 'Listen';
  @override
  String get playing => 'Playing';
  @override
  String get paused => 'Paused';
  @override
  String get readStory => 'Read story';

  // ── Trip ────────────────────────────────────────────────────────────────
  @override
  String get myTrip => 'My Trip';
  @override
  String get planATrip => 'Plan a trip';
  @override
  String dayNumber(int number) => 'Day $number';
  @override
  String get addStop => 'Add stop';
  @override
  String get directions => 'Directions';

  // ── Local Guide ─────────────────────────────────────────────────────────
  @override
  String get localGuide => 'Local Guide';
  @override
  String tunedFor(String style) => 'Tuned for $style';
  @override
  String get askYourLocalFriend => 'Ask your local friend...';
  @override
  String get giveMeMoreOptions => 'Give me more options';
  @override
  String get planATripWithThese => 'Plan a trip with these';
  @override
  String get clearChat => 'Clear chat';
  @override
  String get chatCleared => 'Chat history cleared';
  @override
  String get summarize => 'Summarize';

  // ── Offers & Deals ──────────────────────────────────────────────────────
  @override
  String get activeOffer => 'Active offer';
  @override
  String get claimOffer => 'Claim offer';
  @override
  String get claimed => 'Claimed';
  @override
  String get showAtCounter => 'Show at counter';
  @override
  String get hideQrCode => 'Hide QR code';
  @override
  String get tapToShowQr => 'Tap to show QR code';
  @override
  String get markAsUsed => 'Mark as used';
  @override
  String get offerUsedPerk => 'Offer used! Enjoy your perk.';
  @override
  String get rateAgainIn15Minutes => 'you can rate this place again in 15 minutes';

  // ── Profile & Settings ──────────────────────────────────────────────────
  @override
  String get profile => 'Profile';
  @override
  String get travelStyle => 'Travel style';
  @override
  String get interests => 'Interests';
  @override
  String get notifications => 'Notifications';
  @override
  String get account => 'Account';
  @override
  String get myBookings => 'My Bookings';
  @override
  String get myBookingsSubtitle => 'Upcoming & past bookings';
  @override
  String get passwordAndSecurity => 'Password & security';
  @override
  String get passwordAndSecuritySubtitle => 'Change password, 2FA';
  @override
  String get savedPayment => 'Saved payment';
  @override
  String get privacy => 'Privacy';
  @override
  String get privacySubtitle => 'Location & data sharing';
  @override
  String get helpAndSupport => 'Help & support';
  @override
  String get logOut => 'Log out';

  // ── Auth & Onboarding ───────────────────────────────────────────────────
  @override
  String get welcomeTitle => 'Welcome to Spotto';
  @override
  String get welcomeSubtitle => 'Discover Sri Lanka like a local';
  @override
  String get signIn => 'Sign In';
  @override
  String get createAccount => 'Create Account';
  @override
  String get emailAddress => 'Email address';
  @override
  String get password => 'Password';
  @override
  String get fullName => 'Full name';
  @override
  String get continueWithEmail => 'Continue with Email';
  @override
  String get tellUsYourTravelStyle => 'Tell us your travel style';
}
