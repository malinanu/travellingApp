// Subscriptions.jsx — Free vs Premium table with renewals.

const { useState: useStateSB } = React;

const STATE = {
  active:  { label: 'Active',   color: '#16a34a', bg: 'rgba(22,163,74,0.1)' },
  soon:    { label: 'Renews soon', color: '#d97706', bg: 'rgba(217,119,6,0.1)' },
  overdue: { label: 'Overdue',  color: '#e11d48', bg: 'rgba(225,29,72,0.1)' },
  trial:   { label: 'Trial',    color: '#0b6f86', bg: 'rgba(11,111,134,0.1)' },
  free:    { label: 'Free',     color: '#64748b', bg: 'rgba(100,116,139,0.1)' },
};

function Subscriptions({ accent }) {
  const { SUBS } = window;
  const [tab, setTab] = useStateSB('all');
  const rows = SUBS.filter((s) => tab === 'all' ? true : tab === 'premium' ? s.tier === 'Premium' : s.tier === 'Free');

  const premium = SUBS.filter((s) => s.tier === 'Premium').length;
  const free = SUBS.filter((s) => s.tier === 'Free').length;
  const mrr = SUBS.reduce((a, b) => a + b.mrr, 0);
  const renewSoon = SUBS.filter((s) => s.state === 'soon' || s.state === 'overdue').length;

  return (
    <div className="h-full flex flex-col">
      <div className="mb-4">
        <h1 className="font-bold tracking-tight" style={{ fontSize: 22, color: 'var(--ink)' }}>Subscription manager</h1>
        <p className="text-[13px] font-medium mt-0.5" style={{ color: 'var(--muted)' }}>Vendor tiers, MRR and upcoming renewals</p>
      </div>

      {/* summary */}
      <div className="grid gap-3 mb-4" style={{ gridTemplateColumns: 'repeat(4, 1fr)' }}>
        <SumCard icon="Crown" label="Premium" value={premium} sub={`$${premium * 49}/mo`} accent={accent} />
        <SumCard icon="Users" label="Free tier" value={free} sub="conversion targets" accent="#64748b" />
        <SumCard icon="DollarSign" label="Total MRR" value={`$${mrr}`} sub="+9% MoM" accent="#16a34a" />
        <SumCard icon="CalendarClock" label="Renews ≤7d" value={renewSoon} sub="needs attention" accent="#d97706" />
      </div>

      {/* tabs */}
      <div className="flex items-center gap-2 mb-3">
        {[['all', 'All vendors'], ['premium', 'Premium'], ['free', 'Free']].map(([k, lbl]) => {
          const on = tab === k;
          return (
            <button key={k} onClick={() => setTab(k)}
              className="rounded-lg px-3 py-1.5 text-[13px] font-bold transition-colors"
              style={{ background: on ? 'var(--surface)' : 'transparent', border: on ? `1px solid ${accent}` : '1px solid var(--line)', color: on ? accent : 'var(--muted)' }}>
              {lbl}
            </button>
          );
        })}
        <button className="ml-auto flex items-center gap-2 rounded-lg px-3 py-1.5 text-[13px] font-bold" style={{ background: 'var(--surface)', border: '1px solid var(--line)', color: 'var(--ink)' }}>
          <Icon name="Download" size={14} /> Export CSV
        </button>
      </div>

      {/* table */}
      <div className="flex-1 rounded-xl overflow-hidden flex flex-col min-h-0" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
        <div className="overflow-auto flex-1">
          <table className="w-full" style={{ borderCollapse: 'collapse' }}>
            <thead>
              <tr style={{ position: 'sticky', top: 0, background: 'var(--bg)', zIndex: 1 }}>
                {['Vendor', 'Location', 'Tier', 'MRR', 'Renewal', 'Member since', ''].map((h, i) => (
                  <th key={i} className="text-left px-4 py-2.5 text-[11px] font-bold uppercase tracking-wide"
                      style={{ color: 'var(--muted)', borderBottom: '1px solid var(--line)', textAlign: i === 3 ? 'right' : 'left' }}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((s) => {
                const st = STATE[s.state];
                return (
                  <tr key={s.id} className="group" style={{ borderBottom: '1px solid var(--line)' }}>
                    <td className="px-4 py-3">
                      <div className="font-bold text-[13.5px]" style={{ color: 'var(--ink)' }}>{s.vendor}</div>
                      <div className="font-mono text-[11px]" style={{ color: 'var(--muted)' }}>{s.id}</div>
                    </td>
                    <td className="px-4 py-3 text-[13px] font-medium" style={{ color: 'var(--ink-soft)' }}>{s.town}</td>
                    <td className="px-4 py-3">
                      <span className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-[11.5px] font-bold"
                            style={{ background: s.tier === 'Premium' ? 'rgba(11,111,134,0.1)' : 'var(--bg)', color: s.tier === 'Premium' ? accent : 'var(--muted)' }}>
                        {s.tier === 'Premium' && <Icon name="Crown" size={12} />}{s.tier}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-[13.5px] font-mono font-semibold text-right" style={{ color: s.mrr ? 'var(--ink)' : 'var(--muted)' }}>{s.mrr ? `$${s.mrr}` : '—'}</td>
                    <td className="px-4 py-3">
                      <span className="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-[11.5px] font-bold" style={{ background: st.bg, color: st.color }}>
                        {(s.state === 'soon' || s.state === 'overdue') && <Icon name="Clock" size={11} />}{s.renews === '—' ? st.label : s.renews}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-[13px] font-medium" style={{ color: 'var(--ink-soft)' }}>{s.since}</td>
                    <td className="px-4 py-3 text-right">
                      {s.tier === 'Free' ? (
                        <button className="rounded-md px-2.5 py-1 text-[12px] font-bold" style={{ background: accent, color: '#fff' }}>Upsell</button>
                      ) : (
                        <button className="rounded-md px-2 py-1 opacity-0 group-hover:opacity-100 transition-opacity" style={{ color: 'var(--muted)' }}><Icon name="MoreHorizontal" size={16} /></button>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

function SumCard({ icon, label, value, sub, accent }) {
  return (
    <div className="rounded-xl px-4 py-3.5" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
      <div className="flex items-center gap-2">
        <span className="w-8 h-8 rounded-lg flex items-center justify-center" style={{ background: accent + '1a', color: accent }}><Icon name={icon} size={16} /></span>
        <span className="text-[12px] font-bold uppercase tracking-wide" style={{ color: 'var(--muted)' }}>{label}</span>
      </div>
      <div className="font-bold tracking-tight mt-2" style={{ fontSize: 24, color: 'var(--ink)' }}>{value}</div>
      <div className="text-[12px] font-medium" style={{ color: 'var(--muted)' }}>{sub}</div>
    </div>
  );
}

Object.assign(window, { Subscriptions });
