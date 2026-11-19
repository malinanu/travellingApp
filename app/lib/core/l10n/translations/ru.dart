import '../app_strings.dart';

class RuStrings implements AppStrings {
  const RuStrings();

  // ── Common ──────────────────────────────────────────────────────────────
  @override
  String get save => 'Сохранить';
  @override
  String get saved => 'Сохранено';
  @override
  String get cancel => 'Отмена';
  @override
  String get next => 'Далее';
  @override
  String get back => 'Назад';
  @override
  String get close => 'Закрыть';
  @override
  String get done => 'Готово';
  @override
  String get continueText => 'Продолжить';
  @override
  String get getStarted => 'Поехали!';
  @override
  String get dismiss => 'Закрыть';
  @override
  String get retry => 'Попробовать снова';
  @override
  String get search => 'Поиск';
  @override
  String get loading => 'Секундочку...';
  @override
  String get error => 'Произошла ошибка';
  @override
  String get somethingWentWrong => 'Что-то пошло не так';
  @override
  String get delete => 'Удалить';
  @override
  String get edit => 'Изменить';
  @override
  String get viewAll => 'Все';

  // ── Language Selection ──────────────────────────────────────────────────
  @override
  String get language => 'Язык';
  @override
  String get chooseLanguageTitle => 'Выберите язык';
  @override
  String get chooseLanguageSubtitle => 'На каком языке вам удобнее пользоваться Spotto?';
  @override
  String get changeLanguage => 'Сменить язык';

  // ── Navigation ──────────────────────────────────────────────────────────
  @override
  String get navDiscover => 'Главная';
  @override
  String get navSpots => 'Места';
  @override
  String get navStories => 'Истории';
  @override
  String get navTrip => 'Поездка';

  // ── Discover ────────────────────────────────────────────────────────────
  @override
  String get goodMorning => 'Доброе утро';
  @override
  String get goodAfternoon => 'Добрый день';
  @override
  String get goodEvening => 'Добрый вечер';
  @override
  String get searchPlaceholder => 'Кафе, пляжи, интересные места...';
  @override
  String get recommendedForYou => 'Специально для вас';
  @override
  String get currentVibe => 'Вайб места';
  @override
  String get dailyQuest => 'Квест дня';
  @override
  String get checkIn => 'Чек-ин';
  @override
  String get checkedIn => 'Вы на месте';

  // ── Categories & Spots ──────────────────────────────────────────────────
  @override
  String get categoryCafes => 'Кафе';
  @override
  String get categoryStays => 'Жилье';
  @override
  String get categorySights => 'Интересное';
  @override
  String get categorySurf => 'Серфинг';
  @override
  String get categoryCulture => 'Культура';
  @override
  String get categoryNightlife => 'Вечерняя жизнь';
  @override
  String get openNow => 'Открыто';
  @override
  String get closed => 'Закрыто';
  @override
  String distanceAway(String distance) => '$distance от вас';
  @override
  String get filters => 'Фильтры';
  @override
  String get allRegions => 'Все регионы';

  // ── Stories & Audio ─────────────────────────────────────────────────────
  @override
  String get audioGuides => 'Аудиогиды';
  @override
  String get listen => 'Слушать';
  @override
  String get playing => 'Играет';
  @override
  String get paused => 'Пауза';
  @override
  String get readStory => 'Читать историю';

  // ── Trip ────────────────────────────────────────────────────────────────
  @override
  String get myTrip => 'Моя поездка';
  @override
  String get planATrip => 'Спланировать маршрут';
  @override
  String dayNumber(int number) => 'День $number';
  @override
  String get addStop => 'Добавить остановку';
  @override
  String get directions => 'Маршрут';

  // ── Local Guide ─────────────────────────────────────────────────────────
  @override
  String get localGuide => 'Local Guide';
  @override
  String tunedFor(String style) => 'Под ваш стиль ($style)';
  @override
  String get askYourLocalFriend => 'Спросите у Local Guide...';
  @override
  String get giveMeMoreOptions => 'Покажи еще варианты';
  @override
  String get planATripWithThese => 'Составить маршрут из этих мест';
  @override
  String get clearChat => 'Очистить чат';
  @override
  String get chatCleared => 'История очищена';
  @override
  String get summarize => 'Итог';

  // ── Offers & Deals ──────────────────────────────────────────────────────
  @override
  String get activeOffer => 'Ваша скидка';
  @override
  String get claimOffer => 'Забрать скидку';
  @override
  String get claimed => 'Получено';
  @override
  String get showAtCounter => 'Покажите на кассе';
  @override
  String get hideQrCode => 'Скрыть QR-код';
  @override
  String get tapToShowQr => 'Показать QR-код';
  @override
  String get markAsUsed => 'Использовано';
  @override
  String get offerUsedPerk => 'Скидка использована! Приятного отдыха.';
  @override
  String get markAsUsedTitle => 'Отметить как использованное?';
  @override
  String get markAsUsedConfirm => 'Да, отметить';
  @override
  String get dismissOfferTitle => 'Скрыть предложение?';
  @override
  String get rateAgainIn15Minutes => 'Вы сможете снова оценить это место через 15 минут';

  // ── Profile & Settings ──────────────────────────────────────────────────
  @override
  String get profile => 'Профиль';
  @override
  String get travelStyle => 'Стиль путешествий';
  @override
  String get interests => 'Что вам нравится';
  @override
  String get notifications => 'Уведомления';
  @override
  String get account => 'Аккаунт';
  @override
  String get myBookings => 'Мои бронирования';
  @override
  String get myBookingsSubtitle => 'Предстоящие и прошлые поездки';
  @override
  String get passwordAndSecurity => 'Пароль и безопасность';
  @override
  String get passwordAndSecuritySubtitle => 'Смена пароля, 2FA';
  @override
  String get savedPayment => 'Способы оплаты';
  @override
  String get privacy => 'Конфиденциальность';
  @override
  String get privacySubtitle => 'Геолокация и доступ к данным';
  @override
  String get helpAndSupport => 'Помощь и поддержка';
  @override
  String get logOut => 'Выйти';

  // ── Auth & Onboarding ───────────────────────────────────────────────────
  @override
  String get welcomeTitle => 'Добро пожаловать в Spotto';
  @override
  String get welcomeSubtitle => 'Откройте Шри-Ланку как местный житель';
  @override
  String get signIn => 'Войти';
  @override
  String get createAccount => 'Создать аккаунт';
  @override
  String get emailAddress => 'Электронная почта';
  @override
  String get password => 'Пароль';
  @override
  String get fullName => 'Полное имя';
  @override
  String get continueWithEmail => 'Продолжить через почту';
  @override
  String get tellUsYourTravelStyle => 'Как вы любите путешествовать?';

  // ── Bookings, payments & experiences ─────────────────────────────────────
  @override
  String get bookHeading => 'Бронирование';
  @override
  String get partySizeLabel => 'Количество гостей';
  @override
  String get guestsLabel => 'Гости';
  @override
  String get quantityLabel => 'Количество';
  @override
  String get checkInLabel => 'Заезд';
  @override
  String get checkOutLabel => 'Выезд';
  @override
  String get selectDateLabel => 'Выбрать';
  @override
  String get noAvailabilityThisDay => 'На этот день мест нет';
  @override
  String get couldNotLoadAvailability => 'Не удалось загрузить расписание';
  @override
  String get noUpcomingDates => 'Ближайших дат нет';
  @override
  String spotsLeft(int n) => 'осталось $n';
  @override
  String get bookFree => 'Забронировать бесплатно';
  @override
  String bookAndPay(String amount) => 'Забронировать и оплатить $amount';
  @override
  String get bookingInProgress => 'Бронируем…';
  @override
  String get priceFree => 'Бесплатно';
  @override
  String pricePerPerson(String amount) => '$amount с человека';
  @override
  String pricePerBooking(String amount) => '$amount за бронирование';
  @override
  String pricePerNight(String amount) => '$amount за ночь';
  @override
  String get totalLabel => 'Итого';
  @override
  String freeCancellationUntil(int hours) => 'Бесплатная отмена за $hours ч до начала';
  @override
  String lateCancelRefund(int pct) => 'Позже вернём $pct%';
  @override
  String get lateCancelNoRefund => 'Позже деньги не возвращаются';
  @override
  String get meetingPointLabel => 'Место встречи';
  @override
  String get bookedShowAtCheckIn => 'Забронировано: покажите при входе';
  @override
  String partyOf(int n) => 'Гостей: $n';
  @override
  String get payTitle => 'Завершите оплату';
  @override
  String payHeldUntil(String time) => 'Место за вами до $time';
  @override
  String get paySecurely => 'Оплатить безопасно';
  @override
  String get payWaiting => 'Ждём подтверждения оплаты…';
  @override
  String get payExpired => 'Время брони истекло до оплаты. Забронируйте заново.';
  @override
  String get payNotYet => 'Оплата ещё не поступила.';
  @override
  String get payAgain => 'Оплатить снова';
  @override
  String get payCardNote => 'Оплата проходит на странице платёжного сервиса. Spotto не видит данные вашей карты.';
  @override
  String get couldNotOpenPayment => 'Не удалось открыть страницу оплаты';
  @override
  String get upcomingSection => 'Предстоящие';
  @override
  String get pastSection => 'Прошедшие';
  @override
  String get noBookingsYet => 'Пока нет бронирований';
  @override
  String get noBookingsHint => 'Бронируйте столик, урок сёрфинга, номер или тур с местными на странице места.';
  @override
  String get completePayment => 'Завершить оплату';
  @override
  String get cancelBookingAction => 'Отменить бронь';
  @override
  String get cancellingBooking => 'Отменяем…';
  @override
  String get cancelBookingTitle => 'Отменить это бронирование?';
  @override
  String refundFullBack(String amount) => 'Мы вернём $amount.';
  @override
  String refundPartBack(String refund, String paid) => 'Мы вернём $refund из $paid.';
  @override
  String get refundNone => 'Срок возврата прошёл, деньги не вернутся.';
  @override
  String get keepBooking => 'Оставить';
  @override
  String get confirmCancelBooking => 'Да, отменить';
  @override
  String cancelledWithRefund(String amount) => 'Отменено. $amount вернётся вам.';
  @override
  String get bookingCancelled => 'Бронирование отменено.';
  @override
  String refundedAmount(String amount) => 'Возвращено $amount';
  @override
  String paidAmount(String amount) => 'Оплачено $amount';
  @override
  String get statusAwaitingPayment => 'Ожидает оплаты';
  @override
  String get statusConfirmed => 'Подтверждено';
  @override
  String get statusCancelled => 'Отменено';
  @override
  String get statusCompleted => 'Завершено';
  @override
  String get statusNoShow => 'Неявка';
  @override
  String get statusExpired => 'Истекло';
  @override
  String get experiencesTitle => 'Впечатления и события';
  @override
  String get experiencesSubtitle => 'Туры, мастер-классы и вечеринки от местных';
  @override
  String get experiencesEmpty => 'Здесь пока нечего бронировать. Загляните позже.';
  @override
  String get allCategoriesFilter => 'Все';
  @override
  String nextDateLabel(String date) => 'Ближайшая: $date';
  @override
  String get localHostBadge => 'Местный гид';
  @override
  String get experiencesEntrySubtitle => 'Сёрфинг, киты, кулинария, барные туры…';
  @override
  String get payAtCheckoutTitle => 'Оплата при бронировании';
  @override
  String get payAtCheckoutBody => 'Платные бронирования оплачиваются на защищённой странице платёжного сервиса. Spotto не хранит карты.';
}
