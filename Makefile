SHELL := /bin/sh

test:
	go test -v ./... -race
