import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

import 'storage.dart';

class ApiError extends Error {
  final int code;
  final String message;
  ApiError(this.code, this.message);
  @override
  String toString() => message.isEmpty ? '请求失败' : message;
}

/// 信封 {code,msg,data}: code==0 返回 data(可为 Map/List/标量);否则抛 ApiError(msg)。
dynamic _unwrap(String raw) {
  dynamic j;
  try {
    j = jsonDecode(raw);
  } catch (_) {
    throw ApiError(500, '响应解析失败');
  }
  if (j is Map && j['code'] is int && j['code'] == 0) return j['data'];
  throw ApiError(
      j is Map && j['code'] is int ? j['code'] as int : 500,
      j is Map && j['msg'] is String && (j['msg'] as String).isNotEmpty
          ? j['msg'] as String
          : '请求失败');
}

class ApiClient {
  ApiClient({this.baseUrl = 'http://127.0.0.1:8080/api/v1'});
  final String baseUrl;
  static final client = ApiClient();

  Future<void>? _refreshing;

  /// 401 时用 refresh_token 换新双 token;并发请求共享单次刷新(单活跃)。
  Future<void> _refresh() async {
    final rt = await TokenStorage.refresh;
    if (rt == null || rt.isEmpty) throw ApiError(401, '登录已失效,请重新登录');
    final res = await http.post(Uri.parse('$baseUrl/auth/refresh'),
        headers: const {'Content-Type': 'application/json'},
        body: jsonEncode({'refresh_token': rt}));
    final d = _unwrap(res.body);
    await TokenStorage.save(
        d is Map && d['access_token'] is String ? d['access_token'] as String : null,
        d is Map && d['refresh_token'] is String ? d['refresh_token'] as String : null);
  }

  Future<dynamic> _send(String method, String path, {Map<String, dynamic>? query, Map<String, dynamic>? body, bool auth = true, bool allowRetry = true}) async {
    final qs = query == null || query.isEmpty
        ? ''
        : '?${query.entries.map((e) => '${e.key}=${Uri.encodeQueryComponent('${e.value}')}').join('&')}';
    final headers = <String, String>{'Content-Type': 'application/json'};
    if (auth) {
      final tk = await TokenStorage.access;
      if (tk != null && tk.isNotEmpty) headers['Authorization'] = 'Bearer $tk';
    }
    final uri = Uri.parse('$baseUrl$path$qs');
    final enc = body == null ? null : jsonEncode(body);
    final http.Response res;
    switch (method) {
      case 'GET':
        res = await http.get(uri, headers: headers);
      case 'PUT':
        res = await http.put(uri, headers: headers, body: enc);
      default:
        res = await http.post(uri, headers: headers, body: enc);
    }
    // 401:刷新后重发原请求(仅一次)。刷新失败/再 401 则透出登录失效。
    if (res.statusCode == 401 || _codeOf(res.body) == 401) {
      if (auth && allowRetry) {
        try {
          final lock = _refreshing ?? _refresh();
          _refreshing = lock;
          await lock;
          return _send(method, path, query: query, body: body, auth: auth, allowRetry: false);
        } on ApiError {
          rethrow;
        } finally {
          _refreshing = null;
        }
      }
      throw ApiError(401, '登录已失效,请重新登录');
    }
    return _unwrap(res.body);
  }

  static int? _codeOf(String raw) {
    try {
      final j = jsonDecode(raw);
      if (j is Map && j['code'] is int) return j['code'] as int;
    } catch (_) {}
    return null;
  }

  Future<dynamic> get(String path, {Map<String, dynamic>? query, bool auth = true}) =>
      _send('GET', path, query: query, auth: auth);

  Future<dynamic> post(String path, {Map<String, dynamic>? body, bool auth = true}) =>
      _send('POST', path, body: body, auth: auth);

  /// multipart 上传(multipart 字段名 file;头像用)。
  /// 401 刷新后重发原请求一次,语义与 _send 一致。
  Future<dynamic> postMultipart(String path,
      {required List<int> bytes,
      required String filename,
      bool auth = true,
      bool allowRetry = true}) async {
    final tk = auth ? await TokenStorage.access : null;
    final req = http.MultipartRequest('POST', Uri.parse('$baseUrl$path'));
    if (tk != null && tk.isNotEmpty) req.headers['Authorization'] = 'Bearer $tk';
    req.files.add(http.MultipartFile.fromBytes('file', bytes, filename: filename));
    final res = await http.Response.fromStream(await req.send());
    if (res.statusCode == 401 || _codeOf(res.body) == 401) {
      if (auth && allowRetry) {
        try {
          final lock = _refreshing ?? _refresh();
          _refreshing = lock;
          await lock;
          return postMultipart(path,
              bytes: bytes, filename: filename, auth: auth, allowRetry: false);
        } on ApiError {
          rethrow;
        } finally {
          _refreshing = null;
        }
      }
      throw ApiError(401, '登录已失效,请重新登录');
    }
    return _unwrap(res.body);
  }

  Future<dynamic> put(String path, {Map<String, dynamic>? body, bool auth = true}) =>
      _send('PUT', path, body: body, auth: auth);
}
