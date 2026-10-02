import Flutter
import GoogleMaps
import UIKit

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
    if let configured = Bundle.main.object(forInfoDictionaryKey: "GMSApiKey") as? String,
      !configured.isEmpty
    {
      mapsApiKey = configured
    }
    GMSServices.provideAPIKey(mapsApiKey)
    return super.application(application, didFinishLaunchingWithOptions: launchOptions)
  }

  func didInitializeImplicitFlutterEngine(_ engineBridge: FlutterImplicitEngineBridge) {
    GeneratedPluginRegistrant.register(with: engineBridge.pluginRegistry)
  }
}
