/* Spotto "First Light" — orchestration.
   One rAF loop drives the Three.js render AND all DOM text-beat opacity by
   polling window.scrollY once per frame (not a scroll event listener).
   Falls back to a static, already-complete screen under reduced-motion or
   when WebGL is unavailable. */

const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

const track = document.querySelector('.track');
const canvas = document.getElementById('scene-canvas');
const stories = [...document.querySelectorAll('.story')];

function useFallback() {
  document.body.classList.add('is-fallback');
}

if (reduceMotion) {
  useFallback();
} else {
  boot();
}

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
    sceneApi = createScene(canvas);
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
