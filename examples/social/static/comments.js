// Renderer client de las islas de comentarios (re-render por click).
//
// "comments" y "delete-comment" comparten este renderer: la isla
// delete-comment declara renderer=comments en su directiva @island, asi el
// renderer se registra UNA sola vez (contrato de renderer compartido).
window.islandsRenderers = window.islandsRenderers || {};

function renderComments(comments) {
  // Patron de renderers: cuando el server responde no-2xx con field_errors,
  // el runtime deja pasar el body y el renderer los pinta inline dentro del
  // target re-renderizado (no es un form, no aplica [data-error-for]).
  if (comments && comments.field_errors) {
    return Object.keys(comments.field_errors)
      .map(function (f) {
        return '<p class="comments-error">' + escapeHtml(comments.field_errors[f]) + "</p>";
      })
      .join("");
  }
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
