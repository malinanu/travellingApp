import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:spotto/core/api/api_client.dart';
import 'package:spotto/core/storage/local_store.dart';
import 'package:spotto/core/utils/money.dart';
import 'package:spotto/data/models/models.dart';
import 'package:spotto/data/repositories/repositories.dart';
import 'package:spotto/features/traveller/bookings/booking_payment.dart';
import 'package:spotto/features/traveller/bookings/my_bookings_screen.dart';
import 'package:spotto/features/traveller/place_detail/business_drawer.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.setMockMethodCallHandler(
      const MethodChannel('plugins.it_nomads.com/flutter_secure_storage'),
      (call) async => null,
    );
  });

  final place = Place.fromJson({'id': 'plc_1', 'name': 'Harbour Surf', 'lat': 5.94, 'lng': 80.45});

  BookableService service({String pricingMode = 'free', int price = 0, String scheduling = 'weekly'}) =>
      BookableService.fromJson({
        'id': 'svc_1', 'vendorId': 'ven_1', 'type': 'session', 'name': 'Surf lesson', 'durationMins': 60,
        'capacity': 10, 'partySizeMin': 1, 'partySizeMax': 4, 'pricingMode': pricingMode, 'priceMinor': price,
        'freeCancelHours': 24, 'lateCancelRefundPct': 50, 'scheduling': scheduling,
      });

  final slotStart = DateTime.now().add(const Duration(days: 1, hours: 2)).toUtc();

  Map<String, dynamic> bookingJson({required String status, int amount = 0, String payment = 'not_required'}) => {
        'id': 'bkg_1', 'code': 'BOOK-AB12', 'vendorId': 'ven_1', 'serviceId': 'svc_1', 'type': 'session',
        'placeId': 'plc_1', 'startAt': slotStart.toIso8601String(),
        'endAt': slotStart.add(const Duration(hours: 1)).toIso8601String(), 'partySize': 2, 'status': status,
        'serviceName': 'Surf lesson', 'amountMinor': amount, 'currency': 'LKR', 'paymentStatus': payment,
        'holdExpiresAt': DateTime.now().add(const Duration(minutes: 15)).toUtc().toIso8601String(),
      };

  Future<(ProviderContainer, List<String>)> setUpContainer(
    BookableService svc,
    Map<String, dynamic> Function(RequestOptions) respond, {
    Future<bool> Function(Uri)? launcher,
  }) async {
    SharedPreferences.setMockInitialValues({});
    final store = await LocalStore.init();
    final api = ApiClient(store);
    final calls = <String>[];
    api.dio.interceptors.add(InterceptorsWrapper(onRequest: (options, handler) {
      calls.add('${options.method} ${options.path}');
      handler.resolve(Response(requestOptions: options, statusCode: 200, data: respond(options)));
    }));
    final container = ProviderContainer(overrides: [
      localStoreProvider.overrideWithValue(store),
      apiProvider.overrideWithValue(api),
      placeServicesProvider(place.id).overrideWith((ref) async => [svc]),
      serviceAvailabilityProvider.overrideWith((ref, args) async => {
            'slots': [
              {'start': slotStart.toIso8601String(), 'end': slotStart.add(const Duration(hours: 1)).toIso8601String(), 'remaining': 3},
            ],
          }),
      checkoutLauncherProvider.overrideWithValue(launcher ?? (uri) async => true),
      paymentPollIntervalProvider.overrideWithValue(const Duration(milliseconds: 100)),
    ]);
    return (container, calls);
  }

  Widget host(ProviderContainer c, Widget child) => UncontrolledProviderScope(
        container: c,
        child: MaterialApp(home: Scaffold(body: SingleChildScrollView(child: child))),
      );

  test('formatLkr groups thousands and shows cents only when there are any', () {
    expect(formatLkr(450000), 'LKR 4,500');
    expect(formatLkr(123456789), 'LKR 1,234,567.89');
    expect(formatLkr(5), 'LKR 0.05');
    expect(formatLkr(-30000), '-LKR 300');
  });

  test('BookableService.quote mirrors the server pricing modes', () {
    expect(service(pricingMode: 'per_person', price: 250000).quote(partySize: 3), 750000);
    expect(service(pricingMode: 'per_booking', price: 250000).quote(partySize: 3), 250000);
    expect(service(pricingMode: 'free').quote(partySize: 3), 0);
    final night = BookableService.fromJson({'pricingMode': 'per_night', 'priceMinor': 1000000});
    expect(night.quote(partySize: 2, nights: 3), 3000000);
  });

  test('My Bookings shows upcoming soonest first and past newest first', () {
    Booking b(String id, Duration from, String status) => Booking.fromJson({
          'id': id, 'startAt': DateTime.now().add(from).toUtc().toIso8601String(),
          'endAt': DateTime.now().add(from + const Duration(hours: 1)).toUtc().toIso8601String(),
          'status': status,
        });
    final split = splitBookings([
      b('later', const Duration(days: 5), 'confirmed'),
      b('old', const Duration(days: -10), 'completed'),
      b('soon', const Duration(days: 1), 'confirmed'),
      b('recent', const Duration(days: -1), 'completed'),
      b('cancelled_future', const Duration(days: 3), 'cancelled'),
    ]);
    expect([for (final x in split.upcoming) x.id], ['soon', 'later']);
    expect([for (final x in split.past) x.id], ['cancelled_future', 'recent', 'old']);
  });

  testWidgets('a free booking is confirmed straight away and shows its QR', (tester) async {
    final (container, calls) = await setUpContainer(service(), (o) => bookingJson(status: 'confirmed'));
    addTearDown(container.dispose);
    await tester.pumpWidget(host(container, BookingSection(place: place)));
    await tester.pumpAndSettle();

    expect(find.text('Book for free'), findsOneWidget);
    expect(find.byKey(const Key('booking_total')), findsNothing);
    await tester.tap(find.text(DateFormatForTest.hm(slotStart.toLocal())));
    await tester.pump();
    await tester.tap(find.byKey(const Key('booking_submit')));
    await tester.pumpAndSettle();

    expect(calls, contains('POST /bookings'));
    expect(find.text('BOOK-AB12'), findsOneWidget);
    expect(find.text('Booked: show this at check-in'), findsOneWidget);
    // The confirmation can be dismissed back to the form.
    await tester.tap(find.byKey(const Key('booking_qr_done')));
    await tester.pumpAndSettle();
    expect(find.text('Book for free'), findsOneWidget);
  });

  testWidgets('a paid booking shows its total, opens checkout and confirms after payment', (tester) async {
    var paid = false;
    Uri? opened;
    final (container, calls) = await setUpContainer(
      service(pricingMode: 'per_person', price: 250000),
      (o) {
        if (o.path == '/bookings') {
          return {
            ...bookingJson(status: 'awaiting_payment', amount: 500000, payment: 'pending'),
            'checkout': {'paymentId': 'pay_1', 'url': 'https://api.test/payments/checkout/pay_1', 'returnUrlPrefix': 'https://api.test/payments/return'},
          };
        }
        if (o.path == '/bookings/bkg_1/status') {
          return paid
              ? {'id': 'bkg_1', 'status': 'confirmed', 'paymentStatus': 'paid', 'code': 'BOOK-AB12'}
              : {'id': 'bkg_1', 'status': 'awaiting_payment', 'paymentStatus': 'pending', 'code': 'BOOK-AB12'};
        }
        return <String, dynamic>{};
      },
      launcher: (uri) async {
        opened = uri;
        return true;
      },
    );
    addTearDown(container.dispose);
    await tester.pumpWidget(host(container, BookingSection(place: place)));
    await tester.pumpAndSettle();

    expect(find.text('LKR 2,500 per person'), findsOneWidget);
    expect(find.text('LKR 5,000'), findsOneWidget); // default party of 2
    await tester.tap(find.byKey(const Key('booking_party_plus')));
    await tester.pump();
    expect(find.text('LKR 7,500'), findsOneWidget);
    await tester.tap(find.byIcon(Icons.remove_circle_outline));
    await tester.pump();
    expect(find.text('Book & pay LKR 5,000'), findsOneWidget);

    await tester.tap(find.text(DateFormatForTest.hm(slotStart.toLocal())));
    await tester.pump();
    await tester.tap(find.byKey(const Key('booking_submit')));
    await tester.pumpAndSettle();
    expect(find.text('Complete your payment'), findsOneWidget);

    await tester.tap(find.byKey(const Key('payment_pay_button')));
    await tester.pump();
    expect(opened.toString(), 'https://api.test/payments/checkout/pay_1');
    expect(find.text('Waiting for your payment to come through…'), findsOneWidget);
    expect(find.text('BOOK-AB12'), findsNothing, reason: 'no QR before the payment is confirmed');

    paid = true;
    await tester.pump(const Duration(milliseconds: 150));
    await tester.pump();
    expect(find.text('BOOK-AB12'), findsWidgets);
    expect(calls.where((c) => c == 'GET /bookings/bkg_1/status'), isNotEmpty);
    await tester.tap(find.byKey(const Key('payment_done_button')));
    await tester.pumpAndSettle();
  });

  test('formatMoney shows foreign currencies with two decimals', () {
    expect(formatMoney(1648, 'USD'), 'USD 16.48');
    expect(formatMoney(160000, 'USD'), 'USD 1,600.00');
    expect(formatMoney(-500, 'EUR'), '-EUR 5.00');
    expect(formatMoney(450000, 'LKR'), 'LKR 4,500');
  });

  Map<String, dynamic> paymentOptions(List<Map<String, dynamic>> options) => {'currency': 'LKR', 'options': options};
  final payHereOption = {'gateway': 'payhere', 'displayName': 'PayHere', 'chargeCurrency': 'LKR', 'chargeAmountMinor': 500000, 'converted': false};
  final payPalOption = {'gateway': 'paypal', 'displayName': 'PayPal', 'chargeCurrency': 'USD', 'chargeAmountMinor': 1717, 'converted': true};

  testWidgets('with several gateways the traveller picks one and it is sent with the booking', (tester) async {
    Map<String, dynamic>? sent;
    final (container, _) = await setUpContainer(
      service(pricingMode: 'per_person', price: 250000),
      (o) {
        if (o.path == '/payments/options') return paymentOptions([payHereOption, payPalOption]);
        if (o.path == '/bookings') {
          sent = (o.data as Map).cast<String, dynamic>();
          return {
            ...bookingJson(status: 'awaiting_payment', amount: 500000, payment: 'pending'),
            'checkout': {'paymentId': 'pay_1', 'gateway': 'paypal', 'url': 'https://api.test/payments/checkout/pay_1',
              'chargeAmountMinor': 1717, 'chargeCurrency': 'USD'},
          };
        }
        return {'id': 'bkg_1', 'status': 'awaiting_payment', 'paymentStatus': 'pending', 'code': 'BOOK-AB12'};
      },
    );
    addTearDown(container.dispose);
    await tester.pumpWidget(host(container, BookingSection(place: place)));
    await tester.pumpAndSettle();

    expect(find.text('Pay with'), findsOneWidget);
    expect(find.text('USD 17.17'), findsOneWidget);
    // PayHere, the first option, is the default and charges LKR.
    expect(find.text('Book & pay LKR 5,000'), findsOneWidget);
    expect(find.byKey(const Key('booking_fx_note')), findsNothing);

    await tester.tap(find.byKey(const Key('payment_option_paypal')));
    await tester.pump();
    expect(find.text('Book & pay USD 17.17'), findsOneWidget);
    expect(find.byKey(const Key('booking_fx_note')), findsOneWidget);

    await tester.tap(find.text(DateFormatForTest.hm(slotStart.toLocal())));
    await tester.pump();
    await tester.ensureVisible(find.byKey(const Key('booking_submit')));
    await tester.tap(find.byKey(const Key('booking_submit')));
    await tester.pumpAndSettle();
    expect(sent?['gateway'], 'paypal');
    expect(find.text("You'll be charged USD 17.17"), findsOneWidget);
  });

  testWidgets('with a single gateway there is no picker and no gateway is sent', (tester) async {
    Map<String, dynamic>? sent;
    final (container, _) = await setUpContainer(
      service(pricingMode: 'per_person', price: 250000),
      (o) {
        if (o.path == '/payments/options') return paymentOptions([payHereOption]);
        if (o.path == '/bookings') {
          sent = (o.data as Map).cast<String, dynamic>();
          return {
            ...bookingJson(status: 'awaiting_payment', amount: 500000, payment: 'pending'),
            'checkout': {'paymentId': 'pay_1', 'url': 'https://api.test/payments/checkout/pay_1'},
          };
        }
        return {'id': 'bkg_1', 'status': 'awaiting_payment', 'paymentStatus': 'pending', 'code': 'BOOK-AB12'};
      },
    );
    addTearDown(container.dispose);
    await tester.pumpWidget(host(container, BookingSection(place: place)));
    await tester.pumpAndSettle();

    expect(find.text('Pay with'), findsNothing);
    await tester.tap(find.text(DateFormatForTest.hm(slotStart.toLocal())));
    await tester.pump();
    await tester.tap(find.byKey(const Key('booking_submit')));
    await tester.pumpAndSettle();
    expect(sent, isNotNull);
    expect(sent!.containsKey('gateway'), isFalse);
    expect(find.byKey(const Key('payment_charge_note')), findsNothing);
  });

  testWidgets('cancelling a paid booking shows the refund first', (tester) async {
    SharedPreferences.setMockInitialValues({
      StoreKeys.bookings: jsonEncode([bookingJson(status: 'confirmed', amount: 500000, payment: 'paid')]),
    });
    final store = await LocalStore.init();
    final api = ApiClient(store);
    api.dio.interceptors.add(InterceptorsWrapper(onRequest: (options, handler) {
      final data = switch (options.path) {
        '/bookings/bkg_1/cancel-preview' => {'cancellable': true, 'refundMinor': 250000, 'paidMinor': 500000, 'fullRefund': false},
        '/bookings/bkg_1/cancel' => {'status': 'cancelled', 'refundMinor': 250000},
        _ => <String, Object>{},
      };
      handler.resolve(Response(requestOptions: options, statusCode: 200, data: data));
    }));
    final container = ProviderContainer(overrides: [
      localStoreProvider.overrideWithValue(store),
      apiProvider.overrideWithValue(api),
    ]);
    addTearDown(container.dispose);
    await tester.pumpWidget(UncontrolledProviderScope(container: container, child: const MaterialApp(home: MyBookingsScreen())));
    await tester.pumpAndSettle();

    expect(find.text('UPCOMING'), findsOneWidget);
    await tester.tap(find.byKey(const Key('cancel_booking_bkg_1')));
    await tester.pumpAndSettle();
    expect(find.text("You'll get LKR 2,500 of LKR 5,000 back."), findsOneWidget);
    await tester.tap(find.byKey(const Key('confirm_cancel_booking')));
    await tester.pumpAndSettle();
    expect(find.text('Cancelled. LKR 2,500 is on its way back to you.'), findsOneWidget);
    final b = container.read(bookingsProvider).single;
    expect(b.status, 'cancelled');
    expect(b.paymentStatus, 'partially_refunded');
  });
}

/// The slot picker shows 24h "HH:mm" times.
class DateFormatForTest {
  static String hm(DateTime d) => '${d.hour.toString().padLeft(2, '0')}:${d.minute.toString().padLeft(2, '0')}';
}
