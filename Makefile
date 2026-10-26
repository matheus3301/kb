.PHONY: tools check repo lint test build fuzz security acceptance release-check release-snapshot release benchmark benchmark-smoke retrieval-eval soak-smoke site-check

# Match distributed CLI builds without changing the race-test environment.
CLI_BUILD_ENV := CGO_ENABLED=0 GOWORK=off GOTOOLCHAIN=local

export PATH := $(CURDIR)/.tools/bin:$(PATH)
PYTHON ?= python3
GORELEASER ?= $(if $(wildcard $(CURDIR)/.tools/bin/goreleaser),$(CURDIR)/.tools/bin/goreleaser,goreleaser)

tools:
	$(PYTHON) scripts/quality.py install-tools

repo:
	$(PYTHON) -m unittest discover -s scripts -p 'test_*.py' -v
	$(PYTHON) scripts/quality.py repo

lint:
	$(PYTHON) scripts/quality.py static

test:
	$(PYTHON) scripts/quality.py test

build:
	$(PYTHON) scripts/quality.py build

fuzz:
	$(PYTHON) scripts/quality.py fuzz

security:
	$(PYTHON) scripts/quality.py security

acceptance:
	$(CLI_BUILD_ENV) go build -trimpath -o .build/kb ./cmd/kb
	$(PYTHON) scripts/acceptance.py --binary .build/kb --output .build/acceptance.json
	$(PYTHON) scripts/check_docs.py --binary .build/kb

benchmark:
	$(CLI_BUILD_ENV) go build -trimpath -o .build/kb ./cmd/kb
	$(PYTHON) scripts/benchmark.py --binary .build/kb --output .build/performance.json

benchmark-smoke:
	$(CLI_BUILD_ENV) go build -trimpath -o .build/kb ./cmd/kb
	$(PYTHON) scripts/benchmark.py --binary .build/kb --sizes 100 --samples 2 --repair-samples 1 --output .build/performance-smoke.json

retrieval-eval:
	$(CLI_BUILD_ENV) go build -trimpath -o .build/kb ./cmd/kb
	$(PYTHON) scripts/retrieval_eval.py --binary .build/kb --output .build/retrieval-quality.json

soak-smoke:
	$(CLI_BUILD_ENV) go build -trimpath -o .build/kb ./cmd/kb
	$(PYTHON) scripts/soak.py --binary .build/kb --duration 3 --seeds 1729 --writers 4 --minimum-work 1 --backup-interval 1 --output .build/soak-smoke.json

# The static marketing site has its own required CI job, separate from Go runtime gates.
site-check:
	npm --prefix site ci
	npm --prefix site audit --audit-level=high
	npm --prefix site run check
	KB_SITE_BASE=/kb/ npm --prefix site run build
	cd site && npm exec -- playwright install chromium
	KB_SITE_BASE=/kb/ npm --prefix site test

check: repo lint test build fuzz security acceptance benchmark-smoke retrieval-eval soak-smoke

release-check:
	$(GORELEASER) check

release-snapshot: release-check
	$(GORELEASER) release --snapshot --skip=publish --clean
	$(PYTHON) scripts/release.py formula
	$(PYTHON) scripts/check_release.py

release:
	@echo "Push a new version tag from tested main; see docs/releases.md."
	@exit 1
