package com.spotto.spotto

import com.ryanheise.audioservice.AudioServiceActivity

// AudioServiceActivity (a FlutterActivity) shares the Flutter engine with
// audio_service, so Stories audio keeps playing in the background.
class MainActivity : AudioServiceActivity()
