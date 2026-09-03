import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../pages/auth/login.dart';
import '../stores/user_store.dart';

/// 统一 push 新页面(全屏 MaterialPageRoute)。
Future<T?> go<T>(BuildContext context, Widget page) =>
    Navigator.of(context).push<T>(MaterialPageRoute(builder: (_) => page));

/// 登录门:未登录先推登录页;返回 true 表示已登录/刚登录成功。
Future<bool> ensureLogin(BuildContext context) async {
  final store = context.read<UserStore>();
  if (store.logged) return true;
  final ok = await Navigator.of(context).push<bool>(
      MaterialPageRoute(builder: (_) => const LoginPage()));
  return ok == true;
}
