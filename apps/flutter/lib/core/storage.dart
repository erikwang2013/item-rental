import 'package:shared_preferences/shared_preferences.dart';

/// Token 持久化: access_token / refresh_token,与后端键名对应。
class TokenStorage {
  static const _accessKey = 'access_token';
  static const _refreshKey = 'refresh_token';

  static Future<String?> get access async =>
      (await SharedPreferences.getInstance()).getString(_accessKey);

  static Future<String?> get refresh async =>
      (await SharedPreferences.getInstance()).getString(_refreshKey);

  /// 双 token 齐全才视为已登录(仅 access 失效可自动刷新恢复)。
  static Future<bool> get logged async {
    final p = await SharedPreferences.getInstance();
    return (p.getString(_accessKey) ?? '').isNotEmpty &&
        (p.getString(_refreshKey) ?? '').isNotEmpty;
  }

  static Future<void> save(String? access, String? refresh) async {
    final prefs = await SharedPreferences.getInstance();
    if (access != null) await prefs.setString(_accessKey, access);
    if (refresh != null) await prefs.setString(_refreshKey, refresh);
  }

  static Future<void> clear() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_accessKey);
    await prefs.remove(_refreshKey);
  }
}
