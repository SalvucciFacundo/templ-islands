// Renderer client del panel de tabs (isla post-panel, re-render por click).
//
// Patron de tabs: cada tab es una isla click que manda su data-tab como
// placeholder {tab}; el server responde el contenido del panel y este
// renderer genera el HTML del contenedor #panel-resumen.
window.islandsRenderers = window.islandsRenderers || {};

islandsRenderers["post-panel"] = function (data) {
  return (
    '<h3 class="panel-title">' + escapeHtml(data.titulo) + "</h3>" +
    '<ul class="panel-lines">' +
    (data.lineas || [])
      .map(function (l) {
        return "<li>" + escapeHtml(l) + "</li>";
      })
      .join("") +
    "</ul>"
  );
};
