// AIGuide.jsx — live Claude chat, grounded in the user's prefs + Weligama places.

const { useState: useStateAI, useRef: useRefAI, useEffect: useEffectAI } = React;

function buildSystem(style, interestLabels, weather) {
  const placeLines = window.PLACES.map((p) =>
    `- ${p.name} (${p.type}). Wi-Fi ${p.wifi.score}/10 (${p.wifi.speed}). Hours ${p.hours}. ` +
    `${p.seaView ? 'Has sea view. ' : ''}${p.indoor ? 'INDOOR (great in rain). ' : 'Mostly outdoor. '}` +
    `Best in the ${p.bestTimes.join(' & ')}. ` +
    `Deal: ${p.deal?.active ? p.deal.label : 'none'}. ${p.blurb}`
  ).join('\n');
  const wx = window.WEATHER[weather] || window.WEATHER.sunny;
  const weatherRule = weather === 'rainy'
    ? `IT IS CURRENTLY RAINING in Weligama. Steer travellers toward INDOOR options (e.g. the Clay & Co Pottery Studio workshop or Sea Salt Batik Atelier) and AVOID recommending the surf lesson or open rooftop right now — mention the rain naturally.`
    : `Weather is ${wx.label.toLowerCase()} (${wx.temp}) — outdoor activities like surf and sunset rooftops are great.`;
  return (
    `You are Spotto, a warm, switched-on LOCAL FRIEND showing a traveller around Weligama, Sri Lanka. ` +
    `Talk like a real local mate, not a brochure — friendly, concise, specific.\n\n` +
    `The traveller's style is "${style}". Their interests: ${interestLabels.join(', ') || 'unspecified'}. ` +
    `Always tailor picks to this (e.g. a Digital Nomad cares about Wi-Fi, power, focus & sea views).\n\n` +
    `CURRENT WEATHER: ${weatherRule}\n\n` +
    `These are the ONLY real venues you can recommend by name — use their exact names:\n${placeLines}\n\n` +
    `Rules: Keep replies to 2–4 short sentences. Recommend by exact venue name when relevant. ` +
    `Mention the concrete detail that matters (Wi-Fi speed, the live deal, sunset timing, or the rain). ` +
    `If asked something off-map, be honest and still helpful. No markdown headings, no bullet lists longer than 3 items.`
  );
}

const SUGGEST = [
  'Where can I work today with a sea view?',
  'Best surf spot for a total beginner?',
  "Where's the move for sunset tonight?",
  'Any deals on right now?',
];

function Typing() {
  return (
    <div className="flex gap-1 items-center px-1">
      {[0, 1, 2].map((i) => (
        <span key={i} style={{
          width: 7, height: 7, borderRadius: '50%', background: 'var(--accent)',
          animation: `spottoBounce 1.1s ${i * 0.15}s infinite ease-in-out`,
        }} />
      ))}
    </div>
  );
}

function findPlace(text) {
  if (!text) return null;
  const low = text.toLowerCase();
  return window.PLACES.find((p) =>
    low.includes(p.name.toLowerCase()) ||
    low.includes(p.name.split(' ')[0].toLowerCase() + ' ')
  );
}

function AIGuide({ style, styleLabel, interestLabels, weather, onOpenPlace, onClose }) {
  const isRainy = weather === 'rainy';
  const greeting = isRainy
    ? `Hey! It's raining over Weligama right now 🌧️ — I've bumped indoor spots to the top. Ask me where to go and I'll keep you dry (and caffeinated).`
    : `Hey! I'm your Spotto guide for Weligama 🌴 I've set things up for a ${styleLabel.toLowerCase()} — ask me where to work, surf, eat or catch the sunset and I'll point you to the right spot.`;
  const suggestions = isRainy
    ? ['Where can I work today with a sea view?', "What's good to do in the rain?", 'Any indoor workshops nearby?', 'Any deals on right now?']
    : SUGGEST;
  const [msgs, setMsgs] = useStateAI([{ role: 'assistant', content: greeting }]);
  const [input, setInput] = useStateAI('');
  const [busy, setBusy] = useStateAI(false);
  const scroller = useRefAI(null);

  useEffectAI(() => {
    const el = scroller.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [msgs, busy]);

  async function send(text) {
    const q = (text ?? input).trim();
    if (!q || busy) return;
    const next = [...msgs, { role: 'user', content: q }];
    setMsgs(next);
    setInput('');
    setBusy(true);
    try {
      const sys = buildSystem(styleLabel, interestLabels, weather);
      const convo = next.map((m) => ({ role: m.role, content: m.content }));
      const messages = [
        { role: 'user', content: sys },
        { role: 'assistant', content: "Got it — I'm Spotto, ready to help. What's up?" },
        ...convo,
      ];
      const reply = await window.claude.complete({ messages });
      setMsgs((m) => [...m, { role: 'assistant', content: (reply || '').trim() || "Hmm, I blanked for a sec — try asking again?" }]);
    } catch (e) {
      setMsgs((m) => [...m, { role: 'assistant', content: "I can't reach the network right now, but a Digital-Nomad tip: Driftwood Coffee & Co-Work has the bay's fastest Wi-Fi (9.2/10) and sea-view desks." }]);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="h-full flex flex-col" style={{ background: 'var(--sand)' }}>
      {/* header */}
      <div className="px-5 pb-3 flex items-center gap-3" style={{ paddingTop: 62, background: '#fff', boxShadow: '0 2px 16px -8px rgba(16,61,58,0.25)' }}>
        <div className="w-11 h-11 rounded-2xl flex items-center justify-center flex-shrink-0"
             style={{ background: 'linear-gradient(135deg, var(--accent), var(--accent-2))', color: '#fff' }}>
          <Icon name="Sparkles" size={22} />
        </div>
        <div className="leading-tight">
          <div className="font-extrabold text-[17px]" style={{ color: 'var(--ink)' }}>Local Guide</div>
          <div className="text-[12px] font-semibold flex items-center gap-1.5" style={{ color: 'var(--muted)' }}>
            <span style={{ width: 7, height: 7, borderRadius: '50%', background: '#16a34a', display: 'inline-block' }} />
            Tuned for {styleLabel}
          </div>
        </div>
        {onClose && (
          <button onClick={onClose} aria-label="Close guide"
            className="ml-auto w-9 h-9 rounded-full flex items-center justify-center flex-shrink-0 active:scale-90 transition-transform"
            style={{ background: 'var(--sand)', color: 'var(--ink)' }}>
            <Icon name="X" size={19} strokeWidth={2.5} />
          </button>
        )}
      </div>

      {/* messages */}
      <div ref={scroller} className="flex-1 overflow-y-auto px-4 py-4 space-y-3" style={{ WebkitOverflowScrolling: 'touch' }}>
        {msgs.map((m, i) => {
          const mine = m.role === 'user';
          const place = !mine ? findPlace(m.content) : null;
          return (
            <div key={i} className={`flex ${mine ? 'justify-end' : 'justify-start'}`}>
              <div style={{ maxWidth: '84%' }}>
                <div className="px-3.5 py-2.5 text-[14.5px] leading-relaxed"
                  style={{
                    background: mine ? 'var(--accent)' : '#fff',
                    color: mine ? '#fff' : 'var(--ink)',
                    borderRadius: mine ? '20px 20px 6px 20px' : '20px 20px 20px 6px',
                    boxShadow: mine ? '0 8px 18px -10px var(--accent-shadow)' : '0 2px 12px -4px rgba(16,61,58,0.12)',
                    fontWeight: 500,
                  }}>
                  {m.content}
                </div>
                {place && (
                  <button onClick={() => onOpenPlace(place)}
                    className="mt-1.5 flex items-center gap-2 rounded-2xl px-2.5 py-2 active:scale-[0.98] transition-transform"
                    style={{ background: '#fff', boxShadow: '0 2px 12px -4px rgba(16,61,58,0.18)' }}>
                    <span className="w-8 h-8 rounded-xl flex items-center justify-center flex-shrink-0"
                          style={{ background: 'var(--accent-tint)', color: 'var(--accent)' }}>
                      <Icon name={place.icon} size={16} />
                    </span>
                    <span className="text-left">
                      <span className="block font-extrabold text-[12.5px]" style={{ color: 'var(--ink)' }}>{place.name}</span>
                      <span className="block text-[11px] font-semibold" style={{ color: 'var(--accent)' }}>View on map →</span>
                    </span>
                  </button>
                )}
              </div>
            </div>
          );
        })}
        {busy && (
          <div className="flex justify-start">
            <div className="px-3.5 py-3 rounded-[20px_20px_20px_6px]" style={{ background: '#fff', boxShadow: '0 2px 12px -4px rgba(16,61,58,0.12)' }}>
              <Typing />
            </div>
          </div>
        )}
      </div>

      {/* suggestions */}
      {msgs.length <= 1 && (
        <div className="px-4 pb-2 flex flex-wrap gap-2">
          {suggestions.map((s) => (
            <button key={s} onClick={() => send(s)}
              className="rounded-full px-3 py-2 text-[12.5px] font-bold text-left active:scale-95 transition-transform"
              style={{ background: '#fff', color: 'var(--ink)', border: '1.5px solid rgba(15,61,58,0.1)' }}>
              {s}
            </button>
          ))}
        </div>
      )}

      {/* composer */}
      <div className="px-4 pt-2 pb-8" style={{ background: 'var(--sand)' }}>
        <div className="flex items-end gap-2 rounded-3xl p-1.5" style={{ background: '#fff', boxShadow: '0 4px 18px -8px rgba(16,61,58,0.2)' }}>
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send(); } }}
            rows={1} placeholder="Ask your local friend…"
            className="flex-1 resize-none bg-transparent px-3 py-2.5 text-[14.5px] outline-none"
            style={{ color: 'var(--ink)', maxHeight: 90, fontWeight: 500 }} />
          <button onClick={() => send()} disabled={!input.trim() || busy}
            className="w-11 h-11 rounded-full flex items-center justify-center flex-shrink-0 transition-all active:scale-90"
            style={{ background: input.trim() && !busy ? 'var(--accent)' : 'rgba(15,61,58,0.12)',
                     color: input.trim() && !busy ? '#fff' : 'rgba(15,61,58,0.4)' }}>
            <Icon name="Send" size={19} />
          </button>
        </div>
      </div>
    </div>
  );
}

Object.assign(window, { AIGuide });
