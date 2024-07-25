.PHONY: all
all: build

.PHONY: build
build:
	./scripts/go-dev-bld

.PHONY: rel
rel:
	./scripts/go-rel-bld

.PHONY: completion
completion: build
	t=$(mktemp)
	./landlord completion zsh > $t
	echo "source $t"

.PHONY: setup-rel
setup-rel:
	go install golang.org/x/tools/cmd/goimports@latest

.PHONY: setup
setup: setup-rel
	go install github.com/vektra/mockery/v2@v2.43.0
	go install github.com/a-h/templ/cmd/templ@latest

.PHONY: install
install: build
	./scripts/go-dev-bld install

.PHONY: sim-test
sim-test:
	LANDLORD_SKIP_TESTS=1 ./scripts/go-dev-bld
	cd ./test && ./sim-cleanup.sh && ./sim-test-01.sh && ./sim-cleanup.sh

.PHONY: gen
gen:
	./scripts/generate-go-server.sh \
		./api/openapi.yaml \
		apresdev \
		landlord \
		rest \
		internal/rest
	./scripts/generate-go-client.sh \
		./api/openapi.yaml \
		apresdev \
		landlord \
		client \
		pkg/client

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

.PHONY: api-spec
api-spec:
	mkdir -p ./tmp
	yq eval -o=json ./api/openapi.yaml > ./tmp/openapi.json
	./scripts/add-openapi-extensions.py ./tmp/openapi.json > ./api/openapi-ext.json
