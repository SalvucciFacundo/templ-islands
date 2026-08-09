// islands runtime — generic client runtime for @island components.
//
// Capabilities (all driven by the manifest generated from the Go registry):
//
//  1. MUTATION (click): atomic optimistic ops ("inc", "toggle-text",
//     "class-toggle"), sync with the JSON endpoint, apply the server response
//     (source of truth) or roll back on error. With data-key, the mutation is
//     shared across every instance of the same domain key on the page.
//
//  2. RE-RENDER (input | change): fetch JSON from the endpoint (control value
//     as data-param), load the island's JS renderer and re-render the target.
//
//  3. FORM SUBMIT (submit): intercept the form, POST its data, re-render the
//     target with the response and emit islands:success / islands:error.
//
// The island markup keeps hx-post (or a classic form action) as a built-in
// server-driven fallback for when this runtime does not load.
(function () {
  "use strict";

  var manifest = null;

  // CSRF: si la pagina declara <meta name="csrf-token" content="...">, el
  // runtime lo manda como X-CSRF-Token en las peticiones que mutan.
  var csrfToken = (function () {
    var meta = document.querySelector('meta[name="csrf-token"]');
    return meta ? meta.getAttribute("content") : null;
  })();

  function csrfHeaders(method) {
    if (!csrfToken) return {};
    var m = (method || "GET").toUpperCase();
    if (m === "GET" || m === "HEAD" || m === "OPTIONS") return {};
    return { "X-CSRF-Token": csrfToken };
  }

  function loadManifest() {
    // The manifest URL is derived from this script's own URL, so the runtime
    // works no matter where it is mounted.
    var src = document.currentScript && document.currentScript.src;
    var url = src ? src.replace(/runtime\.js[^/]*$/, "manifest.json") : "/static/islands/manifest.json";
    return fetch(url).then(function (res) {
      if (!res.ok) throw new Error("manifest HTTP " + res.status);
      return res.json();
    });
  }

  // Fill {post_id} placeholders in the endpoint from data-post-id on the root.
  function fillPlaceholders(endpoint, root) {
    return endpoint.replace(/\{(\w+)\}/g, function (match, key) {
      var attr = "data-" + key.replace(/_/g, "-");
      var val = root.getAttribute(attr);
      return val != null ? val : match;
    });
  }

  function emit(name, detail) {
    document.dispatchEvent(new CustomEvent(name, { detail: detail }));
  }

  // ---- instance resolution (shared domain keys) ---------------------------
  // If the root has data-key, the mutation applies to EVERY element with the
  // same island name + key on the page (e.g. the same post in feed and modal).
  function instancesFor(root) {
    if (!root.dataset.key) return [root];
    var key = root.dataset.key;
    return Array.prototype.slice.call(
      document.querySelectorAll('[data-island="' + root.dataset.island + '"][data-key="' + key + '"]')
    );
  }

  // ---- mutation -----------------------------------------------------------

  function elementFor(root, field) {
    return field.selector ? root.querySelector(field.selector) : root;
  }

  // Optimistic: mutate the DOM NOW, before the server answers.
  function optimistic(roots, field, prev) {
    roots.forEach(function (root) {
      var el = elementFor(root, field);
      if (!el) return;
      prev.push({
        el: el,
        text: el.textContent,
        hadClass: field.op === "class-toggle" ? el.classList.contains(field.Class) : null,
        cls: field.Class,
      });
      if (field.op === "inc") {
        el.textContent = (parseInt(el.textContent, 10) || 0) + field.delta;
      } else if (field.op === "toggle-text") {
        el.textContent = el.textContent === field["true"] ? field["false"] : field["true"];
      } else if (field.op === "class-toggle") {
        el.classList.toggle(field.Class);
      }
    });
  }

  // Server wins: apply the response values.
  function applyServer(roots, field, data) {
    roots.forEach(function (root) {
      var el = elementFor(root, field);
      if (!el) return;
      var v = data[field.name];
      if (field.op === "toggle-text") {
        el.textContent = v ? field["true"] : field["false"];
      } else if (field.op === "class-toggle") {
        el.classList.toggle(field.Class, !!v);
      } else {
        el.textContent = v;
      }
    });
  }

  function rollback(prev) {
    prev.forEach(function (p) {
      if (!p.el) return;
      if (p.text != null) p.el.textContent = p.text;
      if (p.hadClass != null) p.el.classList.toggle(p.cls, p.hadClass);
    });
  }

  document.addEventListener(
    "click",
    function (e) {
      var root = e.target.closest("[data-island]");
      if (!root || !manifest) return;
      var cfg = manifest[root.dataset.island];
      if (!cfg) return;

      e.preventDefault();
      e.stopPropagation();

      // Destructive actions ask first (data-confirm="Are you sure?").
      if (root.dataset.confirm && !window.confirm(root.dataset.confirm)) {
        return;
      }

      // click -> re-render (expand comments, refresh a panel).
      if (cfg.render) {
        reRenderClick(root, cfg);
        return;
      }

      var roots = instancesFor(root);
      var prev = [];
      cfg.fields.forEach(function (field) {
        optimistic(roots, field, prev);
      });
      roots.forEach(function (r) {
        r.disabled = true;
        r.setAttribute("aria-disabled", "true");
      });

      fetch(fillPlaceholders(cfg.endpoint, root), { method: cfg.method || "POST", headers: csrfHeaders(cfg.method) })
        .then(function (res) {
          if (!res.ok) throw new Error("HTTP " + res.status);
          return res.json();
        })
        .then(function (data) {
          cfg.fields.forEach(function (field) {
            applyServer(roots, field, data);
          });
          emit("islands:success", { island: root.dataset.island, data: data });
        })
        .catch(function (err) {
          rollback(prev);
          emit("islands:error", { island: root.dataset.island, error: String(err && err.message || err) });
        })
        .finally(function () {
          roots.forEach(function (r) {
            r.disabled = false;
            r.removeAttribute("aria-disabled");
          });
        });
    },
    true
  );

  // ---- re-render (input / change / submit) --------------------------------

  var rendererCache = {};
  var debounceTimer = null;
  var abortControllers = {};

  function loadRenderer(url) {
    if (rendererCache[url]) return Promise.resolve();
    rendererCache[url] = true;
    return new Promise(function (resolve, reject) {
      var s = document.createElement("script");
      s.src = url;
      s.onload = resolve;
      s.onerror = reject;
      document.head.appendChild(s);
    });
  }

  // Renderers may return a list or a single item. By default the target is
  // replaced; with data-swap="append"|"prepend" the HTML is inserted instead
  // (useful for chat/feed deltas and infinite scroll).
  function renderInto(root, cfg, data) {
    var target = document.querySelector(root.dataset.target);
    return loadRenderer(cfg.render).then(function () {
      var renderer = window.islandsRenderers && window.islandsRenderers[root.dataset.island];
      if (!renderer) throw new Error("renderer not found for " + root.dataset.island);
      if (!target) return;
      var html = renderer(data);
      if (root.dataset.swap === "append") {
        target.insertAdjacentHTML("beforeend", html);
      } else if (root.dataset.swap === "prepend") {
        target.insertAdjacentHTML("afterbegin", html);
      } else {
        target.innerHTML = html;
      }
    });
  }

  // input / change: debounced fetch with the control value as data-param.
  // AbortController cancels the previous in-flight request for the same
  // target, so a slow stale response can never overwrite newer data.
  function abortPending(target) {
    if (abortControllers[target]) {
      abortControllers[target].abort();
      delete abortControllers[target];
    }
  }

  function trackFetch(target, url, cfg, then, fail) {
    abortPending(target);
    var controller = new AbortController();
    abortControllers[target] = controller;
    fetchData(url, cfg, controller.signal).then(then, function (err) {
      if (err && err.name === "AbortError") return; // esperado: reemplazado
      fail(err);
    }).finally(function () {
      if (abortControllers[target] === controller) delete abortControllers[target];
    });
  }

  function reRender(root, cfg, onDone, onFail) {
    var param = root.dataset.param || "q";
    // El valor sale del input (búsqueda) o, para intersect, del atributo
    // data-<param> del propio elemento (data-page).
    var value = root.value != null ? root.value : (root.dataset[param] || "");
    var sep = cfg.endpoint.indexOf("?") >= 0 ? "&" : "?";
    var url = cfg.endpoint + sep + param + "=" + encodeURIComponent(value);
    var delay = parseInt(root.dataset.debounce, 10) || 300;

    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(function () {
      trackFetch(root.dataset.target, url, cfg, function (data) {
        return renderInto(root, cfg, data).then(function () {
          emitSuccess(root);
          if (onDone) onDone(data);
        });
      }, function (err) {
        emitError(root, err);
        if (onFail) onFail(err);
      });
    }, delay);
  }

  // click: fetch with {placeholder} tokens filled from data-* attributes.
  function reRenderClick(root, cfg) {
    trackFetch(root.dataset.target, fillPlaceholders(cfg.endpoint, root), cfg, function (data) {
      return renderInto(root, cfg, data).then(function () {
        emitSuccess(root);
      });
    }, function (err) {
      emitError(root, err);
    });
  }

  function fetchData(url, cfg, signal) {
    return fetch(url, { method: cfg.method || "GET", headers: csrfHeaders(cfg.method), signal: signal }).then(function (res) {
      if (!res.ok) throw new Error("HTTP " + res.status);
      return res.json();
    });
  }

  function emitSuccess(root) {
    emit("islands:success", { island: root.dataset.island });
  }

  function emitError(root, err) {
    emit("islands:error", { island: root.dataset.island, error: String(err && err.message || err) });
  }

  // Field errors: the server answers a non-2xx JSON body with
  // {"field_errors": {"email": "Email invalido"}}. The runtime maps each
  // field to [data-error-for="<field>"] elements and marks the inputs.
  function applyFieldErrors(root, fieldErrors) {
    root.querySelectorAll("[data-error-for]").forEach(function (el) {
      el.textContent = "";
      el.classList.remove("show");
    });
    root.querySelectorAll("input.invalid, select.invalid, textarea.invalid").forEach(function (el) {
      el.classList.remove("invalid");
    });
    Object.keys(fieldErrors || {}).forEach(function (field) {
      var msg = fieldErrors[field];
      var target = root.querySelector('[data-error-for="' + field + '"]');
      if (target) {
        target.textContent = msg;
        target.classList.add("show");
      }
      var input = root.querySelector('[name="' + field + '"]');
      if (input) input.classList.add("invalid");
    });
  }

  // submit: POST the form data, re-render the target with the response.
  function submitForm(form, cfg) {
    var body = new URLSearchParams(new FormData(form));
    fetch(cfg.endpoint, { method: cfg.method || "POST", headers: csrfHeaders(cfg.method), body: body })
      .then(function (res) {
        if (!res.ok) {
          return res.json().catch(function () {
            return {};
          }).then(function (body) {
            throw Object.assign(new Error("HTTP " + res.status), { response: body, status: res.status });
          });
        }
        return res.json();
      })
      .then(function (data) {
        applyFieldErrors(form, null); // limpiar errores viejos en exito
        return renderInto(form, cfg, data).then(function () {
          emit("islands:success", { island: form.dataset.island, data: data });
        });
      })
      .catch(function (err) {
        if (err.response && err.response.field_errors) {
          applyFieldErrors(form, err.response.field_errors);
        }
        emit("islands:error", {
          island: form.dataset.island,
          error: String(err && err.message || err),
          status: err && err.status,
          response: err && err.response,
        });
      });
  }

  // One capture listener per supported trigger; the root decides.
  ["input", "change", "submit"].forEach(function (trigger) {
    document.addEventListener(
      trigger,
      function (e) {
        var root = e.target.closest("[data-island][data-trigger]");
        if (!root || !manifest) return;
        if (root.dataset.trigger !== trigger) return;
        var cfg = manifest[root.dataset.island];
        if (!cfg || !cfg.render) return;
        e.preventDefault();
        e.stopPropagation();

        if (trigger === "submit") {
          submitForm(root, cfg);
        } else {
          reRender(root, cfg);
        }
      },
      true
    );
  });

  // ---- real-time streams (SSE) -------------------------------------------
  // A page opts in with <div data-stream="name" data-target="#x"></div>.
  // The runtime opens an EventSource to the island endpoint and re-renders
  // the target with the island's renderer on every event.
  // ---- infinite scroll (intersect) ----------------------------------------
  // A sentinel with data-trigger="intersect" fetches the next page when it
  // enters the viewport (IntersectionObserver). The next page number lives in
  // data-<param> (e.g. data-page="2") and is incremented after each success;
  // when the server answers an empty array the observer is disconnected.
  var intersectBusy = {};
  var intersectObservers = {};

  function startIntersectionObservers() {
    var sentinels = document.querySelectorAll('[data-island][data-trigger="intersect"]');
    if (!sentinels.length) return;
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (entry) {
        if (!entry.isIntersecting) return;
        var root = entry.target;
        var cfg = manifest[root.dataset.island];
        if (!cfg || !cfg.render) return;
        if (intersectBusy[root.dataset.target]) return;
        intersectBusy[root.dataset.target] = true;
        reRender(root, cfg, function (data) {
          intersectBusy[root.dataset.target] = false;
          if (Array.isArray(data) && data.length === 0) {
            io.unobserve(root); // fin de la lista
            delete intersectObservers[root.dataset.target];
            return;
          }
          // avanzar a la proxima pagina para el siguiente intersect
          var param = root.dataset.param || "q";
          var n = parseInt(root.dataset[param], 10);
          if (!isNaN(n)) root.dataset[param] = String(n + 1);
        }, function () {
          intersectBusy[root.dataset.target] = false;
        });
      });
    }, { rootMargin: "200px" });

    sentinels.forEach(function (s) {
      intersectObservers[s.dataset.target] = io;
      io.observe(s);
    });
  }

  function startStreams() {
    var roots = document.querySelectorAll("[data-stream]");
    if (!roots.length) return;
    Array.prototype.forEach.call(roots, function (root) {
      var name = root.dataset.stream;
      var cfg = manifest[name];
      if (!cfg || !cfg.stream) return;

      var es = new EventSource(cfg.endpoint);
      es.onmessage = function (e) {
        var data;
        try {
          data = JSON.parse(e.data);
        } catch (err) {
          return;
        }
        renderInto(root, cfg, data).then(function () {
          emit("islands:success", { island: name, stream: true });
        }, function (err) {
          emit("islands:error", { island: name, stream: true, error: String(err && err.message || err) });
        });
      };
      es.onerror = function () {
        // EventSource reconnects automatically; the server controls the
        // backoff with the SSE "retry:" field.
        emit("islands:error", { island: name, stream: true, error: "stream connection lost" });
      };
    });
  }

  loadManifest().then(function (m) {
    manifest = m;
    startIntersectionObservers();
    startStreams();
  }).catch(function () {
    // No manifest: the runtime stays dormant and the server-driven
    // fallback (hx-post) keeps working untouched.
  });
})();
