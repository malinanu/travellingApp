/* Spotto — Golden Hour site JS. Class toggles and small behaviors only. */
(function () {
  'use strict';

  var reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  /* ---------- Mobile nav ---------- */
  var nav = document.querySelector('.nav');
  var burger = document.querySelector('.nav-burger');
  if (nav && burger) {
    burger.addEventListener('click', function () {
      var open = nav.classList.toggle('nav-open');
      burger.setAttribute('aria-expanded', open ? 'true' : 'false');
    });
    nav.querySelectorAll('.nav-links a').forEach(function (a) {
      a.addEventListener('click', function () {
        nav.classList.remove('nav-open');
        burger.setAttribute('aria-expanded', 'false');
      });
    });
  }

  /* ---------- Scroll reveal ---------- */
  var revealEls = document.querySelectorAll('.rv');
  if (revealEls.length) {
    if (reduceMotion || !('IntersectionObserver' in window)) {
      revealEls.forEach(function (el) { el.classList.add('in'); });
    } else {
      var io = new IntersectionObserver(function (entries) {
        entries.forEach(function (entry) {
          if (entry.isIntersecting) {
            entry.target.classList.add('in');
            io.unobserve(entry.target);
          }
        });
      }, { threshold: 0.15 });
      revealEls.forEach(function (el) { io.observe(el); });
    }
  }

  /* ---------- Photo hydration ----------
     Every .photo[data-img] names its real image file. If the file exists
     (generated or supplied later), it loads in place of the art panel. */
  document.querySelectorAll('.photo[data-img]').forEach(function (fig) {
    var src = fig.getAttribute('data-img');
    var probe = new Image();
    probe.onload = function () {
      var img = document.createElement('img');
      img.src = src;
      img.alt = fig.getAttribute('data-alt') || '';
      if (fig.classList.contains('hero-img')) img.className = reduceMotion ? '' : 'kb';
      fig.prepend(img);
    };
    probe.src = src;
  });

  /* ---------- 8 Wonders rail: arrows + drag ---------- */
  var rail = document.getElementById('rail');
  if (rail) {
    var cardWidth = function () {
      var card = rail.querySelector('.wcard');
      return card ? card.getBoundingClientRect().width + 22 : 340;
    };
    var prev = document.getElementById('railPrev');
    var next = document.getElementById('railNext');
    if (prev) prev.addEventListener('click', function () { rail.scrollBy({ left: -cardWidth(), behavior: reduceMotion ? 'auto' : 'smooth' }); });
    if (next) next.addEventListener('click', function () { rail.scrollBy({ left: cardWidth(), behavior: reduceMotion ? 'auto' : 'smooth' }); });

    var down = false, startX = 0, startScroll = 0, moved = false;
    rail.addEventListener('pointerdown', function (e) {
      if (e.pointerType !== 'mouse') return;
      down = true; moved = false;
      startX = e.clientX;
      startScroll = rail.scrollLeft;
    });
    window.addEventListener('pointermove', function (e) {
      if (!down) return;
      var dx = e.clientX - startX;
      if (Math.abs(dx) > 4) {
        moved = true;
        rail.classList.add('dragging');
        rail.scrollLeft = startScroll - dx;
      }
    });
    window.addEventListener('pointerup', function () {
      down = false;
      rail.classList.remove('dragging');
    });
    rail.addEventListener('click', function (e) {
      if (moved) { e.preventDefault(); moved = false; }
    }, true);
  }

  /* ---------- Pricing toggle ---------- */
  var seg = document.getElementById('billingToggle');
  if (seg) {
    var prices = document.querySelectorAll('[data-monthly]');
    var notes = document.querySelectorAll('[data-note-monthly]');
    seg.querySelectorAll('button').forEach(function (btn) {
      btn.addEventListener('click', function () {
        var yearly = btn.dataset.billing === 'yearly';
        seg.querySelectorAll('button').forEach(function (b) {
          b.classList.toggle('on', b === btn);
          b.setAttribute('aria-pressed', b === btn ? 'true' : 'false');
        });
        prices.forEach(function (p) {
          p.childNodes[0].nodeValue = yearly ? p.dataset.yearly : p.dataset.monthly;
        });
        notes.forEach(function (n) {
          n.textContent = yearly ? n.dataset.noteYearly : n.dataset.noteMonthly;
        });
      });
    });
  }

  /* ---------- Forms: success swap ---------- */
  document.querySelectorAll('form.js-form').forEach(function (form) {
    form.addEventListener('submit', function (e) {
      e.preventDefault();
      var box = document.createElement('div');
      box.className = 'form-success';
      box.innerHTML = '<h3></h3><p></p>';
      box.querySelector('h3').textContent = form.dataset.successTitle || 'Got it.';
      box.querySelector('p').textContent = form.dataset.successBody || 'Thanks, we will be in touch soon.';
      if (form.closest('.band')) {
        box.style.color = 'inherit';
        box.querySelector('p').style.color = 'rgba(245,240,230,0.75)';
        box.style.padding = '20px 0 0';
      }
      form.replaceWith(box);
    });
  });
  /* ---------- Stories (blog.html) ----------
     Published blog posts come from the Spotto API (GET /content?type=blog):
     written in the Command Center and approved by an admin. blog.html?post=
     <slug> shows one post. Text is always set as text; only the Markdown
     renderer below produces HTML, and it escapes everything first. */
  var stories = document.getElementById('stories');
  if (stories) {
    var query = new URLSearchParams(location.search);
    var api = (stories.getAttribute('data-api') || '').replace(/\/$/, '');
    // A local copy of the site can read a local API: blog.html?api=http://localhost:8080
    var localApi = /^(localhost|127\.0\.0\.1)$/.test(location.hostname) ? query.get('api') : null;
    if (localApi) api = localApi.replace(/\/$/, '');
    var mediaUrl = function (u) { return u && u.charAt(0) === '/' && u.charAt(1) !== '/' ? api + u : u; };
    var storyHref = function (slug) {
      return 'blog.html?post=' + encodeURIComponent(slug) + (localApi ? '&api=' + encodeURIComponent(localApi) : '');
    };
    var palettes = ['im-sigiriya', 'im-arugam', 'im-ember', 'im-galle', 'im-yala', 'im-peak', 'im-ella'];
    var el = function (tag, cls, text) {
      var n = document.createElement(tag);
      if (cls) n.className = cls;
      if (text) n.textContent = text;
      return n;
    };
    var category = function (post) {
      var t = (post.tags && post.tags[0]) || 'Story';
      return t.charAt(0).toUpperCase() + t.slice(1);
    };
    var meta = function (post) {
      var parts = [];
      if (post.authorName) parts.push(post.authorName);
      if (post.publishedAt) parts.push(new Date(post.publishedAt).toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' }));
      if (post.regionName) parts.push(post.regionName);
      return parts.join(' · ');
    };
    var cover = function (post, i, cls) {
      var fig = el('figure', 'photo ' + (cls || '') + ' ' + palettes[i % palettes.length]);
      if (post.coverUrl) {
        var img = el('img');
        img.src = mediaUrl(post.coverUrl);
        img.alt = '';
        img.loading = 'lazy';
        fig.appendChild(img);
      }
      return fig;
    };
    var titleLink = function (post) {
      var h = el('h3');
      var a = el('a', null, post.title);
      a.href = storyHref(post.slug);
      h.appendChild(a);
      return h;
    };
    var get = function (path) {
      return fetch(api + path, { headers: { Accept: 'application/json' } }).then(function (res) {
        if (!res.ok) { var err = new Error('HTTP ' + res.status); err.status = res.status; throw err; }
        return res.json();
      });
    };

    var statusEl = document.getElementById('storiesStatus');
    var featuredEl = document.getElementById('storiesFeatured');
    var gridEl = document.getElementById('storiesGrid');
    var moreBtn = document.getElementById('storiesMore');
    var offset = 0;
    var shown = 0;
    var PAGE = 13; // the featured story plus four rows of three

    var addPost = function (post) {
      if (shown === 0) {
        var feat = el('div', 'featured');
        var link = el('a');
        link.href = storyHref(post.slug);
        link.setAttribute('aria-label', post.title);
        link.appendChild(cover(post, 0));
        link.style.display = 'contents';
        feat.appendChild(link);
        var body = el('div', 'featured-body');
        body.appendChild(el('span', 'cat', category(post)));
        body.appendChild(titleLink(post));
        if (post.summary) {
          var p = el('p', 'muted', post.summary);
          p.style.fontWeight = '500';
          body.appendChild(p);
        }
        body.appendChild(el('span', 'meta', meta(post)));
        feat.appendChild(body);
        featuredEl.appendChild(feat);
      } else {
        var card = el('div', 'post');
        var a = el('a');
        a.href = storyHref(post.slug);
        a.setAttribute('aria-label', post.title);
        a.appendChild(cover(post, shown));
        card.appendChild(a);
        card.appendChild(el('span', 'cat', category(post)));
        card.appendChild(titleLink(post));
        card.appendChild(el('p', null, post.summary || ''));
        card.appendChild(el('span', 'meta', meta(post)));
        gridEl.appendChild(card);
      }
      shown++;
    };

    var loadPage = function () {
      moreBtn.disabled = true;
      get('/content?type=blog&limit=' + PAGE + '&offset=' + offset).then(function (res) {
        (res.items || []).forEach(addPost);
        offset += (res.items || []).length;
        statusEl.textContent = shown ? '' : 'No stories yet. Check back soon.';
        moreBtn.hidden = !res.hasMore;
        moreBtn.disabled = false;
      }).catch(function () {
        statusEl.textContent = "Stories couldn't load right now. Please try again in a moment.";
        moreBtn.disabled = false;
      });
    };

    var showStory = function (slug) {
      stories.hidden = true;
      var hero = document.getElementById('storiesHero');
      if (hero) hero.hidden = true;
      var article = document.getElementById('story');
      article.hidden = false;
      var titleEl = document.getElementById('storyTitle');
      titleEl.textContent = 'Loading…';
      get('/content/' + encodeURIComponent(slug)).then(function (post) {
        document.title = post.title + ' · Spotto';
        document.getElementById('storyCat').textContent = category(post);
        titleEl.textContent = post.title;
        document.getElementById('storySummary').textContent = post.summary || '';
        document.getElementById('storyMeta').textContent = meta(post);
        if (post.coverUrl) {
          var fig = document.getElementById('storyCover');
          var img = el('img');
          img.src = mediaUrl(post.coverUrl);
          img.alt = '';
          fig.appendChild(img);
          fig.hidden = false;
        }
        document.getElementById('storyBody').innerHTML = renderMarkdown(post.body || '', mediaUrl);
        var tags = document.getElementById('storyTags');
        (post.tags || []).forEach(function (t) { tags.appendChild(el('span', 'story-tag', '#' + t)); });
      }).catch(function (err) {
        titleEl.textContent = err && err.status === 404 ? 'This story isn’t available.' : "This story couldn't load right now.";
        document.getElementById('storySummary').textContent = err && err.status === 404
          ? 'It may have been taken down. Have a look at our other stories.'
          : 'Please try again in a moment.';
      });
    };

    moreBtn.addEventListener('click', loadPage);
    if (query.get('post')) showStory(query.get('post'));
    else loadPage();
  }

  /* Markdown for story bodies: headings, paragraphs, **bold**, *italic*,
     `code`, [links](url), ![images](url), lists, > quotes and ---, nothing
     else. Escapes all text before adding tags and never passes raw HTML
     through; links and images keep only http(s), mailto and /… URLs. A copy
     of web/packages/shared/src/markdown.ts (the Command Center's preview):
     keep the two in step. */
  function renderMarkdown(md, resolve) {
    var esc = function (s) {
      return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;');
    };
    var safeUrl = function (url) {
      if (/^(https?:\/\/|mailto:)/i.test(url)) return url;
      if (url.charAt(0) === '/' && url.charAt(1) !== '/') return resolve(url);
      return null;
    };
    var inline = function (text) {
      return text.split(/(`[^`]+`)/).map(function (part) {
        if (/^`[^`]+`$/.test(part)) return '<code>' + esc(part.slice(1, -1)) + '</code>';
        var h = esc(part);
        h = h.replace(/!\[([^\]]*)\]\(([^)\s]+)\)/g, function (m, alt, url) {
          var src = safeUrl(url);
          return src ? '<img src="' + src + '" alt="' + alt + '" loading="lazy">' : alt;
        });
        h = h.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, function (m, label, url) {
          var href = safeUrl(url);
          return href ? '<a href="' + href + '" target="_blank" rel="noopener noreferrer">' + label + '</a>' : label;
        });
        h = h.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
        h = h.replace(/(^|[^*])\*([^*\s][^*]*)\*/g, '$1<em>$2</em>');
        h = h.replace(/(^|[\s(])_([^_\s][^_]*)_(?=[\s).,!?:;]|$)/g, '$1<em>$2</em>');
        return h;
      }).join('');
    };
    var out = [], para = [], list = null, quote = [];
    var flushPara = function () { if (para.length) out.push('<p>' + inline(para.join(' ')) + '</p>'); para = []; };
    var flushList = function () {
      if (list) out.push('<' + list.tag + '>' + list.items.map(function (i) { return '<li>' + inline(i) + '</li>'; }).join('') + '</' + list.tag + '>');
      list = null;
    };
    var flushQuote = function () { if (quote.length) out.push('<blockquote>' + inline(quote.join(' ')) + '</blockquote>'); quote = []; };
    md.replace(/\r\n?/g, '\n').split('\n').forEach(function (raw) {
      var t = raw.trim(), m;
      if (!t) { flushPara(); flushList(); flushQuote(); }
      else if ((m = t.match(/^(#{1,6})\s+(.*)$/))) {
        flushPara(); flushList(); flushQuote();
        var level = Math.min(Math.max(m[1].length, 2), 4);
        out.push('<h' + level + '>' + inline(m[2]) + '</h' + level + '>');
      } else if (/^(-{3,}|\*{3,}|_{3,})$/.test(t)) { flushPara(); flushList(); flushQuote(); out.push('<hr>'); }
      else if ((m = t.match(/^>\s?(.*)$/))) { flushPara(); flushList(); quote.push(m[1]); }
      else if ((m = t.match(/^[-*+]\s+(.*)$/)) || (m = t.match(/^\d+[.)]\s+(.*)$/))) {
        flushPara(); flushQuote();
        var tag = /^\d/.test(t) ? 'ol' : 'ul';
        if (!list || list.tag !== tag) { flushList(); list = { tag: tag, items: [] }; }
        list.items.push(m[1]);
      } else if (list && /^\s{2,}/.test(raw)) { list.items[list.items.length - 1] += ' ' + t; }
      else { flushList(); flushQuote(); para.push(t); }
    });
    flushPara(); flushList(); flushQuote();
    return out.join('\n');
  }
})();
