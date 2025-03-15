
.DEFAULT_GOAL:=dev
.PHONY: dev test

name?=local

log?=0

run:
	WS_LOG=$(log) go \
		run ./examples/${name}

dev:
	WS_LOG=$(log) gow -c -r \
		-w . \
		-w ./examples/${name} \
		run ./examples/${name}

test:
	go test ./...
