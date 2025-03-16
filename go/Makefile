
.DEFAULT_GOAL:=dev
.PHONY: dev test

name?=server

log?=1
wslog?=0

run:
	WS_LOG=$(log) go \
		run ./examples/${name}

dev:
	LOG=$(log) WS_LOG=$(wslog) LOG_FILE=$(logile) gow -c -r \
		-w . \
		-w ./examples/${name} \
		run ./examples/${name}

test:
	go test ./...
