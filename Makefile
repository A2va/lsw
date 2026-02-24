.PHONY: build run test

build:
	go build -tags "containers_image_openpgp exclude_graphdriver_btrfs exclude_graphdriver_devicemapper" .

run:
	go run -tags "containers_image_openpgp exclude_graphdriver_btrfs exclude_graphdriver_devicemapper" .

test:
	go test -v -tags "containers_image_openpgp exclude_graphdriver_btrfs exclude_graphdriver_devicemapper" ./pkg/cache
