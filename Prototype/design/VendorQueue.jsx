// VendorQueue.jsx — approval queue with master list + detail panel + actions.

const { useState: useStateVQ } = React;

const PRIO = {
  high:   { label: 'High',   color: '#e11d48', bg: 'rgba(225,29,72,0.1)' },
  normal: { label: 'Normal', color: '#0b6f86', bg: 'rgba(11,111,134,0.1)' },
  low:    { label: 'Low',    color: '#64748b', bg: 'rgba(100,116,139,0.1)' },
};

function QueueRow({ v, active, decided, onClick }) {
  const p = PRIO[v.priority];
  return (
    <button onClick={onClick}
      className="w-full text-left px-4 py-3 transition-colors"
      style={{ background: active ? 'var(--surface)' : 'transparent',
               borderLeft: active ? '3px solid var(--accent)' : '3px solid transparent',
               borderBottom: '1px solid var(--line)', opacity: decided ? 0.5 : 1 }}>
      <div className="flex items-center gap-2">
        <span className="font-mono text-[11px] font-semibold" style={{ color: 'var(--muted)' }}>{v.id}</span>
        <span className="rounded px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide" style={{ background: p.bg, color: p.color }}>{p.label}</span>
        {v.flags.length > 0 && (
          <span className="ml-auto flex items-center gap-1 text-[11px] font-bold" style={{ color: '#d97706' }}>
            <Icon name="AlertTriangle" size={12} />{v.flags.length}
          </span>
        )}
        {decided && <span className="ml-auto text-[11px] font-bold" style={{ color: decided === 'approve' ? '#16a34a' : decided === 'reject' ? '#e11d48' : '#d97706' }}>
          {decided === 'approve' ? 'Approved' : decided === 'reject' ? 'Rejected' : 'Edits sent'}</span>}
      </div>
      <div className="font-bold text-[14px] mt-1" style={{ color: 'var(--ink)' }}>{v.name}</div>
      <div className="flex items-center gap-2 text-[12px] font-medium mt-0.5" style={{ color: 'var(--muted)' }}>
        <span>{v.cat}</span><span>·</span><span className="flex items-center gap-0.5"><Icon name="MapPin" size={11} />{v.town}</span><span>·</span><span>{v.submitted}</span>
      </div>
    </button>
  );
}

function MetaTile({ icon, label, value, ok }) {
  return (
    <div className="rounded-lg px-3 py-2.5" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
      <div className="flex items-center gap-1.5 text-[11px] font-bold uppercase tracking-wide" style={{ color: 'var(--muted)' }}>
        <Icon name={icon} size={13} />{label}
      </div>
      <div className="font-bold text-[14px] mt-1 flex items-center gap-1.5" style={{ color: ok === false ? '#e11d48' : 'var(--ink)' }}>
        {value}
      </div>
    </div>
  );
}

function VendorQueue({ accent }) {
  const { VENDOR_QUEUE } = window;
  const [decisions, setDecisions] = useStateVQ({});
  const [activeId, setActiveId] = useStateVQ(VENDOR_QUEUE[0].id);
  const v = VENDOR_QUEUE.find((x) => x.id === activeId);
  const decided = decisions[activeId];

  function decide(action) {
    setDecisions((d) => ({ ...d, [activeId]: action }));
    // advance to next undecided
    const rest = VENDOR_QUEUE.filter((x) => !decisions[x.id] && x.id !== activeId);
    if (rest[0]) setTimeout(() => setActiveId(rest[0].id), 350);
  }

  const pendingCount = VENDOR_QUEUE.filter((x) => !decisions[x.id]).length;

  return (
    <div className="h-full flex flex-col">
      <div className="flex items-end justify-between gap-4 mb-4">
        <div>
          <h1 className="font-bold tracking-tight" style={{ fontSize: 22, color: 'var(--ink)' }}>Vendor approval queue</h1>
          <p className="text-[13px] font-medium mt-0.5" style={{ color: 'var(--muted)' }}>{pendingCount} businesses awaiting review · sorted by priority</p>
        </div>
        <div className="flex items-center gap-2 text-[12px] font-semibold" style={{ color: 'var(--muted)' }}>
          <kbd className="rounded px-1.5 py-0.5 font-mono" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>A</kbd> approve
          <kbd className="rounded px-1.5 py-0.5 font-mono" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>R</kbd> reject
        </div>
      </div>

      <div className="flex-1 grid gap-4 min-h-0" style={{ gridTemplateColumns: 'minmax(300px, 360px) 1fr' }}>
        {/* list */}
        <div className="rounded-xl overflow-hidden flex flex-col" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
          <div className="px-4 py-2.5 text-[11px] font-bold uppercase tracking-wide flex items-center justify-between" style={{ color: 'var(--muted)', borderBottom: '1px solid var(--line)' }}>
            <span>Incoming · {VENDOR_QUEUE.length}</span>
            <Icon name="ListFilter" size={14} />
          </div>
          <div className="overflow-y-auto flex-1">
            {VENDOR_QUEUE.map((x) => (
              <QueueRow key={x.id} v={x} active={x.id === activeId} decided={decisions[x.id]} onClick={() => setActiveId(x.id)} />
            ))}
          </div>
        </div>

        {/* detail */}
        <div className="rounded-xl overflow-hidden flex flex-col" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
          {/* hero */}
          <div className="px-6 py-5" style={{ borderBottom: '1px solid var(--line)' }}>
            <div className="flex items-start gap-4">
              <div className="w-14 h-14 rounded-xl flex items-center justify-center text-white flex-shrink-0" style={{ background: accent }}>
                <Icon name="Store" size={26} />
              </div>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="font-mono text-[12px] font-semibold" style={{ color: 'var(--muted)' }}>{v.id}</span>
                  <span className="rounded px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide" style={{ background: PRIO[v.priority].bg, color: PRIO[v.priority].color }}>{PRIO[v.priority].label} priority</span>
                </div>
                <h2 className="font-bold tracking-tight mt-1" style={{ fontSize: 20, color: 'var(--ink)' }}>{v.name}</h2>
                <div className="flex items-center gap-2 text-[13px] font-medium mt-0.5" style={{ color: 'var(--muted)' }}>
                  <span>{v.cat}</span><span>·</span><span className="flex items-center gap-1"><Icon name="MapPin" size={12} />{v.town}</span><span>·</span><span>{v.owner}</span>
                </div>
              </div>
            </div>
          </div>

          {/* body */}
          <div className="px-6 py-5 overflow-y-auto flex-1 space-y-5">
            {v.flags.length > 0 && (
              <div className="rounded-lg px-4 py-3 flex items-start gap-2.5" style={{ background: 'rgba(217,119,6,0.08)', border: '1px solid rgba(217,119,6,0.25)' }}>
                <Icon name="AlertTriangle" size={17} style={{ color: '#d97706', flexShrink: 0, marginTop: 1 }} />
                <div>
                  <div className="font-bold text-[13px]" style={{ color: '#b45309' }}>{v.flags.length} item{v.flags.length>1?'s':''} need attention</div>
                  <ul className="text-[12.5px] font-medium mt-0.5" style={{ color: '#92600a' }}>
                    {v.flags.map((f) => <li key={f}>· {f}</li>)}
                  </ul>
                </div>
              </div>
            )}

            <div className="grid grid-cols-3 gap-2.5">
              <MetaTile icon="FileCheck" label="Documents" value={v.docs ? 'Complete' : 'Missing'} ok={v.docs} />
              <MetaTile icon="Image" label="Photos" value={`${v.photos} uploaded`} ok={v.photos >= 4} />
              <MetaTile icon="Wifi" label="Wi-Fi score" value={v.wifi ? `${v.wifi}/10` : 'N/A'} ok={v.wifi != null} />
            </div>

            <div>
              <div className="text-[11px] font-bold uppercase tracking-wide mb-1.5" style={{ color: 'var(--muted)' }}>Owner's note</div>
              <p className="text-[14px] leading-relaxed" style={{ color: 'var(--ink-soft)' }}>{v.note}</p>
            </div>

            {/* photo strip */}
            <div>
              <div className="text-[11px] font-bold uppercase tracking-wide mb-1.5" style={{ color: 'var(--muted)' }}>Submitted photos</div>
              <div className="flex gap-2">
                {Array.from({ length: Math.min(v.photos, 5) }).map((_, i) => (
                  <div key={i} className="rounded-lg flex-1 overflow-hidden" style={{ aspectRatio: '1', position: 'relative', background: 'var(--bg)' }}>
                    <Illustration kind={['surf', 'sunset', 'cowork', 'clay', 'batik'][i % 5]} scrim={false} />
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* action bar */}
          <div className="px-6 py-4 flex items-center gap-3" style={{ borderTop: '1px solid var(--line)', background: 'var(--bg)' }}>
            {decided ? (
              <div className="flex items-center gap-2 font-bold text-[14px]" style={{ color: decided === 'approve' ? '#16a34a' : decided === 'reject' ? '#e11d48' : '#d97706' }}>
                <Icon name={decided === 'approve' ? 'CheckCircle2' : decided === 'reject' ? 'XCircle' : 'PencilLine'} size={18} />
                {decided === 'approve' ? 'Vendor approved & published' : decided === 'reject' ? 'Application rejected' : 'Edit request sent to vendor'}
                <button onClick={() => setDecisions((d) => { const n = { ...d }; delete n[activeId]; return n; })}
                        className="ml-2 text-[12px] font-semibold underline" style={{ color: 'var(--muted)' }}>Undo</button>
              </div>
            ) : (
              <>
                <button data-qa="approve" onClick={() => decide('approve')}
                  className="flex items-center gap-2 rounded-lg px-5 py-2.5 font-bold text-[14px] text-white active:scale-[0.98] transition-transform"
                  style={{ background: '#16a34a' }}>
                  <Icon name="Check" size={17} strokeWidth={2.5} /> Approve
                </button>
                <button data-qa="edits" onClick={() => decide('edits')}
                  className="flex items-center gap-2 rounded-lg px-5 py-2.5 font-bold text-[14px] active:scale-[0.98] transition-transform"
                  style={{ background: 'var(--surface)', border: '1px solid var(--line)', color: 'var(--ink)' }}>
                  <Icon name="PencilLine" size={17} /> Request edits
                </button>
                <button data-qa="reject" onClick={() => decide('reject')}
                  className="flex items-center gap-2 rounded-lg px-5 py-2.5 font-bold text-[14px] active:scale-[0.98] transition-transform ml-auto"
                  style={{ background: 'rgba(225,29,72,0.1)', color: '#e11d48' }}>
                  <Icon name="X" size={17} strokeWidth={2.5} /> Reject
                </button>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

Object.assign(window, { VendorQueue });
