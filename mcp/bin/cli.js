#!/usr/bin/env node
"use strict";

const { execFileSync } = require("child_process");
const path = require("path");
const fs = require("fs");

const ext = process.platform === "win32" ? ".exe" : "";
const binary = path.join(__dirname, `hudl-mcp${ext}`);

function isValidBinary(p) {
  try {
    const stat = fs.statSync(p);
    return stat.isFile() && stat.size > 0;
  } catch {
    return false;
  }
}

function run() {
  try {
    execFileSync(binary, process.argv.slice(2), {
      stdio: "inherit",
      env: process.env,
    });
  } catch (err) {
    if (err.status !== undefined) {
      process.exit(err.status);
    }
    throw err;
  }
}

if (isValidBinary(binary)) {
  run();
} else {
  console.error("hudl-mcp: binary not found, downloading automatically...");
  const { downloadBinary } = require("./download");
  downloadBinary()
    .then(() => run())
    .catch((err) => {
      console.error(`hudl-mcp: automatic download failed: ${err.message}`);
      console.error(
        "hudl-mcp: install manually from https://github.com/Huddle01/get-hudl/releases"
      );
      process.exit(1);
    });
}
