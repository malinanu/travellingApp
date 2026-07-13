// AudioCMS.jsx — geotagged audio guide manager + uploader with depth tiers.

const { useState: useStateAC } = React;

function DepthChip({ depth, active, onToggle, accent }) {
  const info = window.DEPTH_INFO[depth];
  return (
    <button onClick={onToggle}
      className="text-left rounded-xl p-3.5 transition-all"
      style={{ background: active ? accent : 'var(--bg)', color: active ? '#fff' : 'var(--ink)',
               border: active ? `1.5px solid ${accent}` : '1.5px solid var(--line)' }}>
      <div className="flex items-center justify-between">
        <span className="w-8 h-8 rounded-lg flex items-center justify-center" style={{ background: active ? 'rgba(255,255,255,0.2)' : 'var(--surface)', color: active ? '#fff' : accent }}>
          <Icon name={info.icon} size={16} />
        </span>
        {active && <Icon name="Check" size={16} strokeWidth={3} />}
      </div>
      <div className="font-bold text-[14px] mt-2">{depth}</div>
      <div className="text-[11.5px] font-semibold mt-0.5" style={{ color: active ? 'rgba(255,255,255,0.8)' : 'var(--muted)' }}>{info.mins}</div>
      <div className="text-[11.5px] font-medium mt-1 leading-snug" style={{ color: active ? 'rgba(255,255,255,0.85)' : 'var(--ink-soft)' }}>{info.desc}</div>
    </button>
  );
}

function AudioCMS({ accent }) {
  const { AUDIO_GUIDES, DEPTH_INFO } = window;
  const [view, setView] = useStateAC('library'); // library | new
  const [site, setSite] = useStateAC('');
  const [region, setRegion] = useStateAC('Galle');
  const [lat, setLat] = useStateAC('6.0269');
  const [lng, setLng] = useStateAC('80.2168');
  const [depths, setDepths] = useStateAC(['Quick', 'Standard']);
  const [files, setFiles] = useStateAC({});
  const [published, setPublished] = useStateAC(false);

  const toggleDepth = (d) => setDepths((p) => p.includes(d) ? p.filter((x) => x !== d) : [...p, d]);
  const fakeUpload = (d) => setFiles((f) => ({ ...f, [d]: `${site || 'guide'}_${d.toLowerCase()}.mp3` }));

  return (
    <div className="h-full flex flex-col">
      <div className="flex items-end justify-between gap-4 mb-4">
        <div>
          <h1 className="font-bold tracking-tight" style={{ fontSize: 22, color: 'var(--ink)' }}>Audio guide CMS</h1>
          <p className="text-[13px] font-medium mt-0.5" style={{ color: 'var(--muted)' }}>Geotagged audio guides for historical sites · depth tiers</p>
        </div>
        <div className="flex items-center gap-2">
          <button onClick={() => setView('library')} className="rounded-lg px-3 py-2 text-[13px] font-bold" style={{ background: view === 'library' ? 'var(--surface)' : 'transparent', border: view === 'library' ? `1px solid ${accent}` : '1px solid var(--line)', color: view === 'library' ? accent : 'var(--muted)' }}>Library</button>
          <button onClick={() => setView('new')} className="flex items-center gap-2 rounded-lg px-3.5 py-2 text-[13px] font-bold text-white" style={{ background: accent }}><Icon name="Plus" size={15} strokeWidth={2.5} /> New guide</button>
        </div>
      </div>

      {view === 'library' ? (
        <div className="flex-1 overflow-y-auto grid gap-3 content-start" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(330px, 1fr))' }}>
          {AUDIO_GUIDES.map((g) => (
            <div key={g.id} className="rounded-xl overflow-hidden" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
              <div className="h-24 relative" style={{ background: 'linear-gradient(135deg, #134e4a, #0c2b28)' }}>
                <Icon name="Landmark" size={66} style={{ position: 'absolute', right: -6, bottom: -10, color: 'rgba(45,212,191,0.18)' }} />
                <div className="absolute top-3 left-4 right-4 flex items-start justify-between">
                  <div>
                    <div className="font-bold text-[15px] text-white">{g.site}</div>
                    <div className="font-mono text-[11px] flex items-center gap-1 mt-0.5" style={{ color: 'rgba(255,255,255,0.6)' }}>
                      <Icon name="MapPin" size={11} />{g.lat.toFixed(4)}, {g.lng.toFixed(4)}
                    </div>
                  </div>
                  <span className="rounded-full px-2 py-0.5 text-[10px] font-bold" style={{ background: g.published ? 'rgba(22,163,74,0.9)' : 'rgba(217,119,6,0.9)', color: '#fff' }}>
                    {g.published ? 'Published' : 'Draft'}
                  </span>
                </div>
              </div>
              <div className="p-4">
                <div className="flex items-center gap-2 flex-wrap">
                  {g.depths.map((d) => (
                    <span key={d} className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-[11.5px] font-bold" style={{ background: 'var(--bg)', color: 'var(--ink-soft)', border: '1px solid var(--line)' }}>
                      <Icon name={DEPTH_INFO[d].icon} size={11} style={{ color: accent }} />{d} · {g.duration[d]}
                    </span>
                  ))}
                </div>
                <div className="flex items-center justify-between mt-3 pt-3" style={{ borderTop: '1px solid var(--line)' }}>
                  <span className="flex items-center gap-1.5 text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>
                    <Icon name="Play" size={13} />{g.plays.toLocaleString()} plays · {g.lang}
                  </span>
                  <button className="text-[12.5px] font-bold flex items-center gap-1" style={{ color: accent }}>Edit <Icon name="ChevronRight" size={14} /></button>
                </div>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="flex-1 overflow-y-auto">
          <div className="grid gap-4" style={{ gridTemplateColumns: 'minmax(0, 1fr) 320px' }}>
            <div className="rounded-xl p-6 space-y-5" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
              {/* site + geo */}
              <div>
                <label className="block text-[12px] font-bold uppercase tracking-wide mb-1.5" style={{ color: 'var(--muted)' }}>Historical site</label>
                <input value={site} onChange={(e) => setSite(e.target.value)} placeholder="e.g. Galle Fort Ramparts"
                  className="w-full rounded-lg px-3.5 py-2.5 text-[14px] font-semibold outline-none" style={{ background: 'var(--bg)', border: '1px solid var(--line)', color: 'var(--ink)' }} />
              </div>
              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className="block text-[12px] font-bold uppercase tracking-wide mb-1.5" style={{ color: 'var(--muted)' }}>Region</label>
                  <input value={region} onChange={(e) => setRegion(e.target.value)} className="w-full rounded-lg px-3.5 py-2.5 text-[14px] font-semibold outline-none" style={{ background: 'var(--bg)', border: '1px solid var(--line)', color: 'var(--ink)' }} />
                </div>
                <div>
                  <label className="block text-[12px] font-bold uppercase tracking-wide mb-1.5" style={{ color: 'var(--muted)' }}>Latitude</label>
                  <input value={lat} onChange={(e) => setLat(e.target.value)} className="w-full rounded-lg px-3.5 py-2.5 text-[14px] font-mono font-semibold outline-none" style={{ background: 'var(--bg)', border: '1px solid var(--line)', color: 'var(--ink)' }} />
                </div>
                <div>
                  <label className="block text-[12px] font-bold uppercase tracking-wide mb-1.5" style={{ color: 'var(--muted)' }}>Longitude</label>
                  <input value={lng} onChange={(e) => setLng(e.target.value)} className="w-full rounded-lg px-3.5 py-2.5 text-[14px] font-mono font-semibold outline-none" style={{ background: 'var(--bg)', border: '1px solid var(--line)', color: 'var(--ink)' }} />
                </div>
              </div>

              {/* depths */}
              <div>
                <label className="block text-[12px] font-bold uppercase tracking-wide mb-1.5" style={{ color: 'var(--muted)' }}>Depth options · pick what to offer</label>
                <div className="grid grid-cols-3 gap-3">
                  {Object.keys(DEPTH_INFO).map((d) => (
                    <DepthChip key={d} depth={d} active={depths.includes(d)} onToggle={() => toggleDepth(d)} accent={accent} />
                  ))}
                </div>
              </div>

              {/* uploads per depth */}
              <div>
                <label className="block text-[12px] font-bold uppercase tracking-wide mb-1.5" style={{ color: 'var(--muted)' }}>Audio files</label>
                <div className="space-y-2">
                  {depths.length === 0 && <p className="text-[13px] font-medium" style={{ color: 'var(--muted)' }}>Select at least one depth above.</p>}
                  {depths.map((d) => (
                    <div key={d} className="flex items-center gap-3 rounded-lg px-3.5 py-2.5" style={{ background: 'var(--bg)', border: '1px solid var(--line)' }}>
                      <Icon name={window.DEPTH_INFO[d].icon} size={16} style={{ color: accent }} />
                      <span className="font-bold text-[13px] w-20" style={{ color: 'var(--ink)' }}>{d}</span>
                      {files[d] ? (
                        <span className="flex items-center gap-1.5 text-[12.5px] font-mono font-semibold" style={{ color: '#16a34a' }}><Icon name="CheckCircle2" size={14} />{files[d]}</span>
                      ) : (
                        <span className="text-[12.5px] font-medium" style={{ color: 'var(--muted)' }}>No file</span>
                      )}
                      <button onClick={() => fakeUpload(d)} className="ml-auto flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-[12px] font-bold" style={{ background: 'var(--surface)', border: '1px solid var(--line)', color: 'var(--ink)' }}>
                        <Icon name="Upload" size={13} /> {files[d] ? 'Replace' : 'Upload'}
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            {/* preview / publish */}
            <div className="space-y-4">
              <div className="rounded-xl overflow-hidden sticky top-0" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
                <div className="h-28 relative" style={{ background: 'linear-gradient(135deg, #134e4a, #0c2b28)' }}>
                  <Icon name="Landmark" size={72} style={{ position: 'absolute', right: -8, bottom: -12, color: 'rgba(45,212,191,0.18)' }} />
                  <div className="absolute inset-0 p-4 flex flex-col justify-end">
                    <div className="font-bold text-[15px] text-white">{site || 'New audio guide'}</div>
                    <div className="font-mono text-[11px] flex items-center gap-1" style={{ color: 'rgba(255,255,255,0.6)' }}><Icon name="MapPin" size={11} />{lat}, {lng}</div>
                  </div>
                </div>
                <div className="p-4">
                  <div className="text-[11px] font-bold uppercase tracking-wide mb-2" style={{ color: 'var(--muted)' }}>Traveler will see</div>
                  <div className="flex flex-wrap gap-1.5">
                    {depths.length ? depths.map((d) => (
                      <span key={d} className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-[11.5px] font-bold" style={{ background: 'var(--bg)', color: 'var(--ink-soft)', border: '1px solid var(--line)' }}>
                        <Icon name={window.DEPTH_INFO[d].icon} size={11} style={{ color: accent }} />{d}
                      </span>
                    )) : <span className="text-[12px] font-medium" style={{ color: 'var(--muted)' }}>No depths selected</span>}
                  </div>
                  <button onClick={() => { setPublished(true); setTimeout(() => setPublished(false), 2400); }}
                    disabled={!site || depths.length === 0}
                    className="mt-4 w-full rounded-lg py-3 font-bold text-[14px] flex items-center justify-center gap-2 text-white transition-all active:scale-[0.98]"
                    style={{ background: published ? '#16a34a' : (!site || !depths.length) ? 'var(--muted)' : accent, opacity: (!site || !depths.length) ? 0.5 : 1 }}>
                    {published ? <><Icon name="Check" size={17} strokeWidth={3} /> Published!</> : <><Icon name="Globe" size={16} /> Publish guide</>}
                  </button>
                  <p className="text-[11.5px] font-medium text-center mt-2" style={{ color: 'var(--muted)' }}>Goes live at the geotagged location</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

Object.assign(window, { AudioCMS });
