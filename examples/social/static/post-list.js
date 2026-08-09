// Renderer client de la isla post-list (re-render).
//
// Convert JSON posts (same contract as views.Post) into the SAME markup that
// templ renders for views.PostCard. The parity test (parity_test.go) keeps
// these two renderers in sync: if this HTML drifts from templ, the test fails.
window.islandsRenderers = window.islandsRenderers || {};
window.islandsRenderers["post-list"] = function (posts) {
  return posts
    .map(function (p) {
      var liked = p.liked ? " liked" : "";
      var following = p.following ? " following" : "";
      var likeLabel = p.liked ? "Liked" : "Like";
      var followLabel = p.following ? "Following" : "Follow";

      return (
        '<div class="post">' +
        '<div class="post-header">' +
        '<span class="author">Autor ' + p.author_id + "</span>" +
        '<button class="follow-btn' + following + '" data-island="follow" data-key="author-' + p.author_id + '" data-user-id="' + p.author_id + '" hx-post="/follow/' + p.author_id + '" hx-swap="outerHTML" hx-disabled-elt="this">' +
        '<span class="follow-label" data-mutate="follow-label">' + followLabel + "</span>" +
        "</button>" +
        "</div>" +
        '<p class="post-text">' + escapeHtml(p.text) + "</p>" +
        '<div class="post-actions">' +
        '<button class="like-btn' + liked + '" data-island="like" data-key="post-' + p.id + '" data-post-id="' + p.id + '" hx-post="/like/' + p.id + '" hx-swap="outerHTML" hx-disabled-elt="this">' +
        '<span class="label" data-mutate="label">' + likeLabel + "</span> " +
        '<span class="count" data-mutate="likes">' + p.likes + "</span>" +
        "</button>" +
        "</div>" +
        '<div class="comments">' +
        '<button class="comments-toggle" data-island="comments" data-trigger="click" data-post-id="' + p.id + '" data-target="#comments-' + p.id + '">Ver comentarios</button>' +
        '<div id="comments-' + p.id + '" class="comments-box"></div>' +
        "</div>" +
        "</div>"
      );
    })
    .join("");
};

// escapeHtml viene del runtime core (islandsCore) — se expone global.
