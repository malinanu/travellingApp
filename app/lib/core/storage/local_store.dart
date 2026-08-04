// SharedPreferences wrapper mirroring the prototype's localStorage keys.
// The session JWT is the one exception — it's kept in the OS-level secure
// store (Keychain on iOS, Keystore-backed EncryptedSharedPreferences on
// Android) instead of plaintext SharedPreferences, since it's a bearer
// credential: readable via ADB backup extraction or on a rooted device
// otherwise.

import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

abstract final class StoreKeys {
  static const auth = 'spotto_auth';
  static const token = 'spotto_token'; // API JWT — secure storage, see below
  static const prefs = 'spotto_prefs';
  static const group = 'spotto_group';
  static const trip = 'spotto_trip';
  static const allTrips = 'spotto_all_trips';
  static const claims = 'spotto_claims'; // placeId -> claimed deal code
  static const bookings = 'spotto_bookings'; // last good /me/bookings response
  static const places = 'spotto_places'; // last good /places response
  static const weatherPins = 'spotto_weather_pins'; // saved Weather-screen locations
  static const checkIns = 'spotto_checkins'; // placeIds checked into (artisan quest)
  static const favorites = 'spotto_favorites'; // bookmarked placeIds
}

class LocalStore {
  final SharedPreferences _prefs;
  final FlutterSecureStorage _secure;

  // flutter_secure_storage has no synchronous read (Keychain/Keystore access
  // is inherently async on both platforms), but callers throughout the app
  // expect a synchronous getToken() — so it's loaded once into memory here
  // during init() and kept current on every write/clear.
  String? _cachedToken;

  LocalStore(this._prefs, this._secure, this._cachedToken);

  static Future<LocalStore> init() async {
    final prefs = await SharedPreferences.getInstance();
    const secure = FlutterSecureStorage();
    String? token;
    try {
      token = await secure.read(key: StoreKeys.token);
    } catch (_) {
      // Corrupted/undecryptable secure storage (e.g. the Android Keystore
      // key changed since this value was written, such as after a debug
      // reinstall) — can't be recovered, so drop it and continue logged-out
      // instead of throwing here and crashing before runApp() ever runs.
      await secure.delete(key: StoreKeys.token);
    }
    // One-time migration for installs from before secure storage: move a
    // plaintext token left over in SharedPreferences into secure storage.
    final legacy = prefs.getString(StoreKeys.token);
    if (token == null && legacy != null) {
      await secure.write(key: StoreKeys.token, value: legacy);
      token = legacy;
    }
    if (legacy != null) {
      await prefs.remove(StoreKeys.token);
    }
    return LocalStore(prefs, secure, token);
  }

  String? getToken() => _cachedToken;

  Future<void> setToken(String token) async {
    _cachedToken = token;
    await _secure.write(key: StoreKeys.token, value: token);
  }

  Future<void> clearToken() async {
    _cachedToken = null;
    await _secure.delete(key: StoreKeys.token);
  }

  bool getBool(String key) => _prefs.getBool(key) ?? false;
  Future<void> setBool(String key, bool value) => _prefs.setBool(key, value);

  String? getString(String key) => _prefs.getString(key);
  Future<void> setString(String key, String value) => _prefs.setString(key, value);

  Map<String, dynamic>? getJson(String key) {
    final raw = _prefs.getString(key);
    if (raw == null) return null;
    try {
      return jsonDecode(raw) as Map<String, dynamic>;
    } catch (_) {
      return null;
    }
  }

  Future<void> setJson(String key, Map<String, dynamic> value) =>
      _prefs.setString(key, jsonEncode(value));

  List<dynamic>? getJsonList(String key) {
    final raw = _prefs.getString(key);
    if (raw == null) return null;
    try {
      return jsonDecode(raw) as List<dynamic>;
    } catch (_) {
      return null;
    }
  }

  Future<void> setJsonList(String key, List<dynamic> value) =>
      _prefs.setString(key, jsonEncode(value));

  Future<void> remove(String key) => _prefs.remove(key);
}
