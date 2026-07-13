// OfferCreator.jsx — create a live deal + manage active offers.

const { useState: useStateOC } = React;

const QUICK_DURATIONS = [
  { label: 'Next 2 hours', mins: 120 },
  { label: 'Today', mins: 0, special: 'today' },
  { label: 'This weekend', mins: 0, special: 'weekend' },
  { label: 'Custom', mins: 0, special: 'custom' },
];

function Field({ label, children, hint }) {
  return (
    <label className="block">
      <div className="flex items-center justify-between mb-1.5">
        <span className="text-[13px] font-extrabold" style={{ color: 'var(--ink)' }}>{label}</span>
        {hint && <span className="text-[12px] font-semibold" style={{ color: 'var(--muted)' }}>{hint}</span>}
      </div>
      {children}
    </label>
  );
}

const inputCls = "w-full rounded-xl px-3.5 py-3 text-[14.5px] font-semibold outline-none transition-all";
function inputStyle(focus) {
  return { background: '#fff', border: '1.5px solid var(--line)', color: 'var(--ink)' };
}

// map an offer's stored type label back to a DISCOUNT_TYPES entry
function offerTypeMeta(label = '') {
  const l = label.toLowerCase();
  if (l.includes('percent')) return { id: 'pct', icon: 'Percent' };
  if (l.includes('amount')) return { id: 'amount', icon: 'BadgeDollarSign' };
  if (l.includes('free')) return { id: 'freebie', icon: 'Gift' };
  if (l.includes('bundle')) return { id: 'bundle', icon: 'Package' };
  return { id: 'pct', icon: 'Tag' };
}

// status chip meta — 'ending' is a live offer that's about to close
const OFFER_STATUS = {
  live:   { label: 'Live',        bg: 'rgba(22,163,74,0.1)',   fg: '#16a34a' },
  ending: { label: 'Ending soon', bg: 'rgba(246,184,30,0.18)', fg: '#a9750a', icon: 'Clock' },
  draft:  { label: 'Draft',       bg: 'rgba(15,61,58,0.07)',   fg: 'var(--ink-soft)', icon: 'PencilLine' },
  ended:  { label: 'Ended',       bg: 'rgba(15,61,58,0.05)',   fg: 'var(--muted)', icon: 'Flag' },
};

function OfferStatusChip({ status }) {
  const s = OFFER_STATUS[status] || OFFER_STATUS.live;
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[10.5px] font-extrabold uppercase tracking-wide"
          style={{ background: s.bg, color: s.fg }}>
      {s.icon
        ? <Icon name={s.icon} size={11} />
        : <span style={{ width: 6, height: 6, borderRadius: '50%', background: s.fg, display: 'inline-block' }} />}
      {s.label}
    </span>
  );
}

// ── one offer, card form ───────────────────────────────────────
function OfferCard({ o, accent, onEdit, onDelete, onStatus, confirming, editing }) {
  const unlimited = o.limit >= 9999;
  const pct = unlimited ? 0 : Math.min(100, Math.round((o.claimed / o.limit) * 100));
  const isLive = o.status === 'live' || o.status === 'ending';
  const isDraft = o.status === 'draft';
  const isEnded = o.status === 'ended';
  const hot = o.status === 'ending' || (isLive && pct >= 85);

  // middle lifecycle action per status
  const statusAction = isLive
    ? { label: 'End', icon: 'Flag', to: 'ended', tone: 'var(--muted)' }
    : isDraft
      ? { label: 'Publish', icon: 'Zap', to: 'live', tone: accent }
      : { label: 'Relaunch', icon: 'RotateCcw', to: 'live', tone: accent };

  return (
    <div className="rounded-2xl bg-white flex flex-col" style={{
      border: editing ? `1.5px solid ${accent}` : isDraft ? '1.5px dashed var(--line)' : '1px solid var(--line)',
      boxShadow: editing ? '0 10px 24px -14px var(--accent-shadow)' : '0 1px 2px rgba(15,61,58,0.04)',
      opacity: isEnded ? 0.75 : 1,
    }}>
      <div className="p-4 flex-1">
        {/* header */}
        <div className="flex items-start gap-3">
          <span className="w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0"
                style={{ background: isEnded ? 'rgba(15,61,58,0.06)' : 'var(--accent-tint)', color: isEnded ? 'var(--muted)' : accent }}>
            <Icon name={offerTypeMeta(o.type).icon} size={19} />
          </span>
          <div className="flex-1 min-w-0">
            <div className="font-extrabold text-[14.5px] leading-snug" style={{ color: 'var(--ink)' }}>{o.title}</div>
            <div className="flex items-center gap-1.5 mt-1.5 flex-wrap">
              <span className="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-extrabold"
                    style={{ background: isEnded ? 'rgba(15,61,58,0.07)' : 'var(--sun)', color: isEnded ? 'var(--muted)' : '#5a3b00' }}>
                <Icon name="Tag" size={10} strokeWidth={2.8} /> {o.value}
              </span>
              <OfferStatusChip status={o.status} />
            </div>
          </div>
        </div>
        {/* claims progress (drafts have none yet) */}
        {isDraft ? (
          <div className="mt-3.5 rounded-xl px-3 py-2.5 text-[12px] font-semibold flex items-center gap-2"
               style={{ background: 'var(--sand)', color: 'var(--muted)' }}>
            <Icon name="EyeOff" size={13} /> Not visible to travelers yet — publish when ready.
          </div>
        ) : (
          <div className="mt-3.5">
            <div className="flex items-center justify-between text-[11.5px] font-bold" style={{ color: 'var(--muted)' }}>
              <span>{unlimited ? `${o.claimed.toLocaleString()} claimed · unlimited` : `${o.claimed}/${o.limit} claimed`}</span>
              <span>{o.redeemed} redeemed</span>
            </div>
            <div className="h-2 rounded-full mt-1.5 overflow-hidden" style={{ background: 'rgba(15,61,58,0.08)' }}>
              <div className="h-full rounded-full transition-all duration-500"
                   style={{ width: unlimited ? '100%' : `${pct}%`,
                            background: isEnded ? 'rgba(15,61,58,0.18)' : unlimited ? 'var(--accent-tint)' : hot ? 'var(--sun)' : accent }} />
            </div>
          </div>
        )}
        {/* schedule */}
        <div className="flex items-center gap-1.5 mt-3 text-[12px] font-semibold" style={{ color: 'var(--muted)' }}>
          <Icon name="CalendarClock" size={13} /> {isDraft ? 'Schedule set on publish' : `${o.start} → ${o.end}`}
        </div>
      </div>
      {/* actions: Edit · lifecycle · Delete */}
      <div className="flex items-stretch" style={{ borderTop: '1px solid var(--line)' }}>
        <button onClick={onEdit}
          className="flex-1 flex items-center justify-center gap-1.5 py-2.5 text-[13px] font-extrabold transition-colors"
          style={{ color: accent }}>
          <Icon name="Pencil" size={14} /> Edit
        </button>
        <span style={{ width: 1, background: 'var(--line)' }} />
        <button onClick={() => onStatus(statusAction.to)}
          className="flex-1 flex items-center justify-center gap-1.5 py-2.5 text-[13px] font-extrabold transition-colors"
          style={{ color: statusAction.tone }}>
          <Icon name={statusAction.icon} size={14} /> {statusAction.label}
        </button>
        <span style={{ width: 1, background: 'var(--line)' }} />
        <button onClick={onDelete}
          className="flex-1 flex items-center justify-center gap-1.5 py-2.5 text-[13px] font-extrabold transition-all"
          style={{ color: confirming ? '#fff' : 'var(--muted)', background: confirming ? '#e11d48' : 'transparent',
                   borderBottomRightRadius: 15 }}>
          <Icon name="Trash2" size={14} /> {confirming ? 'Sure?' : 'Delete'}
        </button>
      </div>
    </div>
  );
}

function OfferCreator({ accent, offers, setOffers }) {
  const { DISCOUNT_TYPES } = window;
  const [dtype, setDtype] = useStateOC('pct');
  const [title, setTitle] = useStateOC('');
  const [amount, setAmount] = useStateOC('20');
  const [dur, setDur] = useStateOC(0);
  const [start, setStart] = useStateOC('07:00');
  const [end, setEnd] = useStateOC('17:00');
  const [limit, setLimit] = useStateOC('50');
  const [unlimited, setUnlimited] = useStateOC(false);
  const [created, setCreated] = useStateOC(false);
  const [editingId, setEditingId] = useStateOC(null);
  const [confirmDel, setConfirmDel] = useStateOC(null);
  const [filter, setFilter] = useStateOC('all');
  const editingOffer = editingId ? offers.find((o) => o.id === editingId) : null;

  const isLiveish = (o) => o.status === 'live' || o.status === 'ending';
  const FILTERS = [
    { id: 'all',   label: 'All',   match: () => true },
    { id: 'live',  label: 'Live',  match: isLiveish },
    { id: 'draft', label: 'Draft', match: (o) => o.status === 'draft' },
    { id: 'ended', label: 'Ended', match: (o) => o.status === 'ended' },
  ];
  const visible = offers.filter(FILTERS.find((f) => f.id === filter).match);

  const valueText = dtype === 'pct' ? `${amount || 0}% off`
    : dtype === 'amount' ? `Rs ${amount || 0} off`
    : dtype === 'freebie' ? 'Free item'
    : 'Bundle deal';

  const previewTitle = title || (dtype === 'pct' ? `${amount || 20}% off coffee` : 'Your new offer');

  function publish() {
    if (editingOffer) {
      // save changes — keep id, claim stats, status and original start
      setOffers(offers.map((o) => o.id !== editingOffer.id ? o : {
        ...o, title: previewTitle,
        type: DISCOUNT_TYPES.find((d) => d.id === dtype)?.label || o.type,
        value: valueText, end: unlimited ? 'No limit' : end,
        limit: unlimited ? 9999 : (parseInt(limit) || o.limit),
      }));
      setEditingId(null);
    } else {
      const newOffer = {
        id: 'o' + Date.now(), title: previewTitle,
        type: DISCOUNT_TYPES.find((d) => d.id === dtype)?.label || 'Offer',
        value: valueText, status: 'live',
        start: QUICK_DURATIONS[dur].label, end: unlimited ? 'No limit' : end,
        claimed: 0, limit: unlimited ? 9999 : (parseInt(limit) || 50), redeemed: 0,
      };
      setOffers([newOffer, ...offers]);
    }
    setCreated(true);
    setTimeout(() => setCreated(false), 2600);
    setTitle('');
  }

  function startEdit(o) {
    setEditingId(o.id);
    setTitle(o.title);
    setDtype(offerTypeMeta(o.type).id);
    const digits = (o.value || '').replace(/\D/g, '');
    if (digits) setAmount(digits);
    setUnlimited(o.limit >= 9999);
    if (o.limit < 9999) setLimit(String(o.limit));
    if (/^\d{1,2}:\d{2}$/.test(o.end)) setEnd(o.end);
    const sm = document.getElementById('scroll-main');
    if (sm) sm.scrollTo({ top: 0, behavior: 'smooth' });
  }

  function cancelEdit() { setEditingId(null); setTitle(''); }

  function removeOffer(id) {
    if (confirmDel !== id) { setConfirmDel(id); setTimeout(() => setConfirmDel(null), 2400); return; }
    setOffers(offers.filter((o) => o.id !== id));
    if (editingId === id) cancelEdit();
    setConfirmDel(null);
  }

  function setStatus(id, to) {
    setOffers(offers.map((o) => {
      if (o.id !== id) return o;
      if (to === 'live') {
        return { ...o, status: 'live',
                 start: (o.start === '—' || o.status === 'ended') ? 'Today' : o.start,
                 end: o.end === '—' ? 'Today 23:59' : o.end };
      }
      return { ...o, status: to };
    }));
  }

  function saveDraft() {
    const draft = {
      id: 'o' + Date.now(), title: previewTitle,
      type: DISCOUNT_TYPES.find((d) => d.id === dtype)?.label || 'Offer',
      value: valueText, status: 'draft', start: '—', end: '—',
      claimed: 0, limit: unlimited ? 9999 : (parseInt(limit) || 50), redeemed: 0,
    };
    setOffers([draft, ...offers]);
    setTitle('');
    setFilter('draft');
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="font-extrabold tracking-tight" style={{ fontSize: 28, color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>
          {editingOffer ? 'Edit offer' : 'Create a live offer'}
        </h1>
        <p className="text-[14.5px] font-semibold mt-1" style={{ color: 'var(--muted)' }}>
          {editingOffer ? 'Claim stats and status stay untouched — only the details change.' : "Spin up a deal in under a minute — it goes live on every traveler's map instantly."}
        </p>
      </div>

      {editingOffer && (
        <div className="rounded-xl px-4 py-3 flex items-center gap-3" style={{ background: 'var(--accent-tint)', border: `1.5px solid ${accent}` }}>
          <Icon name="Pencil" size={16} style={{ color: accent }} />
          <span className="text-[13.5px] font-extrabold flex-1 truncate" style={{ color: 'var(--ink)' }}>
            Editing: {editingOffer.title}
          </span>
          <button onClick={cancelEdit} className="flex items-center gap-1 text-[12.5px] font-extrabold" style={{ color: accent }}>
            <Icon name="X" size={14} strokeWidth={2.5} /> Cancel
          </button>
        </div>
      )}

      <div className="grid gap-6" style={{ gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 360px)' }}>
        {/* form */}
        <div className="rounded-2xl p-6 bg-white space-y-5" style={{ border: '1px solid var(--line)' }}>
          {/* discount type */}
          <Field label="Discount type">
            <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
              {DISCOUNT_TYPES.map((d) => {
                const on = dtype === d.id;
                return (
                  <button key={d.id} onClick={() => setDtype(d.id)}
                    className="rounded-xl p-3 flex flex-col items-center gap-1.5 transition-all active:scale-[0.97]"
                    style={{ background: on ? accent : '#fff', color: on ? '#fff' : 'var(--ink)',
                             border: on ? `1.5px solid ${accent}` : '1.5px solid var(--line)' }}>
                    <Icon name={d.icon} size={20} />
                    <span className="text-[12px] font-extrabold text-center leading-tight">{d.label}</span>
                  </button>
                );
              })}
            </div>
          </Field>

          {/* title + amount */}
          <div className="grid gap-4" style={{ gridTemplateColumns: (dtype === 'pct' || dtype === 'amount') ? '1fr 120px' : '1fr' }}>
            <Field label="Offer title" hint="what travelers see">
              <input className={inputCls} style={inputStyle()} value={title}
                     onChange={(e) => setTitle(e.target.value)}
                     placeholder={dtype === 'pct' ? 'e.g. 20% off coffee' : 'Name your deal'} />
            </Field>
            {(dtype === 'pct' || dtype === 'amount') && (
              <Field label={dtype === 'pct' ? 'Percent' : 'Amount'}>
                <div className="relative">
                  <input className={inputCls} style={{ ...inputStyle(), paddingRight: 34 }} value={amount}
                         inputMode="numeric" onChange={(e) => setAmount(e.target.value.replace(/\D/g, ''))} />
                  <span className="absolute right-3.5 top-1/2 -translate-y-1/2 font-extrabold text-[14px]" style={{ color: 'var(--muted)' }}>
                    {dtype === 'pct' ? '%' : 'Rs'}
                  </span>
                </div>
              </Field>
            )}
          </div>

          {/* duration */}
          <Field label="Duration">
            <div className="flex flex-wrap gap-2">
              {QUICK_DURATIONS.map((q, i) => {
                const on = dur === i;
                return (
                  <button key={q.label} onClick={() => setDur(i)}
                    className="rounded-full px-3.5 py-2 text-[13px] font-extrabold transition-all active:scale-95"
                    style={{ background: on ? 'var(--accent-tint)' : '#fff', color: on ? accent : 'var(--ink-soft)',
                             border: on ? `1.5px solid ${accent}` : '1.5px solid var(--line)' }}>
                    {q.label}
                  </button>
                );
              })}
            </div>
          </Field>

          {/* start / end time */}
          <div className="grid grid-cols-2 gap-4">
            <Field label="Start time">
              <input type="time" className={inputCls} style={inputStyle()} value={start} onChange={(e) => setStart(e.target.value)} />
            </Field>
            <Field label="End time">
              <input type="time" className={inputCls} style={inputStyle()} value={end} onChange={(e) => setEnd(e.target.value)} />
            </Field>
          </div>

          {/* claim limit */}
          <Field label="Claim limit" hint="cap how many travelers can claim">
            <div className="flex items-center gap-3">
              <input className={inputCls} style={{ ...inputStyle(), opacity: unlimited ? 0.45 : 1, flex: 1 }}
                     disabled={unlimited} value={unlimited ? '∞' : limit} inputMode="numeric"
                     onChange={(e) => setLimit(e.target.value.replace(/\D/g, ''))} />
              <button onClick={() => setUnlimited(!unlimited)}
                className="flex items-center gap-2 rounded-xl px-3.5 py-3 text-[13.5px] font-extrabold transition-all flex-shrink-0"
                style={{ background: unlimited ? accent : '#fff', color: unlimited ? '#fff' : 'var(--ink-soft)',
                         border: unlimited ? `1.5px solid ${accent}` : '1.5px solid var(--line)' }}>
                <Icon name={unlimited ? 'CheckCircle2' : 'Infinity'} size={16} /> Unlimited
              </button>
            </div>
          </Field>
        </div>

        {/* live preview */}
        <div className="space-y-4">
          <div className="rounded-2xl p-5 bg-white sticky top-2" style={{ border: '1px solid var(--line)' }}>
            <div className="text-[12px] font-extrabold uppercase tracking-wide flex items-center gap-1.5" style={{ color: 'var(--muted)' }}>
              <Icon name="Smartphone" size={14} /> Traveler preview
            </div>
            {/* mock map card */}
            <div className="mt-3 rounded-2xl overflow-hidden" style={{ border: '1px solid var(--line)' }}>
              <div style={{ height: 84, background: 'var(--sand)', position: 'relative' }}>
                <Illustration kind="cowork" scrim={false} />
                <div className="absolute left-3 top-3 inline-flex items-center gap-1 rounded-full px-2 py-1"
                     style={{ background: 'var(--sun)', color: '#5a3b00' }}>
                  <Icon name="Tag" size={12} strokeWidth={2.5} /><span className="font-extrabold text-[11px]">{valueText}</span>
                </div>
              </div>
              <div className="p-3">
                <div className="font-extrabold text-[14px]" style={{ color: 'var(--ink)' }}>{window.VENDOR.name}</div>
                <div className="text-[12.5px] font-bold mt-1 leading-snug" style={{ color: accent }}>{previewTitle}</div>
                <div className="text-[11.5px] font-semibold mt-1" style={{ color: 'var(--muted)' }}>
                  {QUICK_DURATIONS[dur].label} · {unlimited ? 'Unlimited claims' : `${limit} claims left`}
                </div>
              </div>
            </div>
            <button onClick={publish}
              className="mt-4 w-full rounded-xl py-3.5 font-extrabold text-[15px] flex items-center justify-center gap-2 text-white active:scale-[0.98] transition-transform"
              style={{ background: created ? '#16a34a' : accent, boxShadow: '0 10px 22px -10px var(--accent-shadow)' }}>
              {created
                ? <><Icon name="Check" size={18} strokeWidth={3} /> {editingOffer ? 'Saved!' : 'Offer is live!'}</>
                : editingOffer
                  ? <><Icon name="Check" size={18} strokeWidth={2.5} /> Save changes</>
                  : <><Icon name="Zap" size={18} /> Publish offer</>}
            </button>
            {editingOffer ? (
              <button onClick={cancelEdit}
                className="mt-2 w-full rounded-xl py-2.5 font-extrabold text-[13.5px]"
                style={{ background: 'var(--sand)', border: '1.5px solid var(--line)', color: 'var(--ink)' }}>
                Cancel editing
              </button>
            ) : (
              <>
                <button onClick={saveDraft}
                  className="mt-2 w-full rounded-xl py-2.5 font-extrabold text-[13.5px] flex items-center justify-center gap-1.5"
                  style={{ background: '#fff', border: '1.5px dashed var(--line)', color: 'var(--ink-soft)' }}>
                  <Icon name="PencilLine" size={15} /> Save as draft
                </button>
                <p className="text-[11.5px] font-semibold text-center mt-2" style={{ color: 'var(--muted)' }}>Publish goes live instantly · drafts stay private</p>
              </>
            )}
          </div>
        </div>
      </div>

      {/* current offers */}
      <div>
        <div className="flex items-center gap-2.5 mb-1 flex-wrap">
          <h3 className="font-extrabold text-[18px]" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Your offers</h3>
          <span className="rounded-full px-2 py-0.5 text-[12px] font-extrabold" style={{ background: 'var(--accent-tint)', color: accent }}>{offers.length}</span>
          {/* status filter */}
          <div className="ml-auto flex items-center gap-1.5">
            {FILTERS.map((f) => {
              const on = filter === f.id;
              const n = offers.filter(f.match).length;
              return (
                <button key={f.id} onClick={() => setFilter(f.id)}
                  className="rounded-full px-3 py-1.5 text-[12.5px] font-extrabold transition-all active:scale-95"
                  style={{ background: on ? accent : '#fff', color: on ? '#fff' : 'var(--ink-soft)',
                           border: on ? `1.5px solid ${accent}` : '1.5px solid var(--line)' }}>
                  {f.label} <span style={{ opacity: 0.7 }}>{n}</span>
                </button>
              );
            })}
          </div>
        </div>
        <p className="text-[13px] font-semibold mb-4" style={{ color: 'var(--muted)' }}>Live offers show on every traveler's map · drafts stay private · ended offers keep their stats.</p>
        {visible.length === 0 ? (
          <div className="rounded-2xl bg-white px-6 py-10 flex flex-col items-center text-center" style={{ border: '2px dashed var(--line)' }}>
            <span className="w-12 h-12 rounded-2xl flex items-center justify-center mb-3" style={{ background: 'var(--accent-tint)', color: accent }}>
              <Icon name={filter === 'draft' ? 'PencilLine' : filter === 'ended' ? 'Flag' : 'Tag'} size={22} />
            </span>
            <div className="font-extrabold text-[15px]" style={{ color: 'var(--ink)' }}>
              {filter === 'all' ? 'No offers yet' : `No ${FILTERS.find((f) => f.id === filter).label.toLowerCase()} offers`}
            </div>
            <p className="text-[13px] font-semibold mt-1" style={{ color: 'var(--muted)' }}>
              {filter === 'draft' ? 'Use "Save as draft" above to park an idea.' : 'Publish your first one above — it goes live instantly.'}
            </p>
          </div>
        ) : (
          <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(320px, 100%), 1fr))' }}>
            {visible.map((o) => (
              <OfferCard key={o.id} o={o} accent={accent}
                editing={editingId === o.id}
                confirming={confirmDel === o.id}
                onEdit={() => startEdit(o)}
                onStatus={(to) => setStatus(o.id, to)}
                onDelete={() => removeOffer(o.id)} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

Object.assign(window, { OfferCreator });
