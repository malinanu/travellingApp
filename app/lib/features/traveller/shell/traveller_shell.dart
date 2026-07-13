// Traveller shell: 3 bottom tabs (Discover / Spots / Trip) + floating Guide FAB.

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/palette.dart';

class TravellerShell extends StatelessWidget {
  final StatefulNavigationShell shell;
  const TravellerShell({super.key, required this.shell});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: shell,
      floatingActionButton: Container(
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
      bottomNavigationBar: NavigationBar(
        selectedIndex: shell.currentIndex,
        onDestinationSelected: (i) => shell.goBranch(i, initialLocation: i == shell.currentIndex),
        backgroundColor: Colors.white,
        indicatorColor: SpottoColors.accentTint,
        destinations: const [
          NavigationDestination(icon: Icon(Icons.explore_outlined), selectedIcon: Icon(Icons.explore), label: 'Discover'),
          NavigationDestination(icon: Icon(Icons.place_outlined), selectedIcon: Icon(Icons.place), label: 'Spots'),
          NavigationDestination(icon: Icon(Icons.route_outlined), selectedIcon: Icon(Icons.route), label: 'Trip'),
        ],
      ),
    );
  }
}
