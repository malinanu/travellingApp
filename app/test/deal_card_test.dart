import 'dart:convert';
import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:spotto/core/api/api_client.dart';
import 'package:spotto/core/storage/local_store.dart';
import 'package:spotto/data/models/models.dart';
import 'package:spotto/data/repositories/repositories.dart';
import 'package:spotto/features/traveller/place_detail/business_drawer.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  setUp(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.setMockMethodCallHandler(
      const MethodChannel('plugins.it_nomads.com/flutter_secure_storage'),
      (call) async => null,
    );
  });

  Place makePlace({String id = 'plc_deal_1', bool activeDeal = true}) {
    return Place.fromJson({
      'id': id,
      'name': 'Cafe Mirissa',
      'category': 'cafe',
      'lat': 5.94,
      'lng': 80.45,
      'deal': {
        'id': 'off_1',
        'active': activeDeal,
        'label': '20% off all drinks',
        'tag': '20% OFF',
      },
    });
  }

  test('ClaimsNotifier.removeClaim removes claim from state and storage', () async {
    SharedPreferences.setMockInitialValues({
      StoreKeys.claims: jsonEncode({'plc_deal_1': 'SPOTTO-ABCD'}),
    });
    final store = await LocalStore.init();
    final container = ProviderContainer(
      overrides: [
        localStoreProvider.overrideWithValue(store),
      ],
    );
    addTearDown(container.dispose);

    expect(container.read(claimsProvider)['plc_deal_1'], 'SPOTTO-ABCD');

    container.read(claimsProvider.notifier).removeClaim('plc_deal_1');
    expect(container.read(claimsProvider)['plc_deal_1'], isNull);

    final saved = store.getJson(StoreKeys.claims);
    expect(saved?['plc_deal_1'], isNull);
  });

  test('RedeemedDealsNotifier tracks and persists redeemed places', () async {
    SharedPreferences.setMockInitialValues({});
    final store = await LocalStore.init();
    final container = ProviderContainer(
      overrides: [
        localStoreProvider.overrideWithValue(store),
      ],
    );
    addTearDown(container.dispose);

    expect(container.read(redeemedDealsProvider).contains('plc_deal_1'), isFalse);

    container.read(redeemedDealsProvider.notifier).markRedeemed('plc_deal_1');
    expect(container.read(redeemedDealsProvider).contains('plc_deal_1'), isTrue);

    final savedList = store.getJsonList(StoreKeys.redeemedDeals);
    expect(savedList?.contains('plc_deal_1'), isTrue);
  });

  testWidgets('DealCard expands, collapses, and toggles open/close state', (tester) async {
    SharedPreferences.setMockInitialValues({});
    final store = await LocalStore.init();
    final place = makePlace();

    final container = ProviderContainer(
      overrides: [
        localStoreProvider.overrideWithValue(store),
      ],
    );
    addTearDown(container.dispose);

    // 1. Unclaimed state: shows Claim button
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          home: Scaffold(
            body: DealCard(place: place, claimedCode: null),
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.byKey(const Key('deal_card_claim_button')), findsOneWidget);

    // 2. Claimed state: starts expanded (open)
    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          home: Scaffold(
            body: DealCard(place: place, claimedCode: 'SPOTTO-VG6F'),
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.text('SPOTTO-VG6F'), findsOneWidget);
    expect(find.text('Hide QR'), findsOneWidget);
    expect(find.text('Show at counter'), findsOneWidget);
    expect(find.byKey(const Key('deal_card_collapse_button')), findsOneWidget);
    expect(find.byKey(const Key('deal_card_mark_used_button')), findsOneWidget);

    // 3. Tap "Hide QR" to collapse (close)
    await tester.tap(find.byKey(const Key('deal_card_collapse_button')));
    await tester.pump(const Duration(milliseconds: 300));

    expect(find.text('Tap to show QR code'), findsOneWidget);
    expect(find.text('Hide QR'), findsNothing);
    expect(find.text('Show at counter'), findsNothing);
    expect(find.byKey(const Key('deal_card_expand_button')), findsOneWidget);

    // 4. Tap to expand again (open)
    await tester.tap(find.byKey(const Key('deal_card_expand_button')));
    await tester.pump(const Duration(milliseconds: 300));

    expect(find.text('Hide QR'), findsOneWidget);
    expect(find.text('Show at counter'), findsOneWidget);
  });

  testWidgets('Mark as used immediately removes offer card from app', (tester) async {
    SharedPreferences.setMockInitialValues({});
    final store = await LocalStore.init();
    final place = makePlace();

    final container = ProviderContainer(
      overrides: [
        localStoreProvider.overrideWithValue(store),
      ],
    );
    addTearDown(container.dispose);

    container.read(claimsProvider.notifier).state = {'plc_deal_1': 'SPOTTO-VG6F'};

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          home: Scaffold(
            body: DealCard(place: place, claimedCode: 'SPOTTO-VG6F'),
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.text('SPOTTO-VG6F'), findsOneWidget);

    // Tap "Mark as used"
    await tester.tap(find.byKey(const Key('deal_card_mark_used_button')));
    await tester.pumpAndSettle();

    // Confirmation dialog appears
    expect(find.text('Mark as Used?'), findsOneWidget);

    // Confirm dialog
    await tester.tap(find.text('Yes, Mark as Used'));
    await tester.pumpAndSettle();

    // Verify card is now SizedBox.shrink() / removed
    expect(find.text('SPOTTO-VG6F'), findsNothing);
    expect(find.byKey(const Key('deal_card_claim_button')), findsNothing);
    expect(container.read(claimsProvider)['plc_deal_1'], isNull);
    expect(container.read(redeemedDealsProvider).contains('plc_deal_1'), isTrue);

    // Celebration toast shown
    expect(find.text('🎉 Offer used! Enjoy your discount.'), findsOneWidget);
  });

  testWidgets('Auto-disappearance when polling detects redeemed status', (tester) async {
    SharedPreferences.setMockInitialValues({});
    final store = await LocalStore.init();
    final place = makePlace();

    // Setup ApiClient that returns status: "redeemed"
    final api = ApiClient(store);
    api.dio.interceptors.add(InterceptorsWrapper(
      onRequest: (options, handler) {
        if (options.path.contains('/status')) {
          handler.resolve(Response(
            requestOptions: options,
            statusCode: 200,
            data: {'code': 'SPOTTO-VG6F', 'status': 'redeemed', 'redeemedAt': '2026-11-04T12:00:00Z'},
          ));
        } else {
          handler.next(options);
        }
      },
    ));

    final container = ProviderContainer(
      overrides: [
        localStoreProvider.overrideWithValue(store),
        apiProvider.overrideWithValue(api),
      ],
    );
    addTearDown(container.dispose);

    container.read(claimsProvider.notifier).state = {'plc_deal_1': 'SPOTTO-VG6F'};

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          home: Scaffold(
            body: DealCard(place: place, claimedCode: 'SPOTTO-VG6F'),
          ),
        ),
      ),
    );
    await tester.pump();

    expect(find.text('SPOTTO-VG6F'), findsOneWidget);

    // Advance timer by 2 seconds to trigger polling check
    await tester.pump(const Duration(seconds: 2));
    await tester.pumpAndSettle();

    // Polling detected redemption! Card automatically disappeared
    expect(find.text('SPOTTO-VG6F'), findsNothing);
    expect(container.read(claimsProvider)['plc_deal_1'], isNull);
    expect(container.read(redeemedDealsProvider).contains('plc_deal_1'), isTrue);
    expect(find.text('🎉 Offer used! Enjoy your discount.'), findsOneWidget);
  });

  testWidgets('Current Vibe row does not overflow on a narrow 300px screen', (tester) async {
    SharedPreferences.setMockInitialValues({});
    final store = await LocalStore.init();
    final place = makePlace();

    final container = ProviderContainer(
      overrides: [
        localStoreProvider.overrideWithValue(store),
        placeDetailsProvider(place.id).overrideWith((ref) async => place),
        devicePositionProvider.overrideWith((ref) async => ((lat: 5.94, lng: 80.45), DeviceLocationResult.granted)),
      ],
    );
    addTearDown(container.dispose);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          home: Scaffold(
            body: Center(
              child: SizedBox(
                width: 300,
                child: VibeCheck(place: place),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.pump();

    // Verify CURRENT VIBE renders and zero exceptions/overflows were thrown
    expect(find.text('CURRENT VIBE'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
