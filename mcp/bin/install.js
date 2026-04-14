#!/usr/bin/env node
"use strict";

const https = require("https");
const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");

const REPO = "Huddle01/get-hudl";
const BINARY = "hudl-mcp";
const INSTALL_DIR = path.join(__dirname, "..", "bin");

function getPlatform() {
  const platform = process.platform;
  const arch = process.arch;

  const osMap = { linux: "linux", darwin: "darwin", win32: "windows" };
  const archMap = { x64: "amd64", arm64: "arm64" };

  const os = osMap[platform];
  const cpu = archMap[arch];

  if (!os || !cpu) {
    console.error(`Unsupported platform: ${platform}/${arch}`);
    process.exit(1);
  }

  return { os, arch: cpu, ext: os === "windows" ? ".exe" : "" };
}

function fetch(url) {
  return new Promise((resolve, reject) => {
    https
      .get(url, { headers: { "User-Agent": "hudl-mcp-installer" } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          return fetch(res.headers.location).then(resolve, reject);
        }
        if (res.statusCode !== 200) {
          return reject(new Error(`HTTP ${res.statusCode} for ${url}`));
        }
        const chunks = [];
        res.on("data", (c) => chunks.push(c));
        res.on("end", () => resolve(Buffer.concat(chunks)));
        res.on("error", reject);
      })
      .on("error", reject);
  });
}

async function getLatestVersion() {
  const data = await fetch(`https://api.github.com/repos/${REPO}/releases/latest`);
  const release = JSON.parse(data.toString());
  return release.tag_name;
}

async function main() {
  try {
    const { os, arch, ext } = getPlatform();

    console.log(`hudl-mcp: detecting platform... ${os}/${arch}`);

    const version = await getLatestVersion();
    console.log(`hudl-mcp: latest version ${version}`);

    const binaryName = `${BINARY}-${os}-${arch}${ext}`;
    const url = `https://github.com/${REPO}/releases/download/${version}/${binaryName}`;
    const dest = path.join(INSTALL_DIR, `${BINARY}${ext}`);

    console.log(`hudl-mcp: downloading ${binaryName}...`);
    const binary = await fetch(url);

    fs.writeFileSync(dest, binary);
    fs.chmodSync(dest, 0o755);

    console.log(`hudl-mcp: installed to ${dest}`);
  } catch (err) {
    console.error(`hudl-mcp: installation failed: ${err.message}`);
    console.error(
      "hudl-mcp: you can install manually from https://github.com/Huddle01/get-hudl/releases"
    );
    process.exit(1);
  }
}

main();
