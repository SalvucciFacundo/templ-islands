// Renderer client de las islas de comentarios (re-render por click).
//
// Registra dos islas que comparten el mismo renderer:
//   - "comments":        boton "Ver comentarios" -> re-render del contenedor
//   - "delete-comment":  boton "Eliminar" (data-confirm) -> re-render
//
// El markup debe coincidir con el HTML que generaria templ para los mismos
// datos; parity_test.go mantiene templ y JS en sync.
window.islandsRenderers = window.islandsRenderers || {};

function renderComments(comments) {
  if (!comments || !comments.length) {
    return '<p class="comments-empty">Sin comentarios.</p>';
  }
  return comments
    .map(function (c) {
      return (
        '<div class="comment">' +
        '<span class="comment-text">' + escapeHtml(c.text) + "</span>" +
        '<button class="comment-delete" data-island="delete-comment" data-trigger="click" ' +
        'data-confirm="Eliminar comentario?" data-comment-id="' + c.id + '" ' +
        'data-target="#comments-' + c.post_id + '" ' +
        'hx-post="/delete_comment/' + c.id + '" hx-swap="outerHTML">Eliminar</button>' +
        "</div>"
      );
    })
    .join("");
}

islandsRenderers["comments"] = renderComments;
islandsRenderers["delete-comment"] = renderComments;
