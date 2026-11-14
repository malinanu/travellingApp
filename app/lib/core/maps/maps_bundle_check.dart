// Detects an iOS install whose bundle ID was changed by re-signing.
//
// Sideload tools (Sideloadly, AltStore) re-sign the IPA under a personal
// Apple ID and rewrite the bundle ID, e.g. `com.spotto.spotto.ABCDE12345`.
// The Maps key is restricted to the real bundle ID, so Google then refuses
// tiles: the map shows markers and the Google logo on a blank background.
// There is no console on a sideloaded phone, so Discover surfaces this.

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:package_info_plus/package_info_plus.dart';

const kExpectedIosBundleId = 'com.spotto.spotto';

/// True when [runtimeId] is a real bundle ID that differs from the one the
/// Maps key is restricted to. Empty/unknown IDs are not treated as re-signed.
bool isResignedBundle(String runtimeId) {
  final id = runtimeId.trim();
  return id.isNotEmpty && id != kExpectedIosBundleId;
}

/// The re-signed bundle ID this iOS install runs under, or null when the
/// install is correctly signed, not iOS, or the ID can't be read.
final resignedBundleIdProvider = FutureProvider<String?>((ref) async {
  if (kIsWeb || defaultTargetPlatform != TargetPlatform.iOS) return null;
  try {
    final id = (await PackageInfo.fromPlatform()).packageName;
    return isResignedBundle(id) ? id.trim() : null;
  } catch (_) {
    return null;
  }
});
