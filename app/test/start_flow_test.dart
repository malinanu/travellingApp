// The start-up order is the same on every platform: onboarding questions on
// a first launch, then sign-in, then the app (app/router.dart).

import 'package:flutter_test/flutter_test.dart';
import 'package:spotto/app/router.dart';

void main() {
  test('first launch starts with the onboarding questions', () {
    expect(startRedirect(signedIn: false, hasPrefs: false, onboarded: false), '/t/onboarding');
  });

  test('after onboarding, sign-in comes next', () {
    expect(startRedirect(signedIn: false, hasPrefs: true, onboarded: true), '/t/auth');
  });

  test('signing out leads to sign-in, not back through the questions', () {
    // Sign-out clears the saved preferences but not the onboarded flag.
    expect(startRedirect(signedIn: false, hasPrefs: false, onboarded: true), '/t/auth');
  });

  test('a signed-in account without preferences answers the questions', () {
    expect(startRedirect(signedIn: true, hasPrefs: false, onboarded: true), '/t/onboarding');
  });

  test('signed in with preferences goes straight in', () {
    expect(startRedirect(signedIn: true, hasPrefs: true, onboarded: true), isNull);
    expect(startRedirect(signedIn: true, hasPrefs: true, onboarded: false), isNull);
  });
}
