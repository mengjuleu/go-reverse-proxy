CMD=github.com/go-reverse-proxy

all: test test-slow

test: .gotdeps
	go test -race -v ./...

test-slow: .gotdeps
	go test -tags=slow -race -v ./...

lint: .gotlint
	gometalinter --fast \
	--enable gofmt \
	--disable gotype \
	--disable gocyclo \
	--exclude="file permissions" --exclude="Errors unhandled" \
	./...

setup: .gotlint

install: .gotdeps
	go install $(CMD)

.gotlint:
	go get github.com/alecthomas/gometalinter
	gometalinter --install
	touch $@

.gotglide:
	go get github.com/Masterminds/glide
	touch $@

.gotdeps: .gotglide glide.lock
	glide install
	touch $@