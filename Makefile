
all: bin/web_server ./web_assets/main.js

bin/web_server: $(wildcard cmd/web_server/*.go)
	go build -o bin/web_server ./cmd/web_server

clean:
	rm -rf ./bin/* ./web_assets/*

./web_assets/main.js: ./javascript/main.ts
	./node_modules/.bin/esbuild ./javascript/main.ts --bundle --sourcemap --outfile=./web_assets/main.js
