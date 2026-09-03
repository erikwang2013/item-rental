/// 展示用格式化小工具(金额为元,后端以 float 元返回)。
String fmtMoney(num v) {
  final s = v.toStringAsFixed(2);
  return s.endsWith('.00') ? s.substring(0, s.length - 3) : s;
}

/// 2026-09-03 → 2026-09-03(Go time.Time JSON 为 RFC3339 或 date)。
String fmtDate(String raw) => raw.length >= 10 ? raw.substring(0, 10) : raw;

/// 创建订单需要 yyyy-MM-dd;input 若已是该格式则原样返回。
String toDateStr(DateTime d) =>
    '${d.year.toString().padLeft(4, '0')}-${d.month.toString().padLeft(2, '0')}-${d.day.toString().padLeft(2, '0')}';
