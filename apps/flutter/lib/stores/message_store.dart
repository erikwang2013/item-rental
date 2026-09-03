import 'package:flutter/foundation.dart';

import '../api/endpoints.dart';

/// 未读角标计数;登出后清零。列表页数据各自持有,不入此 store。
class MessageStore extends ChangeNotifier {
  int _unread = 0;
  int get unread => _unread;

  Future<void> refresh() async {
    try {
      final d = await api.messages(pageSize: 1);
      _unread = d['unread'] as int? ?? 0;
      notifyListeners();
    } catch (_) {} // 未登录/失败时保持现状
  }

  void clear() {
    _unread = 0;
    notifyListeners();
  }
}
