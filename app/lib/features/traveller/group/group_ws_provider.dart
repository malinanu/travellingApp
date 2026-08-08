import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:google_maps_flutter/google_maps_flutter.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import '../../../core/api/api_client.dart';
import '../../../data/repositories/repositories.dart';

class GroupLiveLocationsNotifier extends Notifier<Map<String, LatLng>> {
  WebSocketChannel? _channel;

  @override
  Map<String, LatLng> build() {
    ref.onDispose(() => _channel?.sink.close());

    final me = ref.watch(meProvider);
    final group = ref.watch(groupProvider);
    
    // Disconnect if we leave the group
    if (me == null || group.memberIds.isEmpty) {
      _channel?.sink.close();
      _channel = null;
      return {};
    }

    final myId = me['id'] as String;
    
    // Only connect once per group config
    if (_channel == null) {
      _connect(group.code, myId);
    }
    
    return state;
  }

  void _connect(String groupId, String myId) {
    try {
      final wsUrl = apiBaseUrl.replaceFirst('http', 'ws') + '/ws/group?groupId=$groupId&userId=$myId';
      _channel = WebSocketChannel.connect(Uri.parse(wsUrl));
      
      _channel!.stream.listen(
        (message) {
          try {
            final data = jsonDecode(message as String);
            final userId = data['userId'] as String;
            final lat = (data['lat'] as num).toDouble();
            final lng = (data['lng'] as num).toDouble();
            
            state = {
              ...state,
              userId: LatLng(lat, lng),
            };
          } catch (e) {
            debugPrint('Failed to parse WS message: $e');
          }
        },
        onError: (err) {
          debugPrint('Group WS Error: $err');
        },
        onDone: () {
          _channel = null;
        },
      );
    } catch (e) {
      debugPrint('Group WS Connection Error: $e');
    }
  }

  void sendLocation(LatLng pos) {
    if (_channel != null) {
      _channel!.sink.add(jsonEncode({
        'lat': pos.latitude,
        'lng': pos.longitude,
      }));
    }
  }
}

final groupLiveLocationsProvider =
    NotifierProvider<GroupLiveLocationsNotifier, Map<String, LatLng>>(GroupLiveLocationsNotifier.new);
