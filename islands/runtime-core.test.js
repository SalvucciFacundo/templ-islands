// Tests del runtime core (funciones puras). Correr con:
//
//	node --test islands/runtime-core.test.js
//
// El runtime.js NO se testea aqui (depende del DOM); solo las funciones
// puras que extrajo a islandsCore.
const { test } = require("node:test");
const assert = require("node:assert");
const core = require("./runtime-core.js");

test("fillPlaceholders reemplaza tokens con data-*", () => {
  assert.strictEqual(
    core.fillPlaceholders("/api/like/{post_id}", { "data-post-id": "7" }),
    "/api/like/7"
  );
});

test("fillPlaceholders convierte guion bajo a guion", () => {
  assert.strictEqual(
    core.fillPlaceholders("/api/{user_id}", { "data-user-id": "3" }),
    "/api/3"
  );
});

test("fillPlaceholders deja tokens sin dato intactos", () => {
  assert.strictEqual(core.fillPlaceholders("/api/{x}", {}), "/api/{x}");
});

test("optimisticValue inc suma el delta", () => {
  assert.strictEqual(core.optimisticValue({ op: "inc", delta: 1 }, "8"), "9");
  assert.strictEqual(core.optimisticValue({ op: "inc", delta: -1 }, "8"), "7");
});

test("optimisticValue inc con texto no numerico arranca de 0", () => {
  assert.strictEqual(core.optimisticValue({ op: "inc", delta: 1 }, "abc"), "1");
});

test("optimisticValue toggle-text alterna", () => {
  const f = { op: "toggle-text", true: "Liked", false: "Like" };
  assert.strictEqual(core.optimisticValue(f, "Like"), "Liked");
  assert.strictEqual(core.optimisticValue(f, "Liked"), "Like");
});

test("optimisticValue deja intacto lo que no es texto", () => {
  assert.strictEqual(core.optimisticValue({ op: "class-toggle" }, "x"), "x");
});

test("controlValue usa el input si existe", () => {
  assert.strictEqual(core.controlValue("hola", "q", {}), "hola");
  assert.strictEqual(core.controlValue("", "q", {}), "");
});

test("controlValue cae al data-<param> si no hay input", () => {
  assert.strictEqual(core.controlValue(undefined, "page", { page: "2" }), "2");
  assert.strictEqual(core.controlValue(undefined, "page", {}), "");
});

test("debounceMs usa el valor configurado", () => {
  assert.strictEqual(core.debounceMs("100", 300), 100);
  assert.strictEqual(core.debounceMs("500", 300), 500);
});

test("debounceMs cae al fallback con valores invalidos", () => {
  assert.strictEqual(core.debounceMs("abc", 300), 300);
  assert.strictEqual(core.debounceMs("-5", 300), 300);
});
