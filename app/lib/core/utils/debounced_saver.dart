import 'dart:async';

/// Debounced auto-save: call [schedule] on every edit, and it fires [save]
/// once ~[delay] after the last call with no further edits — never
/// synchronously inside a `setState`/state-update call, only from the
/// timer's own callback, by construction. Generalizes the one existing
/// precedent in this codebase, `TripNotifier._schedulePush` (see
/// `data/repositories/repositories.dart`), which does the same
/// cancel-and-reschedule dance inline for trip updates.
///
/// Call [flush] to cancel any pending debounce and save immediately —
/// wire it to a field losing focus (`FocusNode` listener) so leaving a
/// field never waits out the debounce, and to any manual "Save" button
/// kept as a fallback. Call [dispose] from the owning State's `dispose()`.
class DebouncedSaver {
  DebouncedSaver({required this.save, this.delay = const Duration(milliseconds: 900)});

  final FutureOr<void> Function() save;
  final Duration delay;
  Timer? _timer;

  void schedule() {
    _timer?.cancel();
    _timer = Timer(delay, () {
      _timer = null;
      save();
    });
  }

  Future<void> flush() async {
    _timer?.cancel();
    _timer = null;
    await save();
  }

  void dispose() {
    _timer?.cancel();
    _timer = null;
  }
}
