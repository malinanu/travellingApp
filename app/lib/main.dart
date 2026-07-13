import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'app/router.dart';
import 'core/storage/local_store.dart';
import 'core/theme/theme.dart';
import 'data/repositories/repositories.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final store = await LocalStore.init();
  runApp(
    ProviderScope(
      overrides: [localStoreProvider.overrideWithValue(store)],
      child: const SpottoApp(),
    ),
  );
}

class SpottoApp extends ConsumerWidget {
  const SpottoApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return MaterialApp.router(
      title: 'Spotto',
      debugShowCheckedModeBanner: false,
      theme: spottoTheme(),
      routerConfig: ref.watch(routerProvider),
    );
  }
}
