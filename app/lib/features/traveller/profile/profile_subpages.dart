// Profile sub-pages. Security and Privacy are wired to the real API;
// Payment is a deferred "coming soon" stub (needs a payment processor
// decision); Help is legitimate static content.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:qr_flutter/qr_flutter.dart';

import '../../../core/api/api_client.dart';
import '../../../core/theme/palette.dart';
import '../../../data/repositories/repositories.dart';
import 'payment_methods_screen.dart';
import 'profile_screen.dart';

class _SubPage extends StatelessWidget {
  final String title;
  final VoidCallback onBack;
  final List<Widget> children;
  const _SubPage({required this.title, required this.onBack, required this.children});

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return Scaffold(
      backgroundColor: SpottoColors.sand,
      body: SafeArea(
        child: Column(
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
              child: Row(
                children: [
                  circleButton(Icons.chevron_left, onBack),
                  const SizedBox(width: 12),
                  Text(title, style: text.titleMedium?.copyWith(fontSize: 17)),
                ],
              ),
            ),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(20, 6, 20, 40),
                children: children,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class SecurityPage extends ConsumerStatefulWidget {
  final VoidCallback onBack;
  const SecurityPage({super.key, required this.onBack});

  @override
  ConsumerState<SecurityPage> createState() => _SecurityPageState();
}

class _SecurityPageState extends ConsumerState<SecurityPage> {
  final _current = TextEditingController();
  final _pw = TextEditingController();
  final _pw2 = TextEditingController();
  bool _saved = false;
  bool _saving = false;
  String? _error;
  bool _twofaBusy = false;

  @override
  void dispose() {
    _current.dispose();
    _pw.dispose();
    _pw2.dispose();
    super.dispose();
  }

  Future<void> _changePassword() async {
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      final res = await ref.read(apiProvider).dio.put('/auth/password', data: {
        'currentPassword': _current.text,
        'newPassword': _pw.text,
      });
      await ref.read(localStoreProvider).setToken(res.data['token'] as String);
      _current.clear();
      _pw.clear();
      _pw2.clear();
      setState(() {
        _saving = false;
        _saved = true;
      });
      Future.delayed(const Duration(milliseconds: 1800), () {
        if (mounted) setState(() => _saved = false);
      });
    } catch (e) {
      setState(() {
        _saving = false;
        _error = ApiClient.errorMessage(e);
      });
    }
  }

  Future<void> _toggleTwoFactor(bool enable) async {
    setState(() => _twofaBusy = true);
    try {
      if (enable) {
        final setup = await ref.read(apiProvider).dio.post('/auth/2fa/setup');
        if (!mounted) return;
        final confirmed = await _showTwoFactorSetupSheet(
          secret: setup.data['secret'] as String,
          otpauthUrl: setup.data['otpauthUrl'] as String,
        );
        if (confirmed) await ref.read(meProvider.notifier).refresh();
      } else {
        final code = await _promptForCode(
          title: 'Disable two-factor authentication',
          confirmLabel: 'Disable',
        );
        if (code == null) return;
        final res =
            await ref.read(apiProvider).dio.post('/auth/2fa/disable', data: {'code': code});
        await ref.read(localStoreProvider).setToken(res.data['token'] as String);
        await ref.read(meProvider.notifier).refresh();
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
      }
    } finally {
      if (mounted) setState(() => _twofaBusy = false);
    }
  }

  /// Shows the QR + manual-entry secret, then a code field to confirm setup
  /// via POST /auth/2fa/enable. Returns true once enabling succeeds.
  Future<bool> _showTwoFactorSetupSheet({
    required String secret,
    required String otpauthUrl,
  }) async {
    final codeCtrl = TextEditingController();
    var enabling = false;
    String? error;
    final confirmed = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setSheetState) => Padding(
          padding: EdgeInsets.only(
            left: 20,
            right: 20,
            top: 20,
            bottom: MediaQuery.of(ctx).viewInsets.bottom + 20,
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text('Scan with your authenticator app',
                  style: Theme.of(ctx).textTheme.titleMedium),
              const SizedBox(height: 16),
              Center(
                child: QrImageView(data: otpauthUrl, size: 200),
              ),
              const SizedBox(height: 12),
              Text('Or enter this code manually:',
                  style: Theme.of(ctx)
                      .textTheme
                      .bodySmall
                      ?.copyWith(color: SpottoColors.muted)),
              const SizedBox(height: 4),
              SelectableText(secret,
                  style: Theme.of(ctx)
                      .textTheme
                      .titleSmall
                      ?.copyWith(letterSpacing: 1.2)),
              const SizedBox(height: 20),
              profField(ctx, '6-digit code', codeCtrl, Icons.pin_outlined),
              if (error != null) ...[
                const SizedBox(height: 8),
                Text(error!, style: TextStyle(color: Theme.of(ctx).colorScheme.error)),
              ],
              const SizedBox(height: 16),
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: enabling
                      ? null
                      : () async {
                          setSheetState(() {
                            enabling = true;
                            error = null;
                          });
                          try {
                            await ref
                                .read(apiProvider)
                                .dio
                                .post('/auth/2fa/enable', data: {'code': codeCtrl.text});
                            if (ctx.mounted) Navigator.of(ctx).pop(true);
                          } catch (e) {
                            setSheetState(() {
                              enabling = false;
                              error = ApiClient.errorMessage(e);
                            });
                          }
                        },
                  child: Text(enabling ? 'Enabling…' : 'Enable'),
                ),
              ),
            ],
          ),
        ),
      ),
    );
    return confirmed ?? false;
  }

  /// Prompts for a 6-digit code (used to confirm disabling 2FA). Returns
  /// null if the user cancels.
  Future<String?> _promptForCode({required String title, required String confirmLabel}) {
    final codeCtrl = TextEditingController();
    return showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(title),
        content: profField(ctx, '6-digit code', codeCtrl, Icons.pin_outlined),
        actions: [
          TextButton(onPressed: () => Navigator.of(ctx).pop(), child: const Text('Cancel')),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(codeCtrl.text),
            child: Text(confirmLabel),
          ),
        ],
      ),
    );
  }

  ({int score, String label, Color color}) get _strength {
    final v = _pw.text;
    var s = 0;
    if (v.length >= 8) s++;
    if (v.contains(RegExp(r'[A-Z]')) && v.contains(RegExp(r'[a-z]'))) s++;
    if (v.contains(RegExp(r'\d'))) s++;
    if (v.contains(RegExp(r'[^A-Za-z0-9]'))) s++;
    const map = [
      (label: 'Too short', color: Color(0xFFCBD5D3)),
      (label: 'Weak', color: Color(0xFFE11D48)),
      (label: 'Fair', color: Color(0xFFEAA200)),
      (label: 'Good', color: SpottoColors.accent),
      (label: 'Strong', color: Color(0xFF16A34A)),
    ];
    return (score: s, label: map[s].label, color: map[s].color);
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final st = _strength;
    return _SubPage(
      title: 'Password & security',
      onBack: widget.onBack,
      children: [
        sectionCard(context, 'Change password', [
          profField(context, 'Current password', _current, Icons.lock_outline,
              obscure: true),
          const SizedBox(height: 14),
          profField(context, 'New password', _pw, Icons.lock_outline,
              obscure: true, onChanged: () => setState(() {})),
          if (_pw.text.isNotEmpty) ...[
            const SizedBox(height: 8),
            Row(
              children: [
                for (var i = 0; i < 4; i++)
                  Expanded(
                    child: Container(
                      height: 5,
                      margin: EdgeInsets.only(right: i < 3 ? 4 : 0),
                      decoration: BoxDecoration(
                        color: i < st.score ? st.color : SpottoColors.line,
                        borderRadius: BorderRadius.circular(999),
                      ),
                    ),
                  ),
                const SizedBox(width: 10),
                Text(st.label, style: text.labelSmall?.copyWith(color: st.color)),
              ],
            ),
          ],
          const SizedBox(height: 14),
          profField(context, 'Confirm new password', _pw2, Icons.lock_outline,
              obscure: true, onChanged: () => setState(() {})),
          if (_error != null) ...[
            const SizedBox(height: 8),
            Text(_error!, style: TextStyle(color: Theme.of(context).colorScheme.error)),
          ],
          const SizedBox(height: 16),
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              style: _saved ? FilledButton.styleFrom(backgroundColor: const Color(0xFF16A34A)) : null,
              onPressed: !_saving && _pw.text.isNotEmpty && _pw.text == _pw2.text && _current.text.isNotEmpty
                  ? _changePassword
                  : null,
              child: Text(_saved ? 'Password updated' : (_saving ? 'Updating…' : 'Update password')),
            ),
          ),
        ]),
        const SizedBox(height: 22),
        sectionCard(context, 'Two-factor authentication', [
          Row(
            children: [
              iconTile(Icons.verified_user_outlined),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('Authenticator app', style: text.titleSmall),
                    Text('Extra code at sign-in',
                        style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                  ],
                ),
              ),
              _twofaBusy
                  ? const SizedBox(
                      width: 24,
                      height: 24,
                      child: CircularProgressIndicator(strokeWidth: 2.5),
                    )
                  : Switch(
                      value: ref.watch(meProvider)?['totpEnabled'] as bool? ?? false,
                      activeTrackColor: SpottoColors.accent,
                      onChanged: _toggleTwoFactor,
                    ),
            ],
          ),
        ]),
        const SizedBox(height: 22),
        sectionCard(context, 'Devices', [
          for (final (i, d) in const [
            (name: 'Pixel 9', meta: 'Weligama · Active now', icon: Icons.smartphone, current: true),
            (name: 'MacBook Air · Safari', meta: 'Colombo · Yesterday', icon: Icons.laptop_mac, current: false),
          ].indexed) ...[
            if (i > 0) const Divider(height: 1),
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 10),
              child: Row(
                children: [
                  iconTile(d.icon),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(d.name, style: text.titleSmall),
                        Text(d.meta,
                            style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                      ],
                    ),
                  ),
                  if (d.current)
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                      decoration: BoxDecoration(
                        color: SpottoColors.accentTint,
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: Text('THIS DEVICE',
                          style: text.labelSmall
                              ?.copyWith(fontSize: 9, color: SpottoColors.accent)),
                    ),
                ],
              ),
            ),
          ],
        ]),
      ],
    );
  }
}

class PaymentPage extends StatelessWidget {
  final VoidCallback onBack;
  const PaymentPage({super.key, required this.onBack});

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return _SubPage(
      title: 'Saved payment',
      onBack: onBack,
      children: [
        // featured card
        Container(
          padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(
            gradient: const LinearGradient(
              begin: Alignment.topLeft,
              end: Alignment.bottomRight,
              colors: [SpottoColors.deepTeal, SpottoColors.accent],
            ),
            borderRadius: BorderRadius.circular(24),
            boxShadow: const [
              BoxShadow(color: Color(0x660B4A54), blurRadius: 30, offset: Offset(0, 14), spreadRadius: -12)
            ],
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Row(
                children: [
                  Text('VISA',
                      style: TextStyle(
                          fontSize: 16,
                          fontWeight: FontWeight.w900,
                          letterSpacing: 1,
                          color: Colors.white)),
                  Spacer(),
                  Icon(Icons.contactless_outlined, size: 20, color: Colors.white70),
                ],
              ),
              const SizedBox(height: 26),
              const Text('····  ····  ····  4291',
                  style: TextStyle(
                      fontSize: 18,
                      fontWeight: FontWeight.w800,
                      letterSpacing: 2,
                      color: Colors.white)),
              const SizedBox(height: 12),
              Row(
                children: [
                  Text('LENA MÜLLER',
                      style: text.labelSmall
                          ?.copyWith(color: Colors.white70, letterSpacing: 1)),
                  const Spacer(),
                  Text('09/28', style: text.labelSmall?.copyWith(color: Colors.white70)),
                ],
              ),
            ],
          ),
        ),
        const SizedBox(height: 22),
        sectionCard(context, 'Recent activity', [
          for (final (i, r) in const [
            (title: 'Clay & Co taster session', sub: 'Yesterday', amount: 'Rs 1,500'),
            (title: 'Driftwood day pass', sub: 'Tuesday', amount: 'Rs 1,200'),
            (title: 'Bay Surf group lesson', sub: 'Last week', amount: 'Rs 3,500'),
          ].indexed) ...[
            if (i > 0) const Divider(height: 1),
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 10),
              child: Row(
                children: [
                  iconTile(Icons.receipt_long_outlined),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(r.title, style: text.titleSmall),
                        Text(r.sub,
                            style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                      ],
                    ),
                  ),
                  Text(r.amount, style: text.labelLarge),
                ],
              ),
            ),
          ],
        ]),
        const SizedBox(height: 22),
        sectionCard(context, null, [
          profRow(context, Icons.credit_card, 'Manage payment cards', onTap: () {
            Navigator.push(context, MaterialPageRoute(builder: (_) => const PaymentMethodsScreen()));
          }),
        ]),
      ],
    );
  }
}

class PrivacyPage extends ConsumerStatefulWidget {
  final VoidCallback onBack;
  const PrivacyPage({super.key, required this.onBack});

  @override
  ConsumerState<PrivacyPage> createState() => _PrivacyPageState();
}

class _PrivacyPageState extends ConsumerState<PrivacyPage> {
  // 'location' is backed by the real /me/privacy endpoint (see meProvider
  // below); the rest have no backend field yet and stay local-only.
  final Map<String, bool> _toggles = {
    'group': true,
    'vibe': true,
    'analytics': false,
    'marketing': false,
  };
  bool _locationBusy = false;

  Future<void> _setShareLocation(bool value) async {
    setState(() => _locationBusy = true);
    try {
      await ref.read(apiProvider).dio.put('/me/privacy', data: {'shareLocation': value});
      await ref.read(meProvider.notifier).refresh();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
      }
    } finally {
      if (mounted) setState(() => _locationBusy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    Widget toggleRow(String title, String sub, IconData icon,
            {required bool value, required ValueChanged<bool> onChanged, bool busy = false}) =>
        Padding(
          padding: const EdgeInsets.symmetric(vertical: 10),
          child: Row(
            children: [
              iconTile(icon),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(title, style: text.titleSmall),
                    Text(sub, style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                  ],
                ),
              ),
              busy
                  ? const SizedBox(
                      width: 24,
                      height: 24,
                      child: CircularProgressIndicator(strokeWidth: 2.5),
                    )
                  : Switch(
                      value: value,
                      activeTrackColor: SpottoColors.accent,
                      onChanged: onChanged,
                    ),
            ],
          ),
        );

    return _SubPage(
      title: 'Privacy',
      onBack: widget.onBack,
      children: [
        sectionCard(context, 'Location & sharing', [
          toggleRow(
            'Share live location',
            'Other opted-in travellers nearby can see you on the map',
            Icons.place_outlined,
            value: ref.watch(meProvider)?['shareLocation'] as bool? ?? false,
            onChanged: _setShareLocation,
            busy: _locationBusy,
          ),
          const Divider(height: 1),
          toggleRow('Visible to group', 'Appear in the group member list',
              Icons.group_outlined,
              value: _toggles['group']!,
              onChanged: (v) => setState(() => _toggles['group'] = v)),
          const Divider(height: 1),
          toggleRow('Count my vibe votes', 'Include your votes in public vibe stats',
              Icons.monitor_heart_outlined,
              value: _toggles['vibe']!,
              onChanged: (v) => setState(() => _toggles['vibe'] = v)),
        ]),
        const SizedBox(height: 22),
        sectionCard(context, 'Data', [
          toggleRow('Usage analytics', 'Help improve Spotto with anonymous stats',
              Icons.insights_outlined,
              value: _toggles['analytics']!,
              onChanged: (v) => setState(() => _toggles['analytics'] = v)),
          const Divider(height: 1),
          toggleRow('Personalised offers', 'Deals matched to your travel style',
              Icons.campaign_outlined,
              value: _toggles['marketing']!,
              onChanged: (v) => setState(() => _toggles['marketing'] = v)),
        ]),
        const SizedBox(height: 22),
        sectionCard(context, null, [
          profRow(context, Icons.download_outlined, 'Download my data', onTap: () {}),
          const Divider(height: 1),
          profRow(context, Icons.delete_outline, 'Delete account', danger: true, onTap: () {}),
        ]),
      ],
    );
  }
}

class HelpPage extends StatelessWidget {
  final VoidCallback onBack;
  const HelpPage({super.key, required this.onBack});

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return _SubPage(
      title: 'Help & support',
      onBack: onBack,
      children: [
        sectionCard(context, 'FAQ', [
          for (final (i, f) in const [
            (q: 'How do deal codes work?', a: 'Claim a deal in the app, then show the SPOTTO code at the counter. The vendor validates it on their side — no printouts needed.'),
            (q: 'What is the Daily Quest?', a: 'Check in at two local artisan spots (like Clay & Co or the Batik atelier) in one day to unlock a free Ceylon tea.'),
            (q: 'Can friends edit my trip?', a: 'Yes — invite them from the trip planner. Collaborators can add and reorder stops; only you can confirm deletions.'),
            (q: 'Does Spotto work offline?', a: 'The map and your claimed codes are cached on your phone. Live features like group location need a connection.'),
          ].indexed) ...[
            if (i > 0) const Divider(height: 1),
            Theme(
              data: Theme.of(context).copyWith(dividerColor: Colors.transparent),
              child: ExpansionTile(
                tilePadding: EdgeInsets.zero,
                childrenPadding: const EdgeInsets.only(bottom: 12),
                title: Text(f.q, style: text.titleSmall),
                iconColor: SpottoColors.accent,
                collapsedIconColor: SpottoColors.muted,
                children: [
                  Align(
                    alignment: Alignment.centerLeft,
                    child: Text(f.a,
                        style: text.bodyMedium?.copyWith(color: SpottoColors.inkSoft)),
                  ),
                ],
              ),
            ),
          ],
        ]),
        const SizedBox(height: 22),
        sectionCard(context, 'Contact us', [
          profRow(context, Icons.chat_bubble_outline, 'Chat with support',
              sub: 'Usually replies within an hour', onTap: () {}),
          const Divider(height: 1),
          profRow(context, Icons.mail_outline, 'hello@spotto.app', onTap: () {}),
        ]),
      ],
    );
  }
}
