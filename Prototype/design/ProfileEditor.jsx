// ProfileEditor.jsx — Wi-Fi status, sea-view photos, best time to visit, amenities.

const { useState: useStatePE } = React;

const TIME_OPTS = [
  { id: 'morning', label: 'Morning', icon: 'Sunrise' },
  { id: 'afternoon', label: 'Afternoon', icon: 'Sun' },
  { id: 'evening', label: 'Evening', icon: 'Sunset' },
];

const PHOTO_SLOTS = [
  { id: 'p1', label: 'Sea View', grad: 'linear-gradient(135deg, #2bb6c4, #0b6f86)', icon: 'Waves', filled: true },
  { id: 'p2', label: 'Co-work loft', grad: 'linear-gradient(135deg, #0e8c8a, #0a5f6e)', icon: 'Laptop', filled: true },
  { id: 'p3', label: 'Coffee bar', grad: 'linear-gradient(135deg, #c98a3c, #8a5a23)', icon: 'Coffee', filled: true },
  { id: 'p4', label: 'Add photo', grad: null, icon: 'ImagePlus', filled: false },
];

function Toggle({ on, onClick, accent }) {
  return (
    <button onClick={onClick} className="relative rounded-full transition-colors flex-shrink-0"
      style={{ width: 48, height: 28, background: on ? accent : 'rgba(15,61,58,0.18)' }}>
      <span className="absolute top-1 rounded-full bg-white transition-all"
            style={{ width: 20, height: 20, left: on ? 24 : 4, boxShadow: '0 1px 3px rgba(0,0,0,0.2)' }} />
    </button>
  );
}

function Card({ title, sub, children, icon, accent }) {
  return (
    <div className="rounded-2xl p-5 bg-white" style={{ border: '1px solid var(--line)' }}>
      <div className="flex items-center gap-2.5 mb-3">
        {icon && <span className="w-9 h-9 rounded-xl flex items-center justify-center" style={{ background: 'var(--accent-tint)', color: accent }}><Icon name={icon} size={18} /></span>}
        <div>
          <h3 className="font-extrabold text-[15.5px]" style={{ color: 'var(--ink)' }}>{title}</h3>
          {sub && <p className="text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>{sub}</p>}
        </div>
      </div>
      {children}
    </div>
  );
}

function ProfileEditor({ accent, businessType, setBusinessType }) {
  const { PROFILE, VENDOR } = window;
  const [wifi, setWifi] = useStatePE(PROFILE.wifi.enabled);
  const [wifiScore, setWifiScore] = useStatePE(PROFILE.wifi.score);
  const [seaView, setSeaView] = useStatePE(PROFILE.seaView);
  const [times, setTimes] = useStatePE(PROFILE.bestTimes);
  const [blurb, setBlurb] = useStatePE(PROFILE.blurb);
  const [saved, setSaved] = useStatePE(false);

  const toggleTime = (id) => setTimes((t) => t.includes(id) ? t.filter((x) => x !== id) : [...t, id]);
  function save() { setSaved(true); setTimeout(() => setSaved(false), 2400); }

  return (
    <div className="space-y-6">
      <div className="flex items-end justify-between gap-4">
        <div className="min-w-0">
          <h1 className="font-extrabold tracking-tight" style={{ fontSize: 28, lineHeight: 1.15, color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Business profile</h1>
          <p className="text-[14.5px] font-semibold mt-1" style={{ color: 'var(--muted)' }}>This is what travelers see on the Spotto map.</p>
        </div>
        <button onClick={save}
          className="rounded-xl px-5 py-2.5 font-extrabold text-[14px] flex items-center gap-2 text-white active:scale-[0.98] transition-transform flex-shrink-0 whitespace-nowrap"
          style={{ background: saved ? '#16a34a' : accent, boxShadow: '0 8px 18px -8px var(--accent-shadow)' }}>
          {saved ? <><Icon name="Check" size={17} strokeWidth={3} /> Saved</> : <><Icon name="Save" size={17} /> Save changes</>}
        </button>
      </div>

      <div className="grid gap-6" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(330px, 100%), 1fr))' }}>
        {/* business type */}
        <Card title="Business type" sub="What kind of spot are you?" icon="Store" accent={accent}>
          <div className="grid grid-cols-3 gap-2">
            {window.BUSINESS_TYPES.map((b) => {
              const on = businessType === b.id;
              return (
                <button key={b.id} onClick={() => setBusinessType && setBusinessType(b.id)}
                  className="flex flex-col items-center gap-1.5 rounded-xl py-3 px-1 transition-all active:scale-[0.97]"
                  style={{ background: on ? 'var(--accent-tint)' : 'var(--sand)',
                           border: `1.5px solid ${on ? accent : 'var(--line)'}` }}>
                  <Icon name={b.icon} size={19} style={{ color: on ? accent : 'var(--muted)' }} />
                  <span className="text-[11.5px] font-extrabold leading-tight text-center" style={{ color: on ? accent : 'var(--ink-soft)' }}>{b.label}</span>
                </button>
              );
            })}
          </div>
          <div className="flex items-start gap-2 mt-3 rounded-xl px-3 py-2.5" style={{ background: 'var(--sand)' }}>
            <Icon name="UtensilsCrossed" size={15} style={{ color: accent, flexShrink: 0, marginTop: 1 }} />
            <p className="text-[12px] font-semibold leading-snug" style={{ color: 'var(--muted)' }}>
              Restaurants, cafés and bars get a <span style={{ color: 'var(--ink)' }}>Menu</span> tab — upload dishes with photos and prices.
            </p>
          </div>
        </Card>

        {/* photos */}
        <Card title="Photos" sub="Show off your sea view & space" icon="Camera" accent={accent}>
          <div className="grid grid-cols-2 gap-2.5">
            {PHOTO_SLOTS.map((p) => p.filled ? (
              <div key={p.id} className="relative rounded-xl overflow-hidden group" style={{ aspectRatio: '4/3', background: p.grad }}>
                <Icon name={p.icon} size={52} style={{ position: 'absolute', right: -4, bottom: -8, color: 'rgba(255,255,255,0.25)' }} />
                <div className="absolute left-2 bottom-2 rounded-md px-2 py-0.5 text-[11px] font-extrabold text-white" style={{ background: 'rgba(0,0,0,0.3)' }}>{p.label}</div>
                {p.id === 'p1' && (
                  <div className="absolute right-2 top-2 rounded-full px-2 py-0.5 text-[10px] font-extrabold flex items-center gap-1" style={{ background: 'var(--sun)', color: '#5a3b00' }}>
                    <Icon name="Star" size={10} style={{ fill: 'currentColor' }} /> Cover
                  </div>
                )}
              </div>
            ) : (
              <button key={p.id} className="rounded-xl flex flex-col items-center justify-center gap-1 transition-colors hover:opacity-80"
                style={{ aspectRatio: '4/3', border: '2px dashed var(--line)', color: 'var(--muted)' }}>
                <Icon name={p.icon} size={24} />
                <span className="text-[12px] font-bold">{p.label}</span>
              </button>
            ))}
          </div>
        </Card>

        {/* wifi + sea view */}
        <Card title="Connectivity & vibe" sub="What digital nomads check first" icon="Wifi" accent={accent}>
          <div className="flex items-center justify-between py-2.5">
            <div>
              <div className="font-bold text-[14px]" style={{ color: 'var(--ink)' }}>Wi-Fi available</div>
              <div className="text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>Show a Wi-Fi badge on your pin</div>
            </div>
            <Toggle on={wifi} onClick={() => setWifi(!wifi)} accent={accent} />
          </div>
          {wifi && (
            <div className="py-2.5" style={{ borderTop: '1px solid var(--line)' }}>
              <div className="flex items-center justify-between mb-2">
                <span className="font-bold text-[14px]" style={{ color: 'var(--ink)' }}>Speed rating</span>
                <span className="font-extrabold text-[15px]" style={{ color: accent }}>{wifiScore.toFixed(1)} / 10</span>
              </div>
              <input type="range" min="1" max="10" step="0.1" value={wifiScore}
                     onChange={(e) => setWifiScore(parseFloat(e.target.value))}
                     className="w-full" style={{ accentColor: accent }} />
              <div className="flex justify-between text-[11.5px] font-semibold mt-1" style={{ color: 'var(--muted)' }}>
                <span>Patchy</span><span>Blazing fast</span>
              </div>
            </div>
          )}
          <div className="flex items-center justify-between py-2.5" style={{ borderTop: '1px solid var(--line)' }}>
            <div>
              <div className="font-bold text-[14px] flex items-center gap-1.5" style={{ color: 'var(--ink)' }}>
                <Icon name="Waves" size={15} style={{ color: accent }} /> Sea view
              </div>
              <div className="text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>Tag your spot as ocean-facing</div>
            </div>
            <Toggle on={seaView} onClick={() => setSeaView(!seaView)} accent={accent} />
          </div>
        </Card>

        {/* best time */}
        <Card title="Best time to visit" sub="Helps Spotto rank you at the right hours" icon="Clock" accent={accent}>
          <div className="grid grid-cols-3 gap-2.5">
            {TIME_OPTS.map((t) => {
              const on = times.includes(t.id);
              return (
                <button key={t.id} onClick={() => toggleTime(t.id)}
                  className="rounded-xl py-3.5 flex flex-col items-center gap-1.5 transition-all active:scale-95"
                  style={{ background: on ? accent : '#fff', color: on ? '#fff' : 'var(--ink-soft)',
                           border: on ? `1.5px solid ${accent}` : '1.5px solid var(--line)' }}>
                  <Icon name={t.icon} size={22} />
                  <span className="text-[12.5px] font-extrabold">{t.label}</span>
                </button>
              );
            })}
          </div>
          <p className="text-[12px] font-semibold mt-3" style={{ color: 'var(--muted)' }}>Selected windows get a small boost in map ranking during those hours.</p>
        </Card>

        {/* description + amenities */}
        <Card title="About & amenities" sub="A short, honest pitch" icon="FileText" accent={accent}>
          <textarea value={blurb} onChange={(e) => setBlurb(e.target.value)} rows={3}
            className="w-full rounded-xl px-3.5 py-3 text-[13.5px] font-semibold outline-none resize-none leading-relaxed"
            style={{ background: '#fff', border: '1.5px solid var(--line)', color: 'var(--ink)' }} />
          <div className="flex flex-wrap gap-2 mt-3">
            {PROFILE.amenities.map((a) => (
              <span key={a} className="rounded-full px-3 py-1.5 text-[12.5px] font-bold flex items-center gap-1.5"
                    style={{ background: 'var(--sand)', color: 'var(--ink-soft)' }}>
                <Icon name="Check" size={13} style={{ color: accent }} />{a}
              </span>
            ))}
            <button className="rounded-full px-3 py-1.5 text-[12.5px] font-extrabold flex items-center gap-1"
                    style={{ border: `1.5px dashed ${accent}`, color: accent }}>
              <Icon name="Plus" size={13} /> Add
            </button>
          </div>
        </Card>
      </div>
    </div>
  );
}

Object.assign(window, { ProfileEditor });
