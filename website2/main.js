/* Spotto "First Light" — orchestration.
   One rAF loop drives the Three.js render AND all DOM text-beat opacity by
   polling window.scrollY once per frame (not a scroll event listener).
   Falls back to a static, already-complete screen under reduced-motion,
   when WebGL is unavailable, or if the GPU context is later lost. */

const mql = window.matchMedia('(prefers-reduced-motion: reduce)');

const track = document.querySelector('.track');
const canvas = document.getElementById('scene-canvas');
const stories = [...document.querySelectorAll('.story')];

// Set once boot() has a running scene, so useFallback() can tear it down
// cleanly (stop the rAF loop, release GPU resources) from any trigger:
// reduced-motion toggled on mid-session, or the WebGL context dropping.
let stopScene = null;

function useFallback() {
  document.body.classList.add('is-fallback');
  // The fallback is a single full-viewport screen, not a scrollable track —
  // switching to it mid-flight (a live reduced-motion toggle or a lost GPU
  // context, both possible well after the user has scrolled) would otherwise
  // leave the viewport scrolled into the fallback content instead of at its
  // top, overlapping its stacked text.
  window.scrollTo(0, 0);
  if (stopScene) {
    stopScene();
    stopScene = null;
  }
}

if (mql.matches) {
  useFallback();
} else {
  boot();
}

// Live listener, not just the one-time check above — a user (or an
// automated test) can toggle the OS-level reduced-motion setting while the
// tab is open.
mql.addEventListener('change', (e) => {
  if (e.matches) useFallback();
});

async function boot() {
  let createScene;
  try {
    ({ createScene } = await import('./scene.js'));
  } catch (e) {
    console.info('Spotto First Light: scene module failed to load, using static fallback.', e.message);
    useFallback();
    return;
  }

  let sceneApi;
  try {
    sceneApi = createScene(canvas, { onContextLost: () => useFallback() });
  } catch (e) {
    console.info('Spotto First Light: WebGL unavailable, using static fallback.', e.message);
    useFallback();
    return;
  }

  function resize() {
    sceneApi.resize(window.innerWidth, window.innerHeight);
  }
  resize();
  window.addEventListener('resize', resize);

  // The scroll track's height (minus one viewport) defines the flight length.
  function scrollRange() {
    return Math.max(1, track.scrollHeight - window.innerHeight);
  }

  let visible = !document.hidden;
  document.addEventListener('visibilitychange', () => { visible = !document.hidden; });

  let last = performance.now();
  let raf = requestAnimationFrame(frame);
  stopScene = () => {
    cancelAnimationFrame(raf);
    window.removeEventListener('resize', resize);
    sceneApi.dispose();
  };

  function frame(now) {
    raf = requestAnimationFrame(frame);
    if (!visible) { last = now; return; }
    const dt = Math.min(0.05, (now - last) / 1000);
    last = now;

    const progress = Math.max(0, Math.min(1, window.scrollY / scrollRange()));
    sceneApi.update(progress, dt);
    updateStoryBeats(progress);
  }

  function updateStoryBeats(progress) {
    stories.forEach((el) => {
      const from = parseFloat(el.dataset.from);
      const to = parseFloat(el.dataset.to);
      el.classList.toggle('on', progress >= from && progress <= to);
    });
  }
}
