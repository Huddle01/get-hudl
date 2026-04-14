MODULE   := github.com/Huddle01/get-hudl
CLI_CMD  := ./cli/cmd/hudl
MCP_CMD  := ./mcp/cmd/hudl-mcp
BIN_NAME := hudl
MCP_NAME := hudl-mcp
DIST_DIR := dist

VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS  := -ldflags "-s -w -X main.version=$(VERSION)"

# Cross-compile targets (os/arch)
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build build-mcp build-all dev test clean release changelog version tag delete-tag dist \
       mcp-version mcp-pack mcp-publish mcp-publish-dry mcp-publish-dev mcp-publish-beta mcp-smoke

## build: compile the CLI for the current platform
build:
	go build $(LDFLAGS) -o $(BIN_NAME) $(CLI_CMD)

## build-mcp: compile the MCP server for the current platform
build-mcp:
	go build $(LDFLAGS) -o $(MCP_NAME) $(MCP_CMD)

## build-all: compile both CLI and MCP server
build-all: build build-mcp

## dev: build and run the CLI
dev: build
	./$(BIN_NAME)

## test: run tests
test:
	go test ./...

## clean: remove build artifacts
clean:
	rm -rf $(BIN_NAME) $(MCP_NAME) $(DIST_DIR)

## version: print current version and suggest next versions
version:
	@CURRENT=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	MAJOR=$$(echo $$CURRENT | sed 's/^v//' | cut -d. -f1); \
	MINOR=$$(echo $$CURRENT | sed 's/^v//' | cut -d. -f2); \
	PATCH=$$(echo $$CURRENT | sed 's/^v//' | cut -d. -f3); \
	NEXT_PATCH="v$$MAJOR.$$MINOR.$$((PATCH + 1))"; \
	NEXT_MINOR="v$$MAJOR.$$((MINOR + 1)).0"; \
	NEXT_MAJOR="v$$((MAJOR + 1)).0.0"; \
	COMMITS=$$(if [ "$$CURRENT" != "v0.0.0" ]; then git log --oneline $$CURRENT..HEAD 2>/dev/null | wc -l | tr -d ' '; else git log --oneline 2>/dev/null | wc -l | tr -d ' '; fi); \
	echo ""; \
	echo "  current     $$CURRENT"; \
	echo "  commits     $$COMMITS since last tag"; \
	echo ""; \
	echo "  patch       make release v=$$NEXT_PATCH   (bug fixes)"; \
	echo "  minor       make release v=$$NEXT_MINOR   (new features)"; \
	echo "  major       make release v=$$NEXT_MAJOR   (breaking changes)"; \
	echo ""

## changelog: generate changelog from git commits since last tag
changelog:
	@PREV_TAG=$$(git describe --tags --abbrev=0 HEAD^ 2>/dev/null || echo ""); \
	if [ -z "$$PREV_TAG" ]; then \
		echo "## Changelog (all commits)"; \
		echo ""; \
		git log --pretty=format:"- %s (%h)" --no-merges; \
	else \
		echo "## Changelog (since $$PREV_TAG)"; \
		echo ""; \
		git log --pretty=format:"- %s (%h)" --no-merges $$PREV_TAG..HEAD; \
	fi

## dist: cross-compile CLI and MCP server for all platforms
dist: clean
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		out="$(DIST_DIR)/$(BIN_NAME)-$$os-$$arch$$ext"; \
		echo "Building $$out ..."; \
		GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o $$out $(CLI_CMD); \
		mcp_out="$(DIST_DIR)/$(MCP_NAME)-$$os-$$arch$$ext"; \
		echo "Building $$mcp_out ..."; \
		GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o $$mcp_out $(MCP_CMD); \
	done

## tag: create a new git tag (usage: make tag v=v1.0.0)
tag:
	@if [ -z "$(v)" ]; then echo "Usage: make tag v=v1.0.0"; exit 1; fi
	git tag -a "$(v)" -m "Release $(v)"
	@echo "Created tag $(v). Push with: git push origin $(v)"

## delete-tag: delete a tag locally and remotely (usage: make delete-tag v=v1.0.0)
delete-tag:
	@if [ -z "$(v)" ]; then echo "Usage: make delete-tag v=v1.0.0"; exit 1; fi
	@echo "Deleting tag $(v) locally and remotely..."
	-git tag -d "$(v)"
	-git push origin --delete "$(v)"
	@echo "Tag $(v) deleted."

# ─── MCP npm package targets ─────────────────────────────────────────────────

MCP_PKG_DIR := mcp

## mcp-version: show current npm package version
mcp-version:
	@node -p "require('./$(MCP_PKG_DIR)/package.json').version"

## mcp-pack: create a tarball for inspection (does not publish)
mcp-pack:
	cd $(MCP_PKG_DIR) && npm pack --dry-run 2>&1 | head -40
	@echo ""
	@echo "To create the actual tarball: cd $(MCP_PKG_DIR) && npm pack"

## mcp-publish-dry: full publish dry run — shows exactly what would be uploaded
mcp-publish-dry:
	cd $(MCP_PKG_DIR) && npm publish --access public --dry-run

## mcp-publish: interactive publish — suggests version, syncs, and publishes to npm
mcp-publish:
	@CURRENT=$$(node -p "require('./$(MCP_PKG_DIR)/package.json').version"); \
	GIT_TAG=$$(git describe --tags --abbrev=0 2>/dev/null || echo ""); \
	GIT_VER=$$(echo "$$GIT_TAG" | sed 's/^v//'); \
	MAJOR=$$(echo "$$CURRENT" | cut -d. -f1); \
	MINOR=$$(echo "$$CURRENT" | cut -d. -f2); \
	PATCH=$$(echo "$$CURRENT" | cut -d. -f3); \
	NEXT_PATCH="$$MAJOR.$$MINOR.$$((PATCH + 1))"; \
	NEXT_MINOR="$$MAJOR.$$((MINOR + 1)).0"; \
	echo ""; \
	echo "  @huddle01/mcp"; \
	echo ""; \
	echo "  current npm     $$CURRENT"; \
	if [ -n "$$GIT_VER" ]; then echo "  latest git tag   $$GIT_VER"; fi; \
	echo ""; \
	echo "  Suggestions:"; \
	echo "      patch  →  $$NEXT_PATCH"; \
	echo "      minor  →  $$NEXT_MINOR"; \
	if [ -n "$$GIT_VER" ] && [ "$$GIT_VER" != "$$CURRENT" ]; then \
		echo "      sync   →  $$GIT_VER  (match git tag)"; \
	fi; \
	echo ""; \
	printf "  New version [$$NEXT_PATCH]: "; \
	read INPUT_VERSION; \
	VERSION=$${INPUT_VERSION:-$$NEXT_PATCH}; \
	echo ""; \
	cd $(MCP_PKG_DIR) && npm version "$$VERSION" --no-git-tag-version --allow-same-version; \
	echo ""; \
	echo "Publishing @huddle01/mcp@$$VERSION to npm..."; \
	echo ""; \
	if cd $(MCP_PKG_DIR) && npm publish --access public; then \
		echo ""; \
		echo "  @huddle01/mcp@$$VERSION published!"; \
		echo ""; \
		echo "  Users can now run:"; \
		echo "    claude mcp add @huddle01/mcp -- npx -y @huddle01/mcp"; \
	else \
		echo ""; \
		echo "  Publish failed. Check:"; \
		echo "    1. npm login         (are you logged in?)"; \
		echo "    2. npm org ls huddle01  (does the @huddle01 org exist?)"; \
		echo "    3. npm access        (do you have publish rights?)"; \
		exit 1; \
	fi; \
	echo ""

## mcp-publish-dev: publish a dev pre-release to npm under the "dev" tag
mcp-publish-dev:
	@CURRENT=$$(node -p "require('./$(MCP_PKG_DIR)/package.json').version" | sed 's/-.*//' ); \
	COMMIT=$$(git rev-parse --short HEAD); \
	DEV_VERSION="$$CURRENT-dev.$$COMMIT"; \
	echo ""; \
	echo "  Publishing dev pre-release..."; \
	echo "  version:  $$DEV_VERSION"; \
	echo "  tag:      dev"; \
	echo ""; \
	cd $(MCP_PKG_DIR) && npm version "$$DEV_VERSION" --no-git-tag-version --allow-same-version; \
	if npm publish --access public --tag dev; then \
		echo ""; \
		echo "  @huddle01/mcp@$$DEV_VERSION published (tag: dev)"; \
		echo ""; \
		echo "  Install with:"; \
		echo "    npx @huddle01/mcp@dev"; \
		echo "    npm i @huddle01/mcp@dev"; \
	else \
		echo ""; \
		echo "  Publish failed. Run 'npm login' and check @huddle01 org exists."; \
		exit 1; \
	fi; \
	echo ""

## mcp-publish-beta: publish a beta pre-release to npm under the "beta" tag
mcp-publish-beta:
	@CURRENT=$$(node -p "require('./$(MCP_PKG_DIR)/package.json').version" | sed 's/-.*//' ); \
	TIMESTAMP=$$(date +%Y%m%d%H%M%S); \
	BETA_VERSION="$$CURRENT-beta.$$TIMESTAMP"; \
	echo ""; \
	echo "  Publishing beta pre-release..."; \
	echo "  version:  $$BETA_VERSION"; \
	echo "  tag:      beta"; \
	echo ""; \
	cd $(MCP_PKG_DIR) && npm version "$$BETA_VERSION" --no-git-tag-version --allow-same-version; \
	if npm publish --access public --tag beta; then \
		echo ""; \
		echo "  @huddle01/mcp@$$BETA_VERSION published (tag: beta)"; \
		echo ""; \
		echo "  Install with:"; \
		echo "    npx @huddle01/mcp@beta"; \
		echo "    npm i @huddle01/mcp@beta"; \
	else \
		echo ""; \
		echo "  Publish failed. Run 'npm login' and check @huddle01 org exists."; \
		exit 1; \
	fi; \
	echo ""

## mcp-smoke: build MCP binary and verify the npm wrapper launches it
mcp-smoke: build-mcp
	@echo "Smoke testing npm wrapper..."
	@cp $(MCP_NAME) $(MCP_PKG_DIR)/bin/$(MCP_NAME)
	@echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"smoke","version":"1.0"}}}' \
		| node $(MCP_PKG_DIR)/bin/cli.js 2>/dev/null \
		| node -e "const d=JSON.parse(require('fs').readFileSync(0,'utf8'));if(d.result.serverInfo.name!=='hudl-mcp'){process.exit(1)}"
	@rm -f $(MCP_PKG_DIR)/bin/$(MCP_NAME)
	@echo "mcp-smoke: ok — server initialized via npm wrapper"

# ─── Release ─────────────────────────────────────────────────────────────────

## release: interactive release — builds, tags, and creates GitHub release
release:
	@CURRENT=$$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"); \
	MAJOR=$$(echo $$CURRENT | sed 's/^v//' | cut -d. -f1); \
	MINOR=$$(echo $$CURRENT | sed 's/^v//' | cut -d. -f2); \
	PATCH=$$(echo $$CURRENT | sed 's/^v//' | cut -d. -f3); \
	NEXT_PATCH="v$$MAJOR.$$MINOR.$$((PATCH + 1))"; \
	NEXT_MINOR="v$$MAJOR.$$((MINOR + 1)).0"; \
	echo ""; \
	echo "Current version: $$CURRENT"; \
	echo ""; \
	echo "Suggestions:"; \
	echo "    patch  →  $$NEXT_PATCH"; \
	echo "    minor  →  $$NEXT_MINOR"; \
	echo ""; \
	printf "New version [$$NEXT_PATCH]: "; \
	read INPUT_VERSION; \
	VERSION=$${INPUT_VERSION:-$$NEXT_PATCH}; \
	echo ""; \
	if git rev-parse "$$VERSION" >/dev/null 2>&1; then \
		echo "Tag $$VERSION already exists!"; \
		echo ""; \
		printf "Delete existing tag and release? [y/N] "; \
		read confirm; \
		if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
			echo "Deleting existing release and tag..."; \
			gh release delete "$$VERSION" --yes 2>/dev/null || true; \
			git tag -d "$$VERSION" 2>/dev/null || true; \
			git push origin --delete "$$VERSION" 2>/dev/null || true; \
			echo "Deleted."; \
		else \
			echo "Aborted."; \
			exit 1; \
		fi; \
	fi; \
	echo "Building $$VERSION..."; \
	$(MAKE) dist VERSION=$$VERSION; \
	echo ""; \
	echo "Generating changelog..."; \
	if [ "$$CURRENT" = "v0.0.0" ]; then \
		CHANGELOG=$$(git log --pretty=format:"- %s (%h)" --no-merges); \
	else \
		CHANGELOG=$$(git log --pretty=format:"- %s (%h)" --no-merges $$CURRENT..HEAD); \
	fi; \
	echo "$$CHANGELOG"; \
	echo ""; \
	echo "Creating tag $$VERSION..."; \
	git tag -a "$$VERSION" -m "Release $$VERSION"; \
	git push origin "$$VERSION"; \
	echo "Creating GitHub release..."; \
	gh release create "$$VERSION" $(DIST_DIR)/* \
		--title "$$VERSION" \
		--notes "$$CHANGELOG" \
		--latest; \
	echo ""; \
	echo "Release $$VERSION published!"; \
	echo ""; \
	NPM_VER=$$(echo "$$VERSION" | sed 's/^v//'); \
	printf "Publish @huddle01/mcp@$$NPM_VER to npm? [Y/n] "; \
	read npm_confirm; \
	if [ "$$npm_confirm" != "n" ] && [ "$$npm_confirm" != "N" ]; then \
		cd $(MCP_PKG_DIR) && npm version "$$NPM_VER" --no-git-tag-version --allow-same-version; \
		if npm publish --access public; then \
			echo ""; \
			echo "@huddle01/mcp@$$NPM_VER published to npm!"; \
		else \
			echo ""; \
			echo "npm publish failed. Run 'make mcp-publish' to retry."; \
		fi; \
	else \
		echo "Skipped npm publish. Run 'make mcp-publish' later."; \
	fi
