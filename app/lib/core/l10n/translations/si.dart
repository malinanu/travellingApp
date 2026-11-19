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

  // ── Bookings, payments & experiences ─────────────────────────────────────
  @override
  String get bookHeading => 'වෙන්කරන්න';
  @override
  String get partySizeLabel => 'පුද්ගලයින් ගණන';
  @override
  String get guestsLabel => 'අමුත්තන්';
  @override
  String get quantityLabel => 'ප්‍රමාණය';
  @override
  String get checkInLabel => 'පැමිණීම';
  @override
  String get checkOutLabel => 'පිටවීම';
  @override
  String get selectDateLabel => 'තෝරන්න';
  @override
  String get noAvailabilityThisDay => 'මෙදින ඉඩ නැත';
  @override
  String get couldNotLoadAvailability => 'ලබාගත හැකි වේලාවන් පූරණය කළ නොහැක';
  @override
  String get noUpcomingDates => 'ඉදිරි දින නැත';
  @override
  String spotsLeft(int n) => '$n ක් ඉතිරියි';
  @override
  String get bookFree => 'නොමිලේ වෙන්කරන්න';
  @override
  String bookAndPay(String amount) => 'වෙන්කර $amount ගෙවන්න';
  @override
  String get bookingInProgress => 'වෙන්කරමින්…';
  @override
  String get priceFree => 'නොමිලේ';
  @override
  String pricePerPerson(String amount) => 'එක් අයෙකුට $amount';
  @override
  String pricePerBooking(String amount) => 'වෙන්කිරීමකට $amount';
  @override
  String pricePerNight(String amount) => 'රාත්‍රියකට $amount';
  @override
  String get totalLabel => 'මුළු මුදල';
  @override
  String freeCancellationUntil(int hours) => 'පැය $hoursකට පෙර දක්වා නොමිලේ අවලංගු කළ හැක';
  @override
  String lateCancelRefund(int pct) => 'ඉන්පසු $pct% ආපසු';
  @override
  String get lateCancelNoRefund => 'ඉන්පසු මුදල් ආපසු නොදේ';
  @override
  String get meetingPointLabel => 'හමුවන ස්ථානය';
  @override
  String get bookedShowAtCheckIn => 'වෙන්කළා: පැමිණි විට මෙය පෙන්වන්න';
  @override
  String partyOf(int n) => 'පුද්ගලයින් $n';
  @override
  String get payTitle => 'ඔබේ ගෙවීම සම්පූර්ණ කරන්න';
  @override
  String payHeldUntil(String time) => '$time දක්වා ඔබේ ඉඩ රඳවා ඇත';
  @override
  String get paySecurely => 'ආරක්ෂිතව ගෙවන්න';
  @override
  String get payWaiting => 'ඔබේ ගෙවීම ලැබෙන තුරු රැඳී සිටිමු…';
  @override
  String get payExpired => 'ඔබේ ගෙවීමට පෙර රඳවා ගැනීම අවසන් විය. නව වේලාවක් සඳහා නැවත වෙන්කරන්න.';
  @override
  String get payNotYet => 'ගෙවීම තවම ලැබී නැත.';
  @override
  String get payAgain => 'නැවත ගෙවන්න';
  @override
  String get payCardNote => 'ඔබ ගෙවන්නේ ගෙවීම් සේවාවේම පිටුවේ ය. Spotto ඔබේ කාඩ්පත් විස්තර කිසිදා නොදකී.';
  @override
  String get couldNotOpenPayment => 'ගෙවීම් පිටුව විවෘත කළ නොහැක';
  @override
  String get upcomingSection => 'ඉදිරියට';
  @override
  String get pastSection => 'පසුගිය';
  @override
  String get noBookingsYet => 'තවම වෙන්කිරීම් නැත';
  @override
  String get noBookingsHint => 'ඕනෑම ස්ථානයක පිටුවෙන් මේසයක්, සර්ෆ් පන්තියක්, කාමරයක් හෝ ප්‍රදේශවාසියෙකුගේ චාරිකාවක් වෙන්කරන්න.';
  @override
  String get completePayment => 'ගෙවීම සම්පූර්ණ කරන්න';
  @override
  String get cancelBookingAction => 'වෙන්කිරීම අවලංගු කරන්න';
  @override
  String get cancellingBooking => 'අවලංගු කරමින්…';
  @override
  String get cancelBookingTitle => 'මෙම වෙන්කිරීම අවලංගු කරන්නද?';
  @override
  String refundFullBack(String amount) => 'ඔබට $amount ආපසු ලැබේ.';
  @override
  String refundPartBack(String refund, String paid) => 'ඔබට $paid න් $refund ආපසු ලැබේ.';
  @override
  String get refundNone => 'ආපසු ගෙවීමේ කාලය ඉක්මවා ඇති බැවින් කිසිවක් ආපසු නොලැබේ.';
  @override
  String get keepBooking => 'තබා ගන්න';
  @override
  String get confirmCancelBooking => 'ඔව්, අවලංගු කරන්න';
  @override
  String cancelledWithRefund(String amount) => 'අවලංගු කළා. $amount ඔබට ආපසු එයි.';
  @override
  String get bookingCancelled => 'වෙන්කිරීම අවලංගු කළා.';
  @override
  String refundedAmount(String amount) => '$amount ආපසු ගෙවා ඇත';
  @override
  String paidAmount(String amount) => '$amount ගෙවා ඇත';
  @override
  String get statusAwaitingPayment => 'ගෙවීම බලාපොරොත්තුවෙන්';
  @override
  String get statusConfirmed => 'තහවුරුයි';
  @override
  String get statusCancelled => 'අවලංගුයි';
  @override
  String get statusCompleted => 'අවසන්';
  @override
  String get statusNoShow => 'නොපැමිණි';
  @override
  String get statusExpired => 'කල් ඉකුත් විය';
  @override
  String get experiencesTitle => 'අත්දැකීම් සහ උත්සව';
  @override
  String get experiencesSubtitle => 'ප්‍රදේශවාසීන් පවත්වන චාරිකා, පන්ති සහ රාත්‍රී සාද';
  @override
  String get experiencesEmpty => 'මෙහි තවම වෙන්කිරීමට කිසිවක් නැත. ඉක්මනින් නැවත බලන්න.';
  @override
  String get allCategoriesFilter => 'සියල්ල';
  @override
  String nextDateLabel(String date) => 'ඊළඟ: $date';
  @override
  String get localHostBadge => 'ප්‍රදේශවාසී සත්කාරක';
  @override
  String get experiencesEntrySubtitle => 'සර්ෆ්, තල්මසුන් නැරඹීම, ආහාර පිසීම, බාර් චාරිකා…';
  @override
  String get payAtCheckoutTitle => 'ඔබ ගෙවන්නේ වෙන්කරන විටයි';
  @override
  String get payAtCheckoutBody => 'ගෙවිය යුතු දෙයක් වෙන්කරන විට ඔබ ගෙවන්නේ ගෙවීම් සේවාවේ ආරක්ෂිත පිටුවේ ය. Spotto ඔබේ කාඩ්පත් ගබඩා නොකරයි.';
}
