/* Spotto — Golden Hour site JS & Payload CMS Integration. */
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

  /* ---------- Global Utility & Endpoint Discovery ---------- */
  function escapeHtml(str) {
    return String(str || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;');
  }

  function wrapHighlight(fullText, hlSnippet) {
    if (!fullText) return '';
    var escFull = escapeHtml(fullText);
    if (!hlSnippet) return escFull;
    var escHl = escapeHtml(hlSnippet);
    var lowerFull = escFull.toLowerCase();
    var lowerHl = escHl.toLowerCase();
    var idx = lowerFull.indexOf(lowerHl);
    if (idx !== -1) {
      var before = escFull.slice(0, idx);
      var matched = escFull.slice(idx, idx + escHl.length);
      var after = escFull.slice(idx + escHl.length);
      return before + '<span class="hl">' + matched + '</span>' + after;
    }
    return escFull;
  }

  var query = new URLSearchParams(location.search);
  var isLocal = /^(localhost|127\.0\.0\.1)$/.test(location.hostname) || location.protocol === 'file:' || !location.hostname;
  var bodyEl = document.body;
  var plansContainer = document.getElementById('pricingPlansContainer');

  // Resolve Spotto Go API base URL
  var localApi = isLocal ? query.get('api') : null;
  var baseApi = (bodyEl.getAttribute('data-api') || (plansContainer && plansContainer.getAttribute('data-api')) || window.SPOTTO_API_URL || '').replace(/\/$/, '');
  if (localApi) baseApi = localApi.replace(/\/$/, '');
  if (!baseApi && isLocal) baseApi = 'http://localhost:8080';

  // Resolve Payload CMS base URL
  var localCms = isLocal ? query.get('cms') : null;
  var baseCms = (bodyEl.getAttribute('data-cms') || bodyEl.getAttribute('data-cms-api') || window.SPOTTO_CMS_URL || '').replace(/\/$/, '');
  if (localCms) baseCms = localCms.replace(/\/$/, '');
  if (!baseCms && isLocal) baseCms = 'http://localhost:3002';

  function getCurrentPageSlug() {
    if (bodyEl && bodyEl.getAttribute('data-cms-page')) {
      return bodyEl.getAttribute('data-cms-page');
    }
    var pathname = location.pathname || '';
    var file = pathname.split('/').pop() || '';
    file = file.replace(/\.html$/, '');
    if (!file || file === 'index') return 'home';
    return file;
  }

  var currentSlug = getCurrentPageSlug();

  /* ---------- Pricing toggle & Dynamic packages from CMS or Go API ---------- */
  var seg = document.getElementById('billingToggle');

  function updateBillingUI(yearly) {
    if (!seg) return;
    var prices = document.querySelectorAll('[data-monthly]');
    var notes = document.querySelectorAll('[data-note-monthly]');
    seg.querySelectorAll('button').forEach(function (b) {
      var isTarget = (b.dataset.billing === 'yearly') === yearly;
      b.classList.toggle('on', isTarget);
      b.setAttribute('aria-pressed', isTarget ? 'true' : 'false');
    });
    prices.forEach(function (p) {
      var val = (yearly ? p.dataset.yearly : p.dataset.monthly) + ' ';
      if (p.childNodes && p.childNodes.length > 0) {
        p.childNodes[0].nodeValue = val;
      } else {
        p.textContent = val;
      }
    });
    notes.forEach(function (n) {
      n.textContent = yearly ? n.dataset.noteYearly : n.dataset.noteMonthly;
    });
  }

  if (seg) {
    seg.querySelectorAll('button').forEach(function (btn) {
      btn.addEventListener('click', function () {
        updateBillingUI(btn.dataset.billing === 'yearly');
      });
    });
  }

  function pkgHasFeature(pkg, featId) {
    if (!pkg || !Array.isArray(pkg.features)) return false;
    return pkg.features.some(function (f) {
      if (typeof f === 'object' && f !== null) {
        return f.id === featId || f.slug === featId;
      }
      return f === featId;
    });
  }

  function rebuildPricingMatrix(matrixTable, packages, featurePool) {
    if (!matrixTable || !Array.isArray(packages) || packages.length === 0) return;

    // Accessible label
    var tierNames = packages.map(function (p) { return p.name; }).join(', ');
    matrixTable.setAttribute('aria-label', 'Feature comparison across ' + tierNames + ' tiers');

    // Update any pre-existing th[data-matrix-col] for backwards compatibility
    packages.forEach(function (pkg) {
      var existingTh = matrixTable.querySelector('th[data-matrix-col="' + pkg.slug + '"]');
      if (existingTh) {
        var priceStr = pkg.monthlyPriceUsd != null ? '$' + Number(pkg.monthlyPriceUsd).toFixed(2) : '';
        existingTh.textContent = pkg.name + ' (' + priceStr + ')';
      }
    });

    // 1. Rebuild or populate <thead>
    var thead = matrixTable.querySelector('thead');
    if (!thead) {
      thead = document.createElement('thead');
      matrixTable.appendChild(thead);
    }
    thead.innerHTML = '';
    var headTr = document.createElement('tr');

    var capTh = document.createElement('th');
    if (capTh.style) capTh.style.width = '34%';
    capTh.textContent = 'Platform Capability';
    headTr.appendChild(capTh);

    var colWidth = (66 / packages.length).toFixed(1) + '%';
    packages.forEach(function (pkg) {
      var th = document.createElement('th');
      if (th.style) th.style.width = colWidth;
      th.setAttribute('data-matrix-col', pkg.slug);
      if (pkg.highlighted) {
        th.className = 'matrix-th-hl';
      }
      var pStr = pkg.monthlyPriceUsd != null ? '$' + Number(pkg.monthlyPriceUsd).toFixed(2) : '';
      th.textContent = pkg.name + ' (' + pStr + ')';
      headTr.appendChild(th);
    });
    thead.appendChild(headTr);

    // 2. Build <tbody>
    var tbody = matrixTable.querySelector('tbody');
    if (!tbody) {
      tbody = document.createElement('tbody');
      matrixTable.appendChild(tbody);
    }
    tbody.innerHTML = '';

    var matrixSections = [
      {
        category: 'Discovery & Trust',
        rows: [
          {
            label: 'Verified Map Pin & Listing',
            resolve: function (pkg) {
              if (pkgHasFeature(pkg, 'feat_vip_top_spot')) return '<span class="matrix-check">✓</span> VIP Golden Pin';
              if (pkgHasFeature(pkg, 'feat_priority_ranking')) return '<span class="matrix-check">✓</span> Priority';
              if (pkgHasFeature(pkg, 'feat_map_pin')) return '<span class="matrix-check">✓</span> Standard';
              return '<span class="matrix-cross">—</span>';
            }
          },
          {
            label: 'Official BR Verification Badge',
            resolve: function (pkg) {
              if (pkgHasFeature(pkg, 'feat_br_badge')) {
                if (pkg.monthlyPriceUsd >= 20 || pkg.slug === 'pro') return '<span class="matrix-check">✓</span> Fast-track verification';
                return '<span class="matrix-check">✓</span> Included';
              }
              return '<span class="matrix-cross">—</span>';
            }
          },
          {
            label: 'Category Ranking',
            resolve: function (pkg) {
              if (pkgHasFeature(pkg, 'feat_vip_top_spot') || pkg.monthlyPriceUsd >= 20) return 'Top of Category island-wide';
              if (pkgHasFeature(pkg, 'feat_priority_ranking')) return 'Priority in town';
              return 'Standard';
            }
          },
          {
            label: 'Discovery Promo Reel',
            resolve: function (pkg) {
              return pkgHasFeature(pkg, 'feat_promo_reel') ? '<span class="matrix-check">✓</span> Included' : '<span class="matrix-cross">—</span>';
            }
          }
        ]
      },
      {
        category: 'Marketing & Urgency',
        rows: [
          {
            label: 'Active Deals & Offers',
            resolve: function (pkg) {
              if (pkg.activeOffers === -1 || pkgHasFeature(pkg, 'feat_active_offers_unlimited')) return 'Unlimited active offers';
              if (pkg.activeOffers > 1) return 'Up to ' + pkg.activeOffers + ' active offers';
              if (pkg.activeOffers === 1 || pkgHasFeature(pkg, 'feat_active_offers_1')) return '1 live offer';
              if (pkgHasFeature(pkg, 'feat_active_offers_5')) return 'Up to 5 active offers';
              return '<span class="matrix-cross">—</span>';
            }
          },
          {
            label: 'Geofenced 3km Flash Sales',
            resolve: function (pkg) {
              if (pkg.flashSalesPerMonth === -1 || pkgHasFeature(pkg, 'feat_flash_sales_unlimited')) return 'Unlimited push broadcasts';
              if (pkg.flashSalesPerMonth > 0) return pkg.flashSalesPerMonth + ' per month';
              if (pkgHasFeature(pkg, 'feat_flash_sales_2')) return '2 per month';
              return '<span class="matrix-cross">—</span>';
            }
          },
          {
            label: 'Town Daily Quest Sponsorship',
            resolve: function (pkg) {
              return pkgHasFeature(pkg, 'feat_quest_sponsorship') ? '<span class="matrix-check">✓</span> Priority checkpoint' : '<span class="matrix-cross">—</span>';
            }
          }
        ]
      },
      {
        category: 'Loyalty & Operations',
        rows: [
          {
            label: 'Single-use Anti-fraud QR Scanner',
            resolve: function (pkg) {
              return pkgHasFeature(pkg, 'feat_qr_scanner') ? '<span class="matrix-check">✓</span> Included' : '<span class="matrix-cross">—</span>';
            }
          },
          {
            label: 'Digital Stamp Cards (Repeat Visits)',
            resolve: function (pkg) {
              return pkgHasFeature(pkg, 'feat_stamp_cards') ? '<span class="matrix-check">✓</span> Automated stamps' : '<span class="matrix-cross">—</span>';
            }
          },
          {
            label: 'Table & Session Reservations',
            resolve: function (pkg) {
              return pkgHasFeature(pkg, 'feat_reservations') ? '<span class="matrix-check">✓</span> Real-time bookings' : '<span class="matrix-cross">—</span>';
            }
          },
          {
            label: 'Team Accounts',
            resolve: function (pkg) {
              if (pkg.teamSeats === -1 || pkgHasFeature(pkg, 'feat_team_seats_unlimited')) return 'Up to 5 managers + unlimited staff';
              if (pkg.teamSeats > 1) return pkg.teamSeats + ' seats (manager + scanners)';
              if (pkg.teamSeats === 1 || pkgHasFeature(pkg, 'feat_staff_seat_1')) return '1 staff scanner seat';
              if (pkgHasFeature(pkg, 'feat_team_seats_3')) return '3 seats (manager + scanners)';
              return '<span class="matrix-cross">—</span>';
            }
          }
        ]
      },
      {
        category: 'Analytics & Support',
        rows: [
          {
            label: 'Geo Demand Heatmaps & Footfall',
            resolve: function (pkg) {
              return pkgHasFeature(pkg, 'feat_geo_heatmap') ? '<span class="matrix-check">✓</span> Live tourist density' : '<span class="matrix-cross">—</span>';
            }
          },
          {
            label: 'Support Channel',
            resolve: function (pkg) {
              if (pkg.monthlyPriceUsd >= 20 || pkgHasFeature(pkg, 'feat_team_seats_unlimited')) return 'Dedicated WhatsApp concierge';
              if (pkg.monthlyPriceUsd >= 10 || pkgHasFeature(pkg, 'feat_reservations')) return 'Email & in-app tickets';
              return 'Email support';
            }
          }
        ]
      }
    ];

    // Detect any extra / custom features allocated across packages that aren't in the standard matrix
    var handledFeatureIds = {
      feat_map_pin: true,
      feat_br_badge: true,
      feat_priority_ranking: true,
      feat_vip_top_spot: true,
      feat_promo_reel: true,
      feat_active_offers_1: true,
      feat_active_offers_5: true,
      feat_active_offers_unlimited: true,
      feat_flash_sales_2: true,
      feat_flash_sales_unlimited: true,
      feat_quest_sponsorship: true,
      feat_qr_scanner: true,
      feat_stamp_cards: true,
      feat_reservations: true,
      feat_staff_seat_1: true,
      feat_team_seats_3: true,
      feat_team_seats_unlimited: true,
      feat_geo_heatmap: true
    };

    var customFeatures = [];
    packages.forEach(function (pkg) {
      if (Array.isArray(pkg.features)) {
        pkg.features.forEach(function (f) {
          var id = (typeof f === 'object' && f) ? f.id : String(f);
          var name = (typeof f === 'object' && f) ? f.name : id.replace(/^feat_/, '').replace(/_/g, ' ');
          if (!handledFeatureIds[id] && !customFeatures.some(function (cf) { return cf.id === id; })) {
            customFeatures.push({ id: id, name: name });
          }
        });
      }
    });

    if (customFeatures.length > 0) {
      matrixSections.push({
        category: 'Custom Capabilities',
        rows: customFeatures.map(function (cf) {
          return {
            label: cf.name,
            resolve: function (pkg) {
              return pkgHasFeature(pkg, cf.id) ? '<span class="matrix-check">✓</span> Included' : '<span class="matrix-cross">—</span>';
            }
          };
        })
      });
    }

    // Render table rows into tbody
    var totalCols = packages.length + 1;
    matrixSections.forEach(function (sec) {
      var catTr = document.createElement('tr');
      catTr.className = 'matrix-cat-row';
      var catTd = document.createElement('td');
      catTd.colSpan = totalCols;
      catTd.textContent = sec.category;
      catTr.appendChild(catTd);
      tbody.appendChild(catTr);

      sec.rows.forEach(function (r) {
        var tr = document.createElement('tr');
        var labelTd = document.createElement('td');
        labelTd.textContent = r.label;
        tr.appendChild(labelTd);

        packages.forEach(function (pkg) {
          var valTd = document.createElement('td');
          if (pkg.highlighted) {
            valTd.className = 'matrix-td-hl';
          }
          valTd.innerHTML = r.resolve(pkg);
          tr.appendChild(valTd);
        });
        tbody.appendChild(tr);
      });
    });
  }

  function renderDynamicPackages(packages, featurePool) {
    if (!plansContainer) return;
    var isYearly = seg ? !!seg.querySelector('button[data-billing="yearly"].on') : false;
    plansContainer.innerHTML = '';
    plansContainer.setAttribute('data-count', String(packages.length));

    // Build feature lookup map from master pool if provided
    var featureMap = {};
    if (Array.isArray(featurePool)) {
      featurePool.forEach(function (f) {
        if (f && f.id) featureMap[f.id] = f;
      });
    }

    packages.forEach(function (pkg, i) {
      var card = document.createElement('div');
      var isHot = !!pkg.highlighted;
      card.className = 'plan rv ' + (isHot ? 'plan--hot ' : '') + 'd' + Math.min(i, 3);
      card.setAttribute('data-tier', pkg.slug);

      var badgeHtml = pkg.badge ? '<span class="badge-pop">' + escapeHtml(pkg.badge) + '</span>' : '';
      var monthlyPriceStr = '$' + (pkg.monthlyPriceUsd != null ? Number(pkg.monthlyPriceUsd).toFixed(2) : '3.99');
      var yearlyRate = (pkg.yearlyPriceUsd != null && !isNaN(pkg.yearlyPriceUsd))
        ? (Number(pkg.yearlyPriceUsd) / 12).toFixed(2)
        : (Number(pkg.monthlyPriceUsd) * 0.83).toFixed(2);
      var yearlyPriceStr = '$' + yearlyRate;
      var yearlyTotalStr = '$' + (pkg.yearlyPriceUsd != null ? Number(pkg.yearlyPriceUsd).toFixed(2) : '39.90');

      var featuresHtml = '';
      if (Array.isArray(pkg.features)) {
        pkg.features.forEach(function (f) {
          var name = '';
          if (typeof f === 'object' && f !== null) {
            name = f.name || (featureMap[f.id] ? featureMap[f.id].name : f.id);
          } else {
            name = featureMap[f] ? featureMap[f].name : String(f);
          }
          if (typeof name === 'string' && name.indexOf('feat_') === 0) {
            name = name.replace(/^feat_/, '').replace(/_/g, ' ');
          }
          if (name) {
            featuresHtml += '<li>' + escapeHtml(name) + '</li>';
          }
        });
      }
      if (Array.isArray(pkg.excludedFeatures)) {
        pkg.excludedFeatures.forEach(function (ex) {
          var label = '';
          if (typeof ex === 'object' && ex !== null) {
            label = ex.label || ex.name || (featureMap[ex.id] ? featureMap[ex.id].name : ex.id);
          } else {
            label = featureMap[ex] ? featureMap[ex].name : String(ex);
          }
          if (typeof label === 'string' && label.indexOf('feat_') === 0) {
            label = label.replace(/^feat_/, '').replace(/_/g, ' ');
          }
          if (label) {
            featuresHtml += '<li class="no">' + escapeHtml(label) + '</li>';
          }
        });
      }

      var btnClass = isHot ? 'btn btn--amber' : 'btn btn--ghost';
      var ctaText = pkg.ctaText || (pkg.slug === 'growth' ? 'Start 14-day trial' : (pkg.slug === 'pro' ? 'Upgrade to Pro' : 'Get Started'));
      var ctaLink = pkg.ctaLink || 'businesses.html#get-started';

      card.innerHTML =
        badgeHtml +
        '<div class="plan-name">' + escapeHtml(pkg.name) + '</div>' +
        '<div class="plan-for">' + escapeHtml(pkg.tagline || '') + '</div>' +
        '<div class="price"><span data-monthly="' + monthlyPriceStr + '" data-yearly="' + yearlyPriceStr + '">' +
        (isYearly ? yearlyPriceStr : monthlyPriceStr) + ' </span><small>/mo</small></div>' +
        '<div class="price-note" data-note-monthly="Billed monthly (' + monthlyPriceStr + '/mo), cancel anytime" ' +
        'data-note-yearly="Billed yearly at ' + yearlyTotalStr + ' (2 months free)">' +
        (isYearly ? 'Billed yearly at ' + yearlyTotalStr + ' (2 months free)' : 'Billed monthly (' + monthlyPriceStr + '/mo), cancel anytime') +
        '</div>' +
        '<ul class="plan-features">' + featuresHtml + '</ul>' +
        '<a class="' + btnClass + '" href="' + escapeHtml(ctaLink) + '">' + escapeHtml(ctaText) + '</a>';

      plansContainer.appendChild(card);
    });

    // Synchronize and dynamically rebuild Feature Comparison Matrix table if present
    var matrixTable = document.getElementById('pricingMatrixTable');
    if (matrixTable) {
      rebuildPricingMatrix(matrixTable, packages, featurePool);
    }

    // Re-apply billing toggle styles to newly inserted dynamic cards
    updateBillingUI(isYearly);
  }

  function hydratePricingPlans() {
    if (!plansContainer) return;

    var endpoints = [];
    if (baseCms) {
      endpoints.push(baseCms + '/api/pricing-plans?depth=2&sort=sortOrder');
    }
    if (baseApi) {
      endpoints.push(baseApi + '/api/v1/packages');
      endpoints.push(baseApi + '/packages');
    }
    endpoints.push('/cms-api/pricing-plans?depth=2&sort=sortOrder');
    endpoints.push('/api/pricing-plans?depth=2&sort=sortOrder');
    endpoints.push('/api/v1/packages');
    endpoints.push('/packages');
    endpoints.push('http://localhost:3002/api/pricing-plans?depth=2&sort=sortOrder');
    endpoints.push('http://localhost:8080/api/v1/packages');
    endpoints.push('https://spottoapi.threelogiq.com/api/v1/packages');

    function tryFetch(idx) {
      if (idx >= endpoints.length) return;
      fetch(endpoints[idx], { headers: { 'Accept': 'application/json' } })
        .then(function (res) {
          if (!res.ok) throw new Error('status ' + res.status);
          return res.json();
        })
        .then(function (data) {
          // Payload CMS shape: { docs: [...] }
          if (data && Array.isArray(data.docs) && data.docs.length > 0) {
            var pkgs = data.docs.map(function (doc) {
              return {
                id: doc.id,
                slug: doc.slug,
                name: doc.name,
                tagline: doc.tagline,
                monthlyPriceUsd: doc.monthlyPriceUsd,
                yearlyPriceUsd: doc.yearlyPriceUsd,
                badge: doc.badge,
                highlighted: !!doc.highlighted,
                features: doc.features || [],
                excludedFeatures: doc.excludedFeatures || [],
                ctaText: doc.ctaText,
                ctaLink: doc.ctaLink,
                sortOrder: doc.sortOrder,
              };
            });
            renderDynamicPackages(pkgs, null);
            return;
          }

          // Spotto Go API shape: { packages: [...], featurePool: [...] }
          if (data && Array.isArray(data.packages) && data.packages.length > 0) {
            renderDynamicPackages(data.packages, data.featurePool);
            return;
          }

          throw new Error('No package data in response');
        })
        .catch(function () {
          tryFetch(idx + 1);
        });
    }

    tryFetch(0);
  }

  hydratePricingPlans();

  /* ---------- Page Content & Hero Hydration from CMS ---------- */
  function hydratePageContent(slug) {
    var endpoints = [];
    if (baseCms) {
      endpoints.push(baseCms + '/api/pages?where[slug][equals]=' + encodeURIComponent(slug) + '&depth=2');
    }
    endpoints.push('/cms-api/pages?where[slug][equals]=' + encodeURIComponent(slug) + '&depth=2');
    endpoints.push('/api/pages?where[slug][equals]=' + encodeURIComponent(slug) + '&depth=2');
    endpoints.push('http://localhost:3002/api/pages?where[slug][equals]=' + encodeURIComponent(slug) + '&depth=2');

    function tryFetch(idx) {
      if (idx >= endpoints.length) return;
      fetch(endpoints[idx], { headers: { 'Accept': 'application/json' } })
        .then(function (res) {
          if (!res.ok) throw new Error('status ' + res.status);
          return res.json();
        })
        .then(function (data) {
          var doc = null;
          if (data && Array.isArray(data.docs) && data.docs.length > 0) {
            doc = data.docs[0];
          } else if (data && data.slug) {
            doc = data;
          }
          if (!doc) throw new Error('No page found');

          // 1. Meta & Title
          if (doc.title) {
            document.title = doc.title;
          }
          if (doc.metaDescription) {
            var metaDesc = document.querySelector('meta[name="description"]');
            if (metaDesc) metaDesc.setAttribute('content', doc.metaDescription);
          }

          // 2. Hero Section
          if (doc.hero) {
            var h1 = document.querySelector('.hero h1, .page-hero h1');
            var eyebrow = document.querySelector('.hero .eyebrow, .page-hero .eyebrow');
            if (doc.hero.eyebrow) {
              if (eyebrow) {
                eyebrow.textContent = doc.hero.eyebrow;
              } else if (h1 && h1.parentNode) {
                var newEyebrow = document.createElement('span');
                newEyebrow.className = 'eyebrow';
                newEyebrow.textContent = doc.hero.eyebrow;
                h1.parentNode.insertBefore(newEyebrow, h1);
              }
            }

            if (h1 && doc.hero.headline) {
              h1.innerHTML = wrapHighlight(doc.hero.headline, doc.hero.highlightText);
            }

            var lead = document.querySelector('.hero .lead, .page-hero .lead');
            if (lead && doc.hero.lead) {
              lead.textContent = doc.hero.lead;
            }

            var primaryCta = document.querySelector('.hero .hero-ctas a.btn--amber, .hero .hero-ctas a.btn--ink, .page-hero .hero-ctas a');
            if (primaryCta) {
              if (doc.hero.primaryCtaText) primaryCta.textContent = doc.hero.primaryCtaText;
              if (doc.hero.primaryCtaLink) primaryCta.setAttribute('href', doc.hero.primaryCtaLink);
            }

            var secondaryCta = document.querySelector('.hero .hero-ctas a.btn--ghost-light');
            if (secondaryCta) {
              if (doc.hero.secondaryCtaText) secondaryCta.textContent = doc.hero.secondaryCtaText;
              if (doc.hero.secondaryCtaLink) secondaryCta.setAttribute('href', doc.hero.secondaryCtaLink);
            }
          }

          // 3. Sections (Stats, CTA Band, Split Features)
          if (Array.isArray(doc.sections)) {
            var splitIdx = 0;
            var splitContainers = document.querySelectorAll('.split');

            doc.sections.forEach(function (sec) {
              if ((sec.blockType === 'statsRow' || sec.slug === 'statsRow') && Array.isArray(sec.stats)) {
                var statEls = document.querySelectorAll('.stats-grid .stat');
                sec.stats.forEach(function (st, sIdx) {
                  if (statEls[sIdx]) {
                    var b = statEls[sIdx].querySelector('b');
                    var span = statEls[sIdx].querySelector('span');
                    if (b && st.value) b.textContent = st.value;
                    if (span && st.label) span.textContent = st.label;
                  }
                });
              } else if (sec.blockType === 'ctaBand' || sec.slug === 'ctaBand') {
                var band = document.querySelector('.band--center');
                if (band) {
                  var bandH2 = band.querySelector('h2');
                  var bandP = band.querySelector('p');
                  var bandBtn = band.querySelector('.btn');
                  if (bandH2 && sec.headline) bandH2.textContent = sec.headline;
                  if (bandP && sec.subcopy) bandP.textContent = sec.subcopy;
                  if (bandBtn) {
                    if (sec.buttonText) bandBtn.textContent = sec.buttonText;
                    if (sec.buttonLink) bandBtn.setAttribute('href', sec.buttonLink);
                  }
                }
              } else if (sec.blockType === 'splitFeature' || sec.slug === 'splitFeature') {
                if (splitContainers[splitIdx]) {
                  var splitEl = splitContainers[splitIdx];
                  var sH2 = splitEl.querySelector('h2');
                  var sBody = splitEl.querySelector('p.muted, p');
                  var sEyebrow = splitEl.querySelector('.eyebrow');
                  var sCap = splitEl.querySelector('.ph-cap');
                  if (sH2 && sec.title) sH2.textContent = sec.title;
                  if (sBody && sec.body) sBody.textContent = sec.body;
                  if (sEyebrow && sec.eyebrow) sEyebrow.textContent = sec.eyebrow;
                  if (sCap && sec.imageTag) sCap.textContent = sec.imageTag;
                  if (Array.isArray(sec.bullets)) {
                    var tickLis = splitEl.querySelectorAll('ul.ticks li, .ticks li');
                    sec.bullets.forEach(function (bItem, bIdx) {
                      var bulletText = typeof bItem === 'object' && bItem !== null ? (bItem.text || bItem.label) : bItem;
                      if (tickLis[bIdx] && bulletText) tickLis[bIdx].textContent = bulletText;
                    });
                  }
                }
                splitIdx++;
              }
            });
          }
        })
        .catch(function () {
          tryFetch(idx + 1);
        });
    }

    tryFetch(0);
  }

  hydratePageContent(currentSlug);

  /* ---------- FAQs Hydration from CMS ---------- */
  function hydrateFAQs(slug) {
    var faqContainers = document.querySelectorAll('.faq');
    if (!faqContainers.length) return;

    var endpoints = [];
    if (baseCms) {
      endpoints.push(baseCms + '/api/faqs?sort=sortOrder&limit=50');
    }
    endpoints.push('/cms-api/faqs?sort=sortOrder&limit=50');
    endpoints.push('/api/faqs?sort=sortOrder&limit=50');
    endpoints.push('http://localhost:3002/api/faqs?sort=sortOrder&limit=50');

    function tryFetch(idx) {
      if (idx >= endpoints.length) return;
      fetch(endpoints[idx], { headers: { 'Accept': 'application/json' } })
        .then(function (res) {
          if (!res.ok) throw new Error('status ' + res.status);
          return res.json();
        })
        .then(function (data) {
          var docs = (data && Array.isArray(data.docs)) ? data.docs : (Array.isArray(data) ? data : null);
          if (!docs || !docs.length) return;

          faqContainers.forEach(function (faqContainer) {
            var targetCat = faqContainer.getAttribute('data-category') || (slug === 'pricing' ? 'pricing' : (slug === 'businesses' ? 'businesses' : null));
            var filtered = targetCat ? docs.filter(function (f) { return !f.category || f.category === targetCat; }) : docs;
            if (!filtered.length) return;

            faqContainer.innerHTML = '';
            filtered.forEach(function (faq) {
              var details = document.createElement('details');
              details.innerHTML = '<summary>' + escapeHtml(faq.question) + '</summary><p>' + escapeHtml(faq.answer) + '</p>';
              faqContainer.appendChild(details);
            });
          });
        })
        .catch(function () {
          tryFetch(idx + 1);
        });
    }

    tryFetch(0);
  }

  hydrateFAQs(currentSlug);

  /* ---------- Testimonials Hydration from CMS ---------- */
  function hydrateTestimonials(slug) {
    var bq = document.querySelector('.bigquote');
    if (!bq) return;

    var endpoints = [];
    if (baseCms) {
      endpoints.push(baseCms + '/api/testimonials?sort=sortOrder&limit=20');
    }
    endpoints.push('/cms-api/testimonials?sort=sortOrder&limit=20');
    endpoints.push('/api/testimonials?sort=sortOrder&limit=20');
    endpoints.push('http://localhost:3002/api/testimonials?sort=sortOrder&limit=20');

    function tryFetch(idx) {
      if (idx >= endpoints.length) return;
      fetch(endpoints[idx], { headers: { 'Accept': 'application/json' } })
        .then(function (res) {
          if (!res.ok) throw new Error('status ' + res.status);
          return res.json();
        })
        .then(function (data) {
          var docs = (data && Array.isArray(data.docs)) ? data.docs : (Array.isArray(data) ? data : null);
          if (!docs || !docs.length) return;

          var matching = docs.filter(function (t) {
            return !t.page || t.page === 'all' || t.page === slug;
          });
          if (!matching.length) matching = docs;

          var primary = matching[0];
          var bqEl = bq.querySelector('blockquote');
          if (bqEl && primary.quote) {
            bqEl.innerHTML = '"' + wrapHighlight(primary.quote, primary.highlightQuote) + '"';
          }

          var whoEl = bq.querySelector('.quote-who');
          if (whoEl) {
            var av = whoEl.querySelector('.avatar');
            if (av && primary.avatarInitials) av.textContent = primary.avatarInitials;
            var b = whoEl.querySelector('b');
            if (b && primary.author) b.textContent = primary.author;
            var desc = whoEl.querySelector('span span');
            if (desc) desc.textContent = [primary.venue, primary.town].filter(Boolean).join(', ');
          }

          var minisContainer = bq.querySelector('.quote-minis');
          var subs = matching.slice(1);
          if (minisContainer && subs.length > 0) {
            minisContainer.innerHTML = '';
            subs.forEach(function (sub, i) {
              var miniCard = document.createElement('div');
              miniCard.className = 'rv d' + ((i % 3) + 1);
              var pEl = document.createElement('p');
              pEl.textContent = '"' + (sub.quote || '') + '"';
              var whoWrap = document.createElement('div');
              whoWrap.className = 'quote-who';
              var avSpan = document.createElement('span');
              avSpan.className = 'avatar';
              avSpan.textContent = sub.avatarInitials || (sub.author ? sub.author.slice(0, 2).toUpperCase() : '');
              var spanWrap = document.createElement('span');
              var bName = document.createElement('b');
              bName.textContent = sub.author || '';
              var subLoc = document.createElement('span');
              subLoc.textContent = [sub.venue, sub.town].filter(Boolean).join(', ');
              spanWrap.appendChild(bName);
              spanWrap.appendChild(subLoc);
              whoWrap.appendChild(avSpan);
              whoWrap.appendChild(spanWrap);
              miniCard.appendChild(pEl);
              miniCard.appendChild(whoWrap);
              minisContainer.appendChild(miniCard);
            });
          }
        })
        .catch(function () {
          tryFetch(idx + 1);
        });
    }

    tryFetch(0);
  }

  hydrateTestimonials(currentSlug);

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

  /* ---------- Stories (blog.html) — Payload CMS + Spotto API ----------
     Published blog posts come from Payload CMS (/api/stories) or the Spotto
     API (GET /content?type=blog). Supports single-post and paginated archive. */
  var stories = document.getElementById('stories');
  if (stories) {
    var mediaUrl = function (u) {
      if (!u) return u;
      if (u.charAt(0) === '/' && u.charAt(1) !== '/') {
        return (baseCms || baseApi) + u;
      }
      return u;
    };
    var storyHref = function (slug) {
      return 'blog.html?post=' + encodeURIComponent(slug) + (localApi ? '&api=' + encodeURIComponent(localApi) : '') + (localCms ? '&cms=' + encodeURIComponent(localCms) : '');
    };
    var palettes = ['im-sigiriya', 'im-arugam', 'im-ember', 'im-galle', 'im-yala', 'im-peak', 'im-ella'];
    var el = function (tag, cls, text) {
      var n = document.createElement(tag);
      if (cls) n.className = cls;
      if (text) n.textContent = text;
      return n;
    };
    var category = function (post) {
      var t = (post.tags && post.tags[0]) || post.category || 'Story';
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

    var statusEl = document.getElementById('storiesStatus');
    var featuredEl = document.getElementById('storiesFeatured');
    var gridEl = document.getElementById('storiesGrid');
    var moreBtn = document.getElementById('storiesMore');
    var offset = 0;
    var shown = 0;
    var PAGE = 13;

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

    var normalizeStory = function (item) {
      var tagList = [];
      if (Array.isArray(item.tags)) {
        tagList = item.tags.map(function (t) {
          return typeof t === 'object' && t !== null ? t.tag : t;
        });
      }
      return {
        id: item.id,
        title: item.title,
        slug: item.slug,
        summary: item.summary,
        body: item.body,
        coverUrl: item.coverUrl,
        category: item.category || 'Story',
        tags: tagList,
        authorName: item.authorName,
        regionName: item.regionName,
        publishedAt: item.publishedAt,
      };
    };

    var loadPage = function () {
      moreBtn.disabled = true;
      var fetchEndpoints = [];
      if (baseCms) {
        fetchEndpoints.push(baseCms + '/api/stories?where[status][equals]=published&sort=-publishedAt&limit=' + PAGE + '&offset=' + offset);
      }
      if (baseApi) {
        fetchEndpoints.push(baseApi + '/content?type=blog&limit=' + PAGE + '&offset=' + offset);
      }
      fetchEndpoints.push('/cms-api/stories?where[status][equals]=published&sort=-publishedAt&limit=' + PAGE + '&offset=' + offset);
      fetchEndpoints.push('/api/stories?where[status][equals]=published&sort=-publishedAt&limit=' + PAGE + '&offset=' + offset);
      fetchEndpoints.push('/content?type=blog&limit=' + PAGE + '&offset=' + offset);
      fetchEndpoints.push('http://localhost:3002/api/stories?where[status][equals]=published&sort=-publishedAt&limit=' + PAGE + '&offset=' + offset);
      fetchEndpoints.push('https://spottoapi.threelogiq.com/content?type=blog&limit=' + PAGE + '&offset=' + offset);

      function tryLoad(idx) {
        if (idx >= fetchEndpoints.length) {
          statusEl.textContent = shown ? '' : "Stories couldn't load right now. Please try again in a moment.";
          moreBtn.disabled = false;
          return;
        }
        fetch(fetchEndpoints[idx], { headers: { Accept: 'application/json' } })
          .then(function (res) {
            if (!res.ok) throw new Error('status ' + res.status);
            return res.json();
          })
          .then(function (res) {
            var items = res.docs || res.items || [];
            var hasMore = res.hasNextPage != null ? res.hasNextPage : !!res.hasMore;
            items.forEach(function (raw) {
              addPost(normalizeStory(raw));
            });
            offset += items.length;
            statusEl.textContent = shown ? '' : 'No stories yet. Check back soon.';
            moreBtn.hidden = !hasMore;
            moreBtn.disabled = false;
          })
          .catch(function () {
            tryLoad(idx + 1);
          });
      }

      tryLoad(0);
    };

    var showStory = function (postSlug) {
      stories.hidden = true;
      var hero = document.getElementById('storiesHero');
      if (hero) hero.hidden = true;
      var article = document.getElementById('story');
      article.hidden = false;
      var titleEl = document.getElementById('storyTitle');
      titleEl.textContent = 'Loading…';

      var storyEndpoints = [];
      if (baseCms) {
        storyEndpoints.push(baseCms + '/api/stories?where[slug][equals]=' + encodeURIComponent(postSlug) + '&limit=1');
      }
      if (baseApi) {
        storyEndpoints.push(baseApi + '/content/' + encodeURIComponent(postSlug));
      }
      storyEndpoints.push('/cms-api/stories?where[slug][equals]=' + encodeURIComponent(postSlug) + '&limit=1');
      storyEndpoints.push('/api/stories?where[slug][equals]=' + encodeURIComponent(postSlug) + '&limit=1');
      storyEndpoints.push('/content/' + encodeURIComponent(postSlug));
      storyEndpoints.push('http://localhost:3002/api/stories?where[slug][equals]=' + encodeURIComponent(postSlug) + '&limit=1');

      function tryStory(idx) {
        if (idx >= storyEndpoints.length) {
          titleEl.textContent = "This story couldn't load right now.";
          document.getElementById('storySummary').textContent = 'Please try again in a moment.';
          return;
        }
        fetch(storyEndpoints[idx], { headers: { Accept: 'application/json' } })
          .then(function (res) {
            if (!res.ok) throw new Error('status ' + res.status);
            return res.json();
          })
          .then(function (data) {
            var post = null;
            if (data && Array.isArray(data.docs) && data.docs.length > 0) {
              post = normalizeStory(data.docs[0]);
            } else if (data && data.title) {
              post = normalizeStory(data);
            }
            if (!post) throw new Error('Story not found');

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
            tags.innerHTML = '';
            (post.tags || []).forEach(function (t) {
              tags.appendChild(el('span', 'story-tag', '#' + t));
            });
          })
          .catch(function () {
            tryStory(idx + 1);
          });
      }

      tryStory(0);
    };

    moreBtn.addEventListener('click', loadPage);
    if (query.get('post')) showStory(query.get('post'));
    else loadPage();
  }

  /* Markdown for story bodies */
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

  /* ---------- Expose Spotto CMS API Client on window ---------- */
  window.SpottoCMS = {
    getCmsUrl: function () { return baseCms; },
    getApiUrl: function () { return baseApi; },
    getCurrentPageSlug: getCurrentPageSlug,
    escapeHtml: escapeHtml,
    wrapHighlight: wrapHighlight,
    renderMarkdown: renderMarkdown,
    hydratePageContent: hydratePageContent,
    hydratePricingPlans: hydratePricingPlans,
    renderDynamicPackages: renderDynamicPackages,
    rebuildPricingMatrix: rebuildPricingMatrix,
    hydrateFAQs: hydrateFAQs,
    hydrateTestimonials: hydrateTestimonials,
    hydrateAll: function () {
      var s = getCurrentPageSlug();
      hydratePageContent(s);
      hydratePricingPlans();
      hydrateFAQs(s);
      hydrateTestimonials(s);
    },
  };

})();
