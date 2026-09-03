import 'dart:async';

import 'package:flutter/material.dart';

import '../../api/endpoints.dart';
import '../../core/format.dart';
import '../../widgets/commons.dart';
import 'detail.dart';

/// 支付页:mock 模式调 unifiedorder(返回 code_url),然后轮询订单状态。
class PayPage extends StatefulWidget {
  final String orderNo;
  final int orderId;
  const PayPage({super.key, required this.orderNo, required this.orderId});

  @override
  State<PayPage> createState() => _PayPageState();
}

class _PayPageState extends State<PayPage> {
  int _status = 0;
  double _amount = 0;
  bool _paying = false;
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    _pollOnce();
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  Future<void> _pollOnce() async {
    try {
      final d = await api.order(widget.orderId);
      if (mounted) {
        setState(() {
          _status = d['status'] as int? ?? _status;
          _amount = (d['rent_amount'] as num? ?? 0).toDouble();
        });
      }
    } catch (_) {}
  }

  Future<void> _pay() async {
    setState(() => _paying = true);
    try {
      await api.unifiedOrder(widget.orderNo);
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('支付已发起(mock),等待回调…'), duration: Duration(seconds: 2)));
      // 轮询支付结果:每 1.5s,最多 20 次。
      var n = 0;
      _timer?.cancel();
      _timer = Timer.periodic(const Duration(milliseconds: 1500), (t) async {
        n++;
        await _pollOnce();
        if (!mounted || _status != 0 || n >= 20) t.cancel();
      });
    } catch (e) {
      toast(e);
    } finally {
      if (mounted) setState(() => _paying = false);
    }
  }

  Future<void> _goDetail() async {
    await Navigator.of(context).pushReplacement(
        MaterialPageRoute(builder: (_) => OrderDetailPage(orderId: widget.orderId)));
  }

  @override
  Widget build(BuildContext context) {
    final paid = _status != 0;
    return Scaffold(
      appBar: AppBar(title: const Text('收银台'), automaticallyImplyLeading: false),
      body: Center(
        child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
          Text('¥${fmtMoney(_amount)}', style: const TextStyle(fontSize: 40, fontWeight: FontWeight.bold)),
          const SizedBox(height: 8),
          Text(paid ? '支付成功' : '待支付', style: TextStyle(fontSize: 16, color: paid ? kGreen : Colors.grey)),
          const SizedBox(height: 12),
          Text(widget.orderNo, style: TextStyle(fontSize: 12, color: Colors.grey.shade500)),
          const SizedBox(height: 60),
          FilledButton(
            style: FilledButton.styleFrom(
                backgroundColor: paid ? Colors.green : const Color(0xFFFB8C00),
                padding: const EdgeInsets.symmetric(horizontal: 60, vertical: 14)),
            onPressed: paid ? _goDetail : (_paying ? null : _pay),
            child: Text(paid ? '查看订单' : (_paying ? '发起中…' : '去支付')),
          ),
        ]),
      ),
    );
  }
}
