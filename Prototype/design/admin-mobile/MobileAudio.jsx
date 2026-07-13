// MobileAudio.jsx — geotagged audio guide manager: library cards + new-guide form.

const { useState: useStateMA } = React;

function DepthRowM({ depth, active, onToggle, accent }) {
  const info = window.DEPTH_INFO[depth];
  return (
    <button onClick={onToggle}
      className="w-full text-left rounded-xl p-3 flex items-start gap-3 transition-all"
      style={{ background: active ? accent : 'var(--bg)', color: active ? '#fff' : 'var(--ink)',
               border: active ? `1.5px solid ${accent}` : '1.5px solid var(--line)' }}>
      <span className="w-9 h-9 rounded-lg flex items-center justify-center flex-shrink-0" style={{ background: active ? 'rgba(255,255,255,0.2)' : 'var(--surface)', color: active ? '#fff' : accent }}>
        <Icon name={info.icon} size={17} />
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="font-bold text-[14px]">{depth}</span>
          <span className="text-[11px] font-semibold" style={{ color: active ? 'rgba(255,255,255,0.8)' : 'var(--muted)' }}>{info.mins}</span>
        </div>
        <div className="text-[11.5px] font-medium mt-0.5 leading-snug" style={{ color: active ? 'rgba(255,255,255,0.85)' : 'var(--ink-soft)' }}>{info.desc}</div>
      </div>
      <span className="w-5 h-5 rounded-full flex items-center justify-center flex-shrink-0 mt-0.5" style={{ border: active ? 'none' : '1.5px solid var(--line)', background: active ? 'rgba(255,255,255,0.25)' : 'transparent' }}>
        {active && <Icon name="Check" size={13} strokeWidth={3} />}
      </span>
    </button>
  );
}

function Field({ label, children }) {
  return (
    <div>
      <label className="block text-[11px] font-bold uppercase tracking-wide mb-1.5" style={{ color: 'var(--muted)' }}>{label}</label>
      {children}
    </div>
  );
}

const INPUT_CLS = "w-full rounded-xl px-3.5 py-2.5 text-[14px] font-semibold outline-none";
const inputStyle = { background: 'var(--bg)', border: '1px solid var(--line)', color: 'var(--ink)' };

function NewGuide({ accent }) {
  const { DEPTH_INFO } = window;
  const [site, setSite] = useStateMA('');
  const [region, setRegion] = useStateMA('Galle');
  const [lat, setLat] = useStateMA('6.0269');
  const [lng, setLng] = useStateMA('80.2168');
  const [depths, setDepths] = useStateMA(['Quick', 'Standard']);
  const [files, setFiles] = useStateMA({});
  const [published, setPublished] = useStateMA(false);

  const toggleDepth = (d) => setDepths((p) => p.includes(d) ? p.filter((x) => x !== d) : [...p, d]);
  const fakeUpload = (d) => setFiles((f) => ({ ...f, [d]: `${site || 'guide'}_${d.toLowerCase()}.mp3` }));
  const disabled = !site || depths.length === 0;

  return (
    <div className="h-full flex flex-col">
      <div className="flex-1 overflow-y-auto noscroll" style={{ padding: 'var(--pad)', display: 'flex', flexDirection: 'column', gap: 16 }}>
        <Field label="Historical site">
          <input value={site} onChange={(e) => setSite(e.target.value)} placeholder="e.g. Galle Fort Ramparts" className={INPUT_CLS} style={inputStyle} />
        </Field>
        <Field label="Region">
          <input value={region} onChange={(e) => setRegion(e.target.value)} className={INPUT_CLS} style={inputStyle} />
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field label="Latitude">
            <input value={lat} onChange={(e) => setLat(e.target.value)} className={INPUT_CLS + ' font-mono'} style={inputStyle} />
          </Field>
          <Field label="Longitude">
            <input value={lng} onChange={(e) => setLng(e.target.value)} className={INPUT_CLS + ' font-mono'} style={inputStyle} />
          </Field>
        </div>

        <Field label="Depth options · pick what to offer">
          <div className="flex flex-col gap-2">
            {Object.keys(DEPTH_INFO).map((d) => (
              <DepthRowM key={d} depth={d} active={depths.includes(d)} onToggle={() => toggleDepth(d)} accent={accent} />
            ))}
          </div>
        </Field>

        <Field label="Audio files">
          <div className="flex flex-col gap-2">
            {depths.length === 0 && <p className="text-[13px] font-medium" style={{ color: 'var(--muted)' }}>Select at least one depth above.</p>}
            {depths.map((d) => (
              <div key={d} className="flex items-center gap-2.5 rounded-xl px-3.5 py-2.5" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
                <Icon name={DEPTH_INFO[d].icon} size={16} style={{ color: accent }} />
                <span className="font-bold text-[13px] w-16 flex-shrink-0" style={{ color: 'var(--ink)' }}>{d}</span>
                {files[d] ? (
                  <span className="flex items-center gap-1 text-[11.5px] font-mono font-semibold truncate" style={{ color: '#16a34a' }}><Icon name="CheckCircle2" size={13} />{files[d]}</span>
                ) : (
                  <span className="text-[12px] font-medium" style={{ color: 'var(--muted)' }}>No file</span>
                )}
                <button onClick={() => fakeUpload(d)} className="ml-auto flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-[12px] font-bold flex-shrink-0" style={{ background: 'var(--surface)', border: '1px solid var(--line)', color: 'var(--ink)' }}>
                  <Icon name="Upload" size={13} /> {files[d] ? 'Replace' : 'Upload'}
                </button>
              </div>
            ))}
          </div>
        </Field>
      </div>

      {/* sticky publish */}
      <div className="flex-shrink-0 px-3.5 py-3" style={{ background: 'var(--surface)', borderTop: '1px solid var(--line)' }}>
        <button onClick={() => { setPublished(true); setTimeout(() => setPublished(false), 2400); }} disabled={disabled}
          className="w-full rounded-xl py-3 font-bold text-[15px] flex items-center justify-center gap-2 text-white active:scale-[0.98] transition-all"
          style={{ background: published ? '#16a34a' : disabled ? 'var(--muted)' : accent, opacity: disabled ? 0.5 : 1 }}>
          {published ? <><Icon name="Check" size={18} strokeWidth={3} /> Published!</> : <><Icon name="Globe" size={17} /> Publish guide</>}
        </button>
        <p className="text-[11px] font-medium text-center mt-1.5" style={{ color: 'var(--muted)' }}>Goes live at the geotagged location</p>
      </div>
    </div>
  );
}

function Library({ accent }) {
  const { AUDIO_GUIDES, DEPTH_INFO } = window;
  return (
    <div className="h-full overflow-y-auto noscroll flex flex-col" style={{ gap: 'var(--gap)', padding: 'var(--pad)' }}>
      {AUDIO_GUIDES.map((g) => (
        <div key={g.id} className="rounded-2xl overflow-hidden flex-shrink-0" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
          <div className="h-24 relative" style={{ background: 'linear-gradient(135deg, #134e4a, #0c2b28)' }}>
            <Icon name="Landmark" size={66} style={{ position: 'absolute', right: -6, bottom: -10, color: 'rgba(45,212,191,0.18)' }} />
            <div className="absolute top-3 left-4 right-4 flex items-start justify-between">
              <div>
                <div className="font-bold text-[15px] text-white">{g.site}</div>
                <div className="font-mono text-[11px] flex items-center gap-1 mt-0.5" style={{ color: 'rgba(255,255,255,0.6)' }}>
                  <Icon name="MapPin" size={11} />{g.lat.toFixed(4)}, {g.lng.toFixed(4)}
                </div>
              </div>
              <span className="rounded-full px-2 py-0.5 text-[10px] font-bold flex-shrink-0" style={{ background: g.published ? 'rgba(22,163,74,0.9)' : 'rgba(217,119,6,0.9)', color: '#fff' }}>
                {g.published ? 'Published' : 'Draft'}
              </span>
            </div>
          </div>
          <div style={{ padding: 'var(--pad)' }}>
            <div className="flex items-center gap-1.5 flex-wrap">
              {g.depths.map((d) => (
                <span key={d} className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-[11px] font-bold" style={{ background: 'var(--bg)', color: 'var(--ink-soft)', border: '1px solid var(--line)' }}>
                  <Icon name={DEPTH_INFO[d].icon} size={11} style={{ color: accent }} />{d} · {g.duration[d]}
                </span>
              ))}
            </div>
            <div className="flex items-center justify-between mt-3 pt-3" style={{ borderTop: '1px solid var(--line)' }}>
              <span className="flex items-center gap-1.5 text-[12px] font-semibold" style={{ color: 'var(--muted)' }}>
                <Icon name="Play" size={13} />{g.plays.toLocaleString()} plays · {g.lang}
              </span>
              <button className="text-[12.5px] font-bold flex items-center gap-1" style={{ color: accent }}>Edit <Icon name="ChevronRight" size={14} /></button>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

function MobileAudio({ accent }) {
  const [view, setView] = useStateMA('library');
  return (
    <div className="h-full flex flex-col">
      <div className="flex-shrink-0" style={{ padding: 'var(--pad)', paddingBottom: 10 }}>
        <h1 className="font-bold tracking-tight" style={{ fontSize: 24, color: 'var(--ink)' }}>Audio guides</h1>
        <div className="flex items-center gap-2 mt-2.5 rounded-full p-1" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
          {[['library', 'Library'], ['new', 'New guide']].map(([k, lbl]) => {
            const on = view === k;
            return (
              <button key={k} onClick={() => setView(k)}
                className="flex-1 rounded-full py-1.5 text-[13px] font-bold transition-colors flex items-center justify-center gap-1.5"
                style={{ background: on ? 'var(--surface)' : 'transparent', color: on ? accent : 'var(--muted)', boxShadow: on ? '0 1px 3px rgba(0,0,0,0.08)' : 'none' }}>
                {k === 'new' && <Icon name="Plus" size={15} strokeWidth={2.5} />}{lbl}
              </button>
            );
          })}
        </div>
      </div>
      <div className="flex-1 min-h-0">
        {view === 'library' ? <Library accent={accent} /> : <NewGuide accent={accent} />}
      </div>
    </div>
  );
}

Object.assign(window, { MobileAudio });
