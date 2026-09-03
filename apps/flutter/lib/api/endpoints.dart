import 'package:image_picker/image_picker.dart';

import '../core/api_client.dart';
import '../core/storage.dart';

/// 端点封装。路径/参数与 server/controllers 实际读取一致:
/// 列表一律 page/page_size;搜索词为 q;/pay/unifiedorder 传 order_no。
class Api {
  final ApiClient c = ApiClient.client;

  // ---- 认证 ----
  Future<void> sms(String phone) =>
      c.post('/auth/sms', body: {'phone': phone}, auth: false);
  Future<Map<String, dynamic>> login(String phone, String code) async =>
      (await c.post('/auth/login', body: {'phone': phone, 'code': code}, auth: false))
          as Map<String, dynamic>;
  /// 登出:携带 refresh_token 让服务端只撤销本端会话(服务端解析失败会降级全端登出)。
  Future<void> logout() async =>
      c.post('/auth/logout', body: {'refresh_token': ?await TokenStorage.refresh});

  // ---- 用户 ----
  Future<Map<String, dynamic>> profile() async =>
      (await c.get('/user/profile')) as Map<String, dynamic>;
  Future<void> updateProfile({String? nickname, String? avatar}) => c.put(
      '/user/profile',
      body: {'nickname': ?nickname, 'avatar': ?avatar});
  /// 头像上传:multipart 字段 file;服务端校验并直接落库,返回新 avatar URL。
  Future<String> uploadAvatar(XFile f) async {
    final d = await c.postMultipart('/user/avatar',
        bytes: await f.readAsBytes(), filename: f.name);
    if (d is Map && d['avatar'] is String) return d['avatar'] as String;
    throw ApiError(0, '上传失败:响应缺少头像地址');
  }
  /// 物品图上传:multipart 字段 files(单张请求),成功返回 URL 数组。
  Future<List<String>> uploadItemImage(XFile f) async {
    final d = await c.postMultipart('/items/upload',
        bytes: await f.readAsBytes(), filename: f.name, field: 'files');
    if (d is Map && d['urls'] is List) {
      return [for (final e in d['urls'] as List) e.toString()];
    }
    throw ApiError(0, '上传失败:响应缺少图片地址');
  }

  // ---- 类目 ----
  /// data 为裸数组。
  Future<List<dynamic>> categories() async =>
      (await c.get('/categories', auth: false)) as List<dynamic>;

  // ---- 物品 ----
  // GET /items 不支持 city(server List 仅 category_id);城市过滤走 searchItems。
  // ownerId 非空 = 「我的物品」视图:需 JWT 本人,含下架;空 = 公开仅上架。
  Future<Map<String, dynamic>> items(
          {int page = 1, int pageSize = 20, int? categoryId, int? ownerId}) async =>
      (await c.get('/items',
          query: {
            'page': page,
            'page_size': pageSize,
            'category_id': ?categoryId,
            'owner_id': ?ownerId,
          },
          auth: ownerId != null)) as Map<String, dynamic>;
  Future<Map<String, dynamic>> searchItems({
    String? q,
    int? categoryId,
    double? minPrice,
    double? maxPrice,
    String? orderBy,
    String? city,
    double? lat,
    double? lng,
    double? radiusKm,
    int page = 1,
    int pageSize = 20,
  }) async =>
      (await c.get('/items/search',
          query: {
            if (q != null && q.isNotEmpty) 'q': q,
            'category_id': ?categoryId,
            'min_price': ?minPrice,
            'max_price': ?maxPrice,
            if (orderBy != null && orderBy.isNotEmpty) 'order_by': orderBy,
            if (city != null && city.isNotEmpty) 'city': city,
            'lat': ?lat,
            'lng': ?lng,
            'radius_km': ?radiusKm,
            'page': page,
            'page_size': pageSize,
          },
          auth: false)) as Map<String, dynamic>;
  Future<Map<String, dynamic>> item(int id) async =>
      (await c.get('/items/$id', auth: false)) as Map<String, dynamic>;
  Future<void> createItem(Map<String, dynamic> d) => c.post('/items', body: d);
  Future<void> offshelfItem(int id) => c.post('/items/$id/offshelf');

  // ---- 订单 ----
  Future<Map<String, dynamic>> createOrder(
          {required String itemId,
          required String startDate,
          required String endDate}) async =>
      (await c.post('/orders',
          body: {'item_id': itemId, 'start_date': startDate, 'end_date': endDate}))
          as Map<String, dynamic>;
  Future<Map<String, dynamic>> orders({int page = 1, int pageSize = 20, int? status}) async =>
      (await c.get('/orders',
          query: {
            'page': page,
            'page_size': pageSize,
            'status': ?status,
          })) as Map<String, dynamic>;
  Future<Map<String, dynamic>> order(int id) async =>
      (await c.get('/orders/$id')) as Map<String, dynamic>;
  Future<void> pickup(int id) => c.post('/orders/$id/pickup');
  Future<void> returnRequest(int id) => c.post('/orders/$id/return_request');
  Future<void> returnConfirm(int id) => c.post('/orders/$id/return_confirm');
  Future<void> breach(int id) => c.post('/orders/$id/breach');
  Future<void> cancel(int id) => c.post('/orders/$id/cancel');

  // ---- 支付 ----
  Future<Map<String, dynamic>> unifiedOrder(String orderNo) async =>
      (await c.post('/pay/unifiedorder',
          body: {'order_no': orderNo, 'channel': 'native'})) as Map<String, dynamic>;

  // ---- 消息 ----
  Future<Map<String, dynamic>> messages({bool unreadOnly = false, int page = 1, int pageSize = 20}) async =>
      (await c.get('/messages',
          query: {
            if (unreadOnly) 'unread': 1,
            'page': page,
            'page_size': pageSize,
          })) as Map<String, dynamic>;
  Future<void> markRead(int id) => c.post('/messages/$id/read');
}

Api api = Api();
