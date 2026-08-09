// islands runtime — generic client runtime for @island mutations.
//
// How it works:
//   1. Loads the manifest (served by the Go registry) once.
//   2. Intercepts clicks on [data-island] elements in CAPTURE phase,
//      BEFORE htmx/hyperscript processes them.
//   3. Applies the declared optimistic mutations instantly (op "inc",
//      "toggle-text", "class-toggle") and disables the element.
//   4. Calls the JSON endpoint (placeholders filled from data-* attrs).
//   5. On success, applies the SERVER response (source of truth).
//   6. On error, rolls back every optimistic change.
//
// The island markup also keeps hx-post (or a classic form) as a built-in
// server-driven fallback for when this runtime does not load.
(function () {
  "use strict";

  // The manifest URL is derived from this script's own URL, so the runtime
  // works no matter where it is mounted. Fallback to the documented default.
  var MANIFEST_URL = (function () {
    var src = document.currentScript && document.currentScript.src;
    return src ? src.replace(/runtime\.js[^/]*$/, "manifest.json") : "/static/islands/manifest.json";
  })();
  var manifest = null;

  function loadManifest() {
    return fetch(MANIFEST_URL).then(function (res) {
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

  function elementFor(root, field) {
    return field.selector ? root.querySelector(field.selector) : root;
  }

  // Optimistic: mutate the DOM NOW, before the server answers.
  function optimistic(root, field, prev) {
    var el = elementFor(root, field);
    if (!el) return;
    prev[field.name + "|" + (field.selector || "$root")] = {
      el: el,
      text: el.textContent,
      hadClass: field.op === "class-toggle" ? el.classList.contains(field.Class) : null,
      cls: field.Class,
    };
    if (field.op === "inc") {
      el.textContent = (parseInt(el.textContent, 10) || 0) + field.delta;
    } else if (field.op === "toggle-text") {
      el.textContent = el.textContent === field["true"] ? field["false"] : field["true"];
    } else if (field.op === "class-toggle") {
      el.classList.toggle(field.Class);
    }
  }

  // Server wins: apply the response values.
  function applyServer(root, field, data) {
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
  }

  function rollback(prev) {
    Object.keys(prev).forEach(function (key) {
      var p = prev[key];
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
      if (!cfg || cfg.render) return; // re-render islands are not click-driven

      // htmx / server-driven fallback must not fire in client mode.
      e.preventDefault();
      e.stopPropagation();

      var prev = {};
      cfg.fields.forEach(function (field) {
        optimistic(root, field, prev);
      });
      root.disabled = true;

      fetch(fillPlaceholders(cfg.endpoint, root), { method: cfg.method || "POST" })
        .then(function (res) {
          if (!res.ok) throw new Error("HTTP " + res.status);
          return res.json();
        })
        .then(function (data) {
          cfg.fields.forEach(function (field) {
            applyServer(root, field, data);
          });
        })
        .catch(function () {
          rollback(prev);
        })
        .finally(function () {
          root.disabled = false;
        });
    },
    true
  );

  // ---- Re-render islands ---------------------------------------------
  // A control (e.g. a search input) declares data-island + data-trigger.
  // On that event, the runtime fetches JSON from the endpoint (with the
  // control's value as data-param), loads the island's JS renderer and
  // re-renders the target container.
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

  function reRender(root, cfg) {
    var target = document.querySelector(root.dataset.target);
    var param = root.dataset.param || "q";
    var value = root.value;
    var sep = cfg.endpoint.indexOf("?") >= 0 ? "&" : "?";
    var url = cfg.endpoint + sep + param + "=" + encodeURIComponent(value);

    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(function () {
      fetch(url, { method: cfg.method || "GET" })
        .then(function (res) {
          if (!res.ok) throw new Error("HTTP " + res.status);
          return res.json();
        })
        .then(function (data) {
          return loadRenderer(cfg.render).then(function () {
            var renderer = window.islandsRenderers && window.islandsRenderers[root.dataset.island];
            if (!renderer) throw new Error("renderer not found for " + root.dataset.island);
            if (target) target.innerHTML = renderer(data);
          });
        })
        .catch(function () {
          // keep the previous render; the error is visible in the console.
        });
    }, 300);
  }

  document.addEventListener(
    "input",
    function (e) {
      var root = e.target.closest("[data-island][data-trigger]");
      if (!root || !manifest) return;
      var cfg = manifest[root.dataset.island];
      if (!cfg || !cfg.render) return;
      e.stopPropagation();
      reRender(root, cfg);
    },
    true
  );

  loadManifest().catch(function () {
    // No manifest: the runtime stays dormant and the server-driven
    // fallback (hx-post) keeps working untouched.
  });
})();
