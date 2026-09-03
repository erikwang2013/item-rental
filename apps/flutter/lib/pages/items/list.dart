import 'package:flutter/material.dart';

import '../../api/endpoints.dart';
import '../../models/item.dart';
import '../../widgets/commons.dart';
import '../../widgets/routes.dart';
import 'detail.dart';

/// 物品列表(类目/城市过滤,分页上拉加载)。
class ItemListPage extends StatefulWidget {
  final int? categoryId;
  final String? city;
  final String? title;
  const ItemListPage({super.key, this.categoryId, this.city, this.title});

  @override
  State<ItemListPage> createState() => _ItemListPageState();
}

class _ItemListPageState extends State<ItemListPage> {
  int? _categoryId;
  String? _city;
  final List<Item> _items = [];
  int _page = 1;
  int _total = 0;
  bool _loading = false;
  bool _ended = false;
  final _scroll = ScrollController();

  @override
  void initState() {
    super.initState();
    _categoryId = widget.categoryId;
    _city = widget.city;
    _scroll.addListener(() {
      if (_scroll.position.pixels > _scroll.position.maxScrollExtent - 200) _loadMore();
    });
    _reload();
  }

  @override
  void dispose() {
    _scroll.dispose();
    super.dispose();
  }

  Future<void> _reload() async {
    setState(() { _page = 1; _ended = false; });
    await _fetch(1, clear: true);
  }

  Future<void> _loadMore() async {
    if (_loading || _ended) return;
    if (_items.length >= _total && _total > 0) {
      setState(() => _ended = true);
      return;
    }
    await _fetch(_page + 1);
  }

  Future<void> _fetch(int page, {bool clear = false}) async {
    setState(() => _loading = true);
    try {
      // GET /items 不支持 city 过滤(仅 category_id),带城市走 /items/search
      final d = (_city != null && _city!.isNotEmpty)
          ? await api.searchItems(categoryId: _categoryId, city: _city, page: page, pageSize: 20)
          : await api.items(page: page, categoryId: _categoryId);
      final list = parseItemList(d);
      setState(() {
        _page = page;
        _total = d['total'] as int? ?? 0;
        if (clear) _items.clear();
        _items.addAll(list);
        _ended = _items.length >= _total;
      });
    } catch (e) {
      toast(e);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.title ?? (widget.city ?? '全部物品')),
        actions: [
          if (widget.city == null)
            TextButton.icon(
              onPressed: () => _pickCity(context),
              icon: const Icon(Icons.location_on_outlined, size: 18),
              label: Text(_city ?? '城市'),
            ),
        ],
      ),
      body: Column(children: [
        Expanded(
          child: RefreshIndicator(
            onRefresh: _reload,
            child: _items.isEmpty
                ? ListView(children: const [SizedBox(height: 200), StatusBox(emptyText: '没有相关物品')])
                : ListView.builder(
                    controller: _scroll,
                    padding: const EdgeInsets.only(bottom: 12),
                    itemCount: _items.length + 1,
                    itemBuilder: (ctx, i) {
                      if (i == _items.length) {
                        return Padding(
                          padding: const EdgeInsets.all(12),
                          child: Center(
                            child: _ended
                                ? Text('已全部加载', style: TextStyle(fontSize: 12, color: Colors.grey.shade500))
                                : (_loading ? const CircularProgressIndicator() : const SizedBox()),
                          ),
                        );
                      }
                      final it = _items[i];
                      return ItemListTile(item: it);
                    },
                  ),
          ),
        ),
      ]),
    );
  }

  Future<void> _pickCity(BuildContext context) async {
    final ctrl = TextEditingController(text: _city ?? '');
    final city = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('选择城市'),
        content: TextField(
            controller: ctrl, decoration: const InputDecoration(hintText: '如:上海')),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, ''), child: const Text('全部城市')),
          FilledButton(
              onPressed: () => Navigator.pop(ctx, ctrl.text.trim()),
              child: const Text('确定')),
        ],
      ),
    );
    if (city != null && mounted) {
      setState(() => _city = city.isEmpty ? null : city);
      _reload();
    }
  }
}

/// 单行物品(列表流式布局)。
class ItemListTile extends StatelessWidget {
  final Item item;
  const ItemListTile({super.key, required this.item});

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: () => go(context, ItemDetailPage(itemId: item.id)),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        child: Row(children: [
          ClipRRect(
            borderRadius: BorderRadius.circular(8),
            child: SizedBox(
              width: 110,
              height: 82,
              child: item.cover.isEmpty
                  ? Container(color: Colors.grey.shade200, child: const Icon(Icons.inventory_2_outlined, color: Colors.grey))
                  : Image.network(item.cover, fit: BoxFit.cover,
                      errorBuilder: (_, _, _) => Container(color: Colors.grey.shade200, child: const Icon(Icons.broken_image_outlined, color: Colors.grey))),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text(item.title, maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 15)),
              const SizedBox(height: 4),
              Text(item.desc, maxLines: 1, overflow: TextOverflow.ellipsis, style: TextStyle(fontSize: 12, color: Colors.grey.shade600)),
              const SizedBox(height: 4),
              Row(children: [
                Text('¥${item.dailyPrice.toStringAsFixed(2)}/天', style: const TextStyle(color: Color(0xFFE53935), fontWeight: FontWeight.bold)),
                const SizedBox(width: 8),
                Text(item.city, style: TextStyle(fontSize: 12, color: Colors.grey.shade500)),
              ]),
            ]),
          ),
        ]),
      ),
    );
  }
}
