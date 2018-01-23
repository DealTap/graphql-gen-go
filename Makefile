# Makefile
#

help:
	@echo "Usage: {options} make [target ...]"
	@echo
	@echo "Commands:"
	@echo "  install          Install required dependencies"
	@echo "  build            Build binary file"
	@echo "  generate-sample  Generate resolver and server code from sample schema file"
	@echo "  run-sample       Start the sample GraphQL server"
	@echo
	@echo "  help            Show available commands"
	@echo
	@echo "Examples:"
	@echo "  # Getting started"
	@echo "  make install"
	@echo "  make build"
	@echo

install:
	@ echo "Download required dependencies"
	@ glide update
	@ echo "Finished downloading required dependencies"

build:
	@ echo "Starting build of binary file"
	@ go install
	@ echo "Finished building binary file"
	
generate-sample:
	@ echo "Generating code for sample schema"
	@ go run main.go ./sample/schema.graphql --out_dir ./sample --pkg api
	@ echo "Formatting generated code"
	@ go fmt ./sample/api
	@ echo "Finished generating code for sample schema"
	
run-sample:
	@ echo "Starting sample GraphQL server on localhost:7050"
	@ go run ./sample/test-server.go
