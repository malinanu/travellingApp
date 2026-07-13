// GrowthTools.jsx — vendor growth: flash sale, stamp cards, BTS uploader, boost.

const { useState: useStateGT, useEffect: useEffectGT, useRef: useRefGT } = React;

// ── 1. Flash Sale "panic button" ──
function FlashSale({ accent }) {
  const { FLASH } = window;
  const [phase, setPhase] = useStateGT('idle'); // idle | confirm | sending | sent
  const [pct, setPct] = useStateGT(30);
  const [hold, setHold] = useStateGT(0);
  const holdRef = useRefGT(null);

  function startHold() {
    if (phase !== 'idle') return;
    setPhase('confirm');
    let h = 0;
    holdRef.current = setInterval(() => {
      h += 4; setHold(h);
      if (h >= 100) { clearInterval(holdRef.current); fire(); }
    }, 24);
  }
  function cancelHold() {
    if (phase === 'confirm') { clearInterval(holdRef.current); setHold(0); setPhase('idle'); }
  }
  function fire() {
    setPhase('sending');
    setTimeout(() => setPhase('sent'), 1400);
  }
  function reset() { setHold(0); setPhase('idle'); }

  return (
    <div className="rounded-2xl overflow-hidden" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
      <div className="px-5 py-4 flex items-center gap-2" style={{ borderBottom: '1px solid var(--line)' }}>
        <span className="w-8 h-8 rounded-lg flex items-center justify-center" style={{ background: 'rgba(225,29,72,0.1)', color: '#e11d48' }}><Icon name="Zap" size={17} /></span>
        <div>
          <h3 className="font-extrabold text-[16px]" style={{ color: 'var(--ink)' }}>Flash sale trigger</h3>
          <p className="text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>Slow hour? Blast a deal to everyone nearby — instantly.</p>
        </div>
      </div>

      <div className="p-5">
        <div className="grid gap-4 items-center grid-cols-1 sm:grid-cols-[1fr_260px]">
          {/* left: config + reach */}
          <div>
            <div className="flex items-center gap-2 mb-3">
              <span className="text-[13px] font-bold" style={{ color: 'var(--ink-soft)' }}>Discount</span>
              {[20, 30, 50].map((v) => (
                <button key={v} onClick={() => setPct(v)} disabled={phase !== 'idle'}
                  className="rounded-lg px-3 py-1.5 text-[13px] font-extrabold transition-all"
                  style={{ background: pct === v ? accent : 'var(--bg)', color: pct === v ? '#fff' : 'var(--ink-soft)', border: `1px solid ${pct === v ? accent : 'var(--line)'}` }}>
                  {v}% off
                </button>
              ))}
            </div>
            <div className="rounded-xl p-4 flex items-center gap-4" style={{ background: 'var(--bg)' }}>
              <div className="relative flex items-center justify-center" style={{ width: 64, height: 64 }}>
                <Icon name="Radar" size={64} style={{ color: accent, opacity: 0.25, position: 'absolute' }} />
                <div className="text-center">
                  <div className="font-extrabold text-[18px] leading-none" style={{ color: 'var(--ink)' }}>{FLASH.inRange}</div>
                  <div className="text-[9px] font-bold uppercase" style={{ color: 'var(--muted)' }}>in range</div>
                </div>
              </div>
              <div>
                <div className="text-[13px] font-bold" style={{ color: 'var(--ink)' }}>{FLASH.inRange} travelers within {FLASH.radiusKm} km</div>
                <div className="text-[12px] font-semibold mt-0.5" style={{ color: 'var(--muted)' }}>{FLASH.nomads} nomads · {FLASH.surfers} surfers active now</div>
                <div className="text-[11.5px] font-semibold mt-1" style={{ color: 'var(--muted)' }}>Last flash sent {Math.floor(FLASH.lastSentMins/60)}h ago</div>
              </div>
            </div>
          </div>

          {/* right: the button */}
          <div className="flex flex-col items-center">
            {phase === 'sent' ? (
              <div className="text-center">
                <div className="w-20 h-20 rounded-full mx-auto flex items-center justify-center" style={{ background: '#e7f6ec', color: '#16a34a' }}>
                  <Icon name="CheckCircle2" size={40} />
                </div>
                <div className="font-extrabold text-[15px] mt-2" style={{ color: 'var(--ink)' }}>Sent to {FLASH.inRange} travelers</div>
                <div className="text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>Flash {pct}% off · live for 2 hours</div>
                <button onClick={reset} className="mt-3 text-[13px] font-bold" style={{ color: accent }}>Done</button>
              </div>
            ) : (
              <>
                <button
                  onMouseDown={startHold} onMouseUp={cancelHold} onMouseLeave={cancelHold}
                  onTouchStart={(e) => { e.preventDefault(); startHold(); }} onTouchEnd={cancelHold}
                  className="relative rounded-full flex flex-col items-center justify-center select-none"
                  style={{ width: 168, height: 168, background: 'radial-gradient(circle at 50% 35%, #f6b81e, #dd9c05)', color: '#5a3b00',
                           boxShadow: phase === 'confirm' ? '0 0 0 8px rgba(246,184,30,0.22), 0 16px 36px -12px rgba(198,141,5,0.7)' : '0 16px 36px -12px rgba(198,141,5,0.6)',
                           transform: phase === 'confirm' ? 'scale(0.97)' : 'scale(1)', transition: 'transform 120ms' }}>
                  {/* progress ring */}
                  <svg style={{ position: 'absolute', inset: 0, transform: 'rotate(-90deg)' }} width="168" height="168">
                    <circle cx="84" cy="84" r="78" fill="none" stroke="rgba(90,59,0,0.2)" strokeWidth="5" />
                    <circle cx="84" cy="84" r="78" fill="none" stroke="#5a3b00" strokeWidth="5" strokeLinecap="round"
                            strokeDasharray={2 * Math.PI * 78} strokeDashoffset={2 * Math.PI * 78 * (1 - hold / 100)} />
                  </svg>
                  <Icon name="Zap" size={40} strokeWidth={2.5} />
                  <span className="font-extrabold text-[15px] mt-1">{phase === 'sending' ? 'Sending…' : phase === 'confirm' ? 'Keep holding' : 'Flash sale'}</span>
                  <span className="text-[11px] font-bold opacity-90">{pct}% OFF · {FLASH.radiusKm}km</span>
                </button>
                <p className="text-[11.5px] font-semibold text-center mt-3" style={{ color: 'var(--muted)' }}>Press &amp; hold to broadcast</p>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

// ── 2. Digital Stamp Card manager ──
function StampCards({ accent }) {
  const { STAMP_CARD } = window;
  const oneAway = STAMP_CARD.buckets[0];
  return (
    <div className="rounded-2xl overflow-hidden" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
      <div className="px-5 py-4 flex items-center gap-2" style={{ borderBottom: '1px solid var(--line)' }}>
        <span className="w-8 h-8 rounded-lg flex items-center justify-center" style={{ background: accent + '1a', color: accent }}><Icon name="Stamp" size={17} /></span>
        <div className="flex-1">
          <h3 className="font-extrabold text-[16px]" style={{ color: 'var(--ink)' }}>Digital stamp cards</h3>
          <p className="text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>{STAMP_CARD.stampsNeeded} stamps → {STAMP_CARD.reward} · {STAMP_CARD.totalHolders} active cards</p>
        </div>
      </div>
      <div className="p-5">
        {/* hero: one stamp away */}
        <div className="rounded-xl p-4 flex items-center gap-4 mb-4" style={{ background: 'linear-gradient(120deg, var(--accent-tint), transparent)', border: `1px solid ${accent}33` }}>
          <div className="font-extrabold tracking-tight" style={{ fontSize: 40, color: accent, fontFamily: 'var(--font-display)' }}>{oneAway.count}</div>
          <div>
            <div className="font-extrabold text-[15px]" style={{ color: 'var(--ink)' }}>travelers are ONE stamp away</div>
            <div className="text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>Nudge them in and you close {oneAway.count} rewards today.</div>
          </div>
          <button className="ml-auto rounded-lg px-3.5 py-2.5 text-[13px] font-extrabold text-white flex items-center gap-1.5 flex-shrink-0" style={{ background: accent }}>
            <Icon name="Send" size={14} /> Nudge all
          </button>
        </div>
        {/* distribution */}
        <div className="grid grid-cols-3 gap-3">
          {STAMP_CARD.buckets.map((b) => (
            <div key={b.away} className="rounded-xl p-3.5" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
              <div className="flex items-center gap-1 mb-2">
                {Array.from({ length: STAMP_CARD.stampsNeeded }).map((_, i) => (
                  <span key={i} className="rounded-full" style={{ width: 9, height: 9, background: i < (STAMP_CARD.stampsNeeded - b.away) ? accent : 'var(--line)' }} />
                ))}
              </div>
              <div className="font-extrabold tracking-tight" style={{ fontSize: 22, color: 'var(--ink)' }}>{b.count}</div>
              <div className="text-[12px] font-bold" style={{ color: 'var(--muted)' }}>{b.label}</div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ── 3. Behind-the-Scenes uploader ──
function BehindScenes({ accent }) {
  const { BTS_CLIPS } = window;
  const [clips, setClips] = useStateGT(BTS_CLIPS);
  const [uploading, setUploading] = useStateGT(0);

  function fakeUpload() {
    if (uploading) return;
    setUploading(1);
    const iv = setInterval(() => setUploading((u) => {
      if (u >= 100) { clearInterval(iv);
        setClips((c) => [{ id: 'b' + Date.now(), title: 'New clip · the tailor at work', secs: 10, views: 0, grad: 'linear-gradient(135deg,#0b6f86,#0b4a54)', icon: 'Scissors' }, ...c]);
        return 0;
      }
      return u + 8;
    }), 90);
  }

  return (
    <div className="rounded-2xl overflow-hidden" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
      <div className="px-5 py-4 flex items-center gap-2" style={{ borderBottom: '1px solid var(--line)' }}>
        <span className="w-8 h-8 rounded-lg flex items-center justify-center" style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}><Icon name="Video" size={17} /></span>
        <div>
          <h3 className="font-extrabold text-[16px]" style={{ color: 'var(--ink)' }}>Behind the scenes</h3>
          <p className="text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>10-second clips of your craft — they build trust & bookings.</p>
        </div>
      </div>
      <div className="p-5">
        <div className="grid gap-3" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(150px, 1fr))' }}>
          {/* upload tile */}
          <button onClick={fakeUpload} className="rounded-xl flex flex-col items-center justify-center gap-2 transition-colors"
            style={{ aspectRatio: '9/13', border: `2px dashed ${uploading ? accent : 'var(--line)'}`, color: uploading ? accent : 'var(--muted)' }}>
            {uploading ? (
              <>
                <div className="font-extrabold text-[18px]" style={{ color: accent }}>{uploading}%</div>
                <div className="w-20 h-1.5 rounded-full overflow-hidden" style={{ background: 'var(--line)' }}>
                  <div className="h-full rounded-full" style={{ width: `${uploading}%`, background: accent }} />
                </div>
                <span className="text-[11.5px] font-bold">Uploading…</span>
              </>
            ) : (
              <>
                <Icon name="Plus" size={26} />
                <span className="text-[12.5px] font-bold text-center px-2">Record / upload<br/>10s clip</span>
                <span className="text-[10.5px] font-semibold">MP4 · max 10s</span>
              </>
            )}
          </button>
          {/* clips */}
          {clips.map((c) => (
            <div key={c.id} className="rounded-xl overflow-hidden relative group" style={{ aspectRatio: '9/13', background: c.grad }}>
              <Icon name={c.icon} size={54} style={{ position: 'absolute', right: -6, bottom: 20, color: 'rgba(255,255,255,0.2)' }} />
              <div className="absolute top-2 left-2 rounded-md px-1.5 py-0.5 text-[10px] font-extrabold flex items-center gap-1" style={{ background: 'rgba(0,0,0,0.35)', color: '#fff' }}>
                <Icon name="Clock" size={10} />{c.secs}s
              </div>
              <div className="absolute inset-0 flex items-center justify-center">
                <span className="w-11 h-11 rounded-full flex items-center justify-center" style={{ background: 'rgba(255,255,255,0.9)', color: '#111' }}><Icon name="Play" size={18} style={{ marginLeft: 2 }} /></span>
              </div>
              <div className="absolute bottom-0 left-0 right-0 p-2.5" style={{ background: 'linear-gradient(180deg, transparent, rgba(0,0,0,0.6))' }}>
                <div className="text-[11.5px] font-bold text-white leading-tight">{c.title}</div>
                <div className="text-[10.5px] font-semibold flex items-center gap-1 mt-0.5" style={{ color: 'rgba(255,255,255,0.8)' }}><Icon name="Eye" size={10} />{c.views.toLocaleString()}</div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// ── 4. Sponsored Boost ──
function SponsoredBoost({ accent }) {
  const { BOOST } = window;
  const [on, setOn] = useStateGT(false);
  const [tier, setTier] = useStateGT('category');
  const sel = BOOST.tiers.find((x) => x.id === tier);
  return (
    <div className="rounded-2xl overflow-hidden" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
      <div className="px-5 py-4 flex items-center gap-2" style={{ borderBottom: '1px solid var(--line)' }}>
        <span className="w-8 h-8 rounded-lg flex items-center justify-center" style={{ background: 'rgba(217,119,6,0.12)', color: '#d97706' }}><Icon name="Rocket" size={17} /></span>
        <div className="flex-1">
          <h3 className="font-extrabold text-[16px]" style={{ color: 'var(--ink)' }}>Promote my event</h3>
          <p className="text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>{BOOST.event.name} · {BOOST.event.when}</p>
        </div>
        <button onClick={() => setOn(!on)} className="relative rounded-full transition-colors flex-shrink-0" style={{ width: 50, height: 29, background: on ? accent : 'var(--line)' }}>
          <span className="absolute top-1 rounded-full bg-white transition-all" style={{ width: 21, height: 21, left: on ? 25 : 4, boxShadow: '0 1px 3px rgba(0,0,0,0.25)' }} />
        </button>
      </div>
      <div className="p-5" style={{ opacity: on ? 1 : 0.5, pointerEvents: on ? 'auto' : 'none', transition: 'opacity 200ms' }}>
        <div className="grid grid-cols-3 gap-3">
          {BOOST.tiers.map((tr) => {
            const active = tier === tr.id;
            return (
              <button key={tr.id} onClick={() => setTier(tr.id)} className="text-left rounded-xl p-3.5 relative transition-all"
                style={{ background: active ? accent : 'var(--bg)', color: active ? '#fff' : 'var(--ink)', border: `1.5px solid ${active ? accent : 'var(--line)'}` }}>
                {tr.best && <span className="absolute -top-2 right-2 rounded-full px-2 py-0.5 text-[9px] font-extrabold uppercase tracking-wide" style={{ background: 'var(--sun)', color: '#5a3b00' }}>Best value</span>}
                <Icon name={tr.icon} size={18} style={{ color: active ? '#fff' : accent }} />
                <div className="font-extrabold text-[13px] mt-2 leading-tight">{tr.label}</div>
                <div className="text-[11.5px] font-semibold mt-1" style={{ color: active ? 'rgba(255,255,255,0.85)' : 'var(--muted)' }}>{tr.reach}</div>
                <div className="text-[11.5px] font-semibold" style={{ color: active ? 'rgba(255,255,255,0.85)' : 'var(--muted)' }}>{tr.days} days</div>
                <div className="font-extrabold text-[18px] mt-1.5">${tr.price}</div>
              </button>
            );
          })}
        </div>
        <div className="flex items-center justify-between mt-4 rounded-xl px-4 py-3" style={{ background: 'var(--bg)' }}>
          <div>
            <div className="text-[12.5px] font-bold" style={{ color: 'var(--muted)' }}>Total to feature {BOOST.event.name.split(' ')[0]}…</div>
            <div className="font-extrabold tracking-tight" style={{ fontSize: 24, color: 'var(--ink)' }}>${sel.price}<span className="text-[13px] font-semibold" style={{ color: 'var(--muted)' }}> / {sel.days} days</span></div>
          </div>
          <button className="rounded-xl px-5 py-3 font-extrabold text-[14px] text-white flex items-center gap-2" style={{ background: accent }}>
            <Icon name="Rocket" size={16} /> Boost event
          </button>
        </div>
      </div>
    </div>
  );
}

function GrowthTools({ accent }) {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="font-extrabold tracking-tight" style={{ fontSize: 28, lineHeight: 1.15, color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Growth tools</h1>
        <p className="text-[14.5px] font-semibold mt-1" style={{ color: 'var(--muted)' }}>Fill slow hours, reward regulars, and get discovered — in a couple of taps.</p>
      </div>
      <FlashSale accent={accent} />
      <div className="grid gap-6" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(380px, 100%), 1fr))' }}>
        <StampCards accent={accent} />
        <SponsoredBoost accent={accent} />
      </div>
      <BehindScenes accent={accent} />
    </div>
  );
}

Object.assign(window, { GrowthTools });
