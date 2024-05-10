.PHONY: all
all: build

.PHONY: build
build:
	./apres-go-bld

.PHONY: completion
completion: build
	t=$(mktemp)
	./landlord completion zsh > $t
	echo "source $t"

.PHONY: setup
setup:
	go install github.com/vektra/mockery/v2@v2.43.0

.PHONY: install
install: build
	./apres-go-bld install

.PHONY: sim-test
sim-test:
	APRES_SKIP_TESTS=1 ./apres-go-bld
	cd ./test && ./sim-cleanup.sh && ./sim-test-01.sh && ./sim-cleanup.sh
