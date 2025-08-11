## Build and start the server
```
go build -o server ./cmd/server
go test -count=1 ./...
set -a; source .env; set +a; ./server 
lsof -ti:8080 | xargs kill -9
cd Users/mrigankapaul/Documents/knowledgebase/audit-query/nlp-processing/genai-processing && ./run_server.sh &
curl -sS -X POST http://localhost:8080/query -H "Content-Type: application/json" -d '{"query":"Who deleted the customer CRD yesterday?","session_id":"test"}' | jq .
```

## Perform health check
```
curl http://localhost:8080/health
```

## TODO
- Log level and log format support LOG_LEVEL=info LOG_FORMAT=json
- Step 8.4: Generic Model Support