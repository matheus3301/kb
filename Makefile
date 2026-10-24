.PHONY: tools check repo lint test build fuzz security acceptance release

export PATH := $(CURDIR)/.tools/bin:$(PATH)
PYTHON ?= python3

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

check: repo lint test build fuzz security acceptance

release:
	$(PYTHON) scripts/release.py --version $(VERSION)
