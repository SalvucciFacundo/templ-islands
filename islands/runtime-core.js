// islands runtime core — funciones puras y testeables (node:test).
//
// Este archivo NO toca el DOM. El runtime (runtime.js) lo usa via
// window.islandsCore; los tests (runtime-core.test.js) lo importan con
// require().
(function (global) {
  "use strict";

  var core = {
    // Rellena {post_id} en el endpoint con los data-* del elemento.
    // attrs = {"data-post-id": "7"} (claves con el guion, como en el HTML).
    fillPlaceholders: function (endpoint, attrs) {
      return endpoint.replace(/\{(\w+)\}/g, function (match, key) {
        var attr = "data-" + key.replace(/_/g, "-");
        var val = attrs[attr];
        return val != null ? val : match;
      });
    },

    // Valor optimistic de un campo sobre el texto actual del elemento.
    // "inc" suma el delta; "toggle-text" alterna; cualquier otra op deja el
    // texto intacto (class-toggle muta la clase, no el texto).
    optimisticValue: function (field, currentText) {
      if (field.op === "inc") {
        return String((parseInt(currentText, 10) || 0) + field.delta);
      }
      if (field.op === "toggle-text") {
        return currentText === field["true"] ? field["false"] : field["true"];
      }
      return currentText;
    },

    // Valor del parametro de re-render: el input manda, si no el data-<param>.
    controlValue: function (inputValue, param, data) {
      if (inputValue != null) return inputValue;
      return data[param] != null ? data[param] : "";
    },

    // Debounce configurable: data-debounce="N" o el fallback (300ms).
    debounceMs: function (raw, fallback) {
      var n = parseInt(raw, 10);
      return isNaN(n) || n < 0 ? fallback : n;
    },

    // Escape HTML para los renderers. Se expone como funcion global
    // (escapeHtml) para que cualquier renderer JS pueda usarla sin depender
    // de otro archivo.
    escapeHtml: function (s) {
      return String(s)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;");
    },
  };

  if (typeof module !== "undefined" && module.exports) {
    module.exports = core;
  }
  global.islandsCore = core;
  global.escapeHtml = core.escapeHtml; // disponible para los renderers
})(typeof window !== "undefined" ? window : globalThis);
