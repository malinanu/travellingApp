// GroupSheet.jsx — manage the travel Group: view members, invite via link/QR,
// add nearby travellers, remove people. Bottom sheet, matches AddStopSheet style.
// Props: open, onClose, group{name,code}, members[], suggestions[], me,
//        onAdd(id), onRemove(id), onLeave().

const { useState: useStateGS, useEffect: useEffectGS } = React;

// initials-bubble avatar with a live status dot
function Avatar({ t, size = 44, dot = true }) {
  return (
    <span style={{ position: 'relative', flexShrink: 0 }}>
      <span style={{
        width: size, height: size, borderRadius: '50%', background: t.color,
        border: '2.5px solid #fff', boxShadow: '0 4px 12px -4px rgba(16,61,58,0.4)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        color: '#fff', fontWeight: 800, fontSize: t.initial.length > 1 ? size * 0.34 : size * 0.42,
      }}>{t.initial}</span>
      {dot && (
        <span style={{
          position: 'absolute', right: -1, bottom: -1, width: size * 0.28, height: size * 0.28,
          minWidth: 11, minHeight: 11, borderRadius: '50%', background: '#3ec97a', border: '2.5px solid #fff',
        }} />
      )}
    </span>
  );
}

function MiniQR({ size = 64 }) {
  // deterministic faux-QR so it looks like a real share code (decorative only)
  const n = 9, cell = size / n;
  const seed = [0x1f4, 0x2a7, 0x35b, 0x1c9, 0x2e6, 0x3b1, 0x14d, 0x2f2, 0x39a];
  const inFinder = (r, c) => (r < 3 && c < 3) || (r < 3 && c > n - 4) || (r > n - 4 && c < 3);
  const cells = [];
  for (let r = 0; r < n; r++) for (let c = 0; c < n; c++) {
    if (!inFinder(r, c) && ((seed[r] >> c) & 1) === 1) cells.push({ r, c });
  }
  const finder = (fr, fc) => (
    <g key={`f${fr}${fc}`}>
      <rect x={fc * cell} y={fr * cell} width={cell * 3} height={cell * 3} rx={cell * 0.5} fill="var(--ink)" />
      <rect x={(fc + 0.6) * cell} y={(fr + 0.6) * cell} width={cell * 1.8} height={cell * 1.8} rx={cell * 0.35} fill="#fff" />
      <rect x={(fc + 1) * cell} y={(fr + 1) * cell} width={cell} height={cell} rx={cell * 0.25} fill="var(--ink)" />
    </g>
  );
  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} style={{ display: 'block' }}>
      {cells.map((x) => (
        <rect key={`${x.r}-${x.c}`} x={x.c * cell + cell * 0.1} y={x.r * cell + cell * 0.1}
              width={cell * 0.8} height={cell * 0.8} rx={cell * 0.2} fill="var(--ink)" />
      ))}
      {finder(0, 0)}{finder(0, n - 3)}{finder(n - 3, 0)}
    </svg>
  );
}

function MemberRow({ t, host, onRemove }) {
  return (
    <div className="rounded-2xl bg-white p-2.5 flex items-center gap-3" style={{ border: '1.5px solid rgba(15,61,58,0.07)' }}>
      <Avatar t={t} />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-1.5">
          <span className="font-extrabold text-[14px] truncate" style={{ color: 'var(--ink)' }}>{t.name}</span>
          {host && (
            <span className="text-[9.5px] font-extrabold uppercase tracking-wide px-1.5 py-0.5 rounded-md"
                  style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}>Host</span>
          )}
        </div>
        <div className="text-[12px] font-semibold truncate flex items-center gap-1" style={{ color: 'var(--muted)' }}>
          <Icon name="MapPin" size={12} style={{ color: 'var(--accent)' }} /> near {t.at}
        </div>
      </div>
      {host ? (
        <span className="text-[11.5px] font-extrabold px-2 whitespace-nowrap" style={{ color: 'var(--muted)' }}>You</span>
      ) : (
        <button onClick={() => onRemove(t.id)} aria-label={`Remove ${t.name}`}
          className="w-9 h-9 rounded-xl flex items-center justify-center flex-shrink-0 active:scale-90 transition-transform"
          style={{ background: 'var(--sand)', color: '#c2666f' }}>
          <Icon name="UserMinus" size={17} strokeWidth={2.2} />
        </button>
      )}
    </div>
  );
}

function SuggestRow({ t, onAdd, justAdded }) {
  return (
    <div className="rounded-2xl p-2.5 flex items-center gap-3" style={{ background: 'rgba(255,255,255,0.6)', border: '1.5px solid rgba(15,61,58,0.07)' }}>
      <Avatar t={t} dot={false} />
      <div className="flex-1 min-w-0">
        <div className="font-extrabold text-[14px] truncate" style={{ color: 'var(--ink)' }}>{t.name}</div>
        <div className="text-[12px] font-semibold truncate" style={{ color: 'var(--muted)' }}>{t.sub}</div>
      </div>
      <button onClick={() => onAdd(t.id)}
        className="rounded-xl px-3 h-9 flex items-center gap-1.5 flex-shrink-0 font-extrabold text-[12.5px] active:scale-95 transition-transform"
        style={{ background: justAdded ? '#3ec97a' : 'var(--accent)', color: '#fff', boxShadow: '0 6px 16px -8px var(--accent-shadow)' }}>
        <Icon name={justAdded ? 'Check' : 'Plus'} size={16} strokeWidth={2.6} /> {justAdded ? 'Added' : 'Add'}
      </button>
    </div>
  );
}

function GroupSheet({ open, onClose, group, members = [], suggestions = [], me, onAdd, onRemove, onLeave }) {
  const [copied, setCopied] = useStateGS(false);
  const [showQR, setShowQR] = useStateGS(false);
  const [justAdded, setJustAdded] = useStateGS(null);
  useEffectGS(() => { if (open) { setCopied(false); setShowQR(false); setJustAdded(null); } }, [open]);
  if (!open) return null;

  const link = `spotto.app/g/${group.code}`;
  function copy() {
    try { navigator.clipboard?.writeText('https://' + link); } catch {}
    setCopied(true); setTimeout(() => setCopied(false), 1800);
  }
  function add(id) { onAdd(id); setJustAdded(id); setTimeout(() => setJustAdded(null), 1200); }

  return (
    <div className="absolute inset-0 z-50 flex flex-col justify-end">
      <div className="absolute inset-0" style={{ background: 'rgba(16,61,58,0.45)' }} onClick={onClose} />
      <div className="relative rounded-t-[28px] flex flex-col" style={{ background: 'var(--sand)', maxHeight: '86%', boxShadow: '0 -16px 44px -20px rgba(16,61,58,0.6)' }}>
        <div className="w-10 h-1.5 rounded-full mx-auto mt-3 mb-2 flex-shrink-0" style={{ background: 'rgba(15,61,58,0.15)' }} />

        {/* header */}
        <div className="px-5 pb-3 flex items-center justify-between flex-shrink-0">
          <div className="min-w-0">
            <div className="text-[11px] font-extrabold uppercase tracking-wider" style={{ color: 'var(--accent)' }}>Travel group</div>
            <h3 className="font-extrabold text-[21px] leading-tight truncate" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>{group.name}</h3>
          </div>
          <div className="flex -space-x-2.5 items-center flex-shrink-0 pl-3">
            {members.slice(0, 4).map((m) => (
              <span key={m.id} style={{ width: 30, height: 30, borderRadius: '50%', background: m.color, border: '2.5px solid var(--sand)',
                                        display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontWeight: 800, fontSize: 12 }}>{m.initial}</span>
            ))}
            <span style={{ width: 30, height: 30, borderRadius: '50%', background: '#fff', border: '2.5px solid var(--sand)',
                           display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--ink)', fontWeight: 800, fontSize: 11.5 }}>{members.length + 1}</span>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto px-5 pb-7">
          {/* invite card */}
          <div className="rounded-3xl p-4" style={{ background: 'linear-gradient(135deg, var(--accent), var(--accent-2))', boxShadow: '0 16px 32px -16px var(--accent-shadow)' }}>
            <div className="flex items-center gap-3">
              <div className="flex-1 min-w-0">
                <div className="text-[12px] font-extrabold flex items-center gap-1.5" style={{ color: 'rgba(255,255,255,0.85)' }}>
                  <Icon name="Link" size={14} /> Invite link
                </div>
                <div className="font-extrabold text-[15px] text-white truncate mt-0.5">{link}</div>
              </div>
              <button onClick={() => setShowQR((v) => !v)} aria-label="Show QR code"
                className="w-11 h-11 rounded-2xl flex items-center justify-center flex-shrink-0 active:scale-95 transition-transform"
                style={{ background: 'rgba(255,255,255,0.2)', color: '#fff' }}>
                <Icon name="QrCode" size={20} />
              </button>
            </div>
            {showQR && (
              <div className="mt-3 rounded-2xl bg-white p-3 flex items-center gap-3.5">
                <div className="rounded-xl p-1.5" style={{ background: '#fff', border: '1.5px solid rgba(15,61,58,0.1)' }}><MiniQR size={66} /></div>
                <div className="min-w-0">
                  <div className="font-extrabold text-[13.5px]" style={{ color: 'var(--ink)' }}>Scan to join</div>
                  <div className="text-[12px] font-semibold leading-snug mt-0.5" style={{ color: 'var(--muted)' }}>Point a camera here — they'll drop straight into {group.name}.</div>
                </div>
              </div>
            )}
            <div className="flex gap-2 mt-3">
              <button onClick={copy}
                className="flex-1 rounded-2xl py-2.5 font-extrabold text-[14px] flex items-center justify-center gap-2 active:scale-[0.98] transition-transform"
                style={{ background: '#fff', color: 'var(--accent)' }}>
                <Icon name={copied ? 'Check' : 'Copy'} size={16} strokeWidth={2.6} /> {copied ? 'Copied!' : 'Copy link'}
              </button>
              <button
                className="flex-1 rounded-2xl py-2.5 font-extrabold text-[14px] flex items-center justify-center gap-2 active:scale-[0.98] transition-transform text-white"
                style={{ background: 'rgba(255,255,255,0.18)', border: '1.5px solid rgba(255,255,255,0.4)' }}>
                <Icon name="Share2" size={16} strokeWidth={2.4} /> Share
              </button>
            </div>
          </div>

          {/* members */}
          <div className="flex items-center justify-between mt-5 mb-2 px-1">
            <span className="text-[11.5px] font-extrabold uppercase tracking-wide" style={{ color: 'var(--muted)' }}>In your group · {members.length + 1}</span>
            <span className="text-[11px] font-bold flex items-center gap-1" style={{ color: '#3ec97a' }}>
              <span style={{ width: 7, height: 7, borderRadius: '50%', background: '#3ec97a', display: 'inline-block' }} /> Live
            </span>
          </div>
          <div className="space-y-2">
            <MemberRow t={me} host />
            {members.map((m) => <MemberRow key={m.id} t={m} onRemove={onRemove} />)}
          </div>

          {/* suggestions */}
          {suggestions.length > 0 && (
            <>
              <div className="flex items-center gap-1.5 mt-5 mb-2 px-1">
                <Icon name="UserPlus" size={13} style={{ color: 'var(--accent)' }} />
                <span className="text-[11.5px] font-extrabold uppercase tracking-wide" style={{ color: 'var(--muted)' }}>Add travellers nearby</span>
              </div>
              <div className="space-y-2">
                {suggestions.map((s) => <SuggestRow key={s.id} t={s} onAdd={add} justAdded={justAdded === s.id} />)}
              </div>
            </>
          )}

          {/* leave */}
          <button onClick={onLeave}
            className="w-full rounded-2xl py-3 mt-5 font-extrabold text-[13.5px] flex items-center justify-center gap-2 active:scale-[0.98] transition-transform"
            style={{ background: 'transparent', color: '#c2666f', border: '1.5px solid rgba(194,102,111,0.35)' }}>
            <Icon name="LogOut" size={15} strokeWidth={2.4} /> Leave group
          </button>
        </div>
      </div>
    </div>
  );
}

Object.assign(window, { GroupSheet });
