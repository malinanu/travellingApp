// MobileQueue.jsx — vendor approval queue: card list → full-screen detail + actions.

const { useState: useStateMQ } = React;

const PRIO_M = {
  high:   { label: 'High',   color: '#e11d48', bg: 'rgba(225,29,72,0.12)' },
  normal: { label: 'Normal', color: '#0b6f86', bg: 'rgba(11,111,134,0.12)' },
  low:    { label: 'Low',    color: '#64748b', bg: 'rgba(100,116,139,0.12)' },
};
const DECIDED_M = {
  approve: { label: 'Approved', color: '#16a34a', icon: 'CheckCircle2' },
  reject:  { label: 'Rejected', color: '#e11d48', icon: 'XCircle' },
  edits:   { label: 'Edits sent', color: '#d97706', icon: 'PencilLine' },
};

function QueueCard({ v, decided, onClick }) {
  const p = PRIO_M[v.priority];
  return (
    <button onClick={onClick}
      className="w-full text-left rounded-2xl active:scale-[0.985] transition-transform"
      style={{ padding: 'var(--pad)', background: 'var(--surface)', border: '1px solid var(--line)', opacity: decided ? 0.6 : 1 }}>
      <div className="flex items-center gap-2">
        <span className="font-mono text-[11px] font-semibold" style={{ color: 'var(--muted)' }}>{v.id}</span>
        <span className="rounded px-1.5 py-0.5 text-[9.5px] font-bold uppercase tracking-wide" style={{ background: p.bg, color: p.color }}>{p.label}</span>
        {!decided && v.flags.length > 0 && (
          <span className="ml-auto flex items-center gap-1 text-[11px] font-bold" style={{ color: '#d97706' }}>
            <Icon name="AlertTriangle" size={12} />{v.flags.length}
          </span>
        )}
        {decided && (
          <span className="ml-auto flex items-center gap-1 text-[11px] font-bold" style={{ color: DECIDED_M[decided].color }}>
            <Icon name={DECIDED_M[decided].icon} size={13} />{DECIDED_M[decided].label}
          </span>
        )}
      </div>
      <div className="font-bold text-[16px] mt-1.5 tracking-tight" style={{ color: 'var(--ink)' }}>{v.name}</div>
      <div className="flex items-center gap-1.5 text-[12.5px] font-medium mt-1 flex-wrap" style={{ color: 'var(--muted)' }}>
        <span>{v.cat}</span><span>·</span>
        <span className="flex items-center gap-0.5"><Icon name="MapPin" size={11} />{v.town}</span><span>·</span>
        <span>{v.submitted}</span>
      </div>
    </button>
  );
}

function MetaTileM({ icon, label, value, ok }) {
  return (
    <div className="rounded-xl px-2.5 py-2.5" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
      <div className="flex items-center gap-1 text-[9.5px] font-bold uppercase tracking-wide" style={{ color: 'var(--muted)' }}>
        <Icon name={icon} size={12} />{label}
      </div>
      <div className="font-bold text-[13px] mt-1" style={{ color: ok === false ? '#e11d48' : 'var(--ink)' }}>{value}</div>
    </div>
  );
}

function QueueDetail({ v, decided, onDecide, onUndo, onClose, onNext, accent }) {
  const d = decided && DECIDED_M[decided];
  return (
    <div className="absolute inset-0 flex flex-col" style={{ background: 'var(--bg)' }}>
      {/* back header */}
      <div className="flex items-center gap-2 flex-shrink-0 px-3 py-2.5" style={{ background: 'var(--surface)', borderBottom: '1px solid var(--line)' }}>
        <button onClick={onClose} className="w-9 h-9 rounded-full flex items-center justify-center active:scale-90 transition-transform" style={{ color: 'var(--ink)' }}>
          <Icon name="ChevronLeft" size={22} />
        </button>
        <span className="font-mono text-[12px] font-semibold" style={{ color: 'var(--muted)' }}>{v.id}</span>
        <span className="rounded px-1.5 py-0.5 text-[9.5px] font-bold uppercase tracking-wide" style={{ background: PRIO_M[v.priority].bg, color: PRIO_M[v.priority].color }}>{PRIO_M[v.priority].label}</span>
      </div>

      {/* body */}
      <div className="flex-1 overflow-y-auto noscroll" style={{ padding: 'var(--pad)' }}>
        {/* hero */}
        <div className="flex items-start gap-3">
          <div className="w-14 h-14 rounded-2xl flex items-center justify-center text-white flex-shrink-0" style={{ background: accent }}>
            <Icon name="Store" size={26} />
          </div>
          <div className="min-w-0 flex-1">
            <h2 className="font-bold tracking-tight" style={{ fontSize: 19, color: 'var(--ink)' }}>{v.name}</h2>
            <div className="text-[12.5px] font-medium mt-0.5" style={{ color: 'var(--muted)' }}>{v.cat} · {v.town}</div>
            <div className="text-[12.5px] font-medium" style={{ color: 'var(--muted)' }}>{v.owner}</div>
          </div>
        </div>

        {/* flags */}
        {v.flags.length > 0 && (
          <div className="rounded-xl px-3.5 py-3 flex items-start gap-2.5 mt-4" style={{ background: 'rgba(217,119,6,0.08)', border: '1px solid rgba(217,119,6,0.25)' }}>
            <Icon name="AlertTriangle" size={16} style={{ color: '#d97706', flexShrink: 0, marginTop: 1 }} />
            <div>
              <div className="font-bold text-[12.5px]" style={{ color: '#b45309' }}>{v.flags.length} item{v.flags.length > 1 ? 's' : ''} need{v.flags.length > 1 ? '' : 's'} attention</div>
              <ul className="text-[12px] font-medium mt-0.5" style={{ color: '#92600a' }}>
                {v.flags.map((f) => <li key={f}>· {f}</li>)}
              </ul>
            </div>
          </div>
        )}

        {/* meta tiles */}
        <div className="grid grid-cols-3 gap-2 mt-4">
          <MetaTileM icon="FileCheck" label="Docs" value={v.docs ? 'Complete' : 'Missing'} ok={v.docs} />
          <MetaTileM icon="Image" label="Photos" value={`${v.photos}`} ok={v.photos >= 4} />
          <MetaTileM icon="Wifi" label="Wi-Fi" value={v.wifi ? `${v.wifi}/10` : 'N/A'} ok={v.wifi != null} />
        </div>

        {/* note */}
        <div className="mt-4">
          <div className="text-[10px] font-bold uppercase tracking-wide mb-1.5" style={{ color: 'var(--muted)' }}>Owner's note</div>
          <p className="text-[14px] leading-relaxed" style={{ color: 'var(--ink-soft)' }}>{v.note}</p>
        </div>

        {/* photos */}
        <div className="mt-4">
          <div className="text-[10px] font-bold uppercase tracking-wide mb-1.5" style={{ color: 'var(--muted)' }}>Submitted photos · {v.photos}</div>
          <div className="flex gap-2 overflow-x-auto noscroll -mx-1 px-1">
            {Array.from({ length: v.photos }).map((_, i) => (
              <div key={i} className="rounded-xl flex-shrink-0 overflow-hidden" style={{ width: 92, height: 92, position: 'relative', background: 'var(--bg)' }}>
                <Illustration kind={['surf', 'sunset', 'cowork', 'clay', 'batik'][i % 5]} scrim={false} />
              </div>
            ))}
          </div>
        </div>
        <div style={{ height: 8 }} />
      </div>

      {/* action bar */}
      <div className="flex-shrink-0 px-3.5 py-3" style={{ background: 'var(--surface)', borderTop: '1px solid var(--line)' }}>
        {d ? (
          <div className="flex items-center gap-2">
            <div className="flex items-center gap-2 font-bold text-[14px] flex-1" style={{ color: d.color }}>
              <Icon name={d.icon} size={18} />{d.label}
            </div>
            <button onClick={onUndo} className="rounded-lg px-3 py-2 text-[13px] font-bold" style={{ background: 'var(--bg)', border: '1px solid var(--line)', color: 'var(--ink)' }}>Undo</button>
            <button onClick={onNext} className="rounded-lg px-3.5 py-2 text-[13px] font-bold text-white flex items-center gap-1" style={{ background: accent }}>Next <Icon name="ArrowRight" size={15} /></button>
          </div>
        ) : (
          <div className="flex flex-col gap-2">
            <button onClick={() => onDecide('approve')}
              className="w-full flex items-center justify-center gap-2 rounded-xl py-3 font-bold text-[15px] text-white active:scale-[0.98] transition-transform whitespace-nowrap" style={{ background: '#16a34a' }}>
              <Icon name="Check" size={18} strokeWidth={2.5} /> Approve &amp; publish
            </button>
            <div className="flex gap-2">
              <button onClick={() => onDecide('edits')}
                className="flex-1 flex items-center justify-center gap-1.5 rounded-xl py-2.5 font-bold text-[13.5px] active:scale-[0.98] transition-transform whitespace-nowrap" style={{ background: 'var(--bg)', border: '1px solid var(--line)', color: 'var(--ink)' }}>
                <Icon name="PencilLine" size={16} /> Request edits
              </button>
              <button onClick={() => onDecide('reject')}
                className="flex-1 flex items-center justify-center gap-1.5 rounded-xl py-2.5 font-bold text-[13.5px] active:scale-[0.98] transition-transform whitespace-nowrap" style={{ background: 'rgba(225,29,72,0.1)', color: '#e11d48' }}>
                <Icon name="X" size={16} strokeWidth={2.5} /> Reject
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function MobileQueue({ accent }) {
  const { VENDOR_QUEUE } = window;
  const [decisions, setDecisions] = useStateMQ({});
  const [openId, setOpenId] = useStateMQ(null);
  const v = VENDOR_QUEUE.find((x) => x.id === openId);
  const pending = VENDOR_QUEUE.filter((x) => !decisions[x.id]).length;

  function decide(action) { setDecisions((dx) => ({ ...dx, [openId]: action })); }
  function undo() { setDecisions((dx) => { const n = { ...dx }; delete n[openId]; return n; }); }
  function next() {
    const rest = VENDOR_QUEUE.filter((x) => !decisions[x.id] && x.id !== openId);
    if (rest[0]) setOpenId(rest[0].id); else setOpenId(null);
  }

  return (
    <div className="relative h-full flex flex-col">
      <div className="flex-shrink-0" style={{ padding: 'var(--pad)', paddingBottom: 8 }}>
        <h1 className="font-bold tracking-tight" style={{ fontSize: 24, color: 'var(--ink)' }}>Approval queue</h1>
        <p className="text-[13px] font-medium mt-0.5" style={{ color: 'var(--muted)' }}>
          <span style={{ color: accent, fontWeight: 700 }}>{pending} pending</span> · sorted by priority
        </p>
      </div>
      <div className="flex-1 overflow-y-auto noscroll flex flex-col" style={{ gap: 'var(--gap)', padding: '0 var(--pad) var(--pad)' }}>
        {VENDOR_QUEUE.map((x) => (
          <QueueCard key={x.id} v={x} decided={decisions[x.id]} onClick={() => setOpenId(x.id)} />
        ))}
      </div>

      {v && (
        <QueueDetail v={v} decided={decisions[openId]} accent={accent}
          onDecide={decide} onUndo={undo} onClose={() => setOpenId(null)} onNext={next} />
      )}
    </div>
  );
}

Object.assign(window, { MobileQueue });
