import 'package:flutter/material.dart';

import '../core/format.dart';
import '../models/item.dart';

/// 物品卡片(列表/网格复用)。
class ItemCard extends StatelessWidget {
  final Item item;
  final VoidCallback onTap;
  const ItemCard({super.key, required this.item, required this.onTap});

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      child: Card(
        clipBehavior: Clip.antiAlias,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            AspectRatio(
              aspectRatio: 16 / 10,
              child: item.cover.isEmpty
                  ? Container(
                      color: Colors.grey.shade200,
                      child: const Icon(Icons.inventory_2_outlined,
                          size: 40, color: Colors.grey),
                    )
                  : Image.network(item.cover, fit: BoxFit.cover,
                      errorBuilder: (_, _, _) => Container(
                          color: Colors.grey.shade200,
                          child: const Icon(Icons.broken_image_outlined,
                              color: Colors.grey))),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(8, 6, 8, 8),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(item.title,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(fontWeight: FontWeight.w600)),
                  const SizedBox(height: 2),
                  Text(item.city.isEmpty ? '同城租赁' : item.city,
                      style: TextStyle(fontSize: 12, color: Colors.grey.shade600)),
                  const SizedBox(height: 4),
                  Row(
                    children: [
                      Text('¥${fmtMoney(item.dailyPrice)}/天',
                          style: const TextStyle(
                              color: Color(0xFFE53935),
                              fontWeight: FontWeight.bold)),
                      const Spacer(),
                      Text('押金¥${fmtMoney(item.deposit)}',
                          style:
                              TextStyle(fontSize: 11, color: Colors.grey.shade500)),
                    ],
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
