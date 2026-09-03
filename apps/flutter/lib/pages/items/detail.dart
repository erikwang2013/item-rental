import 'package:flutter/material.dart';

import '../../api/endpoints.dart';
import '../../models/item.dart';
import '../../widgets/commons.dart';
import '../../widgets/routes.dart';
import '../order/confirm.dart';

/// 物品详情。
class ItemDetailPage extends StatefulWidget {
  final int itemId;
  const ItemDetailPage({super.key, required this.itemId});

  @override
  State<ItemDetailPage> createState() => _ItemDetailPageState();
}

class _ItemDetailPageState extends State<ItemDetailPage> {
  Item? _item;
  bool _loading = true;
  bool _gone = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final d = await api.item(widget.itemId);
      if (mounted) setState(() => _item = Item.fromJson(d));
    } catch (e) {
      toast(e);
    } finally {
      if (mounted) setState(() { _loading = false; _gone = _item == null; });
    }
  }

  Future<void> _rent() async {
    final item = _item;
    if (item == null) return;
    if (!await ensureLogin(context)) return;
    if (!mounted) return;
    await go(context, OrderConfirmPage(item: item));
  }

  @override
  Widget build(BuildContext context) {
    final item = _item;
    return Scaffold(
      appBar: AppBar(title: Text(item?.title ?? '物品详情')),
      body: _loading
          ? const StatusBox(loading: true)
          : _gone
              ? const StatusBox(emptyText: '物品不存在或已下架')
              : RefreshIndicator(
                  onRefresh: _load,
                  child: ListView(children: [
                    _Gallery(images: item!.imageUrls),
                    Padding(
                      padding: const EdgeInsets.all(14),
                      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                        Text(item.title, style: const TextStyle(fontSize: 20, fontWeight: FontWeight.bold)),
                        const SizedBox(height: 6),
                        Row(children: [
                          Text('¥${item.dailyPrice.toStringAsFixed(2)}/天',
                              style: const TextStyle(color: Color(0xFFE53935), fontSize: 20, fontWeight: FontWeight.bold)),
                          const Spacer(),
                          Text('押金 ¥${item.deposit.toStringAsFixed(2)}',
                              style: TextStyle(color: Colors.grey.shade600)),
                        ]),
                        const SizedBox(height: 6),
                        Text('库存 ${item.stock} 件', style: TextStyle(fontSize: 12, color: Colors.grey.shade600)),
                        if (item.city.isNotEmpty)
                          Padding(
                            padding: const EdgeInsets.only(top: 6),
                            child: Row(children: [
                              const Icon(Icons.location_on_outlined, size: 16, color: Colors.grey),
                              const SizedBox(width: 2),
                              Text(item.city, style: TextStyle(color: Colors.grey.shade700)),
                            ]),
                          ),
                        if (item.owner != null && item.owner!.id != 0)
                          Padding(
                            padding: const EdgeInsets.only(top: 10),
                            child: Row(children: [
                              Container(
                                width: 32,
                                height: 32,
                                decoration: BoxDecoration(
                                  color: Colors.grey.shade200,
                                  shape: BoxShape.circle,
                                ),
                                clipBehavior: Clip.antiAlias,
                                child: item.owner!.avatar.isEmpty
                                    ? const Icon(Icons.person, size: 18, color: Colors.grey)
                                    : Image.network(item.owner!.avatar,
                                        fit: BoxFit.cover,
                                        errorBuilder: (_, _, _) => const Icon(
                                            Icons.person, size: 18, color: Colors.grey)),
                              ),
                              const SizedBox(width: 8),
                              Text(item.owner!.nickname,
                                  style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w600)),
                              const SizedBox(width: 8),
                              Container(
                                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
                                decoration: BoxDecoration(
                                  color: kGreen.withValues(alpha: 0.1),
                                  borderRadius: BorderRadius.circular(8),
                                ),
                                child: Text('信用分 ${item.owner!.creditScore}',
                                    style: const TextStyle(fontSize: 11, color: kGreen)),
                              ),
                              const Spacer(),
                              Text('房东', style: TextStyle(fontSize: 12, color: Colors.grey.shade500)),
                            ]),
                          ),
                        const Divider(height: 28),
                        Text('物品描述', style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600)),
                        const SizedBox(height: 8),
                        Text(item.desc.isEmpty ? '暂无描述' : item.desc,
                            style: const TextStyle(height: 1.5)),
                        const SizedBox(height: 90),
                      ]),
                    ),
                  ]),
                ),
      bottomNavigationBar: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 12),
          child: FilledButton(
            style: FilledButton.styleFrom(
                backgroundColor: item == null || !item.onShelf ? Colors.grey : kGreen,
                padding: const EdgeInsets.symmetric(vertical: 14)),
            onPressed: item != null && item.onShelf ? _rent : null,
            child: Text(item == null || item.onShelf ? '立即租借' : '已下架'),
          ),
        ),
      ),
    );
  }
}

class _Gallery extends StatefulWidget {
  final List<String> images;
  const _Gallery({required this.images});

  @override
  State<_Gallery> createState() => _GalleryState();
}

class _GalleryState extends State<_Gallery> {
  int _idx = 0;

  @override
  Widget build(BuildContext context) {
    final imgs = widget.images.isEmpty ? [''] : widget.images;
    return AspectRatio(
      aspectRatio: 16 / 10,
      child: Stack(children: [
        PageView.builder(
          itemCount: imgs.length,
          onPageChanged: (i) => setState(() => _idx = i),
          itemBuilder: (_, i) => imgs[i].isEmpty
              ? Container(color: Colors.grey.shade200, child: const Center(child: Icon(Icons.inventory_2_outlined, size: 60, color: Colors.grey)))
              : Image.network(imgs[i], fit: BoxFit.cover,
                  errorBuilder: (_, _, _) => Container(color: Colors.grey.shade200, child: const Center(child: Icon(Icons.broken_image_outlined, size: 60, color: Colors.grey)))),
        ),
        if (imgs.length > 1)
          Positioned(
            right: 10,
            bottom: 10,
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
              decoration: BoxDecoration(color: Colors.black54, borderRadius: BorderRadius.circular(10)),
              child: Text('${_idx + 1}/${imgs.length}', style: const TextStyle(color: Colors.white, fontSize: 12)),
            ),
          ),
      ]),
    );
  }
}
