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
      });

      fetch(fillPlaceholders(cfg.endpoint, root), { method: cfg.method || "POST" })
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
          });
        });
    },
    true
  );

  // ---- re-render (input / change / submit) --------------------------------

  var rendererCache = {};
  var debounceTimer = null;

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

  function renderInto(root, cfg, data) {
    var target = document.querySelector(root.dataset.target);
    return loadRenderer(cfg.render).then(function () {
      var renderer = window.islandsRenderers && window.islandsRenderers[root.dataset.island];
      if (!renderer) throw new Error("renderer not found for " + root.dataset.island);
      if (target) target.innerHTML = renderer(data);
    });
  }

  // input / change: debounced fetch with the control value as data-param.
  function reRender(root, cfg) {
    var param = root.dataset.param || "q";
    var sep = cfg.endpoint.indexOf("?") >= 0 ? "&" : "?";
    var url = cfg.endpoint + sep + param + "=" + encodeURIComponent(root.value);

    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(function () {
      fetchData(url, cfg)
        .then(function (data) {
          return renderInto(root, cfg, data);
        })
        .then(function () {
          emitSuccess(root);
        }, function (err) {
          emitError(root, err);
        });
    }, 300);
  }

  // click: fetch with {placeholder} tokens filled from data-* attributes.
  function reRenderClick(root, cfg) {
    fetchData(fillPlaceholders(cfg.endpoint, root), cfg)
      .then(function (data) {
        return renderInto(root, cfg, data);
      })
      .then(function () {
        emitSuccess(root);
      }, function (err) {
        emitError(root, err);
      });
  }

  function fetchData(url, cfg) {
    return fetch(url, { method: cfg.method || "GET" }).then(function (res) {
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

  // submit: POST the form data, re-render the target with the response.
  function submitForm(form, cfg) {
    var body = new URLSearchParams(new FormData(form));
    fetch(cfg.endpoint, { method: cfg.method || "POST", body: body })
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
        return renderInto(form, cfg, data).then(function () {
          emit("islands:success", { island: form.dataset.island, data: data });
        });
      })
      .catch(function (err) {
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

  loadManifest().catch(function () {
    // No manifest: the runtime stays dormant and the server-driven
    // fallback (hx-post) keeps working untouched.
  });
})();
