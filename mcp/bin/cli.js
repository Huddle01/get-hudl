#!/usr/bin/env node
"use strict";

const { execFileSync } = require("child_process");
const path = require("path");
const fs = require("fs");

const ext = process.platform === "win32" ? ".exe" : "";
const binary = path.join(__dirname, `hudl-mcp${ext}`);

if (!fs.existsSync(binary)) {
  console.error(
    "hudl-mcp: binary not found. Run `npm rebuild @huddle01/mcp` or reinstall the package."
  );
  process.exit(1);
}

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
