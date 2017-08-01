PKGS = $(shell go list ./... | grep -v /vendor/)

vet:
	go vet $(PKGS)

fmt:
	go fmt $(PKGS)

test:
	go test -race -cover $(PKGS)

update-deps:
	godep save $(PKGS)

.PHONY: vet fmt test update-deps
