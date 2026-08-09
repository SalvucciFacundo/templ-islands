// parity_runner.js — harness del golden test de paridad.
//
// Lee los fixtures JSON de stdin (mismo contrato que views.Post), corre el
// renderer client de post-list y emite el HTML generado por stdout.
// parity_test.go compara ese HTML con el que renderiza templ.
const fs = require("fs");

// El renderer espera window.islandsRenderers; simulamos el entorno del browser.
global.window = {};

eval(fs.readFileSync("static/post-list.js", "utf8"));

const data = JSON.parse(fs.readFileSync(0, "utf8"));
process.stdout.write(window.islandsRenderers["post-list"](data));
