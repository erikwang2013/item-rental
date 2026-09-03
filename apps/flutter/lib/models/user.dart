/// 用户资料,字段对齐 server/models/user.go(profile 接口不含 phone)。
class UserProfile {
  final int id;
  final String nickname;
  final String avatar;
  final String realName;
  final int creditScore;
  final double depositBal;
  final String createdAt;

  UserProfile({
    required this.id,
    required this.nickname,
    required this.avatar,
    required this.realName,
    required this.creditScore,
    required this.depositBal,
    required this.createdAt,
  });

  factory UserProfile.fromJson(Map<String, dynamic> j) => UserProfile(
        id: int.tryParse('${j['id']}') ?? 0,
        nickname: j['nickname'] as String? ?? '',
        avatar: j['avatar'] as String? ?? '',
        realName: j['real_name'] as String? ?? '',
        creditScore: j['credit_score'] as int? ?? 0,
        depositBal: (j['deposit_bal'] as num? ?? 0).toDouble(),
        createdAt: j['created_at'] as String? ?? '',
      );
}
