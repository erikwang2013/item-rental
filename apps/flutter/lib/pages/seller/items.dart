import 'package:flutter/material.dart';

import '../../api/endpoints.dart';
import '../../models/category.dart';
import '../../models/item.dart';
import '../../widgets/commons.dart';
import '../../widgets/routes.dart';
import '../items/detail.dart';

/// 我的物品:发布 + 列表 + 下架。
/// ponytail: 后端无「按 owner 列出我发布物品」的端点(uni-app 同样只能列全量),
/// 列表显示的是平台上架物品;对非本人物品点下架会得到 403 提示。加 owner 过滤时需后端支持。
class SellerItemsPage extends StatefulWidget {
  const SellerItemsPage({super.key});

  @override
  State<SellerItemsPage> createState() => _SellerItemsPageState();
}

class _SellerItemsPageState extends State<SellerItemsPage> {
  final List<Item> _items = [];
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final d = await api.items(pageSize: 50);
      if (mounted) {
        setState(() {
          _items
            ..clear()
            ..addAll(parseItemList(d));
        });
      }
    } catch (e) {
      toast(e);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _publish() async {
    if (!await ensureLogin(context)) return;
    if (!mounted) return;
    await showDialog(context: context, builder: (_) => const _PublishDialog());
    _load();
  }

  Future<void> _offshelf(Item it) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        content: Text('确定下架「${it.title}」?'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('取消')),
          FilledButton(
              onPressed: () => Navigator.pop(ctx, true), child: const Text('下架')),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await api.offshelfItem(it.id);
      toast('已下架');
      _load();
    } catch (e) {
      toast(e); // 非本人物品 → 服务端 403 文案
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('我的物品')),
      floatingActionButton: FloatingActionButton.extended(
        backgroundColor: kGreen,
        onPressed: _publish,
        icon: const Icon(Icons.add),
        label: const Text('发布物品'),
      ),
      body: RefreshIndicator(
        onRefresh: _load,
        child: _loading
            ? const StatusBox(loading: true)
            : _items.isEmpty
                ? ListView(children: const [
                    SizedBox(height: 180),
                    StatusBox(emptyText: '还没有物品,点右下角发布')
                  ])
                : ListView.builder(
                    padding: const EdgeInsets.only(bottom: 90),
                    itemCount: _items.length,
                    itemBuilder: (ctx, i) {
                      final it = _items[i];
                      return ListTile(
                        leading: ClipRRect(
                          borderRadius: BorderRadius.circular(6),
                          child: SizedBox(
                            width: 56,
                            height: 56,
                            child: it.cover.isEmpty
                                ? Container(color: Colors.grey.shade200, child: const Icon(Icons.inventory_2_outlined, color: Colors.grey))
                                : Image.network(it.cover, fit: BoxFit.cover,
                                    errorBuilder: (_, _, _) => const Icon(Icons.broken_image_outlined)),
                          ),
                        ),
                        title: Text(it.title, maxLines: 1, overflow: TextOverflow.ellipsis),
                        subtitle: Text(
                            '¥${it.dailyPrice.toStringAsFixed(2)}/天 · 库存${it.stock} · ${it.city}',
                            style: const TextStyle(fontSize: 12)),
                        onTap: () => go(context, ItemDetailPage(itemId: it.id)),
                        trailing: it.onShelf
                            ? TextButton(
                                onPressed: () => _offshelf(it),
                                child: const Text('下架', style: TextStyle(color: Colors.grey)))
                            : Text('已下架', style: TextStyle(color: Colors.grey.shade400, fontSize: 13)),
                      );
                    },
                  ),
      ),
    );
  }
}

class _PublishDialog extends StatefulWidget {
  const _PublishDialog();

  @override
  State<_PublishDialog> createState() => _PublishDialogState();
}

class _PublishDialogState extends State<_PublishDialog> {
  final _title = TextEditingController();
  final _desc = TextEditingController();
  final _price = TextEditingController();
  final _deposit = TextEditingController();
  final _stock = TextEditingController();
  final _city = TextEditingController();
  final _images = TextEditingController();
  List<Category> _cats = [];
  int? _catId;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    api.categories().then((list) {
      if (mounted) {
        setState(() => _cats = list.whereType<Map<String, dynamic>>().map(Category.fromJson).toList());
      }
    }).catchError((_) {});
  }

  @override
  void dispose() {
    for (final c in [_title, _desc, _price, _deposit, _stock, _city, _images]) {
      c.dispose();
    }
    super.dispose();
  }

  Future<void> _submit() async {
    final price = double.tryParse(_price.text.trim());
    final deposit = double.tryParse(_deposit.text.trim());
    final stock = int.tryParse(_stock.text.trim());
    if (_title.text.trim().isEmpty) return toast('请填写标题');
    if (_catId == null) return toast('请选择类目');
    if (price == null || price <= 0) return toast('日租金须为正数');
    if (deposit == null || deposit < 0) return toast('押金不能为负');
    if (stock == null || stock < 1) return toast('库存至少 1 件');
    setState(() => _busy = true);
    try {
      await api.createItem({
        'title': _title.text.trim(),
        'desc': _desc.text.trim(),
        'category_id': _catId,
        'daily_price': price,
        'deposit': deposit,
        'stock': stock,
        'city': _city.text.trim(),
        'images': _images.text.trim(),
      });
      if (!mounted) return;
      Navigator.of(context).pop();
      ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('发布成功'), duration: Duration(seconds: 1)));
    } catch (e) {
      toast(e);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('发布物品'),
      content: SingleChildScrollView(
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          TextField(controller: _title, decoration: const InputDecoration(labelText: '标题 *')),
          TextField(controller: _desc, decoration: const InputDecoration(labelText: '描述'), maxLines: 2),
          DropdownButtonFormField<int?>(
            initialValue: _catId,
            decoration: const InputDecoration(labelText: '类目 *'),
            items: [for (final c in _cats) DropdownMenuItem(value: c.id, child: Text(c.name))],
            onChanged: (v) => setState(() => _catId = v),
          ),
          TextField(
              controller: _price,
              keyboardType: const TextInputType.numberWithOptions(decimal: true),
              decoration: const InputDecoration(labelText: '日租金 ¥ *')),
          TextField(
              controller: _deposit,
              keyboardType: const TextInputType.numberWithOptions(decimal: true),
              decoration: const InputDecoration(labelText: '押金 ¥ *')),
          TextField(
              controller: _stock,
              keyboardType: TextInputType.number,
              decoration: const InputDecoration(labelText: '库存 *')),
          TextField(controller: _city, decoration: const InputDecoration(labelText: '城市')),
          TextField(
              controller: _images,
              decoration: const InputDecoration(labelText: '图片 URL(逗号分隔多个)'),
              maxLines: 2),
        ]),
      ),
      actions: [
        TextButton(onPressed: () => Navigator.pop(context), child: const Text('取消')),
        FilledButton(
            style: FilledButton.styleFrom(backgroundColor: kGreen),
            onPressed: _busy ? null : _submit,
            child: Text(_busy ? '发布中…' : '发布')),
      ],
    );
  }
}
