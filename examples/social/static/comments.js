// Renderer client de las islas de comentarios (re-render por click).
//
// "comments" y "delete-comment" comparten este renderer: la isla
// delete-comment declara renderer=comments en su directiva @island, asi el
// renderer se registra UNA sola vez (contrato de renderer compartido).
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
