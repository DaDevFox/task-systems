module github.com/DaDevFox/task-systems/user-core

go 1.25.11

replace github.com/DaDevFox/task-systems/shared => ../shared

replace github.com/DaDevFox/task-systems/tasker-core => ../tasker-core

replace github.com/DaDevFox/task-systems/user-core => ../user-core

require (
	github.com/DaDevFox/hof v0.0.0-20260713062638-a32fdf93da66
	github.com/DaDevFox/task-systems/tasker-core v0.0.0-00010101000000-000000000000
	github.com/dgraph-io/badger/v3 v3.2103.5
	github.com/gogo/protobuf v1.3.2
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/ka-weihe/fast-levenshtein v0.0.0-20201227151214-4c99ee36a1ba
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.10.2
	github.com/stretchr/testify v1.12.1
	golang.org/x/crypto v0.55.0
	google.golang.org/grpc v1.83.2
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/DaDevFox/task-systems/shared v0.0.0-00010101000000-000000000000 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgraph-io/ristretto v0.2.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/flatbuffers v25.12.19+incompatible // indirect
	github.com/klauspost/compress v1.19.0 // indirect
	github.com/suryanshu-09/hof v1.0.1 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260819154853-08b0e4226688 // indirect
)
