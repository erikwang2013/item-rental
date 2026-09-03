import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/format.dart';
import '../../models/user.dart';
import '../../stores/user_store.dart';
import '../../widgets/commons.dart';

/// 编辑资料:昵称 / 头像 URL(服务端 PUT 仅落 nickname/avatar 两列)。
class ProfilePage extends StatefulWidget {
  const ProfilePage({super.key});

  @override
  State<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends State<ProfilePage> {
  final _nick = TextEditingController();
  final _avatar = TextEditingController();
  bool _saving = false;

  @override
  void initState() {
    super.initState();
    final p = context.read<UserStore>().profile;
    _nick.text = p?.nickname ?? '';
    _avatar.text = p?.avatar ?? '';
  }

  @override
  void dispose() {
    _nick.dispose();
    _avatar.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    final nick = _nick.text.trim();
    final avatar = _avatar.text.trim();
    if (nick.isEmpty) return toast('昵称不能为空');
    setState(() => _saving = true);
    try {
      await context.read<UserStore>().updateProfile(
          nickname: nick, avatar: avatar.isEmpty ? null : avatar);
      if (!mounted) return;
      toast('保存成功');
      Navigator.of(context).pop();
    } catch (e) {
      toast(e);
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final UserProfile? p = context.read<UserStore>().profile;
    return Scaffold(
      appBar: AppBar(title: const Text('编辑资料')),
      body: ListView(padding: const EdgeInsets.all(16), children: [
        Center(
          child: CircleAvatar(
            radius: 40,
            backgroundColor: kGreen,
            backgroundImage: _avatar.text.isNotEmpty ? NetworkImage(_avatar.text) : null,
            child: _avatar.text.isEmpty
                ? const Icon(Icons.person, color: Colors.white, size: 44)
                : null,
          ),
        ),
        const SizedBox(height: 16),
        TextField(
          controller: _avatar,
          decoration: const InputDecoration(labelText: '头像 URL'),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _nick,
          maxLength: 20,
          decoration: const InputDecoration(labelText: '昵称'),
        ),
        const SizedBox(height: 8),
        if (p != null)
          Text(
            '手机号 ${p.id > 0 ? '(已绑定)' : ''} · 实名 ${p.realName.isEmpty ? '未实名' : p.realName}\n'
            '信用分 ${p.creditScore} · 押金余额 ¥${fmtMoney(p.depositBal)}',
            style: TextStyle(color: Colors.grey.shade600, height: 1.6, fontSize: 13),
          ),
      ]),
      bottomNavigationBar: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 12),
          child: FilledButton(
            style: FilledButton.styleFrom(
                backgroundColor: kGreen, padding: const EdgeInsets.symmetric(vertical: 14)),
            onPressed: _saving ? null : _save,
            child: Text(_saving ? '保存中…' : '保存'),
          ),
        ),
      ),
    );
  }
}
