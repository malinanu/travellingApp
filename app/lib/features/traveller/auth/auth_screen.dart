// Mock consumer auth — login / register / OTP flows from Auth.jsx, simplified:
// any input signs you in (prototype behaviour).

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/illustration.dart';
import '../../../data/repositories/repositories.dart';

enum _Mode { login, register, phone, otp }

class AuthScreen extends ConsumerStatefulWidget {
  const AuthScreen({super.key});

  @override
  ConsumerState<AuthScreen> createState() => _AuthScreenState();
}

class _AuthScreenState extends ConsumerState<AuthScreen> {
  _Mode mode = _Mode.login;

  void _finish() {
    ref.read(authProvider.notifier).signIn();
    final hasPrefs = ref.read(prefsProvider) != null;
    context.go(hasPrefs ? '/t' : '/t/onboarding');
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
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
              child: switch (mode) {
                _Mode.login => _LoginForm(
                    onSubmit: _finish,
                    onRegister: () => setState(() => mode = _Mode.register),
                    onPhone: () => setState(() => mode = _Mode.phone),
                  ),
                _Mode.register => _RegisterForm(
                    onSubmit: _finish,
                    onBack: () => setState(() => mode = _Mode.login),
                  ),
                _Mode.phone => _PhoneForm(
                    onSubmit: () => setState(() => mode = _Mode.otp),
                    onBack: () => setState(() => mode = _Mode.login),
                  ),
                _Mode.otp => _OtpForm(onSubmit: _finish),
              },
            ),
          ),
        ],
      ),
    );
  }
}

class _LoginForm extends StatelessWidget {
  final VoidCallback onSubmit;
  final VoidCallback onRegister;
  final VoidCallback onPhone;
  const _LoginForm({required this.onSubmit, required this.onRegister, required this.onPhone});

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text('Welcome back', style: text.headlineSmall),
        const SizedBox(height: 16),
        const TextField(decoration: InputDecoration(hintText: 'Email')),
        const SizedBox(height: 12),
        const TextField(obscureText: true, decoration: InputDecoration(hintText: 'Password')),
        const SizedBox(height: 20),
        FilledButton(onPressed: onSubmit, child: const Text('Sign in')),
        const SizedBox(height: 12),
        OutlinedButton.icon(
          onPressed: onPhone,
          icon: const Icon(Icons.phone_iphone),
          label: const Text('Continue with phone'),
        ),
        const SizedBox(height: 20),
        Center(
          child: TextButton(
            onPressed: onRegister,
            child: Text('New here? Create an account',
                style: text.labelLarge?.copyWith(color: SpottoColors.accent)),
          ),
        ),
      ],
    );
  }
}

class _RegisterForm extends StatelessWidget {
  final VoidCallback onSubmit;
  final VoidCallback onBack;
  const _RegisterForm({required this.onSubmit, required this.onBack});

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text('Create your account', style: text.headlineSmall),
        const SizedBox(height: 16),
        const TextField(decoration: InputDecoration(hintText: 'Name')),
        const SizedBox(height: 12),
        const TextField(decoration: InputDecoration(hintText: 'Email')),
        const SizedBox(height: 12),
        const TextField(obscureText: true, decoration: InputDecoration(hintText: 'Password')),
        const SizedBox(height: 20),
        FilledButton(onPressed: onSubmit, child: const Text('Create account')),
        TextButton(onPressed: onBack, child: const Text('Back to sign in')),
      ],
    );
  }
}

class _PhoneForm extends StatelessWidget {
  final VoidCallback onSubmit;
  final VoidCallback onBack;
  const _PhoneForm({required this.onSubmit, required this.onBack});

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text('Sign in with phone', style: text.headlineSmall),
        const SizedBox(height: 16),
        const TextField(
          keyboardType: TextInputType.phone,
          decoration: InputDecoration(hintText: '+94 7X XXX XXXX'),
        ),
        const SizedBox(height: 20),
        FilledButton(onPressed: onSubmit, child: const Text('Send code')),
        TextButton(onPressed: onBack, child: const Text('Back to sign in')),
      ],
    );
  }
}

class _OtpForm extends StatelessWidget {
  final VoidCallback onSubmit;
  const _OtpForm({required this.onSubmit});

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text('Enter the code', style: text.headlineSmall),
        const SizedBox(height: 8),
        Text('We sent a 4-digit code to your phone (any code works in this demo).',
            style: text.bodyMedium),
        const SizedBox(height: 16),
        const TextField(
          keyboardType: TextInputType.number,
          maxLength: 4,
          textAlign: TextAlign.center,
          decoration: InputDecoration(hintText: '••••', counterText: ''),
        ),
        const SizedBox(height: 20),
        FilledButton(onPressed: onSubmit, child: const Text('Verify')),
      ],
    );
  }
}
