// Renderer client del chat (form submit + stream SSE).
//
// "chat-form" declara renderer=chat-stream, asi el renderer se registra una
// sola vez bajo "chat-stream" y ambas islas lo reusan.
window.islandsRenderers = window.islandsRenderers || {};

function renderChat(data) {
  if (!data.messages || !data.messages.length) {
    return '<p class="chat-empty">Sin mensajes todavia.</p>';
  }
  return data.messages
    .map(function (m) {
      var cls = m.from === "agent" ? "agent" : "user";
      return (
        '<div class="chat-msg ' + cls + '">' +
        '<span class="chat-who">' + escapeHtml(m.from) + ":</span> " +
        escapeHtml(m.text) +
        "</div>"
      );
    })
    .join("");
}

islandsRenderers["chat-stream"] = renderChat;
