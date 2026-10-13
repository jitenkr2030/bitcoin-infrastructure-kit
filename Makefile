.PHONY: test build run clean
test:
	go test ./...
	go vet ./...
build:
	./scripts/build.sh
run:
	go run . help
clean:
	rm -rf bin
