// Flat SVG scene per spot category — port of the prototype's <Illustration/>.

import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';

class Illustration extends StatelessWidget {
  final String kind; // cowork | surf | sunset | clay | batik
  final bool scrim;

  const Illustration({super.key, required this.kind, this.scrim = true});

  static const _kinds = {'cowork', 'surf', 'sunset', 'clay', 'batik'};

  @override
  Widget build(BuildContext context) {
    final k = _kinds.contains(kind) ? kind : 'cowork';
    return Stack(
      fit: StackFit.expand,
      children: [
        SvgPicture.asset('assets/illustrations/$k.svg', fit: BoxFit.cover),
        if (scrim)
          const DecoratedBox(
            decoration: BoxDecoration(
              gradient: LinearGradient(
                begin: Alignment.topCenter,
                end: Alignment.bottomCenter,
                stops: [0.42, 1.0],
                colors: [Colors.transparent, Color(0xB8103D3A)],
              ),
            ),
          ),
      ],
    );
  }
}
