.PHONY: yts yts-all yts-failing
yts: yaml-test-suite/testdata/data-2022-01-17/229Q
	go test ./yaml-test-suite/ -count=1
yts-all: yaml-test-suite/testdata/data-2022-01-17/229Q
	RUNALL=1 go test ./yaml-test-suite -count=1 -v | awk '/     --- (PASS|FAIL): / {print $$2}' | sort | uniq -c
yts-failing: yaml-test-suite/testdata/data-2022-01-17/229Q
	RUNFAILING=1 go test ./yaml-test-suite -count=1 -v | awk '/     --- (PASS|FAIL): / {print $$2}' | sort | uniq -c
yaml-test-suite/testdata/data-2022-01-17/229Q:
	git submodule update --init --recursive yaml-test-suite/testdata/data-2022-01-17
