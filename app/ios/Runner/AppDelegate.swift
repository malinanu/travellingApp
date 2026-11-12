import Flutter
import GoogleMaps
import UIKit
import UserNotifications
import native_geofence

@main
@objc class AppDelegate: FlutterAppDelegate, FlutterImplicitEngineDelegate {
  override func application(
    _ application: UIApplication,
    didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
  ) -> Bool {
    // Comes from GMSApiKey in Info.plist ($(MAPS_API_KEY), see
    // ios/Flutter/Secrets.xcconfig.example). The Maps SDK raises an
    // uncatchable exception — killing the app the moment the first map is
    // created — if provideAPIKey was never called, so always call it. With no
    // real key (e.g. a CI build without the IOS_MAPS_API_KEY secret) a
    // placeholder keeps the app alive; the map just won't load tiles.
    var mapsApiKey = "MISSING_MAPS_API_KEY"
    if let raw = Bundle.main.object(forInfoDictionaryKey: "GMSApiKey") as? String {
      let cleaned = raw
        .trimmingCharacters(in: .whitespacesAndNewlines)
        .trimmingCharacters(in: CharacterSet(charactersIn: "\"'"))
      if !cleaned.isEmpty, !cleaned.hasPrefix("$(") {
        mapsApiKey = cleaned
      }
    }
    let provided = GMSServices.provideAPIKey(mapsApiKey)
    let bundleId = Bundle.main.bundleIdentifier ?? "unknown"
    if mapsApiKey == "MISSING_MAPS_API_KEY" {
      NSLog("[Spotto iOS] ⚠️ GMSApiKey is MISSING or unresolved in Info.plist (bundleId=%@)! Google Maps tiles will not load. Ensure MAPS_API_KEY is configured in Secrets.xcconfig.", bundleId)
    } else {
      NSLog("[Spotto iOS] ✅ Google Maps SDK initialized (bundleId=%@, prefix=%@..., len=%d, ok=%@)", bundleId, String(mapsApiKey.prefix(4)), mapsApiKey.count, provided ? "true" : "false")
    }
    // Nearby saved-place alerts: the geofence callback runs in a background
    // engine, which needs the app's plugins (e.g. local notifications).
    NativeGeofencePlugin.setPluginRegistrantCallback { registry in
      GeneratedPluginRegistrant.register(with: registry)
    }
    // Show local notifications while the app is in the foreground too.
    UNUserNotificationCenter.current().delegate = self
    return super.application(application, didFinishLaunchingWithOptions: launchOptions)
  }

  func didInitializeImplicitFlutterEngine(_ engineBridge: FlutterImplicitEngineBridge) {
    GeneratedPluginRegistrant.register(with: engineBridge.pluginRegistry)
  }
}
