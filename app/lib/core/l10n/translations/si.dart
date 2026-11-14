import '../app_strings.dart';

class SiStrings implements AppStrings {
  const SiStrings();

  // ── Common ──────────────────────────────────────────────────────────────
  @override
  String get save => 'සුරකින්න';
  @override
  String get saved => 'සුරැකුණා';
  @override
  String get cancel => 'අවලංගු කරන්න';
  @override
  String get next => 'ඊළඟ';
  @override
  String get back => 'ආපසු';
  @override
  String get close => 'වසන්න';
  @override
  String get done => 'හරි';
  @override
  String get continueText => 'ඉදිරියට යන්න';
  @override
  String get getStarted => 'ආරම්භ කරමු';
  @override
  String get dismiss => 'ඉවත් කරන්න';
  @override
  String get retry => 'නැවත බලන්න';
  @override
  String get search => 'සොයන්න';
  @override
  String get loading => 'මඳක් රැඳෙන්න...';
  @override
  String get error => 'දෝෂයකි';
  @override
  String get somethingWentWrong => 'යම් දෝෂයක් සිදුවිය';
  @override
  String get delete => 'මකන්න';
  @override
  String get edit => 'වෙනස් කරන්න';
  @override
  String get viewAll => 'සියල්ල බලන්න';

  // ── Language Selection ──────────────────────────────────────────────────
  @override
  String get language => 'භාෂාව';
  @override
  String get chooseLanguageTitle => 'ඔබේ භාෂාව තෝරන්න';
  @override
  String get chooseLanguageSubtitle => 'Spotto භාවිතයට ඔබ කැමති භාෂාව තෝරන්න';
  @override
  String get changeLanguage => 'භාෂාව වෙනස් කරන්න';

  // ── Navigation ──────────────────────────────────────────────────────────
  @override
  String get navDiscover => 'ගවේෂණය';
  @override
  String get navSpots => 'ස්ථාන';
  @override
  String get navStories => 'කතා';
  @override
  String get navTrip => 'චාරිකාව';

  // ── Discover ────────────────────────────────────────────────────────────
  @override
  String get goodMorning => 'සුබ උදෑසනක්';
  @override
  String get goodAfternoon => 'සුබ දහවලක්';
  @override
  String get goodEvening => 'සුබ සැන්දෑවක්';
  @override
  String get searchPlaceholder => 'කැෆේ, වෙරළ හෝ තැන් සොයන්න...';
  @override
  String get recommendedForYou => 'ඔබ වෙනුවෙන්මයි';
  @override
  String get currentVibe => 'දැන් තියෙන වයිබ් එක';
  @override
  String get dailyQuest => 'අද මෙහෙයුම';
  @override
  String get checkIn => 'චෙක්-ඉන් වෙන්න';
  @override
  String get checkedIn => 'චෙක්-ඉන් වුණා';

  // ── Categories & Spots ──────────────────────────────────────────────────
  @override
  String get categoryCafes => 'කැෆේ';
  @override
  String get categoryStays => 'නවාතැන්';
  @override
  String get categorySights => 'නැරඹුම් ස්ථාන';
  @override
  String get categorySurf => 'සර්ෆින්';
  @override
  String get categoryCulture => 'සංස්කෘතිය';
  @override
  String get categoryNightlife => 'රාත්‍රී විනෝදය';
  @override
  String get openNow => 'දැන් විවෘතයි';
  @override
  String get closed => 'වසා ඇත';
  @override
  String distanceAway(String distance) => '$distance දුරින්';
  @override
  String get filters => 'පෙරහන්';
  @override
  String get allRegions => 'සියලු ප්‍රදේශ';

  // ── Stories & Audio ─────────────────────────────────────────────────────
  @override
  String get audioGuides => 'හඬ මඟපෙන්වීම්';
  @override
  String get listen => 'අහන්න';
  @override
  String get playing => 'වාදනය වෙනවා';
  @override
  String get paused => 'නවත්වා ඇත';
  @override
  String get readStory => 'කතාව කියවන්න';

  // ── Trip ────────────────────────────────────────────────────────────────
  @override
  String get myTrip => 'මගේ චාරිකාව';
  @override
  String get planATrip => 'චාරිකාවක් හදන්න';
  @override
  String dayNumber(int number) => '$number වන දිනය';
  @override
  String get addStop => 'නැවතුමක් එක් කරන්න';
  @override
  String get directions => 'පාර පෙන්වන්න';

  // ── Local Guide ─────────────────────────────────────────────────────────
  @override
  String get localGuide => 'දේශීය මඟපෙන්වන්නා';
  @override
  String tunedFor(String style) => '$style විලාසිතාවට අනුව';
  @override
  String get askYourLocalFriend => 'දේශීය මිතුරාගෙන් අහන්න...';
  @override
  String get giveMeMoreOptions => 'තවත් විකල්ප පෙන්වන්න';
  @override
  String get planATripWithThese => 'මේවායින් චාරිකාවක් හදන්න';
  @override
  String get clearChat => 'කතාබහ මකන්න';
  @override
  String get chatCleared => 'කතාබස් ඉතිහාසය මැකුවා';
  @override
  String get summarize => 'කෙටි සාරාංශය';

  // ── Offers & Deals ──────────────────────────────────────────────────────
  @override
  String get activeOffer => 'විශේෂ දීමනාව';
  @override
  String get claimOffer => 'දීමනාව ගන්න';
  @override
  String get claimed => 'ලබාගත්තා';
  @override
  String get showAtCounter => 'කවුන්ටරයට පෙන්වන්න';
  @override
  String get hideQrCode => 'QR කේතය සඟවන්න';
  @override
  String get tapToShowQr => 'QR කේතය පෙන්වන්න තට්ටු කරන්න';
  @override
  String get markAsUsed => 'භාවිතා කළා';
  @override
  String get offerUsedPerk => 'දීමනාව ලබාගත්තා! වාසිය භුක්ති විඳින්න.';
  @override
  String get markAsUsedTitle => 'භාවිතා කළ ලෙස සලකුණු කරන්නද?';
  @override
  String get markAsUsedConfirm => 'ඔව්, භාවිතා කළා';
  @override
  String get dismissOfferTitle => 'දීමනාව ඉවත් කරන්නද?';
  @override
  String get rateAgainIn15Minutes => 'විනාඩි 15කින් ඔබට නැවත රේට් කරන්න පුළුවන්';

  // ── Profile & Settings ──────────────────────────────────────────────────
  @override
  String get profile => 'ගිණුම';
  @override
  String get travelStyle => 'සංචාරක රටාව';
  @override
  String get interests => 'කැමති දේවල්';
  @override
  String get notifications => 'දැනුම්දීම්';
  @override
  String get account => 'ගිණුම';
  @override
  String get myBookings => 'මගේ වෙන්කිරීම්';
  @override
  String get myBookingsSubtitle => 'ඉදිරි සහ පසුගිය වෙන්කිරීම්';
  @override
  String get passwordAndSecurity => 'මුරපදය සහ ආරක්ෂාව';
  @override
  String get passwordAndSecuritySubtitle => 'මුරපදය වෙනස් කිරීම, 2FA';
  @override
  String get savedPayment => 'ගෙවීම් ක්‍රම';
  @override
  String get privacy => 'පෞද්ගලිකත්වය';
  @override
  String get privacySubtitle => 'ස්ථානය සහ දත්ත බෙදාගැනීම';
  @override
  String get helpAndSupport => 'උදව් සහ සහාය';
  @override
  String get logOut => 'පිටවන්න';

  // ── Auth & Onboarding ───────────────────────────────────────────────────
  @override
  String get welcomeTitle => 'Spotto වෙත සාදරයෙන් පිළිගනිමු';
  @override
  String get welcomeSubtitle => 'දේශීයයෙකු මෙන් ශ්‍රී ලංකාව අත්විඳින්න';
  @override
  String get signIn => 'ඇතුල් වන්න';
  @override
  String get createAccount => 'ගිණුමක් සාදන්න';
  @override
  String get emailAddress => 'විද්‍යුත් තැපෑල';
  @override
  String get password => 'මුරපදය';
  @override
  String get fullName => 'සම්පූර්ණ නම';
  @override
  String get continueWithEmail => 'විද්‍යුත් තැපෑලෙන් ඉදිරියට යන්න';
  @override
  String get tellUsYourTravelStyle => 'ඔබේ සංචාරක රටාව කියන්න';
}
