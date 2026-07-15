VERSION ?= 0.1.2
GOCACHE ?= /tmp/ezc-go-cache
DIST := $(CURDIR)/dist
LDFLAGS := -s -w -X github.com/mornhussakuyo-hub/ezc/internal/app.Version=$(VERSION)

.PHONY: all test cross-check build package arch-package clean

all: build

test:
	GOCACHE=$(GOCACHE) go test ./...

cross-check:
	GOCACHE=$(GOCACHE) GOOS=windows GOARCH=amd64 go test -exec=/bin/true ./...

build:
	mkdir -p $(DIST)
	GOCACHE=$(GOCACHE) GOOS=linux GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/ezc-linux-amd64 ./cmd/ezc
	GOCACHE=$(GOCACHE) GOOS=windows GOARCH=amd64 go build -buildvcs=false -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/ezc-windows-amd64.exe ./cmd/ezc

package: build arch-package
	mkdir -p $(DIST)/linux-package $(DIST)/windows-package
	cp $(DIST)/ezc-linux-amd64 $(DIST)/linux-package/ezc
	cp README.md DESIGN.md LICENSE $(DIST)/linux-package/
	cp $(DIST)/ezc-windows-amd64.exe $(DIST)/windows-package/ezc.exe
	cp README.md DESIGN.md LICENSE $(DIST)/windows-package/
	tar -czf $(DIST)/ezc_$(VERSION)_linux_amd64.tar.gz -C $(DIST)/linux-package ezc README.md DESIGN.md LICENSE
	bsdtar -a -cf $(DIST)/ezc_$(VERSION)_windows_amd64.zip -C $(DIST)/windows-package ezc.exe README.md DESIGN.md LICENSE
	cd $(DIST) && sha256sum ezc-linux-amd64 ezc-windows-amd64.exe ezc_$(VERSION)_linux_amd64.tar.gz ezc_$(VERSION)_windows_amd64.zip ezc-$(VERSION)-1-x86_64.pkg.tar.zst > SHA256SUMS

arch-package: build
	cp $(DIST)/ezc-linux-amd64 packaging/arch/ezc
	cp README.md DESIGN.md LICENSE packaging/arch/
	cd packaging/arch && PKGDEST=$(DIST) makepkg --force --nodeps --clean

clean:
	rm -rf $(DIST) packaging/arch/ezc packaging/arch/README.md packaging/arch/DESIGN.md packaging/arch/LICENSE packaging/arch/pkg packaging/arch/src
