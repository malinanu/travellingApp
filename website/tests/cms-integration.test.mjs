import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import vm from 'node:vm';
import http from 'node:http';

const siteJsCode = fs.readFileSync(path.resolve('website/site.js'), 'utf-8');
const seedData = JSON.parse(fs.readFileSync(path.resolve('website/cms/seed-data.json'), 'utf-8'));

class MockElement {
  constructor(tagName) {
    this.tagName = (tagName || 'div').toUpperCase();
    this.children = [];
    this.attributes = {};
    const set = new Set();
    this.classList = {
      add: (c) => set.add(c),
      remove: (c) => set.delete(c),
      contains: (c) => set.has(c),
      toggle: (c, force) => {
        if (force === true) { set.add(c); return true; }
        if (force === false) { set.delete(c); return false; }
        if (set.has(c)) { set.delete(c); return false; }
        set.add(c); return true;
      },
      clear: () => set.clear(),
      [Symbol.iterator]: () => set[Symbol.iterator](),
    };
    this.dataset = {};
    this.style = {};
    this.childNodes = [{ nodeValue: '' }];
    this._textContent = '';
    this._innerHTML = '';
    this.hidden = false;
  }

  get className() {
    return Array.from(this.classList).join(' ');
  }

  set className(val) {
    this.classList.clear();
    (val || '').split(/\s+/).filter(Boolean).forEach((c) => this.classList.add(c));
  }

  get textContent() {
    return this._textContent;
  }

  set textContent(val) {
    this._textContent = String(val);
  }

  get innerHTML() {
    if (this._innerHTML) return this._innerHTML;
    if (this.children.length > 0) {
      return this.children.map((c) => `<${c.tagName.toLowerCase()}>${c.innerHTML}</${c.tagName.toLowerCase()}>`).join('');
    }
    return this._textContent;
  }

  set innerHTML(val) {
    this._innerHTML = String(val);
    if (!val) {
      this.children = [];
    }
  }

  setAttribute(k, v) {
    this.attributes[k] = String(v);
    if (k.startsWith('data-')) {
      const camel = k.slice(5).replace(/-([a-z])/g, (_, l) => l.toUpperCase());
      this.dataset[camel] = String(v);
    }
  }

  getAttribute(k) {
    return this.attributes[k] ?? null;
  }

  appendChild(child) {
    this.children.push(child);
    child.parentElement = this;
    return child;
  }

  prepend(child) {
    this.children.unshift(child);
    child.parentElement = this;
    return child;
  }

  get parentNode() {
    return this.parentElement;
  }

  insertBefore(newChild, refChild) {
    const idx = this.children.indexOf(refChild);
    if (idx !== -1) {
      this.children.splice(idx, 0, newChild);
    } else {
      this.children.push(newChild);
    }
    newChild.parentElement = this;
    return newChild;
  }

  replaceWith(newEl) {
    if (this.parentElement) {
      const idx = this.parentElement.children.indexOf(this);
      if (idx !== -1) this.parentElement.children[idx] = newEl;
    }
  }

  querySelector(sel) {
    const all = this.querySelectorAll(sel);
    return all.length > 0 ? all[0] : null;
  }

  querySelectorAll(sel) {
    const results = [];
    const check = (node) => {
      if (matchesSelector(node, sel)) results.push(node);
      for (const ch of node.children) check(ch);
    };
    for (const ch of this.children) check(ch);
    return results;
  }

  addEventListener(event, fn) {
    if (!this._listeners) this._listeners = {};
    if (!this._listeners[event]) this._listeners[event] = [];
    this._listeners[event].push(fn);
  }

  dispatchEvent(event) {
    const type = typeof event === 'string' ? event : event.type;
    if (this._listeners && this._listeners[type]) {
      this._listeners[type].forEach((fn) => fn({ target: this, currentTarget: this, type }));
    }
  }

  click() {
    this.dispatchEvent('click');
  }
}

function matchesSelector(node, sel) {
  if (!node || !sel) return false;
  const parts = sel.split(',').map((s) => s.trim());
  return parts.some((p) => {
    if (p.includes(' ')) {
      const subParts = p.split(/\s+/);
      const targetSub = subParts[subParts.length - 1];
      if (!matchSimple(node, targetSub)) return false;
      let ancestor = node.parentElement;
      let i = subParts.length - 2;
      while (ancestor && i >= 0) {
        if (matchSimple(ancestor, subParts[i])) i--;
        ancestor = ancestor.parentElement;
      }
      return i < 0;
    }
    return matchSimple(node, p);
  });
}

function matchSimple(node, p) {
  if (p === ':scope > div' || p === '> div') return node.tagName.toLowerCase() === 'div';
  if (p.startsWith(':scope > ')) {
    const subTag = p.slice(9).toLowerCase();
    return node.tagName.toLowerCase() === subTag;
  }
  if (p.startsWith('#')) return node.id === p.slice(1);
  const attrMatch = p.match(/^([a-zA-Z0-9-_]+)?\[([a-zA-Z0-9-_]+)(?:="([^"]+)")?\]$/);
  if (attrMatch) {
    const [, tag, attr, val] = attrMatch;
    if (tag && node.tagName.toLowerCase() !== tag.toLowerCase()) return false;
    return val !== undefined ? node.getAttribute(attr) === val : node.getAttribute(attr) !== null;
  }
  if (p.includes('.')) {
    const parts = p.split('.').filter(Boolean);
    const tag = p.startsWith('.') ? '' : parts[0];
    const classes = p.startsWith('.') ? parts : parts.slice(1);
    if (tag && node.tagName.toLowerCase() !== tag.toLowerCase()) return false;
    return classes.every((c) => node.classList.contains(c));
  }
  return node.tagName.toLowerCase() === p.toLowerCase();
}

function createMockEnvironment(pathname = '/pricing.html', search = '') {
  const body = new MockElement('body');
  const head = new MockElement('head');
  const elementsById = new Map();

  const mockDoc = {
    body,
    head,
    title: 'Initial Title',
    getElementById(id) {
      return elementsById.get(id) || null;
    },
    querySelector(sel) {
      if (sel.startsWith('#')) return elementsById.get(sel.slice(1)) || null;
      return body.querySelector(sel);
    },
    querySelectorAll(sel) {
      return body.querySelectorAll(sel);
    },
    createElement(tag) {
      return new MockElement(tag);
    },
  };

  const loc = {
    pathname,
    search,
    hostname: 'localhost',
  };

  const mockWindow = {
    document: mockDoc,
    location: loc,
    matchMedia: () => ({ matches: false }),
    SPOTTO_CMS_URL: 'http://localhost:3002',
    SPOTTO_API_URL: 'http://localhost:8080',
    fetch: async () => ({ ok: false, status: 404 }),
    Image: class {
      set src(v) { this._src = v; }
      get src() { return this._src; }
    },
    URLSearchParams,
    Number,
    String,
    Array,
    Date,
    encodeURIComponent,
    decodeURIComponent,
    setTimeout,
    clearTimeout,
    console,
  };

  mockWindow.window = mockWindow;
  mockWindow.global = mockWindow;
  mockWindow.self = mockWindow;

  return { mockWindow, mockDoc, body, elementsById };
}

test('1. Seed data integrity for 3 tiers and master features', () => {
  assert.ok(seedData.features.length >= 15, 'Master feature pool must have at least 15 features');
  assert.equal(seedData.pricingPlans.length, 3, 'Must define exactly 3 vendor plans');

  const starter = seedData.pricingPlans.find((p) => p.slug === 'starter');
  const growth = seedData.pricingPlans.find((p) => p.slug === 'growth');
  const pro = seedData.pricingPlans.find((p) => p.slug === 'pro');

  assert.ok(starter, 'Starter tier exists');
  assert.ok(growth, 'Growth tier exists');
  assert.ok(pro, 'Pro tier exists');

  assert.equal(starter.monthlyPriceUsd, 3.99, 'Starter monthly is $3.99');
  assert.equal(starter.yearlyPriceUsd, 39.9, 'Starter yearly is $39.90');

  assert.equal(growth.monthlyPriceUsd, 11.99, 'Growth monthly is $11.99');
  assert.equal(growth.yearlyPriceUsd, 119.9, 'Growth yearly is $119.90');
  assert.equal(growth.badge, 'Most Popular', 'Growth has Most Popular badge');
  assert.equal(growth.highlighted, true, 'Growth is highlighted');

  assert.equal(pro.monthlyPriceUsd, 22.99, 'Pro monthly is $22.99');
  assert.equal(pro.yearlyPriceUsd, 229.9, 'Pro yearly is $229.90');

  // Verify all 6 pages exist in seed
  assert.equal(seedData.pages.length, 6, 'Must seed home, features, businesses, pricing, about, contact');
  const pageSlugs = seedData.pages.map((p) => p.slug);
  assert.ok(pageSlugs.includes('home'));
  assert.ok(pageSlugs.includes('features'));
  assert.ok(pageSlugs.includes('businesses'));
  assert.ok(pageSlugs.includes('pricing'));
  assert.ok(pageSlugs.includes('about'));
  assert.ok(pageSlugs.includes('contact'));

  // Verify FAQs and Testimonials exist
  assert.ok(seedData.faqs.length >= 5, 'Must seed FAQs');
  assert.ok(seedData.testimonials.length >= 5, 'Must seed testimonials');
  assert.ok(seedData.stories.length >= 2, 'Must seed stories');
});

test('2. SpottoCMS client is exposed on window and provides helper methods', () => {
  const { mockWindow } = createMockEnvironment('/index.html');
  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  assert.ok(mockWindow.SpottoCMS, 'SpottoCMS object must be attached to window');
  assert.equal(typeof mockWindow.SpottoCMS.getCmsUrl, 'function');
  assert.equal(typeof mockWindow.SpottoCMS.getApiUrl, 'function');
  assert.equal(typeof mockWindow.SpottoCMS.getCurrentPageSlug, 'function');
  assert.equal(typeof mockWindow.SpottoCMS.escapeHtml, 'function');
  assert.equal(typeof mockWindow.SpottoCMS.renderMarkdown, 'function');
  assert.equal(typeof mockWindow.SpottoCMS.hydratePageContent, 'function');
  assert.equal(typeof mockWindow.SpottoCMS.hydratePricingPlans, 'function');
  assert.equal(typeof mockWindow.SpottoCMS.hydrateFAQs, 'function');
  assert.equal(typeof mockWindow.SpottoCMS.hydrateTestimonials, 'function');

  assert.equal(mockWindow.SpottoCMS.getCmsUrl(), 'http://localhost:3002');
  assert.equal(mockWindow.SpottoCMS.getApiUrl(), 'http://localhost:8080');
  assert.equal(mockWindow.SpottoCMS.getCurrentPageSlug(), 'home');
});

test('3. Escape HTML helper guards against XSS injection', () => {
  const { mockWindow } = createMockEnvironment();
  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  const esc = mockWindow.SpottoCMS.escapeHtml;
  assert.equal(esc('<script>alert("xss")</script>'), '&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;');
  assert.equal(esc('Tom & Jerry'), 'Tom &amp; Jerry');
  assert.equal(esc("It's fine"), 'It&#39;s fine');
});

test('4. Markdown renderer handles headings, bold, links, and escapes raw tags', () => {
  const { mockWindow } = createMockEnvironment();
  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  const md = mockWindow.SpottoCMS.renderMarkdown;
  const input = '# Heading\n\nThis is **bold** and [Link](https://example.com).\n\n<script>bad()</script>';
  const output = md(input, (u) => u);

  assert.ok(output.includes('<h2>Heading</h2>'), 'H1 is constrained to h2');
  assert.ok(output.includes('<strong>bold</strong>'), 'Renders bold tag');
  assert.ok(output.includes('<a href="https://example.com"'), 'Renders safe link');
  assert.ok(output.includes('&lt;script&gt;bad()&lt;/script&gt;'), 'Escapes dangerous HTML tags');
});

test('5. Pricing plans hydration renders 3 tiers and synchronizes matrix headers', async () => {
  const { mockWindow, mockDoc, elementsById, body } = createMockEnvironment('/pricing.html');

  const container = new MockElement('div');
  container.id = 'pricingPlansContainer';
  elementsById.set('pricingPlansContainer', container);
  body.appendChild(container);

  const matrixTable = new MockElement('table');
  matrixTable.id = 'pricingMatrixTable';
  const thStarter = new MockElement('th');
  thStarter.setAttribute('data-matrix-col', 'starter');
  const thGrowth = new MockElement('th');
  thGrowth.setAttribute('data-matrix-col', 'growth');
  const thPro = new MockElement('th');
  thPro.setAttribute('data-matrix-col', 'pro');
  matrixTable.appendChild(thStarter);
  matrixTable.appendChild(thGrowth);
  matrixTable.appendChild(thPro);
  elementsById.set('pricingMatrixTable', matrixTable);
  body.appendChild(matrixTable);

  mockWindow.fetch = async (url) => {
    if (url.includes('/api/pricing-plans')) {
      return {
        ok: true,
        status: 200,
        json: async () => ({ docs: seedData.pricingPlans }),
      };
    }
    return { ok: false, status: 404 };
  };

  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  mockWindow.SpottoCMS.hydratePricingPlans();

  await new Promise((resolve) => setTimeout(resolve, 80));

  assert.equal(container.children.length, 3, 'Container should hold exactly 3 plan cards');

  const starterCard = container.children[0];
  const growthCard = container.children[1];
  const proCard = container.children[2];

  assert.equal(starterCard.getAttribute('data-tier'), 'starter');
  assert.equal(growthCard.getAttribute('data-tier'), 'growth');
  assert.equal(proCard.getAttribute('data-tier'), 'pro');

  assert.ok(growthCard.classList.contains('plan--hot'), 'Growth card must have plan--hot class');
  assert.ok(growthCard.innerHTML.includes('Most Popular'), 'Growth card must display badge');
  assert.ok(growthCard.innerHTML.includes('$11.99'), 'Growth card must display $11.99 price');
  assert.ok(starterCard.innerHTML.includes('$3.99'), 'Starter card must display $3.99 price');
  assert.ok(proCard.innerHTML.includes('$22.99'), 'Pro card must display $22.99 price');

  assert.equal(thStarter.textContent, 'Starter ($3.99)');
  assert.equal(thGrowth.textContent, 'Growth ($11.99)');
  assert.equal(thPro.textContent, 'Pro ($22.99)');
});

test('6. FAQs hydration populates accordion details cleanly', async () => {
  const { mockWindow, mockDoc, body } = createMockEnvironment('/pricing.html');

  const faqContainer = new MockElement('div');
  faqContainer.className = 'faq';
  faqContainer.setAttribute('data-category', 'pricing');
  body.appendChild(faqContainer);

  mockWindow.fetch = async (url) => {
    if (url.includes('/api/faqs')) {
      return {
        ok: true,
        status: 200,
        json: async () => ({ docs: seedData.faqs }),
      };
    }
    return { ok: false, status: 404 };
  };

  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  mockWindow.SpottoCMS.hydrateFAQs('pricing');

  await new Promise((resolve) => setTimeout(resolve, 80));

  assert.ok(faqContainer.children.length >= 5, 'Should render pricing FAQs');
  const firstFaq = faqContainer.children[0];
  assert.equal(firstFaq.tagName, 'DETAILS');
  assert.ok(firstFaq.innerHTML.includes('Do travellers ever pay for Spotto?'));
});

test('7. Testimonials hydration populates primary quote and mini cards', async () => {
  const { mockWindow, mockDoc, body } = createMockEnvironment('/index.html');

  const bq = new MockElement('div');
  bq.className = 'bigquote';
  const blockquote = new MockElement('blockquote');
  blockquote.textContent = 'Old quote';
  const who = new MockElement('div');
  who.className = 'quote-who';
  const av = new MockElement('span');
  av.className = 'avatar';
  const authorB = new MockElement('b');
  const descWrap = new MockElement('span');
  const desc = new MockElement('span');
  descWrap.appendChild(desc);
  who.appendChild(av);
  who.appendChild(authorB);
  who.appendChild(descWrap);

  const minis = new MockElement('div');
  minis.className = 'quote-minis';
  const mini1 = new MockElement('div');
  mini1.appendChild(new MockElement('p'));
  const mini2 = new MockElement('div');
  mini2.appendChild(new MockElement('p'));
  minis.appendChild(mini1);
  minis.appendChild(mini2);

  bq.appendChild(blockquote);
  bq.appendChild(who);
  bq.appendChild(minis);
  body.appendChild(bq);

  mockWindow.fetch = async (url) => {
    if (url.includes('/api/testimonials')) {
      return {
        ok: true,
        status: 200,
        json: async () => ({ docs: seedData.testimonials }),
      };
    }
    return { ok: false, status: 404 };
  };

  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  mockWindow.SpottoCMS.hydrateTestimonials('home');

  await new Promise((resolve) => setTimeout(resolve, 80));

  assert.ok(blockquote.innerHTML.includes('<span class="hl">Best meal of the whole trip.</span>'), 'Highlights phrase');
  assert.equal(authorB.textContent, 'Léa Fontaine');
  assert.equal(av.textContent, 'LF');
});

test('8. Error resilience: failed CMS fetch preserves original fallback HTML', async () => {
  const { mockWindow, mockDoc, elementsById, body } = createMockEnvironment('/pricing.html');

  const container = new MockElement('div');
  container.id = 'pricingPlansContainer';
  container.innerHTML = '<div class="static-fallback-plan">Static Plan</div>';
  elementsById.set('pricingPlansContainer', container);
  body.appendChild(container);

  mockWindow.fetch = async () => {
    throw new Error('Network offline');
  };

  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  mockWindow.SpottoCMS.hydratePricingPlans();

  await new Promise((resolve) => setTimeout(resolve, 80));

  assert.equal(container.innerHTML, '<div class="static-fallback-plan">Static Plan</div>', 'Fallback content preserved without error');
});

test('9. Page & Hero hydration updates headline with highlight, eyebrow, lead, CTAs, and stats counters', async () => {
  const { mockWindow, mockDoc, body } = createMockEnvironment('/index.html');

  // Hero elements
  const hero = new MockElement('header');
  hero.className = 'hero';
  const eyebrow = new MockElement('span');
  eyebrow.className = 'eyebrow';
  eyebrow.textContent = 'Default Eyebrow';
  const h1 = new MockElement('h1');
  h1.textContent = 'Default Headline';
  const lead = new MockElement('p');
  lead.className = 'lead';
  lead.textContent = 'Default Lead';
  const ctas = new MockElement('div');
  ctas.className = 'hero-ctas';
  const primaryBtn = new MockElement('a');
  primaryBtn.className = 'btn btn--amber';
  primaryBtn.textContent = 'Default CTA';
  primaryBtn.setAttribute('href', '#default');
  ctas.appendChild(primaryBtn);

  hero.appendChild(eyebrow);
  hero.appendChild(h1);
  hero.appendChild(lead);
  hero.appendChild(ctas);
  body.appendChild(hero);

  // Stats elements
  const statsGrid = new MockElement('div');
  statsGrid.className = 'stats-grid';
  for (let i = 0; i < 4; i++) {
    const stat = new MockElement('div');
    stat.className = 'stat';
    const b = new MockElement('b');
    b.textContent = '0';
    const span = new MockElement('span');
    span.textContent = 'None';
    stat.appendChild(b);
    stat.appendChild(span);
    statsGrid.appendChild(stat);
  }
  body.appendChild(statsGrid);

  const homePageData = seedData.pages.find((p) => p.slug === 'home');
  mockWindow.fetch = async (url) => {
    if (url.includes('/api/pages')) {
      return {
        ok: true,
        status: 200,
        json: async () => ({ docs: [homePageData] }),
      };
    }
    return { ok: false, status: 404 };
  };

  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  mockWindow.SpottoCMS.hydratePageContent('home');

  await new Promise((resolve) => setTimeout(resolve, 80));

  assert.equal(eyebrow.textContent, 'Made in Sri Lanka');
  assert.ok(h1.innerHTML.includes('<span class="hl">know a local</span>'), 'Highlights key phrase in headline');
  assert.ok(lead.textContent.includes('Curated spots, real deals'));
  assert.equal(primaryBtn.textContent, 'Get the app');
  assert.equal(primaryBtn.getAttribute('href'), 'features.html#get');

  // Verify stats row counters
  const statNodes = statsGrid.querySelectorAll('.stat');
  assert.equal(statNodes[0].querySelector('b').textContent, '120+');
  assert.equal(statNodes[0].querySelector('span').textContent, 'Curated spots');
  assert.equal(statNodes[1].querySelector('b').textContent, '8,400');
  assert.equal(statNodes[1].querySelector('span').textContent, 'Deals claimed');
});

test('10. Stories data normalization handles array of tags and markdown body', () => {
  const rawStory = seedData.stories[0];
  assert.equal(rawStory.slug, 'ella-odyssey-train-guide');
  assert.ok(Array.isArray(rawStory.tags), 'Tags are present');
  assert.ok(rawStory.body.includes('### 1. Board at Peradeniya'), 'Markdown body formatted properly');
});

test('11. wrapHighlight matches case-insensitively while preserving original text casing', () => {
  const { mockWindow } = createMockEnvironment();
  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  const wrapHl = mockWindow.SpottoCMS.wrapHighlight;
  assert.equal(typeof wrapHl, 'function', 'wrapHighlight exposed on SpottoCMS');

  // Exact match
  assert.equal(
    wrapHl('Travel like you know a local.', 'know a local'),
    'Travel like you <span class="hl">know a local</span>.'
  );

  // Case-insensitive match (highlight text in uppercase)
  assert.equal(
    wrapHl('Travel like you know a local.', 'KNOW A LOCAL'),
    'Travel like you <span class="hl">know a local</span>.'
  );

  // Mixed case in headline preserved
  assert.equal(
    wrapHl('Start Free, Grow When It Works.', 'when it works'),
    'Start Free, Grow <span class="hl">When It Works</span>.'
  );

  // HTML escaping with highlight
  assert.equal(
    wrapHl('Tom & Jerry <friends>', 'Jerry'),
    'Tom &amp; <span class="hl">Jerry</span> &lt;friends&gt;'
  );
});

test('12. Auto-insertion of eyebrow tag on pages with no initial eyebrow element', async () => {
  const { mockWindow, body } = createMockEnvironment('/features.html');

  const pageHero = new MockElement('section');
  pageHero.className = 'page-hero';
  const container = new MockElement('div');
  container.className = 'container';
  const h1 = new MockElement('h1');
  h1.textContent = 'One app, the whole island.';
  container.appendChild(h1);
  pageHero.appendChild(container);
  body.appendChild(pageHero);

  // Initially NO eyebrow element exists
  assert.equal(pageHero.querySelector('.eyebrow'), null);

  const featuresPageData = seedData.pages.find((p) => p.slug === 'features');
  mockWindow.fetch = async (url) => {
    if (url.includes('/api/pages')) {
      return {
        ok: true,
        status: 200,
        json: async () => ({ docs: [featuresPageData] }),
      };
    }
    return { ok: false, status: 404 };
  };

  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  mockWindow.SpottoCMS.hydratePageContent('features');
  await new Promise((resolve) => setTimeout(resolve, 80));

  const insertedEyebrow = pageHero.querySelector('.eyebrow');
  assert.ok(insertedEyebrow, 'Eyebrow tag must be auto-inserted before H1');
  assert.equal(insertedEyebrow.textContent, 'Traveller Experience');
  assert.equal(container.children[0], insertedEyebrow, 'Eyebrow must be first child before H1');
  assert.equal(container.children[1], h1, 'H1 must follow eyebrow');
});

test('13. Testimonials hydration dynamically creates mini cards when CMS returns > 2 items', async () => {
  const { mockWindow, body } = createMockEnvironment('/businesses.html');

  const bq = new MockElement('div');
  bq.className = 'bigquote';
  const blockquote = new MockElement('blockquote');
  blockquote.textContent = 'Old quote';
  const who = new MockElement('div');
  who.className = 'quote-who';
  who.appendChild(new MockElement('span'));
  bq.appendChild(blockquote);
  bq.appendChild(who);

  // Container with 0 static minis initially
  const minisContainer = new MockElement('div');
  minisContainer.className = 'quote-minis';
  bq.appendChild(minisContainer);
  body.appendChild(bq);

  const bizTestimonials = seedData.testimonials.filter((t) => t.page === 'businesses');
  mockWindow.fetch = async (url) => {
    if (url.includes('/api/testimonials')) {
      return {
        ok: true,
        status: 200,
        json: async () => ({ docs: bizTestimonials }),
      };
    }
    return { ok: false, status: 404 };
  };

  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  mockWindow.SpottoCMS.hydrateTestimonials('businesses');
  await new Promise((resolve) => setTimeout(resolve, 80));

  assert.equal(minisContainer.children.length, 2, 'Should create 2 mini cards dynamically');
  assert.ok(minisContainer.children[0].innerHTML.includes('Ishara Fernando'));
  assert.ok(minisContainer.children[1].innerHTML.includes('Kasun Jayasuriya'));
});

test('14. CTA band and Split Feature sections hydration from CMS sections', async () => {
  const { mockWindow, body } = createMockEnvironment('/index.html');

  // CTA band
  const band = new MockElement('section');
  band.className = 'band band--center';
  const bandH2 = new MockElement('h2');
  bandH2.textContent = 'Old band heading';
  const bandP = new MockElement('p');
  bandP.textContent = 'Old band subcopy';
  const bandBtn = new MockElement('a');
  bandBtn.className = 'btn btn--paper';
  bandBtn.textContent = 'Old CTA';
  bandBtn.setAttribute('href', '#old');
  band.appendChild(bandH2);
  band.appendChild(bandP);
  band.appendChild(bandBtn);
  body.appendChild(band);

  // Split feature
  const splitSec = new MockElement('div');
  splitSec.className = 'container split';
  const splitH2 = new MockElement('h2');
  splitH2.textContent = 'Old split title';
  const splitBody = new MockElement('p');
  splitBody.className = 'muted';
  splitBody.textContent = 'Old split body';
  const ticksUl = new MockElement('ul');
  ticksUl.className = 'ticks';
  const tick1 = new MockElement('li');
  tick1.textContent = 'Old tick 1';
  ticksUl.appendChild(tick1);
  splitSec.appendChild(splitH2);
  splitSec.appendChild(splitBody);
  splitSec.appendChild(ticksUl);
  body.appendChild(splitSec);

  const customPageData = {
    slug: 'home',
    title: 'Custom Title',
    sections: [
      {
        blockType: 'ctaBand',
        headline: 'New CMS CTA Band Headline',
        subcopy: 'New CMS subcopy for travellers.',
        buttonText: 'Claim Your Spot',
        buttonLink: 'businesses.html#signup',
      },
      {
        blockType: 'splitFeature',
        title: 'Fresh CMS Split Title',
        body: 'Fresh CMS description paragraph.',
        bullets: [{ text: 'Dynamic bullet point 1' }],
      },
    ],
  };

  mockWindow.fetch = async (url) => {
    if (url.includes('/api/pages')) {
      return {
        ok: true,
        status: 200,
        json: async () => ({ docs: [customPageData] }),
      };
    }
    return { ok: false, status: 404 };
  };

  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  mockWindow.SpottoCMS.hydratePageContent('home');
  await new Promise((resolve) => setTimeout(resolve, 80));

  assert.equal(bandH2.textContent, 'New CMS CTA Band Headline');
  assert.equal(bandP.textContent, 'New CMS subcopy for travellers.');
  assert.equal(bandBtn.textContent, 'Claim Your Spot');
  assert.equal(bandBtn.getAttribute('href'), 'businesses.html#signup');

  assert.equal(splitH2.textContent, 'Fresh CMS Split Title');
  assert.equal(splitBody.textContent, 'Fresh CMS description paragraph.');
  assert.equal(tick1.textContent, 'Dynamic bullet point 1');
});

test('15. Multi-container FAQ hydration with separate categories', async () => {
  const { mockWindow, body } = createMockEnvironment('/features.html');

  const faqContainer1 = new MockElement('div');
  faqContainer1.className = 'faq';
  faqContainer1.setAttribute('data-category', 'pricing');
  body.appendChild(faqContainer1);

  const faqContainer2 = new MockElement('div');
  faqContainer2.className = 'faq';
  faqContainer2.setAttribute('data-category', 'businesses');
  body.appendChild(faqContainer2);

  mockWindow.fetch = async (url) => {
    if (url.includes('/api/faqs')) {
      return {
        ok: true,
        status: 200,
        json: async () => ({ docs: seedData.faqs }),
      };
    }
    return { ok: false, status: 404 };
  };

  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  mockWindow.SpottoCMS.hydrateFAQs('features');
  await new Promise((resolve) => setTimeout(resolve, 80));

  assert.ok(faqContainer1.children.length >= 4, 'Pricing FAQ container hydrated');
  assert.ok(faqContainer2.children.length >= 1, 'Businesses FAQ container hydrated');
  assert.ok(faqContainer1.children[0].innerHTML.includes('Do travellers ever pay for Spotto?'));
  assert.ok(faqContainer2.children[0].innerHTML.includes('How fast does my listing go live?'));
});

test('16. Local file:// protocol detection and fallback behavior', () => {
  const { mockWindow } = createMockEnvironment();
  mockWindow.location = {
    protocol: 'file:',
    pathname: 'C:/dev/personal/TravelApp/website/index.html',
    hostname: '',
    search: '',
  };

  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  assert.equal(mockWindow.SpottoCMS.getCmsUrl(), 'http://localhost:3002');
  assert.equal(mockWindow.SpottoCMS.getApiUrl(), 'http://localhost:8080');
});

test('17. End-to-end REST API client simulation for Payload CMS pricing and pages', async () => {
  const reqLog = [];
  const fakePayloadServer = async (url, options = {}) => {
    reqLog.push({ url, headers: options.headers });
    if (url.includes('/api/pricing-plans')) {
      return {
        ok: true,
        status: 200,
        json: async () => ({ docs: seedData.pricingPlans }),
      };
    }
    if (url.includes('/api/pages')) {
      return {
        ok: true,
        status: 200,
        json: async () => ({ docs: [seedData.pages[0]] }),
      };
    }
    return { ok: false, status: 404 };
  };

  const { mockWindow, body, elementsById } = createMockEnvironment('/pricing.html');
  mockWindow.SPOTTO_CMS_URL = 'http://localhost:3002';
  mockWindow.fetch = fakePayloadServer;

  const plansContainer = new MockElement('div');
  plansContainer.id = 'pricingPlansContainer';
  elementsById.set('pricingPlansContainer', plansContainer);
  body.appendChild(plansContainer);

  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  mockWindow.SpottoCMS.hydratePricingPlans();
  await new Promise((resolve) => setTimeout(resolve, 80));

  assert.equal(plansContainer.children.length, 3, 'Payload CMS response must hydrate exactly 3 plans');
  assert.ok(plansContainer.children[0].innerHTML.includes('$3.99'));
  assert.ok(plansContainer.children[1].innerHTML.includes('$11.99'));
  assert.ok(plansContainer.children[2].innerHTML.includes('$22.99'));

  // Verify headers and endpoints
  assert.ok(reqLog.length > 0, 'Must have made HTTP request');
  const planReq = reqLog.find((r) => r.url.includes('/api/pricing-plans'));
  assert.ok(planReq, 'Must query /api/pricing-plans');
  assert.equal(planReq.headers['Accept'], 'application/json');
});

test('18. Sync bridge preserves features relationship in payloadDoc', () => {
  const bridgeCode = fs.readFileSync(path.resolve('website/cms/sync-bridge.ts'), 'utf-8');
  assert.ok(bridgeCode.includes('features: pkg.features || []'), 'Sync bridge must include features in payloadDoc');
});

test('19. Dynamic multi-tier rendering: 4 tiers created by Super Admin renders 4 cards with data-count=4', async () => {
  const { mockWindow, body, elementsById } = createMockEnvironment('/pricing.html');

  const container = new MockElement('div');
  container.id = 'pricingPlansContainer';
  elementsById.set('pricingPlansContainer', container);
  body.appendChild(container);

  const fourTiers = [
    {
      id: 'pkg_starter',
      slug: 'starter',
      name: 'Starter',
      tagline: 'Basic pin',
      monthlyPriceUsd: 3.99,
      yearlyPriceUsd: 39.90,
      highlighted: false,
      features: [{ id: 'feat_map_pin', name: 'Verified Map Pin' }],
    },
    {
      id: 'pkg_growth',
      slug: 'growth',
      name: 'Growth',
      tagline: 'Busy spots',
      monthlyPriceUsd: 11.99,
      yearlyPriceUsd: 119.90,
      badge: 'Most Popular',
      highlighted: true,
      features: [
        { id: 'feat_map_pin', name: 'Verified Map Pin' },
        { id: 'feat_stamp_cards', name: 'Digital Stamp Cards' },
      ],
    },
    {
      id: 'pkg_pro',
      slug: 'pro',
      name: 'Pro',
      tagline: 'Premier venues',
      monthlyPriceUsd: 22.99,
      yearlyPriceUsd: 229.90,
      badge: 'Best Value',
      highlighted: false,
      features: [
        { id: 'feat_map_pin', name: 'Verified Map Pin' },
        { id: 'feat_stamp_cards', name: 'Digital Stamp Cards' },
        { id: 'feat_geo_heatmap', name: 'Geo Demand Heatmaps' },
      ],
    },
    {
      id: 'pkg_enterprise',
      slug: 'enterprise',
      name: 'Enterprise',
      tagline: 'Resort chains & multi-locations',
      monthlyPriceUsd: 49.99,
      yearlyPriceUsd: 499.90,
      badge: 'VIP Exclusive',
      highlighted: true,
      features: [
        { id: 'feat_map_pin', name: 'Verified Map Pin' },
        { id: 'feat_stamp_cards', name: 'Digital Stamp Cards' },
        { id: 'feat_geo_heatmap', name: 'Geo Demand Heatmaps' },
        { id: 'feat_white_glove', name: 'White-Glove Support' },
      ],
    },
  ];

  mockWindow.fetch = async (url) => {
    if (url.includes('/packages')) {
      return {
        ok: true,
        status: 200,
        json: async () => ({ packages: fourTiers }),
      };
    }
    return { ok: false, status: 404 };
  };

  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  mockWindow.SpottoCMS.hydratePricingPlans();
  await new Promise((resolve) => setTimeout(resolve, 80));

  assert.equal(container.getAttribute('data-count'), '4', 'Container must have data-count="4" for 4 packages');
  assert.equal(container.children.length, 4, 'Must render exactly 4 plan cards');

  const [c0, c1, c2, c3] = container.children;
  assert.equal(c0.getAttribute('data-tier'), 'starter');
  assert.equal(c1.getAttribute('data-tier'), 'growth');
  assert.equal(c2.getAttribute('data-tier'), 'pro');
  assert.equal(c3.getAttribute('data-tier'), 'enterprise');

  assert.ok(c3.innerHTML.includes('Enterprise'), '4th card must display Enterprise title');
  assert.ok(c3.innerHTML.includes('$49.99'), '4th card must display $49.99 monthly price');
  assert.ok(c3.innerHTML.includes('VIP Exclusive'), '4th card must display VIP Exclusive badge');
  assert.ok(c3.classList.contains('plan--hot'), '4th card with highlighted: true must have plan--hot class');
});

test('20. Dynamic Matrix Regeneration: renders columns matching exact shuffled package order', async () => {
  const { mockWindow, body, elementsById } = createMockEnvironment('/pricing.html');

  const container = new MockElement('div');
  container.id = 'pricingPlansContainer';
  elementsById.set('pricingPlansContainer', container);
  body.appendChild(container);

  const matrixTable = new MockElement('table');
  matrixTable.id = 'pricingMatrixTable';
  elementsById.set('pricingMatrixTable', matrixTable);
  body.appendChild(matrixTable);

  // Shuffled packages: Pro first, Enterprise second, Growth third, Starter last
  const shuffledTiers = [
    {
      id: 'pkg_pro',
      slug: 'pro',
      name: 'Pro',
      monthlyPriceUsd: 22.99,
      highlighted: false,
      features: [{ id: 'feat_map_pin', name: 'Verified Map Pin' }],
    },
    {
      id: 'pkg_enterprise',
      slug: 'enterprise',
      name: 'Enterprise',
      monthlyPriceUsd: 49.99,
      highlighted: true,
      features: [{ id: 'feat_map_pin', name: 'Verified Map Pin' }],
    },
    {
      id: 'pkg_growth',
      slug: 'growth',
      name: 'Growth',
      monthlyPriceUsd: 11.99,
      highlighted: false,
      features: [{ id: 'feat_map_pin', name: 'Verified Map Pin' }],
    },
    {
      id: 'pkg_starter',
      slug: 'starter',
      name: 'Starter',
      monthlyPriceUsd: 3.99,
      highlighted: false,
      features: [{ id: 'feat_map_pin', name: 'Verified Map Pin' }],
    },
  ];

  mockWindow.fetch = async (url) => {
    if (url.includes('/packages')) {
      return {
        ok: true,
        status: 200,
        json: async () => ({ packages: shuffledTiers }),
      };
    }
    return { ok: false, status: 404 };
  };

  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  mockWindow.SpottoCMS.hydratePricingPlans();
  await new Promise((resolve) => setTimeout(resolve, 80));

  // Check card container ordering
  assert.equal(container.children.length, 4);
  assert.equal(container.children[0].getAttribute('data-tier'), 'pro');
  assert.equal(container.children[1].getAttribute('data-tier'), 'enterprise');
  assert.equal(container.children[2].getAttribute('data-tier'), 'growth');
  assert.equal(container.children[3].getAttribute('data-tier'), 'starter');

  // Check matrix table header columns
  const thead = matrixTable.querySelector('thead');
  assert.ok(thead, 'Matrix table must have rebuilt thead');
  const thList = thead.querySelectorAll('th');
  assert.equal(thList.length, 5, 'Must have 1 capability header + 4 package headers');

  assert.equal(thList[0].textContent, 'Platform Capability');
  assert.equal(thList[1].getAttribute('data-matrix-col'), 'pro');
  assert.equal(thList[1].textContent, 'Pro ($22.99)');

  assert.equal(thList[2].getAttribute('data-matrix-col'), 'enterprise');
  assert.equal(thList[2].textContent, 'Enterprise ($49.99)');
  assert.ok(thList[2].classList.contains('matrix-th-hl'), 'Highlighted Enterprise tier must have matrix-th-hl class');

  assert.equal(thList[3].getAttribute('data-matrix-col'), 'growth');
  assert.equal(thList[3].textContent, 'Growth ($11.99)');

  assert.equal(thList[4].getAttribute('data-matrix-col'), 'starter');
  assert.equal(thList[4].textContent, 'Starter ($3.99)');

  // Verify accessible label
  assert.equal(matrixTable.getAttribute('aria-label'), 'Feature comparison across Pro, Enterprise, Growth, Starter tiers');
});

test('21. Dynamic Matrix Checkmarks & Crossmarks accurately reflect feature allocations and custom features', async () => {
  const { mockWindow, body, elementsById } = createMockEnvironment('/pricing.html');

  const container = new MockElement('div');
  container.id = 'pricingPlansContainer';
  elementsById.set('pricingPlansContainer', container);
  body.appendChild(container);

  const matrixTable = new MockElement('table');
  matrixTable.id = 'pricingMatrixTable';
  elementsById.set('pricingMatrixTable', matrixTable);
  body.appendChild(matrixTable);

  const tiers = [
    {
      id: 'pkg_starter',
      slug: 'starter',
      name: 'Starter',
      monthlyPriceUsd: 3.99,
      activeOffers: 1,
      flashSalesPerMonth: 0,
      teamSeats: 1,
      features: [{ id: 'feat_map_pin', name: 'Verified Map Pin' }],
    },
    {
      id: 'pkg_growth',
      slug: 'growth',
      name: 'Growth',
      monthlyPriceUsd: 11.99,
      activeOffers: 5,
      flashSalesPerMonth: 2,
      teamSeats: 3,
      features: [
        { id: 'feat_map_pin', name: 'Verified Map Pin' },
        { id: 'feat_stamp_cards', name: 'Digital Stamp Cards' },
        { id: 'feat_custom_perk', name: 'Custom Secret Perk' },
      ],
    },
  ];

  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  mockWindow.SpottoCMS.renderDynamicPackages(tiers, null);

  const tbody = matrixTable.querySelector('tbody');
  assert.ok(tbody, 'Tbody must be created');
  const trs = tbody.querySelectorAll('tr');
  assert.ok(trs.length > 5, 'Tbody must have multiple category and capability rows');

  // Find the row for Digital Stamp Cards
  const stampRow = trs.find((tr) => tr.children[0] && tr.children[0].textContent.includes('Digital Stamp Cards'));
  assert.ok(stampRow, 'Must have Digital Stamp Cards row');

  // Starter does NOT have stamp cards -> crossmark
  assert.ok(stampRow.children[1].innerHTML.includes('matrix-cross'), 'Starter should have crossmark for stamp cards');
  // Growth DOES have stamp cards -> checkmark
  assert.ok(stampRow.children[2].innerHTML.includes('matrix-check'), 'Growth should have checkmark for stamp cards');

  // Find the custom perk row
  const customRow = trs.find((tr) => tr.children[0] && tr.children[0].textContent.includes('Custom Secret Perk'));
  assert.ok(customRow, 'Must dynamically render row for custom perk');
  assert.ok(customRow.children[1].innerHTML.includes('matrix-cross'), 'Starter should have crossmark for custom perk');
  assert.ok(customRow.children[2].innerHTML.includes('matrix-check'), 'Growth should have checkmark for custom perk');
});

test('22. Dynamic Matrix adapts when packages are reduced or deleted, updating columns and data-count', async () => {
  const { mockWindow, body, elementsById } = createMockEnvironment('/pricing.html');

  const container = new MockElement('div');
  container.id = 'pricingPlansContainer';
  elementsById.set('pricingPlansContainer', container);
  body.appendChild(container);

  const matrixTable = new MockElement('table');
  matrixTable.id = 'pricingMatrixTable';
  elementsById.set('pricingMatrixTable', matrixTable);
  body.appendChild(matrixTable);

  const singleTier = [
    {
      id: 'pkg_all_inclusive',
      slug: 'all-inclusive',
      name: 'All Inclusive Pass',
      monthlyPriceUsd: 19.99,
      features: [{ id: 'feat_map_pin', name: 'Verified Map Pin' }],
    },
  ];

  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  mockWindow.SpottoCMS.renderDynamicPackages(singleTier, null);

  assert.equal(container.getAttribute('data-count'), '1', 'Container data-count must be 1');
  assert.equal(container.children.length, 1);

  const thead = matrixTable.querySelector('thead');
  const thList = thead.querySelectorAll('th');
  assert.equal(thList.length, 2, 'Should have 1 capability header + 1 package header');
  assert.equal(thList[1].getAttribute('data-matrix-col'), 'all-inclusive');
  assert.equal(thList[1].textContent, 'All Inclusive Pass ($19.99)');
});

test('23. Dynamic Grid & Matrix handles 6+ package tiers cleanly', () => {
  const { mockWindow, body, elementsById } = createMockEnvironment('/pricing.html');

  const container = new MockElement('div');
  container.id = 'pricingPlansContainer';
  elementsById.set('pricingPlansContainer', container);
  body.appendChild(container);

  const matrixTable = new MockElement('table');
  matrixTable.id = 'pricingMatrixTable';
  elementsById.set('pricingMatrixTable', matrixTable);
  body.appendChild(matrixTable);

  const sixTiers = [
    { id: 'p1', slug: 'tier1', name: 'Tier 1', monthlyPriceUsd: 1, features: [] },
    { id: 'p2', slug: 'tier2', name: 'Tier 2', monthlyPriceUsd: 2, features: [] },
    { id: 'p3', slug: 'tier3', name: 'Tier 3', monthlyPriceUsd: 3, features: [] },
    { id: 'p4', slug: 'tier4', name: 'Tier 4', monthlyPriceUsd: 4, features: [] },
    { id: 'p5', slug: 'tier5', name: 'Tier 5', monthlyPriceUsd: 5, features: [] },
    { id: 'p6', slug: 'tier6', name: 'Tier 6', monthlyPriceUsd: 6, features: [] },
  ];

  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  mockWindow.SpottoCMS.renderDynamicPackages(sixTiers, null);

  assert.equal(container.getAttribute('data-count'), '6', 'Container data-count must be 6');
  assert.equal(container.children.length, 6, 'Should render 6 cards');

  const thead = matrixTable.querySelector('thead');
  const thList = thead.querySelectorAll('th');
  assert.equal(thList.length, 7, 'Must have 1 capability header + 6 package headers');
  assert.equal(thList[6].getAttribute('data-matrix-col'), 'tier6');
});

test('24. Billing toggle switching correctly recalculates yearly prices and free tiers', () => {
  const { mockWindow, body, elementsById } = createMockEnvironment('/pricing.html');

  const seg = new MockElement('div');
  seg.id = 'billingToggle';
  const monthlyBtn = new MockElement('button');
  monthlyBtn.setAttribute('data-billing', 'monthly');
  monthlyBtn.className = 'on';
  const yearlyBtn = new MockElement('button');
  yearlyBtn.setAttribute('data-billing', 'yearly');
  seg.appendChild(monthlyBtn);
  seg.appendChild(yearlyBtn);
  elementsById.set('billingToggle', seg);
  body.appendChild(seg);

  const container = new MockElement('div');
  container.id = 'pricingPlansContainer';
  elementsById.set('pricingPlansContainer', container);
  body.appendChild(container);

  const tiers = [
    {
      id: 'pkg_free',
      slug: 'free',
      name: 'Community',
      monthlyPriceUsd: 0,
      yearlyPriceUsd: 0,
      features: [{ id: 'feat_map_pin', name: 'Verified Map Pin' }],
    },
    {
      id: 'pkg_pro',
      slug: 'pro',
      name: 'Pro',
      monthlyPriceUsd: 20,
      yearlyPriceUsd: 200,
      features: [{ id: 'feat_map_pin', name: 'Verified Map Pin' }],
    },
  ];

  const context = vm.createContext(mockWindow);
  vm.runInContext(siteJsCode, context);

  mockWindow.SpottoCMS.renderDynamicPackages(tiers, null);

  const freeCard = container.children[0];
  const proCard = container.children[1];

  // Initially monthly
  assert.ok(freeCard.innerHTML.includes('$0.00'), 'Free tier must format as $0.00');
  assert.ok(proCard.innerHTML.includes('$20.00'), 'Pro tier must format as $20.00');

  // Trigger yearly billing button click
  yearlyBtn.dispatchEvent('click');

  // After yearly toggle: Pro yearly rate should be $200 / 12 = $16.67/mo
  assert.ok(proCard.innerHTML.includes('$16.67'), 'Pro yearly rate must switch to $16.67');
  assert.ok(freeCard.innerHTML.includes('$0.00'), 'Free tier yearly remains $0.00');
});



