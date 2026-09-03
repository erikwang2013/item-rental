import 'package:flutter/material.dart';
import 'package:image_picker/image_picker.dart';
import 'package:provider/provider.dart';

import '../../api/endpoints.dart';
import '../../core/format.dart';
import '../../models/user.dart';
import '../../stores/user_store.dart';
import '../../widgets/commons.dart';

/// 编辑资料:头像(选图上传,服务端直接落库)+ 昵称(PUT /user/profile 仅落 nickname)。
class ProfilePage extends StatefulWidget {
  const ProfilePage({super.key});

  @override
  State<ProfilePage> createState() => _ProfilePageState();
}

class _ProfilePageState extends State<ProfilePage> {
  final _nick = TextEditingController();
  String _avatarUrl = '';
  bool _saving = false;
  bool _uploading = false;

  @override
  void initState() {
    super.initState();
    final p = context.read<UserStore>().profile;
    _nick.text = p?.nickname ?? '';
    _avatarUrl = p?.avatar ?? '';
  }

  @override
  void dispose() {
    _nick.dispose();
    super.dispose();
  }

  Future<void> _pickAvatar() async {
    final XFile? f;
    try {
      f = await ImagePicker().pickImage(source: ImageSource.gallery);
    } catch (_) {
      return toast('无法打开相册');
    }
    if (f == null || !mounted) return;
    setState(() => _uploading = true);
    try {
      final url = await api.uploadAvatar(f);
      if (!mounted) return;
      setState(() => _avatarUrl = url);
      context.read<UserStore>().loadProfile(); // 同步我的页头像
      toast('头像已更新');
    } catch (e) {
      toast(e);
    } finally {
      if (mounted) setState(() => _uploading = false);
    }
  }

  Future<void> _save() async {
    final nick = _nick.text.trim();
    if (nick.isEmpty) return toast('昵称不能为空');
    setState(() => _saving = true);
    try {
      await context.read<UserStore>().updateProfile(nickname: nick);
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
          child: Column(children: [
            Stack(children: [
              CircleAvatar(
                radius: 40,
                backgroundColor: kGreen,
                backgroundImage:
                    _avatarUrl.isNotEmpty ? NetworkImage(_avatarUrl) : null,
                child: _avatarUrl.isEmpty
                    ? const Icon(Icons.person, color: Colors.white, size: 44)
                    : null,
              ),
              if (_uploading)
                const Positioned.fill(
                  child: ColoredBox(
                    color: Colors.black38,
                    child: Center(
                        child: SizedBox(
                            width: 24,
                            height: 24,
                            child: CircularProgressIndicator(
                                strokeWidth: 2, color: Colors.white))),
                  ),
                ),
            ]),
            const SizedBox(height: 8),
            TextButton.icon(
              onPressed: _uploading ? null : _pickAvatar,
              icon: const Icon(Icons.photo_library_outlined, size: 18),
              label: const Text('更换头像'),
            ),
          ]),
        ),
        const SizedBox(height: 8),
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
