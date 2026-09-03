import 'package:flutter/material.dart';

import '../../api/endpoints.dart';
import '../../core/format.dart';
import '../../models/item.dart';
import '../../widgets/commons.dart';
import 'pay.dart';

/// 下单确认:选日期区间 → 展示天数/金额 → 创建订单 → 跳支付。
class OrderConfirmPage extends StatefulWidget {
  final Item item;
  const OrderConfirmPage({super.key, required this.item});

  @override
  State<OrderConfirmPage> createState() => _OrderConfirmPageState();
}

class _OrderConfirmPageState extends State<OrderConfirmPage> {
  DateTime? _start;
  DateTime? _end;
  bool _creating = false;

  int get _days {
    if (_start == null || _end == null || !_end!.isAfter(_start!)) return 0;
    return _end!.difference(_start!).inDays;
  }

  Future<void> _pickStart() async {
    final d = await showDatePicker(
        context: context,
        initialDate: _start ?? DateTime.now().add(const Duration(days: 1)),
        firstDate: DateTime.now(),
        lastDate: DateTime.now().add(const Duration(days: 365)));
    if (d != null) setState(() => _start = d);
  }

  Future<void> _pickEnd() async {
    final first = _start ?? DateTime.now().add(const Duration(days: 1));
    final d = await showDatePicker(
        context: context,
        initialDate: _end ?? first.add(const Duration(days: 1)),
        firstDate: first.add(const Duration(days: 1)),
        lastDate: DateTime.now().add(const Duration(days: 400)));
    if (d != null) setState(() => _end = d);
  }

  Future<void> _submit() async {
    if (_days <= 0) return toast('请选择正确的租赁起止日期');
    setState(() => _creating = true);
    try {
      final o = await api.createOrder(
          itemId: widget.item.id.toString(),
          startDate: toDateStr(_start!),
          endDate: toDateStr(_end!));
      if (!mounted) return;
      await Navigator.of(context).pushReplacement(MaterialPageRoute(
          builder: (_) => PayPage(
              orderNo: o['order_no'] as String? ?? '',
              orderId: int.tryParse('${o['id']}') ?? 0)));
    } catch (e) {
      toast(e);
    } finally {
      if (mounted) setState(() => _creating = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final it = widget.item;
    final rent = it.dailyPrice * _days;
    final deposit = it.deposit;
    return Scaffold(
      appBar: AppBar(title: const Text('确认订单')),
      body: ListView(padding: const EdgeInsets.all(14), children: [
        ListTile(
          contentPadding: EdgeInsets.zero,
          leading: it.cover.isEmpty
              ? const Icon(Icons.inventory_2_outlined, size: 40)
              : ClipRRect(
                  borderRadius: BorderRadius.circular(6),
                  child: Image.network(it.cover, width: 52, height: 52, fit: BoxFit.cover,
                      errorBuilder: (_, _, _) => const Icon(Icons.broken_image_outlined, size: 40))),
          title: Text(it.title, maxLines: 1, overflow: TextOverflow.ellipsis),
          subtitle: Text('¥${fmtMoney(it.dailyPrice)}/天 · 押金 ¥${fmtMoney(it.deposit)}'),
        ),
        const Divider(),
        ListTile(
          contentPadding: EdgeInsets.zero,
          title: const Text('开始日期'),
          trailing: Text(_start == null ? '请选择' : toDateStr(_start!)),
          onTap: _pickStart,
        ),
        ListTile(
          contentPadding: EdgeInsets.zero,
          title: const Text('结束日期'),
          trailing: Text(_end == null ? '请选择' : toDateStr(_end!)),
          onTap: _pickEnd,
        ),
        const Divider(),
        ListTile(
          contentPadding: EdgeInsets.zero,
          title: const Text('租赁天数'),
          trailing: Text('$_days 天'),
        ),
        ListTile(
          contentPadding: EdgeInsets.zero,
          title: const Text('租金小计'),
          trailing: Text('¥${fmtMoney(rent)}',
              style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
        ),
        ListTile(
          contentPadding: EdgeInsets.zero,
          title: const Text('押金(取货时冻结)'),
          trailing: Text('¥${fmtMoney(deposit)}'),
        ),
        ListTile(
          contentPadding: EdgeInsets.zero,
          title: const Text('合计(租金)'),
          trailing: Text('¥${fmtMoney(rent)}',
              style: const TextStyle(color: Color(0xFFE53935), fontSize: 20, fontWeight: FontWeight.bold)),
        ),
      ]),
      bottomNavigationBar: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 12),
          child: FilledButton(
            style: FilledButton.styleFrom(
                backgroundColor: kGreen, padding: const EdgeInsets.symmetric(vertical: 14)),
            onPressed: _creating ? null : _submit,
            child: Text(_creating ? '提交中…' : '提交订单'),
          ),
        ),
      ),
    );
  }
}
