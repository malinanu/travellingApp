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
  static const refreshToken = 'spotto_refresh_token'; // session refresh token — secure storage too
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
  static const favoritesUploaded = 'spotto_favorites_uploaded'; // local bookmarks sent to /me/saved-places once
  static const boardLinks = 'spotto_board_links'; // last /me/inspirations response
  static const boardMigrated = 'spotto_board_migrated'; // trip-only inspiration links uploaded once
  static const savedPlaces = 'spotto_saved_places'; // last /me/saved-places response
  static const categories = 'spotto_categories'; // fetched dynamic categories
  static const categoriesAt = 'spotto_categories_at'; // when categories were last fetched (ms epoch)
  static const lastPosition = 'spotto_last_pos'; // {lat, lng, at} of the last device fix
  // Last good responses for the rest of the Discover screen, so it opens
  // fully populated instead of on placeholders while the network catches up.
  static const me = 'spotto_me'; // last /me
  static const quest = 'spotto_quest'; // {day, data} of the last /me/quest
  static const weather = 'spotto_weather'; // {at, data} of the last current-location /weather
}

class LocalStore {
  final SharedPreferences _prefs;
  final FlutterSecureStorage _secure;

  // flutter_secure_storage has no synchronous read (Keychain/Keystore access
  // is inherently async on both platforms), but callers throughout the app
  // expect a synchronous getToken() — so it's loaded once into memory here
  // during init() and kept current on every write/clear.
  String? _cachedToken;
  // Same in-memory copy for the refresh token, which keeps the session alive
  // past the 1-hour access token (see ApiClient's refresh-and-retry).
  String? _cachedRefresh;

  LocalStore(this._prefs, this._secure, this._cachedToken, [this._cachedRefresh]);

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
      try {
        await secure.delete(key: StoreKeys.token);
      } catch (_) {
        // Keychain unusable altogether (e.g. a re-signed sideloaded iOS
        // build) — still must not stop the app from launching.
      }
    }
    // One-time migration for installs from before secure storage: move a
    // plaintext token left over in SharedPreferences into secure storage.
    final legacy = prefs.getString(StoreKeys.token);
    var migrated = true;
    if (token == null && legacy != null) {
      try {
        await secure.write(key: StoreKeys.token, value: legacy);
      } catch (_) {
        migrated = false; // keychain unusable: keep the legacy copy
      }
      token = legacy;
    }
    if (legacy != null && migrated) {
      await prefs.remove(StoreKeys.token);
    }
    String? refresh;
    try {
      refresh = await secure.read(key: StoreKeys.refreshToken);
    } catch (_) {
      // Same unrecoverable-keystore case as the token above; without it the
      // app just signs in again once the access token expires.
    }
    return LocalStore(prefs, secure, token, refresh);
  }

  String? getToken() => _cachedToken;
  String? getRefreshToken() => _cachedRefresh;

  /// Saves a new access token, and the refresh token when the server sent one.
  Future<void> setToken(String token, {String? refreshToken}) async {
    _cachedToken = token;
    if (refreshToken != null) _cachedRefresh = refreshToken;
    await _secure.write(key: StoreKeys.token, value: token);
    if (refreshToken != null) await _secure.write(key: StoreKeys.refreshToken, value: refreshToken);
  }

  /// Signs this device out locally: drops both the access and refresh token.
  Future<void> clearToken() async {
    _cachedToken = null;
    _cachedRefresh = null;
    await _secure.delete(key: StoreKeys.token);
    await _secure.delete(key: StoreKeys.refreshToken);
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

  Future<void> setJson(String key, Map<String, dynamic> value) => _prefs.setString(key, jsonEncode(value));

  List<dynamic>? getJsonList(String key) {
    final raw = _prefs.getString(key);
    if (raw == null) return null;
    try {
      return jsonDecode(raw) as List<dynamic>;
    } catch (_) {
      return null;
    }
  }

  Future<void> setJsonList(String key, List<dynamic> value) => _prefs.setString(key, jsonEncode(value));

  Future<void> remove(String key) => _prefs.remove(key);
}
