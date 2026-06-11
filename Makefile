SHELL := /bin/bash
BIN := ruleconv
RULE_REPO ?= ../Rule
CORE_DIR := $(CURDIR)/bin

# Core versions to fetch (linux-amd64). Keep in sync with ../Rule/.ruleconv/config.yaml.
MIHOMO_VERSION ?= v1.19.2
SINGBOX_VERSION ?= 1.13.13

.PHONY: all tidy build test vet check cores demo demo-nobins clean

all: build test vet

tidy:
	go mod tidy

build: tidy
	go build -o $(BIN) .

test:
	go test ./...

vet:
	go vet ./...

check: build test vet

# Download the proxy cores into ./bin (linux-amd64). Run once before `make demo`.
cores:
	mkdir -p $(CORE_DIR)
	curl -fsSL "https://github.com/MetaCubeX/mihomo/releases/download/$(MIHOMO_VERSION)/mihomo-linux-amd64-$(MIHOMO_VERSION).gz" -o /tmp/mihomo.gz
	gunzip -f /tmp/mihomo.gz && install -m 0755 /tmp/mihomo $(CORE_DIR)/mihomo
	curl -fsSL "https://github.com/SagerNet/sing-box/releases/download/v$(SINGBOX_VERSION)/sing-box-$(SINGBOX_VERSION)-linux-amd64.tar.gz" -o /tmp/sb.tgz
	tar -xzf /tmp/sb.tgz -C /tmp && install -m 0755 /tmp/sing-box-$(SINGBOX_VERSION)-linux-amd64/sing-box $(CORE_DIR)/sing-box
	$(CORE_DIR)/mihomo -v && $(CORE_DIR)/sing-box version

# Full sync against the Rule repo, compiling binaries (requires `make cores`).
demo: build
	MIHOMO_BIN=$(CORE_DIR)/mihomo SINGBOX_BIN=$(CORE_DIR)/sing-box ./$(BIN) sync --all --repo $(RULE_REPO)

# Sync sources + READMEs only, no cores needed.
demo-nobins: build
	./$(BIN) sync --all --repo $(RULE_REPO) --skip-binary

clean:
	rm -f $(BIN)
	rm -rf $(CORE_DIR)
