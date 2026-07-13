// Consumer auth — login / register against the Spotto API.
// Demo account: lena@spotto.dev / spotto123 (created by the server seed).

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/illustration.dart';
import '../../../data/repositories/repositories.dart';

enum _Mode { login, register }

class AuthScreen extends ConsumerStatefulWidget {
  const AuthScreen({super.key});

  @override
  ConsumerState<AuthScreen> createState() => _AuthScreenState();
}

class _AuthScreenState extends ConsumerState<AuthScreen> {
  _Mode mode = _Mode.login;
  final _name = TextEditingController();
  final _email = TextEditingController();
  final _password = TextEditingController();
  bool _busy = false;
  String? _error;

  @override
  void dispose() {
    _name.dispose();
    _email.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    if (_busy) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    final auth = ref.read(authProvider.notifier);
    final err = mode == _Mode.login
        ? await auth.login(_email.text.trim(), _password.text)
        : await auth.register(_name.text.trim(), _email.text.trim(), _password.text);
    if (!mounted) return;
    if (err != null) {
      setState(() {
        _busy = false;
        _error = err;
      });
      return;
    }
    final hasPrefs = ref.read(prefsProvider) != null;
    context.go(hasPrefs ? '/t' : '/t/onboarding');
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final isLogin = mode == _Mode.login;
    return Scaffold(
      body: Column(
        children: [
          SizedBox(
            height: 240,
            width: double.infinity,
            child: Stack(
              fit: StackFit.expand,
              children: [
                const Illustration(kind: 'sunset'),
                Positioned(
                  left: 24,
                  bottom: 20,
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text('Spotto',
                          style: text.displaySmall?.copyWith(color: Colors.white)),
                      Text('Weligama, in your pocket',
                          style: text.bodyLarge
                              ?.copyWith(color: Colors.white.withValues(alpha: 0.9))),
                    ],
                  ),
                ),
              ],
            ),
          ),
          Expanded(
            child: SingleChildScrollView(
              padding: const EdgeInsets.all(24),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(isLogin ? 'Welcome back' : 'Create your account',
                      style: text.headlineSmall),
                  const SizedBox(height: 16),
                  if (!isLogin) ...[
                    TextField(
                        controller: _name,
                        textCapitalization: TextCapitalization.words,
                        decoration: const InputDecoration(hintText: 'Name')),
                    const SizedBox(height: 12),
                  ],
                  TextField(
                      controller: _email,
                      keyboardType: TextInputType.emailAddress,
                      autocorrect: false,
                      decoration: const InputDecoration(hintText: 'Email')),
                  const SizedBox(height: 12),
                  TextField(
                      controller: _password,
                      obscureText: true,
                      onSubmitted: (_) => _submit(),
                      decoration: const InputDecoration(hintText: 'Password')),
                  if (_error != null) ...[
                    const SizedBox(height: 12),
                    Text(_error!,
                        style: text.bodyMedium
                            ?.copyWith(color: const Color(0xFFB3261E))),
                  ],
                  const SizedBox(height: 20),
                  FilledButton(
                    onPressed: _busy ? null : _submit,
                    child: _busy
                        ? const SizedBox(
                            width: 18,
                            height: 18,
                            child: CircularProgressIndicator(strokeWidth: 2))
                        : Text(isLogin ? 'Sign in' : 'Create account'),
                  ),
                  const SizedBox(height: 20),
                  Center(
                    child: TextButton(
                      onPressed: _busy
                          ? null
                          : () => setState(() {
                                mode = isLogin ? _Mode.register : _Mode.login;
                                _error = null;
                              }),
                      child: Text(
                          isLogin
                              ? 'New here? Create an account'
                              : 'Have an account? Sign in',
                          style: text.labelLarge
                              ?.copyWith(color: SpottoColors.accent)),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}
