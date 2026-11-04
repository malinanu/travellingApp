import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:spotto/core/api/api_client.dart';
import 'package:flutter/services.dart';
import 'package:spotto/core/storage/local_store.dart';
import 'package:spotto/data/repositories/nearby_alerts_repository.dart';
import 'package:spotto/data/repositories/repositories.dart';
import 'package:spotto/features/traveller/onboarding/onboarding_screen.dart';
import 'package:spotto/features/traveller/profile/profile_subpages.dart';

class _FakeMeNotifier extends MeNotifier {
  final Map<String, dynamic> data;
  _FakeMeNotifier(this.data);
  @override
  Map<String, dynamic>? build() => data;
}

class _FakeNearbyAlertsNotifier extends NearbyAlertsNotifier {
  @override
  NearbyAlertsState build() => const NearbyAlertsState(enabled: false);
}

class _CustomAdapter implements HttpClientAdapter {
  final String Function(RequestOptions) handler;
  _CustomAdapter(this.handler);

  @override
  Future<ResponseBody> fetch(RequestOptions options, Stream<dynamic>? requestStream, Future<void>? cancelFuture) async {
    return ResponseBody.fromString(
      handler(options),
      200,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  final secureStorageMap = <String, String>{};
  TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger.setMockMethodCallHandler(
    const MethodChannel('plugins.it_nomads.com/flutter_secure_storage'),
    (call) async {
      if (call.method == 'write') {
        final key = call.arguments['key'] as String?;
        final val = call.arguments['value'] as String?;
        if (key != null && val != null) {
          secureStorageMap[key] = val;
        }
        return null;
      }
      if (call.method == 'read') {
        final key = call.arguments['key'] as String?;
        return key != null ? secureStorageMap[key] : null;
      }
      if (call.method == 'delete') {
        final key = call.arguments['key'] as String?;
        if (key != null) secureStorageMap.remove(key);
        return null;
      }
      return null;
    },
  );

  test('AuthNotifier handles twoFactorRequired without TypeError', () async {
    SharedPreferences.setMockInitialValues({});
    final store = await LocalStore.init();

    var loginVerifyCalled = false;

    final client = ApiClient(store);
    client.dio.httpClientAdapter = _CustomAdapter((opts) {
      if (opts.path == '/auth/login') {
        return '{"twoFactorRequired":true,"twoFactorToken":"pending-2fa-token"}';
      }
      if (opts.path == '/auth/2fa/login-verify') {
        loginVerifyCalled = true;
        return '{"token":"jwt-abc","user":{"id":"usr_1","email":"test@example.com"}}';
      }
      return '{}';
    });

    final container = ProviderContainer(
      overrides: [
        localStoreProvider.overrideWithValue(store),
        apiProvider.overrideWithValue(client),
      ],
    );

    final auth = container.read(authProvider.notifier);
    final err = await auth.login('test@example.com', 'password123');
    expect(err, isNull);
    expect(auth.pendingTwoFactorToken, 'pending-2fa-token');
    expect(container.read(authProvider), isFalse);

    // Now verify TOTP code
    final verifyErr = await auth.verifyTwoFactorLogin('123456');
    expect(verifyErr, isNull);
    expect(loginVerifyCalled, isTrue);
    expect(auth.pendingTwoFactorToken, isNull);
    expect(container.read(authProvider), isTrue);
  });

  testWidgets('OnboardingScreen Step 3 displays all 6 privacy toggles', (tester) async {
    SharedPreferences.setMockInitialValues({});
    final store = await LocalStore.init();
    final client = ApiClient(store);

    tester.view.physicalSize = const Size(900, 2400);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          localStoreProvider.overrideWithValue(store),
          apiProvider.overrideWithValue(client),
          nearbyAlertsProvider.overrideWith(() => _FakeNearbyAlertsNotifier()),
        ],
        child: const MaterialApp(home: OnboardingScreen()),
      ),
    );
    await tester.pump();

    // Welcome -> Step 1
    await tester.tap(find.text('Let’s go'));
    await tester.pump(const Duration(milliseconds: 100));

    // Step 1: Pick a style
    await tester.tap(find.text('Digital Nomad'));
    await tester.pump(const Duration(milliseconds: 100));
    await tester.tap(find.text('Next'));
    await tester.pump(const Duration(milliseconds: 100));

    // Step 2: Pick an interest
    await tester.tap(find.text('Beaches'));
    await tester.pump(const Duration(milliseconds: 100));
    await tester.tap(find.text('Next'));
    await tester.pump(const Duration(milliseconds: 100));

    // Step 3: Privacy & Permissions
    expect(find.text('Privacy & Permissions'), findsOneWidget);
    expect(find.text('Share live location'), findsOneWidget);
    expect(find.text('Nearby saved-place alerts'), findsOneWidget);
    expect(find.text('Visible to group'), findsOneWidget);
    expect(find.text('Count my vibe votes'), findsOneWidget);
    expect(find.text('Usage analytics'), findsOneWidget);
    expect(find.text('Personalised offers'), findsOneWidget);
    expect(find.text('Start exploring'), findsOneWidget);
  });

  testWidgets('PrivacyPage displays toggles and Download my data button', (tester) async {
    SharedPreferences.setMockInitialValues({});
    final store = await LocalStore.init();
    final client = ApiClient(store);

    tester.view.physicalSize = const Size(900, 2400);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          localStoreProvider.overrideWithValue(store),
          apiProvider.overrideWithValue(client),
          nearbyAlertsProvider.overrideWith(() => _FakeNearbyAlertsNotifier()),
          meProvider.overrideWith(() => _FakeMeNotifier({
                'shareLocation': true,
                'visibleToGroup': true,
                'countVibeVotes': true,
                'usageAnalytics': true,
                'personalisedOffers': false,
              })),
        ],
        child: MaterialApp(home: PrivacyPage(onBack: () {})),
      ),
    );
    await tester.pump();

    expect(find.text('Share live location'), findsOneWidget);
    expect(find.text('Nearby saved-place alerts'), findsOneWidget);
    expect(find.text('Visible to group'), findsOneWidget);
    expect(find.text('Count my vibe votes'), findsOneWidget);
    expect(find.text('Usage analytics'), findsOneWidget);
    expect(find.text('Personalised offers'), findsOneWidget);
    expect(find.text('Download my data'), findsOneWidget);
  });

  testWidgets('SecurityPage displays live sessions with THIS DEVICE · Active now and allows revocation', (tester) async {
    SharedPreferences.setMockInitialValues({});
    final store = await LocalStore.init();
    final client = ApiClient(store);

    var revokedSessionId = '';

    client.dio.httpClientAdapter = _CustomAdapter((opts) {
      if (opts.path == '/me/sessions') {
        if (opts.method == 'GET') {
          return '[{"id":"ses_current","client":"app","deviceLabel":"Pixel 9 Pro","isCurrent":true,"createdAt":"2026-11-04T08:00:00Z"},'
              '{"id":"ses_other","client":"vendor","deviceLabel":"MacBook Air","isCurrent":false,"lastUsedAt":"2026-11-03T10:00:00Z"}]';
        }
      }
      if (opts.path == '/me/sessions/ses_other' && opts.method == 'DELETE') {
        revokedSessionId = 'ses_other';
        return '{"status":"ok"}';
      }
      return '{}';
    });

    tester.view.physicalSize = const Size(900, 2400);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          localStoreProvider.overrideWithValue(store),
          apiProvider.overrideWithValue(client),
          meProvider.overrideWith(() => _FakeMeNotifier({'totpEnabled': false})),
        ],
        child: MaterialApp(home: SecurityPage(onBack: () {})),
      ),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));

    // Devices section rendered
    expect(find.text('Pixel 9 Pro'), findsOneWidget);
    expect(find.text('THIS DEVICE · Active now'), findsOneWidget);
    expect(find.text('THIS DEVICE'), findsOneWidget);
    expect(find.text('MacBook Air'), findsOneWidget);

    // Revocation icon should be available for other session
    final revokeBtn = find.byTooltip('Revoke session');
    expect(revokeBtn, findsOneWidget);

    // Tap revoke button to open confirmation dialog
    await tester.tap(revokeBtn);
    await tester.pumpAndSettle();

    expect(find.text('Revoke session?'), findsOneWidget);
    expect(find.text('Sign out device'), findsOneWidget);

    // Confirm revocation
    await tester.tap(find.text('Sign out device'));
    await tester.pumpAndSettle();

    expect(revokedSessionId, 'ses_other');
  });

  test('AuthNotifier verifyTwoFactorLogin sanitizes spaced TOTP codes', () async {
    SharedPreferences.setMockInitialValues({});
    final store = await LocalStore.init();

    String? receivedCode;
    final client = ApiClient(store);
    client.dio.httpClientAdapter = _CustomAdapter((opts) {
      if (opts.path == '/auth/login') {
        return '{"twoFactorRequired":true,"twoFactorToken":"pending-token-spaced"}';
      }
      if (opts.path == '/auth/2fa/login-verify') {
        final body = opts.data as Map?;
        receivedCode = body?['code'] as String?;
        return '{"token":"jwt-token","user":{"id":"usr_2","email":"spaced@example.com"}}';
      }
      return '{}';
    });

    final container = ProviderContainer(
      overrides: [
        localStoreProvider.overrideWithValue(store),
        apiProvider.overrideWithValue(client),
      ],
    );

    final auth = container.read(authProvider.notifier);
    await auth.login('spaced@example.com', 'pass123456');
    final err = await auth.verifyTwoFactorLogin('123 456');
    expect(err, isNull);
    expect(receivedCode, '123456');
  });
}

