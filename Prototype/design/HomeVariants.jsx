// HomeVariants.jsx — alternative "Discover" home layouts that reuse the same
// PLACES data + design system as MapView. Two concepts:
//   • HomeFeed  — editorial, scrollable discovery feed (hero pick + live rail + list)
//   • HomeToday — personal "right now" briefing (weather hero + agenda + quest)

const { useMemo: useMemoHV } = React;

// photo placeholders — flat Sunny Postcard scenes from illustrations.jsx

function hvGreeting(time) {
  return { morning: 'Good morning', afternoon: 'Good afternoon', evening: 'Good evening' }[time] || 'Hello';
}

function hvTopPick(places, time, isRainy) {
  const pool = isRainy ? places.filter((p) => p.indoor) : places.filter((p) => p.bestTimes.includes(time));
  return (pool.length ? pool : places).slice().sort((a, b) => b.rating - a.rating)[0];
}

// compact horizontal time pills shared by both layouts
function TimeChips({ time, setTime }) {
  return (
    <div className="flex items-center gap-1.5">
      {window.TIMES.map((t) => {
        const on = t.id === time;
        return (
          <button key={t.id} onClick={() => setTime(t.id)}
            className="flex items-center gap-1.5 rounded-full pl-2 pr-3 py-1.5 active:scale-95 transition-all"
            style={{ background: on ? 'var(--accent)' : 'rgba(255,255,255,0.92)', color: on ? '#fff' : 'var(--ink)',
                     boxShadow: '0 4px 14px -8px rgba(16,61,58,0.4)' }}>
            <Icon name={t.icon} size={15} strokeWidth={2.4} />
            <span className="text-[12px] font-extrabold">{t.label}</span>
          </button>
        );
      })}
    </div>
  );
}

function ProfileAvatar({ onProfile }) {
  return (
    <button onClick={onProfile} aria-label="Profile"
      className="w-11 h-11 rounded-2xl flex items-center justify-center font-extrabold text-white flex-shrink-0 active:scale-95 transition-transform"
      style={{ background: 'linear-gradient(140deg, var(--accent), var(--accent-2))', boxShadow: '0 6px 20px -8px rgba(16,61,58,0.4)', border: '2px solid rgba(255,255,255,0.92)' }}>
      <span className="text-[14px]">LM</span>
    </button>
  );
}

function QuestBanner({ quest, onQuestTap }) {
  const done = quest.done >= quest.goal;
  return (
    <button onClick={onQuestTap}
      className="w-full rounded-2xl p-2.5 flex items-center gap-3 text-left active:scale-[0.99] transition-transform"
      style={{ background: done ? 'linear-gradient(120deg,#fff6df,#ffe9c2)' : '#fff',
               boxShadow: '0 10px 28px -16px rgba(16,61,58,0.35)',
               border: done ? '1.5px solid var(--sun)' : '1.5px solid rgba(15,61,58,0.06)' }}>
      <div className="w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0"
           style={{ background: done ? 'var(--sun)' : 'rgba(246,184,30,0.16)', color: done ? '#5a3b00' : '#c98a00' }}>
        <Icon name={done ? 'Gift' : 'Compass'} size={20} />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5">
          <span className="text-[9.5px] font-extrabold uppercase tracking-wide px-1.5 py-0.5 rounded" style={{ background: 'rgba(246,184,30,0.18)', color: '#a9750a' }}>Daily quest</span>
          <span className="text-[10.5px] font-bold" style={{ color: 'var(--muted)' }}>{quest.done}/{quest.goal}</span>
        </div>
        <div className="font-extrabold text-[12.5px] mt-0.5 truncate" style={{ color: 'var(--ink)' }}>
          {done ? '🎉 Free Ceylon Tea unlocked!' : 'Visit 2 local artisans → free Ceylon Tea'}
        </div>
        <div className="h-1.5 rounded-full mt-1.5 overflow-hidden" style={{ background: 'rgba(15,61,58,0.1)' }}>
          <div className="h-full rounded-full transition-all duration-500" style={{ width: `${Math.min(100, quest.done / quest.goal * 100)}%`, background: 'var(--sun)' }} />
        </div>
      </div>
      <Icon name="ChevronRight" size={20} style={{ color: 'var(--muted)' }} />
    </button>
  );
}

/* ════════════════════════════════════════════════════════════
   HOME · FEED  — editorial discovery feed
   ════════════════════════════════════════════════════════════ */
function HomeFeed({ places, onSelect, styleLabel, weather, time, setTime, quest, onQuestTap, onProfile }) {
  const isRainy = weather === 'rainy';
  const wx = window.WEATHER[weather] || window.WEATHER.sunny;
  const top = useMemoHV(() => hvTopPick(places, time, isRainy), [places, time, isRainy]);
  const live = useMemoHV(() => places.filter((p) => p.deal?.active && p.id !== (top && top.id)), [places, top]);
  const rest = useMemoHV(() => places.filter((p) => p.id !== (top && top.id)), [places, top]);

  return (
    <div className="h-full overflow-y-auto" style={{ background: 'var(--sand)' }}>
      {/* header */}
      <div className="px-5" style={{ paddingTop: 58 }}>
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="text-[12.5px] font-bold uppercase tracking-wide" style={{ color: 'var(--accent)' }}>{hvGreeting(time)}</div>
            <h1 className="font-extrabold leading-tight mt-0.5" style={{ fontSize: 27, color: 'var(--ink)' }}>Weligama Bay</h1>
            <div className="flex items-center gap-1.5 mt-1 text-[12.5px] font-bold" style={{ color: 'var(--muted)' }}>
              <Icon name={wx.icon} size={15} style={{ color: isRainy ? '#3d6b9e' : 'var(--accent)' }} />
              {wx.label} · {wx.temp} · {styleLabel}
            </div>
          </div>
          <ProfileAvatar onProfile={onProfile} />
        </div>
        {/* search */}
        <button className="w-full mt-3 rounded-2xl px-3.5 h-12 flex items-center gap-2.5"
          style={{ background: '#fff', boxShadow: '0 6px 20px -14px rgba(16,61,58,0.4)', color: 'var(--muted)' }}>
          <Icon name="Search" size={18} />
          <span className="text-[13.5px] font-semibold">Search cafés, surf, artisans…</span>
        </button>
        <div className="mt-3 overflow-x-auto -mx-5 px-5 pb-1">
          <TimeChips time={time} setTime={setTime} />
        </div>
      </div>

      {/* hero — top pick */}
      {top && (
        <div className="px-4 mt-4">
          <button onClick={() => onSelect(top)}
            className="w-full text-left rounded-3xl overflow-hidden active:scale-[0.99] transition-transform"
            style={{ boxShadow: '0 18px 40px -20px rgba(16,61,58,0.4) ' }}>
            <div style={{ position: 'relative', height: 198, background: 'var(--sand)' }}>
              <Illustration kind={top.photo} />
              <div className="absolute top-3.5 left-3.5 inline-flex items-center gap-1.5 rounded-full px-3 py-1.5"
                   style={{ background: 'rgba(255,255,255,0.95)', color: 'var(--accent)' }}>
                <Icon name="Sparkles" size={13} strokeWidth={2.6} />
                <span className="text-[11px] font-extrabold uppercase tracking-wide">{isRainy ? 'Best indoor pick' : 'Top pick now'}</span>
              </div>
              {top.deal?.active && (
                <div className="absolute top-3.5 right-3.5 inline-flex items-center gap-1 rounded-full px-2.5 py-1.5"
                     style={{ background: 'var(--sun)', color: '#5a3b00' }}>
                  <Icon name="Tag" size={12} strokeWidth={2.6} /><span className="text-[11.5px] font-extrabold">{top.deal.tag}</span>
                </div>
              )}
              <div className="absolute left-4 right-4 bottom-3.5">
                <div className="text-white font-extrabold leading-tight" style={{ fontSize: 22, textShadow: '0 2px 12px rgba(0,0,0,0.4)' }}>{top.name}</div>
                <div className="flex items-center gap-3 mt-1.5 text-white text-[12.5px] font-bold" style={{ textShadow: '0 1px 6px rgba(0,0,0,0.5)' }}>
                  <span className="flex items-center gap-1"><Icon name="Star" size={13} style={{ fill: 'var(--sun)', color: 'var(--sun)' }} />{top.rating}</span>
                  <span className="flex items-center gap-1"><Icon name="Navigation" size={12} />{top.distance}</span>
                  <span className="flex items-center gap-1"><Icon name="Wifi" size={13} />{top.wifi.score}</span>
                </div>
              </div>
            </div>
          </button>
        </div>
      )}

      {/* live nearby rail */}
      {live.length > 0 && (
        <div className="mt-5">
          <div className="px-5 flex items-center justify-between">
            <h2 className="font-extrabold text-[16px] whitespace-nowrap" style={{ color: 'var(--ink)' }}>Happening now</h2>
            <span className="flex items-center gap-1 text-[11px] font-extrabold rounded-full px-2 py-0.5" style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}>
              <span style={{ width: 6, height: 6, borderRadius: '50%', background: 'var(--accent)', display: 'inline-block' }} /> LIVE
            </span>
          </div>
          <div className="mt-2.5 flex gap-3 overflow-x-auto px-5 pb-1">
            {live.map((p) => (
              <button key={p.id} onClick={() => onSelect(p)}
                className="text-left rounded-2xl overflow-hidden flex-shrink-0 active:scale-[0.98] transition-transform"
                style={{ width: 168, background: '#fff', boxShadow: '0 10px 26px -16px rgba(16,61,58,0.4)' }}>
                <div style={{ position: 'relative', height: 92, background: 'var(--sand)' }}>
                  <Illustration kind={p.photo} scrim={false} />
                  <div className="absolute top-2 left-2 inline-flex items-center gap-1 rounded-full px-2 py-0.5" style={{ background: 'var(--sun)', color: '#5a3b00' }}>
                    <Icon name="Tag" size={10} strokeWidth={2.8} /><span className="text-[10px] font-extrabold">{p.deal.tag}</span>
                  </div>
                </div>
                <div className="px-3 py-2.5">
                  <div className="font-extrabold text-[13px] leading-tight truncate" style={{ color: 'var(--ink)' }}>{p.name}</div>
                  <div className="flex items-center gap-2 mt-1 text-[11.5px] font-bold" style={{ color: 'var(--muted)' }}>
                    <span className="flex items-center gap-0.5"><Icon name="Star" size={11} style={{ fill: 'var(--sun)', color: 'var(--sun)' }} />{p.rating}</span>
                    <span className="flex items-center gap-0.5"><Icon name="Navigation" size={11} />{p.distance}</span>
                  </div>
                </div>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* quest */}
      <div className="px-4 mt-5"><QuestBanner quest={quest} onQuestTap={onQuestTap} /></div>

      {/* more spots */}
      <div className="px-5 mt-5 flex items-center justify-between">
        <h2 className="font-extrabold text-[16px]" style={{ color: 'var(--ink)' }}>More in the bay</h2>
        <span className="text-[12px] font-bold" style={{ color: 'var(--muted)' }}>{rest.length} spots</span>
      </div>
      <div className="px-4 mt-2.5 pb-28 space-y-2.5">
        {rest.map((p) => (
          <button key={p.id} onClick={() => onSelect(p)}
            className="w-full text-left rounded-2xl p-2.5 flex items-center gap-3 active:scale-[0.99] transition-transform"
            style={{ background: '#fff', boxShadow: '0 8px 22px -16px rgba(16,61,58,0.35)' }}>
            <div className="w-14 h-14 rounded-2xl flex items-center justify-center flex-shrink-0" style={{ background: `${ILLO_TONE[p.photo] || ILLO_TONE.cowork}18`, color: ILLO_TONE[p.photo] || ILLO_TONE.cowork }}>
              <Icon name={p.icon} size={24} />
            </div>
            <div className="min-w-0 flex-1">
              <div className="font-extrabold text-[14px] leading-tight truncate" style={{ color: 'var(--ink)' }}>{p.name}</div>
              <div className="flex items-center gap-2 mt-1 text-[12px] font-bold" style={{ color: 'var(--muted)' }}>
                <span className="rounded-full px-2 py-0.5 text-[11px]" style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}>{p.type}</span>
                <span className="flex items-center gap-0.5"><Icon name="Star" size={11} style={{ fill: 'var(--sun)', color: 'var(--sun)' }} />{p.rating}</span>
              </div>
            </div>
            <Icon name="ChevronRight" size={20} style={{ color: 'var(--muted)' }} />
          </button>
        ))}
      </div>
    </div>
  );
}

/* ════════════════════════════════════════════════════════════
   HOME · TODAY  — personal "right now" briefing
   ════════════════════════════════════════════════════════════ */
function HomeToday({ places, onSelect, styleLabel, weather, time, setTime, members = [], groupMode, quest, onQuestTap, onProfile, onOpenGroup }) {
  const isRainy = weather === 'rainy';
  const wx = window.WEATHER[weather] || window.WEATHER.sunny;
  const openCount = places.filter((p) => p.openNow).length;
  const dealCount = places.filter((p) => p.deal?.active).length;
  const agenda = useMemoHV(() => {
    const pool = isRainy ? places.filter((p) => p.indoor) : places.filter((p) => p.bestTimes.includes(time));
    return (pool.length ? pool : places).slice().sort((a, b) => b.rating - a.rating).slice(0, 3);
  }, [places, time, isRainy]);

  return (
    <div className="h-full overflow-y-auto" style={{ background: 'var(--sand)' }}>
      {/* hero */}
      <div style={{ position: 'relative', background: 'linear-gradient(150deg, var(--accent), var(--accent-2))',
                    borderBottomLeftRadius: 30, borderBottomRightRadius: 30, paddingTop: 58 }}
           className="px-5 pb-6">
        <div style={{ position: 'absolute', inset: 0, opacity: 0.5, borderRadius: 'inherit',
                      background: 'radial-gradient(110% 80% at 88% -8%, rgba(255,255,255,0.4), transparent 55%)' }} />
        <div className="relative flex items-start justify-between gap-3">
          <div>
            <div className="text-white/85 text-[12.5px] font-bold uppercase tracking-wide">{hvGreeting(time)}, Lena</div>
            <h1 className="text-white font-extrabold leading-tight mt-0.5" style={{ fontSize: 26 }}>Here's your Weligama</h1>
          </div>
          <ProfileAvatar onProfile={onProfile} />
        </div>
        {/* weather block */}
        <div className="relative mt-4 flex items-center gap-3">
          <div className="flex items-center justify-center rounded-2xl w-16 h-16 flex-shrink-0"
               style={{ background: 'rgba(255,255,255,0.18)', color: '#fff', backdropFilter: 'blur(6px)' }}>
            <Icon name={wx.icon} size={34} />
          </div>
          <div className="leading-none">
            <div className="text-white font-extrabold" style={{ fontSize: 34 }}>{wx.temp}</div>
            <div className="text-white/90 text-[13px] font-bold mt-1">{wx.label} · {wx.tip}</div>
          </div>
        </div>
        {/* stat row */}
        <div className="relative mt-4 grid grid-cols-3 gap-2">
          {[
            { n: places.length, l: 'spots near', i: 'MapPin' },
            { n: openCount, l: 'open now', i: 'Clock' },
            { n: dealCount, l: 'live deals', i: 'Tag' },
          ].map((s) => (
            <div key={s.l} className="rounded-2xl px-3 py-2.5" style={{ background: 'rgba(255,255,255,0.16)', backdropFilter: 'blur(6px)' }}>
              <div className="flex items-center gap-1.5 text-white/85"><Icon name={s.i} size={13} /><span className="text-[10.5px] font-bold uppercase tracking-wide">{s.l}</span></div>
              <div className="text-white font-extrabold mt-0.5" style={{ fontSize: 22 }}>{s.n}</div>
            </div>
          ))}
        </div>
      </div>

      <div className="px-5 -mt-3 relative">
        {/* group strip */}
        <button onClick={onOpenGroup}
          className="w-full rounded-2xl p-3 flex items-center gap-3 text-left active:scale-[0.99] transition-transform"
          style={{ background: '#fff', boxShadow: '0 12px 30px -18px rgba(16,61,58,0.45)' }}>
          {groupMode && members.length > 0 ? (
            <>
              <span className="flex -space-x-2.5 items-center flex-shrink-0">
                {members.slice(0, 3).map((m) => (
                  <span key={m.id} style={{ width: 32, height: 32, borderRadius: '50%', background: m.color, border: '2.5px solid #fff', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontWeight: 800, fontSize: 13 }}>{m.initial}</span>
                ))}
              </span>
              <div className="min-w-0 flex-1">
                <div className="font-extrabold text-[13.5px]" style={{ color: 'var(--ink)' }}>{members.length} friends exploring nearby</div>
                <div className="text-[12px] font-semibold truncate" style={{ color: 'var(--muted)' }}>{members.map((m) => m.name).join(', ')}</div>
              </div>
            </>
          ) : (
            <>
              <div className="w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0" style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}>
                <Icon name="Users" size={20} />
              </div>
              <div className="min-w-0 flex-1">
                <div className="font-extrabold text-[13.5px]" style={{ color: 'var(--ink)' }}>Explore together</div>
                <div className="text-[12px] font-semibold" style={{ color: 'var(--muted)' }}>Start a group & share the bay</div>
              </div>
            </>
          )}
          <Icon name="ChevronRight" size={20} style={{ color: 'var(--muted)' }} />
        </button>
      </div>

      {/* agenda */}
      <div className="px-5 mt-5 flex items-center justify-between">
        <h2 className="font-extrabold text-[16px]" style={{ color: 'var(--ink)' }}>
          {isRainy ? 'Dry-day picks' : `Picked for your ${time}`}
        </h2>
        <div className="overflow-x-auto"><TimeChips time={time} setTime={setTime} /></div>
      </div>
      <div className="px-4 mt-2.5 space-y-2.5">
        {agenda.map((p, i) => (
          <button key={p.id} onClick={() => onSelect(p)}
            className="w-full text-left rounded-2xl overflow-hidden flex items-stretch active:scale-[0.99] transition-transform"
            style={{ background: '#fff', boxShadow: '0 8px 24px -16px rgba(16,61,58,0.4)' }}>
            <div style={{ width: 88, background: 'var(--sand)', position: 'relative', flexShrink: 0 }}>
              <Illustration kind={p.photo} scrim={false} />
              <div className="absolute top-2 left-2 w-6 h-6 rounded-full flex items-center justify-center text-[12px] font-extrabold" style={{ background: 'rgba(255,255,255,0.95)', color: 'var(--accent)' }}>{i + 1}</div>
            </div>
            <div className="px-3.5 py-3 min-w-0 flex-1 flex flex-col justify-center">
              <div className="font-extrabold text-[14.5px] leading-tight truncate" style={{ color: 'var(--ink)' }}>{p.name}</div>
              <div className="flex items-center gap-2.5 mt-1.5 text-[12px] font-bold" style={{ color: 'var(--muted)' }}>
                <span className="flex items-center gap-1"><Icon name="Star" size={12} style={{ fill: 'var(--sun)', color: 'var(--sun)' }} />{p.rating}</span>
                <span className="flex items-center gap-1"><Icon name="Navigation" size={11} />{p.distance}</span>
                {p.deal?.active && <span className="flex items-center gap-1" style={{ color: 'var(--accent)' }}><Icon name="Tag" size={11} />{p.deal.tag}</span>}
              </div>
            </div>
            <div className="flex items-center pr-2"><Icon name="ChevronRight" size={20} style={{ color: 'var(--muted)' }} /></div>
          </button>
        ))}
      </div>

      {/* quest */}
      <div className="px-4 mt-4 pb-28"><QuestBanner quest={quest} onQuestTap={onQuestTap} /></div>
    </div>
  );
}

Object.assign(window, { HomeFeed, HomeToday });
