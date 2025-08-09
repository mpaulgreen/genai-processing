## Build and start the server
```
go build -o server ./cmd/server
go test -count=1 ./...
set -a; source .env; set +a; ./server 
```

## Perform health check
```
curl http://localhost:8080/health
```

## TODO
- Log level and log format support LOG_LEVEL=info LOG_FORMAT=json
- Step 8.4: Generic Model Support