RUNBIN   := ../../bin/diecast-$(shell go env GOOS)-$(shell go env GOARCH)
OD        = od --width=32768 -a --format 'a' - | head -n1 | tr -d ' ' | tail -c +8
SUM       = $(OD) | sha256sum - | cut -d' ' -f1
LOGLEVEL ?= notice

.PHONY: $(RUNBIN) $(wildcard *.txt)
.EXPORT_ALL_VARIABLES:

all: $(wildcard *.txt)

test-render-kv.txt:
	@ -rm -f $(@:%.txt=%.out)
	@$(RUNBIN) render \
		-d testkv.id=hello \
		-d testkv.items.0.id=1 \
		-d testkv.items.1.id=3 \
		-d testkv.items.2.id=5 \
		$(@) > $(@:%.txt=%.out)
	@test "$$(cat $(@:%.txt=%.out) | $(OD))" = "$$(echo -n 'name=hellonl-sp1nl-sp3nl-sp5nl' | $(OD))"
	@echo "[RENDER] $(@): ok"

test-render-file.txt:
	@ -rm -f $(@:%.txt=%.out)
	@$(RUNBIN) render \
		-D data/other.json \
		-D data/test.json \
		$(@) > $(@:%.txt=%.out)
	@test "$$(cat $(@:%.txt=%.out) | $(SUM))" = "$$(echo -n 'type=othernl-sp1nl-sp3nl-sp5nldebug=true' | $(SUM))"
	@echo "[RENDER] $(@): ok"

test-render-dir.txt:
	@ -rm -f $(@:%.txt=%.out)
	@$(RUNBIN) render \
		-D data \
		$(@) > $(@:%.txt=%.out)
	@test "$$(cat $(@:%.txt=%.out) | $(SUM))" = "$$(echo -n 'type=othernl-sp1nl-sp3nl-sp5nldebug=true' | $(SUM))"
	@echo "[RENDER] $(@): ok"