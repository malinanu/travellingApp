// Id → Traveler lookup pools for tripWho() (trip_common.dart) and the
// trip-share sheet. Both start empty and are filled from real data only:
// `travelers` from group members and nearby travellers (GroupNotifier,
// NearbyTravellersNotifier), `me` from /me (meProvider).

import '../models/models.dart';

var travelers = <Traveler>[];

var me = Traveler.fromMe(null);
