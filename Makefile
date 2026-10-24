.PHONY: tools check repo lint test build fuzz security release

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

check: repo lint test build fuzz security

release:
	$(PYTHON) scripts/release.py --version $(VERSION)
