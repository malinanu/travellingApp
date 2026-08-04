import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../data/repositories/repositories.dart';
import '../features/traveller/auth/auth_screen.dart';
import '../features/traveller/bookings/my_bookings_screen.dart';
import '../features/traveller/guide/guide_screen.dart';
import '../features/traveller/onboarding/onboarding_screen.dart';
import '../features/traveller/profile/profile_screen.dart';
import '../features/traveller/shell/traveller_shell.dart';
import '../features/traveller/discover/discover_screen.dart';
import '../features/traveller/spots/spots_screen.dart';
import '../features/traveller/trip/trip_screen.dart';
import '../features/traveller/weather/weather_screen.dart';

final rootNavigatorKey = GlobalKey<NavigatorState>();

final routerProvider = Provider<GoRouter>((ref) {
  return GoRouter(
    navigatorKey: rootNavigatorKey,
    initialLocation: '/t',
    routes: [
      GoRoute(path: '/t/auth', builder: (context, state) => const AuthScreen()),
      GoRoute(path: '/t/onboarding', builder: (context, state) => const OnboardingScreen()),
      GoRoute(path: '/t/guide', builder: (context, state) => const GuideScreen()),
      GoRoute(path: '/t/profile', builder: (context, state) => const ProfileScreen()),
      GoRoute(path: '/t/weather', builder: (context, state) => const WeatherScreen()),
      GoRoute(path: '/t/bookings', builder: (context, state) => const MyBookingsScreen()),
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
    ],
  );
});
