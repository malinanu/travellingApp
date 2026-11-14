import 'package:flutter_test/flutter_test.dart';
import 'package:spotto/core/maps/maps_bundle_check.dart';

void main() {
  test('the real bundle ID is not re-signed', () {
    expect(isResignedBundle('com.spotto.spotto'), isFalse);
    expect(isResignedBundle(' com.spotto.spotto\n'), isFalse);
  });

  test('a suffixed or prefixed bundle ID is re-signed', () {
    expect(isResignedBundle('com.spotto.spotto.ABCDE12345'), isTrue);
    expect(isResignedBundle('com.ABCDE12345.com.spotto.spotto'), isTrue);
  });

  test('an empty or unknown bundle ID is not flagged', () {
    expect(isResignedBundle(''), isFalse);
    expect(isResignedBundle('   '), isFalse);
  });
}
