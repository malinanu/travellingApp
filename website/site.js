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
})();
