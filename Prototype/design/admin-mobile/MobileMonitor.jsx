// MobileMonitor.jsx — live content feed with filter chips, pause, inline moderation.

const { useState: useStateMM, useEffect: useEffectMM, useRef: useRefMM } = React;

const ST_M = {
  ok:     { color: '#16a34a', dot: '#16a34a', icon: 'CheckCircle2' },
  review: { color: '#d97706', dot: '#d97706', icon: 'Eye' },
  flag:   { color: '#e11d48', dot: '#e11d48', icon: 'Flag' },
};

let _mid = 2000;

function FeedCardM({ item, fresh, onAction }) {
  const s = ST_M[item.status];
  return (
    <div className="rounded-2xl" style={{ padding: 'var(--pad)', background: 'var(--surface)', border: '1px solid var(--line)',
         animation: fresh ? 'mFresh 1.3s ease' : 'none' }}>
      <div className="flex items-center gap-2 flex-wrap">
        <span className="w-2 h-2 rounded-full" style={{ background: s.dot }} />
        <span className="rounded px-1.5 py-0.5 text-[9.5px] font-bold uppercase tracking-wide" style={{ background: 'var(--bg)', color: 'var(--muted)', border: '1px solid var(--line)' }}>{item.type}</span>
        <span className="font-bold text-[13.5px]" style={{ color: 'var(--ink)' }}>{item.vendor}</span>
        <span className="ml-auto text-[11px] font-mono" style={{ color: 'var(--muted)' }}>{item.t}</span>
      </div>
      <div className="flex items-center gap-1 text-[11.5px] font-medium mt-1" style={{ color: 'var(--muted)' }}>
        <Icon name="MapPin" size={11} />{item.town}
      </div>
      <div className="text-[14px] font-medium mt-1.5 leading-snug" style={{ color: 'var(--ink-soft)' }}>"{item.text}"</div>
      {item.reason && (
        <div className="flex items-center gap-1.5 mt-2 text-[12px] font-semibold" style={{ color: s.color }}>
          <Icon name={s.icon} size={13} /> {item.reason}
        </div>
      )}
      {item.status !== 'ok' && !item.resolved && (
        <div className="flex items-center gap-2 mt-3">
          <button onClick={() => onAction(item.id, 'approved')} className="flex-1 rounded-lg py-2 text-[12.5px] font-bold" style={{ background: 'rgba(22,163,74,0.1)', color: '#16a34a' }}>Approve</button>
          <button onClick={() => onAction(item.id, 'removed')} className="flex-1 rounded-lg py-2 text-[12.5px] font-bold" style={{ background: 'rgba(225,29,72,0.1)', color: '#e11d48' }}>Remove</button>
          <button onClick={() => onAction(item.id, 'warned')} className="flex-1 rounded-lg py-2 text-[12.5px] font-bold" style={{ background: 'var(--bg)', color: 'var(--ink-soft)', border: '1px solid var(--line)' }}>Warn</button>
        </div>
      )}
      {item.resolved && (
        <div className="mt-2.5 text-[12px] font-bold" style={{ color: item.resolved === 'removed' ? '#e11d48' : item.resolved === 'warned' ? '#d97706' : '#16a34a' }}>
          ✓ {item.resolved === 'removed' ? 'Content removed' : item.resolved === 'warned' ? 'Vendor warned' : 'Approved'}
        </div>
      )}
    </div>
  );
}

function MobileMonitor({ accent }) {
  const { FEED_SEED, FEED_TEMPLATES } = window;
  const [feed, setFeed] = useStateMM(() => FEED_SEED.map((x) => ({ ...x })));
  const [paused, setPaused] = useStateMM(false);
  const [filter, setFilter] = useStateMM('all');
  const [freshId, setFreshId] = useStateMM(null);
  const pausedRef = useRefMM(paused);
  pausedRef.current = paused;

  useEffectMM(() => {
    const iv = setInterval(() => {
      if (pausedRef.current) return;
      const tpl = FEED_TEMPLATES[Math.floor(Math.random() * FEED_TEMPLATES.length)];
      const id = 'M' + (++_mid);
      setFeed((f) => [{ ...tpl, id, t: 'now' }, ...f.map((x) => ({ ...x, t: bumpTimeM(x.t) }))].slice(0, 40));
      setFreshId(id);
      setTimeout(() => setFreshId(null), 1300);
    }, 3200);
    return () => clearInterval(iv);
  }, []);

  function action(id, resolved) { setFeed((f) => f.map((x) => x.id === id ? { ...x, resolved } : x)); }

  const counts = {
    all: feed.length,
    flag: feed.filter((x) => x.status === 'flag' && !x.resolved).length,
    review: feed.filter((x) => x.status === 'review' && !x.resolved).length,
  };
  const shown = feed.filter((x) => filter === 'all' ? true : x.status === filter);

  return (
    <div className="h-full flex flex-col">
      <div className="flex-shrink-0" style={{ padding: 'var(--pad)', paddingBottom: 10 }}>
        <div className="flex items-start gap-3">
          <div className="flex-1">
            <h1 className="font-bold tracking-tight" style={{ fontSize: 24, color: 'var(--ink)' }}>Live monitor</h1>
            <div className="flex items-center gap-1.5 text-[12.5px] font-bold mt-0.5" style={{ color: paused ? 'var(--muted)' : '#16a34a' }}>
              <span className="w-2 h-2 rounded-full" style={{ background: paused ? 'var(--muted)' : '#16a34a', boxShadow: paused ? 'none' : '0 0 0 3px rgba(22,163,74,0.18)' }} />
              {paused ? 'Stream paused' : 'Streaming live'}
            </div>
          </div>
          <button onClick={() => setPaused(!paused)}
            className="w-11 h-11 rounded-full flex items-center justify-center active:scale-90 transition-transform"
            style={{ background: paused ? accent : 'var(--surface)', color: paused ? '#fff' : 'var(--ink)', border: '1px solid var(--line)' }}>
            <Icon name={paused ? 'Play' : 'Pause'} size={19} />
          </button>
        </div>

        {/* filter chips */}
        <div className="flex items-center gap-2 mt-3 overflow-x-auto noscroll">
          {[['all', 'All'], ['flag', 'Flagged'], ['review', 'Needs review']].map(([k, lbl]) => {
            const on = filter === k;
            const c = k === 'flag' ? '#e11d48' : k === 'review' ? '#d97706' : accent;
            return (
              <button key={k} onClick={() => setFilter(k)}
                className="flex items-center gap-1.5 rounded-full px-3 py-1.5 text-[12.5px] font-bold flex-shrink-0 transition-colors"
                style={{ background: on ? c : 'var(--surface)', border: on ? `1px solid ${c}` : '1px solid var(--line)', color: on ? '#fff' : 'var(--muted)' }}>
                {lbl}
                <span className="rounded-full px-1.5 text-[10px] font-bold" style={{ background: on ? 'rgba(255,255,255,0.25)' : 'var(--bg)', color: on ? '#fff' : 'var(--muted)' }}>{counts[k] ?? 0}</span>
              </button>
            );
          })}
        </div>
      </div>

      {/* feed */}
      <div className="flex-1 overflow-y-auto noscroll flex flex-col" style={{ gap: 'var(--gap)', padding: '0 var(--pad) var(--pad)' }}>
        {shown.map((item) => <FeedCardM key={item.id} item={item} fresh={item.id === freshId} onAction={action} />)}
      </div>
    </div>
  );
}

function bumpTimeM(t) {
  if (t === 'now') return '1m';
  const m = parseInt(t);
  return isNaN(m) ? t : (m + 1) + 'm';
}

Object.assign(window, { MobileMonitor });
