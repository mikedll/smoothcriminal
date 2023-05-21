
all: bin/web_server ./javascript/esbuild/main.js

bin/web_server: $(wildcard cmd/web_server/*.go)
	go build -o bin/web_server ./cmd/web_server

clean:
	rm -rf ./bin/* ./javascript/esbuild/*

./javascript/esbuild/main.js: ./javascript/main.ts
	./node_modules/.bin/esbuild ./javascript/main.ts --bundle --sourcemap --outfile=./javascript/esbuild/main.js
