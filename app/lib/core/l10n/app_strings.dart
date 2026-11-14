/// Base contract for all translated strings across Spotto.
/// Adding a new language simply requires implementing this interface.
abstract class AppStrings {
  // ── Common ──────────────────────────────────────────────────────────────
  String get save;
  String get saved;
  String get cancel;
  String get next;
  String get back;
  String get close;
  String get done;
  String get continueText;
  String get getStarted;
  String get dismiss;
  String get retry;
  String get search;
  String get loading;
  String get error;
  String get somethingWentWrong;
  String get delete;
  String get edit;
  String get viewAll;

  // ── Language Selection ──────────────────────────────────────────────────
  String get language;
  String get chooseLanguageTitle;
  String get chooseLanguageSubtitle;
  String get changeLanguage;

  // ── Navigation ──────────────────────────────────────────────────────────
  String get navDiscover;
  String get navSpots;
  String get navStories;
  String get navTrip;

  // ── Discover ────────────────────────────────────────────────────────────
  String get goodMorning;
  String get goodAfternoon;
  String get goodEvening;
  String get searchPlaceholder;
  String get recommendedForYou;
  String get currentVibe;
  String get dailyQuest;
  String get checkIn;
  String get checkedIn;

  // ── Categories & Spots ──────────────────────────────────────────────────
  String get categoryCafes;
  String get categoryStays;
  String get categorySights;
  String get categorySurf;
  String get categoryCulture;
  String get categoryNightlife;
  String get openNow;
  String get closed;
  String distanceAway(String distance);
  String get filters;
  String get allRegions;

  // ── Stories & Audio ─────────────────────────────────────────────────────
  String get audioGuides;
  String get listen;
  String get playing;
  String get paused;
  String get readStory;

  // ── Trip ────────────────────────────────────────────────────────────────
  String get myTrip;
  String get planATrip;
  String dayNumber(int number);
  String get addStop;
  String get directions;

  // ── Local Guide ─────────────────────────────────────────────────────────
  String get localGuide;
  String tunedFor(String style);
  String get askYourLocalFriend;
  String get giveMeMoreOptions;
  String get planATripWithThese;
  String get clearChat;
  String get chatCleared;
  String get summarize;

  // ── Offers & Deals ──────────────────────────────────────────────────────
  String get activeOffer;
  String get claimOffer;
  String get claimed;
  String get showAtCounter;
  String get hideQrCode;
  String get tapToShowQr;
  String get markAsUsed;
  String get offerUsedPerk;
  String get markAsUsedTitle;
  String get markAsUsedConfirm;
  String get dismissOfferTitle;
  String get rateAgainIn15Minutes;

  // ── Profile & Settings ──────────────────────────────────────────────────
  String get profile;
  String get travelStyle;
  String get interests;
  String get notifications;
  String get account;
  String get myBookings;
  String get myBookingsSubtitle;
  String get passwordAndSecurity;
  String get passwordAndSecuritySubtitle;
  String get savedPayment;
  String get privacy;
  String get privacySubtitle;
  String get helpAndSupport;
  String get logOut;

  // ── Auth & Onboarding ───────────────────────────────────────────────────
  String get welcomeTitle;
  String get welcomeSubtitle;
  String get signIn;
  String get createAccount;
  String get emailAddress;
  String get password;
  String get fullName;
  String get continueWithEmail;
  String get tellUsYourTravelStyle;
}
