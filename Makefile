.PHONY: build run tidy vet clean

build:
	CGO_ENABLED=0 go build -o build/art-of-balance-nex .

run: build
	./build/art-of-balance-nex

tidy:
	go mod tidy

vet:
	go vet ./...

clean:
	rm -rf build log
