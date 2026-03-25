import 'dart:convert';

import 'package:http/http.dart' as http;

import 'models.dart';

class ApiClient {
  ApiClient({
    required this.baseUrl,
    http.Client? httpClient,
  }) : _httpClient = httpClient ?? http.Client();

  final String baseUrl;
  final http.Client _httpClient;

  String? accessToken;
  String? refreshToken;

  Future<User> register({
    required String email,
    required String password,
  }) async {
    final payload = await _post('/auth/register', body: {
      'email': email,
      'password': password,
    });
    return User.fromJson(payload);
  }

  Future<AuthSession> login({
    required String email,
    required String password,
  }) async {
    final payload = await _post('/auth/login', body: {
      'email': email,
      'password': password,
    });
    final session = AuthSession.fromJson(payload);
    accessToken = session.accessToken;
    refreshToken = session.refreshToken;
    return session;
  }

  Future<User> verifyEmail() async {
    final payload = await _post('/auth/verify-email');
    return User.fromJson(payload);
  }

  Future<List<Plan>> listPlans() async {
    final payload = await _get('/plans') as List<dynamic>;
    return payload
        .map((item) => Plan.fromJson(item as Map<String, dynamic>))
        .toList();
  }

  Future<WalletSnapshot> walletSnapshot() async {
    final payload = await _get('/wallet/ledger');
    return WalletSnapshot.fromJson(payload as Map<String, dynamic>);
  }

  Future<Map<String, dynamic>> createTopup(int points) {
    return _post('/wallet/topups/alipay', body: {'points': points});
  }

  Future<Map<String, dynamic>> confirmTopup(String orderId, String tradeNo) {
    return _post('/wallet/topups/alipay/callback', body: {
      'order_id': orderId,
      'trade_no': tradeNo,
      'status': 'paid',
    });
  }

  Future<Map<String, dynamic>> redeem(String planId) {
    return _post('/redeems', body: {'plan_id': planId});
  }

  Future<List<Entitlement>> listEntitlements() async {
    final payload = await _get('/entitlements') as List<dynamic>;
    return payload
        .map((item) => Entitlement.fromJson(item as Map<String, dynamic>))
        .toList();
  }

  Future<List<Device>> listDevices() async {
    final payload = await _get('/devices') as List<dynamic>;
    return payload
        .map((item) => Device.fromJson(item as Map<String, dynamic>))
        .toList();
  }

  Future<Device> bindDevice({
    required String name,
    required String platform,
  }) async {
    final payload = await _post('/devices/bind', body: {
      'name': name,
      'platform': platform,
    });
    return Device.fromJson(payload);
  }

  Future<List<Node>> listNodes() async {
    final payload = await _get('/nodes') as List<dynamic>;
    return payload
        .map((item) => Node.fromJson(item as Map<String, dynamic>))
        .toList();
  }

  Future<ConnectionSession> connect({
    required String deviceId,
    required String entitlementId,
    required String nodeId,
    required String protocol,
  }) async {
    final payload = await _post('/sessions/connect', body: {
      'device_id': deviceId,
      'entitlement_id': entitlementId,
      'node_id': nodeId,
      'protocol': protocol,
    });
    return ConnectionSession.fromJson(payload);
  }

  Future<ProtocolProfile> getProfile({
    required String protocol,
    required String deviceId,
    required String entitlementId,
    required String nodeId,
  }) async {
    final payload = await _get(
      '/profiles/$protocol?device_id=$deviceId&entitlement_id=$entitlementId&node_id=$nodeId',
    );
    return ProtocolProfile.fromJson(payload);
  }

  Future<dynamic> _get(String path) async {
    final response = await _httpClient.get(
      Uri.parse('$baseUrl$path'),
      headers: _headers(),
    );
    return _decode(response);
  }

  Future<Map<String, dynamic>> _post(
    String path, {
    Map<String, dynamic>? body,
  }) async {
    final response = await _httpClient.post(
      Uri.parse('$baseUrl$path'),
      headers: _headers(),
      body: body == null ? null : jsonEncode(body),
    );
    return _decode(response);
  }

  Map<String, String> _headers() {
    final headers = <String, String>{'Content-Type': 'application/json'};
    if (accessToken != null && accessToken!.isNotEmpty) {
      headers['Authorization'] = 'Bearer $accessToken';
    }
    return headers;
  }

  dynamic _decode(http.Response response) {
    final jsonBody = jsonDecode(response.body) as Object?;
    if (response.statusCode >= 400) {
      final payload = jsonBody is Map<String, dynamic> ? jsonBody : const <String, dynamic>{};
      throw StateError(payload['error']?.toString() ?? 'Request failed');
    }
    if (jsonBody is Map<String, dynamic> || jsonBody is List<dynamic>) {
      return jsonBody;
    }
    throw StateError('Expected JSON object or array response.');
  }
}
