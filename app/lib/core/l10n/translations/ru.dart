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
}
