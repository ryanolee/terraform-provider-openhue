BINARY_NAME=terraform-provider-openhue

build:
	mkdir -p dist
	go build -o dist/$(BINARY_NAME) 

watch:
	air --build.cmd="go build -o dist/$(BINARY_NAME) main.go" --build.bin "echo 'build complete'"

setup-local-dev:
	
	mv ~/.terraformrc ~/.terraformrc.bak.$(shell date +%s) || true
	sed examples/.terraformrc.tpl -e 's|$$PWD|$(shell pwd)|g' > ~/.terraformrc