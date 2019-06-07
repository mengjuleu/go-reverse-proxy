CMD=github.com/go-reverse-proxy

all: test test-slow

test:
	go test -race -v ./...

test-slow:
	go test -tags=slow -race -v ./...

lint: .gotlint
	gometalinter --fast \
	--enable gofmt \
	--disable gotype \
	--disable gocyclo \
	--exclude="file permissions" --exclude="Errors unhandled" \
	./...

setup: .gotlint

install:
	go install $(CMD)

.gotlint:
	go get github.com/alecthomas/gometalinter
	gometalinter --install
	touch $@