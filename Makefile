.PHONY: update master release setup update_master update_release build clean

setup:
	git config --global --add url."git@gitlab.com:".insteadOf "https://gitlab.com/"

clean:
	rm -rf vendor/
	go mod vendor

update:
	-GOFLAGS="" go get -u all

build:
	go build ./...
	go mod tidy

update_release:
	GOFLAGS="" go get -u gitlab.com/elixxir/primitives@release
	GOFLAGS="" go get -u gitlab.com/elixxir/crypto@release
	GOFLAGS="" go get -u gitlab.com/elixxir/comms@release

update_master:
	GOFLAGS="" go get -u gitlab.com/elixxir/primitives@master
	GOFLAGS="" go get -u gitlab.com/elixxir/crypto@master
	GOFLAGS="" go get -u gitlab.com/elixxir/comms@master

master: clean update_master update build

release: clean update_release update build
