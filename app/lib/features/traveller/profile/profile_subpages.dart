// Profile sub-pages (all mock/local state): security, payment, privacy, help.

import 'package:flutter/material.dart';

import '../../../core/theme/palette.dart';
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

class SecurityPage extends StatefulWidget {
  final VoidCallback onBack;
  const SecurityPage({super.key, required this.onBack});

  @override
  State<SecurityPage> createState() => _SecurityPageState();
}

class _SecurityPageState extends State<SecurityPage> {
  final _current = TextEditingController();
  final _pw = TextEditingController();
  final _pw2 = TextEditingController();
  bool _twofa = true;
  bool _saved = false;

  @override
  void dispose() {
    _current.dispose();
    _pw.dispose();
    _pw2.dispose();
    super.dispose();
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
          const SizedBox(height: 16),
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              style: _saved ? FilledButton.styleFrom(backgroundColor: const Color(0xFF16A34A)) : null,
              onPressed: _pw.text.isNotEmpty && _pw.text == _pw2.text
                  ? () {
                      setState(() => _saved = true);
                      _current.clear();
                      _pw.clear();
                      _pw2.clear();
                      Future.delayed(const Duration(milliseconds: 1800), () {
                        if (mounted) setState(() => _saved = false);
                      });
                    }
                  : null,
              child: Text(_saved ? 'Password updated' : 'Update password'),
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
              Switch(
                value: _twofa,
                activeTrackColor: SpottoColors.accent,
                onChanged: (v) => setState(() => _twofa = v),
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
          profRow(context, Icons.add_card, 'Add payment method', onTap: () {}),
        ]),
      ],
    );
  }
}

class PrivacyPage extends StatefulWidget {
  final VoidCallback onBack;
  const PrivacyPage({super.key, required this.onBack});

  @override
  State<PrivacyPage> createState() => _PrivacyPageState();
}

class _PrivacyPageState extends State<PrivacyPage> {
  final Map<String, bool> _toggles = {
    'location': true,
    'group': true,
    'vibe': true,
    'analytics': false,
    'marketing': false,
  };

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    Widget toggleRow(String key, IconData icon, String title, String sub) => Padding(
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
              Switch(
                value: _toggles[key] ?? false,
                activeTrackColor: SpottoColors.accent,
                onChanged: (v) => setState(() => _toggles[key] = v),
              ),
            ],
          ),
        );

    return _SubPage(
      title: 'Privacy',
      onBack: widget.onBack,
      children: [
        sectionCard(context, 'Location & sharing', [
          toggleRow('location', Icons.place_outlined, 'Share live location',
              'Friends in your group can see you on the map'),
          const Divider(height: 1),
          toggleRow('group', Icons.group_outlined, 'Visible to group',
              'Appear in the group member list'),
          const Divider(height: 1),
          toggleRow('vibe', Icons.monitor_heart_outlined, 'Count my vibe votes',
              'Include your votes in public vibe stats'),
        ]),
        const SizedBox(height: 22),
        sectionCard(context, 'Data', [
          toggleRow('analytics', Icons.insights_outlined, 'Usage analytics',
              'Help improve Spotto with anonymous stats'),
          const Divider(height: 1),
          toggleRow('marketing', Icons.campaign_outlined, 'Personalised offers',
              'Deals matched to your travel style'),
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
