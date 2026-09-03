import 'package:flutter/foundation.dart';

import '../api/endpoints.dart';
import '../core/storage.dart';
import '../models/user.dart';

class UserStore extends ChangeNotifier {
  bool _logged = false;
  bool get logged => _logged;
  int? _uid;
  int? get uid => _uid;
  UserProfile? _profile;
  UserProfile? get profile => _profile;

  Future<void> init() async {
    _logged = await TokenStorage.logged;
    notifyListeners();
  }

  Future<void> setTokens(String access, String refresh) async {
    await TokenStorage.save(access, refresh);
    _logged = true;
    notifyListeners();
  }

  Future<void> login(String phone, String code) async {
    final d = await api.login(phone, code);
    await setTokens(d['access_token'] as String, d['refresh_token'] as String);
  }

  /// 登出:先调 POST /auth/logout 使服务端 refresh 会话失效,再清本地。
  Future<void> logout() async {
    try {
      await api.logout();
    } catch (_) {} // 网络异常也继续本地清理
    await TokenStorage.clear();
    _logged = false;
    _profile = null;
    notifyListeners();
  }

  Future<void> loadProfile() async {
    try {
      final d = await api.profile();
      _profile = UserProfile.fromJson(d);
      _uid = _profile!.id > 0 ? _profile!.id : null;
      notifyListeners();
    } catch (_) {} // 未登录/网络失败静默,页面自行兜底
  }

  Future<void> updateProfile({String? nickname, String? avatar}) async {
    await api.updateProfile(nickname: nickname, avatar: avatar);
    await loadProfile();
  }
}
