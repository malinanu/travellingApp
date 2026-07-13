// SharedPreferences wrapper mirroring the prototype's localStorage keys.

import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

abstract final class StoreKeys {
  static const auth = 'spotto_auth';
  static const token = 'spotto_token'; // API JWT
  static const prefs = 'spotto_prefs';
  static const group = 'spotto_group';
  static const trip = 'spotto_trip';
  static const claims = 'spotto_claims'; // placeId -> claimed deal code
  static const places = 'spotto_places'; // last good /places response
}

class LocalStore {
  final SharedPreferences _prefs;
  LocalStore(this._prefs);

  static Future<LocalStore> init() async => LocalStore(await SharedPreferences.getInstance());

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
