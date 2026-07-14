// @spotto/charts — typed React port of the prototype's dependency-free
// SVG charts (Prototype/design/Charts.jsx), rendering logic kept verbatim.
'use client';

import React from 'react';

const FONT = 'Nunito, ui-rounded, system-ui, sans-serif';

export interface TrafficPoint {
  day: string;
  views: number;
  redemptions: number;
}

// ── Weekly traffic: grouped bars (views) with redemption overlay ──
export function TrafficChart({ data, accent }: { data: TrafficPoint[]; accent: string }) {
  const W = 640, H = 240, padX = 12, padTop = 16, padBot = 34;
  const maxViews = Math.max(...data.map((d) => d.views));
  const max = maxViews * 1.1;
  const bw = (W - padX * 2) / data.length;
  const y = (v: number) => padTop + (1 - v / max) * (H - padTop - padBot);
  return (
    <svg viewBox={`0 0 ${W} ${H}`} style={{ width: '100%', height: 'auto', display: 'block' }}>
      {[0.25, 0.5, 0.75, 1].map((g, i) => (
        <line key={i} x1={padX} x2={W - padX} y1={y(max * g)} y2={y(max * g)}
          stroke="rgba(15,61,58,0.08)" strokeWidth="1" />
      ))}
      {data.map((d, i) => {
        const cx = padX + bw * i + bw / 2;
        const barW = Math.min(34, bw * 0.5);
        const vh = H - padTop - padBot - (y(d.views) - padTop);
        return (
          <g key={i}>
            <rect x={cx - barW / 2} y={y(d.views)} width={barW} height={vh} rx="6"
              fill={accent} opacity="0.18" />
            <rect x={cx - barW / 2} y={y(d.views)} width={barW}
              height={Math.min(vh, vh * (d.redemptions / d.views) + 10)} rx="6" fill={accent} />
            <text x={cx} y={H - 14} textAnchor="middle" fontSize="13" fontWeight="700"
              fill="#6c8582" fontFamily={FONT}>{d.day}</text>
            <text x={cx} y={y(d.views) - 7} textAnchor="middle" fontSize="11.5" fontWeight="800"
              fill="#103d3a" fontFamily={FONT}>{d.views}</text>
          </g>
        );
      })}
    </svg>
  );
}

export interface PopularPoint {
  h: number;
  v: number;
}

// ── Popular times: vertical histogram ──
export function PopularTimes({ data, accent }: { data: PopularPoint[]; accent: string }) {
  const W = 640, H = 180, padX = 8, padBot = 26, padTop = 10;
  const bw = (W - padX * 2) / data.length;
  const peak = Math.max(...data.map((d) => d.v));
  return (
    <svg viewBox={`0 0 ${W} ${H}`} style={{ width: '100%', height: 'auto', display: 'block' }}>
      {data.map((d, i) => {
        const cx = padX + bw * i + bw / 2;
        const barW = Math.min(26, bw * 0.62);
        const h = (d.v / 100) * (H - padTop - padBot);
        const isPeak = d.v === peak;
        const label = d.h > 12 ? `${d.h - 12}` : `${d.h}`;
        return (
          <g key={i}>
            <rect x={cx - barW / 2} y={H - padBot - h} width={barW} height={h} rx="5"
              fill={isPeak ? accent : 'rgba(15,61,58,0.16)'} />
            {d.h % 2 === 1 && (
              <text x={cx} y={H - 9} textAnchor="middle" fontSize="11" fontWeight="700"
                fill="#6c8582" fontFamily={FONT}>{label}{d.h >= 12 ? 'p' : 'a'}</text>
            )}
          </g>
        );
      })}
    </svg>
  );
}

export interface TrendPoint {
  wk: string;
  you: number;
  cat: number;
}

// ── Trend comparison: two smooth area/lines (you vs category) ──
export function TrendChart({ data, accent }: { data: TrendPoint[]; accent: string }) {
  const W = 640, H = 230, padX = 16, padTop = 18, padBot = 30;
  const max = Math.max(...data.flatMap((d) => [d.you, d.cat])) * 1.15;
  const xx = (i: number) => padX + (i / (data.length - 1)) * (W - padX * 2);
  const yy = (v: number) => padTop + (1 - v / max) * (H - padTop - padBot);

  const line = (key: 'you' | 'cat') =>
    data.map((d, i) => `${i === 0 ? 'M' : 'L'}${xx(i)},${yy(d[key])}`).join(' ');
  const area = (key: 'you' | 'cat') =>
    `${line(key)} L${xx(data.length - 1)},${H - padBot} L${xx(0)},${H - padBot} Z`;

  return (
    <svg viewBox={`0 0 ${W} ${H}`} style={{ width: '100%', height: 'auto', display: 'block' }}>
      <defs>
        <linearGradient id="youFill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0" stopColor={accent} stopOpacity="0.28" />
          <stop offset="1" stopColor={accent} stopOpacity="0.02" />
        </linearGradient>
      </defs>
      {[0.33, 0.66, 1].map((g, i) => (
        <line key={i} x1={padX} x2={W - padX} y1={yy(max * g)} y2={yy(max * g)}
          stroke="rgba(15,61,58,0.07)" strokeWidth="1" />
      ))}
      <path d={line('cat')} fill="none" stroke="#9bb0ad" strokeWidth="2.5" strokeDasharray="6 5"
        strokeLinecap="round" strokeLinejoin="round" />
      <path d={area('you')} fill="url(#youFill)" />
      <path d={line('you')} fill="none" stroke={accent} strokeWidth="3.5"
        strokeLinecap="round" strokeLinejoin="round" />
      {data.map((d, i) => (
        <g key={i}>
          <circle cx={xx(i)} cy={yy(d.you)} r="4.5" fill="#fff" stroke={accent} strokeWidth="3" />
          <text x={xx(i)} y={H - 9} textAnchor="middle" fontSize="12" fontWeight="700"
            fill="#6c8582" fontFamily={FONT}>{d.wk}</text>
        </g>
      ))}
    </svg>
  );
}

// ── Donut progress (claim limit) ──
export function Donut({ value, max, accent, size = 56, stroke = 8 }: {
  value: number; max: number; accent: string; size?: number; stroke?: number;
}) {
  const r = (size - stroke) / 2;
  const c = 2 * Math.PI * r;
  const pct = Math.min(1, value / max);
  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
      <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="rgba(15,61,58,0.1)" strokeWidth={stroke} />
      <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke={accent} strokeWidth={stroke}
        strokeLinecap="round" strokeDasharray={c} strokeDashoffset={c * (1 - pct)}
        transform={`rotate(-90 ${size / 2} ${size / 2})`} />
    </svg>
  );
}

// ── Sparkline ──
export function Sparkline({ values, accent, w = 96, h = 30 }: {
  values: number[]; accent: string; w?: number; h?: number;
}) {
  const max = Math.max(...values), min = Math.min(...values);
  const xx = (i: number) => (i / (values.length - 1)) * w;
  const yy = (v: number) => h - ((v - min) / (max - min || 1)) * (h - 4) - 2;
  const d = values.map((v, i) => `${i === 0 ? 'M' : 'L'}${xx(i)},${yy(v)}`).join(' ');
  return (
    <svg width={w} height={h} viewBox={`0 0 ${w} ${h}`}>
      <path d={`${d} L${w},${h} L0,${h} Z`} fill={accent} opacity="0.12" />
      <path d={d} fill="none" stroke={accent} strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}
