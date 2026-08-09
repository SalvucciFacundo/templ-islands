// Renderer client del chat (form submit + stream SSE).
//
// Registra dos islas que comparten el mismo renderer:
//   - "chat-form":   el form submit re-renderiza al enviar.
//   - "chat-stream": el EventSource re-renderiza cuando llega un evento SSE
//     (mensajes del agente que llegan solos).
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

islandsRenderers["chat-form"] = renderChat;
islandsRenderers["chat-stream"] = renderChat;
