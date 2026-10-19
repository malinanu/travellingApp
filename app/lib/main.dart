import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'app/router.dart';
import 'core/api/push_notification_service.dart';
import 'core/share/incoming_links.dart';
import 'core/storage/local_store.dart';
import 'core/theme/theme.dart';
import 'core/utils/analytics_service.dart';
import 'data/repositories/inspiration_repository.dart';
import 'data/repositories/repositories.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  AnalyticsService.instance.initialize();
  final LocalStore store;
  try {
    store = await LocalStore.init();
  } catch (e, st) {
    // Show the failure instead of leaving a blank launch screen.
    AnalyticsService.instance.logCrash(e, st, context: 'LocalStore.init');
    runApp(_StartupErrorApp(error: e));
    return;
  }
  // Built outside the widget tree so push setup below can share the app's
  // ApiClient — the one whose 401 handler signs the user out.
  final container = ProviderContainer(
    // Riverpod 3 auto-retries a failing provider by default. This app
    // already has its own hand-rolled cache/retry patterns (manual
    // Timer + ref.invalidateSelf TTLs on the .family FutureProviders) —
    // stacking Riverpod's own retry on top risks double-retrying and
    // delaying/masking error surfacing that existing UI wasn't designed
    // for. Disabled to keep pre-3.0 behavior: fail once, surface it.
    retry: (retryCount, error) => null,
    overrides: [localStoreProvider.overrideWithValue(store)],
  );
  runApp(UncontrolledProviderScope(container: container, child: const SpottoApp()));
  // After the first frame is scheduled, never before: push setup can fail or
  // hang on iOS builds without the push entitlement and must not block the UI.
  unawaited(PushNotificationService.instance.initialize(
    container.read(apiProvider),
    onOpen: (data) => IncomingLinks.openNotification(container, data),
    onForeground: (data) {
      if (data['type'] == 'inspiration_ready') container.read(boardLinksProvider.notifier).refresh();
    },
  ));
  IncomingLinks.start(container);
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

/// Minimal fallback shown when startup fails before the real app can build.
class _StartupErrorApp extends StatelessWidget {
  const _StartupErrorApp({required this.error});
  final Object error;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      debugShowCheckedModeBanner: false,
      home: Scaffold(
        body: Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Text("Spotto couldn't start.\n\n$error", textAlign: TextAlign.center),
          ),
        ),
      ),
    );
  }
}
