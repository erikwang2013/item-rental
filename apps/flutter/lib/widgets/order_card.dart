import 'package:flutter/material.dart';

import '../core/format.dart';
import '../models/order.dart';

/// 订单卡片(列表页复用)。
class OrderCard extends StatelessWidget {
  final Order order;
  final VoidCallback onTap;
  const OrderCard({super.key, required this.order, required this.onTap});

  @override
  Widget build(BuildContext context) {
    final colors = <int, Color>{
      0: Colors.orange,
      1: Colors.blue,
      2: Colors.teal,
      3: Colors.deepPurple,
      4: Colors.green,
      5: Colors.grey,
      6: Colors.red,
    };
    return InkWell(
      onTap: onTap,
      child: Card(
        margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 5),
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Text(order.orderNo.isEmpty ? '#${order.id}' : order.orderNo,
                      style: TextStyle(fontSize: 12, color: Colors.grey.shade600)),
                  const Spacer(),
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                    decoration: BoxDecoration(
                      color: (colors[order.status] ?? Colors.grey).withValues(alpha: .12),
                      borderRadius: BorderRadius.circular(10),
                    ),
                    child: Text(order.statusText,
                        style: TextStyle(
                            fontSize: 12,
                            color: colors[order.status] ?? Colors.grey)),
                  ),
                ],
              ),
              const SizedBox(height: 6),
              Text('${order.startDate} ~ ${order.endDate} 共${order.days}天',
                  style: const TextStyle(fontWeight: FontWeight.w600)),
              const SizedBox(height: 2),
              Text(
                  '租金 ¥${fmtMoney(order.rentAmount)}  押金 ¥${fmtMoney(order.deposit)}',
                  style: TextStyle(fontSize: 13, color: Colors.grey.shade700)),
            ],
          ),
        ),
      ),
    );
  }
}
