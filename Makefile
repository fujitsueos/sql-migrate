test:
	go test -race -cover ./...

update-deps:
	dep ensure -update

.PHONY: test update-deps
