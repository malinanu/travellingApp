// Onboarding.jsx — step-through: welcome → travel style → interests.
// calls onDone({ style, interests })

const { useState } = React;

function StyleCard({ s, active, onClick }) {
  return (
    <button onClick={onClick}
      className="relative text-left rounded-3xl p-4 transition-all duration-200 active:scale-[0.98]"
      style={{
        background: active ? 'var(--accent)' : '#fff',
        boxShadow: active ? '0 12px 26px -10px var(--accent-shadow)' : '0 2px 10px rgba(15,61,58,0.06)',
        border: active ? '1.5px solid var(--accent)' : '1.5px solid rgba(15,61,58,0.07)',
      }}>
      <div className="flex items-center justify-center w-12 h-12 rounded-2xl mb-3"
           style={{ background: active ? 'rgba(255,255,255,0.22)' : 'var(--accent-tint)',
                    color: active ? '#fff' : 'var(--accent)' }}>
        <Icon name={s.icon} size={24} />
      </div>
      <div className="font-extrabold text-[17px] leading-tight"
           style={{ color: active ? '#fff' : 'var(--ink)' }}>{s.label}</div>
      <div className="text-[12.5px] mt-1 leading-snug"
           style={{ color: active ? 'rgba(255,255,255,0.85)' : 'var(--muted)' }}>{s.blurb}</div>
      {active && (
        <div className="absolute top-3 right-3 w-6 h-6 rounded-full bg-white flex items-center justify-center"
             style={{ color: 'var(--accent)' }}>
          <Icon name="Check" size={15} strokeWidth={3} />
        </div>
      )}
    </button>
  );
}

function InterestChip({ it, active, onClick }) {
  return (
    <button onClick={onClick}
      className="flex items-center gap-2 rounded-full pl-3 pr-4 py-2.5 transition-all duration-150 active:scale-95"
      style={{
        background: active ? 'var(--accent)' : '#fff',
        color: active ? '#fff' : 'var(--ink)',
        border: active ? '1.5px solid var(--accent)' : '1.5px solid rgba(15,61,58,0.10)',
        boxShadow: active ? '0 8px 18px -8px var(--accent-shadow)' : 'none',
      }}>
      <Icon name={it.icon} size={17} />
      <span className="font-bold text-[14px]">{it.label}</span>
    </button>
  );
}

function Onboarding({ onDone }) {
  const [step, setStep] = useState(0);
  const [styles, setStyles] = useState([]);
  const [interests, setInterests] = useState([]);

  const toggleStyle = (id) =>
    setStyles((p) => p.includes(id) ? p.filter((x) => x !== id) : [...p, id]);
  const toggle = (id) =>
    setInterests((p) => p.includes(id) ? p.filter((x) => x !== id) : [...p, id]);

  const canNext = step === 0 || (step === 1 && styles.length > 0) || (step === 2 && interests.length > 0);

  const next = () => {
    if (step < 2) setStep(step + 1);
    else onDone({ style: styles[0], styles, interests });
  };

  return (
    <div className="h-full flex flex-col" style={{ background: 'var(--sand)' }}>
      {/* top: brand + progress */}
      <div className="px-6" style={{ paddingTop: 64 }}>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <SpottoMark size={26} />
            <span className="font-extrabold text-[19px] tracking-tight" style={{ color: 'var(--ink)' }}>Spotto</span>
          </div>
          {step > 0 && (
            <button onClick={() => setStep(step - 1)}
              className="w-9 h-9 rounded-full flex items-center justify-center"
              style={{ background: '#fff', color: 'var(--ink)', boxShadow: '0 2px 8px rgba(15,61,58,0.08)' }}>
              <Icon name="ChevronLeft" size={20} />
            </button>
          )}
        </div>
        <div className="flex gap-1.5 mt-4">
          {[0, 1, 2].map((i) => (
            <div key={i} className="h-1.5 rounded-full transition-all duration-300"
              style={{ flex: i === step ? 2.2 : 1,
                       background: i <= step ? 'var(--accent)' : 'rgba(15,61,58,0.12)' }} />
          ))}
        </div>
      </div>

      {/* body */}
      <div className="flex-1 overflow-y-auto px-6 pt-6">
        {step === 0 && (
          <div className="h-full flex flex-col">
            <div className="rounded-[28px] overflow-hidden relative" style={{ height: 256, boxShadow: '0 20px 40px -18px rgba(15,61,58,0.4)' }}>
              <div style={{ position: 'absolute', inset: 0 }}>
                <SatelliteMap time="evening" />
              </div>
              <div className="absolute inset-0 flex flex-col justify-end p-5"
                   style={{ background: 'linear-gradient(180deg, rgba(0,0,0,0) 35%, rgba(10,30,40,0.55))' }}>
                <div className="inline-flex items-center gap-1.5 self-start rounded-full px-2.5 py-1 mb-2"
                     style={{ background: 'rgba(255,255,255,0.22)', backdropFilter: 'blur(6px)' }}>
                  <Icon name="MapPin" size={13} style={{ color: '#fff' }} />
                  <span className="text-white text-[12px] font-bold">Weligama, Sri Lanka</span>
                </div>
              </div>
            </div>
            <h1 className="font-extrabold mt-7 leading-[1.08]" style={{ fontSize: 34, color: 'var(--ink)' }}>
              Travel like you've<br/>got a friend here.
            </h1>
            <p className="mt-3 text-[15.5px] leading-relaxed" style={{ color: 'var(--muted)' }}>
              Spotto reads the bay like a local — the good coffee, the right break, the spot for sunset. Tell us your vibe and we'll do the rest.
            </p>
          </div>
        )}

        {step === 1 && (
          <div>
            <h2 className="font-extrabold leading-tight" style={{ fontSize: 27, color: 'var(--ink)' }}>What's your travel style?</h2>
            <p className="mt-2 text-[14.5px]" style={{ color: 'var(--muted)' }}>We'll tune every recommendation around it. Pick all that fit.</p>
            <div className="grid grid-cols-2 gap-3 mt-5">
              {window.TRAVEL_STYLES.map((s) => (
                <StyleCard key={s.id} s={s} active={styles.includes(s.id)} onClick={() => toggleStyle(s.id)} />
              ))}
            </div>
          </div>
        )}

        {step === 2 && (
          <div>
            <h2 className="font-extrabold leading-tight" style={{ fontSize: 27, color: 'var(--ink)' }}>What are you into?</h2>
            <p className="mt-2 text-[14.5px]" style={{ color: 'var(--muted)' }}>Choose a few — the map fills in around these.</p>
            <div className="flex flex-wrap gap-2.5 mt-5">
              {window.INTERESTS.map((it) => (
                <InterestChip key={it.id} it={it} active={interests.includes(it.id)} onClick={() => toggle(it.id)} />
              ))}
            </div>
          </div>
        )}
      </div>

      {/* footer cta */}
      <div className="px-6 pb-10 pt-3">
        <button onClick={next} disabled={!canNext}
          className="w-full rounded-2xl py-4 font-extrabold text-[16px] flex items-center justify-center gap-2 transition-all duration-200 active:scale-[0.98]"
          style={{
            background: canNext ? 'var(--accent)' : 'rgba(15,61,58,0.10)',
            color: canNext ? '#fff' : 'rgba(15,61,58,0.35)',
            boxShadow: canNext ? '0 14px 28px -12px var(--accent-shadow)' : 'none',
          }}>
          {step === 0 ? 'Get started' : step === 2 ? 'Explore Weligama' : 'Continue'}
          <Icon name="ArrowRight" size={19} strokeWidth={2.5} />
        </button>
      </div>
    </div>
  );
}

// little brand mark — sun + pin
function SpottoMark({ size = 26 }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" fill="none">
      <path d="M16 3C10.5 3 6 7.3 6 12.7 6 19.5 16 29 16 29s10-9.5 10-16.3C26 7.3 21.5 3 16 3z"
            fill="var(--accent)" />
      <circle cx="16" cy="12.5" r="4.4" fill="var(--sun)" />
    </svg>
  );
}

Object.assign(window, { Onboarding, SpottoMark });
