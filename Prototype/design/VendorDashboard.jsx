// VendorDashboard.jsx — overview: KPIs, traffic, active offers, live redemption feed.

const TONE = {
  teal:  { fg: '#0b6f86', bg: '#d6efec' },
  coral: { fg: '#f0682f', bg: '#ffe5d6' },
  sun:   { fg: '#c98a00', bg: '#fbeec4' },
};

function KpiCard({ k, accent }) {
  const tone = TONE[k.tone] || TONE.teal;
  return (
    <div className="rounded-2xl p-5 bg-white" style={{ border: '1px solid var(--line)', boxShadow: '0 1px 2px rgba(15,61,58,0.04)' }}>
      <div className="flex items-start justify-between">
        <div className="w-11 h-11 rounded-xl flex items-center justify-center"
             style={{ background: tone.bg, color: tone.fg }}>
          <Icon name={k.icon} size={22} />
        </div>
        {k.delta != null && (
          <div className="flex items-center gap-1 rounded-full px-2 py-1 text-[12.5px] font-extrabold"
               style={{ background: k.delta >= 0 ? '#e7f6ec' : '#fde8e4', color: k.delta >= 0 ? '#16a34a' : '#dc4a2d' }}>
            <Icon name={k.delta >= 0 ? 'TrendingUp' : 'TrendingDown'} size={13} />
            {k.delta >= 0 ? '+' : ''}{k.delta}%
          </div>
        )}
      </div>
      <div className="mt-4 font-extrabold tracking-tight" style={{ fontSize: 34, color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>
        {k.value.toLocaleString()}
      </div>
      <div className="flex items-center gap-2 mt-1">
        <span className="text-[14px] font-bold" style={{ color: 'var(--ink-soft)' }}>{k.label}</span>
        <span className="text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>· {k.hint}</span>
      </div>
    </div>
  );
}

function OfferRow({ o, accent }) {
  const live = o.status === 'live';
  return (
    <div className="flex items-center gap-4 py-3.5" style={{ borderTop: '1px solid var(--line)' }}>
      <Donut value={o.claimed} max={o.limit} accent={o.status === 'ending' ? '#f0682f' : accent} size={50} stroke={7} />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="font-bold text-[14.5px] truncate" style={{ color: 'var(--ink)' }}>{o.title}</span>
          <span className="flex-shrink-0 rounded-full px-2 py-0.5 text-[11px] font-extrabold uppercase tracking-wide"
                style={{ background: o.status === 'ending' ? '#fde8e4' : '#e7f6ec', color: o.status === 'ending' ? '#dc4a2d' : '#16a34a' }}>
            {o.status === 'ending' ? 'Ending soon' : 'Live'}
          </span>
        </div>
        <div className="text-[12.5px] font-semibold mt-0.5" style={{ color: 'var(--muted)' }}>
          {o.claimed}/{o.limit} claimed · {o.redeemed} redeemed · ends {o.end}
        </div>
      </div>
    </div>
  );
}

function styleDot(style) {
  return { 'Digital Nomad': '#0b6f86', 'Surf Traveler': '#f0682f', 'Luxury': '#c98a00', 'Backpacker': '#5a5f9e' }[style] || '#6c8582';
}

function VendorDashboard({ accent, onNav }) {
  const { KPIS, WEEK, OFFERS, REDEMPTIONS, VENDOR } = window;
  return (
    <div className="space-y-6">
      {/* header */}
      <div className="flex items-end justify-between gap-4">
        <div className="min-w-0">
          <h1 className="font-extrabold tracking-tight" style={{ fontSize: 28, lineHeight: 1.15, color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>
            Good morning, Tharindu 👋
          </h1>
          <p className="text-[14.5px] font-semibold mt-1.5" style={{ color: 'var(--muted)' }}>
            Here's how {VENDOR.name.split(' ')[0]} is doing in {VENDOR.area} today.
          </p>
        </div>
        <button onClick={() => onNav('offers')}
          className="rounded-xl px-4 py-2.5 font-extrabold text-[14px] flex items-center gap-2 text-white active:scale-[0.98] transition-transform flex-shrink-0 whitespace-nowrap"
          style={{ background: accent, boxShadow: '0 8px 18px -8px var(--accent-shadow)' }}>
          <Icon name="Plus" size={18} strokeWidth={2.5} /> New offer
        </button>
      </div>

      {/* KPIs */}
      <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))' }}>
        {KPIS.map((k) => <KpiCard key={k.id} k={k} accent={accent} />)}
      </div>

      <div className="grid gap-6 grid-cols-1 lg:[grid-template-columns:minmax(0,1.5fr)_minmax(0,1fr)]">
        {/* traffic */}
        <div className="rounded-2xl p-5 bg-white" style={{ border: '1px solid var(--line)' }}>
          <div className="flex items-center justify-between">
            <div>
              <h3 className="font-extrabold text-[16px]" style={{ color: 'var(--ink)' }}>Traffic this week</h3>
              <p className="text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>Profile views · redemptions</p>
            </div>
            <div className="flex items-center gap-3 text-[12px] font-bold" style={{ color: 'var(--muted)' }}>
              <span className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-sm" style={{ background: accent, opacity: 0.25 }} />Views</span>
              <span className="flex items-center gap-1.5"><span className="w-2.5 h-2.5 rounded-sm" style={{ background: accent }} />Redeemed</span>
            </div>
          </div>
          <div className="mt-3"><TrafficChart data={WEEK} accent={accent} /></div>
        </div>

        {/* live redemptions */}
        <div className="rounded-2xl p-5 bg-white" style={{ border: '1px solid var(--line)' }}>
          <div className="flex items-center justify-between">
            <h3 className="font-extrabold text-[16px]" style={{ color: 'var(--ink)' }}>Live redemptions</h3>
            <span className="flex items-center gap-1.5 text-[12px] font-bold" style={{ color: '#16a34a' }}>
              <span className="w-2 h-2 rounded-full" style={{ background: '#16a34a', boxShadow: '0 0 0 3px #e7f6ec' }} /> Live
            </span>
          </div>
          <div className="mt-1">
            {REDEMPTIONS.map((r) => (
              <div key={r.id} className="flex items-center gap-3 py-2.5" style={{ borderTop: '1px solid var(--line)' }}>
                <div className="w-9 h-9 rounded-full flex items-center justify-center text-[12px] font-extrabold flex-shrink-0"
                     style={{ background: 'var(--sand)', color: 'var(--ink)' }}>
                  {r.name.split(' ')[0][0]}
                </div>
                <div className="min-w-0 flex-1">
                  <div className="font-bold text-[13.5px] truncate" style={{ color: 'var(--ink)' }}>{r.name}</div>
                  <div className="text-[12px] font-semibold truncate" style={{ color: 'var(--muted)' }}>{r.offer}</div>
                </div>
                <div className="text-right flex-shrink-0">
                  <div className="flex items-center gap-1 justify-end text-[11px] font-bold" style={{ color: styleDot(r.style) }}>
                    <span className="w-1.5 h-1.5 rounded-full" style={{ background: styleDot(r.style) }} />{r.style}
                  </div>
                  <div className="text-[11.5px] font-semibold" style={{ color: 'var(--muted)' }}>{r.time}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* active offers */}
      <div className="rounded-2xl p-5 bg-white" style={{ border: '1px solid var(--line)' }}>
        <div className="flex items-center justify-between mb-1">
          <h3 className="font-extrabold text-[16px]" style={{ color: 'var(--ink)' }}>Active offers</h3>
          <button onClick={() => onNav('offers')} className="text-[13px] font-extrabold flex items-center gap-1" style={{ color: accent }}>
            Manage <Icon name="ArrowRight" size={15} />
          </button>
        </div>
        {OFFERS.filter((o) => o.status === 'live' || o.status === 'ending').map((o) => <OfferRow key={o.id} o={o} accent={accent} />)}
      </div>
    </div>
  );
}

Object.assign(window, { VendorDashboard });
