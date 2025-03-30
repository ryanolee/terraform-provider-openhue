BINARY_NAME=terraform-provider-openhue
.PHONY: all build watch setup-local-dev docs

build:
	mkdir -p dist
	go build -o dist/$(BINARY_NAME) 

watch:
	air --build.cmd="go build -o dist/$(BINARY_NAME) main.go" --build.bin "echo 'build complete'"

setup-local-dev:
	mv ~/.terraformrc ~/.terraformrc.bak.$(shell date +%s) || true
	sed examples/.terraformrc.tpl -e 's|$$PWD|$(shell pwd)|g' > ~/.terraformrc

docs-install:
	export GOBIN=$PWD/bin
	export PATH=$GOBIN:$PATH
	go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs

docs: docs-install
	tfplugindocs generate --provider-name openhue

docs-validate:
	tfplugindocs validate