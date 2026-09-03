import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../api/endpoints.dart';
import '../../models/category.dart';
import '../../models/item.dart';
import '../../stores/message_store.dart';
import '../../widgets/commons.dart';
import '../../widgets/item_card.dart';
import '../../widgets/routes.dart';
import '../items/detail.dart';
import '../items/list.dart';
import '../items/search.dart';
import '../messages/index.dart';

/// 首页:搜索 + 类目 + 推荐列表 + 消息入口(未读角标)。
class HomePage extends StatefulWidget {
  const HomePage({super.key});

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  final List<Category> _cats = [];
  final List<Item> _items = [];
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
    context.read<MessageStore>().refresh();
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    try {
      final results = await Future.wait([
        api.categories(),
        api.items(pageSize: 10),
      ]);
      _cats
        ..clear()
        ..addAll((results[0] as List).whereType<Map<String, dynamic>>().map(Category.fromJson));
      _items
        ..clear()
        ..addAll(parseItemList(results[1] as Map<String, dynamic>));
    } catch (e) {
      toast(e);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final ms = context.watch<MessageStore>();
    return Scaffold(
      appBar: AppBar(
        backgroundColor: kGreen,
        foregroundColor: Colors.white,
        title: const Text('闲租'),
        actions: [
          IconButton(
            icon: Badge(
              isLabelVisible: ms.unread > 0,
              label: Text('${ms.unread > 99 ? '99+' : ms.unread}'),
              child: const Icon(Icons.notifications_outlined),
            ),
            onPressed: () async {
              await go(context, const MessagesPage());
              ms.refresh();
            },
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _load,
        child: ListView(
          padding: const EdgeInsets.all(12),
          children: [
            // 搜索入口
            InkWell(
              borderRadius: BorderRadius.circular(24),
              onTap: () => go(context, const SearchPage()),
              child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
                decoration: BoxDecoration(
                    color: Colors.grey.shade200, borderRadius: BorderRadius.circular(24)),
                child: Row(children: [
                  Icon(Icons.search, color: Colors.grey.shade600),
                  const SizedBox(width: 8),
                  Text('搜索物品 / 城市', style: TextStyle(color: Colors.grey.shade600)),
                ]),
              ),
            ),
            const SizedBox(height: 14),
            // 类目横滚
            SizedBox(
              height: 76,
              child: ListView.separated(
                scrollDirection: Axis.horizontal,
                itemCount: _cats.length + 1,
                separatorBuilder: (_, _) => const SizedBox(width: 10),
                itemBuilder: (ctx, i) {
                  if (i == 0) {
                    return _CatChip(name: '全部', icon: Icons.apps, onTap: () => go(ctx, ItemListPage(categoryId: null)));
                  }
                  final c = _cats[i - 1];
                  return _CatChip(
                    name: c.name,
                    icon: Icons.category_outlined,
                    onTap: () => go(ctx, ItemListPage(categoryId: c.id, title: c.name)),
                  );
                },
              ),
            ),
            const SizedBox(height: 6),
            Row(
              children: [
                const Text('推荐', style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
                const Spacer(),
                TextButton(
                    onPressed: () => go(context, const ItemListPage()),
                    child: const Text('更多 >')),
              ],
            ),
            if (_loading)
              const Padding(padding: EdgeInsets.all(40), child: Center(child: CircularProgressIndicator()))
            else if (_items.isEmpty)
              const Padding(padding: EdgeInsets.all(40), child: Center(child: Text('暂无推荐物品', style: TextStyle(color: Colors.grey))))
            else
              GridView.count(
                shrinkWrap: true,
                physics: const NeverScrollableScrollPhysics(),
                crossAxisCount: 2,
                mainAxisSpacing: 10,
                crossAxisSpacing: 10,
                childAspectRatio: .78,
                children: _items.map((it) => ItemCard(item: it, onTap: () => go(context, ItemDetailPage(itemId: it.id)))).toList(),
              ),
            const SizedBox(height: 20),
          ],
        ),
      ),
    );
  }
}

class _CatChip extends StatelessWidget {
  final String name;
  final IconData icon;
  final VoidCallback onTap;
  const _CatChip({required this.name, required this.icon, required this.onTap});

  @override
  Widget build(BuildContext context) => InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Container(
          width: 64,
          decoration: BoxDecoration(
              color: Colors.green.shade50, borderRadius: BorderRadius.circular(12)),
          child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
            Icon(icon, color: kGreen),
            const SizedBox(height: 4),
            Text(name, style: const TextStyle(fontSize: 12)),
          ]),
        ),
      );
}
