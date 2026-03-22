#!/usr/bin/env node
"use strict";

const { execFileSync } = require("child_process");
const path = require("path");
const fs = require("fs");

const ext = process.platform === "win32" ? ".exe" : "";
const binPath = path.join(__dirname, `aicli-switch${ext}`);

if (!fs.existsSync(binPath)) {
  console.error("aicli-switch binary not found. Try reinstalling:");
  console.error("  npm install -g aicli-switch");
  process.exit(1);
}

try {
  const result = execFileSync(binPath, process.argv.slice(2), {
    stdio: "inherit",
    windowsHide: true,
  });
} catch (err) {
  process.exit(err.status || 1);
}
