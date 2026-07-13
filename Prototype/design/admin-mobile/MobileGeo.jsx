// MobileGeo.jsx — Sri Lanka map + selected region detail + acquisition opportunities.

const { useState: useStateMG } = React;

function MetricM({ label, value, accent }) {
  return (
    <div className="rounded-xl px-3 py-2.5" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
      <div className="text-[10px] font-bold uppercase tracking-wide" style={{ color: 'var(--muted)' }}>{label}</div>
      <div className="flex items-end gap-1 mt-1">
        <span className="font-bold tracking-tight" style={{ fontSize: 24, color: 'var(--ink)' }}>{value}</span>
        <span className="text-[11px] font-semibold mb-1" style={{ color: 'var(--muted)' }}>/100</span>
      </div>
      <div className="h-1.5 rounded-full mt-1.5 overflow-hidden" style={{ background: 'var(--line)' }}>
        <div className="h-full rounded-full" style={{ width: `${value}%`, background: accent }} />
      </div>
    </div>
  );
}

function MobileGeo({ accent }) {
  const { REGIONS } = window;
  const ranked = [...REGIONS].sort((a, b) => (b.travelers - b.vendors) - (a.travelers - a.vendors));
  const [sel, setSel] = useStateMG(ranked[0].id);
  const r = REGIONS.find((x) => x.id === sel);
  const gap = r.travelers - r.vendors;
  const tn = window.regionTone(gap);
  const gaps = ranked.filter((x) => (x.travelers - x.vendors) >= 12);

  return (
    <div className="h-full flex flex-col">
      <div className="flex-shrink-0" style={{ padding: 'var(--pad)', paddingBottom: 8 }}>
        <h1 className="font-bold tracking-tight" style={{ fontSize: 24, color: 'var(--ink)' }}>Geo-analytics</h1>
        <p className="text-[13px] font-medium mt-0.5" style={{ color: 'var(--muted)' }}>Traveler demand vs vendor supply</p>
      </div>

      <div className="flex-1 overflow-y-auto noscroll flex flex-col" style={{ gap: 'var(--gap)', padding: '0 var(--pad) var(--pad)' }}>
        {/* map */}
        <div className="rounded-2xl overflow-hidden relative flex-shrink-0" style={{ height: 300, background: 'radial-gradient(120% 100% at 50% 0%, #0c2b28, #0b2320)', border: '1px solid var(--line)' }}>
          <div className="absolute top-3 left-4 z-10">
            <div className="font-bold text-[12px] text-white">Sri Lanka · live</div>
            <div className="text-[10px] font-medium" style={{ color: 'rgba(255,255,255,0.5)' }}>Tap a region</div>
          </div>
          <div className="absolute top-3 right-4 z-10 flex flex-col gap-1 items-end text-[10px] font-semibold" style={{ color: 'rgba(255,255,255,0.6)' }}>
            <span className="flex items-center gap-1.5">Demand <span className="w-3 h-3 rounded-full border-2" style={{ borderColor: 'rgba(255,255,255,0.6)' }} /></span>
            <span className="flex items-center gap-1.5">Supply <span className="w-2.5 h-2.5 rounded-full" style={{ background: 'rgba(255,255,255,0.6)' }} /></span>
          </div>
          <SriLankaMap regions={REGIONS} selected={sel} onSelect={setSel} accent={accent} />
        </div>

        {/* selected region */}
        <div className="rounded-2xl flex-shrink-0" style={{ padding: 'var(--pad)', background: 'var(--surface)', border: '1px solid var(--line)' }}>
          <div className="flex items-center justify-between">
            <h3 className="font-bold text-[17px]" style={{ color: 'var(--ink)' }}>{r.name}</h3>
            <span className="rounded-full px-2.5 py-1 text-[11px] font-bold" style={{ background: tn.ring + '1a', color: tn.ring }}>{tn.label}</span>
          </div>
          <div className="grid grid-cols-2 gap-2.5 mt-3">
            <MetricM label="Traveler demand" value={r.travelers} accent={accent} />
            <MetricM label="Vendor supply" value={r.vendors} accent="#64748b" />
          </div>
          <div className="mt-3 rounded-xl px-3.5 py-3" style={{ background: gap >= 12 ? 'rgba(225,29,72,0.07)' : 'var(--bg)', border: `1px solid ${gap >= 12 ? 'rgba(225,29,72,0.2)' : 'var(--line)'}` }}>
            <div className="flex items-center justify-between">
              <span className="text-[11px] font-bold uppercase tracking-wide" style={{ color: 'var(--muted)' }}>Supply gap</span>
              <span className="font-mono font-bold text-[16px]" style={{ color: tn.ring }}>{gap > 0 ? '+' : ''}{gap}</span>
            </div>
            <p className="text-[12.5px] font-medium mt-1.5 leading-relaxed" style={{ color: 'var(--ink-soft)' }}>
              {gap >= 28 ? 'High interest, very few vendors. Prioritise outreach & onboarding incentives here.'
               : gap >= 12 ? 'Demand outpaces supply. Good target for vendor acquisition campaigns.'
               : gap <= -10 ? 'Well-supplied — focus on quality & retention over acquisition.'
               : 'Healthy demand-supply balance. Monitor for shifts.'}
            </p>
          </div>
        </div>

        {/* opportunities */}
        <div className="rounded-2xl overflow-hidden flex-shrink-0" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
          <div className="flex items-center gap-2 px-4 py-3" style={{ borderBottom: '1px solid var(--line)' }}>
            <Icon name="Target" size={15} style={{ color: '#e11d48' }} />
            <span className="font-bold text-[13px]" style={{ color: 'var(--ink)' }}>Top acquisition opportunities</span>
          </div>
          {gaps.map((g, i) => {
            const gp = g.travelers - g.vendors;
            const t = window.regionTone(gp);
            return (
              <button key={g.id} onClick={() => setSel(g.id)}
                className="w-full text-left px-4 py-2.5 flex items-center gap-3"
                style={{ borderBottom: i < gaps.length - 1 ? '1px solid var(--line)' : 'none', background: sel === g.id ? 'var(--bg)' : 'transparent' }}>
                <span className="font-mono text-[12px] font-bold w-4" style={{ color: 'var(--muted)' }}>{i + 1}</span>
                <div className="flex-1 min-w-0">
                  <div className="font-bold text-[13.5px]" style={{ color: 'var(--ink)' }}>{g.name}</div>
                  <div className="h-1.5 rounded-full mt-1 overflow-hidden" style={{ background: 'var(--bg)' }}>
                    <div className="h-full rounded-full" style={{ width: `${g.travelers}%`, background: t.ring, opacity: 0.45 }} />
                  </div>
                </div>
                <span className="font-mono font-bold text-[14px]" style={{ color: t.ring }}>+{gp}</span>
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}

Object.assign(window, { MobileGeo });
