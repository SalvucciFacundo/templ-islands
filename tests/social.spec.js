// E2E del ejemplo social en un browser real (Chromium).
//
// Estos tests son los que faltaban: verifican el COMPORTAMIENTO real
// (optimistic UI, re-render, confirm, SSE, infinite scroll), no solo la
// paridad del render. Un bug como el del post-more (renderer inexistente)
// fallaba solo en browser — estos tests lo cazarian.
const { test, expect } = require("@playwright/test");

test("like: optimistic mutation with shared key and rollback target", async ({ page }) => {
  await page.goto("/");
  const first = page.locator(".like-btn").first();
  const count = page.locator(".like-btn .count").first();
  const before = parseInt(await count.textContent(), 10);

  // optimistic: el contador sube al instante (sin esperar el server)
  await first.click();
  await expect(count).toHaveText(String(before + 1));
  await expect(first).toHaveClass(/liked/);

  // toggle: segundo click lo deshace
  await first.click();
  await expect(count).toHaveText(String(before));
  await expect(first).not.toHaveClass(/liked/);
});

test("search: re-render filters the feed (data-debounce)", async ({ page }) => {
  await page.goto("/");
  await expect(page.locator(".post").first()).toBeVisible();

  await page.fill(".search", "3");
  // debounce 300ms + fetch + re-render: el feed queda solo con el post #3
  await expect(page.locator(".post")).toHaveCount(1);
  await expect(page.locator(".post-text").first()).toContainText("#3");
});

test("comments: click re-render expands, delete asks confirm", async ({ page }) => {
  await page.goto("/");
  await page.locator(".comments-toggle").first().click();
  await expect(page.locator(".comment")).toHaveCount(2);

  // el delete pregunta (data-confirm) y el dialogo se acepta
  page.on("dialog", (d) => d.accept());
  await page.locator(".comment-delete").first().click();
  await expect(page.locator(".comment")).toHaveCount(1);
});

test("chat: form submit sends and SSE delivers the agent reply alone", async ({ page }) => {
  await page.goto("/chat");

  await page.fill(".chat-form input", "hola e2e");
  await page.locator(".chat-form button").click();

  // el mensaje del usuario aparece al instante (form submit re-render)
  await expect(page.locator(".chat-msg.user").last()).toContainText("hola e2e");

  // la respuesta del agente llega SOLA por SSE, sin que el usuario haga nada
  await expect(page.locator(".chat-msg.agent").last()).toContainText("Procesado: hola e2e", {
    timeout: 8000,
  });
});

test("field errors: empty post shows inline error", async ({ page }) => {
  await page.goto("/");
  // el input es required (HTML5) — llenamos con un espacio que el server
  // trimea a vacio y responde field_errors
  await page.locator(".new-post input[type=text]").fill(" ");
  await page.locator(".new-post button").click();
  await expect(page.locator("[data-error-for=text]")).toHaveClass(/show/);
  await expect(page.locator(".new-post input[type=text]")).toHaveClass(/invalid/);
});

test("renderers: field errors from a re-render render inline", async ({ page }) => {
  await page.goto("/");

  // El click de "Ver comentarios" hace GET /api/comments/1. Interceptamos la
  // respuesta: el server "devuelve" 400 con field_errors y el renderer los
  // pinta inline dentro del target (patron de renderers, no es un form).
  await page.route("**/api/comments/*", (route) =>
    route.fulfill({
      status: 400,
      contentType: "application/json",
      body: JSON.stringify({ field_errors: { text: "Comentario invalido" } }),
    })
  );

  await page.locator(".comments-toggle").first().click();
  await expect(page.locator(".comments-error").first()).toContainText("Comentario invalido");
});

test("upload: file input previews instantly and the post carries the image", async ({ page }) => {
  await page.goto("/");

  // PNG real de 1x1 para que el server lo valide por contenido
  const png = Buffer.from(
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==",
    "base64"
  );

  // al elegir el archivo aparece la vista previa optimista (sin subir nada)
  await page.setInputFiles(".upload-label input[type=file]", {
    name: "foto.png",
    mimeType: "image/png",
    buffer: png,
  });
  await expect(page.locator("#new-post-previews img")).toHaveCount(1);

  // publicar: el submit va como multipart y el post aparece con la imagen
  await page.fill(".new-post input[type=text]", "post con foto");
  await page.locator(".new-post button").click();
  await expect(page.locator(".post-text").last()).toContainText("post con foto");

  const img = page.locator(".post").last().locator(".post-image");
  await expect(img).toBeVisible();
  const src = await img.getAttribute("src");
  expect(src).toMatch(/^\/static\/uploads\//);
});

test("multi-tab: like in one tab syncs to the other instantly", async ({ page }) => {
  // segunda pestana del MISMO context: comparte el BroadcastChannel
  const pageB = await page.context().newPage();
  await page.goto("/");
  await pageB.goto("/");
  await expect(page.locator(".like-btn .count").first()).toBeVisible();
  await expect(pageB.locator(".like-btn .count").first()).toBeVisible();

  const countA = page.locator(".like-btn .count").first();
  const countB = pageB.locator(".like-btn .count").first();
  const before = parseInt(await countB.textContent(), 10);

  await page.locator(".like-btn").first().click();

  // A: optimista + server
  await expect(countA).toHaveText(String(before + 1));
  // B: la respuesta del server llega por el canal, sin recargar
  await expect(countB).toHaveText(String(before + 1));
  await expect(pageB.locator(".like-btn").first()).toHaveClass(/liked/);
});

test("multi-tab: new post in one tab refreshes the feed in the other", async ({ page }) => {
  const pageB = await page.context().newPage();
  await page.goto("/");
  await pageB.goto("/");
  await expect(page.locator(".new-post input[type=text]")).toBeVisible();
  await expect(pageB.locator(".post").first()).toBeVisible();

  await page.fill(".new-post input[type=text]", "post multi-tab");
  await page.locator(".new-post button").click();

  await expect(page.locator(".post-text", { hasText: "post multi-tab" })).toBeVisible();
  // B se entera por refresh y re-fetchea su feed
  await expect(pageB.locator(".post-text", { hasText: "post multi-tab" })).toBeVisible();
});

test("infinite scroll: sentinel appends pages until the end", async ({ page }) => {
  await page.goto("/");
  // Con un feed corto, el sentinel puede auto-disparar al cargar (rootMargin
  // 200px). Esperamos que se estabilice y capturamos el estado inicial.
  await page.waitForTimeout(1200);
  const initial = await page.locator(".post").count();
  expect(initial).toBeGreaterThan(0);

  await page.locator(".sentinel").scrollIntoViewIfNeeded();
  // El intersect agrega la pagina siguiente; el count debe crecer.
  await expect(async () => {
    const n = await page.locator(".post").count();
    expect(n).toBeGreaterThan(initial);
  }).toPass({ timeout: 8000 });
});
