import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'app/router.dart';
import 'core/api/api_client.dart';
import 'core/api/push_notification_service.dart';
import 'core/storage/local_store.dart';
import 'core/theme/theme.dart';
import 'core/theme/theme_mode_provider.dart';
import 'core/utils/analytics_service.dart';
import 'data/repositories/repositories.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  AnalyticsService.instance.initialize();
  final store = await LocalStore.init();
  final api = ApiClient(store);
  await PushNotificationService.instance.initialize(api);
  runApp(
    ProviderScope(
      // Riverpod 3 auto-retries a failing provider by default. This app
      // already has its own hand-rolled cache/retry patterns (manual
      // Timer + ref.invalidateSelf TTLs on the .family FutureProviders) —
      // stacking Riverpod's own retry on top risks double-retrying and
      // delaying/masking error surfacing that existing UI wasn't designed
      // for. Disabled to keep pre-3.0 behavior: fail once, surface it.
      retry: (retryCount, error) => null,
      overrides: [localStoreProvider.overrideWithValue(store)],
      child: const SpottoApp(),
    ),
  );
}

class SpottoApp extends ConsumerWidget {
  const SpottoApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final themeMode = ref.watch(themeModeProvider);
    return MaterialApp.router(
      title: 'Spotto',
      debugShowCheckedModeBanner: false,
      theme: spottoTheme(),
      darkTheme: spottoThemeDark(),
      themeMode: themeMode,
      routerConfig: ref.watch(routerProvider),
    );
  }
}
