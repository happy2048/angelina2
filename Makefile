#!/usr/bin/make
.PHONY: angelina
all:
	@test -d bin || mkdir bin
	@go build angelina-controller.go
	@go build angelina-runner.go
	@go build angelina.go
	@mv angelina angelina-controller angelina-runner bin
