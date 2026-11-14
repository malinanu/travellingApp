import 'package:flutter_test/flutter_test.dart';
import 'package:spotto/core/l10n/app_language.dart';
import 'package:spotto/core/l10n/app_strings.dart';
import 'package:spotto/core/l10n/translations/translations.dart';

void main() {
  group('AppLanguage Tests', () {
    test('fromCode resolves valid codes correctly', () {
      expect(AppLanguage.fromCode('en'), equals(AppLanguage.en));
      expect(AppLanguage.fromCode('nl'), equals(AppLanguage.nl));
      expect(AppLanguage.fromCode('de'), equals(AppLanguage.de));
      expect(AppLanguage.fromCode('fr'), equals(AppLanguage.fr));
      expect(AppLanguage.fromCode('ru'), equals(AppLanguage.ru));
      expect(AppLanguage.fromCode('si'), equals(AppLanguage.si));
    });

    test('fromCode handles regional tags (e.g. de-DE, nl_NL, en-US)', () {
      expect(AppLanguage.fromCode('de-DE'), equals(AppLanguage.de));
      expect(AppLanguage.fromCode('nl_NL'), equals(AppLanguage.nl));
      expect(AppLanguage.fromCode('en-US'), equals(AppLanguage.en));
      expect(AppLanguage.fromCode('fr_FR'), equals(AppLanguage.fr));
      expect(AppLanguage.fromCode('ru-RU'), equals(AppLanguage.ru));
      expect(AppLanguage.fromCode('si-LK'), equals(AppLanguage.si));
    });

    test('fromCode handles uppercase and mixed case tags', () {
      expect(AppLanguage.fromCode('EN'), equals(AppLanguage.en));
      expect(AppLanguage.fromCode('NL'), equals(AppLanguage.nl));
      expect(AppLanguage.fromCode('DE-de'), equals(AppLanguage.de));
      expect(AppLanguage.fromCode('Fr_FR'), equals(AppLanguage.fr));
      expect(AppLanguage.fromCode('RU'), equals(AppLanguage.ru));
      expect(AppLanguage.fromCode('SI-lk'), equals(AppLanguage.si));
    });

    test('fromCode defaults to English for unknown, empty, or null codes', () {
      expect(AppLanguage.fromCode(null), equals(AppLanguage.en));
      expect(AppLanguage.fromCode(''), equals(AppLanguage.en));
      expect(AppLanguage.fromCode('  '), equals(AppLanguage.en));
      expect(AppLanguage.fromCode('xx'), equals(AppLanguage.en));
      expect(AppLanguage.fromCode('unknown'), equals(AppLanguage.en));
    });

    test('Every language has native name, English name, and flag', () {
      for (final lang in AppLanguage.values) {
        expect(lang.code.isNotEmpty, isTrue);
        expect(lang.nativeName.isNotEmpty, isTrue);
        expect(lang.englishName.isNotEmpty, isTrue);
        expect(lang.flag.isNotEmpty, isTrue);
      }
    });
  });

  group('Translations Completeness & Quality Tests', () {
    test('All 6 languages are registered in translations map', () {
      expect(translations.containsKey(AppLanguage.en), isTrue);
      expect(translations.containsKey(AppLanguage.nl), isTrue);
      expect(translations.containsKey(AppLanguage.de), isTrue);
      expect(translations.containsKey(AppLanguage.fr), isTrue);
      expect(translations.containsKey(AppLanguage.ru), isTrue);
      expect(translations.containsKey(AppLanguage.si), isTrue);
    });

    for (final lang in AppLanguage.values) {
      test('Language ${lang.name} (${lang.nativeName}) has no empty or placeholder strings', () {
        final AppStrings strings = lookupStrings(lang);

        void verifyString(String val) {
          expect(val.trim().isNotEmpty, isTrue);
          expect(val.contains('TODO'), isFalse);
          expect(val.contains('MISSING'), isFalse);
          expect(val.contains('???'), isFalse);
        }

        // Common
        verifyString(strings.save);
        verifyString(strings.saved);
        verifyString(strings.cancel);
        verifyString(strings.next);
        verifyString(strings.back);
        verifyString(strings.close);
        verifyString(strings.done);
        verifyString(strings.continueText);
        verifyString(strings.getStarted);
        verifyString(strings.dismiss);
        verifyString(strings.retry);
        verifyString(strings.search);
        verifyString(strings.loading);
        verifyString(strings.error);
        verifyString(strings.somethingWentWrong);
        verifyString(strings.delete);
        verifyString(strings.edit);
        verifyString(strings.viewAll);

        // Language Picker
        verifyString(strings.language);
        verifyString(strings.chooseLanguageTitle);
        verifyString(strings.chooseLanguageSubtitle);
        verifyString(strings.changeLanguage);

        // Navigation
        verifyString(strings.navDiscover);
        verifyString(strings.navSpots);
        verifyString(strings.navStories);
        verifyString(strings.navTrip);

        // Discover
        verifyString(strings.goodMorning);
        verifyString(strings.goodAfternoon);
        verifyString(strings.goodEvening);
        verifyString(strings.searchPlaceholder);
        verifyString(strings.recommendedForYou);
        verifyString(strings.currentVibe);
        verifyString(strings.dailyQuest);
        verifyString(strings.checkIn);
        verifyString(strings.checkedIn);

        // Categories & Spots
        verifyString(strings.categoryCafes);
        verifyString(strings.categoryStays);
        verifyString(strings.categorySights);
        verifyString(strings.categorySurf);
        verifyString(strings.categoryCulture);
        verifyString(strings.categoryNightlife);
        verifyString(strings.openNow);
        verifyString(strings.closed);
        final distanceStr = strings.distanceAway('2.5 km');
        verifyString(distanceStr);
        expect(distanceStr.contains('2.5 km'), isTrue);
        verifyString(strings.filters);
        verifyString(strings.allRegions);

        // Stories
        verifyString(strings.audioGuides);
        verifyString(strings.listen);
        verifyString(strings.playing);
        verifyString(strings.paused);
        verifyString(strings.readStory);

        // Trip
        verifyString(strings.myTrip);
        verifyString(strings.planATrip);
        final dayStr = strings.dayNumber(3);
        verifyString(dayStr);
        expect(dayStr.contains('3'), isTrue);
        verifyString(strings.addStop);
        verifyString(strings.directions);

        // Local Guide
        verifyString(strings.localGuide);
        final tunedStr = strings.tunedFor('Luxury');
        verifyString(tunedStr);
        expect(tunedStr.contains('Luxury'), isTrue);
        verifyString(strings.askYourLocalFriend);
        verifyString(strings.giveMeMoreOptions);
        verifyString(strings.planATripWithThese);
        verifyString(strings.clearChat);
        verifyString(strings.chatCleared);
        verifyString(strings.summarize);

        // Offers & Deals
        verifyString(strings.activeOffer);
        verifyString(strings.claimOffer);
        verifyString(strings.claimed);
        verifyString(strings.showAtCounter);
        verifyString(strings.hideQrCode);
        verifyString(strings.tapToShowQr);
        verifyString(strings.markAsUsed);
        verifyString(strings.offerUsedPerk);
        verifyString(strings.markAsUsedTitle);
        verifyString(strings.markAsUsedConfirm);
        verifyString(strings.dismissOfferTitle);
        verifyString(strings.rateAgainIn15Minutes);

        // Profile & Settings
        verifyString(strings.profile);
        verifyString(strings.travelStyle);
        verifyString(strings.interests);
        verifyString(strings.notifications);
        verifyString(strings.account);
        verifyString(strings.myBookings);
        verifyString(strings.myBookingsSubtitle);
        verifyString(strings.passwordAndSecurity);
        verifyString(strings.passwordAndSecuritySubtitle);
        verifyString(strings.savedPayment);
        verifyString(strings.privacy);
        verifyString(strings.privacySubtitle);
        verifyString(strings.helpAndSupport);
        verifyString(strings.logOut);

        // Auth & Onboarding
        verifyString(strings.welcomeTitle);
        verifyString(strings.welcomeSubtitle);
        verifyString(strings.signIn);
        verifyString(strings.createAccount);
        verifyString(strings.emailAddress);
        verifyString(strings.password);
        verifyString(strings.fullName);
        verifyString(strings.continueWithEmail);
        verifyString(strings.tellUsYourTravelStyle);
      });
    }
  });

  group('Meaningful Vocabulary & Natural Phrasing Tests', () {
    test('Sinhala uses natural modern conversational phrasing without big words', () {
      final si = lookupStrings(AppLanguage.si);
      expect(si.navDiscover, equals('ගවේෂණය'));
      expect(si.navSpots, equals('ස්ථාන'));
      expect(si.navStories, equals('කතා'));
      expect(si.navTrip, equals('චාරිකාව'));
      expect(si.chooseLanguageTitle, equals('ඔබේ භාෂාව තෝරන්න'));
      expect(si.language, equals('භාෂාව'));

      // Meaningful, everyday vocabulary checks (avoiding bureaucratic jargon)
      expect(si.claimOffer, equals('දීමනාව ගන්න'));
      expect(si.activeOffer, equals('විශේෂ දීමනාව'));
      expect(si.markAsUsed, equals('භාවිතා කළා'));
      expect(si.currentVibe, equals('දැන් තියෙන වයිබ් එක'));
      expect(si.dailyQuest, equals('අද මෙහෙයුම'));
      expect(si.interests, equals('කැමති දේවල්'));
      expect(si.checkIn, equals('චෙක්-ඉන් වෙන්න'));
      expect(si.loading, equals('මඳක් රැඳෙන්න...'));
      expect(si.rateAgainIn15Minutes, contains('විනාඩි 15කින්'));
      expect(si.offerUsedPerk, contains('භුක්ති විඳින්න'));
    });

    test('French uses modern app conventions (avoiding literal "réclamer")', () {
      final fr = lookupStrings(AppLanguage.fr);
      expect(fr.navDiscover, equals('Découvrir'));
      expect(fr.navSpots, equals('Lieux'));
      expect(fr.navStories, equals('Histoires'));
      expect(fr.navTrip, equals('Voyage'));
      expect(fr.chooseLanguageTitle, equals('Choisissez votre langue'));
      expect(fr.language, equals('Langue'));

      // Natural phrasing
      expect(fr.claimOffer, equals('Profiter de l\'offre'));
      expect(fr.activeOffer, equals('Bon plan disponible'));
      expect(fr.markAsUsed, equals('Déjà utilisé'));
      expect(fr.currentVibe, equals('Ambiance du lieu'));
      expect(fr.askYourLocalFriend, equals('Posez une question à votre guide...'));
      expect(fr.rateAgainIn15Minutes, contains('15 minutes'));
    });

    test('German uses clear mobile actions (sichern vs eingelöst)', () {
      final de = lookupStrings(AppLanguage.de);
      expect(de.navDiscover, equals('Entdecken'));
      expect(de.navSpots, equals('Orte'));
      expect(de.navStories, equals('Geschichten'));
      expect(de.navTrip, equals('Reise'));
      expect(de.chooseLanguageTitle, equals('Wähle deine Sprache'));
      expect(de.language, equals('Sprache'));

      // Natural phrasing
      expect(de.claimOffer, equals('Angebot sichern'));
      expect(de.activeOffer, equals('Dein Angebot'));
      expect(de.markAsUsed, equals('Eingelöst'));
      expect(de.dailyQuest, equals('Tages-Challenge'));
      expect(de.askYourLocalFriend, equals('Frag deinen Local Guide...'));
    });

    test('Dutch uses punchy, authentic traveler phrasing', () {
      final nl = lookupStrings(AppLanguage.nl);
      expect(nl.navDiscover, equals('Ontdek'));
      expect(nl.navSpots, equals('Plekken'));
      expect(nl.navStories, equals('Verhalen'));
      expect(nl.navTrip, equals('Reis'));
      expect(nl.chooseLanguageTitle, equals('Kies je taal'));
      expect(nl.language, equals('Taal'));

      // Natural phrasing
      expect(nl.claimOffer, equals('Pak deze deal'));
      expect(nl.activeOffer, equals('Jouw deal'));
      expect(nl.markAsUsed, equals('Gebruikt'));
      expect(nl.askYourLocalFriend, equals('Vraag het je Local Guide...'));
    });

    test('Russian uses conversational mobile travel copy', () {
      final ru = lookupStrings(AppLanguage.ru);
      expect(ru.navDiscover, equals('Главная'));
      expect(ru.navSpots, equals('Места'));
      expect(ru.navStories, equals('Истории'));
      expect(ru.navTrip, equals('Поездка'));
      expect(ru.chooseLanguageTitle, equals('Выберите язык'));
      expect(ru.language, equals('Язык'));

      // Natural phrasing
      expect(ru.claimOffer, equals('Забрать скидку'));
      expect(ru.activeOffer, equals('Ваша скидка'));
      expect(ru.markAsUsed, equals('Использовано'));
      expect(ru.currentVibe, equals('Вайб места'));
      expect(ru.askYourLocalFriend, equals('Спросите у Local Guide...'));
    });
  });
}
