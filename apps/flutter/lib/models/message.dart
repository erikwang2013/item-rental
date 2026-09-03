/// 站内消息,字段对齐 server/models/message.go。
class Msg {
  final int id;
  final String type;
  final String title;
  final String content;
  bool read; // 标记已读后本地置 true
  final String createdAt;

  Msg({
    required this.id,
    required this.type,
    required this.title,
    required this.content,
    required this.read,
    required this.createdAt,
  });

  factory Msg.fromJson(Map<String, dynamic> j) => Msg(
        id: j['id'] as int? ?? 0,
        type: j['type'] as String? ?? '',
        title: j['title'] as String? ?? '',
        content: j['content'] as String? ?? '',
        read: j['read'] as bool? ?? false,
        createdAt: j['created_at'] as String? ?? '',
      );

  String get typeText => switch (type) {
        'payment_success' => '支付成功',
        'payment_refunded' => '退款到账',
        'return_confirmed' => '归还确认',
        'breach' => '违约通知',
        'order_cancelled' => '订单取消',
        _ => type,
      };
}

/// 解析 {messages,total,page,unread}。
class MessagePage {
  final List<Msg> messages;
  final int total;
  final int unread;
  MessagePage(this.messages, {required this.total, required this.unread});
}
