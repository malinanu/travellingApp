// Consumer auth — login / register against the Spotto API.
// Demo account: lena@spotto.dev / spotto123 (created by the server seed).

import 'package:flutter/foundation.dart' show defaultTargetPlatform, TargetPlatform;
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:google_sign_in/google_sign_in.dart';
import 'package:sign_in_with_apple/sign_in_with_apple.dart';

import '../../../core/theme/palette.dart';
import '../../../core/widgets/illustration.dart';
import '../../../data/repositories/repositories.dart';

enum _Mode { login, register, forgot }

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
  bool _agreeTos = false;
  bool _busy = false;
  bool _forgotSent = false;
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
    if (mode == _Mode.forgot) {
      final err = await auth.forgotPassword(_email.text.trim());
      if (!mounted) return;
      if (err != null) {
        setState(() {
          _busy = false;
          _error = err;
        });
        return;
      }
      setState(() {
        _busy = false;
        _forgotSent = true;
      });
      return;
    }

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

  /// The Web OAuth client id Google ID tokens are issued for (passed as
  /// `serverClientId`; the server verifies the token's audience against it).
  /// Supplied per build with `--dart-define=GOOGLE_SERVER_CLIENT_ID=...`; the
  /// button stays hidden without it because Google sign-in can't work then.
  static const _googleServerClientId = String.fromEnvironment('GOOGLE_SERVER_CLIENT_ID');

  /// Google sign-in is offered on Android only (Apple's rules require Sign in
  /// with Apple to be offered alongside Google on iOS).
  bool get _googleSignInAvailable =>
      _googleServerClientId.isNotEmpty &&
      defaultTargetPlatform == TargetPlatform.android;

  Future<void> _signInWithGoogle() async {
    if (_busy) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    String? err;
    var cancelled = false;
    try {
      await GoogleSignIn.instance.initialize(serverClientId: _googleServerClientId);
      final account = await GoogleSignIn.instance.authenticate();
      final idToken = account.authentication.idToken;
      if (idToken == null) {
        err = 'Google did not return a sign-in token. Please try again.';
      } else {
        err = await ref.read(authProvider.notifier).googleAuth(idToken);
      }
    } on GoogleSignInException catch (e) {
      debugPrint('Google sign-in: ${e.code.name} ${e.description ?? ''}');
      // Android's Credential Manager also reports setup problems (e.g. this
      // build's signing SHA-1 not registered) as `canceled`, but with a
      // description; only a bare cancel is the user closing the sheet.
      final detail = e.description?.trim() ?? '';
      if (e.code == GoogleSignInExceptionCode.canceled && detail.isEmpty) {
        cancelled = true;
      } else {
        err = detail.isEmpty
            ? 'Google sign-in failed (${e.code.name}).'
            : 'Google sign-in failed (${e.code.name}): $detail';
      }
    } catch (e) {
      debugPrint('Google sign-in: $e');
      err = 'Google sign-in is not available right now.';
    }
    if (!mounted) return;
    if (err != null) {
      setState(() {
        _busy = false;
        _error = err;
      });
      return;
    }
    if (cancelled) {
      setState(() => _busy = false);
      return;
    }
    final hasPrefs = ref.read(prefsProvider) != null;
    context.go(hasPrefs ? '/t' : '/t/onboarding');
  }

  /// Sign in with Apple needs the capability that only a paid Apple Developer
  /// signing provides, so it's off unless the build opts in with
  /// `--dart-define=ENABLE_APPLE_SIGN_IN=true` (sideloaded free-account builds
  /// can't use it and would just show an error).
  static const _appleSignInEnabled = bool.fromEnvironment('ENABLE_APPLE_SIGN_IN');

  bool get _appleSignInAvailable =>
      _appleSignInEnabled && defaultTargetPlatform == TargetPlatform.iOS;

  Future<void> _signInWithApple() async {
    if (_busy) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    String? err;
    var cancelled = false;
    try {
      final credential = await SignInWithApple.getAppleIDCredential(
        scopes: [
          AppleIDAuthorizationScopes.email,
          AppleIDAuthorizationScopes.fullName,
        ],
      );
      final token = credential.identityToken;
      if (token == null) {
        err = 'Apple did not return a sign-in token. Please try again.';
      } else {
        // Apple only provides the name on the very first sign-in.
        final fullName = [credential.givenName, credential.familyName]
            .whereType<String>()
            .where((p) => p.isNotEmpty)
            .join(' ');
        err = await ref.read(authProvider.notifier).appleAuth(
              token,
              name: fullName.isNotEmpty ? fullName : null,
            );
      }
    } on SignInWithAppleAuthorizationException catch (e) {
      // Closing the Apple sheet isn't an error worth showing.
      if (e.code == AuthorizationErrorCode.canceled) {
        cancelled = true;
      } else {
        err = 'Sign in with Apple failed (${e.code.name}).';
      }
    } catch (e) {
      err = 'Sign in with Apple is not available right now.';
    }
    if (!mounted) return;
    if (err != null) {
      setState(() {
        _busy = false;
        _error = err;
      });
      return;
    }
    if (cancelled) {
      // User closed the Apple sheet: stay on the screen.
      setState(() => _busy = false);
      return;
    }
    final hasPrefs = ref.read(prefsProvider) != null;
    context.go(hasPrefs ? '/t' : '/t/onboarding');
  }

  @override
  Widget build(BuildContext context) {
    final text = Theme.of(context).textTheme;
    final isLogin = mode == _Mode.login;
    final isForgot = mode == _Mode.forgot;

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
                      Text('Sri Lanka, in your pocket',
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
                  Text(
                    isForgot
                        ? 'Reset password'
                        : isLogin
                            ? 'Welcome back'
                            : 'Create your account',
                    style: text.headlineSmall,
                  ),
                  const SizedBox(height: 8),
                  if (isForgot)
                    Text(
                      'We will email you instructions to reset your password.',
                      style: text.bodyMedium?.copyWith(color: SpottoColors.muted),
                    ),
                  const SizedBox(height: 16),

                  if (_forgotSent && isForgot) ...[
                    Container(
                      padding: const EdgeInsets.all(16),
                      decoration: BoxDecoration(
                        color: SpottoColors.accentTint,
                        borderRadius: BorderRadius.circular(16),
                      ),
                      child: Text(
                        'Check your inbox — password reset link sent!',
                        style: text.bodyMedium?.copyWith(
                            color: SpottoColors.ink, fontWeight: FontWeight.bold),
                      ),
                    ),
                    const SizedBox(height: 20),
                  ] else ...[
                    if (!isLogin && !isForgot) ...[
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
                    if (!isForgot) ...[
                      const SizedBox(height: 12),
                      TextField(
                          controller: _password,
                          obscureText: true,
                          onSubmitted: (_) => _submit(),
                          decoration: const InputDecoration(hintText: 'Password')),
                    ],
                    if (isLogin) ...[
                      Align(
                        alignment: Alignment.centerRight,
                        child: TextButton(
                          onPressed: () => setState(() {
                            mode = _Mode.forgot;
                            _error = null;
                            _forgotSent = false;
                          }),
                          child: Text('Forgot password?',
                              style: text.bodySmall?.copyWith(
                                color: SpottoColors.accent,
                                fontWeight: FontWeight.bold,
                              )),
                        ),
                      ),
                    ],
                    if (!isLogin && !isForgot) ...[
                      const SizedBox(height: 12),
                      Row(
                        children: [
                          SizedBox(
                            width: 24,
                            height: 24,
                            child: Checkbox(
                              value: _agreeTos,
                              onChanged: (v) => setState(() => _agreeTos = v ?? false),
                              activeColor: SpottoColors.accent,
                            ),
                          ),
                          const SizedBox(width: 8),
                          Expanded(
                            child: Text(
                              'I agree to the Terms of Service & Privacy Policy',
                              style: text.bodySmall?.copyWith(color: SpottoColors.inkSoft),
                            ),
                          ),
                        ],
                      ),
                    ],
                  ],

                  if (_error != null) ...[
                    const SizedBox(height: 12),
                    Text(_error!,
                        style: text.bodyMedium
                            ?.copyWith(color: const Color(0xFFB3261E))),
                  ],
                  const SizedBox(height: 20),
                  if (!_forgotSent || !isForgot) ...[
                    FilledButton(
                      onPressed: (_busy || (!isLogin && !isForgot && !_agreeTos)) ? null : _submit,
                      child: _busy
                          ? const SizedBox(
                              width: 18,
                              height: 18,
                              child: CircularProgressIndicator(strokeWidth: 2))
                          : Text(isForgot
                              ? 'Send reset link'
                              : isLogin
                                  ? 'Sign in'
                                  : 'Create account'),
                    ),
                    if (!isForgot && _googleSignInAvailable) ...[
                      const SizedBox(height: 12),
                      OutlinedButton.icon(
                        onPressed: _busy ? null : _signInWithGoogle,
                        icon: const Icon(Icons.g_mobiledata, size: 28, color: Colors.black),
                        label: const Text('Continue with Google',
                            style: TextStyle(color: Colors.black, fontWeight: FontWeight.bold)),
                        style: OutlinedButton.styleFrom(
                          padding: const EdgeInsets.symmetric(vertical: 12),
                          side: const BorderSide(color: Colors.black26),
                        ),
                      ),
                    ],
                    if (!isForgot && _appleSignInAvailable) ...[
                      const SizedBox(height: 12),
                      OutlinedButton.icon(
                        onPressed: _busy ? null : _signInWithApple,
                        icon: const Icon(Icons.apple, size: 20, color: Colors.black),
                        label: const Text('Sign in with Apple',
                            style: TextStyle(color: Colors.black, fontWeight: FontWeight.bold)),
                        style: OutlinedButton.styleFrom(
                          padding: const EdgeInsets.symmetric(vertical: 12),
                          side: const BorderSide(color: Colors.black26),
                        ),
                      ),
                    ],
                  ],
                  const SizedBox(height: 20),
                  Center(
                    child: TextButton(
                      onPressed: _busy
                          ? null
                          : () => setState(() {
                                mode = isForgot
                                    ? _Mode.login
                                    : isLogin
                                        ? _Mode.register
                                        : _Mode.login;
                                _error = null;
                                _forgotSent = false;
                              }),
                      child: Text(
                          isForgot
                              ? '← Back to sign in'
                              : isLogin
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
