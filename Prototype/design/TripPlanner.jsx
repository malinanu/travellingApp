// TripPlanner.jsx — 'Trip' tab: day-by-day itinerary with times, budgets,
// travel legs (walk / tuk-tuk) and real Google Maps direction deep-links.
// Props: style (travel style id), onOpenPlace(spottoId) → opens business drawer.

const { useState: useStateTrip, useEffect: useEffectTrip, useMemo: useMemoTrip } = React;

const TRIP_LS_KEY = 'spotto_trip';

function tripWithDefaults(t) {
  return { startDate: tripTodayISO(), collaborators: [], code: 'TRP-' + Math.random().toString(36).slice(2, 5).toUpperCase(), ...t };
}
function tripLoad() {
  try { return tripWithDefaults(JSON.parse(localStorage.getItem(TRIP_LS_KEY) || 'null') || window.TRIP_SEED); }
  catch { return tripWithDefaults(window.TRIP_SEED); }
}

// local (not UTC) yyyy-mm-dd so the picker's "today" matches the default
function tripTodayISO() {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

// resolve a traveller id → {name, initial, color}; 'me' → ME
function tripWho(id) {
  if (!id || id === 'me') return window.ME;
  return (window.TRAVELERS || []).find((x) => x.id === id) || null;
}

// phone contacts NOT on Spotto yet — invite them with a link
const TRIP_CONTACTS = [
  { id: 'c_ava',   name: 'Ava Fernando',  phone: '+94 77 214 8890', initial: 'A', color: '#f0682f' },
  { id: 'c_liam',  name: 'Liam Perera',   phone: '+94 76 551 2034', initial: 'L', color: '#0369a1' },
  { id: 'c_nadia', name: 'Nadia Khan',    phone: '+44 7700 900221', initial: 'N', color: '#7c5cff' },
  { id: 'c_ruth',  name: 'Ruth Silva',    phone: '+61 412 776 305', initial: 'R', color: '#0b6f86' },
];

let tripUid = 100;
const newId = (p) => `${p}${++tripUid}_${Date.now() % 10000}`;

// ── small bits ─────────────────────────────────────────────────
function TimeChip({ mins, onChange }) {
  const hh = String(Math.floor(mins / 60)).padStart(2, '0');
  const mm = String(mins % 60).padStart(2, '0');
  return (
    <label className="relative inline-flex items-center gap-1 rounded-full px-2.5 py-1 cursor-pointer flex-shrink-0"
           style={{ background: 'var(--accent-tint)' }}>
      <Icon name="Clock" size={12} style={{ color: 'var(--accent)' }} />
      <span className="text-[12px] font-extrabold whitespace-nowrap" style={{ color: 'var(--accent)' }}>{window.tripFmtTime(mins)}</span>
      <input type="time" value={`${hh}:${mm}`}
        onChange={(e) => { const [h, m] = e.target.value.split(':').map(Number); if (!isNaN(h)) onChange(h * 60 + (m || 0)); }}
        className="absolute inset-0 opacity-0 w-full cursor-pointer" />
    </label>
  );
}

function LegRow({ leg }) {
  return (
    <div className="flex items-center gap-3 pl-[26px]" style={{ height: 44 }}>
      <div className="self-stretch w-0 border-l-2 border-dashed" style={{ borderColor: 'rgba(15,61,58,0.18)' }} />
      <Icon name={leg.icon} size={15} style={{ color: 'var(--muted)' }} />
      <span className="text-[12px] font-bold whitespace-nowrap" style={{ color: 'var(--muted)' }}>{leg.mins} min {leg.mode.toLowerCase()} · {leg.dist}</span>
      <a href={leg.gmaps} target="_blank" rel="noopener"
         className="ml-auto flex items-center gap-1 rounded-full px-2.5 py-1 text-[11.5px] font-extrabold whitespace-nowrap active:scale-95 transition-transform"
         style={{ background: '#fff', color: 'var(--accent)', border: '1.5px solid rgba(15,61,58,0.1)' }}>
        <Icon name="Navigation" size={12} /> Directions
      </a>
    </div>
  );
}

function StopCard({ stop, place, first, last, by, onTime, onUp, onDown, onRemove, onOpen }) {
  const credit = by && by.id !== 'me' ? by : null;
  return (
    <div className="rounded-3xl bg-white p-3.5 flex items-center gap-3" style={{ boxShadow: '0 2px 14px -8px rgba(15,61,58,0.25)' }}>
      <button onClick={onOpen} disabled={!place.spottoId}
        className="w-12 h-12 rounded-2xl flex items-center justify-center flex-shrink-0 active:scale-95 transition-transform"
        style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}>
        <Icon name={place.icon || 'MapPin'} size={22} />
      </button>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-extrabold text-[14px] truncate" style={{ color: 'var(--ink)' }}>{place.name}</span>
        </div>
        <div className="text-[12px] font-semibold truncate" style={{ color: 'var(--muted)' }}>{place.type} · {window.tripFmtRs(place.cost)}</div>
        <div className="mt-1.5 flex items-center gap-2">
          <TimeChip mins={stop.time} onChange={onTime} />
          {credit && (
            <span className="inline-flex items-center gap-1 rounded-full pr-2 pl-0.5 py-0.5" style={{ background: 'var(--sand)' }} title={`Added by ${credit.name}`}>
              <span style={{ width: 16, height: 16, borderRadius: '50%', background: credit.color, color: '#fff', fontSize: 9, fontWeight: 800, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>{credit.initial}</span>
              <span className="text-[10.5px] font-bold" style={{ color: 'var(--muted)' }}>{credit.name}</span>
            </span>
          )}
        </div>
      </div>
      <div className="flex flex-col items-center gap-0.5 flex-shrink-0">
        <button onClick={onUp} disabled={first} className="w-7 h-7 rounded-lg flex items-center justify-center active:scale-90 transition-transform"
          style={{ color: first ? 'rgba(15,61,58,0.18)' : 'var(--ink-soft)', background: 'var(--sand)' }}>
          <Icon name="ChevronUp" size={16} strokeWidth={3} />
        </button>
        <button onClick={onDown} disabled={last} className="w-7 h-7 rounded-lg flex items-center justify-center active:scale-90 transition-transform"
          style={{ color: last ? 'rgba(15,61,58,0.18)' : 'var(--ink-soft)', background: 'var(--sand)' }}>
          <Icon name="ChevronDown" size={16} strokeWidth={3} />
        </button>
      </div>
      <button onClick={onRemove} className="w-7 h-7 rounded-lg flex items-center justify-center flex-shrink-0 active:scale-90 transition-transform" style={{ color: '#c2666f' }}>
        <Icon name="X" size={15} strokeWidth={3} />
      </button>
    </div>
  );
}

// ── add-stop bottom sheet ──────────────────────────────────────
function AddStopSheet({ open, onClose, onAdd, styleId, usedIds }) {
  const [q, setQ] = useStateTrip('');
  useEffectTrip(() => { if (open) setQ(''); }, [open]);
  if (!open) return null;

  const catalog = window.TRIP_CATALOG;
  const query = q.trim().toLowerCase();
  const results = query
    ? catalog.filter((p) => (p.name + ' ' + p.type).toLowerCase().includes(query))
    : [];
  const suggested = catalog.filter((p) => (p.styles || []).includes(styleId) && !usedIds.includes(p.id)).slice(0, 5);
  const list = query ? results : suggested;

  return (
    <div className="absolute inset-0 z-50 flex flex-col justify-end">
      <div className="absolute inset-0" style={{ background: 'rgba(16,61,58,0.45)' }} onClick={onClose} />
      <div className="relative rounded-t-[28px] flex flex-col" style={{ background: 'var(--sand)', maxHeight: '78%', boxShadow: '0 -16px 44px -20px rgba(16,61,58,0.6)' }}>
        <div className="w-10 h-1.5 rounded-full mx-auto mt-3 mb-2" style={{ background: 'rgba(15,61,58,0.15)' }} />
        <div className="px-5 pb-3">
          <h3 className="font-extrabold text-[19px]" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Add a stop</h3>
          <div className="flex items-center gap-2.5 rounded-2xl px-3.5 mt-3" style={{ height: 48, background: '#fff', border: '1.5px solid rgba(15,61,58,0.12)' }}>
            <Icon name="Search" size={17} style={{ color: 'var(--muted)' }} />
            <input autoFocus value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search any place — hotel, café, beach…"
              className="flex-1 bg-transparent outline-none text-[14px] font-bold w-full" style={{ color: 'var(--ink)' }} />
            {q && <button onClick={() => setQ('')} style={{ color: 'var(--muted)' }}><Icon name="X" size={15} /></button>}
          </div>
        </div>
        <div className="flex-1 overflow-y-auto px-5 pb-8">
          {!query && suggested.length > 0 && (
            <div className="flex items-center gap-1.5 mb-2 px-1">
              <Icon name="Sparkles" size={13} style={{ color: 'var(--accent)' }} />
              <span className="text-[11.5px] font-extrabold uppercase tracking-wide" style={{ color: 'var(--muted)' }}>Suggested for your style</span>
            </div>
          )}
          <div className="space-y-2">
            {list.map((p) => {
              const used = usedIds.includes(p.id);
              return (
                <button key={p.id} onClick={() => !used && onAdd(p)} disabled={used}
                  className="w-full rounded-2xl bg-white p-3 flex items-center gap-3 text-left active:scale-[0.99] transition-transform"
                  style={{ opacity: used ? 0.45 : 1, border: '1.5px solid rgba(15,61,58,0.08)' }}>
                  <span className="w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0" style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}>
                    <Icon name={p.icon} size={19} />
                  </span>
                  <span className="flex-1 min-w-0">
                    <span className="block font-extrabold text-[13.5px] truncate" style={{ color: 'var(--ink)' }}>{p.name}</span>
                    <span className="block text-[12px] font-semibold" style={{ color: 'var(--muted)' }}>{p.type} · {window.tripFmtRs(p.cost)}</span>
                  </span>
                  {used
                    ? <span className="text-[11px] font-extrabold whitespace-nowrap" style={{ color: 'var(--muted)' }}>Added</span>
                    : <Icon name="Plus" size={18} style={{ color: 'var(--accent)' }} />}
                </button>
              );
            })}
            {query && (
              <button onClick={() => onAdd({ id: newId('c'), name: q.trim(), type: 'Custom stop', icon: 'MapPin', lat: 5.9749, lng: 80.4293, cost: 0, customFlag: true })}
                className="w-full rounded-2xl p-3 flex items-center gap-3 text-left active:scale-[0.99] transition-transform"
                style={{ background: 'var(--accent-tint)', border: '1.5px dashed var(--accent)' }}>
                <span className="w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0" style={{ background: '#fff', color: 'var(--accent)' }}>
                  <Icon name="MapPin" size={19} />
                </span>
                <span className="flex-1 min-w-0">
                  <span className="block font-extrabold text-[13.5px] truncate" style={{ color: 'var(--ink)' }}>Add “{q.trim()}”</span>
                  <span className="block text-[12px] font-semibold" style={{ color: 'var(--ink-soft)' }}>Custom stop · pin near Weligama town</span>
                </span>
                <Icon name="Plus" size={18} style={{ color: 'var(--accent)' }} />
              </button>
            )}
            {query && results.length === 0 && (
              <p className="text-center text-[12.5px] font-bold pt-1" style={{ color: 'var(--muted)' }}>No matches in Spotto — add it as a custom stop above.</p>
            )}
            {!query && suggested.length === 0 && (
              <p className="text-center text-[12.5px] font-bold pt-2" style={{ color: 'var(--muted)' }}>Type to search hotels, cafés, beaches, bars…</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

// ── inline calendar date picker (tap field → pick date & month) ──
function DateField({ value, onChange }) {
  const [open, setOpen] = useStateTrip(false);
  const sel = new Date(value + 'T00:00:00');
  const [mon, setMon] = useStateTrip(() => new Date(sel.getFullYear(), sel.getMonth(), 1));
  const keyOf = (d) => `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`;
  const today = new Date(); today.setHours(0, 0, 0, 0);
  const y = mon.getFullYear(), m = mon.getMonth();
  const startDow = new Date(y, m, 1).getDay();
  const dim = new Date(y, m + 1, 0).getDate();
  const cells = [];
  for (let i = 0; i < startDow; i++) cells.push(null);
  for (let d = 1; d <= dim; d++) cells.push(d);
  const WD = ['S', 'M', 'T', 'W', 'T', 'F', 'S'];
  function pick(d) {
    const dt = new Date(y, m, d);
    onChange(`${dt.getFullYear()}-${String(dt.getMonth() + 1).padStart(2, '0')}-${String(dt.getDate()).padStart(2, '0')}`);
    setOpen(false);
  }
  return (
    <div className="mb-4">
      <button type="button" onClick={() => setOpen((o) => !o)}
        className="relative flex items-center gap-2.5 rounded-2xl px-3.5 w-full active:scale-[0.99] transition-transform"
        style={{ height: 50, background: '#fff', border: `1.5px solid ${open ? 'var(--accent)' : 'rgba(15,61,58,0.12)'}` }}>
        <Icon name="CalendarDays" size={17} style={{ color: open ? 'var(--accent)' : 'var(--muted)' }} />
        <span className="flex-1 text-left text-[14.5px] font-bold" style={{ color: 'var(--ink)' }}>
          {sel.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric', year: 'numeric' })}
        </span>
        <Icon name="ChevronDown" size={16} style={{ color: 'var(--muted)', transform: open ? 'rotate(180deg)' : 'none', transition: 'transform 200ms' }} />
      </button>
      {open && (
        <div className="mt-2 rounded-2xl p-3" style={{ background: '#fff', boxShadow: '0 12px 30px -14px rgba(16,61,58,0.4)', border: '1.5px solid rgba(15,61,58,0.08)' }}>
          <div className="flex items-center justify-between mb-2">
            <button type="button" onClick={() => setMon(new Date(y, m - 1, 1))} aria-label="Previous month" className="w-8 h-8 rounded-lg flex items-center justify-center active:scale-90 transition-transform" style={{ background: 'var(--sand)', color: 'var(--ink)' }}>
              <Icon name="ChevronLeft" size={16} strokeWidth={2.5} />
            </button>
            <span className="font-extrabold text-[15px] whitespace-nowrap" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>
              {mon.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })}
            </span>
            <button type="button" onClick={() => setMon(new Date(y, m + 1, 1))} aria-label="Next month" className="w-8 h-8 rounded-lg flex items-center justify-center active:scale-90 transition-transform" style={{ background: 'var(--sand)', color: 'var(--ink)' }}>
              <Icon name="ChevronRight" size={16} strokeWidth={2.5} />
            </button>
          </div>
          <div className="grid grid-cols-7 mb-1">
            {WD.map((w, i) => <span key={i} className="text-center text-[10.5px] font-extrabold uppercase" style={{ color: 'var(--muted)' }}>{w}</span>)}
          </div>
          <div className="grid grid-cols-7 gap-1">
            {cells.map((d, i) => {
              if (d == null) return <span key={i} />;
              const dt = new Date(y, m, d);
              const isPast = dt < today;
              const isSel = keyOf(dt) === keyOf(sel);
              const isToday = keyOf(dt) === keyOf(today);
              return (
                <button key={i} type="button" disabled={isPast} onClick={() => pick(d)}
                  className="rounded-lg flex items-center justify-center text-[13px] font-bold active:scale-90 transition-transform"
                  style={{ height: 36,
                    background: isSel ? 'var(--accent)' : 'transparent',
                    color: isSel ? '#fff' : (isPast ? 'rgba(15,61,58,0.22)' : 'var(--ink)'),
                    border: isToday && !isSel ? '1.5px solid var(--accent)' : '1.5px solid transparent' }}>
                  {d}
                </button>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

// ── new-trip sheet ─────────────────────────────────────────────
function NewTripSheet({ open, onClose, onCreate }) {
  const [name, setName] = useStateTrip('My Weligama Trip');
  const [days, setDays] = useStateTrip(2);
  const [date, setDate] = useStateTrip(() => tripTodayISO());
  useEffectTrip(() => { if (open) { setName('My Weligama Trip'); setDays(2); setDate(tripTodayISO()); } }, [open]);
  if (!open) return null;
  const opts = [
    { n: 1, label: 'One day', sub: 'A single timeline', icon: 'Sun' },
    { n: 2, label: 'Weekend', sub: '2 days', icon: 'CalendarDays' },
    { n: 3, label: 'Long stay', sub: '3 days', icon: 'CalendarRange' },
  ];
  return (
    <div className="absolute inset-0 z-50 flex flex-col justify-end">
      <div className="absolute inset-0" style={{ background: 'rgba(16,61,58,0.45)' }} onClick={onClose} />
      <div className="relative rounded-t-[28px] px-5 pt-3 pb-9" style={{ background: 'var(--sand)', boxShadow: '0 -16px 44px -20px rgba(16,61,58,0.6)' }}>
        <div className="w-10 h-1.5 rounded-full mx-auto mb-3" style={{ background: 'rgba(15,61,58,0.15)' }} />
        <h3 className="font-extrabold text-[19px] mb-4" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Plan a new trip</h3>
        <label className="block mb-4">
          <span className="block text-[12px] font-extrabold mb-1.5" style={{ color: 'var(--ink-soft)' }}>Trip name</span>
          <div className="flex items-center gap-2.5 rounded-2xl px-3.5" style={{ height: 50, background: '#fff', border: '1.5px solid rgba(15,61,58,0.12)' }}>
            <Icon name="Route" size={17} style={{ color: 'var(--muted)' }} />
            <input value={name} onChange={(e) => setName(e.target.value)} className="flex-1 bg-transparent outline-none text-[14.5px] font-bold w-full" style={{ color: 'var(--ink)' }} />
          </div>
        </label>
        <span className="block text-[12px] font-extrabold mb-2" style={{ color: 'var(--ink-soft)' }}>Start date</span>
        <DateField value={date} onChange={setDate} />
        <span className="block text-[12px] font-extrabold mb-2" style={{ color: 'var(--ink-soft)' }}>How long?</span>
        <div className="grid grid-cols-3 gap-2.5 mb-5">
          {opts.map((o) => {
            const on = days === o.n;
            return (
              <button key={o.n} onClick={() => setDays(o.n)}
                className="rounded-2xl p-3 flex flex-col items-center gap-1.5 transition-all active:scale-[0.97]"
                style={{ background: on ? 'var(--accent)' : '#fff', color: on ? '#fff' : 'var(--ink)', border: `1.5px solid ${on ? 'var(--accent)' : 'rgba(15,61,58,0.12)'}` }}>
                <Icon name={o.icon} size={20} />
                <span className="font-extrabold text-[13px] whitespace-nowrap">{o.label}</span>
                <span className="text-[11px] font-bold" style={{ color: on ? 'rgba(255,255,255,0.8)' : 'var(--muted)' }}>{o.sub}</span>
              </button>
            );
          })}
        </div>
        <button onClick={() => onCreate(name.trim() || 'My Weligama Trip', days, date)}
          className="w-full rounded-2xl py-4 font-extrabold text-[16px] flex items-center justify-center gap-2 whitespace-nowrap text-white active:scale-[0.98] transition-transform"
          style={{ background: 'var(--accent)', boxShadow: '0 14px 28px -12px var(--accent-shadow)' }}>
          Start planning <Icon name="ArrowRight" size={18} strokeWidth={2.5} />
        </button>
      </div>
    </div>
  );
}

// ── trip share / collaborators sheet ───────────────────────────
function TripShareSheet({ open, onClose, trip, onAddCollab, onRemoveCollab }) {
  const [copied, setCopied] = useStateTrip(false);
  const [inviteTo, setInviteTo] = useStateTrip(null);     // contact | 'link' → show invitation card
  const [sentIds, setSentIds] = useStateTrip([]);         // contacts already invited
  useEffectTrip(() => { if (open) { setCopied(false); setInviteTo(null); setSentIds([]); } }, [open]);
  if (!open) return null;
  const link = `spotto.app/t/${trip.code}`;
  const collabs = (trip.collaborators || []).map(tripWho).filter(Boolean);
  const collabIds = trip.collaborators || [];
  const canAdd = (window.TRAVELERS || []).filter((x) => !collabIds.includes(x.id));
  function copy() { try { navigator.clipboard?.writeText('https://' + link); } catch {} setCopied(true); setTimeout(() => setCopied(false), 1700); }
  function markSent(c) { if (c && c !== 'link') setSentIds((s) => s.includes(c.id) ? s : [...s, c.id]); setInviteTo(null); }

  const Bubble = ({ t, size = 40 }) => (
    <span style={{ width: size, height: size, borderRadius: '50%', background: t.color, border: '2.5px solid #fff', flexShrink: 0,
                   display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontWeight: 800, fontSize: t.initial.length > 1 ? size * 0.34 : size * 0.42 }}>{t.initial}</span>
  );

  return (
    <div className="absolute inset-0 z-50 flex flex-col justify-end">
      <div className="absolute inset-0" style={{ background: 'rgba(16,61,58,0.45)' }} onClick={onClose} />
      <div className="relative rounded-t-[28px] flex flex-col" style={{ background: 'var(--sand)', maxHeight: '86%', boxShadow: '0 -16px 44px -20px rgba(16,61,58,0.6)' }}>
        <div className="w-10 h-1.5 rounded-full mx-auto mt-3 mb-2 flex-shrink-0" style={{ background: 'rgba(15,61,58,0.15)' }} />
        <div className="px-5 pb-3 flex-shrink-0">
          <div className="text-[11px] font-extrabold uppercase tracking-wider" style={{ color: 'var(--accent)' }}>Plan together</div>
          <h3 className="font-extrabold text-[21px] leading-tight truncate" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>{trip.name}</h3>
        </div>

        <div className="flex-1 overflow-y-auto px-5 pb-7">
          {/* invite link */}
          <div className="rounded-3xl p-4" style={{ background: 'linear-gradient(135deg, var(--accent), var(--accent-2))', boxShadow: '0 16px 32px -16px var(--accent-shadow)' }}>
            <div className="text-[12px] font-extrabold flex items-center gap-1.5" style={{ color: 'rgba(255,255,255,0.85)' }}>
              <Icon name="Link" size={14} /> Trip invite link
            </div>
            <div className="font-extrabold text-[15px] text-white truncate mt-0.5">{link}</div>
            <div className="flex gap-2 mt-3">
              <button onClick={copy} className="flex-1 rounded-2xl py-2.5 font-extrabold text-[14px] flex items-center justify-center gap-2 active:scale-[0.98] transition-transform" style={{ background: '#fff', color: 'var(--accent)' }}>
                <Icon name={copied ? 'Check' : 'Copy'} size={16} strokeWidth={2.6} /> {copied ? 'Copied!' : 'Copy link'}
              </button>
              <button className="flex-1 rounded-2xl py-2.5 font-extrabold text-[14px] flex items-center justify-center gap-2 active:scale-[0.98] transition-transform text-white" style={{ background: 'rgba(255,255,255,0.18)', border: '1.5px solid rgba(255,255,255,0.4)' }}>
                <Icon name="Share2" size={16} strokeWidth={2.4} /> Share
              </button>
            </div>
          </div>

          {/* send a trip invitation */}
          <button onClick={() => setInviteTo('link')}
            className="w-full rounded-2xl py-3 mt-2.5 font-extrabold text-[14px] flex items-center justify-center gap-2 active:scale-[0.98] transition-transform"
            style={{ background: '#fff', color: 'var(--ink)', border: '1.5px solid rgba(15,61,58,0.12)' }}>
            <Icon name="Send" size={16} style={{ color: 'var(--accent)' }} /> Preview &amp; send trip invitation
          </button>

          {/* who can edit */}
          <div className="text-[11.5px] font-extrabold uppercase tracking-wide mt-5 mb-2 px-1" style={{ color: 'var(--muted)' }}>Can edit this trip · {collabs.length + 1}</div>
          <div className="space-y-2">
            <div className="rounded-2xl bg-white p-2.5 flex items-center gap-3" style={{ border: '1.5px solid rgba(15,61,58,0.07)' }}>
              <Bubble t={window.ME} />
              <div className="flex-1 min-w-0">
                <div className="font-extrabold text-[14px]" style={{ color: 'var(--ink)' }}>You</div>
                <div className="text-[12px] font-semibold" style={{ color: 'var(--muted)' }}>Trip owner · approves deletions</div>
              </div>
              <span className="text-[9.5px] font-extrabold uppercase tracking-wide px-1.5 py-0.5 rounded-md" style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}>Owner</span>
            </div>
            {collabs.map((c) => (
              <div key={c.id} className="rounded-2xl bg-white p-2.5 flex items-center gap-3" style={{ border: '1.5px solid rgba(15,61,58,0.07)' }}>
                <Bubble t={c} />
                <div className="flex-1 min-w-0">
                  <div className="font-extrabold text-[14px] truncate" style={{ color: 'var(--ink)' }}>{c.name}</div>
                  <div className="text-[12px] font-semibold" style={{ color: 'var(--muted)' }}>Editor · can add &amp; update stops</div>
                </div>
                <button onClick={() => onRemoveCollab(c.id)} aria-label={`Remove ${c.name}`}
                  className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0 active:scale-90 transition-transform" style={{ background: 'var(--sand)', color: '#c2666f' }}>
                  <Icon name="UserMinus" size={17} strokeWidth={2.2} />
                </button>
              </div>
            ))}
          </div>

          {/* invite more */}
          {canAdd.length > 0 && (
            <>
              <div className="flex items-center gap-1.5 mt-5 mb-2 px-1">
                <Icon name="UserPlus" size={13} style={{ color: 'var(--accent)' }} />
                <span className="text-[11.5px] font-extrabold uppercase tracking-wide" style={{ color: 'var(--muted)' }}>Invite travellers</span>
              </div>
              <div className="space-y-2">
                {canAdd.map((t) => (
                  <div key={t.id} className="rounded-2xl p-2.5 flex items-center gap-3" style={{ background: 'rgba(255,255,255,0.6)', border: '1.5px solid rgba(15,61,58,0.07)' }}>
                    <Bubble t={t} />
                    <div className="flex-1 min-w-0">
                      <div className="font-extrabold text-[14px] truncate" style={{ color: 'var(--ink)' }}>{t.name}</div>
                      <div className="text-[12px] font-semibold truncate" style={{ color: 'var(--muted)' }}>{t.sub}</div>
                    </div>
                    <button onClick={() => onAddCollab(t.id)}
                      className="rounded-xl px-3 h-9 flex items-center gap-1.5 flex-shrink-0 font-extrabold text-[12.5px] text-white active:scale-95 transition-transform"
                      style={{ background: 'var(--accent)', boxShadow: '0 6px 16px -8px var(--accent-shadow)' }}>
                      <Icon name="Plus" size={16} strokeWidth={2.6} /> Invite
                    </button>
                  </div>
                ))}
              </div>
            </>
          )}

          {/* invite from contacts (not on Spotto yet) */}
          <div className="flex items-center gap-1.5 mt-5 mb-1 px-1">
            <Icon name="Contact" size={13} style={{ color: 'var(--accent)' }} />
            <span className="text-[11.5px] font-extrabold uppercase tracking-wide" style={{ color: 'var(--muted)' }}>Invite from contacts</span>
          </div>
          <p className="text-[12px] font-semibold mb-2 px-1" style={{ color: 'var(--muted)' }}>Not on Spotto yet — send them an invite link.</p>
          <div className="space-y-2">
            {TRIP_CONTACTS.map((c) => {
              const sent = sentIds.includes(c.id);
              return (
                <div key={c.id} className="rounded-2xl bg-white p-2.5 flex items-center gap-3" style={{ border: '1.5px solid rgba(15,61,58,0.07)' }}>
                  <Bubble t={c} />
                  <div className="flex-1 min-w-0">
                    <div className="font-extrabold text-[14px] truncate" style={{ color: 'var(--ink)' }}>{c.name}</div>
                    <div className="text-[12px] font-semibold truncate" style={{ color: 'var(--muted)' }}>{c.phone}</div>
                  </div>
                  <button onClick={() => !sent && setInviteTo(c)} disabled={sent}
                    className="rounded-xl px-3 h-9 flex items-center gap-1.5 flex-shrink-0 font-extrabold text-[12.5px] active:scale-95 transition-transform"
                    style={{ background: sent ? 'var(--sand)' : 'var(--accent)', color: sent ? 'var(--muted)' : '#fff', boxShadow: sent ? 'none' : '0 6px 16px -8px var(--accent-shadow)' }}>
                    <Icon name={sent ? 'Check' : 'Send'} size={15} strokeWidth={2.5} /> {sent ? 'Invited' : 'Invite'}
                  </button>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      <TripInvitation trip={trip} to={inviteTo} link={link} onClose={() => setInviteTo(null)} onSent={markSent} />
    </div>
  );
}

// ── shareable trip invitation card ─────────────────────────────
function TripInvitation({ trip, to, link, onClose, onSent }) {
  const [copied, setCopied] = useStateTrip(false);
  useEffectTrip(() => { setCopied(false); }, [to]);
  if (!to) return null;
  const named = to !== 'link';
  const first = window.tripDayDate(trip.startDate, 0);
  const last = window.tripDayDate(trip.startDate, trip.days.length - 1);
  const range = trip.days.length === 1
    ? first.toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' })
    : `${first.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })} – ${last.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}`;
  function copy() { try { navigator.clipboard?.writeText(`You're invited to "${trip.name}" on Spotto — ${range}. Join: https://${link}`); } catch {} setCopied(true); setTimeout(() => setCopied(false), 1700); }

  return (
    <div className="absolute inset-0 z-[60] flex items-center justify-center px-6">
      <div className="absolute inset-0" style={{ background: 'rgba(16,61,58,0.55)' }} onClick={onClose} />
      <div className="relative w-full rounded-[28px] overflow-hidden" style={{ boxShadow: '0 28px 64px -22px rgba(16,61,58,0.7)' }}>
        {/* invitation header */}
        <div className="relative px-6 pt-7 pb-6 text-center" style={{ background: 'linear-gradient(150deg, var(--accent), var(--accent-2))' }}>
          <button onClick={onClose} aria-label="Close" className="absolute top-3 right-3 w-8 h-8 rounded-full flex items-center justify-center" style={{ background: 'rgba(255,255,255,0.22)', color: '#fff' }}>
            <Icon name="X" size={16} strokeWidth={2.6} />
          </button>
          <div className="inline-flex items-center justify-center w-12 h-12 rounded-2xl mb-3" style={{ background: 'rgba(255,255,255,0.22)', color: '#fff' }}>
            <Icon name="Plane" size={24} />
          </div>
          <div className="text-[11px] font-extrabold uppercase tracking-[0.18em]" style={{ color: 'rgba(255,255,255,0.85)' }}>You're invited</div>
          <h3 className="font-extrabold text-[24px] leading-tight mt-1 text-white" style={{ fontFamily: 'var(--font-display)' }}>{trip.name}</h3>
          <div className="inline-flex items-center gap-1.5 rounded-full px-3 py-1 mt-2.5" style={{ background: 'rgba(255,255,255,0.2)' }}>
            <Icon name="CalendarDays" size={13} style={{ color: '#fff' }} />
            <span className="text-[12.5px] font-extrabold text-white">{range} · {trip.days.length} day{trip.days.length > 1 ? 's' : ''}</span>
          </div>
          <div className="flex items-center justify-center gap-2 mt-3">
            <span style={{ width: 26, height: 26, borderRadius: '50%', background: '#fff', color: 'var(--accent)', fontSize: 11, fontWeight: 800, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>{window.ME.initial}</span>
            <span className="text-[12.5px] font-bold text-white">Hosted by you · Weligama, Sri Lanka</span>
          </div>
        </div>

        {/* body */}
        <div className="bg-white p-5">
          {named && (
            <div className="flex items-center gap-2.5 rounded-2xl p-2.5 mb-3" style={{ background: 'var(--sand)' }}>
              <span style={{ width: 36, height: 36, borderRadius: '50%', background: to.color, color: '#fff', fontWeight: 800, fontSize: 15, display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>{to.initial}</span>
              <div className="min-w-0">
                <div className="text-[10px] font-extrabold uppercase tracking-wide" style={{ color: 'var(--muted)' }}>Sending to</div>
                <div className="font-extrabold text-[14px] truncate" style={{ color: 'var(--ink)' }}>{to.name} · {to.phone}</div>
              </div>
            </div>
          )}
          <div className="flex items-center gap-2 rounded-2xl px-3 py-2.5 mb-3" style={{ border: '1.5px dashed rgba(15,61,58,0.2)' }}>
            <Icon name="Link" size={15} style={{ color: 'var(--accent)' }} />
            <span className="flex-1 text-[13px] font-bold truncate" style={{ color: 'var(--ink)' }}>{link}</span>
          </div>
          <button onClick={() => onSent(to)}
            className="w-full rounded-2xl py-3.5 font-extrabold text-[15px] flex items-center justify-center gap-2 text-white active:scale-[0.98] transition-transform"
            style={{ background: 'var(--accent)', boxShadow: '0 14px 28px -12px var(--accent-shadow)' }}>
            <Icon name={named ? 'MessageCircle' : 'Send'} size={17} strokeWidth={2.4} /> {named ? `Send to ${to.name.split(' ')[0]}` : 'Send invitation'}
          </button>
          <div className="flex gap-2 mt-2">
            <button onClick={copy} className="flex-1 rounded-2xl py-2.5 font-extrabold text-[13.5px] flex items-center justify-center gap-1.5 active:scale-[0.98] transition-transform" style={{ background: 'var(--sand)', color: 'var(--ink)' }}>
              <Icon name={copied ? 'Check' : 'Copy'} size={15} strokeWidth={2.5} /> {copied ? 'Copied!' : 'Copy'}
            </button>
            <button className="flex-1 rounded-2xl py-2.5 font-extrabold text-[13.5px] flex items-center justify-center gap-1.5 active:scale-[0.98] transition-transform" style={{ background: 'var(--sand)', color: 'var(--ink)' }}>
              <Icon name="Share2" size={15} strokeWidth={2.5} /> Share
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

// ── owner-confirmation before deleting a stop ──────────────────
function ConfirmDelete({ pending, onCancel, onConfirm }) {
  if (!pending) return null;
  const credit = pending.by && pending.by.id !== 'me' ? pending.by : null;
  return (
    <div className="absolute inset-0 z-[60] flex items-center justify-center px-7">
      <div className="absolute inset-0" style={{ background: 'rgba(16,61,58,0.5)' }} onClick={onCancel} />
      <div className="relative rounded-3xl bg-white w-full p-5" style={{ boxShadow: '0 24px 60px -20px rgba(16,61,58,0.6)' }}>
        <div className="w-12 h-12 rounded-2xl flex items-center justify-center mb-3" style={{ background: 'rgba(194,102,111,0.14)', color: '#c2666f' }}>
          <Icon name="ShieldAlert" size={24} />
        </div>
        <h3 className="font-extrabold text-[18px] leading-tight" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Remove “{pending.name}”?</h3>
        <p className="text-[13.5px] font-semibold mt-1.5 leading-snug" style={{ color: 'var(--muted)' }}>
          {credit ? <>This stop was added by {credit.name}. </> : null}
          As the trip owner, you need to confirm before it's removed for everyone.
        </p>
        <div className="flex gap-2.5 mt-4">
          <button onClick={onCancel} className="flex-1 rounded-2xl py-3 font-extrabold text-[14.5px] active:scale-[0.98] transition-transform" style={{ background: 'var(--sand)', color: 'var(--ink)' }}>Keep it</button>
          <button onClick={onConfirm} className="flex-1 rounded-2xl py-3 font-extrabold text-[14.5px] text-white active:scale-[0.98] transition-transform" style={{ background: '#c2666f', boxShadow: '0 12px 24px -10px rgba(194,102,111,0.6)' }}>Confirm remove</button>
        </div>
      </div>
    </div>
  );
}

// ── month calendar — pick a date to open its single-day plan ────
function CalendarMonth({ trip, monthDate, onPrevMonth, onNextMonth, onPick, onAddDay }) {
  const keyOf = (d) => `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`;
  const y = monthDate.getFullYear(), m = monthDate.getMonth();
  const startDow = new Date(y, m, 1).getDay();
  const daysInMonth = new Date(y, m + 1, 0).getDate();
  const todayKey = keyOf(new Date());

  const planned = {};
  trip.days.forEach((d, i) => { planned[keyOf(window.tripDayDate(trip.startDate, i))] = { idx: i, stops: d.stops.length }; });

  const firstDt = window.tripDayDate(trip.startDate, 0);
  const lastDt = window.tripDayDate(trip.startDate, trip.days.length - 1);
  const rangeLabel = `${firstDt.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })} – ${lastDt.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}`;

  const cells = [];
  for (let i = 0; i < startDow; i++) cells.push(null);
  for (let d = 1; d <= daysInMonth; d++) cells.push(d);
  const WD = ['S', 'M', 'T', 'W', 'T', 'F', 'S'];

  return (
    <div className="px-5 pt-4">
      {/* trip range */}
      <div className="flex items-center gap-1.5 mb-3">
        <span className="rounded-full px-2.5 py-1 text-[11.5px] font-extrabold inline-flex items-center gap-1.5" style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}>
          <Icon name="CalendarDays" size={13} /> {trip.days.length}-day trip · {rangeLabel}
        </span>
      </div>

      {/* month nav */}
      <div className="flex items-center justify-between mb-2">
        <span className="font-extrabold text-[17px] whitespace-nowrap" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>
          {monthDate.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })}
        </span>
        <div className="flex gap-1.5">
          <button onClick={onPrevMonth} aria-label="Previous month" className="w-8 h-8 rounded-xl flex items-center justify-center active:scale-90 transition-transform" style={{ background: '#fff', color: 'var(--ink)', boxShadow: '0 2px 8px rgba(15,61,58,0.08)' }}>
            <Icon name="ChevronLeft" size={18} strokeWidth={2.5} />
          </button>
          <button onClick={onNextMonth} aria-label="Next month" className="w-8 h-8 rounded-xl flex items-center justify-center active:scale-90 transition-transform" style={{ background: '#fff', color: 'var(--ink)', boxShadow: '0 2px 8px rgba(15,61,58,0.08)' }}>
            <Icon name="ChevronRight" size={18} strokeWidth={2.5} />
          </button>
        </div>
      </div>

      {/* weekday header */}
      <div className="grid grid-cols-7 mb-1.5">
        {WD.map((w, i) => <span key={i} className="text-center text-[11px] font-extrabold uppercase" style={{ color: 'var(--muted)' }}>{w}</span>)}
      </div>

      {/* day grid */}
      <div className="grid grid-cols-7 gap-1.5">
        {cells.map((d, i) => {
          if (d == null) return <span key={i} />;
          const k = `${y}-${m}-${d}`;
          const pl = planned[k];
          const today = k === todayKey;
          if (pl) {
            return (
              <button key={i} onClick={() => onPick(pl.idx)}
                className="rounded-xl flex flex-col items-center justify-center active:scale-95 transition-transform"
                style={{ height: 46, background: 'var(--accent)', color: '#fff', boxShadow: '0 8px 16px -8px var(--accent-shadow)' }}>
                <span className="text-[15px] font-extrabold leading-none">{d}</span>
                <span className="text-[8.5px] font-bold mt-0.5" style={{ color: 'rgba(255,255,255,0.85)' }}>{pl.stops ? `${pl.stops} stop${pl.stops > 1 ? 's' : ''}` : 'Day ' + (pl.idx + 1)}</span>
              </button>
            );
          }
          return (
            <span key={i} className="rounded-xl flex items-center justify-center text-[14px] font-bold"
              style={{ height: 46, color: today ? 'var(--accent)' : 'rgba(15,61,58,0.45)', border: today ? '1.5px solid var(--accent)' : '1.5px solid transparent' }}>
              {d}
            </span>
          );
        })}
      </div>

      {/* hint + add day */}
      <p className="text-center text-[12px] font-semibold mt-4" style={{ color: 'var(--muted)' }}>Tap a highlighted day to open its plan.</p>
      {trip.days.length < 4 && (
        <button onClick={onAddDay}
          className="w-full rounded-2xl py-3 mt-3 font-extrabold text-[13.5px] flex items-center justify-center gap-2 active:scale-[0.98] transition-transform"
          style={{ background: 'transparent', color: 'var(--accent)', border: '1.5px dashed var(--accent)' }}>
          <Icon name="Plus" size={15} strokeWidth={2.6} /> Add another day
        </button>
      )}
    </div>
  );
}

// ── main view ──────────────────────────────────────────────────
function TripView({ styleId, onOpenPlace }) {
  const [trip, setTrip] = useStateTrip(tripLoad);
  const [dayIx, setDayIx] = useStateTrip(0);
  const [addOpen, setAddOpen] = useStateTrip(false);
  const [newOpen, setNewOpen] = useStateTrip(false);
  const [shareOpen, setShareOpen] = useStateTrip(false);
  const [pendingDel, setPendingDel] = useStateTrip(null);
  const [view, setView] = useStateTrip('cal');   // 'cal' (month) → 'day' (single day)
  const [calMonth, setCalMonth] = useStateTrip(() => {
    const t = tripLoad();
    const d = new Date((t.startDate || new Date().toISOString().slice(0, 10)) + 'T00:00:00');
    return new Date(d.getFullYear(), d.getMonth(), 1);
  });

  useEffectTrip(() => { try { localStorage.setItem(TRIP_LS_KEY, JSON.stringify(trip)); } catch {} }, [trip]);

  const day = trip?.days?.[dayIx] || trip?.days?.[0];
  const stops = day?.stops || [];
  const places = stops.map((s) => window.tripPlace(s)).filter(Boolean);
  const legs = useMemoTrip(() => stops.slice(1).map((s, i) => window.tripLeg(window.tripPlace(stops[i]), window.tripPlace(s))), [stops]);
  const totalCost = places.reduce((a, p) => a + (p.cost || 0), 0);
  const totalTravel = legs.reduce((a, l) => a + l.mins, 0);
  const dayRoute = window.tripDayGmaps(stops);
  const usedIds = stops.map((s) => s.placeId).filter(Boolean);

  function patchDay(fn) {
    setTrip((t) => ({ ...t, days: t.days.map((d, i) => (i === dayIx ? { ...d, stops: fn(d.stops) } : d)) }));
  }
  function move(ix, dir) {
    patchDay((st) => {
      const next = st.slice(); const j = ix + dir;
      if (j < 0 || j >= next.length) return st;
      // swap stops but keep the time slots in place → timeline stays chronological
      const a = { ...next[ix] }, b = { ...next[j] };
      const tA = a.time; a.time = b.time; b.time = tA;
      next[ix] = b; next[j] = a;
      return next;
    });
  }
  function addStop(p) {
    patchDay((st) => {
      const [h, m] = (p.hint || '9:00').split(':').map(Number);
      const t = st.length ? Math.min(23 * 60 + 30, st[st.length - 1].time + 90) : (h * 60 + (m || 0));
      const stop = p.customFlag
        ? { id: newId('s'), time: t, by: 'me', custom: { name: p.name, type: p.type, icon: p.icon, lat: p.lat, lng: p.lng, cost: 0 } }
        : { id: newId('s'), placeId: p.id, time: t, by: 'me' };
      return [...st, stop];
    });
    setAddOpen(false);
  }
  function createTrip(name, n, startDate) {
    setTrip((prev) => ({ name, startDate, collaborators: [], code: prev.code || ('TRP-' + Math.random().toString(36).slice(2, 5).toUpperCase()),
                         days: Array.from({ length: n }, () => ({ id: newId('d'), stops: [] })) }));
    const d = new Date(startDate + 'T00:00:00');
    setCalMonth(new Date(d.getFullYear(), d.getMonth(), 1));
    setDayIx(0); setView('cal'); setNewOpen(false);
  }
  function addDay() { setTrip((t) => ({ ...t, days: [...t.days, { id: newId('d'), stops: [] }] })); }
  function pickDay(i) { setDayIx(i); setView('day'); }
  function goDay(dir) { setDayIx((i) => Math.min(trip.days.length - 1, Math.max(0, i + dir))); }
  const swipe = React.useRef({ x: 0, y: 0 });
  function onSwipeDown(e) { const t = e.touches ? e.touches[0] : e; swipe.current = { x: t.clientX, y: t.clientY }; }
  function onSwipeUp(e) {
    const t = e.changedTouches ? e.changedTouches[0] : e;
    const dx = t.clientX - swipe.current.x, dy = t.clientY - swipe.current.y;
    if (Math.abs(dx) > 55 && Math.abs(dx) > Math.abs(dy) * 1.4) goDay(dx < 0 ? 1 : -1);
  }
  function addCollab(id) { setTrip((t) => (t.collaborators || []).includes(id) ? t : { ...t, collaborators: [...(t.collaborators || []), id] }); }
  function removeCollab(id) { setTrip((t) => ({ ...t, collaborators: (t.collaborators || []).filter((x) => x !== id) })); }
  function confirmDelete() {
    if (!pendingDel) return;
    const id = pendingDel.stopId;
    patchDay((st) => st.filter((x) => x.id !== id));
    setPendingDel(null);
  }

  if (!trip || !trip.days) return null;

  return (
    <div className="h-full flex flex-col" style={{ background: 'var(--sand)' }}>
      {/* header */}
      <div className="px-5 flex-shrink-0" style={{ paddingTop: 64 }}>
        <div className="flex items-center justify-between gap-2">
          <div className="min-w-0">
            <div className="text-[11.5px] font-extrabold uppercase tracking-wider" style={{ color: 'var(--accent)' }}>Trip planner</div>
            <h1 className="font-extrabold leading-tight truncate" style={{ fontSize: 21, color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>{trip.name}</h1>
          </div>
          <div className="flex items-center gap-2 flex-shrink-0">
            {/* collaborators / invite */}
            <button onClick={() => setShareOpen(true)} aria-label="Plan together"
              className="h-10 rounded-2xl flex items-center gap-1.5 pl-1.5 pr-2.5 active:scale-95 transition-transform"
              style={{ background: '#fff', boxShadow: '0 2px 10px rgba(15,61,58,0.1)' }}>
              <span className="flex -space-x-2 items-center">
                {[window.ME, ...(trip.collaborators || []).map(tripWho).filter(Boolean)].slice(0, 3).map((m, i) => (
                  <span key={i} style={{ width: 24, height: 24, borderRadius: '50%', background: m.color, border: '2px solid #fff',
                                         display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontWeight: 800, fontSize: m.initial.length > 1 ? 8.5 : 10.5 }}>{m.initial}</span>
                ))}
              </span>
              <Icon name="UserPlus" size={15} strokeWidth={2.4} style={{ color: 'var(--accent)' }} />
            </button>
            <button onClick={() => setNewOpen(true)} aria-label="New trip"
              className="w-10 h-10 rounded-2xl flex items-center justify-center flex-shrink-0 active:scale-95 transition-transform"
              style={{ background: '#fff', color: 'var(--ink)', boxShadow: '0 2px 10px rgba(15,61,58,0.1)' }}>
              <Icon name="Plus" size={19} strokeWidth={2.5} />
            </button>
          </div>
        </div>
        {/* day sub-header (only in single-day view) */}
        {view === 'day' && (() => {
          const dt = window.tripDayDate(trip.startDate, dayIx);
          return (
            <div className="flex items-center gap-2.5 mt-3">
              <button onClick={() => setView('cal')} aria-label="Back to calendar"
                className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0 active:scale-90 transition-transform"
                style={{ background: '#fff', color: 'var(--ink)', boxShadow: '0 2px 8px rgba(15,61,58,0.1)' }}>
                <Icon name="ChevronLeft" size={20} strokeWidth={2.5} />
              </button>
              <div className="flex items-baseline gap-2 min-w-0">
                <span className="font-extrabold text-[16px] truncate" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>
                  {dt.toLocaleDateString('en-US', { weekday: 'long', month: 'short', day: 'numeric' })}
                </span>
                <span className="text-[12px] font-extrabold flex-shrink-0" style={{ color: 'var(--accent)' }}>Day {dayIx + 1}</span>
              </div>
              <div className="flex items-center gap-1.5 ml-auto flex-shrink-0">
                <button onClick={() => goDay(-1)} disabled={dayIx === 0} aria-label="Previous day"
                  className="w-9 h-9 rounded-xl flex items-center justify-center active:scale-90 transition-transform"
                  style={{ background: '#fff', color: dayIx === 0 ? 'rgba(15,61,58,0.25)' : 'var(--ink)', boxShadow: '0 2px 8px rgba(15,61,58,0.1)' }}>
                  <Icon name="ChevronLeft" size={18} strokeWidth={2.5} />
                </button>
                <button onClick={() => goDay(1)} disabled={dayIx === trip.days.length - 1} aria-label="Next day"
                  className="w-9 h-9 rounded-xl flex items-center justify-center active:scale-90 transition-transform"
                  style={{ background: '#fff', color: dayIx === trip.days.length - 1 ? 'rgba(15,61,58,0.25)' : 'var(--ink)', boxShadow: '0 2px 8px rgba(15,61,58,0.1)' }}>
                  <Icon name="ChevronRight" size={18} strokeWidth={2.5} />
                </button>
              </div>
            </div>
          );
        })()}
      </div>

      {/* body: month calendar OR the selected day's timeline */}
      {view === 'cal' ? (
        <div className="flex-1 overflow-y-auto" style={{ paddingBottom: 16 }}>
          <CalendarMonth trip={trip} monthDate={calMonth} onPick={pickDay} onAddDay={addDay}
            onPrevMonth={() => setCalMonth(new Date(calMonth.getFullYear(), calMonth.getMonth() - 1, 1))}
            onNextMonth={() => setCalMonth(new Date(calMonth.getFullYear(), calMonth.getMonth() + 1, 1))} />
        </div>
      ) : (
      <div className="flex-1 overflow-y-auto px-5 pt-3" style={{ paddingBottom: 16, touchAction: 'pan-y' }}
           onPointerDown={onSwipeDown} onPointerUp={onSwipeUp}>
        {stops.length === 0 ? (
          <div className="flex flex-col items-center text-center pt-14 px-6">
            <div className="w-16 h-16 rounded-full flex items-center justify-center mb-4" style={{ background: 'var(--accent-tint)' }}>
              <Icon name="Route" size={28} style={{ color: 'var(--accent)' }} />
            </div>
            <h3 className="font-extrabold text-[18px]" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Nothing planned yet</h3>
            <p className="text-[13.5px] font-semibold mt-1.5 leading-snug" style={{ color: 'var(--muted)' }}>Add your first stop — search any place or pick a suggestion for your style.</p>
          </div>
        ) : (
          <div>
            {stops.map((s, i) => {
              const p = window.tripPlace(s);
              return (
                <div key={s.id}>
                  {i > 0 && legs[i - 1] && <LegRow leg={legs[i - 1]} />}
                  <StopCard stop={s} place={p} first={i === 0} last={i === stops.length - 1} by={tripWho(s.by)}
                    onTime={(t) => patchDay((st) => st.map((x) => (x.id === s.id ? { ...x, time: t } : x)))}
                    onUp={() => move(i, -1)} onDown={() => move(i, 1)}
                    onRemove={() => setPendingDel({ stopId: s.id, name: p.name, by: tripWho(s.by) })}
                    onOpen={() => p.spottoId && onOpenPlace(p.spottoId)} />
                </div>
              );
            })}
          </div>
        )}

        {/* add stop */}
        <button onClick={() => setAddOpen(true)}
          className="w-full rounded-3xl py-3.5 mt-3 font-extrabold text-[14.5px] flex items-center justify-center gap-2 whitespace-nowrap active:scale-[0.98] transition-transform"
          style={{ background: 'transparent', color: 'var(--accent)', border: '2px dashed var(--accent)' }}>
          <Icon name="Plus" size={17} strokeWidth={2.5} /> Add a stop
        </button>
      </div>
      )}

      {/* day summary */}
      {view === 'day' && stops.length > 0 && (
        <div className="flex-shrink-0 px-5 pt-3 pb-3" style={{ background: '#fff', boxShadow: '0 -4px 20px -12px rgba(16,61,58,0.3)' }}>
          <div className="flex items-center justify-between mb-2.5">
            <div className="flex items-center gap-4">
              <span className="flex items-center gap-1.5 text-[12.5px] font-extrabold whitespace-nowrap" style={{ color: 'var(--ink)' }}>
                <Icon name="MapPin" size={14} style={{ color: 'var(--accent)' }} /> {stops.length} stops
              </span>
              <span className="flex items-center gap-1.5 text-[12.5px] font-extrabold whitespace-nowrap" style={{ color: 'var(--ink)' }}>
                <Icon name="Timer" size={14} style={{ color: 'var(--accent)' }} /> {totalTravel} min travel
              </span>
              <span className="flex items-center gap-1.5 text-[12.5px] font-extrabold whitespace-nowrap" style={{ color: 'var(--ink)' }}>
                <Icon name="Wallet" size={14} style={{ color: 'var(--accent)' }} /> {window.tripFmtRs(totalCost)}
              </span>
            </div>
          </div>
          {dayRoute && (
            <a href={dayRoute} target="_blank" rel="noopener"
              className="w-full rounded-2xl py-3 font-extrabold text-[14px] flex items-center justify-center gap-2 whitespace-nowrap text-white active:scale-[0.98] transition-transform"
              style={{ background: 'var(--accent)', boxShadow: '0 10px 22px -10px var(--accent-shadow)' }}>
              <Icon name="Navigation" size={16} /> Open day route in Google Maps
            </a>
          )}
        </div>
      )}

      <AddStopSheet open={addOpen} onClose={() => setAddOpen(false)} onAdd={addStop} styleId={styleId} usedIds={usedIds} />
      <NewTripSheet open={newOpen} onClose={() => setNewOpen(false)} onCreate={createTrip} />
      <TripShareSheet open={shareOpen} onClose={() => setShareOpen(false)} trip={trip} onAddCollab={addCollab} onRemoveCollab={removeCollab} />
      <ConfirmDelete pending={pendingDel} onCancel={() => setPendingDel(null)} onConfirm={confirmDelete} />
    </div>
  );
}

Object.assign(window, { TripView });
