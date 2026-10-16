import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../data/repositories/repositories.dart';
import 'package:qr_flutter/qr_flutter.dart';

Future<void> showQuestSheet(BuildContext context) {
  return showModalBottomSheet(
    context: context,
    isScrollControlled: true,
    backgroundColor: Colors.transparent,
    builder: (_) => const _QuestSheet(),
  );
}

class _QuestSheet extends ConsumerWidget {
  const _QuestSheet();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final questAsync = ref.watch(dailyQuestProvider);
    final quest = questAsync.value ?? ref.watch(cachedQuestProvider) ?? kDefaultQuest;

    final done = quest.progress.clamp(0, quest.goal);
    final isComplete = quest.isComplete;
    final sponsor = quest.sponsorName ?? 'Spotto App';
    final me = ref.watch(meProvider);
    final myId = me?['id'] as String? ?? '';

    final textTheme = Theme.of(context).textTheme;

    return Container(
      padding: const EdgeInsets.all(24),
      decoration: const BoxDecoration(
        color: Colors.white,
        borderRadius: BorderRadius.vertical(top: Radius.circular(24)),
      ),
      child: SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 48,
              height: 4,
              decoration: BoxDecoration(
                color: SpottoColors.muted.withValues(alpha: 0.2),
                borderRadius: BorderRadius.circular(2),
              ),
            ),
            const SizedBox(height: 24),
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(Icons.emoji_events, color: SpottoColors.sun, size: 28),
                const SizedBox(width: 8),
                Text('Daily Quest', style: textTheme.headlineSmall),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              'Sponsored by $sponsor',
              style: textTheme.bodyMedium?.copyWith(color: SpottoColors.accent, fontWeight: FontWeight.w700),
            ),
            const SizedBox(height: 32),
            _buildStampCard(done, quest.goal),
            const SizedBox(height: 32),
            Text(
              isComplete ? 'Reward Unlocked!' : 'Check in at ${quest.goal} spots today',
              style: textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w800),
            ),
            const SizedBox(height: 8),
            Text(
              isComplete
                  ? 'Show this screen at $sponsor to claim your ${quest.rewardDescription}.'
                  : 'Earn a ${quest.rewardDescription} from $sponsor.',
              textAlign: TextAlign.center,
              style: textTheme.bodyMedium?.copyWith(color: SpottoColors.muted),
            ),
            const SizedBox(height: 32),
            if (isComplete)
              Container(
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                  color: SpottoColors.sand,
                  borderRadius: BorderRadius.circular(16),
                  border: Border.all(color: SpottoColors.sun, width: 2),
                ),
                child: Column(
                  children: [
                    Container(
                      color: Colors.white,
                      padding: const EdgeInsets.all(8),
                      child: QrImageView(
                        data: 'QUEST-$myId',
                        version: QrVersions.auto,
                        size: 150.0,
                        backgroundColor: Colors.white,
                      ),
                    ),
                    const SizedBox(height: 12),
                    const Text(
                      'Vendor Scan Code',
                      style: TextStyle(fontWeight: FontWeight.w700, color: SpottoColors.ink),
                    ),
                  ],
                ),
              )
            else
              PressScale(
                onTap: () => Navigator.pop(context),
                child: Container(
                  width: double.infinity,
                  padding: const EdgeInsets.symmetric(vertical: 16),
                  alignment: Alignment.center,
                  decoration: BoxDecoration(color: SpottoColors.ink, borderRadius: BorderRadius.circular(999)),
                  child: const Text(
                    'Find spots nearby',
                    style: TextStyle(color: Colors.white, fontSize: 16, fontWeight: FontWeight.w800),
                  ),
                ),
              ),
            const SizedBox(height: 12),
          ],
        ),
      ),
    );
  }

  Widget _buildStampCard(int done, int goal) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 32),
      decoration: BoxDecoration(
        color: SpottoColors.ink,
        borderRadius: BorderRadius.circular(20),
        boxShadow: const [BoxShadow(color: SpottoColors.inkShadow, blurRadius: 16, offset: Offset(0, 8))],
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceEvenly,
        children: List.generate(goal, (index) {
          final isStamped = index < done;
          return _buildStampNode(isStamped, index + 1);
        }),
      ),
    );
  }

  Widget _buildStampNode(bool isStamped, int number) {
    return Container(
      width: 72,
      height: 72,
      decoration: BoxDecoration(
        color: isStamped ? SpottoColors.sun : Colors.transparent,
        shape: BoxShape.circle,
        border: Border.all(color: isStamped ? SpottoColors.sun : Colors.white.withValues(alpha: 0.2), width: 3),
      ),
      child: Center(
        child: isStamped
            ? const Icon(Icons.check_rounded, color: SpottoColors.ink, size: 40)
            : Text(
                '$number',
                style: TextStyle(color: Colors.white.withValues(alpha: 0.3), fontSize: 24, fontWeight: FontWeight.w800),
              ),
      ),
    );
  }
}
