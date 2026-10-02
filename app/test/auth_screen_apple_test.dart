// Sign in with Apple is only offered on iOS — Android must not show it.

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:spotto/features/traveller/auth/auth_screen.dart';

Future<void> _pump(WidgetTester tester) async {
  tester.view.physicalSize = const Size(900, 2400);
  tester.view.devicePixelRatio = 1.0;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(
    const ProviderScope(child: MaterialApp(home: AuthScreen())),
  );
  await tester.pump();
}

void main() {
  testWidgets('hides Sign in with Apple on Android', (tester) async {
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    try {
      await _pump(tester);
      expect(find.text('Sign in'), findsWidgets);
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
