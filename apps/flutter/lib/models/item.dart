import 'dart:convert';

import 'user.dart';

/// Item 模型,json 字段对齐 server/models/item.go。
class Item {
  final int id;
  final int ownerId;
  final UserProfile? owner; // 详情接口富化:房东公开信息(列表不含)
  final int categoryId;
  final String title;
  final String desc;
  final String images; // 逗号分隔 URL
  final double dailyPrice;
  final double deposit;
  final int stock;
  final int status; // 1上架 0下架
  final String city;
  final double lat;
  final double lng;
  final String createdAt;

  Item({
    required this.id,
    required this.ownerId,
    this.owner,
    required this.categoryId,
    required this.title,
    required this.desc,
    required this.images,
    required this.dailyPrice,
    required this.deposit,
    required this.stock,
    required this.status,
    required this.city,
    required this.lat,
    required this.lng,
    required this.createdAt,
  });

  factory Item.fromJson(Map<String, dynamic> j) => Item(
        id: j['id'] as int? ?? 0,
        ownerId: j['owner_id'] as int? ?? 0,
        owner: j['owner'] is Map<String, dynamic>
            ? UserProfile.fromJson(j['owner'] as Map<String, dynamic>)
            : null,
        categoryId: j['category_id'] as int? ?? 0,
        title: j['title'] as String? ?? '',
        desc: j['desc'] as String? ?? '',
        images: j['images'] as String? ?? '',
        dailyPrice: (j['daily_price'] as num? ?? 0).toDouble(),
        deposit: (j['deposit'] as num? ?? 0).toDouble(),
        stock: j['stock'] as int? ?? 0,
        status: j['status'] as int? ?? 0,
        city: j['city'] as String? ?? '',
        lat: (j['lat'] as num? ?? 0).toDouble(),
        lng: (j['lng'] as num? ?? 0).toDouble(),
        createdAt: j['created_at'] as String? ?? '',
      );

  bool get onShelf => status == 1;
  // server 契约:images 为 JSON 数组串(≤9);旧数据可能是逗号分隔串,回退兼容。
  List<String> get imageUrls {
    final raw = images.trim();
    if (raw.isEmpty) return const [];
    try {
      final d = jsonDecode(raw);
      if (d is List) {
        return [for (final e in d) e.toString().trim()]
            .where((s) => s.isNotEmpty)
            .toList();
      }
    } catch (_) {
      // 非 JSON:旧逗号分隔格式
    }
    return raw.split(',').where((s) => s.isNotEmpty).toList();
  }
  String get cover => imageUrls.isEmpty ? '' : imageUrls.first;
}

/// 解析 {items,total,page} 信封 data。
List<Item> parseItemList(Map<String, dynamic> d) =>
    (d['items'] as List<dynamic>? ?? [])
        .whereType<Map<String, dynamic>>()
        .map(Item.fromJson)
        .toList();
