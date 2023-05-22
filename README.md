
### Building

    npm install
    make

### Running tests

    go test ./pkg/*.go
    
Or if you need finer-grained control:
  
    go test -v ./pkg/hub_test.go pkg/hub.go ./pkg/locks.go ./pkg/general.go 
    go test -v ./pkg/locks_test.go pkg/hub.go ./pkg/locks.go ./pkg/general.go
    
