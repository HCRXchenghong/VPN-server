import 'package:client_sdk/client_sdk.dart';
import 'package:flutter/material.dart';

void main() {
  runApp(const WindowsVpnApp());
}

class WindowsVpnApp extends StatelessWidget {
  const WindowsVpnApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'VPN Windows',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFFB65D2C)),
        useMaterial3: true,
      ),
      home: const WindowsDashboard(),
    );
  }
}

class WindowsDashboard extends StatefulWidget {
  const WindowsDashboard({super.key});

  @override
  State<WindowsDashboard> createState() => _WindowsDashboardState();
}

class _WindowsDashboardState extends State<WindowsDashboard> {
  final api = ApiClient(baseUrl: 'http://localhost:8080');

  List<Node> nodes = const [];
  String status = '等待加载节点';

  @override
  void initState() {
    super.initState();
    _loadNodes();
  }

  Future<void> _loadNodes() async {
    try {
      final nextNodes = await api.listNodes();
      if (!mounted) return;
      setState(() {
        nodes = nextNodes;
        status = '已加载 ${nextNodes.length} 个节点';
      });
    } catch (error) {
      if (!mounted) return;
      setState(() => status = error.toString());
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Windows Control')),
      body: Row(
        children: [
          SizedBox(
            width: 300,
            child: Card(
              margin: const EdgeInsets.all(16),
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text('Tunnel Providers'),
                    const SizedBox(height: 12),
                    const Text('WireGuard'),
                    const Text('IKEv2'),
                    const SizedBox(height: 24),
                    Text(status),
                    const SizedBox(height: 12),
                    FilledButton(
                      onPressed: _loadNodes,
                      child: const Text('Reload Nodes'),
                    ),
                  ],
                ),
              ),
            ),
          ),
          Expanded(
            child: ListView(
              padding: const EdgeInsets.fromLTRB(0, 16, 16, 16),
              children: nodes
                  .map(
                    (node) => Card(
                      child: ListTile(
                        title: Text(node.name),
                        subtitle: Text('${node.region} · ${node.groupId}'),
                      ),
                    ),
                  )
                  .toList(),
            ),
          ),
        ],
      ),
    );
  }
}
