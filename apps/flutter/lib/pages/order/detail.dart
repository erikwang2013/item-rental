import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/endpoints.dart';
import '../../core/format.dart';
import '../../models/order.dart';
import '../../models/user.dart';
import '../../stores/user_store.dart';
import '../../widgets/commons.dart';
import 'pay.dart';

/// 订单详情:状态信息 + 角色相关流转按钮。
class OrderDetailPage extends StatefulWidget {
  final int orderId;
  const OrderDetailPage({super.key, required this.orderId});

  @override
  State<OrderDetailPage> createState() => _OrderDetailPageState();
}

class _OrderDetailPageState extends State<OrderDetailPage> {
  Order? _o;
  bool _loading = true;
  bool _acting = false;

  @override
  void initState() {
    super.initState();
    _load();
    context.read<UserStore>().loadProfile(); // 拿 uid 判断角色(renter/owner)
  }

  Future<void> _load() async {
    try {
      final d = await api.order(widget.orderId);
      if (mounted) setState(() { _o = Order.fromJson(d); _loading = false; });
    } catch (e) {
      toast(e);
      if (mounted) setState(() => _loading = false);
    }
  }

  bool get _isOwner {
    final uid = context.read<UserStore>().uid;
    return uid != null && _o != null && uid == _o!.ownerId;
  }

  bool get _isRenter {
    final uid = context.read<UserStore>().uid;
    return uid != null && _o != null && uid == _o!.renterId;
  }

  Future<void> _act(String label, Future<void> Function() fn) async {
    setState(() => _acting = true);
    try {
      await fn();
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text('$label成功'), duration: const Duration(seconds: 1)));
      await _load();
    } catch (e) {
      toast(e);
    } finally {
      if (mounted) setState(() => _acting = false);
    }
  }

  Future<void> _confirmDialog(String msg, Future<void> Function() fn) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        content: Text(msg),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(
              onPressed: () => Navigator.pop(ctx, true), child: const Text('确认')),
        ],
      ),
    );
    if (ok == true) await _act('操作', fn);
  }

  @override
  Widget build(BuildContext context) {
    final o = _o;
    return Scaffold(
      appBar: AppBar(title: const Text('订单详情')),
      body: _loading
          ? const StatusBox(loading: true)
          : o == null
              ? const StatusBox(emptyText: '订单不存在')
              : RefreshIndicator(
                  onRefresh: _load,
                  child: ListView(padding: const EdgeInsets.all(14), children: [
                    Row(children: [
                      Text(o.statusText,
                          style: TextStyle(fontSize: 22, fontWeight: FontWeight.bold,
                              color: o.status == 4 || o.status == 1 ? kGreen : Colors.black87)),
                      const Spacer(),
                      Text(o.orderNo, style: TextStyle(fontSize: 12, color: Colors.grey.shade500)),
                    ]),
                    const SizedBox(height: 4),
                    if (o.cancelReason.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.only(bottom: 4),
                        child: Text('取消原因:$o.cancelReason', style: const TextStyle(color: Colors.grey)),
                      ),
                    const Divider(),
                    _row('物品编号', '#${o.itemId}'),
                    _row('租期', '${o.startDate} ~ ${o.endDate}'),
                    _row('天数', '${o.days} 天'),
                    _row('租金', '¥${fmtMoney(o.rentAmount)}'),
                    _row('押金', '¥${fmtMoney(o.deposit)}'),
                    _row('角色', _isOwner ? '我是出租方' : (_isRenter ? '我是租客' : '')),
                    ..._counterpartRows(o),
                    if (o.payTradeNo.isNotEmpty) _row('支付流水', o.payTradeNo),
                    _row('下单时间', fmtDate(o.createdAt)),
                  ]),
                ),
      bottomNavigationBar: o == null ? null : _actions(o),
    );
  }

  Widget _row(String k, String v) => Padding(
        padding: const EdgeInsets.symmetric(vertical: 6),
        child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(k, style: TextStyle(color: Colors.grey.shade600)),
          const SizedBox(width: 16),
          Expanded(child: Text(v, textAlign: TextAlign.right)),
        ]),
      );

  // 详情富化:显示对方(我是房东→租客,我是租客→房东)公开信息;缺失隐藏。
  List<Widget> _counterpartRows(Order o) {
    final UserProfile? p = _isOwner
        ? o.renter
        : (_isRenter ? o.owner : null);
    if (p == null || p.id == 0) return const [];
    return [
      Padding(
        padding: const EdgeInsets.only(top: 4),
        child: Row(children: [
          Container(
            width: 28,
            height: 28,
            decoration: BoxDecoration(
              color: Colors.grey.shade200,
              shape: BoxShape.circle,
            ),
            clipBehavior: Clip.antiAlias,
            child: p.avatar.isEmpty
                ? const Icon(Icons.person, size: 15, color: Colors.grey)
                : Image.network(p.avatar,
                    fit: BoxFit.cover,
                    errorBuilder: (_, _, _) =>
                        const Icon(Icons.person, size: 15, color: Colors.grey)),
          ),
          const SizedBox(width: 8),
          Text(p.nickname, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600)),
          const SizedBox(width: 8),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
            decoration: BoxDecoration(
              color: kGreen.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Text('信用分 ${p.creditScore}',
                style: const TextStyle(fontSize: 11, color: kGreen)),
          ),
          const Spacer(),
          Text(_isOwner ? '租客' : '房东', style: TextStyle(fontSize: 12, color: Colors.grey.shade500)),
        ]),
      ),
    ];
  }

  Widget? _actions(Order o) {
    final List<Widget> btns = [];
    final bool renterAction = _isOwner ? false : true; // 租客侧动作
    final bool ownerAction = _isRenter ? false : true; // 出租方侧动作
    if (o.status == 0 && renterAction) {
      // 待支付:租客可支付/取消
      btns.add(FilledButton.icon(
          style: FilledButton.styleFrom(backgroundColor: const Color(0xFFFB8C00)),
          icon: const Icon(Icons.payment),
          label: const Text('去支付'),
          onPressed: _acting ? null : () async {
            await Navigator.of(context).pushReplacement(MaterialPageRoute(
                builder: (_) => PayPage(orderNo: o.orderNo, orderId: o.id)));
          }));
      btns.add(OutlinedButton(
          onPressed: _acting ? null : () => _confirmDialog('确定取消该订单?租金将原路退回', () => api.cancel(o.id)),
          child: const Text('取消订单')));
    }
    if (o.status == 1 && renterAction) {
      // 待取:租客取货(冻结押金)
      btns.add(FilledButton.icon(
          style: FilledButton.styleFrom(backgroundColor: kGreen),
          icon: const Icon(Icons.download_done),
          label: const Text('确认取货(冻结押金)'),
          onPressed: _acting ? null : () => _confirmDialog('确认已取到物品?将冻结押金 ¥${fmtMoney(o.deposit)}', () => api.pickup(o.id))));
    }
    if (o.status == 2 && renterAction) {
      // 租赁中:租客申请归还
      btns.add(FilledButton.icon(
          icon: const Icon(Icons.upload),
          label: const Text('申请归还'),
          onPressed: _acting ? null : () => _confirmDialog('确认归还物品,等待出租方验收?', () => api.returnRequest(o.id))));
    }
    if (o.status == 3 && ownerAction) {
      // 待归还:出租方确认归还 / 违约
      btns.add(FilledButton.icon(
          style: FilledButton.styleFrom(backgroundColor: kGreen),
          icon: const Icon(Icons.check_circle),
          label: const Text('确认归还(结算)'),
          onPressed: _acting ? null : () => _confirmDialog('确认验收通过,解除押金冻结并结算租金?', () => api.returnConfirm(o.id))));
      btns.add(OutlinedButton.icon(
          style: OutlinedButton.styleFrom(foregroundColor: Colors.red),
          icon: const Icon(Icons.gavel),
          label: const Text('判定违约(扣押金)'),
          onPressed: _acting ? null : () => _confirmDialog('物品损坏/逾期将扣押金 ¥${fmtMoney(o.deposit)},确定?', () => api.breach(o.id))));
    }
    if (btns.isEmpty) return null;
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 6, 16, 12),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          for (var i = 0; i < btns.length; i++) ...[
            SizedBox(
                width: double.infinity,
                height: 46,
                child: btns[i]),
            if (i < btns.length - 1) const SizedBox(height: 8),
          ]
        ]),
      ),
    );
  }
}
