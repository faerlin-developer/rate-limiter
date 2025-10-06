
.PHONY: demo test

demo:
	go run demo/main.go

test:
	go clean -testcache && go test -v ./...
