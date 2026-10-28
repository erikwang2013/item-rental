/// 用户资料,字段对齐 server/models/user.go(profile 接口不含 phone)。
class UserProfile {
  final int id;
  final String nickname;
  final String avatar;
  final String realName;
  final String createdAt;

  UserProfile({
    required this.id,
    required this.nickname,
    required this.avatar,
    required this.realName,
    required this.createdAt,
  });

  factory UserProfile.fromJson(Map<String, dynamic> j) => UserProfile(
        id: int.tryParse('${j['id']}') ?? 0,
        nickname: j['nickname'] as String? ?? '',
        avatar: j['avatar'] as String? ?? '',
        realName: j['real_name'] as String? ?? '',
        createdAt: j['created_at'] as String? ?? '',
      );
}
