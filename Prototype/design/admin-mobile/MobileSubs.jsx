// MobileSubs.jsx — subscription manager: 2×2 stat grid + tabs + vendor cards.

const { useState: useStateMS } = React;

const STATE_M = {
  active:  { label: 'Active',      color: '#16a34a', bg: 'rgba(22,163,74,0.1)' },
  soon:    { label: 'Renews soon', color: '#d97706', bg: 'rgba(217,119,6,0.1)' },
  overdue: { label: 'Overdue',     color: '#e11d48', bg: 'rgba(225,29,72,0.1)' },
  trial:   { label: 'Trial',       color: '#0b6f86', bg: 'rgba(11,111,134,0.1)' },
  free:    { label: 'Free',        color: '#64748b', bg: 'rgba(100,116,139,0.1)' },
};

function SumCardM({ icon, label, value, sub, accent }) {
  return (
    <div className="rounded-2xl px-3.5 py-3" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
      <div className="flex items-center gap-2">
        <span className="w-7 h-7 rounded-lg flex items-center justify-center" style={{ background: accent + '1a', color: accent }}><Icon name={icon} size={15} /></span>
        <span className="text-[10.5px] font-bold uppercase tracking-wide" style={{ color: 'var(--muted)' }}>{label}</span>
      </div>
      <div className="font-bold tracking-tight mt-1.5" style={{ fontSize: 22, color: 'var(--ink)' }}>{value}</div>
      <div className="text-[11px] font-medium" style={{ color: 'var(--muted)' }}>{sub}</div>
    </div>
  );
}

function MobileSubs({ accent }) {
  const { SUBS } = window;
  const [tab, setTab] = useStateMS('all');
  const rows = SUBS.filter((s) => tab === 'all' ? true : tab === 'premium' ? s.tier === 'Premium' : s.tier === 'Free');

  const premium = SUBS.filter((s) => s.tier === 'Premium').length;
  const free = SUBS.filter((s) => s.tier === 'Free').length;
  const mrr = SUBS.reduce((a, b) => a + b.mrr, 0);
  const renewSoon = SUBS.filter((s) => s.state === 'soon' || s.state === 'overdue').length;

  return (
    <div className="h-full flex flex-col">
      <div className="flex-shrink-0" style={{ padding: 'var(--pad)', paddingBottom: 8 }}>
        <h1 className="font-bold tracking-tight" style={{ fontSize: 24, color: 'var(--ink)' }}>Subscriptions</h1>
        <p className="text-[13px] font-medium mt-0.5" style={{ color: 'var(--muted)' }}>Vendor tiers, MRR & renewals</p>
      </div>

      <div className="flex-1 overflow-y-auto noscroll flex flex-col" style={{ gap: 'var(--gap)', padding: '0 var(--pad) var(--pad)' }}>
        {/* stat grid */}
        <div className="grid grid-cols-2 gap-2.5 flex-shrink-0">
          <SumCardM icon="Crown" label="Premium" value={premium} sub={`$${premium * 49}/mo`} accent={accent} />
          <SumCardM icon="DollarSign" label="Total MRR" value={`$${mrr}`} sub="+9% MoM" accent="#16a34a" />
          <SumCardM icon="Users" label="Free tier" value={free} sub="upsell targets" accent="#64748b" />
          <SumCardM icon="CalendarClock" label="Renews ≤7d" value={renewSoon} sub="needs attention" accent="#d97706" />
        </div>

        {/* tabs */}
        <div className="flex items-center gap-2 flex-shrink-0">
          {[['all', 'All'], ['premium', 'Premium'], ['free', 'Free']].map(([k, lbl]) => {
            const on = tab === k;
            return (
              <button key={k} onClick={() => setTab(k)}
                className="rounded-full px-3.5 py-1.5 text-[12.5px] font-bold transition-colors"
                style={{ background: on ? accent : 'var(--surface)', border: on ? `1px solid ${accent}` : '1px solid var(--line)', color: on ? '#fff' : 'var(--muted)' }}>
                {lbl}
              </button>
            );
          })}
          <button className="ml-auto flex items-center gap-1.5 rounded-full px-3 py-1.5 text-[12.5px] font-bold" style={{ background: 'var(--surface)', border: '1px solid var(--line)', color: 'var(--ink)' }}>
            <Icon name="Download" size={14} /> CSV
          </button>
        </div>

        {/* vendor cards */}
        {rows.map((s) => {
          const st = STATE_M[s.state];
          return (
            <div key={s.id} className="rounded-2xl flex-shrink-0" style={{ padding: 'var(--pad)', background: 'var(--surface)', border: '1px solid var(--line)' }}>
              <div className="flex items-start gap-2">
                <div className="min-w-0 flex-1">
                  <div className="font-bold text-[14.5px] tracking-tight" style={{ color: 'var(--ink)' }}>{s.vendor}</div>
                  <div className="flex items-center gap-1.5 text-[11.5px] font-medium mt-0.5" style={{ color: 'var(--muted)' }}>
                    <span className="font-mono">{s.id}</span><span>·</span>
                    <span className="flex items-center gap-0.5"><Icon name="MapPin" size={10} />{s.town}</span><span>·</span>
                    <span>since {s.since}</span>
                  </div>
                </div>
                <span className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-bold flex-shrink-0"
                  style={{ background: s.tier === 'Premium' ? accent + '1a' : 'var(--bg)', color: s.tier === 'Premium' ? accent : 'var(--muted)' }}>
                  {s.tier === 'Premium' && <Icon name="Crown" size={12} />}{s.tier}
                </span>
              </div>
              <div className="flex items-center gap-2 mt-3 pt-3" style={{ borderTop: '1px solid var(--line)' }}>
                <span className="font-mono font-bold text-[15px]" style={{ color: s.mrr ? 'var(--ink)' : 'var(--muted)' }}>{s.mrr ? `$${s.mrr}` : '$0'}</span>
                <span className="text-[11px] font-semibold" style={{ color: 'var(--muted)' }}>/mo</span>
                <span className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-bold" style={{ background: st.bg, color: st.color }}>
                  {(s.state === 'soon' || s.state === 'overdue') && <Icon name="Clock" size={11} />}
                  {s.renews === '—' ? st.label : s.renews}
                </span>
                {s.tier === 'Free' ? (
                  <button className="ml-auto rounded-lg px-3 py-1.5 text-[12px] font-bold text-white" style={{ background: accent }}>Upsell</button>
                ) : (
                  <button className="ml-auto w-8 h-8 rounded-lg flex items-center justify-center" style={{ color: 'var(--muted)', background: 'var(--bg)' }}><Icon name="MoreHorizontal" size={16} /></button>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

Object.assign(window, { MobileSubs });
