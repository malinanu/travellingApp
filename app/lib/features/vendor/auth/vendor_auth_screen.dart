// Vendor auth — mobile take on VendorAuth.jsx: login + 3-step register.
// Any input signs in (mock).

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/press_scale.dart';
import '../../../core/widgets/spotto_icons.dart';
import '../../../data/mock/vendor_seed.dart';
import '../../../data/repositories/repositories.dart';

class VendorAuthScreen extends ConsumerStatefulWidget {
  const VendorAuthScreen({super.key});

  @override
  ConsumerState<VendorAuthScreen> createState() => _VendorAuthScreenState();
}

class _VendorAuthScreenState extends ConsumerState<VendorAuthScreen> {
  bool _register = false;
  int _step = 0;
  final _email = TextEditingController();
  final _password = TextEditingController();
  final _bizName = TextEditingController();
  final _owner = TextEditingController();
  String _typeId = 'cafe';

  @override
  void dispose() {
    _email.dispose();
    _password.dispose();
    _bizName.dispose();
    _owner.dispose();
    super.dispose();
  }

  void _signIn() {
    ref.read(roleProvider.notifier).set('vendor');
    ref.read(vendorAuthProvider.notifier).signIn();
    context.go('/v');
  }

  void _registerDone() {
    final type = businessTypes.firstWhere((t) => t.id == _typeId);
    final name = _bizName.text.trim();
    final initials = name.isEmpty
        ? 'NB'
        : name
            .split(' ')
            .where((w) => w.isNotEmpty)
            .map((w) => w[0])
            .take(2)
            .join()
            .toUpperCase();
    ref.read(roleProvider.notifier).set('vendor');
    ref.read(vendorAuthProvider.notifier).signIn(VendorBusiness(
          name: name.isEmpty ? 'New Business' : name,
          type: type.label,
          typeId: type.id,
          area: 'Weligama',
          initials: initials,
          owner: _owner.text.trim().isEmpty ? 'Owner' : _owner.text.trim(),
        ));
    context.go('/v');
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return Scaffold(
      backgroundColor: SpottoColors.sand,
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: Column(
              children: [
                // brand header
                Container(
                  width: 64,
                  height: 64,
                  decoration: BoxDecoration(
                    gradient: const LinearGradient(
                      begin: Alignment.topLeft,
                      end: Alignment.bottomRight,
                      colors: [SpottoColors.accent, SpottoColors.accent2],
                    ),
                    borderRadius: BorderRadius.circular(20),
                    boxShadow: const [
                      BoxShadow(color: Color(0x660B6F86), blurRadius: 26, offset: Offset(0, 12), spreadRadius: -10)
                    ],
                  ),
                  child: const Icon(Icons.storefront, size: 30, color: Colors.white),
                ),
                const SizedBox(height: 16),
                Text('Spotto for Business', style: text.headlineSmall),
                const SizedBox(height: 4),
                Text('Reach the travellers already exploring Weligama.',
                    textAlign: TextAlign.center,
                    style: text.bodyMedium?.copyWith(color: SpottoColors.muted)),
                const SizedBox(height: 24),
                Container(
                  padding: const EdgeInsets.all(20),
                  decoration: BoxDecoration(
                    color: Colors.white,
                    borderRadius: BorderRadius.circular(24),
                    border: Border.all(color: SpottoColors.line),
                    boxShadow: const [
                      BoxShadow(color: Color(0x1F103D3A), blurRadius: 24, offset: Offset(0, 8), spreadRadius: -10)
                    ],
                  ),
                  child: _register ? _registerCard(text) : _loginCard(text),
                ),
                const SizedBox(height: 16),
                TextButton(
                  onPressed: () => setState(() {
                    _register = !_register;
                    _step = 0;
                  }),
                  child: Text(_register
                      ? 'Already on Spotto? Sign in'
                      : 'New here? Register your business'),
                ),
                TextButton(
                  onPressed: () => context.go('/'),
                  child: const Text('← Back to role selection'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _loginCard(TextTheme text) => Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Sign in', style: text.titleMedium?.copyWith(fontSize: 18)),
          const SizedBox(height: 16),
          TextField(
            controller: _email,
            keyboardType: TextInputType.emailAddress,
            decoration: const InputDecoration(
              hintText: 'Business email',
              prefixIcon: Icon(Icons.mail_outline, size: 18, color: SpottoColors.muted),
            ),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _password,
            obscureText: true,
            decoration: const InputDecoration(
              hintText: 'Password',
              prefixIcon: Icon(Icons.lock_outline, size: 18, color: SpottoColors.muted),
            ),
          ),
          const SizedBox(height: 8),
          Align(
            alignment: Alignment.centerRight,
            child: TextButton(
              onPressed: () => ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('Reset link sent (demo)'))),
              child: const Text('Forgot password?'),
            ),
          ),
          const SizedBox(height: 4),
          SizedBox(
            width: double.infinity,
            child: FilledButton(onPressed: _signIn, child: const Text('Sign in')),
          ),
          const SizedBox(height: 10),
          Center(
            child: Text('Demo: any details sign you into Driftwood Coffee.',
                style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
          ),
        ],
      );

  Widget _registerCard(TextTheme text) {
    final steps = ['Business', 'Type', 'Confirm'];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            for (final (i, s) in steps.indexed) ...[
              if (i > 0)
                Expanded(
                  child: Container(
                    height: 2,
                    margin: const EdgeInsets.symmetric(horizontal: 6),
                    color: i <= _step ? SpottoColors.accent : SpottoColors.line,
                  ),
                ),
              Column(
                children: [
                  Container(
                    width: 26,
                    height: 26,
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      color: i <= _step ? SpottoColors.accent : SpottoColors.sand,
                      shape: BoxShape.circle,
                    ),
                    child: i < _step
                        ? const Icon(Icons.check, size: 14, color: Colors.white)
                        : Text('${i + 1}',
                            style: TextStyle(
                                fontSize: 12,
                                fontWeight: FontWeight.w800,
                                color:
                                    i <= _step ? Colors.white : SpottoColors.muted)),
                  ),
                  const SizedBox(height: 4),
                  Text(s,
                      style: text.labelSmall?.copyWith(
                          color: i <= _step ? SpottoColors.accent : SpottoColors.muted)),
                ],
              ),
            ],
          ],
        ),
        const SizedBox(height: 20),
        switch (_step) {
          0 => Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                TextField(
                  controller: _bizName,
                  decoration: const InputDecoration(
                    hintText: 'Business name',
                    prefixIcon:
                        Icon(Icons.storefront, size: 18, color: SpottoColors.muted),
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _owner,
                  decoration: const InputDecoration(
                    hintText: 'Owner name',
                    prefixIcon:
                        Icon(Icons.person_outline, size: 18, color: SpottoColors.muted),
                  ),
                ),
              ],
            ),
          1 => GridView.count(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              crossAxisCount: 2,
              mainAxisSpacing: 10,
              crossAxisSpacing: 10,
              childAspectRatio: 2.6,
              children: [
                for (final t in businessTypes)
                  PressScale(
                    scale: 0.97,
                    onTap: () => setState(() => _typeId = t.id),
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 12),
                      decoration: BoxDecoration(
                        color: _typeId == t.id ? SpottoColors.accent : SpottoColors.sand,
                        borderRadius: BorderRadius.circular(14),
                      ),
                      child: Row(
                        children: [
                          Icon(spottoIcon(t.icon),
                              size: 18,
                              color: _typeId == t.id ? Colors.white : SpottoColors.ink),
                          const SizedBox(width: 8),
                          Expanded(
                            child: Text(t.label,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: TextStyle(
                                    fontSize: 12.5,
                                    fontWeight: FontWeight.w800,
                                    color: _typeId == t.id
                                        ? Colors.white
                                        : SpottoColors.ink)),
                          ),
                        ],
                      ),
                    ),
                  ),
              ],
            ),
          _ => Container(
              padding: const EdgeInsets.all(14),
              decoration: BoxDecoration(
                color: SpottoColors.sand,
                borderRadius: BorderRadius.circular(16),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(_bizName.text.trim().isEmpty ? 'New Business' : _bizName.text.trim(),
                      style: text.titleMedium),
                  const SizedBox(height: 2),
                  Text(
                      '${businessTypes.firstWhere((t) => t.id == _typeId).label} · Weligama · '
                      '${_owner.text.trim().isEmpty ? 'Owner' : _owner.text.trim()}',
                      style: text.bodySmall?.copyWith(color: SpottoColors.muted)),
                  const SizedBox(height: 8),
                  Text('Your listing goes live after a quick review (instant in the demo).',
                      style: text.bodySmall?.copyWith(color: SpottoColors.inkSoft)),
                ],
              ),
            ),
        },
        const SizedBox(height: 20),
        Row(
          children: [
            if (_step > 0)
              Expanded(
                child: OutlinedButton(
                  onPressed: () => setState(() => _step--),
                  child: const Text('Back'),
                ),
              ),
            if (_step > 0) const SizedBox(width: 10),
            Expanded(
              flex: 2,
              child: FilledButton(
                onPressed: () =>
                    _step < 2 ? setState(() => _step++) : _registerDone(),
                child: Text(_step < 2 ? 'Continue' : 'Create my listing'),
              ),
            ),
          ],
        ),
      ],
    );
  }
}
