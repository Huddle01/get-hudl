#!/usr/bin/env node
"use strict";

const { downloadBinary } = require("./download");

downloadBinary().catch((err) => {
  console.error(`hudl-mcp: installation failed: ${err.message}`);
  console.error(
    "hudl-mcp: you can install manually from https://github.com/Huddle01/get-hudl/releases"
  );
  process.exit(1);
});
