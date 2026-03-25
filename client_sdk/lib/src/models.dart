class User {
  const User({
    required this.id,
    required this.email,
    required this.emailVerified,
    required this.status,
  });

  final String id;
  final String email;
  final bool emailVerified;
  final String status;

  factory User.fromJson(Map<String, dynamic> json) => User(
        id: json['id'] as String,
        email: json['email'] as String,
        emailVerified: json['email_verified'] as bool? ?? false,
        status: json['status'] as String? ?? 'unknown',
      );
}

class AuthSession {
  const AuthSession({
    required this.user,
    required this.accessToken,
    required this.refreshToken,
  });

  final User user;
  final String accessToken;
  final String refreshToken;

  factory AuthSession.fromJson(Map<String, dynamic> json) => AuthSession(
        user: User.fromJson(json['user'] as Map<String, dynamic>),
        accessToken: json['access_token'] as String,
        refreshToken: json['refresh_token'] as String,
      );
}

class Plan {
  const Plan({
    required this.id,
    required this.name,
    required this.description,
    required this.pricePoints,
    required this.durationDays,
    required this.maxBoundDevices,
    required this.maxConcurrentSessions,
    required this.supportedProtocols,
  });

  final String id;
  final String name;
  final String description;
  final int pricePoints;
  final int durationDays;
  final int maxBoundDevices;
  final int maxConcurrentSessions;
  final List<String> supportedProtocols;

  factory Plan.fromJson(Map<String, dynamic> json) => Plan(
        id: json['id'] as String,
        name: json['name'] as String,
        description: json['description'] as String? ?? '',
        pricePoints: json['price_points'] as int? ?? 0,
        durationDays: json['duration_days'] as int? ?? 0,
        maxBoundDevices: json['max_bound_devices'] as int? ?? 0,
        maxConcurrentSessions:
            json['max_concurrent_sessions'] as int? ?? 0,
        supportedProtocols: (json['supported_protocols'] as List<dynamic>? ?? [])
            .cast<String>(),
      );
}

class WalletLedgerEntry {
  const WalletLedgerEntry({
    required this.type,
    required this.pointsDelta,
    required this.balance,
    required this.note,
  });

  final String type;
  final int pointsDelta;
  final int balance;
  final String note;

  factory WalletLedgerEntry.fromJson(Map<String, dynamic> json) =>
      WalletLedgerEntry(
        type: json['type'] as String? ?? '',
        pointsDelta: json['points_delta'] as int? ?? 0,
        balance: json['balance'] as int? ?? 0,
        note: json['note'] as String? ?? '',
      );
}

class WalletSnapshot {
  const WalletSnapshot({
    required this.balance,
    required this.items,
  });

  final int balance;
  final List<WalletLedgerEntry> items;

  factory WalletSnapshot.fromJson(Map<String, dynamic> json) => WalletSnapshot(
        balance: json['balance'] as int? ?? 0,
        items: (json['items'] as List<dynamic>? ?? [])
            .map((item) => WalletLedgerEntry.fromJson(item as Map<String, dynamic>))
            .toList(),
      );
}

class Entitlement {
  const Entitlement({
    required this.id,
    required this.planId,
    required this.status,
    required this.maxBoundDevices,
    required this.maxConcurrentSessions,
    required this.supportedProtocols,
    required this.endsAt,
  });

  final String id;
  final String planId;
  final String status;
  final int maxBoundDevices;
  final int maxConcurrentSessions;
  final List<String> supportedProtocols;
  final DateTime endsAt;

  factory Entitlement.fromJson(Map<String, dynamic> json) => Entitlement(
        id: json['id'] as String,
        planId: json['plan_id'] as String? ?? '',
        status: json['status'] as String? ?? '',
        maxBoundDevices: json['max_bound_devices'] as int? ?? 0,
        maxConcurrentSessions:
            json['max_concurrent_sessions'] as int? ?? 0,
        supportedProtocols:
            (json['supported_protocols'] as List<dynamic>? ?? []).cast<String>(),
        endsAt: DateTime.parse(json['ends_at'] as String),
      );
}

class Device {
  const Device({
    required this.id,
    required this.name,
    required this.platform,
    required this.status,
  });

  final String id;
  final String name;
  final String platform;
  final String status;

  factory Device.fromJson(Map<String, dynamic> json) => Device(
        id: json['id'] as String,
        name: json['name'] as String? ?? '',
        platform: json['platform'] as String? ?? '',
        status: json['status'] as String? ?? '',
      );
}

class Node {
  const Node({
    required this.id,
    required this.name,
    required this.region,
    required this.groupId,
  });

  final String id;
  final String name;
  final String region;
  final String groupId;

  factory Node.fromJson(Map<String, dynamic> json) => Node(
        id: json['id'] as String,
        name: json['name'] as String? ?? '',
        region: json['region'] as String? ?? '',
        groupId: json['group_id'] as String? ?? '',
      );
}

class ConnectionSession {
  const ConnectionSession({
    required this.id,
    required this.protocol,
    required this.nodeId,
    required this.status,
  });

  final String id;
  final String protocol;
  final String nodeId;
  final String status;

  factory ConnectionSession.fromJson(Map<String, dynamic> json) =>
      ConnectionSession(
        id: json['id'] as String,
        protocol: json['protocol'] as String? ?? '',
        nodeId: json['node_id'] as String? ?? '',
        status: json['status'] as String? ?? '',
      );
}

class ProtocolProfile {
  const ProtocolProfile({
    required this.id,
    required this.protocol,
    required this.config,
  });

  final String id;
  final String protocol;
  final String config;

  factory ProtocolProfile.fromJson(Map<String, dynamic> json) => ProtocolProfile(
        id: json['id'] as String,
        protocol: json['protocol'] as String? ?? '',
        config: json['config'] as String? ?? '',
      );
}
