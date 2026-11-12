// Traveller shell: 4 bottom tabs (Discover / Spots / Stories / Trip), the
// Stories mini-player above them while audio is loaded, and the floating
// Guide FAB.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/l10n/l10n_provider.dart';
import '../../../core/theme/palette.dart';
import '../stories/story_player.dart';

class TravellerShell extends ConsumerStatefulWidget {
  final StatefulNavigationShell shell;

  /// The current route, so the mini-player can hide on the story it plays.
  final String location;
  const TravellerShell({super.key, required this.shell, required this.location});

  @override
  ConsumerState<TravellerShell> createState() => _TravellerShellState();
}

class _TravellerShellState extends ConsumerState<TravellerShell> {
  Offset _fabPosition = const Offset(300, 500);

  @override
  Widget build(BuildContext context) {
    final l10n = ref.watch(l10nProvider);

    return PopScope(
      canPop: widget.shell.currentIndex == 0,
      onPopInvokedWithResult: (didPop, result) {
        if (didPop) return;
        if (widget.shell.currentIndex != 0) {
          widget.shell.goBranch(0);
        }
      },
      child: Scaffold(
        body: Stack(
          children: [
            widget.shell,
            Positioned(
              left: _fabPosition.dx,
              top: _fabPosition.dy,
              child: GestureDetector(
                onPanUpdate: (details) {
                  setState(() {
                    _fabPosition += details.delta;
                  });
                },
                child: Container(
                  width: 58,
                  height: 58,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    gradient: const LinearGradient(
                      begin: Alignment.topLeft,
                      end: Alignment.bottomRight,
                      colors: [SpottoColors.accent2, SpottoColors.accent],
                    ),
                    border: Border.all(color: Colors.white, width: 3),
                    boxShadow: const [BoxShadow(color: SpottoColors.inkShadow, blurRadius: 12, offset: Offset(0, 5))],
                  ),
                  child: Material(
                    color: Colors.transparent,
                    shape: const CircleBorder(),
                    child: InkWell(
                      customBorder: const CircleBorder(),
                      onTap: () => context.push('/t/guide'),
                      child: const Icon(Icons.emoji_objects_outlined, color: Colors.white, size: 26),
                    ),
                  ),
                ),
              ),
            ),
          ],
        ),
        bottomNavigationBar: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            StoryMiniPlayer(location: widget.location),
            NavigationBar(
              selectedIndex: widget.shell.currentIndex,
              onDestinationSelected: (i) => widget.shell.goBranch(i, initialLocation: i == widget.shell.currentIndex),
              backgroundColor: Colors.white,
              indicatorColor: SpottoColors.accentTint,
              destinations: [
                NavigationDestination(
                  icon: const Icon(Icons.explore_outlined),
                  selectedIcon: const Icon(Icons.explore),
                  label: l10n.navDiscover,
                ),
                NavigationDestination(
                  icon: const Icon(Icons.place_outlined),
                  selectedIcon: const Icon(Icons.place),
                  label: l10n.navSpots,
                ),
                NavigationDestination(
                  icon: const Icon(Icons.auto_stories_outlined),
                  selectedIcon: const Icon(Icons.auto_stories),
                  label: l10n.navStories,
                ),
                NavigationDestination(
                  icon: const Icon(Icons.route_outlined),
                  selectedIcon: const Icon(Icons.route),
                  label: l10n.navTrip,
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
