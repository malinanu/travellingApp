// Analytics & Crash Reporting service for Spotto.
// Standardized logger for errors, user flow events, screen views, and diagnostic telemetry.
// Integrates with FlutterError.onError and PlatformDispatcher.instance.onError for
// zero-leak crash capturing across native & Dart runtime.

import 'dart:developer' as developer;
import 'package:flutter/foundation.dart';

enum AnalyticsEvent {
  screenView,
  tripCreated,
  inspirationAdded,
  checkInCompleted,
  offlineTilesDownloaded,
  searchPerformed,
  errorOccurred,
}

class AnalyticsService {
  static final AnalyticsService instance = AnalyticsService._();
  AnalyticsService._();

  bool _initialized = false;
  final List<Map<String, dynamic>> _bufferedEvents = [];

  /// Initializes global crash reporting hooks for Dart and Flutter errors.
  void initialize() {
    if (_initialized) return;
    _initialized = true;

    // Capture uncaught Flutter framework errors (widget build, layout failures)
    final originalOnError = FlutterError.onError;
    FlutterError.onError = (FlutterErrorDetails details) {
      logCrash(details.exception, details.stack, context: details.context?.toString());
      originalOnError?.call(details);
    };

    // Capture asynchronous uncaught Dart errors outside Flutter's build scope
    PlatformDispatcher.instance.onError = (Object error, StackTrace stack) {
      logCrash(error, stack, context: 'PlatformDispatcher.onError');
      return true;
    };

    developer.log('[AnalyticsService] Crash reporting & telemetry initialized');
  }

  /// Logs a structured analytics event.
  void logEvent(AnalyticsEvent event, {Map<String, dynamic>? properties}) {
    final payload = {
      'event': event.name,
      'timestamp': DateTime.now().toIso8601String(),
      ...?properties,
    };

    _bufferedEvents.add(payload);
    if (_bufferedEvents.length > 100) {
      _bufferedEvents.removeAt(0); // Bounded ring buffer for memory safety
    }

    if (kDebugMode) {
      developer.log('[Analytics] ${event.name}: $properties');
    }
  }

  /// Logs a crash or error with stack trace.
  void logCrash(Object exception, StackTrace? stackTrace, {String? context}) {
    final errorRecord = {
      'type': 'crash',
      'context': context ?? 'general',
      'exception': exception.toString(),
      'stack': stackTrace?.toString(),
      'timestamp': DateTime.now().toIso8601String(),
    };

    _bufferedEvents.add(errorRecord);

    if (kDebugMode) {
      developer.log(
        '[CrashReport] ($context) ${exception.toString()}',
        error: exception,
        stackTrace: stackTrace,
      );
    }
  }

  /// Returns recent in-memory telemetry buffer for diagnostics.
  List<Map<String, dynamic>> getRecentEvents() => List.unmodifiable(_bufferedEvents);
}
