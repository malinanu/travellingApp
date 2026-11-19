// Money in Spotto is an integer number of minor units (cents), LKR for now.

/// "LKR 4,500", or "LKR 4,500.50" when there are cents.
String formatLkr(int minor) {
  final negative = minor < 0;
  final abs = minor.abs();
  final whole = abs ~/ 100;
  final cents = abs % 100;
  final digits = whole.toString();
  final buf = StringBuffer();
  for (var i = 0; i < digits.length; i++) {
    if (i > 0 && (digits.length - i) % 3 == 0) buf.write(',');
    buf.write(digits[i]);
  }
  final amount = cents == 0 ? buf.toString() : '$buf.${cents.toString().padLeft(2, '0')}';
  return '${negative ? '-' : ''}LKR $amount';
}
