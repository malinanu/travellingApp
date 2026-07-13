// ContentMonitor.jsx — real-time feed of offers/events with moderation.

const { useState: useStateCM, useEffect: useEffectCM, useRef: useRefCM } = React;

const ST = {
  ok:     { label: 'Auto-approved', color: '#16a34a', bg: 'rgba(22,163,74,0.1)',  dot: '#16a34a', icon: 'CheckCircle2' },
  review: { label: 'Needs review',  color: '#d97706', bg: 'rgba(217,119,6,0.1)',  dot: '#d97706', icon: 'Eye' },
  flag:   { label: 'Flagged',       color: '#e11d48', bg: 'rgba(225,29,72,0.1)',  dot: '#e11d48', icon: 'Flag' },
};

let _cid = 1000;

function FeedItem({ item, fresh, onAction }) {
  const s = ST[item.status];
  return (
    <div className="px-4 py-3 flex items-start gap-3" style={{ borderBottom: '1px solid var(--line)',
         background: fresh ? 'rgba(11,111,134,0.05)' : 'transparent', transition: 'background 1.2s ease' }}>
      <div className="flex flex-col items-center pt-0.5">
        <span className="w-2 h-2 rounded-full" style={{ background: s.dot }} />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="rounded px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wide" style={{ background: 'var(--bg)', color: 'var(--muted)', border: '1px solid var(--line)' }}>{item.type}</span>
          <span className="font-bold text-[13px]" style={{ color: 'var(--ink)' }}>{item.vendor}</span>
          <span className="text-[12px] font-medium flex items-center gap-0.5" style={{ color: 'var(--muted)' }}><Icon name="MapPin" size={11} />{item.town}</span>
          <span className="ml-auto text-[11px] font-mono" style={{ color: 'var(--muted)' }}>{item.t}</span>
        </div>
        <div className="text-[13.5px] font-medium mt-1" style={{ color: 'var(--ink-soft)' }}>"{item.text}"</div>
        {item.reason && (
          <div className="flex items-center gap-1.5 mt-1.5 text-[12px] font-semibold" style={{ color: s.color }}>
            <Icon name={s.icon} size={13} /> {item.reason}
          </div>
        )}
        {item.status !== 'ok' && !item.resolved && (
          <div className="flex items-center gap-2 mt-2">
            <button onClick={() => onAction(item.id, 'approved')} className="rounded-md px-2.5 py-1 text-[12px] font-bold" style={{ background: 'rgba(22,163,74,0.1)', color: '#16a34a' }}>Approve</button>
            <button onClick={() => onAction(item.id, 'removed')} className="rounded-md px-2.5 py-1 text-[12px] font-bold" style={{ background: 'rgba(225,29,72,0.1)', color: '#e11d48' }}>Remove</button>
            <button onClick={() => onAction(item.id, 'warned')} className="rounded-md px-2.5 py-1 text-[12px] font-bold" style={{ background: 'var(--bg)', color: 'var(--ink-soft)', border: '1px solid var(--line)' }}>Warn vendor</button>
          </div>
        )}
        {item.resolved && (
          <div className="mt-2 text-[12px] font-bold" style={{ color: item.resolved === 'removed' ? '#e11d48' : item.resolved === 'warned' ? '#d97706' : '#16a34a' }}>
            ✓ {item.resolved === 'removed' ? 'Content removed' : item.resolved === 'warned' ? 'Vendor warned' : 'Approved'}
          </div>
        )}
      </div>
    </div>
  );
}

function ContentMonitor({ accent }) {
  const { FEED_SEED, FEED_TEMPLATES } = window;
  const [feed, setFeed] = useStateCM(() => FEED_SEED.map((x) => ({ ...x })));
  const [paused, setPaused] = useStateCM(false);
  const [filter, setFilter] = useStateCM('all');
  const [freshId, setFreshId] = useStateCM(null);
  const pausedRef = useRefCM(paused);
  pausedRef.current = paused;

  useEffectCM(() => {
    const iv = setInterval(() => {
      if (pausedRef.current) return;
      const tpl = FEED_TEMPLATES[Math.floor(Math.random() * FEED_TEMPLATES.length)];
      const id = 'C' + (++_cid);
      setFeed((f) => [{ ...tpl, id, t: 'now' }, ...f.map((x) => ({ ...x, t: bumpTime(x.t) }))].slice(0, 40));
      setFreshId(id);
      setTimeout(() => setFreshId(null), 1300);
    }, 3200);
    return () => clearInterval(iv);
  }, []);

  function action(id, resolved) {
    setFeed((f) => f.map((x) => x.id === id ? { ...x, resolved } : x));
  }

  const counts = {
    all: feed.length,
    flag: feed.filter((x) => x.status === 'flag' && !x.resolved).length,
    review: feed.filter((x) => x.status === 'review' && !x.resolved).length,
  };
  const shown = feed.filter((x) => filter === 'all' ? true : x.status === filter);

  return (
    <div className="h-full flex flex-col">
      <div className="flex items-end justify-between gap-4 mb-4">
        <div>
          <h1 className="font-bold tracking-tight" style={{ fontSize: 22, color: 'var(--ink)' }}>Live content monitor</h1>
          <p className="text-[13px] font-medium mt-0.5" style={{ color: 'var(--muted)' }}>All active offers & events across Sri Lanka · streaming</p>
        </div>
        <button onClick={() => setPaused(!paused)}
          className="flex items-center gap-2 rounded-lg px-3.5 py-2 font-bold text-[13px]"
          style={{ background: paused ? accent : 'var(--surface)', color: paused ? '#fff' : 'var(--ink)', border: '1px solid var(--line)' }}>
          <Icon name={paused ? 'Play' : 'Pause'} size={15} /> {paused ? 'Resume stream' : 'Pause stream'}
        </button>
      </div>

      {/* filter tabs */}
      <div className="flex items-center gap-2 mb-3">
        {[['all', 'All activity'], ['flag', 'Flagged'], ['review', 'Needs review']].map(([k, lbl]) => {
          const on = filter === k;
          const c = k === 'flag' ? '#e11d48' : k === 'review' ? '#d97706' : accent;
          return (
            <button key={k} onClick={() => setFilter(k)}
              className="flex items-center gap-2 rounded-lg px-3 py-1.5 text-[13px] font-bold transition-colors"
              style={{ background: on ? 'var(--surface)' : 'transparent', border: on ? `1px solid ${c}` : '1px solid var(--line)', color: on ? c : 'var(--muted)' }}>
              {lbl}
              <span className="rounded-full px-1.5 py-0.5 text-[10px] font-bold" style={{ background: on ? c : 'var(--bg)', color: on ? '#fff' : 'var(--muted)' }}>{counts[k] ?? 0}</span>
            </button>
          );
        })}
        <div className="ml-auto flex items-center gap-1.5 text-[12px] font-bold" style={{ color: paused ? 'var(--muted)' : '#16a34a' }}>
          <span className="w-2 h-2 rounded-full" style={{ background: paused ? 'var(--muted)' : '#16a34a', boxShadow: paused ? 'none' : '0 0 0 3px rgba(22,163,74,0.18)' }} />
          {paused ? 'Paused' : 'Live'}
        </div>
      </div>

      {/* feed */}
      <div className="flex-1 rounded-xl overflow-hidden flex flex-col min-h-0" style={{ background: 'var(--surface)', border: '1px solid var(--line)' }}>
        <div className="overflow-y-auto flex-1">
          {shown.map((item) => <FeedItem key={item.id} item={item} fresh={item.id === freshId} onAction={action} />)}
        </div>
      </div>
    </div>
  );
}

function bumpTime(t) {
  if (t === 'now') return '1m';
  const m = parseInt(t);
  return isNaN(m) ? t : (m + 1) + 'm';
}

Object.assign(window, { ContentMonitor });
