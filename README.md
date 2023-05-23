
### Building

    > npm install
    [...]
    
    > make    
    go build -o bin/smoothcriminal ./cmd/web_server
    ./node_modules/.bin/esbuild ./javascript/main.ts --bundle --sourcemap --outfile=./web_assets/main.js

      web_assets/main.js      2.7kb
      web_assets/main.js.map  4.6kb

    ⚡ Done in 10ms    

### Running tests

    go test ./pkg/*.go
    
Or if you need finer-grained control:
  
    go test -v ./pkg/hub_test.go pkg/hub.go ./pkg/locks.go ./pkg/general.go 
    go test -v ./pkg/locks_test.go pkg/hub.go ./pkg/locks.go ./pkg/general.go
    
