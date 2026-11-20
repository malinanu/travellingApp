// Money in Spotto is an integer number of minor units (cents). Bookings are
// priced in LKR; some payment gateways charge another currency.

String _groupedWhole(int whole) {
  final digits = whole.toString();
  final buf = StringBuffer();
  for (var i = 0; i < digits.length; i++) {
    if (i > 0 && (digits.length - i) % 3 == 0) buf.write(',');
    buf.write(digits[i]);
  }
  return buf.toString();
}

/// "LKR 4,500", or "LKR 4,500.50" when there are cents.
String formatLkr(int minor) {
  final abs = minor.abs();
  final cents = abs % 100;
  final whole = _groupedWhole(abs ~/ 100);
  final amount = cents == 0 ? whole : '$whole.${cents.toString().padLeft(2, '0')}';
  return '${minor < 0 ? '-' : ''}LKR $amount';
}

/// Minor units of any two-decimal currency: "USD 16.48", "USD 16.00".
/// LKR keeps formatLkr's style.
String formatMoney(int minor, String currency) {
  if (currency == 'LKR') return formatLkr(minor);
  final abs = minor.abs();
  return '${minor < 0 ? '-' : ''}$currency ${_groupedWhole(abs ~/ 100)}.${(abs % 100).toString().padLeft(2, '0')}';
}
