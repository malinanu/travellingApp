// Small dependency-free charts for the vendor portal — port of Charts.jsx.

import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../theme/palette.dart';
import '../../data/mock/vendor_seed.dart';

const _grid = Color(0x14103D3A);

TextPainter _label(String s, double size, Color color,
    {FontWeight weight = FontWeight.w700}) {
  final tp = TextPainter(
    text: TextSpan(
        text: s, style: TextStyle(fontSize: size, fontWeight: weight, color: color)),
    textDirection: TextDirection.ltr,
  )..layout();
  return tp;
}

/// Weekly traffic: view bars with a solid redemption overlay + value labels.
class TrafficChart extends StatelessWidget {
  final List<DayTraffic> data;
  const TrafficChart({super.key, this.data = weekTraffic});

  @override
  Widget build(BuildContext context) {
    return AspectRatio(
      aspectRatio: 640 / 240,
      child: CustomPaint(painter: _TrafficPainter(data)),
    );
  }
}

class _TrafficPainter extends CustomPainter {
  final List<DayTraffic> data;
  _TrafficPainter(this.data);

  @override
  void paint(Canvas canvas, Size size) {
    const padX = 12.0, padTop = 16.0, padBot = 34.0;
    final maxV = data.map((d) => d.views).reduce(math.max) * 1.1;
    final bw = (size.width - padX * 2) / data.length;
    double y(num v) => padTop + (1 - v / maxV) * (size.height - padTop - padBot);

    final gridPaint = Paint()
      ..color = _grid
      ..strokeWidth = 1;
    for (final g in [0.25, 0.5, 0.75, 1.0]) {
      canvas.drawLine(
          Offset(padX, y(maxV * g)), Offset(size.width - padX, y(maxV * g)), gridPaint);
    }

    for (final (i, d) in data.indexed) {
      final cx = padX + bw * i + bw / 2;
      final barW = math.min(34.0, bw * 0.5);
      final top = y(d.views);
      final vh = size.height - padBot - top;
      final rh = math.min(vh, vh * (d.redemptions / d.views) + 10);

      canvas.drawRRect(
        RRect.fromRectAndRadius(
            Rect.fromLTWH(cx - barW / 2, top, barW, vh), const Radius.circular(6)),
        Paint()..color = SpottoColors.accent.withValues(alpha: 0.18),
      );
      canvas.drawRRect(
        RRect.fromRectAndRadius(
            Rect.fromLTWH(cx - barW / 2, size.height - padBot - rh, barW, rh),
            const Radius.circular(6)),
        Paint()..color = SpottoColors.accent,
      );

      final day = _label(d.day, 13, SpottoColors.muted);
      day.paint(canvas, Offset(cx - day.width / 2, size.height - 22));
      final val = _label('${d.views}', 11.5, SpottoColors.ink, weight: FontWeight.w800);
      val.paint(canvas, Offset(cx - val.width / 2, top - 16));
    }
  }

  @override
  bool shouldRepaint(_TrafficPainter oldDelegate) => oldDelegate.data != data;
}

/// Popular times: hourly footfall histogram, peak hour highlighted.
class PopularTimesChart extends StatelessWidget {
  final List<HourFootfall> data;
  const PopularTimesChart({super.key, this.data = popularTimes});

  @override
  Widget build(BuildContext context) {
    return AspectRatio(
      aspectRatio: 640 / 180,
      child: CustomPaint(painter: _PopularTimesPainter(data)),
    );
  }
}

class _PopularTimesPainter extends CustomPainter {
  final List<HourFootfall> data;
  _PopularTimesPainter(this.data);

  @override
  void paint(Canvas canvas, Size size) {
    const padX = 8.0, padBot = 26.0, padTop = 10.0;
    final bw = (size.width - padX * 2) / data.length;
    final peak = data.map((d) => d.v).reduce(math.max);

    for (final (i, d) in data.indexed) {
      final cx = padX + bw * i + bw / 2;
      final barW = math.min(26.0, bw * 0.62);
      final h = (d.v / 100) * (size.height - padTop - padBot);
      canvas.drawRRect(
        RRect.fromRectAndRadius(
            Rect.fromLTWH(cx - barW / 2, size.height - padBot - h, barW, h),
            const Radius.circular(5)),
        Paint()
          ..color = d.v == peak
              ? SpottoColors.accent
              : SpottoColors.ink.withValues(alpha: 0.16),
      );
      if (d.h.isOdd) {
        final hour = d.h > 12 ? d.h - 12 : d.h;
        final tp = _label('$hour${d.h >= 12 ? 'p' : 'a'}', 11, SpottoColors.muted);
        tp.paint(canvas, Offset(cx - tp.width / 2, size.height - 17));
      }
    }
  }

  @override
  bool shouldRepaint(_PopularTimesPainter oldDelegate) => oldDelegate.data != data;
}

/// You-vs-category trend: accent area/line over a dashed grey category line.
class TrendChart extends StatelessWidget {
  final List<WeekTrend> data;
  const TrendChart({super.key, this.data = trend});

  @override
  Widget build(BuildContext context) {
    return AspectRatio(
      aspectRatio: 640 / 230,
      child: CustomPaint(painter: _TrendPainter(data)),
    );
  }
}

class _TrendPainter extends CustomPainter {
  final List<WeekTrend> data;
  _TrendPainter(this.data);

  @override
  void paint(Canvas canvas, Size size) {
    const padX = 16.0, padTop = 18.0, padBot = 30.0;
    final maxV =
        data.expand((d) => [d.you, d.cat]).reduce(math.max) * 1.15;
    double xx(int i) => padX + (i / (data.length - 1)) * (size.width - padX * 2);
    double yy(num v) => padTop + (1 - v / maxV) * (size.height - padTop - padBot);

    final gridPaint = Paint()
      ..color = _grid
      ..strokeWidth = 1;
    for (final g in [0.33, 0.66, 1.0]) {
      canvas.drawLine(
          Offset(padX, yy(maxV * g)), Offset(size.width - padX, yy(maxV * g)), gridPaint);
    }

    // category: dashed grey
    final catPaint = Paint()
      ..color = const Color(0xFF9BB0AD)
      ..strokeWidth = 2.5
      ..strokeCap = StrokeCap.round
      ..style = PaintingStyle.stroke;
    for (var i = 1; i < data.length; i++) {
      _dashedLine(canvas, Offset(xx(i - 1), yy(data[i - 1].cat)),
          Offset(xx(i), yy(data[i].cat)), catPaint);
    }

    // you: gradient area + line + dots
    final area = Path()..moveTo(xx(0), yy(data[0].you));
    for (var i = 1; i < data.length; i++) {
      area.lineTo(xx(i), yy(data[i].you));
    }
    final line = Path.from(area);
    area
      ..lineTo(xx(data.length - 1), size.height - padBot)
      ..lineTo(xx(0), size.height - padBot)
      ..close();
    canvas.drawPath(
      area,
      Paint()
        ..shader = LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [
            SpottoColors.accent.withValues(alpha: 0.28),
            SpottoColors.accent.withValues(alpha: 0.02),
          ],
        ).createShader(Rect.fromLTWH(0, 0, size.width, size.height)),
    );
    canvas.drawPath(
      line,
      Paint()
        ..color = SpottoColors.accent
        ..strokeWidth = 3.5
        ..strokeCap = StrokeCap.round
        ..strokeJoin = StrokeJoin.round
        ..style = PaintingStyle.stroke,
    );

    for (final (i, d) in data.indexed) {
      canvas.drawCircle(Offset(xx(i), yy(d.you)), 4.5, Paint()..color = Colors.white);
      canvas.drawCircle(
        Offset(xx(i), yy(d.you)),
        4.5,
        Paint()
          ..color = SpottoColors.accent
          ..strokeWidth = 3
          ..style = PaintingStyle.stroke,
      );
      final tp = _label(d.wk, 12, SpottoColors.muted);
      tp.paint(canvas, Offset(xx(i) - tp.width / 2, size.height - 18));
    }
  }

  void _dashedLine(Canvas canvas, Offset a, Offset b, Paint paint) {
    const dash = 6.0, gap = 5.0;
    final total = (b - a).distance;
    final dir = (b - a) / total;
    var t = 0.0;
    while (t < total) {
      final end = math.min(t + dash, total);
      canvas.drawLine(a + dir * t, a + dir * end, paint);
      t = end + gap;
    }
  }

  @override
  bool shouldRepaint(_TrendPainter oldDelegate) => oldDelegate.data != data;
}

/// Donut progress (e.g. offer claims vs limit).
class Donut extends StatelessWidget {
  final int value;
  final int max;
  final double size;
  final double stroke;
  final Color color;

  const Donut({
    super.key,
    required this.value,
    required this.max,
    this.size = 56,
    this.stroke = 8,
    this.color = SpottoColors.accent,
  });

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: size,
      height: size,
      child: CustomPaint(
          painter: _DonutPainter(value: value, max: max, stroke: stroke, color: color)),
    );
  }
}

class _DonutPainter extends CustomPainter {
  final int value;
  final int max;
  final double stroke;
  final Color color;
  _DonutPainter(
      {required this.value, required this.max, required this.stroke, required this.color});

  @override
  void paint(Canvas canvas, Size size) {
    final r = (size.width - stroke) / 2;
    final center = Offset(size.width / 2, size.height / 2);
    canvas.drawCircle(
      center,
      r,
      Paint()
        ..color = SpottoColors.ink.withValues(alpha: 0.1)
        ..strokeWidth = stroke
        ..style = PaintingStyle.stroke,
    );
    final pct = max == 0 ? 0.0 : math.min(1.0, value / max);
    canvas.drawArc(
      Rect.fromCircle(center: center, radius: r),
      -math.pi / 2,
      2 * math.pi * pct,
      false,
      Paint()
        ..color = color
        ..strokeWidth = stroke
        ..strokeCap = StrokeCap.round
        ..style = PaintingStyle.stroke,
    );
  }

  @override
  bool shouldRepaint(_DonutPainter oldDelegate) =>
      oldDelegate.value != value || oldDelegate.max != max || oldDelegate.color != color;
}

/// Tiny inline sparkline with a soft area fill.
class Sparkline extends StatelessWidget {
  final List<num> values;
  final double width;
  final double height;
  final Color color;

  const Sparkline({
    super.key,
    required this.values,
    this.width = 96,
    this.height = 30,
    this.color = SpottoColors.accent,
  });

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: width,
      height: height,
      child: CustomPaint(painter: _SparklinePainter(values, color)),
    );
  }
}

class _SparklinePainter extends CustomPainter {
  final List<num> values;
  final Color color;
  _SparklinePainter(this.values, this.color);

  @override
  void paint(Canvas canvas, Size size) {
    final maxV = values.reduce(math.max), minV = values.reduce(math.min);
    final range = (maxV - minV) == 0 ? 1 : maxV - minV;
    double xx(int i) => (i / (values.length - 1)) * size.width;
    double yy(num v) => size.height - ((v - minV) / range) * (size.height - 4) - 2;

    final line = Path()..moveTo(xx(0), yy(values[0]));
    for (var i = 1; i < values.length; i++) {
      line.lineTo(xx(i), yy(values[i]));
    }
    final area = Path.from(line)
      ..lineTo(size.width, size.height)
      ..lineTo(0, size.height)
      ..close();

    canvas.drawPath(area, Paint()..color = color.withValues(alpha: 0.12));
    canvas.drawPath(
      line,
      Paint()
        ..color = color
        ..strokeWidth = 2.5
        ..strokeCap = StrokeCap.round
        ..strokeJoin = StrokeJoin.round
        ..style = PaintingStyle.stroke,
    );
  }

  @override
  bool shouldRepaint(_SparklinePainter oldDelegate) =>
      oldDelegate.values != values || oldDelegate.color != color;
}
