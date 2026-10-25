# Spotto Traveller Mobile App (Flutter)

The Spotto iOS & Android traveller app for exploring places, deals, events, vibe check-ins, local guide AI Q&A, and booking services in Sri Lanka.

## Architecture

* **State Management**: [Riverpod](https://riverpod.dev/) (`flutter_riverpod`)
* **HTTP Client**: `Dio` with custom JWT interceptors & error handling ([ApiClient](lib/core/api/api_client.dart))
* **Routing**: `GoRouter` with nested shell routes & deep linking ([app_router.dart](lib/core/router/app_router.dart))
* **Map & Location**: `google_maps_flutter` & `geolocator`
* **Local Storage**: `shared_preferences` key-value store

## Directory Structure

```
app/lib/
├── core/                  # Design system, theme, widgets, API client, router
│   ├── api/               # Dio HTTP client & Google Places service
│   ├── router/            # GoRouter configuration
│   ├── theme/             # Palette & typography tokens
│   └── widgets/           # Reusable UI components (buttons, pills, markers)
├── data/                  # Models, mock data, and Riverpod repositories
│   ├── models/            # Place, Offer, Review, Booking, User models
│   └── repositories/      # Riverpod providers & API stores
└── features/traveller/    # Feature screens
    ├── auth/              # Sign in, registration, & password reset
    ├── bookings/          # Traveller bookings management
    ├── discover/          # Interactive map & Events discovery
    ├── group/             # Group trip invites & QR sharing
    ├── guide/             # Local Guide AI chat assistant
    ├── onboarding/        # Travel style & interest preferences
    ├── place_detail/      # Place drawer, vibe check-ins, & reviews
    └── weather/           # Weather snapshot & activity rules
```

## Running Locally

1. **Prerequisites**:
   * Flutter SDK (>=3.0.0)
   * Android Studio / Xcode for emulator setup

2. **Install Dependencies**:
   ```bash
   flutter pub get
   ```

3. **Run App**:
   ```bash
   flutter run
   ```
