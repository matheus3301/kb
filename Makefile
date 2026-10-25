.PHONY: tools check repo lint test build fuzz security acceptance release-check release-snapshot release benchmark benchmark-smoke retrieval-eval

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
	go build -trimpath -o .build/kb ./cmd/kb
	$(PYTHON) scripts/acceptance.py --binary .build/kb --output .build/acceptance.json
	$(PYTHON) scripts/check_docs.py --binary .build/kb

benchmark:
	go build -trimpath -o .build/kb ./cmd/kb
	$(PYTHON) scripts/benchmark.py --binary .build/kb --output .build/performance.json

benchmark-smoke:
	go build -trimpath -o .build/kb ./cmd/kb
	$(PYTHON) scripts/benchmark.py --binary .build/kb --sizes 100 --samples 2 --repair-samples 1 --output .build/performance-smoke.json

retrieval-eval:
	go build -trimpath -o .build/kb ./cmd/kb
	$(PYTHON) scripts/retrieval_eval.py --binary .build/kb --output .build/retrieval-quality.json

check: repo lint test build fuzz security acceptance benchmark-smoke retrieval-eval

release-check:
	$(GORELEASER) check

release-snapshot: release-check
	$(GORELEASER) release --snapshot --skip=publish --clean
	$(PYTHON) scripts/release.py formula
	$(PYTHON) scripts/check_release.py

release:
	@echo "Push a new version tag from tested main; see docs/releases.md."
	@exit 1
