MODULE   := github.com/Huddle01/get-hudl
CMD_PATH := ./cli/cmd/hudl
BIN_NAME := hudl
DIST_DIR := dist

VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS  := -ldflags "-s -w -X main.version=$(VERSION)"

# Cross-compile targets (os/arch)
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build dev test clean release changelog version tag delete-tag dist

## build: compile for the current platform
build:
	go build $(LDFLAGS) -o $(BIN_NAME) $(CMD_PATH)

## dev: build and run
dev: build
	./$(BIN_NAME)

## test: run tests
test:
	go test ./...

## clean: remove build artifacts
clean:
	rm -rf $(BIN_NAME) $(DIST_DIR)

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

## dist: cross-compile for all platforms
dist: clean
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		out="$(DIST_DIR)/$(BIN_NAME)-$$os-$$arch$$ext"; \
		echo "Building $$out ..."; \
		GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o $$out $(CMD_PATH); \
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
	echo "Release $$VERSION published!"
