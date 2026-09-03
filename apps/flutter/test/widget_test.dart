import 'package:flutter_test/flutter_test.dart';

import 'package:item_rental/models/item.dart';
import 'package:item_rental/models/message.dart';
import 'package:item_rental/models/order.dart';

void main() {
  test('Item 解析:images 逗号分隔/金额为元', () {
    final it = Item.fromJson({
      'id': 1,
      'owner_id': 2,
      'category_id': 3,
      'title': '单反相机',
      'desc': '9成新',
      'images': 'http://a.jpg,http://b.jpg',
      'daily_price': 30,
      'deposit': 200.5,
      'stock': 1,
      'status': 1,
      'city': '上海',
      'lat': 31.2,
      'lng': 121.4,
      'created_at': '2026-09-01T10:00:00Z',
    });
    expect(it.title, '单反相机');
    expect(it.imageUrls.length, 2);
    expect(it.cover, 'http://a.jpg');
    expect(it.onShelf, isTrue);
    expect(it.dailyPrice, 30.0);
    expect(it.deposit, 200.5);
  });

  test('Order 状态文案', () {
    expect(Order.fromJson({'status': 0, 'id': 1}).statusText, '待支付');
    expect(Order.fromJson({'status': 4, 'id': 1}).statusText, '已归还');
    expect(Order.fromJson({'status': 6, 'id': 1}).statusText, '违约');
  });

  test('Message read 默认 false', () {
    final m = Msg.fromJson({'id': 1, 'type': 'breach', 'title': 't', 'content': 'c'});
    expect(m.read, isFalse);
    m.read = true;
    expect(m.read, isTrue);
  });
}
