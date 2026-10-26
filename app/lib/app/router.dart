import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../data/repositories/repositories.dart';
import '../features/traveller/auth/auth_screen.dart';
import '../features/traveller/bookings/my_bookings_screen.dart';
import '../features/traveller/guide/guide_screen.dart';
import '../features/traveller/inspiration/inspiration_board_screen.dart';
import '../features/traveller/inspiration/inspiration_link_screen.dart';
import '../features/traveller/onboarding/onboarding_screen.dart';
import '../features/traveller/profile/profile_screen.dart';
import '../features/traveller/shell/traveller_shell.dart';
import '../features/traveller/discover/discover_screen.dart';
import '../features/traveller/spots/spots_screen.dart';
import '../features/traveller/stories/stories_screen.dart';
import '../features/traveller/stories/story_screen.dart';
import '../features/traveller/trip/trip_screen.dart';
import '../features/traveller/weather/weather_screen.dart';

final rootNavigatorKey = GlobalKey<NavigatorState>();

final routerProvider = Provider<GoRouter>((ref) {
  // The app trusts the stored token at launch and never waits on the server
  // to confirm the session (see AuthNotifier.build). The other half of that
  // deal: when a request later comes back 401 and signs the user out, the
  // router has to re-run its redirects right away rather than leaving them
  // on a signed-in screen with a dead token until their next navigation.
  final authChanges = ValueNotifier<bool>(ref.read(authProvider));
  ref.listen(authProvider, (_, signedIn) => authChanges.value = signedIn);
  ref.onDispose(authChanges.dispose);

  return GoRouter(
    navigatorKey: rootNavigatorKey,
    initialLocation: '/t',
    refreshListenable: authChanges,
    routes: [
      GoRoute(path: '/t/auth', builder: (context, state) => const AuthScreen()),
      GoRoute(path: '/t/onboarding', builder: (context, state) => const OnboardingScreen()),
      GoRoute(path: '/t/guide', builder: (context, state) => const GuideScreen()),
      GoRoute(path: '/t/profile', builder: (context, state) => const ProfileScreen()),
      GoRoute(path: '/t/weather', builder: (context, state) => const WeatherScreen()),
      GoRoute(path: '/t/bookings', builder: (context, state) => const MyBookingsScreen()),
      GoRoute(
        path: '/t/inspiration',
        redirect: (context, state) => ref.read(authProvider) ? null : '/t/auth',
        builder: (context, state) => InspirationBoardScreen(
          sharedUrl: state.uri.queryParameters['add'],
          savedTab: state.uri.queryParameters['tab'] == 'saved',
        ),
      ),
      GoRoute(
        path: '/t/inspiration/:id',
        redirect: (context, state) => ref.read(authProvider) ? null : '/t/auth',
        builder: (context, state) => InspirationLinkScreen(linkId: state.pathParameters['id']!),
      ),
      StatefulShellRoute.indexedStack(
        builder: (context, state, shell) => TravellerShell(shell: shell, location: state.uri.path),
        redirect: (context, state) {
          if (!ref.read(authProvider)) return '/t/auth';
          if (ref.read(prefsProvider) == null) return '/t/onboarding';
          return null;
        },
        branches: [
          StatefulShellBranch(
            routes: [GoRoute(path: '/t', builder: (context, state) => const DiscoverScreen())],
          ),
          StatefulShellBranch(
            routes: [GoRoute(path: '/t/spots', builder: (context, state) => const SpotsScreen())],
          ),
          StatefulShellBranch(
            routes: [
              GoRoute(
                path: '/t/stories',
                builder: (context, state) => const StoriesScreen(),
                routes: [
                  GoRoute(
                    path: ':id',
                    builder: (context, state) => StoryScreen(storyId: state.pathParameters['id']!),
                  ),
                ],
              ),
            ],
          ),
          StatefulShellBranch(
            routes: [GoRoute(path: '/t/trip', builder: (context, state) => const TripScreen())],
          ),
        ],
      ),
    ],
  );
});
