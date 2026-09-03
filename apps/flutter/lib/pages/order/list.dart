import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/endpoints.dart';
import '../../models/order.dart';
import '../../stores/user_store.dart';
import '../../widgets/commons.dart';
import '../../widgets/order_card.dart';
import '../../widgets/routes.dart';
import '../auth/login.dart';
import 'detail.dart';

const _tabs = [
  (null, '全部'),
  (0, '待支付'),
  (1, '待取货'),
  (2, '租赁中'),
  (3, '待归还'),
  (4, '已归还'),
  (5, '已取消'),
  (6, '违约'),
];

/// 订单列表(7 态 tab)。未登录时显示登录引导。
class OrderListPage extends StatefulWidget {
  const OrderListPage({super.key});

  @override
  State<OrderListPage> createState() => _OrderListPageState();
}

class _OrderListPageState extends State<OrderListPage>
    with SingleTickerProviderStateMixin {
  late final _tabsCtl = TabController(length: 8, vsync: this);
  final Map<int, _TabState> _states = {for (var i = 0; i < 8; i++) i: _TabState()};
  bool _needLogin = false;

  @override
  void initState() {
    super.initState();
    _needLogin = !context.read<UserStore>().logged;
  }

  @override
  void dispose() {
    _tabsCtl.dispose();
    super.dispose();
  }

  Future<void> _login() async {
    final ok = await go<bool>(context, const LoginPage());
    if (ok == true && mounted) {
      setState(() => _needLogin = false);
      context.read<UserStore>().loadProfile();
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_needLogin) {
      return Scaffold(
        appBar: AppBar(title: const Text('我的订单')),
        body: Center(
          child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
            const Icon(Icons.lock_outline, size: 56, color: Colors.grey),
            const SizedBox(height: 10),
            const Text('登录后查看订单', style: TextStyle(color: Colors.grey)),
            const SizedBox(height: 16),
            FilledButton(
                style: FilledButton.styleFrom(backgroundColor: kGreen),
                onPressed: _login,
                child: const Text('去登录')),
          ]),
        ),
      );
    }
    return Scaffold(
      appBar: AppBar(
        title: const Text('我的订单'),
        bottom: TabBar(
          controller: _tabsCtl,
          isScrollable: true,
          tabAlignment: TabAlignment.start,
          tabs: [for (final (_, n) in _tabs) Tab(text: n)],
        ),
      ),
      body: TabBarView(
        controller: _tabsCtl,
        children: [for (var i = 0; i < _tabs.length; i++) _OrderTab(index: i, status: _tabs[i].$1, state: _states[i]!)],
      ),
    );
  }
}

class _TabState {
  final List<Order> orders = [];
  bool loading = false;
  bool loadedOnce = false;
  int page = 1;
  int total = 0;
  bool ended = false;
}

class _OrderTab extends StatefulWidget {
  final int index;
  final int? status;
  final _TabState state;
  const _OrderTab({required this.index, required this.status, required this.state});

  @override
  State<_OrderTab> createState() => _OrderTabState();
}

class _OrderTabState extends State<_OrderTab> {
  final _scroll = ScrollController();

  @override
  void initState() {
    super.initState();
    _scroll.addListener(() {
      if (_scroll.position.pixels > _scroll.position.maxScrollExtent - 150) _more();
    });
    _load(clear: true);
  }

  @override
  void dispose() {
    _scroll.dispose();
    super.dispose();
  }

  Future<void> _load({bool clear = false}) async {
    final st = widget.state;
    if (st.loading) return;
    st.loading = true;
    if (mounted) setState(() {});
    try {
      final d = await api.orders(page: clear ? 1 : st.page + 1, status: widget.status);
      final list = parseOrderList(d);
      st.total = d['total'] as int? ?? 0;
      if (clear) st.orders.clear();
      st.orders.addAll(list);
      st.page = clear ? 1 : st.page + 1;
      st.ended = st.orders.length >= st.total;
      st.loadedOnce = true;
    } catch (e) {
      toast(e);
    } finally {
      st.loading = false;
      if (mounted) setState(() {});
    }
  }

  void _more() {
    if (!widget.state.ended && widget.state.orders.isNotEmpty) _load();
  }

  @override
  Widget build(BuildContext context) {
    final st = widget.state;
    if (!st.loadedOnce) return const StatusBox(loading: true);
    if (st.orders.isEmpty) {
      return RefreshIndicator(
          onRefresh: () => _load(clear: true),
          child: ListView(children: const [SizedBox(height: 160), StatusBox(emptyText: '暂无订单')]));
    }
    return RefreshIndicator(
      onRefresh: () => _load(clear: true),
      child: ListView.builder(
        controller: _scroll,
        padding: const EdgeInsets.symmetric(vertical: 6),
        itemCount: st.orders.length + 1,
        itemBuilder: (ctx, i) {
          if (i == st.orders.length) {
            return Center(
                child: Padding(
              padding: const EdgeInsets.all(10),
              child: st.ended
                  ? Text('已全部加载', style: TextStyle(fontSize: 12, color: Colors.grey.shade500))
                  : const SizedBox(),
            ));
          }
          final o = st.orders[i];
          return OrderCard(order: o, onTap: () => _open(ctx, o.id));
        },
      ),
    );
  }

  Future<void> _open(BuildContext ctx, int id) async {
    await go(ctx, OrderDetailPage(orderId: id));
    _load(clear: true);
  }
}
