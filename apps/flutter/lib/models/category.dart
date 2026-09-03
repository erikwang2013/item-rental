/// 类目,字段对齐 server/models/category.go。
class Category {
  final int id;
  final String name;
  final String icon;
  Category({required this.id, required this.name, required this.icon});

  factory Category.fromJson(Map<String, dynamic> j) => Category(
        id: j['id'] as int? ?? 0,
        name: j['name'] as String? ?? '',
        icon: j['icon'] as String? ?? '',
      );
}
