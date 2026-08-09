// Config de los tests E2E. Levanta el ejemplo automaticamente (go run) y
// corre los tests contra un browser real (Chromium headless).
const { defineConfig } = require("@playwright/test");

module.exports = defineConfig({
  testDir: "./tests",
  timeout: 20000,
  retries: process.env.CI ? 1 : 0,
  use: {
    baseURL: "http://localhost:8081",
  },
  webServer: {
    // El ejemplo usa rutas relativas al cwd (http.Dir("static")) — hay que
    // correrlo desde su propia carpeta.
    command: "go run .",
    cwd: "./examples/social",
    port: 8081,
    reuseExistingServer: !process.env.CI,
    timeout: 30000,
  },
});
