const canvas = document.querySelector('#scene-canvas');
const stage = document.querySelector('[data-motion-stage]');
const chapters = [...document.querySelectorAll('[data-scene]')];
const progressBar = document.querySelector('[data-progress-bar]');
const progressLabel = document.querySelector('[data-progress-label]');
const sceneLabel = document.querySelector('[data-scene-label]');
const header = document.querySelector('[data-header]');
const menuButton = document.querySelector('.menu-button');
const navigation = document.querySelector('#main-navigation');
const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)');

const sceneNames = ['LOCAL', 'DELTA', 'CACHE', 'FIFO', 'DIGEST', 'READY'];
const design = { width: 1000, height: 650 };
const clamp = (value, min = 0, max = 1) => Math.min(max, Math.max(min, value));
const mix = (from, to, amount) => from + (to - from) * amount;
const smoothstep = (value) => value * value * (3 - 2 * value);

let context;
let maskCanvas;
let maskContext;
let targets = [];
let pointCount = 0;
let targetProgress = 0;
let currentProgress = 0;
let activeScene = -1;
let frameRequest;
let resizeRequest;
let pointerX = 0;
let pointerY = 0;

function seededRandom(seed) {
  let state = seed >>> 0;
  return () => {
    state = (state * 1664525 + 1013904223) >>> 0;
    return state / 4294967296;
  };
}

function roundedRect(ctx, x, y, width, height, radius) {
  const r = Math.min(radius, width / 2, height / 2);
  ctx.beginPath();
  ctx.moveTo(x + r, y);
  ctx.arcTo(x + width, y, x + width, y + height, r);
  ctx.arcTo(x + width, y + height, x, y + height, r);
  ctx.arcTo(x, y + height, x, y, r);
  ctx.arcTo(x, y, x + width, y, r);
  ctx.closePath();
}

function drawLaptop(ctx) {
  ctx.save();
  ctx.lineCap = 'round';
  ctx.lineJoin = 'round';
  ctx.strokeStyle = '#000';
  ctx.fillStyle = '#000';

  ctx.lineWidth = 12;
  roundedRect(ctx, 154, 86, 692, 420, 22);
  ctx.stroke();

  ctx.lineWidth = 7;
  roundedRect(ctx, 185, 118, 630, 347, 10);
  ctx.stroke();

  ctx.beginPath();
  ctx.moveTo(91, 531);
  ctx.lineTo(909, 531);
  ctx.lineTo(842, 566);
  ctx.lineTo(158, 566);
  ctx.closePath();
  ctx.fill();

  ctx.font = '700 30px monospace';
  ctx.fillText('$ autback exec -- task test', 230, 208);
  ctx.globalAlpha = 0.58;
  ctx.fillRect(230, 248, 365, 10);
  ctx.fillRect(230, 282, 480, 10);
  ctx.fillRect(230, 316, 288, 10);
  ctx.globalAlpha = 1;
  ctx.beginPath();
  ctx.arc(743, 407, 16, 0, Math.PI * 2);
  ctx.fill();
  ctx.restore();
}

function drawTransfer(ctx) {
  ctx.save();
  ctx.strokeStyle = '#000';
  ctx.fillStyle = '#000';
  ctx.lineCap = 'round';
  ctx.lineJoin = 'round';

  ctx.lineWidth = 10;
  roundedRect(ctx, 70, 188, 238, 252, 18);
  ctx.stroke();
  ctx.fillRect(112, 238, 145, 12);
  ctx.fillRect(112, 282, 112, 12);
  ctx.fillRect(112, 326, 158, 12);
  ctx.fillRect(112, 370, 84, 12);

  roundedRect(ctx, 692, 188, 238, 252, 18);
  ctx.stroke();
  ctx.lineWidth = 7;
  for (let row = 0; row < 4; row += 1) {
    roundedRect(ctx, 735, 226 + row * 48, 152, 25, 7);
    ctx.stroke();
  }

  ctx.setLineDash([5, 21]);
  ctx.lineWidth = 7;
  ctx.beginPath();
  ctx.moveTo(330, 314);
  ctx.bezierCurveTo(445, 196, 565, 432, 670, 314);
  ctx.stroke();
  ctx.setLineDash([]);

  const packets = [372, 438, 505, 571, 633];
  packets.forEach((x, index) => {
    const y = 314 + Math.sin(index * 1.7) * 56;
    roundedRect(ctx, x - 18, y - 14, 36, 28, 5);
    index < 2 ? ctx.stroke() : ctx.fill();
  });

  ctx.font = '700 24px monospace';
  ctx.fillText('1,311 files', 86, 142);
  ctx.fillText('0 B missing', 731, 142);
  ctx.restore();
}

function drawWorker(ctx) {
  ctx.save();
  ctx.strokeStyle = '#000';
  ctx.fillStyle = '#000';
  ctx.lineCap = 'round';
  ctx.lineJoin = 'round';

  ctx.lineWidth = 11;
  roundedRect(ctx, 235, 55, 530, 540, 28);
  ctx.stroke();

  ctx.font = '700 25px monospace';
  ctx.fillText('AUTBACK WORKER', 284, 112);
  ctx.fillRect(284, 138, 430, 5);

  for (let row = 0; row < 3; row += 1) {
    ctx.lineWidth = 7;
    roundedRect(ctx, 286, 182 + row * 112, 428, 78, 12);
    ctx.stroke();
    ctx.beginPath();
    ctx.arc(325, 221 + row * 112, 10, 0, Math.PI * 2);
    ctx.fill();
    ctx.fillRect(358, 211 + row * 112, 172 + row * 42, 17);
  }

  ctx.font = '700 34px monospace';
  ctx.fillText('4 vCPU', 286, 555);
  ctx.fillText('8 GB', 584, 555);
  ctx.restore();
}

function drawQueue(ctx) {
  ctx.save();
  ctx.strokeStyle = '#000';
  ctx.fillStyle = '#000';
  ctx.lineCap = 'round';
  ctx.lineJoin = 'round';

  ctx.font = '700 22px monospace';
  ctx.fillText('STRICT FIFO', 93, 93);

  const rows = [146, 252, 358];
  rows.forEach((y, index) => {
    ctx.lineWidth = index === 0 ? 10 : 6;
    roundedRect(ctx, 92, y, 660, 76, 12);
    ctx.stroke();
    ctx.fillText(`0${index + 1}`, 126, y + 47);
    ctx.fillRect(205, y + 30, 195 + index * 35, 14);
    if (index === 0) ctx.fillRect(654, y + 24, 58, 26);
  });

  ctx.lineWidth = 9;
  ctx.beginPath();
  ctx.moveTo(792, 184);
  ctx.lineTo(885, 184);
  ctx.lineTo(855, 154);
  ctx.moveTo(885, 184);
  ctx.lineTo(855, 214);
  ctx.stroke();

  roundedRect(ctx, 788, 272, 130, 162, 18);
  ctx.stroke();
  ctx.fillRect(817, 312, 72, 16);
  ctx.fillRect(817, 348, 72, 16);
  ctx.fillRect(817, 384, 72, 16);

  ctx.font = '700 26px monospace';
  ctx.fillText('ELASTIC CPU + RAM', 92, 526);
  ctx.fillRect(92, 548, 826, 7);
  ctx.restore();
}

function drawDigest(ctx) {
  ctx.save();
  ctx.strokeStyle = '#000';
  ctx.fillStyle = '#000';
  ctx.lineCap = 'round';
  ctx.lineJoin = 'round';

  ctx.lineWidth = 9;
  const layers = [
    [108, 348, 340, 106],
    [160, 274, 340, 106],
    [212, 200, 340, 106],
  ];
  layers.forEach(([x, y, width, height]) => {
    roundedRect(ctx, x, y, width, height, 14);
    ctx.stroke();
  });

  ctx.setLineDash([4, 18]);
  ctx.beginPath();
  ctx.moveTo(570, 326);
  ctx.lineTo(720, 326);
  ctx.stroke();
  ctx.setLineDash([]);

  ctx.beginPath();
  ctx.arc(806, 326, 84, 0, Math.PI * 2);
  ctx.stroke();
  ctx.lineWidth = 14;
  ctx.beginPath();
  ctx.moveTo(766, 327);
  ctx.lineTo(795, 357);
  ctx.lineTo(850, 287);
  ctx.stroke();

  ctx.font = '700 25px monospace';
  ctx.fillText('BUILD + PUSH', 210, 146);
  ctx.fillText('sha256:9f34…', 695, 456);
  ctx.fillRect(696, 474, 197, 7);
  ctx.restore();
}

function drawLandscape(ctx) {
  ctx.save();
  ctx.strokeStyle = '#000';
  ctx.fillStyle = '#000';
  ctx.lineCap = 'round';
  ctx.lineJoin = 'round';

  ctx.lineWidth = 6;
  for (let ridge = 0; ridge < 9; ridge += 1) {
    ctx.globalAlpha = 1 - ridge * 0.07;
    ctx.beginPath();
    ctx.moveTo(-30, 418 + ridge * 17);
    ctx.bezierCurveTo(130, 310 + ridge * 14, 260, 446 + ridge * 13, 400, 350 + ridge * 17);
    ctx.bezierCurveTo(550, 245 + ridge * 16, 660, 432 + ridge * 12, 1030, 300 + ridge * 18);
    ctx.stroke();
  }
  ctx.globalAlpha = 1;

  ctx.lineWidth = 18;
  ctx.beginPath();
  ctx.moveTo(310, 472);
  ctx.bezierCurveTo(326, 385, 296, 335, 330, 250);
  ctx.stroke();

  const canopy = [
    [274, 226, 70], [344, 205, 88], [408, 240, 74], [230, 276, 58], [350, 278, 80],
  ];
  canopy.forEach(([x, y, radius]) => {
    ctx.beginPath();
    ctx.arc(x, y, radius, 0, Math.PI * 2);
    ctx.fill();
  });

  ctx.font = '700 26px monospace';
  ctx.fillText('HEAVY WORK, LIGHT LAPTOP', 508, 522);
  ctx.restore();
}

const targetDrawers = [drawLaptop, drawTransfer, drawWorker, drawQueue, drawDigest, drawLandscape];

function sampleTarget(drawer, seed) {
  maskContext.clearRect(0, 0, design.width, design.height);
  maskContext.createImageData(1, 1);
  drawer(maskContext);

  const pixels = maskContext.getImageData(0, 0, design.width, design.height).data;
  const random = seededRandom(seed);
  const candidates = [];
  const step = window.innerWidth <= 720 ? 4 : 3;

  for (let y = 0; y < design.height; y += step) {
    for (let x = 0; x < design.width; x += step) {
      const alpha = pixels[(y * design.width + x) * 4 + 3];
      if (alpha > 22 && random() < alpha / 255) {
        candidates.push({
          x: (x + (random() - 0.5) * step) / design.width,
          y: (y + (random() - 0.5) * step) / design.height,
        });
      }
    }
  }

  candidates.sort((a, b) => {
    const angleA = Math.atan2(a.y - 0.5, a.x - 0.5);
    const angleB = Math.atan2(b.y - 0.5, b.x - 0.5);
    const ringA = Math.hypot(a.x - 0.5, a.y - 0.5);
    const ringB = Math.hypot(b.x - 0.5, b.y - 0.5);
    return angleA - angleB || ringA - ringB;
  });

  const sampled = [];
  if (candidates.length >= pointCount) {
    const offset = random() * (candidates.length / pointCount);
    for (let index = 0; index < pointCount; index += 1) {
      sampled.push(candidates[Math.floor(offset + index * candidates.length / pointCount) % candidates.length]);
    }
  } else {
    for (let index = 0; index < pointCount; index += 1) {
      const source = candidates[index % candidates.length] || { x: random(), y: random() };
      const duplicate = Math.floor(index / Math.max(1, candidates.length));
      sampled.push({
        x: source.x + (random() - 0.5) * duplicate * 0.003,
        y: source.y + (random() - 0.5) * duplicate * 0.003,
      });
    }
  }
  return sampled;
}

function prepareTargets() {
  const nextPointCount = window.innerWidth <= 720 ? 3000 : 7600;
  if (targets.length && nextPointCount === pointCount) return;
  pointCount = nextPointCount;
  targets = targetDrawers.map((drawer, index) => sampleTarget(drawer, 9013 + index * 7919));
}

function sceneBounds() {
  const mobile = window.innerWidth <= 720;
  if (mobile) {
    return {
      left: -window.innerWidth * 0.07,
      top: window.innerHeight * 0.13,
      width: window.innerWidth * 1.14,
      height: window.innerHeight * 0.48,
    };
  }
  return {
    left: window.innerWidth * 0.018,
    top: window.innerHeight * 0.145,
    width: window.innerWidth * 0.64,
    height: window.innerHeight * 0.74,
  };
}

function drawFrame() {
  if (!context || !targets.length) return;
  const width = window.innerWidth;
  const height = window.innerHeight;
  context.clearRect(0, 0, width, height);

  const sceneFloat = currentProgress * (targets.length - 1);
  const fromIndex = Math.min(targets.length - 1, Math.floor(sceneFloat));
  const toIndex = Math.min(targets.length - 1, fromIndex + 1);
  const rawMix = sceneFloat - fromIndex;
  const amount = smoothstep(rawMix);
  const from = targets[fromIndex];
  const to = targets[toIndex];
  const bounds = sceneBounds();
  const transitionEnergy = Math.sin(amount * Math.PI);
  const time = performance.now() * 0.00022;

  context.save();
  context.fillStyle = '#17130f';
  for (let index = 0; index < pointCount; index += 1) {
    const a = from[index];
    const b = to[index];
    const wave = Math.sin(index * 0.618 + time * 9) * transitionEnergy;
    const curl = Math.cos(index * 0.193 - time * 7) * transitionEnergy;
    const x = bounds.left + mix(a.x, b.x, amount) * bounds.width + wave * 13 + pointerX;
    const y = bounds.top + mix(a.y, b.y, amount) * bounds.height + curl * 8 + pointerY;
    const accent = index % 71 === 0 || (fromIndex === 4 && index % 43 === 0);
    const depth = 0.32 + (index % 17) / 25;
    context.globalAlpha = depth;
    context.fillStyle = accent ? '#c95f2d' : '#17130f';
    const size = accent ? 1.8 : 1.05 + (index % 5) * 0.08;
    context.fillRect(x, y, size, size);
  }
  context.restore();
}

function wrapRevealWords() {
  document.querySelectorAll('[data-reveal]').forEach((heading) => {
    const words = heading.textContent.trim().split(/\s+/);
    const fragment = document.createDocumentFragment();
    words.forEach((word, index) => {
      const span = document.createElement('span');
      span.className = 'word';
      span.textContent = word;
      fragment.append(span);
      if (index < words.length - 1) fragment.append(' ');
    });
    heading.replaceChildren(fragment);
  });
}

function updateChapter() {
  const sceneFloat = targetProgress * (chapters.length - 1);
  const nextScene = Math.min(chapters.length - 1, Math.max(0, Math.round(sceneFloat)));

  if (nextScene !== activeScene) {
    activeScene = nextScene;
    chapters.forEach((chapter, index) => {
      const active = index === activeScene;
      chapter.classList.toggle('is-active', active);
      chapter.setAttribute('aria-hidden', String(!active));
    });
    if (sceneLabel) sceneLabel.textContent = sceneNames[activeScene];
  }

  const local = clamp(sceneFloat - activeScene + 0.54);
  const words = chapters[activeScene]?.querySelectorAll('.word') || [];
  words.forEach((word, index) => {
    word.classList.toggle('is-lit', index < Math.max(1, Math.ceil(local * words.length)));
  });

  if (progressBar) progressBar.style.transform = `scaleX(${targetProgress})`;
  if (progressLabel) progressLabel.textContent = `${String(Math.round(targetProgress * 100)).padStart(2, '0')}%`;
  document.documentElement.style.setProperty('--motion-progress', targetProgress);
}

function calculateScrollProgress() {
  if (!stage || reduceMotion.matches) return 0;
  const bounds = stage.getBoundingClientRect();
  const distance = Math.max(1, stage.offsetHeight - window.innerHeight);
  return clamp(-bounds.top / distance);
}

function animate() {
  frameRequest = undefined;
  const delta = targetProgress - currentProgress;
  currentProgress += reduceMotion.matches ? delta : delta * 0.12;
  if (Math.abs(delta) < 0.00008) currentProgress = targetProgress;
  drawFrame();
  if (Math.abs(targetProgress - currentProgress) > 0.00008) scheduleFrame();
}

function scheduleFrame() {
  if (!frameRequest) frameRequest = requestAnimationFrame(animate);
}

function onScroll() {
  targetProgress = calculateScrollProgress();
  header?.toggleAttribute('data-scrolled', window.scrollY > 18);
  updateChapter();
  scheduleFrame();
}

function resizeCanvas() {
  if (!canvas || !context) return;
  const ratio = Math.min(window.devicePixelRatio || 1, 2);
  canvas.width = Math.round(window.innerWidth * ratio);
  canvas.height = Math.round(window.innerHeight * ratio);
  canvas.style.width = `${window.innerWidth}px`;
  canvas.style.height = `${window.innerHeight}px`;
  context.setTransform(ratio, 0, 0, ratio, 0, 0);
  prepareTargets();
  currentProgress = targetProgress;
  drawFrame();
}

function onResize() {
  window.cancelAnimationFrame(resizeRequest);
  resizeRequest = requestAnimationFrame(resizeCanvas);
}

function enableReducedMotion() {
  chapters.forEach((chapter) => {
    chapter.removeAttribute('aria-hidden');
    chapter.querySelectorAll('.word').forEach((word) => word.classList.add('is-lit'));
  });
}

function closeMenu({ restoreFocus = false } = {}) {
  menuButton?.setAttribute('aria-expanded', 'false');
  navigation?.removeAttribute('data-open');
  if (restoreFocus) menuButton?.focus();
}

menuButton?.addEventListener('click', () => {
  const open = menuButton.getAttribute('aria-expanded') === 'true';
  menuButton.setAttribute('aria-expanded', String(!open));
  navigation?.toggleAttribute('data-open', !open);
});

navigation?.addEventListener('click', (event) => {
  if (event.target instanceof HTMLAnchorElement) closeMenu();
});

document.addEventListener('keydown', (event) => {
  if (event.key === 'Escape' && menuButton?.getAttribute('aria-expanded') === 'true') {
    closeMenu({ restoreFocus: true });
  }
});

window.addEventListener('pointermove', (event) => {
  if (reduceMotion.matches || window.innerWidth <= 720) return;
  pointerX = (event.clientX / window.innerWidth - 0.5) * 8;
  pointerY = (event.clientY / window.innerHeight - 0.5) * 5;
  scheduleFrame();
}, { passive: true });

window.addEventListener('scroll', onScroll, { passive: true });
window.addEventListener('resize', onResize, { passive: true });
reduceMotion.addEventListener('change', () => window.location.reload());

wrapRevealWords();

if (canvas && stage) {
  context = canvas.getContext('2d', { alpha: true });
  maskCanvas = document.createElement('canvas');
  maskCanvas.width = design.width;
  maskCanvas.height = design.height;
  maskContext = maskCanvas.getContext('2d', { willReadFrequently: true });

  if (context && maskContext && typeof maskContext.createImageData === 'function') {
    if (reduceMotion.matches) {
      enableReducedMotion();
    } else {
      resizeCanvas();
      onScroll();
    }
  } else {
    document.documentElement.classList.add('no-canvas');
    enableReducedMotion();
  }
}
