import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'pages/home/index.dart';
import 'pages/order/list.dart';
import 'pages/user/index.dart';
import 'stores/message_store.dart';
import 'stores/user_store.dart';
import 'widgets/commons.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  final us = UserStore();
  final ms = MessageStore();
  us.init().then((_) {
    if (us.logged) {
      us.loadProfile();
      ms.refresh();
    }
  });
  runApp(MultiProvider(
    providers: [
      ChangeNotifierProvider.value(value: us),
      ChangeNotifierProvider.value(value: ms),
    ],
    child: const ItemRentalApp(),
  ));
}

/// 根:底部 tab 首页 / 订单 / 我的。
class ItemRentalApp extends StatelessWidget {
  const ItemRentalApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: '闲租',
      debugShowCheckedModeBanner: false,
      scaffoldMessengerKey: kSnack,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: kGreen),
        useMaterial3: true,
      ),
      home: const RootShell(),
    );
  }
}

class RootShell extends StatefulWidget {
  const RootShell({super.key});

  @override
  State<RootShell> createState() => _RootShellState();
}

class _RootShellState extends State<RootShell> {
  int _tab = 0;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: IndexedStack(
        index: _tab,
        children: const [HomePage(), OrderListPage(), UserPage()],
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _tab,
        onDestinationSelected: (i) {
          setState(() => _tab = i);
          if (i == 2) context.read<UserStore>().loadProfile();
        },
        destinations: const [
          NavigationDestination(icon: Icon(Icons.home_outlined), selectedIcon: Icon(Icons.home), label: '首页'),
          NavigationDestination(icon: Icon(Icons.receipt_long_outlined), selectedIcon: Icon(Icons.receipt_long), label: '订单'),
          NavigationDestination(icon: Icon(Icons.person_outline), selectedIcon: Icon(Icons.person), label: '我的'),
        ],
      ),
    );
  }
}
