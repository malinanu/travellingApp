import 'package:flutter/material.dart';

import '../theme/palette.dart';

/// Bottom-sheet body with the prototype's grab handle + rounded top.
/// Use inside showModalBottomSheet.
class SheetScaffold extends StatelessWidget {
  final Widget child;
  final EdgeInsetsGeometry padding;

  const SheetScaffold({super.key, required this.child, this.padding = EdgeInsets.zero});

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: const BorderRadius.vertical(top: Radius.circular(28)),
      child: Container(
        color: SpottoColors.sand,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const SizedBox(height: 10),
            Container(
              width: 44,
              height: 5,
              decoration: BoxDecoration(
                color: SpottoColors.line,
                borderRadius: BorderRadius.circular(999),
              ),
            ),
            const SizedBox(height: 6),
            Flexible(child: Padding(padding: padding, child: child)),
          ],
        ),
      ),
    );
  }
}
