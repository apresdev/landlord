.PHONY: all
all: build

.PHONY: build
build: gen
	./scripts/go-dev-bld

.PHONY: clean
clean:
	go clean
	go clean -testcache
	rm -f landlord.linux.amd64

.PHONY: rel
rel: gen
	./scripts/go-rel-bld

.PHONY: itest
itest:
	./scripts/go-itest

.PHONY: version
version:
	echo "package appenv" > ./internal/appenv/tag.go
	echo "" >> ./internal/appenv/tag.go
	echo "const TagVersion = \"$(shell git describe --tags --abbrev=0)\"" >> ./internal/appenv/tag.go

.PHONY: completion
completion: build
	t=$(mktemp)
	./landlord completion zsh > $t
	echo "source $t"

.PHONY: setup
setup:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/vektra/mockery/v2@v2.53.0
	go install github.com/a-h/templ/cmd/templ@v0.3.819
	go install golang.org/x/lint/golint@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest

.PHONY: install
install: build
	./scripts/go-dev-bld install

.PHONY: sim-test
sim-test:
	LANDLORD_SKIP_TESTS=1 ./scripts/go-dev-bld
	cd ./test && ./sim-cleanup.sh && ./sim-test-01.sh && ./sim-cleanup.sh

.PHONY: help
help:
	pandoc ./HELP.md --output tmp/help.snippet.html
	echo "{{ define \"help\" }}" > tmp/help.snippet.gohtml
	cat tmp/help.snippet.html >> tmp/help.snippet.gohtml
	echo "{{ end }}" >> tmp/help.snippet.gohtml
	cp -v tmp/help.snippet.gohtml ./internal/webapp/admin-templates/help.gohtml

.PHONY: gen
gen: ./api/landlord-openapi-ext.json

./api/landlord-openapi-ext.json: ./api/landlord-openapi.yaml ./scripts/add-openapi-extensions.py
	./scripts/generate-go-server.sh \
		./api/landlord-openapi.yaml \
		apresdev \
		landlord \
		rest \
		internal/rest
	./scripts/generate-go-client.sh \
		./api/landlord-openapi.yaml \
		apresdev \
		landlord \
		client \
		pkg/client
	mkdir -p ./tmp
	yq eval -o=json ./api/landlord-openapi.yaml > ./tmp/landlord-openapi.json
	./scripts/add-openapi-extensions.py ./tmp/landlord-openapi.json > ./api/landlord-openapi-ext.json

.PHONY: docker-pull
docker-pull:	## - docker pull latest images
	@printf "\033[32m\xE2\x9c\x93 docker pull latest images\n\033[0m"
	@docker pull golang:alpine

.PHONY: docker
docker: docker-pull	## - Build the smallest and secured golang docker image based on scratch
	@printf "\033[32m\xE2\x9c\x93 Build the smallest and secured golang docker image based on scratch\n\033[0m"
	$(eval BUILDER_IMAGE=$(shell docker inspect --format='{{index .RepoDigests 0}}' golang:alpine))
	@export DOCKER_CONTENT_TRUST=1
	@docker build -f Dockerfile --build-arg "BUILDER_IMAGE=$(BUILDER_IMAGE)" -t landlord .

.PHONY: get-htmx
get-htmx:
	curl -o ./internal/server/static/htmx.min.js https://unpkg.com/htmx.org@2.0.2/dist/htmx.min.js
	# We download the unminified version for development purposes, but exclude it from repo.
	curl -o ./internal/server/static/htmx.js https://unpkg.com/htmx.org@2.0.2/dist/htmx.js
