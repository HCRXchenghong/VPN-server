abstract class TunnelProvider {
  String get protocol;

  Future<void> configure(String rawProfile);

  Future<void> connect();

  Future<void> disconnect();
}

class WireGuardTunnelProvider implements TunnelProvider {
  @override
  String get protocol => 'wireguard';

  @override
  Future<void> configure(String rawProfile) async {
    if (rawProfile.isEmpty) {
      throw StateError('WireGuard profile is empty.');
    }
  }

  @override
  Future<void> connect() async {}

  @override
  Future<void> disconnect() async {}
}

class Ikev2TunnelProvider implements TunnelProvider {
  @override
  String get protocol => 'ikev2';

  @override
  Future<void> configure(String rawProfile) async {
    if (rawProfile.isEmpty) {
      throw StateError('IKEv2 profile is empty.');
    }
  }

  @override
  Future<void> connect() async {}

  @override
  Future<void> disconnect() async {}
}
