module kythe.io

require (
	bitbucket.org/creachadair/shell v0.0.6
	bitbucket.org/creachadair/stringset v0.0.9
	cloud.google.com/go v0.90.0 // indirect
	cloud.google.com/go/storage v1.16.0 // indirect
	github.com/DataDog/zstd v1.4.8
	github.com/apache/beam v2.31.0+incompatible
	github.com/bazelbuild/rules_go v0.28.0
	github.com/beevik/etree v1.1.0
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2
	github.com/golang/snappy v0.0.4
	github.com/google/brotli/go/cbrotli v0.0.0-20210804124202-19d86fb9a60a
	github.com/google/go-cmp v0.5.6
	github.com/google/orderedcode v0.0.1
	github.com/google/subcommands v1.2.0
	github.com/google/uuid v1.3.0 // indirect
	github.com/hanwen/go-fuse v1.0.0
	github.com/jmhodges/levigo v1.0.0
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/minio/highwayhash v1.0.2
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/onsi/gomega v1.5.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/sergi/go-diff v1.2.0
	github.com/sourcegraph/go-langserver v2.0.0+incompatible
	github.com/sourcegraph/jsonrpc2 v0.1.0
	github.com/syndtr/goleveldb v1.0.0
	golang.org/x/net v0.0.0-20210726213435-c6fcb2dbf985
	golang.org/x/oauth2 v0.0.0-20210628180205-a41e5a781914
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20211019181941-9d821ace8654
	golang.org/x/text v0.3.7
	golang.org/x/tools v0.1.5
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	google.golang.org/api v0.52.0
	google.golang.org/genproto v0.0.0-20210803142424-70bd63adacf2 // indirect
	google.golang.org/grpc v1.39.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/yaml.v2 v2.4.0 // indirect
	sigs.k8s.io/yaml v1.2.0
)

replace (
	bitbucket.org/creachadair/shell v0.0.6 => ../go-creachadair-shell
	bitbucket.org/creachadair/stringset v0.0.9 => ../go-creachadair-stringset
	github.com/beevik/etree => ../go-etree
	github.com/google/go-cmp => ../go-cmp
	github.com/google/subcommands => ../go-subcommands
	golang.org/x/sync => ../golang-x-sync
	golang.org/x/sys => ../golang-x-sys
	golang.org/x/tools => ../golang-x-tools
)

go 1.18
