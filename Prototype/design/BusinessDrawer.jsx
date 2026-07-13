// BusinessDrawer.jsx — bottom sheet for a selected place. Animates in/out.

const { useState: useStateBD, useEffect: useEffectBD, useRef: useRefBD } = React;

function WifiMeter({ score }) {
  const tone = score >= 8 ? 'var(--accent)' : score >= 6.5 ? '#d9a213' : '#e06a4a';
  const bars = [0.45, 0.65, 0.82, 1].map((h, i) => ({
    h, on: score / 10 >= (i + 1) / 4 - 0.12,
  }));
  return (
    <div className="flex items-end gap-[3px] h-5">
      {bars.map((b, i) => (
        <div key={i} style={{
          width: 5, height: `${b.h * 100}%`, borderRadius: 2,
          background: b.on ? tone : 'rgba(15,61,58,0.14)',
        }} />
      ))}
    </div>
  );
}

function Stat({ icon, label, value, tone }) {
  return (
    <div className="flex-1 rounded-2xl p-3" style={{ background: 'var(--sand)' }}>
      <div className="flex items-center gap-1.5" style={{ color: tone || 'var(--muted)' }}>
        <Icon name={icon} size={15} />
        <span className="text-[11px] font-bold uppercase tracking-wide">{label}</span>
      </div>
      <div className="mt-1 font-extrabold text-[15px] leading-tight" style={{ color: 'var(--ink)' }}>{value}</div>
    </div>
  );
}

function VibeCheck({ placeId, vibes, onVote }) {
  const opts = window.VIBES;
  const total = opts.reduce((a, v) => a + (vibes[v.id] || 0), 0);
  const lead = opts.slice().sort((a, b) => (vibes[b.id] || 0) - (vibes[a.id] || 0))[0];
  const [justVoted, setJustVoted] = useStateBD(null);
  return (
    <div className="rounded-2xl p-4" style={{ background: 'var(--sand)' }}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5 text-[11px] font-extrabold uppercase tracking-wide" style={{ color: 'var(--muted)' }}>
          <Icon name="Activity" size={14} /> Current vibe
        </div>
        {total > 0 && (
          <span className="text-[11.5px] font-bold flex items-center gap-1" style={{ color: lead.color }}>
            <Icon name={lead.icon} size={13} /> Mostly {lead.label.toLowerCase()} · {total} votes
          </span>
        )}
      </div>
      <div className="flex gap-2 mt-3">
        {opts.map((v) => {
          const count = vibes[v.id] || 0;
          const pct = total ? Math.round((count / total) * 100) : 0;
          const mine = justVoted === v.id;
          return (
            <button key={v.id} onClick={() => { onVote(placeId, v.id); setJustVoted(v.id); }}
              className="flex-1 rounded-xl p-2.5 transition-all active:scale-95"
              style={{ background: mine ? v.color : '#fff', color: mine ? '#fff' : 'var(--ink)',
                       border: mine ? `1.5px solid ${v.color}` : '1.5px solid rgba(15,61,58,0.1)' }}>
              <Icon name={v.icon} size={18} style={{ color: mine ? '#fff' : v.color }} />
              <div className="text-[12px] font-extrabold mt-1">{v.label}</div>
              <div className="h-1.5 rounded-full mt-1.5 overflow-hidden" style={{ background: mine ? 'rgba(255,255,255,0.3)' : 'rgba(15,61,58,0.08)' }}>
                <div className="h-full rounded-full transition-all duration-500" style={{ width: `${pct}%`, background: mine ? '#fff' : v.color }} />
              </div>
            </button>
          );
        })}
      </div>
      <p className="text-[11px] font-semibold mt-2" style={{ color: 'var(--muted)' }}>Tap how it feels right now — helps the next traveler.</p>
    </div>
  );
}

// ── menu (food places only) ────────────────────────────────────
function MenuSection({ menu }) {
  const [expanded, setExpanded] = useStateBD(false);
  const total = menu.reduce((a, s) => a + s.items.length, 0);
  const COLLAPSED = 4;

  // flatten while keeping section headers; trim to COLLAPSED items when closed
  let shown = 0;
  const blocks = menu.map((s) => {
    if (!expanded && shown >= COLLAPSED) return null;
    const items = expanded ? s.items : s.items.slice(0, Math.max(0, COLLAPSED - shown));
    shown += items.length;
    if (!items.length) return null;
    return (
      <div key={s.id} className="mt-3 first:mt-0">
        <div className="flex items-center gap-2">
          <span className="text-[12px] font-extrabold uppercase tracking-wide" style={{ color: 'var(--accent)' }}>{s.name}</span>
          <span className="flex-1 h-px" style={{ background: 'rgba(15,61,58,0.08)' }} />
        </div>
        {items.map((it) => (
          <div key={it.id} className="flex items-center gap-3 mt-2.5">
            {it.img ? (
              <img src={it.img} alt={it.name} style={{ width: 44, height: 44, borderRadius: 12, objectFit: 'cover', flexShrink: 0 }} />
            ) : (
              <span className="flex items-center justify-center flex-shrink-0"
                    style={{ width: 44, height: 44, borderRadius: 12, background: 'var(--accent-tint)', color: 'var(--accent)' }}>
                <Icon name={it.icon || 'Utensils'} size={19} />
              </span>
            )}
            <div className="flex-1 min-w-0">
              <div className="font-extrabold text-[14px] leading-tight truncate" style={{ color: 'var(--ink)' }}>{it.name}</div>
              {it.desc && <div className="text-[12px] font-semibold truncate mt-0.5" style={{ color: 'var(--muted)' }}>{it.desc}</div>}
            </div>
            <span className="font-extrabold text-[13.5px] whitespace-nowrap" style={{ color: 'var(--ink)' }}>
              Rs {(it.price || 0).toLocaleString('en-US')}
            </span>
          </div>
        ))}
      </div>
    );
  });

  return (
    <div className="mt-4 rounded-2xl p-4" style={{ background: '#fff', border: '1.5px solid rgba(15,61,58,0.08)' }}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-1.5 text-[11px] font-extrabold uppercase tracking-wide" style={{ color: 'var(--muted)' }}>
          <Icon name="UtensilsCrossed" size={14} /> Menu
        </div>
        <span className="text-[11.5px] font-bold" style={{ color: 'var(--muted)' }}>{total} items</span>
      </div>
      <div className="mt-3">{blocks}</div>
      {total > COLLAPSED && (
        <button onClick={() => setExpanded((e) => !e)}
          className="mt-3.5 w-full rounded-xl py-2.5 font-extrabold text-[13px] flex items-center justify-center gap-1.5 active:scale-[0.99] transition-transform"
          style={{ background: 'var(--sand)', color: 'var(--ink)' }}>
          <Icon name={expanded ? 'ChevronUp' : 'ChevronDown'} size={15} strokeWidth={2.5} />
          {expanded ? 'Show less' : `View full menu (${total} items)`}
        </button>
      )}
    </div>
  );
}

function BusinessDrawer({ place, onClose, vibes = {}, onVote = () => {}, onCheckIn = () => {}, checkedIn = [] }) {
  const open = !!place;
  const [shown, setShown] = useStateBD(place);
  const [claimed, setClaimed] = useStateBD(false);

  useEffectBD(() => {
    if (place) { setShown(place); setClaimed(false); }
    else { const t = setTimeout(() => setShown(null), 320); return () => clearTimeout(t); }
  }, [place]);

  const p = shown;
  const didCheckIn = p && checkedIn.includes(p.id);

  return (
    <>
      {/* backdrop */}
      <div onClick={onClose}
        style={{
          position: 'absolute', inset: 0, zIndex: 40,
          background: 'rgba(16,61,58,0.34)', backdropFilter: 'blur(1.5px)',
          opacity: open ? 1 : 0, pointerEvents: open ? 'auto' : 'none',
          transition: 'opacity 300ms ease',
        }} />
      {/* sheet */}
      <div style={{
        position: 'absolute', left: 0, right: 0, bottom: 0, zIndex: 41,
        transform: open ? 'translateY(0)' : 'translateY(110%)',
        transition: 'transform 360ms cubic-bezier(0.22,1,0.36,1)',
        background: '#fff', borderTopLeftRadius: 30, borderTopRightRadius: 30,
        boxShadow: '0 -16px 40px rgba(16,61,58,0.28)', overflow: 'hidden',
        maxHeight: '86%', display: 'flex', flexDirection: 'column',
      }}>
        {p && (
          <>
            {/* photo header */}
            <div style={{ position: 'relative', height: 158, flexShrink: 0, background: 'var(--sand)' }}>
              <Illustration kind={p.photo} />
              <div className="absolute left-1/2 -translate-x-1/2 top-2.5 w-10 h-1.5 rounded-full" style={{ background: 'rgba(255,255,255,0.7)' }} />
              <button onClick={onClose}
                className="absolute top-3.5 right-3.5 w-9 h-9 rounded-full flex items-center justify-center"
                style={{ background: 'rgba(255,255,255,0.92)', color: 'var(--ink)' }}>
                <Icon name="X" size={18} strokeWidth={2.5} />
              </button>
              {p.deal?.active && (
                <div className="absolute left-4 bottom-3.5 inline-flex items-center gap-1.5 rounded-full px-3 py-1.5"
                     style={{ background: 'var(--sun)', color: '#5a3b00' }}>
                  <Icon name="Tag" size={14} strokeWidth={2.5} />
                  <span className="font-extrabold text-[12.5px]">{p.deal.tag}</span>
                </div>
              )}
            </div>

            {/* scroll content */}
            <div className="overflow-y-auto px-5 pb-6 pt-4" style={{ WebkitOverflowScrolling: 'touch' }}>
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h2 className="font-extrabold leading-tight" style={{ fontSize: 22, color: 'var(--ink)' }}>{p.name}</h2>
                  <div className="mt-1 text-[13.5px] font-semibold" style={{ color: 'var(--muted)' }}>
                    {p.type} · {p.priceLevel}
                  </div>
                </div>
                <div className="flex items-center gap-1 rounded-full px-2.5 py-1.5 flex-shrink-0"
                     style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}>
                  <Icon name="Star" size={14} style={{ fill: 'currentColor' }} />
                  <span className="font-extrabold text-[13.5px]">{p.rating}</span>
                </div>
              </div>

              <div className="flex items-center gap-3 mt-2 text-[13px] font-semibold" style={{ color: 'var(--muted)' }}>
                <span className="flex items-center gap-1"><Icon name="MessageSquare" size={13} />{p.reviews} reviews</span>
                <span className="flex items-center gap-1"><Icon name="Navigation" size={13} />{p.distance} away</span>
              </div>

              {/* quick stats */}
              <div className="flex gap-2.5 mt-4">
                <div className="flex-1 rounded-2xl p-3" style={{ background: 'var(--sand)' }}>
                  <div className="flex items-center gap-1.5" style={{ color: 'var(--muted)' }}>
                    <Icon name="Wifi" size={15} />
                    <span className="text-[11px] font-bold uppercase tracking-wide">Wi-Fi</span>
                  </div>
                  <div className="flex items-center gap-2 mt-1">
                    <span className="font-extrabold text-[15px]" style={{ color: 'var(--ink)' }}>{p.wifi.score}</span>
                    <WifiMeter score={p.wifi.score} />
                  </div>
                  <div className="text-[11px] font-semibold mt-0.5" style={{ color: 'var(--muted)' }}>{p.wifi.label} · {p.wifi.speed}</div>
                </div>
                <Stat icon="Clock" label="Hours"
                  tone={p.openNow ? '#0b6f86' : '#e06a4a'}
                  value={p.openNow ? 'Open now' : 'Opens later'} />
              </div>
              <div className="mt-1.5 text-[12.5px] font-semibold px-1" style={{ color: 'var(--muted)' }}>{p.hours}</div>

              {/* deal */}
              {p.deal?.active && (
                <div className="mt-4 rounded-2xl p-4" style={{
                  background: claimed ? 'var(--accent-tint)' : 'linear-gradient(120deg, #fff6df, #ffe9c2)',
                  border: claimed ? '1.5px solid var(--accent)' : '1.5px solid #f3d690',
                }}>
                  <div className="flex items-center gap-1.5 text-[11px] font-extrabold uppercase tracking-wide"
                       style={{ color: claimed ? 'var(--accent)' : '#a9750a' }}>
                    <Icon name={claimed ? 'BadgeCheck' : 'Sparkles'} size={14} />
                    {claimed ? 'Offer claimed' : "Today's deal"}
                  </div>
                  <div className="font-bold text-[14.5px] mt-1.5 leading-snug" style={{ color: 'var(--ink)' }}>{p.deal.label}</div>
                  {claimed ? (
                    <div className="mt-3 flex items-center justify-between rounded-xl px-3 py-2.5"
                         style={{ background: '#fff', border: '1.5px dashed var(--accent)' }}>
                      <span className="font-mono font-extrabold tracking-widest text-[15px]" style={{ color: 'var(--accent)' }}>SPOTTO-{p.id.slice(0,4).toUpperCase()}</span>
                      <span className="text-[11.5px] font-semibold" style={{ color: 'var(--muted)' }}>Show at counter</span>
                    </div>
                  ) : (
                    <button onClick={() => setClaimed(true)}
                      className="mt-3 w-full rounded-xl py-3 font-extrabold text-[14.5px] flex items-center justify-center gap-2 active:scale-[0.98] transition-transform"
                      style={{ background: 'var(--accent)', color: '#fff', boxShadow: '0 10px 22px -10px var(--accent-shadow)' }}>
                      <Icon name="Ticket" size={17} /> Claim offer
                    </button>
                  )}
                </div>
              )}

              {/* menu (food places) */}
              {p.menu && <MenuSection menu={p.menu} />}

              {/* vibe check */}
              <div className="mt-4">
                <VibeCheck placeId={p.id} vibes={vibes} onVote={onVote} />
              </div>

              {/* artisan quest check-in */}
              {p.artisan && (
                <button onClick={() => onCheckIn(p)} disabled={didCheckIn}
                  className="mt-3 w-full rounded-2xl py-3 font-extrabold text-[14px] flex items-center justify-center gap-2 transition-transform active:scale-[0.98]"
                  style={{ background: didCheckIn ? 'var(--accent-tint)' : 'var(--sun)', color: didCheckIn ? 'var(--accent)' : '#5a3b00' }}>
                  <Icon name={didCheckIn ? 'BadgeCheck' : 'Compass'} size={17} />
                  {didCheckIn ? 'Checked in — quest progress saved' : 'Check in for Daily Quest'}
                </button>
              )}

              {/* blurb */}
              <p className="mt-4 text-[14px] leading-relaxed" style={{ color: 'var(--ink-soft)' }}>{p.blurb}</p>

              {/* tags */}
              <div className="flex flex-wrap gap-2 mt-4">
                {p.tags.map((t) => (
                  <span key={t} className="rounded-full px-3 py-1.5 text-[12.5px] font-bold"
                        style={{ background: 'var(--sand)', color: 'var(--ink-soft)' }}>{t}</span>
                ))}
              </div>

              {/* bottom actions */}
              <div className="flex gap-2.5 mt-5">
                <button className="flex-1 rounded-2xl py-3.5 font-extrabold text-[14.5px] flex items-center justify-center gap-2"
                  style={{ background: 'var(--ink)', color: '#fff' }}>
                  <Icon name="Navigation" size={17} /> Directions
                </button>
                <button className="w-14 rounded-2xl flex items-center justify-center"
                  style={{ background: 'var(--sand)', color: 'var(--ink)' }}>
                  <Icon name="Bookmark" size={19} />
                </button>
              </div>
            </div>
          </>
        )}
      </div>
    </>
  );
}

Object.assign(window, { BusinessDrawer });
