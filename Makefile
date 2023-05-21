
all: bin/web_server

bin/web_server: $(wildcard cmd/web_server/*.go)
	go build -o bin/web_server ./cmd/web_server

clean:
  rm -rf ./bin/*

assets:
	./node_modules/.bin/esbuild ./javascript/main.ts --bundle --sourcemap
