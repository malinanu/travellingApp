/* Spotto "First Light" — Three.js scene module.
   A single ocean-and-island flight scene whose parameters are driven by one
   scrollProgress value (0 to 1) fed in each frame by main.js. Exposes
   update(progress, dt) and resize(). */
import * as THREE from 'three';

const NIGHT = new THREE.Color('#070B14');
const OCEAN_NIGHT_LOW = new THREE.Color('#0B1330');
const OCEAN_NIGHT_HIGH = new THREE.Color('#1B3B63');
const OCEAN_DAWN_LOW = new THREE.Color('#123056');
const OCEAN_DAWN_HIGH = new THREE.Color('#4a7ba8');
const SKY_NIGHT = new THREE.Color('#070B14');
const SKY_DAWN = new THREE.Color('#2b3a52');
const DAWN = new THREE.Color('#FF6A3D');

export function createScene(canvas) {
  const renderer = new THREE.WebGLRenderer({
    canvas,
    antialias: true,
    alpha: false,
    powerPreference: 'low-power'
  });
  renderer.setPixelRatio(Math.min(window.devicePixelRatio || 1, 2));
  renderer.setClearColor(NIGHT, 1);

  const scene = new THREE.Scene();
  scene.fog = new THREE.Fog(SKY_NIGHT.clone(), 40, 180);

  const camera = new THREE.PerspectiveCamera(48, 2, 0.1, 400);
  camera.position.set(0, 22, 70);
  camera.lookAt(0, 6, -40);

  /* ---------- Ocean ---------- */
  const SEG_X = 140, SEG_Z = 90;
  const oceanGeo = new THREE.PlaneGeometry(400, 260, SEG_X, SEG_Z);
  oceanGeo.rotateX(-Math.PI / 2);
  const oceanCount = oceanGeo.attributes.position.count;
  oceanGeo.setAttribute('color', new THREE.BufferAttribute(new Float32Array(oceanCount * 3), 3));
  const ocean = new THREE.Mesh(oceanGeo, new THREE.MeshBasicMaterial({ vertexColors: true, fog: true }));
  ocean.position.set(0, -6, -60);
  scene.add(ocean);
  const oceanPos = oceanGeo.attributes.position;
  const oceanCol = oceanGeo.attributes.color;

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
  const sun = new THREE.Mesh(
    new THREE.SphereGeometry(6, 24, 16),
    new THREE.MeshBasicMaterial({ color: DAWN, fog: false })
  );
  const glow = new THREE.Sprite(new THREE.SpriteMaterial({
    map: sunGlowTex, color: DAWN, transparent: true, blending: THREE.AdditiveBlending, depthWrite: false, fog: false
  }));
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
    for (let i = 0; i < oceanCount; i++) {
      const x = oceanPos.array[i * 3];
      const z = oceanPos.array[i * 3 + 2];
      const h = Math.sin(x * 0.05 + t * 0.6) * 1.6 +
                Math.sin(z * 0.08 - t * 0.8) * 1.1 +
                Math.sin((x + z) * 0.03 + t * 0.4) * 0.8;
      oceanPos.array[i * 3 + 1] = h;
      const k = Math.max(0, Math.min(1, (h + 3.5) / 7));
      oceanCol.array[i * 3] = tmpLow.r + (tmpHigh.r - tmpLow.r) * k;
      oceanCol.array[i * 3 + 1] = tmpLow.g + (tmpHigh.g - tmpLow.g) * k;
      oceanCol.array[i * 3 + 2] = tmpLow.b + (tmpHigh.b - tmpLow.b) * k;
    }
    oceanPos.needsUpdate = true;
    oceanCol.needsUpdate = true;
  }

  let clock = 0;

  function update(progress, dt) {
    clock += dt;
    // progress: 0 = hero (night, far), 1 = climax reached (full dawn, close to island)
    const flight = Math.min(1, progress / 0.82);
    const dawnK = Math.max(0, Math.min(1, (progress - 0.35) / 0.5));

    // Camera flies forward and descends slightly toward the island.
    camera.position.z = 70 - flight * 130;
    camera.position.y = 22 - flight * 10;
    camera.lookAt(0, 4 - flight * 2, -145);

    // Sky / fog
    tmpSky.copy(SKY_NIGHT).lerp(SKY_DAWN, dawnK);
    scene.fog.color.copy(tmpSky);
    renderer.setClearColor(tmpSky, 1);

    // Stars fade as dawn approaches
    starMat.opacity = Math.max(0, 1 - dawnK * 1.3);

    // Sun rises over the second half of the flight
    const sunK = Math.max(0, Math.min(1, (progress - 0.3) / 0.55));
    sun.position.y = -20 + sunK * 46;
    sun.visible = sunK > 0.02;
    glow.material.opacity = 0.55 + sunK * 0.35;

    updateOcean(clock, dawnK);

    renderer.render(scene, camera);
  }

  function resize(w, h) {
    renderer.setSize(w, h, false);
    camera.aspect = w / h;
    camera.updateProjectionMatrix();
  }

  return { update, resize, renderer };
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
