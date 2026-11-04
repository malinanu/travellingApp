// Profile sub-pages. Security and Privacy are wired to the real API; Help
// is static content. Saved cards live on PaymentMethodsScreen.

import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:path_provider/path_provider.dart';
import 'package:qr_flutter/qr_flutter.dart';
import 'package:share_plus/share_plus.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../../core/api/api_client.dart';
import '../../../core/theme/palette.dart';
import '../../../data/repositories/nearby_alerts_repository.dart';
import '../../../data/repositories/repositories.dart';
import '../inspiration/nearby_alerts_ui.dart';
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

  List<Map<String, dynamic>>? _sessions;
  bool _sessionsLoading = false;
  String? _revokingSessionId;

  @override
  void initState() {
    super.initState();
    _loadSessions();
  }

  @override
  void dispose() {
    _current.dispose();
    _pw.dispose();
    _pw2.dispose();
    super.dispose();
  }

  Future<void> _loadSessions() async {
    setState(() => _sessionsLoading = true);
    try {
      final res = await ref.read(apiProvider).dio.get('/me/sessions');
      if (res.data is List && mounted) {
        setState(() {
          _sessions = (res.data as List).map((e) => Map<String, dynamic>.from(e as Map)).toList();
          _sessionsLoading = false;
        });
      }
    } catch (_) {
      if (mounted) setState(() => _sessionsLoading = false);
    }
  }

  Future<void> _revokeSession(String id) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Revoke session?'),
        content: const Text('This device will be signed out immediately.'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: const Color(0xFFE11D48)),
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Sign out device'),
          ),
        ],
      ),
    );
    if (confirmed != true) return;

    setState(() => _revokingSessionId = id);
    try {
      await ref.read(apiProvider).dio.delete('/me/sessions/$id');
      await _loadSessions();
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Device signed out')),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(ApiClient.errorMessage(e))),
        );
      }
    } finally {
      if (mounted) setState(() => _revokingSessionId = null);
    }
  }

  String _formatSessionDate(String? raw) {
    if (raw == null) return '';
    try {
      final dt = DateTime.parse(raw).toLocal();
      final now = DateTime.now();
      final diff = now.difference(dt);
      if (diff.inMinutes < 5) return 'Active now';
      if (diff.inHours < 1) return '${diff.inMinutes}m ago';
      if (diff.inDays == 0) {
        return 'Today at ${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';
      }
      if (diff.inDays == 1) return 'Yesterday';
      if (diff.inDays < 7) return '${diff.inDays} days ago';
      return '${dt.day}/${dt.month}/${dt.year}';
    } catch (_) {
      return raw;
    }
  }

  IconData _deviceIcon(String client, String deviceLabel) {
    final label = deviceLabel.toLowerCase();
    if (client == 'admin' ||
        client == 'vendor' ||
        label.contains('mac') ||
        label.contains('windows') ||
        label.contains('linux') ||
        label.contains('pc')) {
      return Icons.laptop_mac;
    }
    return Icons.smartphone;
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

  /// Shows the QR + manual-entry secret with 1-tap app launch and copy buttons,
  /// then a code field to confirm setup via POST /auth/2fa/enable.
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
          child: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Set up two-factor authentication',
                    style: Theme.of(ctx).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                Text(
                  'Setting up on this device? Tap below to open your authenticator app directly, or copy the setup key. If you are using another phone or computer, you can scan the QR code.',
                  style: Theme.of(ctx).textTheme.bodySmall?.copyWith(color: SpottoColors.muted),
                ),
                const SizedBox(height: 16),
                SizedBox(
                  width: double.infinity,
                  child: OutlinedButton.icon(
                    icon: const Icon(Icons.open_in_new, size: 18),
                    label: const Text('Open in Authenticator App'),
                    onPressed: () async {
                      final uri = Uri.parse(otpauthUrl);
                      try {
                        await launchUrl(uri, mode: LaunchMode.externalApplication);
                      } catch (_) {
                        if (ctx.mounted) {
                          ScaffoldMessenger.of(ctx).showSnackBar(
                            const SnackBar(
                              content: Text('Could not open authenticator app. Please copy the key below.'),
                            ),
                          );
                        }
                      }
                    },
                  ),
                ),
                const SizedBox(height: 12),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: SpottoColors.line),
                  ),
                  child: Row(
                    children: [
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text('Setup key',
                                style: Theme.of(ctx).textTheme.labelSmall?.copyWith(color: SpottoColors.muted)),
                            SelectableText(
                              secret,
                              style: Theme.of(ctx).textTheme.titleSmall?.copyWith(letterSpacing: 1.2),
                            ),
                          ],
                        ),
                      ),
                      TextButton.icon(
                        icon: const Icon(Icons.copy, size: 16),
                        label: const Text('Copy Key'),
                        onPressed: () {
                          Clipboard.setData(ClipboardData(text: secret));
                          ScaffoldMessenger.of(ctx).showSnackBar(
                            const SnackBar(content: Text('Setup key copied to clipboard')),
                          );
                        },
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 16),
                Center(
                  child: Column(
                    children: [
                      QrImageView(data: otpauthUrl, size: 160),
                      const SizedBox(height: 4),
                      Text('Or scan QR with another device',
                          style: Theme.of(ctx).textTheme.bodySmall?.copyWith(color: SpottoColors.muted)),
                    ],
                  ),
                ),
                const SizedBox(height: 16),
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
                              final cleanCode = codeCtrl.text.replaceAll(RegExp(r'\s+|-'), '');
                              await ref
                                  .read(apiProvider)
                                  .dio
                                  .post('/auth/2fa/enable', data: {'code': cleanCode});
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
            onPressed: () => Navigator.of(ctx).pop(codeCtrl.text.replaceAll(RegExp(r'\s+|-'), '')),
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
          if (_sessionsLoading && (_sessions == null || _sessions!.isEmpty))
            const Center(
              child: Padding(
                padding: EdgeInsets.all(16),
                child: SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2)),
              ),
            )
          else if (_sessions == null || _sessions!.isEmpty)
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 12),
              child: Text('No active devices found',
                  style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
            )
          else
            for (var i = 0; i < _sessions!.length; i++) ...[
              if (i > 0) const Divider(height: 1),
              () {
                final d = _sessions![i];
                final id = d['id'] as String? ?? '';
                final isCur = d['isCurrent'] == true || d['is_current'] == true;
                final rawName = d['deviceLabel'] as String? ?? d['device_label'] as String?;
                final name = (rawName?.isNotEmpty == true)
                    ? rawName!
                    : (d['client'] == 'app' ? 'Mobile App' : 'Web Dashboard');
                final lastUsed = d['lastUsedAt'] as String? ??
                    d['last_used_at'] as String? ??
                    d['createdAt'] as String? ??
                    d['created_at'] as String?;
                final meta = isCur ? 'THIS DEVICE · Active now' : _formatSessionDate(lastUsed);
                final client = d['client'] as String? ?? 'app';
                final revoking = _revokingSessionId == id;

                return Padding(
                  padding: const EdgeInsets.symmetric(vertical: 10),
                  child: Row(
                    children: [
                      iconTile(_deviceIcon(client, name)),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(name, style: text.titleSmall),
                            Text(meta,
                                style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                          ],
                        ),
                      ),
                      if (isCur)
                        Container(
                          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                          decoration: BoxDecoration(
                            color: SpottoColors.accentTint,
                            borderRadius: BorderRadius.circular(8),
                          ),
                          child: Text('THIS DEVICE',
                              style: text.labelSmall?.copyWith(
                                  fontSize: 9,
                                  color: SpottoColors.accent,
                                  fontWeight: FontWeight.bold)),
                        )
                      else
                        revoking
                            ? const SizedBox(
                                width: 20,
                                height: 20,
                                child: CircularProgressIndicator(strokeWidth: 2),
                              )
                            : IconButton(
                                icon: const Icon(Icons.logout, size: 20, color: SpottoColors.muted),
                                tooltip: 'Revoke session',
                                onPressed: () => _revokeSession(id),
                              ),
                    ],
                  ),
                );
              }(),
            ],
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
  String? _busyField;
  bool _exporting = false;

  Future<void> _setPrivacy(String field, bool value) async {
    setState(() => _busyField = field);
    try {
      await ref.read(apiProvider).dio.put('/me/privacy', data: {field: value});
      await ref.read(meProvider.notifier).refresh();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(ApiClient.errorMessage(e))));
      }
    } finally {
      if (mounted) setState(() => _busyField = null);
    }
  }

  Future<void> _downloadMyData() async {
    if (_exporting) return;
    setState(() => _exporting = true);
    try {
      final res = await ref.read(apiProvider).dio.get('/me/export');
      final jsonStr = const JsonEncoder.withIndent('  ').convert(res.data);
      final dir = await getTemporaryDirectory();
      final timestamp = DateTime.now().toIso8601String().replaceAll(':', '-').split('.').first;
      final file = File('${dir.path}/spotto_user_data_$timestamp.json');
      await file.writeAsString(jsonStr);

      if (!mounted) return;
      final box = context.findRenderObject() as RenderBox?;
      final origin = box != null ? box.localToGlobal(Offset.zero) & box.size : null;

      await SharePlus.instance.share(
        ShareParams(
          files: [XFile(file.path, mimeType: 'application/json')],
          subject: 'Spotto Data Export',
          sharePositionOrigin: origin,
        ),
      );
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Data export downloaded successfully')),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed to download data: ${ApiClient.errorMessage(e)}')),
        );
      }
    } finally {
      if (mounted) setState(() => _exporting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final me = ref.watch(meProvider) ?? const {};

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
            value: me['shareLocation'] as bool? ?? false,
            onChanged: (v) => _setPrivacy('shareLocation', v),
            busy: _busyField == 'shareLocation',
          ),
          const Divider(height: 1),
          toggleRow(
            'Nearby saved-place alerts',
            "A heads-up when you're close to a place you saved",
            Icons.notifications_active_outlined,
            value: ref.watch(nearbyAlertsProvider).enabled,
            onChanged: (on) => on
                ? enableNearbyAlertsFlow(context, ref)
                : ref.read(nearbyAlertsProvider.notifier).disable(),
          ),
          const Divider(height: 1),
          toggleRow(
            'Visible to group',
            'Appear in the group member list',
            Icons.group_outlined,
            value: me['visibleToGroup'] as bool? ?? true,
            onChanged: (v) => _setPrivacy('visibleToGroup', v),
            busy: _busyField == 'visibleToGroup',
          ),
          const Divider(height: 1),
          toggleRow(
            'Count my vibe votes',
            'Include your votes in public vibe stats',
            Icons.monitor_heart_outlined,
            value: me['countVibeVotes'] as bool? ?? true,
            onChanged: (v) => _setPrivacy('countVibeVotes', v),
            busy: _busyField == 'countVibeVotes',
          ),
        ]),
        const SizedBox(height: 22),
        sectionCard(context, 'Data', [
          toggleRow(
            'Usage analytics',
            'Anonymous stats, including your rough area (about 1 km) when you use the map',
            Icons.insights_outlined,
            value: me['usageAnalytics'] as bool? ?? true,
            onChanged: (v) => _setPrivacy('usageAnalytics', v),
            busy: _busyField == 'usageAnalytics',
          ),
          const Divider(height: 1),
          toggleRow(
            'Personalised offers',
            'Deals matched to your travel style',
            Icons.campaign_outlined,
            value: me['personalisedOffers'] as bool? ?? false,
            onChanged: (v) => _setPrivacy('personalisedOffers', v),
            busy: _busyField == 'personalisedOffers',
          ),
        ]),
        const SizedBox(height: 22),
        sectionCard(context, null, [
          profRow(
            context,
            Icons.download_outlined,
            _exporting ? 'Preparing export…' : 'Download my data',
            onTap: _exporting ? () {} : _downloadMyData,
          ),
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
            (q: 'What is the Daily Quest?', a: "Check in at the day's number of spots to unlock its reward, then show the quest screen at the sponsor to claim it."),
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
