// Traveller profile — port of Profile.jsx: details, travel style, interests,
// notifications, account rows + mock sub-pages, logout.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/mock/places.dart' as data;
import '../../../data/models/models.dart';
import '../../../data/repositories/repositories.dart';
import 'profile_subpages.dart';

const _danger = Color(0xFFE11D48);
const _saved = Color(0xFF16A34A);

class ProfileScreen extends ConsumerStatefulWidget {
  const ProfileScreen({super.key});

  @override
  ConsumerState<ProfileScreen> createState() => _ProfileScreenState();
}

class _ProfileScreenState extends ConsumerState<ProfileScreen> {
  String? _sub; // security | payment | privacy | help
  final _name = TextEditingController(text: 'Lena Müller');
  final _email = TextEditingController(text: 'lena@email.com');
  late String _style;
  late List<String> _interests;
  final Map<String, bool> _notif = {
    'deals': true,
    'quest': true,
    'group': true,
    'email': false,
  };
  bool _savedFlash = false;

  @override
  void initState() {
    super.initState();
    final prefs = ref.read(prefsProvider);
    _style = prefs?.style ?? 'nomad';
    _interests = [...(prefs?.interests ?? const [])];
  }

  @override
  void dispose() {
    _name.dispose();
    _email.dispose();
    super.dispose();
  }

  void _save() {
    ref.read(prefsProvider.notifier).save(UserPrefs(style: _style, interests: _interests));
    setState(() => _savedFlash = true);
    Future.delayed(const Duration(milliseconds: 1800), () {
      if (mounted) setState(() => _savedFlash = false);
    });
  }

  @override
  Widget build(BuildContext context) {
    switch (_sub) {
      case 'security':
        return SecurityPage(onBack: () => setState(() => _sub = null));
      case 'payment':
        return PaymentPage(onBack: () => setState(() => _sub = null));
      case 'privacy':
        return PrivacyPage(onBack: () => setState(() => _sub = null));
      case 'help':
        return HelpPage(onBack: () => setState(() => _sub = null));
    }

    final text = Theme.of(context).textTheme;
    final name = _name.text.trim();
    final initials = name
        .split(' ')
        .where((w) => w.isNotEmpty)
        .map((w) => w[0])
        .take(2)
        .join()
        .toUpperCase();
    final styleObj = data.travelStyles.where((s) => s.id == _style).firstOrNull;

    return Scaffold(
      backgroundColor: SpottoColors.sand,
      body: SafeArea(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
              child: Row(
                children: [
                  circleButton(Icons.chevron_left, () => context.pop()),
                  Expanded(
                    child: Center(
                      child:
                          Text('Profile', style: text.titleMedium?.copyWith(fontSize: 17)),
                    ),
                  ),
                  PressScale(
                    scale: 0.95,
                    onTap: _save,
                    child: Container(
                      height: 40,
                      padding: const EdgeInsets.symmetric(horizontal: 16),
                      decoration: BoxDecoration(
                        color: _savedFlash ? _saved : SpottoColors.accent,
                        borderRadius: BorderRadius.circular(999),
                      ),
                      child: Row(
                        children: [
                          if (_savedFlash) ...[
                            const Icon(Icons.check, size: 15, color: Colors.white),
                            const SizedBox(width: 5),
                          ],
                          Text(_savedFlash ? 'Saved' : 'Save',
                              style: const TextStyle(
                                  fontSize: 13.5,
                                  fontWeight: FontWeight.w800,
                                  color: Colors.white)),
                        ],
                      ),
                    ),
                  ),
                ],
              ),
            ),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(20, 6, 20, 40),
                children: [
                  // hero
                  Column(
                    children: [
                      SizedBox(
                        width: 100,
                        height: 96,
                        child: Stack(
                          children: [
                            Container(
                              width: 92,
                              height: 92,
                              alignment: Alignment.center,
                              decoration: const BoxDecoration(
                                shape: BoxShape.circle,
                                gradient: LinearGradient(
                                  begin: Alignment.topLeft,
                                  end: Alignment.bottomRight,
                                  colors: [SpottoColors.accent, SpottoColors.accent2],
                                ),
                                boxShadow: [
                                  BoxShadow(
                                      color: Color(0x660B6F86),
                                      blurRadius: 30,
                                      offset: Offset(0, 14),
                                      spreadRadius: -12)
                                ],
                              ),
                              child: Text(initials.isEmpty ? 'LM' : initials,
                                  style: const TextStyle(
                                      fontSize: 32,
                                      fontWeight: FontWeight.w800,
                                      color: Colors.white)),
                            ),
                            Positioned(
                              right: 0,
                              bottom: 0,
                              child: Container(
                                width: 32,
                                height: 32,
                                decoration: BoxDecoration(
                                  color: Colors.white,
                                  shape: BoxShape.circle,
                                  border: Border.all(
                                      color: SpottoColors.accentTint, width: 1.5),
                                  boxShadow: const [
                                    BoxShadow(
                                        color: Color(0x33103D3A),
                                        blurRadius: 10,
                                        offset: Offset(0, 3))
                                  ],
                                ),
                                child: const Icon(Icons.photo_camera_outlined,
                                    size: 15, color: SpottoColors.accent),
                              ),
                            ),
                          ],
                        ),
                      ),
                      const SizedBox(height: 12),
                      Text(name.isEmpty ? 'Traveller' : name,
                          style: text.headlineSmall?.copyWith(fontSize: 21)),
                      const SizedBox(height: 6),
                      Container(
                        padding:
                            const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                        decoration: BoxDecoration(
                          color: SpottoColors.accentTint,
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            if (styleObj != null) ...[
                              Icon(spottoIcon(styleObj.icon),
                                  size: 13, color: SpottoColors.accent),
                              const SizedBox(width: 5),
                            ],
                            Text(styleObj?.label ?? 'Traveler',
                                style: const TextStyle(
                                    fontSize: 12.5,
                                    fontWeight: FontWeight.w800,
                                    color: SpottoColors.accent)),
                            const Text(' · Weligama',
                                style: TextStyle(
                                    fontSize: 11,
                                    fontWeight: FontWeight.w700,
                                    color: SpottoColors.muted)),
                          ],
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 24),

                  sectionCard(context, 'Your details', [
                    profField(context, 'Name', _name, Icons.person_outline,
                        onChanged: () => setState(() {})),
                    const SizedBox(height: 14),
                    profField(context, 'Email', _email, Icons.mail_outline),
                  ]),
                  const SizedBox(height: 22),

                  sectionCard(context, 'Travel style', [
                    GridView.count(
                      shrinkWrap: true,
                      physics: const NeverScrollableScrollPhysics(),
                      crossAxisCount: 2,
                      mainAxisSpacing: 10,
                      crossAxisSpacing: 10,
                      childAspectRatio: 3.1,
                      children: [
                        for (final s in data.travelStyles)
                          PressScale(
                            scale: 0.98,
                            onTap: () => setState(() => _style = s.id),
                            child: Container(
                              padding: const EdgeInsets.symmetric(horizontal: 12),
                              decoration: BoxDecoration(
                                color: _style == s.id
                                    ? SpottoColors.accent
                                    : SpottoColors.sand,
                                borderRadius: BorderRadius.circular(16),
                              ),
                              child: Row(
                                children: [
                                  Icon(spottoIcon(s.icon),
                                      size: 18,
                                      color: _style == s.id
                                          ? Colors.white
                                          : SpottoColors.ink),
                                  const SizedBox(width: 10),
                                  Expanded(
                                    child: Text(s.label,
                                        maxLines: 1,
                                        overflow: TextOverflow.ellipsis,
                                        style: TextStyle(
                                            fontSize: 13,
                                            fontWeight: FontWeight.w800,
                                            color: _style == s.id
                                                ? Colors.white
                                                : SpottoColors.ink)),
                                  ),
                                ],
                              ),
                            ),
                          ),
                      ],
                    ),
                  ]),
                  const SizedBox(height: 22),

                  sectionCard(context, 'Interests', [
                    Wrap(
                      spacing: 8,
                      runSpacing: 8,
                      children: [
                        for (final it in data.interests)
                          PressScale(
                            scale: 0.95,
                            onTap: () => setState(() => _interests.contains(it.id)
                                ? _interests.remove(it.id)
                                : _interests.add(it.id)),
                            child: Container(
                              padding: const EdgeInsets.fromLTRB(10, 8, 14, 8),
                              decoration: BoxDecoration(
                                color: _interests.contains(it.id)
                                    ? SpottoColors.accent
                                    : SpottoColors.sand,
                                borderRadius: BorderRadius.circular(999),
                              ),
                              child: Row(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  Icon(spottoIcon(it.icon),
                                      size: 15,
                                      color: _interests.contains(it.id)
                                          ? Colors.white
                                          : SpottoColors.ink),
                                  const SizedBox(width: 6),
                                  Text(it.label,
                                      style: TextStyle(
                                          fontSize: 13,
                                          fontWeight: FontWeight.w700,
                                          color: _interests.contains(it.id)
                                              ? Colors.white
                                              : SpottoColors.ink)),
                                ],
                              ),
                            ),
                          ),
                      ],
                    ),
                  ]),
                  const SizedBox(height: 22),

                  sectionCard(context, 'Notifications', [
                    for (final (i, n) in const [
                      (k: 'deals', icon: Icons.local_offer_outlined, title: 'Deal alerts near me', sub: 'When a perk is within walking distance'),
                      (k: 'quest', icon: Icons.explore_outlined, title: 'Daily quest reminders', sub: "Nudge me to finish the day's quest"),
                      (k: 'group', icon: Icons.group_outlined, title: 'Group activity', sub: 'When friends check in nearby'),
                      (k: 'email', icon: Icons.mail_outline, title: 'Email updates', sub: 'Occasional travel tips & news'),
                    ].indexed) ...[
                      if (i > 0) const Divider(height: 1),
                      Padding(
                        padding: const EdgeInsets.symmetric(vertical: 10),
                        child: Row(
                          children: [
                            iconTile(n.icon),
                            const SizedBox(width: 12),
                            Expanded(
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text(n.title, style: text.titleSmall),
                                  Text(n.sub,
                                      style: text.bodySmall
                                          ?.copyWith(color: SpottoColors.muted)),
                                ],
                              ),
                            ),
                            Switch(
                              value: _notif[n.k] ?? false,
                              activeTrackColor: SpottoColors.accent,
                              onChanged: (v) => setState(() => _notif[n.k] = v),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ]),
                  const SizedBox(height: 22),

                  sectionCard(context, 'Account', [
                    profRow(context, Icons.lock_outline, 'Password & security',
                        sub: 'Change password, 2FA',
                        onTap: () => setState(() => _sub = 'security')),
                    const Divider(height: 1),
                    profRow(context, Icons.credit_card, 'Saved payment',
                        sub: 'Visa ·· 4291',
                        onTap: () => setState(() => _sub = 'payment')),
                    const Divider(height: 1),
                    profRow(context, Icons.verified_user_outlined, 'Privacy',
                        sub: 'Location & data sharing',
                        onTap: () => setState(() => _sub = 'privacy')),
                    const Divider(height: 1),
                    profRow(context, Icons.help_outline, 'Help & support',
                        onTap: () => setState(() => _sub = 'help')),
                  ]),
                  const SizedBox(height: 22),

                  sectionCard(context, null, [
                    profRow(context, Icons.logout, 'Log out', danger: true, onTap: () {
                      ref.read(authProvider.notifier).signOut();
                      context.go('/t/auth');
                    }),
                  ]),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ── shared profile building blocks (also used by the sub-pages) ──

Widget circleButton(IconData icon, VoidCallback onTap) => PressScale(
      scale: 0.9,
      onTap: onTap,
      child: Container(
        width: 40,
        height: 40,
        decoration: const BoxDecoration(
          color: Colors.white,
          shape: BoxShape.circle,
          boxShadow: [BoxShadow(color: Color(0x14103D3A), blurRadius: 8, offset: Offset(0, 2))],
        ),
        child: Icon(icon, size: 22, color: SpottoColors.ink),
      ),
    );

Widget iconTile(IconData icon, {bool danger = false}) => Container(
      width: 36,
      height: 36,
      decoration: BoxDecoration(
        color: danger ? _danger.withValues(alpha: 0.1) : SpottoColors.accentTint,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Icon(icon, size: 18, color: danger ? _danger : SpottoColors.accent),
    );

Widget sectionCard(BuildContext context, String? title, List<Widget> children) {
  final text = Theme.of(context).textTheme;
  return Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      if (title != null)
        Padding(
          padding: const EdgeInsets.only(left: 4, bottom: 8),
          child: Text(title.toUpperCase(),
              style: text.labelSmall?.copyWith(color: SpottoColors.muted, letterSpacing: 0.6)),
        ),
      Container(
        width: double.infinity,
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: Colors.white,
          borderRadius: BorderRadius.circular(24),
          boxShadow: const [
            BoxShadow(color: Color(0x33103D3A), blurRadius: 14, offset: Offset(0, 2), spreadRadius: -8)
          ],
        ),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: children),
      ),
    ],
  );
}

Widget profField(BuildContext context, String label, TextEditingController controller,
    IconData icon,
    {VoidCallback? onChanged, bool obscure = false}) {
  final text = Theme.of(context).textTheme;
  return Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      Text(label, style: text.labelMedium?.copyWith(color: SpottoColors.inkSoft)),
      const SizedBox(height: 6),
      TextField(
        controller: controller,
        obscureText: obscure,
        onChanged: onChanged == null ? null : (_) => onChanged(),
        decoration: InputDecoration(
          fillColor: SpottoColors.sand,
          prefixIcon: Icon(icon, size: 17, color: SpottoColors.muted),
        ),
      ),
    ],
  );
}

Widget profRow(BuildContext context, IconData icon, String title,
    {String? sub, VoidCallback? onTap, bool danger = false}) {
  final text = Theme.of(context).textTheme;
  return PressScale(
    onTap: onTap,
    child: Padding(
      padding: const EdgeInsets.symmetric(vertical: 12),
      child: Row(
        children: [
          iconTile(icon, danger: danger),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title,
                    style: text.titleSmall?.copyWith(color: danger ? _danger : null)),
                if (sub != null)
                  Text(sub, style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
              ],
            ),
          ),
          if (!danger)
            const Icon(Icons.chevron_right, size: 18, color: SpottoColors.muted),
        ],
      ),
    ),
  );
}
