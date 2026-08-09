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
  await page.locator(".new-post input").fill(" ");
  await page.locator(".new-post button").click();
  await expect(page.locator("[data-error-for=text]")).toHaveClass(/show/);
  await expect(page.locator(".new-post input")).toHaveClass(/invalid/);
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
