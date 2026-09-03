import 'user.dart';

/// Order 模型,字段对齐 server/models/order.go。
class Order {
  final int id;
  final String orderNo;
  final int itemId;
  final int renterId;
  final int ownerId;
  final UserProfile? owner; // 详情接口富化:房东公开信息(列表不含)
  final UserProfile? renter; // 详情接口富化:租客公开信息(列表不含)
  final String startDate;
  final String endDate;
  final int days;
  final double rentAmount; // 元
  final double deposit; // 元
  final int status;
  final String payTradeNo;
  final String cancelReason;
  final String createdAt;

  Order({
    required this.id,
    required this.orderNo,
    required this.itemId,
    required this.renterId,
    required this.ownerId,
    this.owner,
    this.renter,
    required this.startDate,
    required this.endDate,
    required this.days,
    required this.rentAmount,
    required this.deposit,
    required this.status,
    required this.payTradeNo,
    required this.cancelReason,
    required this.createdAt,
  });

  factory Order.fromJson(Map<String, dynamic> j) => Order(
        id: j['id'] as int? ?? 0,
        orderNo: j['order_no'] as String? ?? '',
        itemId: j['item_id'] as int? ?? 0,
        renterId: j['renter_id'] as int? ?? 0,
        ownerId: j['owner_id'] as int? ?? 0,
        owner: j['owner'] is Map<String, dynamic>
            ? UserProfile.fromJson(j['owner'] as Map<String, dynamic>)
            : null,
        renter: j['renter'] is Map<String, dynamic>
            ? UserProfile.fromJson(j['renter'] as Map<String, dynamic>)
            : null,
        startDate: j['start_date'] as String? ?? '',
        endDate: j['end_date'] as String? ?? '',
        days: j['days'] as int? ?? 0,
        rentAmount: (j['rent_amount'] as num? ?? 0).toDouble(),
        deposit: (j['deposit'] as num? ?? 0).toDouble(),
        status: j['status'] as int? ?? 0,
        payTradeNo: j['pay_trade_no'] as String? ?? '',
        cancelReason: j['cancel_reason'] as String? ?? '',
        createdAt: j['created_at'] as String? ?? '',
      );

  String get statusText => switch (status) {
        0 => '待支付',
        1 => '待取货',
        2 => '租赁中',
        3 => '待归还',
        4 => '已归还',
        5 => '已取消',
        6 => '违约',
        _ => '未知($status)',
      };
}

List<Order> parseOrderList(Map<String, dynamic> d) =>
    (d['orders'] as List<dynamic>? ?? [])
        .whereType<Map<String, dynamic>>()
        .map(Order.fromJson)
        .toList();
