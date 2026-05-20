.PHONY: build test install uninstall install-cli uninstall-cli clean help

ROOT := $(shell pwd)
SKILL_DIR := $(HOME)/.claude/skills/md2html
BIN := $(ROOT)/bin/md2html
PREFIX ?= $(HOME)/.local
CLI_LINK := $(PREFIX)/bin/md2html

help:
	@echo "make build         — compile src/ → bin/md2html"
	@echo "make test          — run go test ./..."
	@echo "make install       — symlink SKILL.md and bin/md2html into $(SKILL_DIR)"
	@echo "make uninstall     — remove the skill symlinks (keeps source untouched)"
	@echo "make install-cli   — symlink bin/md2html into $(CLI_LINK) (override with PREFIX=...)"
	@echo "make uninstall-cli — remove the CLI symlink"
	@echo "make clean         — remove bin/md2html"

build:
	cd src && go build -o $(BIN) .

test:
	cd src && go test ./...

install: build
	mkdir -p $(SKILL_DIR)/bin
	ln -sf $(ROOT)/SKILL.md $(SKILL_DIR)/SKILL.md
	ln -sf $(BIN) $(SKILL_DIR)/bin/md2html
	@echo "installed → $(SKILL_DIR)"

uninstall:
	rm -f $(SKILL_DIR)/SKILL.md $(SKILL_DIR)/bin/md2html
	@rmdir $(SKILL_DIR)/bin $(SKILL_DIR) 2>/dev/null || true

install-cli: build
	mkdir -p $(PREFIX)/bin
	ln -sf $(BIN) $(CLI_LINK)
	@echo "installed → $(CLI_LINK)"

uninstall-cli:
	rm -f $(CLI_LINK)
	@echo "removed → $(CLI_LINK)"

clean:
	rm -f $(BIN)
