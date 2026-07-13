// MapView.jsx — satellite map + pins + time filter + Smart Companion layer:
// weather bar, group mode, draggable "you" + live distance, daily quest.

const { useMemo: useMemoMV, useState: useStateMV, useRef: useRefMV, useEffect: useEffectMV } = React;

const PIN_COLORS = {
  teal:  { bg: '#0b6f86', ring: '#0b6f86' },
  coral: { bg: '#f0682f', ring: '#f0682f' },
  sun:   { bg: '#eaa200', ring: '#eaa200' },
  clay:  { bg: '#c2683f', ring: '#c2683f' },
  plum:  { bg: '#5a5f9e', ring: '#5a5f9e' },
};

function Pin({ place, prioritized, onClick }) {
  const c = PIN_COLORS[place.color] || PIN_COLORS.teal;
  return (
    <button onClick={onClick}
      style={{
        position: 'absolute', left: `${place.x}%`, top: `${place.y}%`,
        transform: `translate(-50%,-100%) scale(${prioritized ? 1 : 0.82})`,
        transformOrigin: 'bottom center', zIndex: prioritized ? 20 : 10,
        transition: 'transform 400ms cubic-bezier(0.22,1,0.36,1), opacity 400ms ease',
        opacity: prioritized ? 1 : 0.62,
        display: 'flex', flexDirection: 'column', alignItems: 'center',
      }}>
      {prioritized && (
        <div className="mb-1 whitespace-nowrap rounded-full px-2.5 py-1 flex items-center gap-1"
             style={{ background: '#fff', boxShadow: '0 6px 16px -6px rgba(16,61,58,0.4)' }}>
          <span className="font-extrabold text-[11.5px]" style={{ color: 'var(--ink)' }}>{place.type}</span>
        </div>
      )}
      <div style={{ position: 'relative' }}>
        {prioritized && (
          <span style={{
            position: 'absolute', left: '50%', top: '40%', width: 46, height: 46,
            transform: 'translate(-50%,-50%)', borderRadius: '50%',
            background: c.ring, opacity: 0.28, animation: 'spottoPulse 2.2s ease-out infinite',
          }} />
        )}
        <div style={{
          width: 40, height: 40, borderRadius: '50% 50% 50% 0',
          transform: 'rotate(-45deg)', background: c.bg,
          border: '2.5px solid #fff', boxShadow: '0 8px 18px -6px rgba(16,61,58,0.5)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}>
          <span style={{ transform: 'rotate(45deg)', color: '#fff', display: 'flex' }}>
            <Icon name={place.icon} size={19} strokeWidth={2.4} />
          </span>
        </div>
      </div>
    </button>
  );
}

function FriendAvatar({ f }) {
  return (
    <div style={{ position: 'absolute', left: `${f.x}%`, top: `${f.y}%`, transform: 'translate(-50%,-50%)', zIndex: 15,
                  display: 'flex', flexDirection: 'column', alignItems: 'center', pointerEvents: 'none' }}>
      <div style={{ width: 34, height: 34, borderRadius: '50%', background: f.color, border: '3px solid #fff',
                    boxShadow: '0 6px 14px -4px rgba(16,61,58,0.5)', display: 'flex', alignItems: 'center', justifyContent: 'center',
                    color: '#fff', fontWeight: 800, fontSize: 14 }}>{f.initial}</div>
      <div className="mt-1 whitespace-nowrap rounded-full px-2 py-0.5 text-[10px] font-extrabold"
           style={{ background: 'rgba(16,61,58,0.65)', color: '#fff', backdropFilter: 'blur(4px)' }}>{f.name}</div>
    </div>
  );
}

// meters between two map-percent points (simulated GEODIST)
function geodist(a, b) {
  const MX = 16, MY = 23; // metres per percent (x, y)
  const dx = (a.x - b.x) * MX, dy = (a.y - b.y) * MY;
  return Math.round(Math.sqrt(dx * dx + dy * dy) / 10) * 10;
}

function MapView({ time, setTime, places, onSelect, styleLabel, weather, cycleWeather, groupMode, setGroupMode, members = [], onOpenGroup, quest, onQuestTap, onProfile }) {
  const isRainy = weather === 'rainy';
  const wx = window.WEATHER[weather] || window.WEATHER.sunny;

  // prioritise: when rainy, indoor places win; otherwise time-of-day match
  const prioritized = useMemoMV(() => {
    if (isRainy) return new Set(places.filter((p) => p.indoor).map((p) => p.id));
    return new Set(places.filter((p) => p.bestTimes.includes(time)).map((p) => p.id));
  }, [places, time, isRainy]);

  const topPick = useMemoMV(() => {
    const pool = isRainy ? places.filter((p) => p.indoor) : places.filter((p) => p.bestTimes.includes(time));
    return (pool.length ? pool : places).slice().sort((a, b) => b.rating - a.rating)[0];
  }, [places, time, isRainy]);

  const timeHint = isRainy ? 'Rainy — indoor spots boosted'
    : { morning: 'Great for focused mornings', afternoon: 'Best surf & café hours', evening: 'Golden-hour picks rising' }[time];

  // ── draggable "you" + live distance ──
  const wrapRef = useRefMV(null);
  const [you, setYou] = useStateMV({ x: 45, y: 70 });
  const [timeOpen, setTimeOpen] = useStateMV(false);
  const dragging = useRefMV(false);

  const dealTarget = useMemoMV(() => {
    const deals = places.filter((p) => p.deal?.active);
    if (!deals.length) return null;
    return deals.map((p) => ({ p, d: geodist(you, p) })).sort((a, b) => a.d - b.d)[0];
  }, [places, you]);

  function moveTo(clientX, clientY) {
    const r = wrapRef.current?.getBoundingClientRect();
    if (!r) return;
    const x = Math.max(4, Math.min(96, ((clientX - r.left) / r.width) * 100));
    const y = Math.max(30, Math.min(94, ((clientY - r.top) / r.height) * 100));
    setYou({ x, y });
  }
  useEffectMV(() => {
    const mv = (e) => { if (!dragging.current) return; const t = e.touches ? e.touches[0] : e; moveTo(t.clientX, t.clientY); };
    const up = () => { dragging.current = false; };
    window.addEventListener('pointermove', mv); window.addEventListener('pointerup', up);
    return () => { window.removeEventListener('pointermove', mv); window.removeEventListener('pointerup', up); };
  }, []);

  return (
    <div ref={wrapRef} className="h-full relative" style={{ overflow: 'hidden' }}>
      <SatelliteMap time={isRainy ? 'afternoon' : time} />

      {/* rain overlay */}
      {isRainy && (
        <div style={{ position: 'absolute', inset: 0, zIndex: 5, pointerEvents: 'none',
          background: 'linear-gradient(180deg, rgba(40,55,70,0.45), rgba(30,45,60,0.25))' }}>
          <div className="spotto-rain" style={{ position: 'absolute', inset: 0 }} />
        </div>
      )}

      {/* tap-away catcher for the time menu */}
      {timeOpen && <div className="absolute inset-0" style={{ zIndex: 29 }} onClick={() => setTimeOpen(false)} />}

      {/* top floating bar */}
      <div className="absolute left-0 right-0 px-4 z-30" style={{ top: 58 }}>
        <div className="flex items-center justify-between gap-2">
          {/* weather-integrated location chip */}
          <button onClick={cycleWeather}
            className="rounded-2xl px-3 py-2 flex items-center gap-2.5 active:scale-95 transition-transform"
            style={{ background: 'rgba(255,255,255,0.92)', backdropFilter: 'blur(10px)', boxShadow: '0 6px 20px -8px rgba(16,61,58,0.35)' }}>
            <div className="flex items-center justify-center w-9 h-9 rounded-xl"
                 style={{ background: isRainy ? 'rgba(80,120,160,0.18)' : 'var(--accent-tint)', color: isRainy ? '#3d6b9e' : 'var(--accent)' }}>
              <Icon name={wx.icon} size={20} />
            </div>
            <div className="leading-none text-left">
              <div className="font-extrabold text-[14px] flex items-center gap-1" style={{ color: 'var(--ink)' }}>
                {wx.label} · {wx.temp}
              </div>
              <div className="text-[10.5px] font-semibold mt-0.5" style={{ color: 'var(--muted)' }}>Weligama · {styleLabel}</div>
            </div>
          </button>

          <div className="flex items-center gap-2">
            {/* group button — opens the manage/invite sheet */}
            <button onClick={onOpenGroup}
              className="h-11 rounded-2xl flex items-center gap-1.5 pl-2.5 pr-3 active:scale-95 transition-all"
              style={{ background: groupMode ? 'var(--accent)' : 'rgba(255,255,255,0.92)', color: groupMode ? '#fff' : 'var(--ink)',
                       backdropFilter: 'blur(10px)', boxShadow: '0 6px 20px -8px rgba(16,61,58,0.35)' }}>
              {groupMode && members.length > 0 ? (
                <span className="flex -space-x-2 items-center">
                  {members.slice(0, 3).map((m) => (
                    <span key={m.id} style={{ width: 22, height: 22, borderRadius: '50%', background: m.color, border: '2px solid var(--accent)',
                                              display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontWeight: 800, fontSize: 10 }}>{m.initial}</span>
                  ))}
                </span>
              ) : (
                <Icon name="Users" size={18} strokeWidth={2.3} />
              )}
              <span className="text-[12px] font-extrabold">{groupMode ? members.length + 1 : 'Group'}</span>
            </button>
            <button className="w-11 h-11 rounded-2xl flex items-center justify-center"
                    style={{ background: 'rgba(255,255,255,0.92)', backdropFilter: 'blur(10px)', color: 'var(--ink)', boxShadow: '0 6px 20px -8px rgba(16,61,58,0.35)' }}>
              <Icon name="Search" size={20} />
            </button>
            <button onClick={onProfile} aria-label="Profile"
                    className="w-11 h-11 rounded-2xl flex items-center justify-center font-extrabold text-white active:scale-95 transition-transform"
                    style={{ background: 'linear-gradient(140deg, var(--accent), var(--accent-2))', boxShadow: '0 6px 20px -8px rgba(16,61,58,0.4)', border: '2px solid rgba(255,255,255,0.92)' }}>
              <span className="text-[14px]">LM</span>
            </button>
          </div>
        </div>

        {/* time-of-day selector — compact button that expands to the other times */}
        <div className="mt-3 relative" style={{ zIndex: 32 }}>
          {(() => {
            const cur = window.TIMES.find((x) => x.id === time) || window.TIMES[0];
            return (
              <button onClick={() => setTimeOpen((v) => !v)} aria-label={`Time of day: ${cur.label}`}
                className="flex items-center justify-center rounded-2xl active:scale-95 transition-transform"
                style={{ width: 44, height: 44, background: 'rgba(255,255,255,0.92)', backdropFilter: 'blur(10px)', boxShadow: '0 6px 20px -8px rgba(16,61,58,0.35)', color: 'var(--accent)' }}>
                <Icon name={cur.icon} size={21} />
              </button>
            );
          })()}
          {timeOpen && (
            <div className="absolute left-0 mt-2 rounded-2xl p-1.5 flex flex-col gap-0.5"
                 style={{ top: '100%', minWidth: 168, background: '#fff', boxShadow: '0 16px 36px -12px rgba(16,61,58,0.5)',
                          animation: 'spottoGuideIn 200ms cubic-bezier(0.22,1,0.36,1)' }}>
              {window.TIMES.filter((x) => x.id !== time).map((t) => (
                <button key={t.id} onClick={() => { setTime(t.id); setTimeOpen(false); }}
                  className="flex items-center gap-2.5 rounded-xl px-2 py-2 active:scale-[0.98] transition-transform"
                  style={{ color: 'var(--ink)' }}>
                  <span className="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0" style={{ background: 'var(--sand)', color: 'var(--accent)' }}>
                    <Icon name={t.icon} size={16} />
                  </span>
                  <span className="text-[13px] font-extrabold">{t.label}</span>
                </button>
              ))}
            </div>
          )}
        </div>
        <div className="text-center mt-2">
          <span className="rounded-full px-3 py-1 text-[11.5px] font-bold inline-flex items-center gap-1.5"
                style={{ background: 'rgba(16,61,58,0.55)', color: '#fff', backdropFilter: 'blur(6px)' }}>
            <Icon name={isRainy ? 'CloudRain' : 'Sparkles'} size={12} /> {timeHint}
          </span>
        </div>
      </div>

      {/* pins */}
      {places.map((p) => (
        <Pin key={p.id} place={p} prioritized={prioritized.has(p.id)} onClick={() => onSelect(p)} />
      ))}

      {/* friends (group mode) */}
      {groupMode && members.map((f) => <FriendAvatar key={f.id} f={f} />)}

      {/* draggable "you" */}
      <div
        onPointerDown={(e) => { dragging.current = true; e.currentTarget.setPointerCapture?.(e.pointerId); }}
        style={{ position: 'absolute', left: `${you.x}%`, top: `${you.y}%`, transform: 'translate(-50%,-50%)',
                 zIndex: 25, cursor: 'grab', touchAction: 'none' }}>
        <span style={{ position: 'absolute', left: '50%', top: '50%', width: 40, height: 40, transform: 'translate(-50%,-50%)',
                       borderRadius: '50%', background: '#2f7df6', opacity: 0.25, animation: 'spottoPulse 2.4s ease-out infinite' }} />
        <div style={{ width: 22, height: 22, borderRadius: '50%', background: '#2f7df6', border: '3px solid #fff',
                      boxShadow: '0 4px 12px -2px rgba(16,61,58,0.6)' }} />
      </div>

      {/* live distance badge (GEODIST) */}
      {dealTarget && (
        <div className="absolute left-1/2 z-30" style={{ top: 188, transform: 'translateX(-50%)' }}>
          <div className="rounded-full pl-1.5 pr-3 py-1.5 flex items-center gap-2 whitespace-nowrap"
               style={{ background: '#fff', boxShadow: '0 10px 24px -10px rgba(16,61,58,0.5)' }}>
            <span className="w-7 h-7 rounded-full flex items-center justify-center" style={{ background: '#eaf2ff', color: '#2f7df6' }}>
              <Icon name="Navigation" size={14} />
            </span>
            <span className="text-[12.5px] font-extrabold" style={{ color: 'var(--ink)' }}>
              {dealTarget.d}m to {dealTarget.p.deal.tag}
            </span>
            <span className="text-[10px] font-bold rounded px-1.5 py-0.5" style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}>LIVE</span>
          </div>
          <div className="text-center mt-1 text-[10px] font-bold" style={{ color: 'rgba(255,255,255,0.85)', textShadow: '0 1px 4px rgba(0,0,0,0.5)' }}>
            drag the blue dot · {dealTarget.p.name.split(' ').slice(0,2).join(' ')}
          </div>
        </div>
      )}

      {/* bottom stack: daily quest + top pick */}
      <div className="absolute left-4 right-4 z-30 space-y-2.5" style={{ bottom: 16 }}>
        {/* daily quest */}
        <button onClick={onQuestTap}
          className="w-full rounded-2xl p-2.5 flex items-center gap-3 text-left active:scale-[0.99] transition-transform"
          style={{ background: quest.done >= quest.goal ? 'linear-gradient(120deg,#fff6df,#ffe9c2)' : 'rgba(255,255,255,0.96)',
                   backdropFilter: 'blur(12px)', boxShadow: '0 10px 28px -14px rgba(16,61,58,0.4)',
                   border: quest.done >= quest.goal ? '1.5px solid var(--sun)' : '1.5px solid transparent' }}>
          <div className="w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0"
               style={{ background: quest.done >= quest.goal ? 'var(--sun)' : 'rgba(246,184,30,0.16)', color: quest.done >= quest.goal ? '#5a3b00' : '#c98a00' }}>
            <Icon name={quest.done >= quest.goal ? 'Gift' : 'Compass'} size={20} />
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-1.5">
              <span className="text-[9.5px] font-extrabold uppercase tracking-wide px-1.5 py-0.5 rounded" style={{ background: 'rgba(246,184,30,0.18)', color: '#a9750a' }}>Daily quest</span>
              <span className="text-[10.5px] font-bold" style={{ color: 'var(--muted)' }}>{quest.done}/{quest.goal}</span>
            </div>
            <div className="font-extrabold text-[12.5px] mt-0.5 truncate" style={{ color: 'var(--ink)' }}>
              {quest.done >= quest.goal ? '🎉 Free Ceylon Tea unlocked!' : 'Visit 2 local artisans → free Ceylon Tea'}
            </div>
            <div className="h-1.5 rounded-full mt-1.5 overflow-hidden" style={{ background: 'rgba(15,61,58,0.1)' }}>
              <div className="h-full rounded-full transition-all duration-500" style={{ width: `${Math.min(100, quest.done / quest.goal * 100)}%`, background: 'var(--sun)' }} />
            </div>
          </div>
          <Icon name="ChevronRight" size={20} style={{ color: 'var(--muted)' }} />
        </button>

        {/* top pick */}
        {topPick && (
          <button onClick={() => onSelect(topPick)}
            className="w-full rounded-3xl p-2.5 flex items-center gap-3 text-left active:scale-[0.99] transition-transform"
            style={{ background: 'rgba(255,255,255,0.96)', backdropFilter: 'blur(12px)', boxShadow: '0 14px 34px -12px rgba(16,61,58,0.45)' }}>
            <div className="w-14 h-14 rounded-2xl flex items-center justify-center flex-shrink-0"
                 style={{ background: (PIN_COLORS[topPick.color]||PIN_COLORS.teal).bg, color: '#fff' }}>
              <Icon name={topPick.icon} size={24} />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-1.5">
                <span className="text-[10.5px] font-extrabold uppercase tracking-wide px-1.5 py-0.5 rounded"
                      style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}>{isRainy ? 'Best indoor pick' : 'Top pick now'}</span>
              </div>
              <div className="font-extrabold text-[14.5px] mt-1 truncate" style={{ color: 'var(--ink)' }}>{topPick.name}</div>
              <div className="flex items-center gap-2 text-[12px] font-semibold mt-0.5" style={{ color: 'var(--muted)' }}>
                <span className="flex items-center gap-1"><Icon name="Star" size={12} style={{ fill: 'var(--sun)', color: 'var(--sun)' }} />{topPick.rating}</span>
                {topPick.deal?.active && <span className="flex items-center gap-1" style={{ color: 'var(--accent)' }}><Icon name="Tag" size={12} />{topPick.deal.tag}</span>}
              </div>
            </div>
            <Icon name="ChevronRight" size={22} style={{ color: 'var(--muted)' }} />
          </button>
        )}
      </div>
    </div>
  );
}

Object.assign(window, { MapView });
