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
