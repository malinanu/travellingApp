// App.jsx — Spotto shell: onboarding gate, tabs, scaling, tweaks, drawer,
// + Smart Companion state (weather, group mode, daily quest, vibe votes).

const { useState: useStateApp, useEffect: useEffectApp, useRef: useRefApp } = React;

const PALETTES = {
  '#0b6f86': { accent: '#0b6f86', accent2: '#0e8c8a', tint: '#ddedef', shadow: 'rgba(11,111,134,0.35)' },
  '#0e8c8a': { accent: '#0e8c8a', accent2: '#2ba69a', tint: '#dcefec', shadow: 'rgba(14,140,138,0.35)' },
  '#0b4a54': { accent: '#0b4a54', accent2: '#0b6f86', tint: '#dbe8ea', shadow: 'rgba(11,74,84,0.35)' },
  '#c07a3c': { accent: '#c07a3c', accent2: '#d9974f', tint: '#f6e8d6', shadow: 'rgba(192,122,60,0.35)' },
};

const TWEAK_DEFAULTS = /*EDITMODE-BEGIN*/{
  "accent": "#0b6f86",
  "displayFont": "Fredoka",
  "homeLayout": "map",
  "authLayout": "hero",
  "weather": "sunny",
  "groupMode": false
}/*EDITMODE-END*/;

const WEATHER_CYCLE = ['sunny', 'cloudy', 'rainy'];

// ── scaling stage ────────────────────────────────────────────
function useFitScale(w, h, pad = 24) {
  const [scale, setScale] = useStateApp(1);
  useEffectApp(() => {
    const fit = () => setScale(Math.min((window.innerWidth - pad) / w, (window.innerHeight - pad) / h, 1));
    fit();
    window.addEventListener('resize', fit);
    return () => window.removeEventListener('resize', fit);
  }, [w, h, pad]);
  return scale;
}

// real phone viewport → skip the decorative device bezel and fill the screen;
// wider viewports (desktop/tablet) keep the scaled IOSDevice mockup as-is
function useIsCompactViewport(breakpoint = 640) {
  const [compact, setCompact] = useStateApp(() => window.innerWidth < breakpoint);
  useEffectApp(() => {
    const check = () => setCompact(window.innerWidth < breakpoint);
    check();
    window.addEventListener('resize', check);
    return () => window.removeEventListener('resize', check);
  }, [breakpoint]);
  return compact;
}

// ── tab bar (Guide is a separate floating button — see GuideFab) ──
function TabBar({ tab, setTab, guideActive }) {
  const tabs = [
    { id: 'map', label: 'Discover', icon: 'Map' },
    { id: 'spots', label: 'Spots', icon: 'LayoutGrid' },
    { id: 'trip', label: 'Trip', icon: 'Route' },
  ];
  return (
    <div className="flex-shrink-0 px-6 pt-2" style={{ paddingBottom: 26, background: '#fff', boxShadow: '0 -2px 20px -10px rgba(16,61,58,0.18)' }}>
      <div className="flex items-center justify-around">
        {tabs.map((t) => {
          const on = t.id === tab && !guideActive;
          return (
            <button key={t.id} onClick={() => setTab(t.id)}
              className="flex flex-col items-center gap-1 py-1.5 px-2 transition-colors"
              style={{ color: on ? 'var(--accent)' : 'rgba(15,61,58,0.4)' }}>
              <div className="flex items-center justify-center rounded-2xl transition-all"
                   style={{ width: 52, height: 32, background: on ? 'var(--accent-tint)' : 'transparent' }}>
                <Icon name={t.icon} size={21} strokeWidth={on ? 2.5 : 2} />
              </div>
              <span className="text-[11px] font-extrabold">{t.label}</span>
            </button>
          );
        })}
      </div>
    </div>
  );
}

// floating Guide launcher — right side, within thumb reach, above the nav
function GuideFab({ onClick }) {
  return (
    <button onClick={onClick} aria-label="Open Local Guide"
      className="absolute z-40 flex items-center justify-center rounded-full active:scale-90 transition-transform"
      style={{ right: 16, bottom: 300, width: 58, height: 58,
               background: 'linear-gradient(140deg, var(--accent), var(--accent-2))', color: '#fff',
               border: '3px solid #fff', boxShadow: '0 16px 32px -10px var(--accent-shadow), 0 4px 12px -4px rgba(16,61,58,0.3)' }}>
      <Icon name="Sparkles" size={26} strokeWidth={2.3} />
    </button>
  );
}

function App() {
  const [t, setTweak] = useTweaks(TWEAK_DEFAULTS);
  const pal = PALETTES[t.accent] || PALETTES['#0b6f86'];

  const [authed, setAuthed] = useStateApp(() => {
    try { return localStorage.getItem('spotto_auth') === '1'; } catch { return false; }
  });
  const [onboarded, setOnboarded] = useStateApp(() => {
    try { return JSON.parse(localStorage.getItem('spotto_prefs') || 'null'); } catch { return null; }
  });
  const [tab, setTab] = useStateApp('map');
  const [showProfile, setShowProfile] = useStateApp(false);
  const [showGuide, setShowGuide] = useStateApp(false);
  const [time, setTime] = useStateApp('afternoon');
  const [selected, setSelected] = useStateApp(null);

  // Smart Companion state
  const [weather, setWeather] = useStateApp(t.weather || 'sunny');
  const [groupMode, setGroupMode] = useStateApp(!!t.groupMode);
  const [showGroup, setShowGroup] = useStateApp(false);
  const [group, setGroup] = useStateApp(() => {
    try { return JSON.parse(localStorage.getItem('spotto_group') || 'null') || window.GROUP_SEED; }
    catch { return window.GROUP_SEED; }
  });
  const [quest, setQuest] = useStateApp({ done: 0, goal: 2 });
  const [checkedIn, setCheckedIn] = useStateApp([]);            // artisan ids visited
  const [vibes, setVibes] = useStateApp(() => JSON.parse(JSON.stringify(window.VIBE_SEED || {})));

  // keep weather/group in sync with tweaks panel
  useEffectApp(() => { if (t.weather && t.weather !== weather) setWeather(t.weather); }, [t.weather]);
  useEffectApp(() => { setGroupMode(!!t.groupMode); }, [t.groupMode]);

  const scale = useFitScale(402, 874);
  const compact = useIsCompactViewport();

  function finishAuth() {
    setAuthed(true);
    try { localStorage.setItem('spotto_auth', '1'); } catch {}
  }
  function logout() {
    try { localStorage.removeItem('spotto_auth'); localStorage.removeItem('spotto_prefs'); localStorage.removeItem('spotto_trip'); localStorage.removeItem('spotto_group'); } catch {}
    setAuthed(false); setOnboarded(null); setTab('map'); setSelected(null);
  }
  function saveProfile(p) {
    setOnboarded((prev) => { const next = { ...(prev || {}), ...p }; try { localStorage.setItem('spotto_prefs', JSON.stringify(next)); } catch {} return next; });
  }
  function finishOnboarding(prefs) {
    setOnboarded(prefs);
    try { localStorage.setItem('spotto_prefs', JSON.stringify(prefs)); } catch {}
  }
  function restart() {
    try { localStorage.removeItem('spotto_prefs'); } catch {}
    setOnboarded(null); setTab('map'); setSelected(null);
  }

  function cycleWeather() {
    const next = WEATHER_CYCLE[(WEATHER_CYCLE.indexOf(weather) + 1) % WEATHER_CYCLE.length];
    setWeather(next); setTweak('weather', next);
  }
  function toggleGroup(v) { setGroupMode(v); setTweak('groupMode', v); if (!v) setShowGroup(false); }

  // ── group membership ──
  useEffectApp(() => { try { localStorage.setItem('spotto_group', JSON.stringify(group)); } catch {} }, [group]);
  const members = (window.TRAVELERS || []).filter((x) => group.memberIds.includes(x.id));
  const suggestions = (window.TRAVELERS || []).filter((x) => !group.memberIds.includes(x.id));
  function addMember(id) { setGroup((g) => g.memberIds.includes(id) ? g : { ...g, memberIds: [...g.memberIds, id] }); }
  function removeMember(id) { setGroup((g) => ({ ...g, memberIds: g.memberIds.filter((m) => m !== id) })); }
  function openGroup() { if (!groupMode) toggleGroup(true); setShowGroup(true); }
  function leaveGroup() { setShowGroup(false); toggleGroup(false); }

  function checkIn(place) {
    if (!place.artisan || checkedIn.includes(place.id)) return;
    setCheckedIn((c) => [...c, place.id]);
    setQuest((q) => ({ ...q, done: Math.min(q.goal, q.done + 1) }));
  }
  function voteVibe(placeId, vibeId) {
    setVibes((v) => ({ ...v, [placeId]: { ...(v[placeId] || {}), [vibeId]: ((v[placeId] || {})[vibeId] || 0) + 1 } }));
  }

  const styleObj = (window.TRAVEL_STYLES.find((s) => s.id === onboarded?.style)) || window.TRAVEL_STYLES[0];
  const styleLabel = styleObj.label;
  const interestLabels = (onboarded?.interests || []).map((id) => (window.INTERESTS.find((x) => x.id === id) || {}).label).filter(Boolean);

  function openPlace(p) { setSelected(p); }
  function openPlaceFromGuide(p) { setShowGuide(false); setTab('map'); setSelected(p); }
  function questTap() {
    const artisan = window.PLACES.find((p) => p.artisan && !checkedIn.includes(p.id)) || window.PLACES.find((p) => p.artisan);
    if (artisan) { setTab('map'); setSelected(artisan); }
  }

  const cssVars = {
    '--accent': pal.accent, '--accent-2': pal.accent2, '--accent-tint': pal.tint, '--accent-shadow': pal.shadow,
    '--sun': '#f6b81e', '--ink': '#103d3a', '--ink-soft': '#3a5a57', '--muted': '#6c8582', '--sand': '#fbf6ed',
    '--font-display': `'${t.displayFont}', system-ui`, '--font-body': "'Nunito', system-ui",
  };

  const screens = (
    <div className="h-full flex flex-col relative" style={{ background: 'var(--sand)' }}>
      {!authed ? (
        <Auth onAuthed={finishAuth} layout={t.authLayout} />
      ) : !onboarded ? (
        <Onboarding onDone={finishOnboarding} />
      ) : (
        <>
          <div className="flex-1 relative overflow-hidden">
            {tab === 'map' && (
              t.homeLayout === 'feed' ? (
                <HomeFeed places={window.PLACES} onSelect={openPlace} styleLabel={styleLabel}
                          weather={weather} time={time} setTime={setTime}
                          quest={quest} onQuestTap={questTap} onProfile={() => setShowProfile(true)} />
              ) : t.homeLayout === 'today' ? (
                <HomeToday places={window.PLACES} onSelect={openPlace} styleLabel={styleLabel}
                           weather={weather} time={time} setTime={setTime}
                           members={members} groupMode={groupMode} onOpenGroup={openGroup}
                           quest={quest} onQuestTap={questTap} onProfile={() => setShowProfile(true)} />
              ) : (
                <MapView time={time} setTime={setTime} places={window.PLACES} onSelect={openPlace}
                         styleLabel={styleLabel} weather={weather} cycleWeather={cycleWeather}
                         groupMode={groupMode} setGroupMode={toggleGroup} members={members} onOpenGroup={openGroup}
                         quest={quest} onQuestTap={questTap}
                         onProfile={() => setShowProfile(true)} />
              )
            )}
            {tab === 'spots' && <SpotsList places={window.PLACES} onSelect={openPlace} styleLabel={styleLabel} />}
            {tab === 'trip' && <TripView styleId={onboarded.style} onOpenPlace={(sid) => { const p = window.PLACES.find((x) => x.id === sid); if (p) setSelected(p); }} />}
          </div>
          <TabBar tab={tab} setTab={setTab} guideActive={showGuide} />
          {!showGuide && <GuideFab onClick={() => setShowGuide(true)} />}
          {showGuide && (
            <div className="absolute inset-0 z-40" style={{ animation: 'spottoGuideIn 260ms cubic-bezier(0.22,1,0.36,1)' }}>
              <AIGuide style={onboarded.style} styleLabel={styleLabel} interestLabels={interestLabels}
                       weather={weather} onOpenPlace={openPlaceFromGuide} onClose={() => setShowGuide(false)} />
            </div>
          )}
          <BusinessDrawer place={selected} onClose={() => setSelected(null)}
                          vibes={selected ? (vibes[selected.id] || {}) : {}} onVote={voteVibe}
                          onCheckIn={checkIn} checkedIn={checkedIn} />
          <GroupSheet open={showGroup} onClose={() => setShowGroup(false)} group={group}
                      members={members} suggestions={suggestions} me={window.ME}
                      onAdd={addMember} onRemove={removeMember} onLeave={leaveGroup} />
          {showProfile && <Profile prefs={onboarded} onSave={saveProfile} onClose={() => setShowProfile(false)} onLogout={logout} />}
        </>
      )}
    </div>
  );

  return (
    <div style={{ position: 'fixed', inset: 0, background: 'radial-gradient(120% 120% at 50% 0%, #ffffff, #f3ead9)', ...cssVars, fontFamily: 'var(--font-body)' }}>
      {compact ? (
        <div style={{ position: 'absolute', inset: 0 }}>{screens}</div>
      ) : (
        <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
          <div style={{ transform: `scale(${scale})`, transformOrigin: 'center center' }}>
            <IOSDevice>{screens}</IOSDevice>
          </div>
        </div>
      )}

      <TweaksPanel>
        <TweakSection label="Theme" />
        <TweakColor label="Accent" value={t.accent}
          options={['#0b6f86', '#0e8c8a', '#0b4a54', '#c07a3c']}
          onChange={(v) => setTweak('accent', v)} />
        <TweakSelect label="Heading font" value={t.displayFont}
          options={['Fredoka', 'Baloo 2', 'Quicksand', 'Nunito']}
          onChange={(v) => setTweak('displayFont', v)} />
        <TweakSection label="Home page" />
        <TweakRadio label="Layout" value={t.homeLayout} options={['map', 'feed', 'today']}
          onChange={(v) => setTweak('homeLayout', v)} />
        <TweakSection label="Smart Companion" />
        <TweakRadio label="Weather" value={weather} options={['sunny', 'cloudy', 'rainy']}
          onChange={(v) => { setWeather(v); setTweak('weather', v); }} />
        <TweakToggle label="Group mode" value={groupMode} onChange={toggleGroup} />
        <TweakSection label="Sign-in" />
        <TweakRadio label="Layout" value={t.authLayout} options={['hero', 'immersive', 'minimal']}
          onChange={(v) => setTweak('authLayout', v)} />
        <TweakSection label="Demo" />
        <TweakButton label="Restart onboarding" onClick={restart} />
        <TweakButton label="Log out" onClick={logout} secondary />
      </TweaksPanel>
    </div>
  );
}

ReactDOM.createRoot(document.getElementById('root')).render(<App />);
