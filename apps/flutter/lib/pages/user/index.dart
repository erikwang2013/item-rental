import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/format.dart';
import '../../models/user.dart';
import '../../stores/message_store.dart';
import '../../stores/user_store.dart';
import '../../widgets/commons.dart';
import '../../widgets/routes.dart';
import '../auth/login.dart';
import '../messages/index.dart';
import '../order/list.dart';
import '../seller/items.dart';
import 'profile.dart';

/// 我的:资料 + 订单/我的物品/消息 + 登出(调后端)。
class UserPage extends StatefulWidget {
  const UserPage({super.key});

  @override
  State<UserPage> createState() => _UserPageState();
}

class _UserPageState extends State<UserPage> {
  @override
  Widget build(BuildContext context) {
    final us = context.watch<UserStore>();
    final ms = context.watch<MessageStore>();
    final UserProfile? p = us.profile;
    return Scaffold(
      appBar: AppBar(backgroundColor: kGreen, foregroundColor: Colors.white, title: const Text('我的')),
      body: us.logged
          ? ListView(children: [
              _Header(p: p, onTap: () => go(context, const ProfilePage()).then((_) => us.loadProfile())),
              ListTile(
                leading: const Icon(Icons.receipt_long_outlined),
                title: const Text('我的订单'),
                trailing: const Icon(Icons.chevron_right),
                onTap: () => go(context, const OrderListPage()),
              ),
              ListTile(
                leading: const Icon(Icons.storefront_outlined),
                title: const Text('我的物品'),
                trailing: const Icon(Icons.chevron_right),
                onTap: () => go(context, SellerItemsPage()),
              ),
              ListTile(
                leading: const Icon(Icons.notifications_outlined),
                title: const Text('消息中心'),
                trailing: Row(mainAxisSize: MainAxisSize.min, children: [
                  if (ms.unread > 0)
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                      decoration: BoxDecoration(color: Colors.red, borderRadius: BorderRadius.circular(10)),
                      child: Text('${ms.unread}', style: const TextStyle(color: Colors.white, fontSize: 12)),
                    ),
                  const Icon(Icons.chevron_right),
                ]),
                onTap: () async {
                  await go(context, const MessagesPage());
                  ms.refresh();
                },
              ),
              const Divider(),
              ListTile(
                leading: const Icon(Icons.logout, color: Colors.red),
                title: const Text('退出登录', style: TextStyle(color: Colors.red)),
                onTap: () async {
                  final ok = await showDialog<bool>(
                    context: context,
                    builder: (ctx) => AlertDialog(
                      content: const Text('确定退出登录?'),
                      actions: [
                        TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
                        FilledButton(
                            onPressed: () => Navigator.pop(ctx, true), child: const Text('退出')),
                      ],
                    ),
                  );
                  if (ok == true) {
                    await us.logout(); // 内部调 POST /auth/logout + 本地清 token
                    ms.clear();
                  }
                },
              ),
            ])
          : Center(
              child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
                const Icon(Icons.account_circle_outlined, size: 72, color: Colors.grey),
                const SizedBox(height: 12),
                FilledButton(
                  style: FilledButton.styleFrom(backgroundColor: kGreen),
                  onPressed: () async {
                    await go<bool>(context, const LoginPage());
                    us.loadProfile();
                  },
                  child: const Text('登录 / 注册'),
                ),
              ]),
            ),
    );
  }
}

class _Header extends StatelessWidget {
  final UserProfile? p;
  final VoidCallback onTap;
  const _Header({required this.p, required this.onTap});

  @override
  Widget build(BuildContext context) {
    final UserProfile? up = p; // 局部变量以支持空安全提升
    return InkWell(
      onTap: onTap,
      child: Container(
        color: kGreen.withValues(alpha: .08),
        padding: const EdgeInsets.all(16),
        child: Row(children: [
          CircleAvatar(
            radius: 28,
            backgroundColor: kGreen,
            backgroundImage:
                up != null && up.avatar.isNotEmpty ? NetworkImage(up.avatar) : null,
            child: up == null || up.avatar.isEmpty
                ? const Icon(Icons.person, color: Colors.white, size: 30)
                : null,
          ),
          const SizedBox(width: 14),
          Expanded(
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text(up?.nickname ?? '加载中…',
                  style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
              if (up != null)
                Text('信用分 ${up.creditScore} · 押金余额 ¥${fmtMoney(up.depositBal)}',
                    style: TextStyle(fontSize: 12, color: Colors.grey.shade600)),
            ]),
          ),
          const Icon(Icons.chevron_right),
        ]),
      ),
    );
  }
}
