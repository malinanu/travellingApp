// SpotsList.jsx — business listings as scrollable cards.

function SpotCard({ p, onClick }) {
  return (
    <button onClick={onClick}
      className="w-full text-left rounded-3xl overflow-hidden active:scale-[0.99] transition-transform"
      style={{ background: '#fff', boxShadow: '0 8px 24px -14px rgba(16,61,58,0.25)' }}>
      <div style={{ position: 'relative', height: 116, background: 'var(--sand)' }}>
        <Illustration kind={p.photo} />
        <div className="absolute top-3 left-3 inline-flex items-center gap-1 rounded-full px-2.5 py-1"
             style={{ background: 'rgba(255,255,255,0.92)', color: 'var(--ink)' }}>
          <Icon name="Star" size={12} style={{ fill: 'var(--sun)', color: 'var(--sun)' }} />
          <span className="font-extrabold text-[12px]">{p.rating}</span>
        </div>
        {p.deal?.active && (
          <div className="absolute top-3 right-3 inline-flex items-center gap-1 rounded-full px-2.5 py-1"
               style={{ background: 'var(--sun)', color: '#5a3b00' }}>
            <Icon name="Tag" size={12} strokeWidth={2.5} />
            <span className="font-extrabold text-[11.5px]">{p.deal.tag}</span>
          </div>
        )}
        <div className="absolute bottom-2.5 left-3 right-3">
          <div className="text-white font-extrabold text-[16px] leading-tight" style={{ textShadow: '0 1px 8px rgba(0,0,0,0.35)' }}>{p.name}</div>
        </div>
      </div>
      <div className="px-3.5 py-3 flex items-center gap-2.5">
        <span className="text-[12.5px] font-bold rounded-full px-2.5 py-1" style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}>{p.type}</span>
        <span className="flex items-center gap-1 text-[12.5px] font-bold" style={{ color: p.wifi.score >= 8 ? 'var(--accent)' : 'var(--muted)' }}>
          <Icon name="Wifi" size={14} />{p.wifi.score}
        </span>
        <span className="flex items-center gap-1 text-[12.5px] font-semibold ml-auto" style={{ color: p.openNow ? 'var(--accent)' : 'var(--muted)' }}>
          <Icon name="Clock" size={13} />{p.openNow ? 'Open' : 'Closed'}
        </span>
        <span className="flex items-center gap-1 text-[12.5px] font-semibold" style={{ color: 'var(--muted)' }}>
          <Icon name="Navigation" size={13} />{p.distance}
        </span>
      </div>
    </button>
  );
}

function SpotsList({ places, onSelect, styleLabel }) {
  return (
    <div className="h-full overflow-y-auto" style={{ background: 'var(--sand)' }}>
      <div className="px-5 pb-3" style={{ paddingTop: 64 }}>
        <div className="text-[12.5px] font-bold uppercase tracking-wide" style={{ color: 'var(--accent)' }}>Curated for {styleLabel}</div>
        <h1 className="font-extrabold leading-tight mt-1" style={{ fontSize: 30, color: 'var(--ink)' }}>Spots in the bay</h1>
        <p className="text-[14px] mt-1 font-semibold" style={{ color: 'var(--muted)' }}>{places.length} places worth your time today</p>
      </div>
      <div className="px-4 pb-28 space-y-3.5">
        {places.map((p) => <SpotCard key={p.id} p={p} onClick={() => onSelect(p)} />)}
      </div>
    </div>
  );
}

Object.assign(window, { SpotsList });
