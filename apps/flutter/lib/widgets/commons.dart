import 'package:flutter/material.dart';

import '../core/api_client.dart';

/// 主题绿。
const kGreen = Color(0xFF2E7D32);

/// 全局 SnackBar 出口(MaterialApp.scaffoldMessengerKey),避免跨 async gap 用 context。
final GlobalKey<ScaffoldMessengerState> kSnack = GlobalKey();

/// 加载/空态。
class StatusBox extends StatelessWidget {
  final bool loading;
  final String emptyText;
  const StatusBox({super.key, this.loading = false, this.emptyText = '暂无数据'});

  @override
  Widget build(BuildContext context) => Center(
        child: loading
            ? const CircularProgressIndicator()
            : Text(emptyText, style: const TextStyle(color: Colors.grey)),
      );
}

/// 错误信息 toast 化(ApiError 取用户可读 msg)。
void toast(Object e) {
  final msg = e is ApiError ? e.message : e.toString().replaceFirst('Exception: ', '');
  kSnack.currentState?.showSnackBar(
      SnackBar(content: Text(msg), duration: const Duration(seconds: 2)));
}
