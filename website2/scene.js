/* Spotto "First Light" — Three.js scene module.
   A single ocean-and-island flight scene whose parameters are driven by one
   scrollProgress value (0 to 1) fed in each frame by main.js. Exposes
   update(progress, dt), resize(), and dispose(). */
import * as THREE from 'three';

const NIGHT = new THREE.Color('#070B14');
const OCEAN_NIGHT_LOW = new THREE.Color('#0B1330');
const OCEAN_NIGHT_HIGH = new THREE.Color('#1B3B63');
const OCEAN_DAWN_LOW = new THREE.Color('#123056');
const OCEAN_DAWN_HIGH = new THREE.Color('#4a7ba8');
const SKY_NIGHT = new THREE.Color('#070B14');
const SKY_DAWN = new THREE.Color('#2b3a52');
const DAWN = new THREE.Color('#FF6A3D');

// Flight-timing constants for update() — all expressed as fractions of
// scrollProgress (0 = hero, 1 = climax reached).
const FLIGHT_END = 0.82; // camera finishes its forward/descent move
const DAWN_START = 0.35; // sky/ocean/fog start shifting from night to dawn colors
const DAWN_SPAN = 0.5; // how much progress the night->dawn shift takes
const SUN_RISE_START = 0.3; // sun starts climbing above the horizon
const SUN_RISE_SPAN = 0.55; // how much progress the sunrise takes
const STAR_FADE_RATE = 1.3; // stars fade out faster than dawnK alone (fully gone before full dawn)

const OCEAN_VERTEX_SHADER = /* glsl */ `
  uniform float uTime;
  varying float vK;
  #include <fog_pars_vertex>

  void main() {
    float h = sin(position.x * 0.05 + uTime * 0.6) * 1.6
            + sin(position.z * 0.08 - uTime * 0.8) * 1.1
            + sin((position.x + position.z) * 0.03 + uTime * 0.4) * 0.8;
    vK = clamp((h + 3.5) / 7.0, 0.0, 1.0);
    vec3 transformed = vec3(position.x, h, position.z);
    vec4 mvPosition = modelViewMatrix * vec4(transformed, 1.0);
    gl_Position = projectionMatrix * mvPosition;
    #include <fog_vertex>
  }
`;

const OCEAN_FRAGMENT_SHADER = /* glsl */ `
  uniform vec3 uLowColor;
  uniform vec3 uHighColor;
  varying float vK;
  #include <fog_pars_fragment>

  void main() {
    vec3 color = mix(uLowColor, uHighColor, vK);
    gl_FragColor = vec4(color, 1.0);
    #include <fog_fragment>
  }
`;

export function createScene(canvas, { onContextLost } = {}) {
  const renderer = new THREE.WebGLRenderer({
    canvas,
    antialias: true,
    alpha: false,
    powerPreference: 'low-power'
  });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
  renderer.setClearColor(NIGHT, 1);

  // A dropped GPU context (thermal throttling, driver reset, etc.) makes
  // renderer.render() a silent no-op — the page would otherwise freeze on
  // whatever frame was last drawn. preventDefault() keeps the context
  // eligible for restoration, but we don't attempt to resume the animated
  // scene ourselves; the caller (main.js) falls back to the static screen.
  canvas.addEventListener('webglcontextlost', (e) => {
    e.preventDefault();
    if (onContextLost) onContextLost();
  });

  const scene = new THREE.Scene();
  scene.fog = new THREE.Fog(SKY_NIGHT.clone(), 40, 180);

  const camera = new THREE.PerspectiveCamera(48, 2, 0.1, 400);
  camera.position.set(0, 22, 70);
  camera.lookAt(0, 6, -40);

  /* ---------- Ocean ----------
     Wave height + color are computed entirely in the vertex/fragment shader
     (GPU) rather than a per-frame CPU loop over ~12.8k vertices — the JS
     side only pushes 3 uniforms a frame. */
  const SEG_X = 140, SEG_Z = 90;
  const oceanGeo = new THREE.PlaneGeometry(400, 260, SEG_X, SEG_Z);
  oceanGeo.rotateX(-Math.PI / 2);
  const oceanUniforms = THREE.UniformsUtils.merge([
    THREE.UniformsLib.fog,
    {
      uTime: { value: 0 },
      uLowColor: { value: new THREE.Color() },
      uHighColor: { value: new THREE.Color() }
    }
  ]);
  const oceanMat = new THREE.ShaderMaterial({
    uniforms: oceanUniforms,
    vertexShader: OCEAN_VERTEX_SHADER,
    fragmentShader: OCEAN_FRAGMENT_SHADER,
    fog: true
  });
  const ocean = new THREE.Mesh(oceanGeo, oceanMat);
  ocean.position.set(0, -6, -60);
  scene.add(ocean);

  /* ---------- Island silhouette ---------- */
  const islandShape = new THREE.Shape();
  islandShape.moveTo(-46, 0);
  islandShape.lineTo(-30, 16);
  islandShape.lineTo(-14, 9);
  islandShape.lineTo(0, 26);
  islandShape.lineTo(16, 12);
  islandShape.lineTo(32, 20);
  islandShape.lineTo(46, 0);
  islandShape.lineTo(-46, 0);
  const islandGeo = new THREE.ExtrudeGeometry(islandShape, { depth: 10, bevelEnabled: false });
  const islandMat = new THREE.MeshBasicMaterial({ color: 0x040608, fog: true });
  const island = new THREE.Mesh(islandGeo, islandMat);
  island.position.set(0, -2, -145);
  scene.add(island);

  /* ---------- Stars ---------- */
  const STAR_COUNT = 500;
  const starPos = new Float32Array(STAR_COUNT * 3);
  for (let i = 0; i < STAR_COUNT; i++) {
    starPos[i * 3] = (Math.random() - 0.5) * 300;
    starPos[i * 3 + 1] = Math.random() * 110 + 10;
    starPos[i * 3 + 2] = -Math.random() * 250 - 20;
  }
  const starGeo = new THREE.BufferGeometry();
  starGeo.setAttribute('position', new THREE.BufferAttribute(starPos, 3));
  const starMat = new THREE.PointsMaterial({ color: 0xffffff, size: 0.9, transparent: true, opacity: 1, fog: false });
  const stars = new THREE.Points(starGeo, starMat);
  scene.add(stars);

  /* ---------- Sun ---------- */
  const sunGlowTex = makeGlowTexture();
  const sunGeo = new THREE.SphereGeometry(6, 24, 16);
  const sunMat = new THREE.MeshBasicMaterial({ color: DAWN, fog: false });
  const sun = new THREE.Mesh(sunGeo, sunMat);
  const glowMat = new THREE.SpriteMaterial({
    map: sunGlowTex, color: DAWN, transparent: true, blending: THREE.AdditiveBlending, depthWrite: false, fog: false
  });
  const glow = new THREE.Sprite(glowMat);
  glow.scale.set(70, 70, 1);
  sun.add(glow);
  sun.position.set(0, -20, -150);
  scene.add(sun);

  const tmpLow = new THREE.Color();
  const tmpHigh = new THREE.Color();
  const tmpSky = new THREE.Color();

  function updateOcean(t, dawnK) {
    tmpLow.copy(OCEAN_NIGHT_LOW).lerp(OCEAN_DAWN_LOW, dawnK);
    tmpHigh.copy(OCEAN_NIGHT_HIGH).lerp(OCEAN_DAWN_HIGH, dawnK);
    oceanUniforms.uTime.value = t;
    oceanUniforms.uLowColor.value.copy(tmpLow);
    oceanUniforms.uHighColor.value.copy(tmpHigh);
  }

  let clock = 0;

  function update(progress, dt) {
    clock += dt;
    // progress: 0 = hero (night, far), 1 = climax reached (full dawn, close to island)
    const flight = Math.min(1, progress / FLIGHT_END);
    const dawnK = Math.max(0, Math.min(1, (progress - DAWN_START) / DAWN_SPAN));

    // Camera flies forward and descends slightly toward the island.
    camera.position.z = 70 - flight * 130;
    camera.position.y = 22 - flight * 10;
    camera.lookAt(0, 4 - flight * 2, -145);

    // Sky / fog
    tmpSky.copy(SKY_NIGHT).lerp(SKY_DAWN, dawnK);
    scene.fog.color.copy(tmpSky);
    renderer.setClearColor(tmpSky, 1);

    // Stars fade as dawn approaches
    starMat.opacity = Math.max(0, 1 - dawnK * STAR_FADE_RATE);

    // Sun rises over the second half of the flight
    const sunK = Math.max(0, Math.min(1, (progress - SUN_RISE_START) / SUN_RISE_SPAN));
    sun.position.y = -20 + sunK * 46;
    sun.visible = sunK > 0.02;
    glow.material.opacity = 0.55 + sunK * 0.35;

    updateOcean(clock, dawnK);

    renderer.render(scene, camera);
  }

  function resize(w, h) {
    // Re-read devicePixelRatio on every resize, not just at creation — a
    // window dragged across monitors with different DPR would otherwise
    // keep rendering at the old ratio.
    renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
    renderer.setSize(w, h, false);
    camera.aspect = w / h;
    camera.updateProjectionMatrix();
  }

  function dispose() {
    oceanGeo.dispose();
    oceanMat.dispose();
    islandGeo.dispose();
    islandMat.dispose();
    starGeo.dispose();
    starMat.dispose();
    sunGeo.dispose();
    sunMat.dispose();
    glowMat.dispose();
    sunGlowTex.dispose();
    renderer.dispose();
  }

  return { update, resize, dispose, renderer };
}

function makeGlowTexture() {
  const c = document.createElement('canvas');
  c.width = c.height = 128;
  const ctx = c.getContext('2d');
  const grad = ctx.createRadialGradient(64, 64, 4, 64, 64, 64);
  grad.addColorStop(0, 'rgba(255,255,255,0.95)');
  grad.addColorStop(0.4, 'rgba(255,255,255,0.35)');
  grad.addColorStop(1, 'rgba(255,255,255,0)');
  ctx.fillStyle = grad;
  ctx.fillRect(0, 0, 128, 128);
  return new THREE.CanvasTexture(c);
}
