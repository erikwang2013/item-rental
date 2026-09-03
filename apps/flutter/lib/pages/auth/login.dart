import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/endpoints.dart';
import '../../stores/user_store.dart';
import '../../widgets/commons.dart';

/// 手机号 + 验证码登录(验证码 60s 限频)。
class LoginPage extends StatefulWidget {
  const LoginPage({super.key});

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final _phone = TextEditingController();
  final _code = TextEditingController();
  bool _loading = false;
  int _countdown = 0;
  Timer? _timer;

  @override
  void dispose() {
    _timer?.cancel();
    _phone.dispose();
    _code.dispose();
    super.dispose();
  }

  bool get _phoneOk => RegExp(r'^1\d{10}$').hasMatch(_phone.text.trim());

  Future<void> _sendCode() async {
    if (!_phoneOk) {
      toast('请输入正确的手机号');
      return;
    }
    try {
      await api.sms(_phone.text.trim());
      toast('验证码已发送');
      setState(() => _countdown = 60);
      _timer = Timer.periodic(const Duration(seconds: 1), (t) {
        if (_countdown <= 1) {
          t.cancel();
          setState(() => _countdown = 0);
        } else {
          setState(() => _countdown--);
        }
      });
    } catch (e) {
      toast(e);
    }
  }

  Future<void> _login() async {
    if (!_phoneOk) return toast('请输入正确的手机号');
    if (_code.text.trim().length < 4) return toast('请输入验证码');
    setState(() => _loading = true);
    try {
      await context.read<UserStore>().login(_phone.text.trim(), _code.text.trim());
      if (!mounted) return;
      Navigator.of(context).pop(true);
    } catch (e) {
      toast(e);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('登录 / 注册')),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const SizedBox(height: 20),
            const Text('闲租', style: TextStyle(fontSize: 34, fontWeight: FontWeight.bold, color: kGreen)),
            const Text('手机号登录,新用户自动注册', style: TextStyle(color: Colors.grey)),
            const SizedBox(height: 32),
            TextField(
              controller: _phone,
              keyboardType: TextInputType.phone,
              maxLength: 11,
              decoration: const InputDecoration(
                  labelText: '手机号', prefixIcon: Icon(Icons.phone_android),
                  counterText: ''),
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _code,
                    keyboardType: TextInputType.number,
                    maxLength: 6,
                    decoration: const InputDecoration(
                        labelText: '验证码', prefixIcon: Icon(Icons.sms_outlined),
                        counterText: ''),
                  ),
                ),
                const SizedBox(width: 12),
                OutlinedButton(
                  onPressed: _countdown > 0 ? null : _sendCode,
                  child: Text(_countdown > 0 ? '${_countdown}s后重发' : '获取验证码'),
                ),
              ],
            ),
            const SizedBox(height: 28),
            FilledButton(
              style: FilledButton.styleFrom(backgroundColor: kGreen, padding: const EdgeInsets.symmetric(vertical: 14)),
              onPressed: _loading ? null : _login,
              child: Text(_loading ? '登录中…' : '登录'),
            ),
          ],
        ),
      ),
    );
  }
}
