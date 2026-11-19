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
  String get markAsUsedTitle => 'Mark as Used?';
  @override
  String get markAsUsedConfirm => 'Yes, Mark as Used';
  @override
  String get dismissOfferTitle => 'Dismiss Offer?';
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

  // ── Bookings, payments & experiences ─────────────────────────────────────
  @override
  String get bookHeading => 'Book';
  @override
  String get partySizeLabel => 'Party size';
  @override
  String get guestsLabel => 'Guests';
  @override
  String get quantityLabel => 'Quantity';
  @override
  String get checkInLabel => 'Check-in';
  @override
  String get checkOutLabel => 'Check-out';
  @override
  String get selectDateLabel => 'Select';
  @override
  String get noAvailabilityThisDay => 'No availability this day';
  @override
  String get couldNotLoadAvailability => 'Could not load availability';
  @override
  String get noUpcomingDates => 'No upcoming dates';
  @override
  String spotsLeft(int n) => '$n left';
  @override
  String get bookFree => 'Book for free';
  @override
  String bookAndPay(String amount) => 'Book & pay $amount';
  @override
  String get bookingInProgress => 'Booking…';
  @override
  String get priceFree => 'Free';
  @override
  String pricePerPerson(String amount) => '$amount per person';
  @override
  String pricePerBooking(String amount) => '$amount per booking';
  @override
  String pricePerNight(String amount) => '$amount per night';
  @override
  String get totalLabel => 'Total';
  @override
  String freeCancellationUntil(int hours) => 'Free cancellation up to ${hours}h before';
  @override
  String lateCancelRefund(int pct) => 'After that, $pct% back';
  @override
  String get lateCancelNoRefund => 'After that, no refund';
  @override
  String get meetingPointLabel => 'Meeting point';
  @override
  String get bookedShowAtCheckIn => 'Booked: show this at check-in';
  @override
  String partyOf(int n) => 'Party of $n';
  @override
  String get payTitle => 'Complete your payment';
  @override
  String payHeldUntil(String time) => 'Your spot is held until $time';
  @override
  String get paySecurely => 'Pay securely';
  @override
  String get payWaiting => 'Waiting for your payment to come through…';
  @override
  String get payExpired => 'The hold ran out before your payment arrived. Book again to pick a new time.';
  @override
  String get payNotYet => 'We haven\'t received the payment yet.';
  @override
  String get payAgain => 'Pay again';
  @override
  String get payCardNote => 'You pay on the payment provider\'s own page. Spotto never sees your card details.';
  @override
  String get couldNotOpenPayment => 'Couldn\'t open the payment page';
  @override
  String get upcomingSection => 'Upcoming';
  @override
  String get pastSection => 'Past';
  @override
  String get noBookingsYet => 'No bookings yet';
  @override
  String get noBookingsHint => 'Book a table, a surf lesson, a room or a local\'s tour from any place\'s page.';
  @override
  String get completePayment => 'Complete payment';
  @override
  String get cancelBookingAction => 'Cancel booking';
  @override
  String get cancellingBooking => 'Cancelling…';
  @override
  String get cancelBookingTitle => 'Cancel this booking?';
  @override
  String refundFullBack(String amount) => 'You\'ll get $amount back.';
  @override
  String refundPartBack(String refund, String paid) => 'You\'ll get $refund of $paid back.';
  @override
  String get refundNone => 'This booking is past its refund window, so nothing is refunded.';
  @override
  String get keepBooking => 'Keep it';
  @override
  String get confirmCancelBooking => 'Yes, cancel';
  @override
  String cancelledWithRefund(String amount) => 'Cancelled. $amount is on its way back to you.';
  @override
  String get bookingCancelled => 'Booking cancelled.';
  @override
  String refundedAmount(String amount) => 'Refunded $amount';
  @override
  String paidAmount(String amount) => 'Paid $amount';
  @override
  String get statusAwaitingPayment => 'Awaiting payment';
  @override
  String get statusConfirmed => 'Confirmed';
  @override
  String get statusCancelled => 'Cancelled';
  @override
  String get statusCompleted => 'Completed';
  @override
  String get statusNoShow => 'No-show';
  @override
  String get statusExpired => 'Expired';
  @override
  String get experiencesTitle => 'Experiences & events';
  @override
  String get experiencesSubtitle => 'Tours, classes and nights out run by locals';
  @override
  String get experiencesEmpty => 'Nothing to book here yet. Check back soon.';
  @override
  String get allCategoriesFilter => 'All';
  @override
  String nextDateLabel(String date) => 'Next: $date';
  @override
  String get localHostBadge => 'Local host';
  @override
  String get experiencesEntrySubtitle => 'Surf, whale watching, cooking, bar tours…';
  @override
  String get payAtCheckoutTitle => 'You pay at checkout';
  @override
  String get payAtCheckoutBody => 'When you book something paid, you pay on the payment provider\'s secure page. Spotto doesn\'t store your cards.';
}
