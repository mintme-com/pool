#!/bin/bash

git clone --depth=1 https://go.googlesource.com/sys "$HOME/go/src/golang.org/x/sys"
git clone --depth=1 https://go.googlesource.com/crypto "$HOME/go/src/golang.org/x/crypto"

URLLIST="
gopkg.in/bsm/ratelimit.v1
gopkg.in/karalabe/cookiejar.v2
gopkg.in/redis.v3
github.com/denisbrodbeck/machineid
github.com/fatih/color
github.com/golang/snappy
github.com/gorilla/mux
github.com/mattn/go-colorable
github.com/mattn/go-isatty
github.com/mintme-com/cpuid
github.com/rcrowley/go-metrics
github.com/syndtr/goleveldb
github.com/webchain-network/cryptonight
github.com/webchain-network/webchaind
github.com/yvasiyarov/go-metrics
github.com/yvasiyarov/gorelic
github.com/yvasiyarov/newrelic_platform_go
"

for URL in $URLLIST; do
	git clone --depth=1 "https://${URL}" "$HOME/go/src/${URL}"
done
