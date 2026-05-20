.PHONY: build test install uninstall clean help

ROOT := $(shell pwd)
SKILL_DIR := $(HOME)/.claude/skills/md2html
BIN := $(ROOT)/bin/md2html

help:
	@echo "make build    — compile src/ → bin/md2html"
	@echo "make test     — run go test ./..."
	@echo "make install  — symlink SKILL.md and bin/md2html into $(SKILL_DIR)"
	@echo "make uninstall— remove the symlinks (keeps source untouched)"
	@echo "make clean    — remove bin/md2html"

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

clean:
	rm -f $(BIN)
