#!/usr/bin/env node
"use strict";

const https = require("https");
const fs = require("fs");
const path = require("path");
const os = require("os");

const REPO = "Huddle01/get-hudl";
const BINARY = "hudl-mcp";
const INSTALL_DIR = __dirname;
const MAX_REDIRECTS = 5;
const TIMEOUT_MS = 30_000;

function getPlatform() {
  const platform = process.platform;
  const arch = process.arch;

  const osMap = { linux: "linux", darwin: "darwin", win32: "windows" };
  const archMap = { x64: "amd64", arm64: "arm64" };

  const osName = osMap[platform];
  const cpu = archMap[arch];

  if (!osName || !cpu) {
    throw new Error(`Unsupported platform: ${platform}/${arch}`);
  }

  return { os: osName, arch: cpu, ext: osName === "windows" ? ".exe" : "" };
}

function fetch(url, redirects = 0) {
  if (redirects > MAX_REDIRECTS) {
    return Promise.reject(new Error(`Too many redirects (>${MAX_REDIRECTS})`));
  }
  return new Promise((resolve, reject) => {
    const req = https
      .get(url, { headers: { "User-Agent": "hudl-mcp-installer" }, timeout: TIMEOUT_MS }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          res.resume();
          return fetch(res.headers.location, redirects + 1).then(resolve, reject);
        }
        if (res.statusCode === 403 || res.statusCode === 429) {
          res.resume();
          return reject(
            new Error(
              `HTTP ${res.statusCode} for ${url} — likely GitHub API rate limit. ` +
              `Try again later or install manually from https://github.com/${REPO}/releases`
            )
          );
        }
        if (res.statusCode !== 200) {
          res.resume();
          return reject(new Error(`HTTP ${res.statusCode} for ${url}`));
        }
        const chunks = [];
        res.on("data", (c) => chunks.push(c));
        res.on("end", () => resolve(Buffer.concat(chunks)));
        res.on("error", reject);
      })
      .on("error", reject)
      .on("timeout", () => {
        req.destroy();
        reject(new Error(`Request timed out after ${TIMEOUT_MS / 1000}s: ${url}`));
      });
  });
}

async function getLatestVersion() {
  const data = await fetch(`https://api.github.com/repos/${REPO}/releases/latest`);
  const release = JSON.parse(data.toString());
  return release.tag_name;
}

async function downloadBinary() {
  const { os: osName, arch, ext } = getPlatform();

  console.error(`hudl-mcp: detecting platform... ${osName}/${arch}`);

  const version = await getLatestVersion();
  console.error(`hudl-mcp: latest version ${version}`);

  const binaryName = `${BINARY}-${osName}-${arch}${ext}`;
  const url = `https://github.com/${REPO}/releases/download/${version}/${binaryName}`;
  const dest = path.join(INSTALL_DIR, `${BINARY}${ext}`);

  console.error(`hudl-mcp: downloading ${binaryName}...`);
  const binary = await fetch(url);

  if (binary.length === 0) {
    throw new Error("Downloaded binary is empty");
  }

  // Write to a temp file first, then atomically rename to avoid corrupt partial writes
  const tmp = path.join(os.tmpdir(), `hudl-mcp-${process.pid}-${Date.now()}${ext}`);
  try {
    fs.writeFileSync(tmp, binary);
    fs.chmodSync(tmp, 0o755);
    fs.renameSync(tmp, dest);
  } catch (err) {
    // renameSync fails across devices — fall back to copy + unlink
    if (err.code === "EXDEV") {
      fs.copyFileSync(tmp, dest);
      fs.chmodSync(dest, 0o755);
      fs.unlinkSync(tmp);
    } else {
      try { fs.unlinkSync(tmp); } catch (_) {}
      throw err;
    }
  }

  console.error(`hudl-mcp: installed to ${dest}`);
  return dest;
}

module.exports = { downloadBinary };
