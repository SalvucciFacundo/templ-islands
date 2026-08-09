// Listener del demo: pinta la barra de progreso del form new-post con los
// eventos islands:progress que emite el runtime durante la subida (XHR).
// La barra la define el HTML; este archivo solo la rellena y la muestra.
(function () {
  "use strict";

  var bar = document.querySelector(".upload-progress");
  if (!bar) return;

  function set(value, visible) {
    bar.value = value;
    bar.classList.toggle("show", visible);
  }

  document.addEventListener("islands:progress", function (e) {
    if (e.detail.island !== "new-post") return;
    set(e.detail.percent, true);
  });

  // Al terminar (exito o error) la barra vuelve a cero y se oculta.
  ["islands:success", "islands:error"].forEach(function (name) {
    document.addEventListener(name, function (e) {
      if (e.detail.island !== "new-post") return;
      set(0, false);
    });
  });
})();
