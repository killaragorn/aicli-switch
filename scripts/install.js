#!/usr/bin/env node
"use strict";

const https = require("https");
const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");

const REPO = "killaragorn/aicli-switch";
const BIN_DIR = path.join(__dirname, "..", "bin");
const VERSION = require("../package.json").version;

const PLATFORM_MAP = {
  win32:  { os: "windows", arch: "amd64", ext: ".exe" },
  darwin: { os: "darwin",  arch: "amd64", ext: "" },
  linux:  { os: "linux",   arch: "amd64", ext: "" },
};

const ARM64_OVERRIDE = {
  darwin: { os: "darwin", arch: "arm64", ext: "" },
  linux:  { os: "linux",  arch: "arm64", ext: "" },
};

function getPlatform() {
  const platform = process.platform;
  const arch = process.arch;

  if (arch === "arm64" && ARM64_OVERRIDE[platform]) {
    return ARM64_OVERRIDE[platform];
  }

  const info = PLATFORM_MAP[platform];
  if (!info) {
    console.error(`Unsupported platform: ${platform}/${arch}`);
    console.error("Please build from source: go build -o aicli-switch .");
    process.exit(0); // Don't fail npm install
  }
  return info;
}

function downloadFile(url, dest) {
  return new Promise((resolve, reject) => {
    function follow(u) {
      https.get(u, { headers: { "User-Agent": "aicli-switch-npm" } }, (res) => {
        if (res.statusCode === 302 || res.statusCode === 301) {
          follow(res.headers.location);
          return;
        }
        if (res.statusCode !== 200) {
          reject(new Error(`Download failed: HTTP ${res.statusCode}`));
          return;
        }
        const file = fs.createWriteStream(dest);
        res.pipe(file);
        file.on("finish", () => { file.close(); resolve(); });
        file.on("error", reject);
      }).on("error", reject);
    }
    follow(url);
  });
}

async function main() {
  const { os: goos, arch, ext } = getPlatform();
  const binName = `aicli-switch${ext}`;
  const assetName = `aicli-switch-${goos}-${arch}${ext}`;
  const url = `https://github.com/${REPO}/releases/download/v${VERSION}/${assetName}`;
  const dest = path.join(BIN_DIR, binName);

  // Skip if binary already exists (e.g., local development)
  if (fs.existsSync(dest)) {
    console.log(`aicli-switch binary already exists at ${dest}`);
    return;
  }

  fs.mkdirSync(BIN_DIR, { recursive: true });
  console.log(`Downloading aicli-switch v${VERSION} for ${goos}/${arch}...`);
  console.log(`  From: ${url}`);

  try {
    await downloadFile(url, dest);

    // Make executable on Unix
    if (process.platform !== "win32") {
      fs.chmodSync(dest, 0o755);
    }

    console.log(`aicli-switch installed successfully!`);
  } catch (err) {
    console.error(`\nFailed to download pre-built binary: ${err.message}`);
    console.error(`\nYou can build from source instead:`);
    console.error(`  git clone https://github.com/${REPO}.git`);
    console.error(`  cd aicli-switch && go build -o aicli-switch .`);
    // Don't fail npm install - user can build manually
  }
}

main();
