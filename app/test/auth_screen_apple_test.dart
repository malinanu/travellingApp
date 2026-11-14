// Sign in with Apple is only offered on iOS — Android must not show it.

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:spotto/core/storage/local_store.dart';
import 'package:spotto/data/repositories/repositories.dart';
import 'package:spotto/features/traveller/auth/auth_screen.dart';

Future<void> _pump(WidgetTester tester) async {
  tester.view.physicalSize = const Size(900, 2400);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.reset);
  // The screen reads its strings from the saved app language, so it needs a
  // local store.
  SharedPreferences.setMockInitialValues({});
  final store = await LocalStore.init();
  await tester.pumpWidget(
    ProviderScope(
      overrides: [localStoreProvider.overrideWithValue(store)],
      child: const MaterialApp(home: AuthScreen()),
    ),
  );
  await tester.pump();
}

void main() {
  setUp(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(
      const MethodChannel('plugins.it_nomads.com/flutter_secure_storage'),
      (call) async => null,
    );
  });

  testWidgets('hides Sign in with Apple on Android', (tester) async {
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    try {
      await _pump(tester);
      expect(find.text('Sign In'), findsWidgets);
      expect(find.text('Sign in with Apple'), findsNothing);
    } finally {
      debugDefaultTargetPlatformOverride = null;
    }
  });

  // Apple sign-in is opt-in per build (--dart-define=ENABLE_APPLE_SIGN_IN=true),
  // so the default build, even on iOS, doesn't show a button that can't work.
  testWidgets('hides Sign in with Apple on iOS unless the build enables it',
      (tester) async {
    debugDefaultTargetPlatformOverride = TargetPlatform.iOS;
    try {
      await _pump(tester);
      expect(find.text('Sign in with Apple'), findsNothing);
    } finally {
      debugDefaultTargetPlatformOverride = null;
    }
  });
}
