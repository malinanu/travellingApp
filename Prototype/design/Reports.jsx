// Reports.jsx — weekly traffic, popular times, deals vs category trends.

function StatPill({ icon, label, value, sub, accent }) {
  return (
    <div className="rounded-2xl p-4 bg-white" style={{ border: '1px solid var(--line)' }}>
      <div className="flex items-center gap-2 text-[12.5px] font-bold" style={{ color: 'var(--muted)' }}>
        <Icon name={icon} size={15} style={{ color: accent }} />{label}
      </div>
      <div className="font-extrabold tracking-tight mt-1.5" style={{ fontSize: 24, color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>{value}</div>
      <div className="text-[12px] font-semibold mt-0.5" style={{ color: 'var(--muted)' }}>{sub}</div>
    </div>
  );
}

function Reports({ accent }) {
  const { WEEK, POPULAR, TREND } = window;
  const totalViews = WEEK.reduce((a, b) => a + b.views, 0);
  const totalRedeem = WEEK.reduce((a, b) => a + b.redemptions, 0);
  const peakHour = POPULAR.reduce((a, b) => b.v > a.v ? b : a);
  const peakLabel = `${peakHour.h > 12 ? peakHour.h - 12 : peakHour.h}${peakHour.h >= 12 ? 'pm' : 'am'}`;
  const youLatest = TREND[TREND.length - 1].you, catLatest = TREND[TREND.length - 1].cat;
  const lead = Math.round(((youLatest - catLatest) / catLatest) * 100);

  return (
    <div className="space-y-6">
      <div className="flex items-end justify-between gap-4">
        <div className="min-w-0">
          <h1 className="font-extrabold tracking-tight" style={{ fontSize: 28, lineHeight: 1.15, color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Weekly report</h1>
          <p className="text-[14.5px] font-semibold mt-1" style={{ color: 'var(--muted)' }}>Sep 16 – Sep 22 · Weligama café category</p>
        </div>
        <button className="rounded-xl px-4 py-2.5 font-extrabold text-[14px] flex items-center gap-2 flex-shrink-0 whitespace-nowrap"
                style={{ background: '#fff', border: '1px solid var(--line)', color: 'var(--ink)' }}>
          <Icon name="Download" size={17} /> Export PDF
        </button>
      </div>

      {/* top stats */}
      <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(190px, 1fr))' }}>
        <StatPill icon="Eye" label="Profile views" value={totalViews.toLocaleString()} sub="+12% vs last week" accent={accent} />
        <StatPill icon="Ticket" label="Redemptions" value={totalRedeem} sub="+18% vs last week" accent={accent} />
        <StatPill icon="Clock" label="Peak hour" value={peakLabel} sub="busiest footfall" accent={accent} />
        <StatPill icon="Award" label="vs category" value={`+${lead}%`} sub="above Weligama avg" accent={accent} />
      </div>

      {/* trend comparison */}
      <div className="rounded-2xl p-5 bg-white" style={{ border: '1px solid var(--line)' }}>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h3 className="font-extrabold text-[16px]" style={{ color: 'var(--ink)' }}>Your deals vs category trends</h3>
            <p className="text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>Redemption index · 6 weeks · Weligama cafés</p>
          </div>
          <div className="flex items-center gap-4 text-[12.5px] font-bold">
            <span className="flex items-center gap-1.5" style={{ color: 'var(--ink)' }}>
              <span className="w-4 h-1 rounded-full" style={{ background: accent }} />You
            </span>
            <span className="flex items-center gap-1.5" style={{ color: 'var(--muted)' }}>
              <span className="w-4 h-0.5 rounded-full" style={{ background: '#9bb0ad', borderTop: '2px dashed #9bb0ad' }} />Category avg
            </span>
          </div>
        </div>
        <div className="mt-3"><TrendChart data={TREND} accent={accent} /></div>
        <div className="rounded-xl p-3 mt-2 flex items-start gap-2.5" style={{ background: 'var(--accent-tint)' }}>
          <Icon name="Sparkles" size={18} style={{ color: accent, flexShrink: 0, marginTop: 1 }} />
          <p className="text-[13px] font-semibold leading-relaxed" style={{ color: 'var(--ink-soft)' }}>
            Your flash offers are pulling <b style={{ color: accent }}>{lead}% more redemptions</b> than the average Weligama café. Mornings (9–10am) are your strongest window — consider a recurring breakfast deal.
          </p>
        </div>
      </div>

      <div className="grid gap-6" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(320px, 100%), 1fr))' }}>
        {/* traffic */}
        <div className="rounded-2xl p-5 bg-white" style={{ border: '1px solid var(--line)' }}>
          <h3 className="font-extrabold text-[16px]" style={{ color: 'var(--ink)' }}>Daily traffic</h3>
          <p className="text-[12.5px] font-semibold mb-2" style={{ color: 'var(--muted)' }}>Views with redemption share</p>
          <TrafficChart data={WEEK} accent={accent} />
        </div>
        {/* popular times */}
        <div className="rounded-2xl p-5 bg-white" style={{ border: '1px solid var(--line)' }}>
          <h3 className="font-extrabold text-[16px]" style={{ color: 'var(--ink)' }}>Popular times</h3>
          <p className="text-[12.5px] font-semibold mb-2" style={{ color: 'var(--muted)' }}>Avg footfall by hour · peak {peakLabel}</p>
          <PopularTimes data={POPULAR} accent={accent} />
        </div>
      </div>
    </div>
  );
}

Object.assign(window, { Reports });
