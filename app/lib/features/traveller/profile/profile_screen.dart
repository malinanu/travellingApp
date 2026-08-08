// Traveller profile — port of Profile.jsx: details, travel style, interests,
// notifications, account rows + mock sub-pages, logout.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:image_picker/image_picker.dart';

import '../../../core/api/api_client.dart';
import '../../../core/api/upload_service.dart';
import '../../../core/theme/palette.dart';
import '../../../core/theme/theme_mode_provider.dart';
import '../../../core/utils/debounced_saver.dart';
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
  bool _uploadingAvatar = false;
  final _name = TextEditingController();
  final _email = TextEditingController();
  late Set<String> _styles;
  late List<String> _interests;
  final Map<String, bool> _notif = {
    'deals': true,
    'quest': true,
    'group': true,
    'email': false,
  };
  bool _savedFlash = false;
  // Auto-save: debounced while editing, plus an immediate flush() wired to
  // the Name field losing focus and to the header's manual "Save" pill
  // (kept as a fallback) — see DebouncedSaver's own doc comment.
  late final _saver = DebouncedSaver(save: () async => _save());

  @override
  void initState() {
    super.initState();
    final prefs = ref.read(prefsProvider);
    _styles = {...(prefs?.styles ?? const ['nomad'])};
    _interests = [...(prefs?.interests ?? const [])];
    final me = ref.read(meProvider);
    _name.text = me?['name'] as String? ?? '';
    _email.text = me?['email'] as String? ?? '';
  }

  @override
  void dispose() {
    _saver.dispose();
    _name.dispose();
    _email.dispose();
    super.dispose();
  }

  void _save() {
    ref.read(prefsProvider.notifier).save(UserPrefs(styles: _styles.toList(), interests: _interests));
    // Email isn't editable server-side yet (handleUpdateMe has no email
    // field — changing it would need a verification flow), so only name
    // round-trips here; the field stays display-only for email.
    ref.read(apiProvider).dio.put('/me', data: {'name': _name.text.trim()}).ignore();
    setState(() => _savedFlash = true);
    Future.delayed(const Duration(milliseconds: 1800), () {
      if (mounted) setState(() => _savedFlash = false);
    });
  }

  Future<void> _pickAvatar() async {
    if (_uploadingAvatar) return;
    final userId = ref.read(meProvider)?['id'] as String? ?? '';
    if (userId.isEmpty) return;
    final XFile? file = await ImagePicker().pickImage(
      source: ImageSource.gallery,
      maxWidth: 1024,
      imageQuality: 85,
    );
    if (file == null) return;
    final contentType = mimeForUpload(file.mimeType, file.name);
    if (contentType == null) {
      _snack('Unsupported image type.');
      return;
    }
    setState(() => _uploadingAvatar = true);
    try {
      final bytes = await file.readAsBytes();
      await ref.read(uploadServiceProvider).upload(
            bytes: bytes,
            contentType: contentType,
            ownerType: 'user',
            ownerId: userId,
            role: 'avatar',
          );
      await ref.read(meProvider.notifier).refresh();
    } catch (_) {
      _snack('Upload failed — try again.');
    } finally {
      if (mounted) setState(() => _uploadingAvatar = false);
    }
  }

  void _snack(String message) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(message)));
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
    final avatarUrl = ref.watch(meProvider)?['avatarUrl'] as String? ?? '';
    // /me loads asynchronously (Future.microtask in MeNotifier.build()), so
    // it may still be null when this screen first builds — seed the
    // controllers the moment real data actually arrives, once, rather than
    // only in initState (which can run before /me has resolved).
    ref.listen<Map<String, dynamic>?>(meProvider, (prev, next) {
      if (prev == null && next != null) {
        setState(() {
          _name.text = next['name'] as String? ?? '';
          _email.text = next['email'] as String? ?? '';
        });
      }
    });
    // Same async-timing issue as name/email above: prefsProvider may still
    // be un-hydrated (from MeNotifier.refresh()) when initState ran.
    ref.listen<UserPrefs?>(prefsProvider, (prev, next) {
      if (prev == null && next != null) {
        setState(() {
          _styles = {...next.styles};
          _interests = [...next.interests];
        });
      }
    });
    final name = _name.text.trim();
    final initials = name
        .split(' ')
        .where((w) => w.isNotEmpty)
        .map((w) => w[0])
        .take(2)
        .join()
        .toUpperCase();
    // Badge shows the first selected style — a small simplification rather
    // than redesigning the pill to list several.
    final styleObj =
        data.travelStyles.where((s) => s.id == _styles.firstOrNull).firstOrNull;

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
                    onTap: _saver.flush,
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
                      PressScale(
                        scale: 0.97,
                        onTap: _pickAvatar,
                        child: SizedBox(
                        width: 100,
                        height: 96,
                        child: Stack(
                          children: [
                            Container(
                              width: 92,
                              height: 92,
                              alignment: Alignment.center,
                              clipBehavior: Clip.antiAlias,
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
                              child: avatarUrl.isNotEmpty
                                  ? Image.network(resolveMediaUrl(avatarUrl),
                                      width: 92,
                                      height: 92,
                                      fit: BoxFit.cover,
                                      errorBuilder: (_, __, ___) => Text(
                                          initials.isEmpty ? 'LM' : initials,
                                          style: const TextStyle(
                                              fontSize: 32,
                                              fontWeight: FontWeight.w800,
                                              color: Colors.white)))
                                  : Text(initials.isEmpty ? 'LM' : initials,
                                      style: const TextStyle(
                                          fontSize: 32,
                                          fontWeight: FontWeight.w800,
                                          color: Colors.white)),
                            ),
                            if (_uploadingAvatar)
                              Container(
                                width: 92,
                                height: 92,
                                decoration: const BoxDecoration(
                                    shape: BoxShape.circle, color: Colors.black38),
                                child: const Center(
                                  child: SizedBox(
                                    width: 22,
                                    height: 22,
                                    child: CircularProgressIndicator(
                                        strokeWidth: 2.5, color: Colors.white),
                                  ),
                                ),
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
                          ],
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 24),

                  sectionCard(context, 'Your details', [
                    Focus(
                      onFocusChange: (hasFocus) {
                        if (!hasFocus) _saver.flush();
                      },
                      child: profField(context, 'Name', _name, Icons.person_outline,
                          onChanged: () {
                            setState(() {});
                            _saver.schedule();
                          }),
                    ),
                    const SizedBox(height: 14),
                    profField(context, 'Email', _email, Icons.mail_outline,
                        readOnly: true, helperText: 'Contact support to change your email'),
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
                            onTap: () {
                              setState(() {
                                _styles.contains(s.id) ? _styles.remove(s.id) : _styles.add(s.id);
                              });
                              _saver.schedule();
                            },
                            child: Container(
                              padding: const EdgeInsets.symmetric(horizontal: 12),
                              decoration: BoxDecoration(
                                color: _styles.contains(s.id)
                                    ? SpottoColors.accent
                                    : SpottoColors.sand,
                                borderRadius: BorderRadius.circular(16),
                              ),
                              child: Row(
                                children: [
                                  Icon(spottoIcon(s.icon),
                                      size: 18,
                                      color: _styles.contains(s.id)
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
                                            color: _styles.contains(s.id)
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
                            onTap: () {
                              setState(() => _interests.contains(it.id)
                                  ? _interests.remove(it.id)
                                  : _interests.add(it.id));
                              _saver.schedule();
                            },
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

                  sectionCard(context, 'Appearance', [
                    Builder(builder: (ctx) {
                      final mode = ref.watch(themeModeProvider);
                      final modeLabel = switch (mode) {
                        ThemeMode.dark => 'Dark',
                        ThemeMode.light => 'Light',
                        ThemeMode.system => 'System',
                      };
                      return Padding(
                        padding: const EdgeInsets.symmetric(vertical: 6),
                        child: Row(
                          children: [
                            iconTile(Icons.brightness_4_outlined),
                            const SizedBox(width: 12),
                            Expanded(
                              child: Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text('Theme', style: text.titleSmall),
                                  Text('$modeLabel mode',
                                      style: text.bodySmall
                                          ?.copyWith(color: SpottoColors.muted)),
                                ],
                              ),
                            ),
                            SegmentedButton<ThemeMode>(
                              style: SegmentedButton.styleFrom(
                                padding: const EdgeInsets.symmetric(horizontal: 8),
                                visualDensity: VisualDensity.compact,
                                selectedBackgroundColor: SpottoColors.accent,
                                selectedForegroundColor: Colors.white,
                                foregroundColor: SpottoColors.muted,
                              ),
                              segments: const [
                                ButtonSegment(
                                  value: ThemeMode.system,
                                  icon: Icon(Icons.brightness_auto, size: 15),
                                  tooltip: 'System',
                                ),
                                ButtonSegment(
                                  value: ThemeMode.light,
                                  icon: Icon(Icons.light_mode, size: 15),
                                  tooltip: 'Light',
                                ),
                                ButtonSegment(
                                  value: ThemeMode.dark,
                                  icon: Icon(Icons.dark_mode, size: 15),
                                  tooltip: 'Dark',
                                ),
                              ],
                              selected: {mode},
                              onSelectionChanged: (s) =>
                                  ref.read(themeModeProvider.notifier).setMode(s.first),
                            ),
                          ],
                        ),
                      );
                    }),
                  ]),
                  const SizedBox(height: 22),

                  sectionCard(context, 'Account', [
                    profRow(context, Icons.event_available_outlined, 'My Bookings',
                        sub: 'Upcoming & past bookings',
                        onTap: () => context.push('/t/bookings')),
                    const Divider(height: 1),
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
    {VoidCallback? onChanged, bool obscure = false, bool readOnly = false, String? helperText}) {
  final text = Theme.of(context).textTheme;
  return Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      Text(label, style: text.labelMedium?.copyWith(color: SpottoColors.inkSoft)),
      const SizedBox(height: 6),
      TextField(
        controller: controller,
        obscureText: obscure,
        readOnly: readOnly,
        onChanged: onChanged == null ? null : (_) => onChanged(),
        decoration: InputDecoration(
          fillColor: readOnly ? SpottoColors.line.withValues(alpha: 0.4) : SpottoColors.sand,
          prefixIcon: Icon(icon, size: 17, color: SpottoColors.muted),
          helperText: helperText,
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
