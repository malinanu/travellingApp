// QRValidator.jsx — simulated code scanner + manual entry + result card.

const { useState: useStateQR, useRef: useRefQR } = React;

// fake QR glyph
function QRGlyph({ size = 132, color = '#103d3a' }) {
  const cells = [
    '1111111 0 1011101','1000001 0 1110001','1011101 0 0010101','1011101 1 1101101',
    '1011101 0 0011001','1000001 1 1010101','1111111 0 1010101','0000000 0 0000000',
    '1101011 1 0110111','0010100 0 1101001','1110111 1 0011101','0100010 0 1110011',
    '1011101 1 0101001','0000110 0 1011011','1111111 1 0110101','1000001 0 1010011','1011101 1 0011101',
  ].map((r) => r.replace(/ /g, ''));
  const n = cells.length;
  const s = size / n;
  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
      {cells.map((row, y) => row.split('').map((c, x) => c === '1' ? (
        <rect key={`${x}-${y}`} x={x * s} y={y * s} width={s} height={s} fill={color} rx={s * 0.18} />
      ) : null))}
    </svg>
  );
}

function QRValidator({ accent }) {
  const { VALID_CODES } = window;
  const [phase, setPhase] = useStateQR('idle'); // idle | scanning | result
  const [result, setResult] = useStateQR(null);  // {ok, code, data}
  const [manual, setManual] = useStateQR('');
  const [redeemed, setRedeemed] = useStateQR(false);

  function validate(code) {
    const norm = code.trim().toUpperCase();
    const data = VALID_CODES[norm];
    setResult({ ok: !!data, code: norm, data });
    setPhase('result');
    setRedeemed(false);
  }

  function simulateScan() {
    setPhase('scanning');
    const codes = Object.keys(VALID_CODES);
    // 80% chance valid
    const pick = Math.random() < 0.8 ? codes[Math.floor(Math.random() * codes.length)] : 'SPOTTO-BADX';
    setTimeout(() => validate(pick), 1900);
  }

  function reset() { setPhase('idle'); setResult(null); setManual(''); setRedeemed(false); }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="font-extrabold tracking-tight" style={{ fontSize: 28, color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Validate a redemption</h1>
        <p className="text-[14.5px] font-semibold mt-1" style={{ color: 'var(--muted)' }}>Scan the traveler's Spotto code, or type it in. Confirm in one tap.</p>
      </div>

      <div className="grid gap-6" style={{ gridTemplateColumns: 'minmax(0, 1fr) minmax(0, 380px)' }}>
        {/* scanner */}
        <div className="rounded-2xl bg-white overflow-hidden" style={{ border: '1px solid var(--line)' }}>
          <div className="relative flex items-center justify-center"
               style={{ aspectRatio: '4 / 3', background: 'radial-gradient(120% 100% at 50% 0%, #14323a, #081c20)' }}>
            {phase === 'result' && result ? (
              <ResultCard result={result} accent={accent} redeemed={redeemed} setRedeemed={setRedeemed} onReset={reset} />
            ) : (
              <>
                {/* camera reticle */}
                <div className="relative" style={{ width: 220, height: 220 }}>
                  {phase === 'scanning' && (
                    <div className="rounded-2xl" style={{ position: 'absolute', inset: 18, background: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                      <QRGlyph size={150} />
                    </div>
                  )}
                  {['tl','tr','bl','br'].map((cnr) => (
                    <span key={cnr} style={{
                      position: 'absolute', width: 40, height: 40, borderColor: accent, borderStyle: 'solid',
                      borderWidth: 0,
                      ...(cnr.includes('t') ? { top: 0, borderTopWidth: 5 } : { bottom: 0, borderBottomWidth: 5 }),
                      ...(cnr.includes('l') ? { left: 0, borderLeftWidth: 5 } : { right: 0, borderRightWidth: 5 }),
                      borderTopLeftRadius: cnr === 'tl' ? 14 : 0, borderTopRightRadius: cnr === 'tr' ? 14 : 0,
                      borderBottomLeftRadius: cnr === 'bl' ? 14 : 0, borderBottomRightRadius: cnr === 'br' ? 14 : 0,
                    }} />
                  ))}
                  {phase === 'scanning' && (
                    <div style={{ position: 'absolute', left: 8, right: 8, height: 3, borderRadius: 3, background: accent,
                                  boxShadow: `0 0 14px 2px ${accent}`, animation: 'scanLine 1.6s ease-in-out infinite' }} />
                  )}
                  {phase === 'idle' && (
                    <div className="absolute inset-0 flex flex-col items-center justify-center gap-2" style={{ color: 'rgba(255,255,255,0.6)' }}>
                      <Icon name="QrCode" size={52} />
                      <span className="text-[13px] font-bold">Point at traveler's code</span>
                    </div>
                  )}
                </div>
                {/* status pill */}
                <div className="absolute bottom-4 left-1/2 -translate-x-1/2">
                  <span className="rounded-full px-3.5 py-1.5 text-[12.5px] font-bold flex items-center gap-2"
                        style={{ background: 'rgba(255,255,255,0.12)', color: '#fff', backdropFilter: 'blur(6px)' }}>
                    <span className="w-2 h-2 rounded-full" style={{ background: phase === 'scanning' ? '#fbbf24' : '#16a34a' }} />
                    {phase === 'scanning' ? 'Scanning…' : 'Camera ready'}
                  </span>
                </div>
              </>
            )}
          </div>
          {phase !== 'result' && (
            <div className="p-4">
              <button onClick={simulateScan} disabled={phase === 'scanning'}
                className="w-full rounded-xl py-3.5 font-extrabold text-[15px] flex items-center justify-center gap-2 text-white transition-all active:scale-[0.98]"
                style={{ background: accent, opacity: phase === 'scanning' ? 0.6 : 1, boxShadow: '0 10px 22px -10px var(--accent-shadow)' }}>
                <Icon name="ScanLine" size={19} /> {phase === 'scanning' ? 'Scanning…' : 'Simulate scan'}
              </button>
            </div>
          )}
        </div>

        {/* manual + tips */}
        <div className="space-y-4">
          <div className="rounded-2xl p-5 bg-white" style={{ border: '1px solid var(--line)' }}>
            <h3 className="font-extrabold text-[15px]" style={{ color: 'var(--ink)' }}>Enter code manually</h3>
            <p className="text-[12.5px] font-semibold mt-0.5 mb-3" style={{ color: 'var(--muted)' }}>If the camera won't focus.</p>
            <div className="flex items-center gap-2">
              <input value={manual} onChange={(e) => setManual(e.target.value.toUpperCase())}
                     onKeyDown={(e) => e.key === 'Enter' && manual.trim() && validate(manual)}
                     placeholder="SPOTTO-XXXX"
                     className="flex-1 rounded-xl px-3.5 py-3 text-[14.5px] font-mono font-bold tracking-wider outline-none"
                     style={{ background: '#fff', border: '1.5px solid var(--line)', color: 'var(--ink)' }} />
              <button onClick={() => manual.trim() && validate(manual)}
                className="rounded-xl px-4 py-3 font-extrabold text-[14px] text-white flex-shrink-0 active:scale-95 transition-transform"
                style={{ background: 'var(--ink)' }}>Check</button>
            </div>
            <div className="mt-3 flex flex-wrap gap-1.5">
              <span className="text-[11.5px] font-semibold w-full" style={{ color: 'var(--muted)' }}>Try a demo code:</span>
              {Object.keys(VALID_CODES).map((c) => (
                <button key={c} onClick={() => { setManual(c); validate(c); }}
                  className="rounded-lg px-2 py-1 text-[11.5px] font-mono font-bold"
                  style={{ background: 'var(--sand)', color: 'var(--ink-soft)' }}>{c}</button>
              ))}
            </div>
          </div>

          <div className="rounded-2xl p-5" style={{ background: 'var(--accent-tint)' }}>
            <div className="flex items-center gap-2 font-extrabold text-[14px]" style={{ color: accent }}>
              <Icon name="ShieldCheck" size={18} /> Fraud-safe
            </div>
            <p className="text-[13px] font-semibold mt-1.5 leading-relaxed" style={{ color: 'var(--ink-soft)' }}>
              Each code can only be redeemed once. Spotto auto-expires it the moment you confirm.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

function ResultCard({ result, accent, redeemed, setRedeemed, onReset }) {
  const ok = result.ok;
  return (
    <div className="absolute inset-0 flex items-center justify-center p-5">
      <div className="w-full max-w-[330px] rounded-2xl bg-white p-5 text-center" style={{ boxShadow: '0 20px 50px -16px rgba(0,0,0,0.5)' }}>
        <div className="w-16 h-16 rounded-full mx-auto flex items-center justify-center"
             style={{ background: ok ? '#e7f6ec' : '#fde8e4', color: ok ? '#16a34a' : '#dc4a2d' }}>
          <Icon name={ok ? (redeemed ? 'BadgeCheck' : 'CheckCircle2') : 'XCircle'} size={36} />
        </div>
        {ok ? (
          <>
            <div className="font-extrabold text-[19px] mt-3" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>
              {redeemed ? 'Redeemed!' : 'Valid code'}
            </div>
            <div className="font-mono font-extrabold tracking-widest text-[13px] mt-0.5" style={{ color: accent }}>{result.code}</div>
            <div className="rounded-xl p-3 mt-3 text-left" style={{ background: 'var(--sand)' }}>
              <Row label="Traveler" value={`${result.data.name} · ${result.data.country}`} />
              <Row label="Style" value={result.data.style} />
              <Row label="Offer" value={result.data.offer} />
              <Row label="Value" value={result.data.value} bold accent={accent} />
            </div>
            {redeemed ? (
              <button onClick={onReset} className="mt-4 w-full rounded-xl py-3 font-extrabold text-[14.5px]"
                style={{ background: 'var(--sand)', color: 'var(--ink)' }}>Scan next traveler</button>
            ) : (
              <button onClick={() => setRedeemed(true)} className="mt-4 w-full rounded-xl py-3 font-extrabold text-[14.5px] text-white active:scale-[0.98] transition-transform"
                style={{ background: accent }}>Confirm redemption</button>
            )}
          </>
        ) : (
          <>
            <div className="font-extrabold text-[19px] mt-3" style={{ color: 'var(--ink)', fontFamily: 'var(--font-display)' }}>Invalid code</div>
            <p className="text-[13.5px] font-semibold mt-1 px-2" style={{ color: 'var(--muted)' }}>
              <span className="font-mono">{result.code}</span> isn't recognised, or it's already been used.
            </p>
            <button onClick={onReset} className="mt-4 w-full rounded-xl py-3 font-extrabold text-[14.5px] text-white" style={{ background: 'var(--ink)' }}>Try again</button>
          </>
        )}
      </div>
    </div>
  );
}

function Row({ label, value, bold, accent }) {
  return (
    <div className="flex items-baseline justify-between gap-3 py-1">
      <span className="text-[12px] font-bold flex-shrink-0" style={{ color: 'var(--muted)' }}>{label}</span>
      <span className="text-[12.5px] font-extrabold text-right" style={{ color: bold ? accent : 'var(--ink)' }}>{value}</span>
    </div>
  );
}

Object.assign(window, { QRValidator });
