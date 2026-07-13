import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../data/repositories/repositories.dart';
import '../features/traveller/auth/auth_screen.dart';
import '../features/traveller/guide/guide_screen.dart';
import '../features/traveller/onboarding/onboarding_screen.dart';
import '../features/traveller/profile/profile_screen.dart';
import '../features/traveller/shell/traveller_shell.dart';
import '../features/traveller/discover/discover_screen.dart';
import '../features/traveller/spots/spots_screen.dart';
import '../features/traveller/trip/trip_screen.dart';
import '../features/vendor/account/account_screen.dart';
import '../features/vendor/auth/vendor_auth_screen.dart';
import '../features/vendor/dashboard/dashboard_screen.dart';
import '../features/vendor/growth/growth_screen.dart';
import '../features/vendor/menu/menu_manager_screen.dart';
import '../features/vendor/more/more_screen.dart';
import '../features/vendor/offers/offers_screen.dart';
import '../features/vendor/profile/profile_editor_screen.dart';
import '../features/vendor/reports/reports_screen.dart';
import '../features/vendor/shell/vendor_shell.dart';
import '../features/vendor/validate/validate_screen.dart';
import 'role_gate.dart';

final rootNavigatorKey = GlobalKey<NavigatorState>();

final routerProvider = Provider<GoRouter>((ref) {
  return GoRouter(
    navigatorKey: rootNavigatorKey,
    initialLocation: switch (ref.read(roleProvider)) {
      'traveller' => '/t',
      'vendor' => '/v',
      _ => '/',
    },
    routes: [
      GoRoute(path: '/', builder: (context, state) => const RoleGateScreen()),
      GoRoute(path: '/t/auth', builder: (context, state) => const AuthScreen()),
      GoRoute(path: '/t/onboarding', builder: (context, state) => const OnboardingScreen()),
      GoRoute(path: '/t/guide', builder: (context, state) => const GuideScreen()),
      GoRoute(path: '/t/profile', builder: (context, state) => const ProfileScreen()),
      StatefulShellRoute.indexedStack(
        builder: (context, state, shell) => TravellerShell(shell: shell),
        redirect: (context, state) {
          if (!ref.read(authProvider)) return '/t/auth';
          if (ref.read(prefsProvider) == null) return '/t/onboarding';
          return null;
        },
        branches: [
          StatefulShellBranch(routes: [
            GoRoute(path: '/t', builder: (context, state) => const DiscoverScreen()),
          ]),
          StatefulShellBranch(routes: [
            GoRoute(path: '/t/spots', builder: (context, state) => const SpotsScreen()),
          ]),
          StatefulShellBranch(routes: [
            GoRoute(path: '/t/trip', builder: (context, state) => const TripScreen()),
          ]),
        ],
      ),
      GoRoute(path: '/v/auth', builder: (context, state) => const VendorAuthScreen()),
      StatefulShellRoute.indexedStack(
        builder: (context, state, shell) => VendorShell(shell: shell),
        redirect: (context, state) =>
            ref.read(vendorAuthProvider) == null ? '/v/auth' : null,
        branches: [
          StatefulShellBranch(routes: [
            GoRoute(path: '/v', builder: (context, state) => const VendorDashboardScreen()),
          ]),
          StatefulShellBranch(routes: [
            GoRoute(
                path: '/v/offers',
                builder: (context, state) => const VendorOffersScreen()),
          ]),
          StatefulShellBranch(routes: [
            GoRoute(
                path: '/v/validate',
                builder: (context, state) => const VendorValidateScreen()),
          ]),
          StatefulShellBranch(routes: [
            GoRoute(
                path: '/v/growth',
                builder: (context, state) => const VendorGrowthScreen()),
          ]),
          StatefulShellBranch(routes: [
            GoRoute(
              path: '/v/more',
              builder: (context, state) => const VendorMoreScreen(),
              routes: [
                GoRoute(
                    path: 'menu',
                    parentNavigatorKey: rootNavigatorKey,
                    builder: (context, state) => const VendorMenuScreen()),
                GoRoute(
                    path: 'reports',
                    parentNavigatorKey: rootNavigatorKey,
                    builder: (context, state) => const VendorReportsScreen()),
                GoRoute(
                    path: 'profile',
                    parentNavigatorKey: rootNavigatorKey,
                    builder: (context, state) => const VendorProfileEditorScreen()),
                GoRoute(
                    path: 'account',
                    parentNavigatorKey: rootNavigatorKey,
                    builder: (context, state) => const VendorAccountScreen()),
              ],
            ),
          ]),
        ],
      ),
    ],
  );
});
