module github.com/DaDevFox/task-systems/workflows/backend

go 1.24.2

require (
	github.com/DaDevFox/task-systems/inventory-core/backend v0.0.0-00010101000000-000000000000
	github.com/DaDevFox/task-systems/shared v0.0.0
	github.com/DaDevFox/task-systems/shared/events v0.0.0-00010101000000-000000000000
	github.com/DaDevFox/task-systems/tasker-core v0.0.0-00010101000000-000000000000
	github.com/google/go-cmp v0.7.0
	github.com/improbable-eng/grpc-web v0.15.0
	github.com/sirupsen/logrus v1.9.3
	google.golang.org/grpc v1.75.0
	google.golang.org/protobuf v1.36.8
)

require (
	github.com/cenkalti/backoff/v4 v4.1.1 // indirect
	github.com/desertbit/timer v0.0.0-20180107155436-c41aec40b27f // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/rs/cors v1.7.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250728155136-f173205681a0 // indirect
	nhooyr.io/websocket v1.8.6 // indirect
)

// Local module replacements for development
replace github.com/DaDevFox/task-systems/workflows/backend => ./

replace github.com/DaDevFox/task-systems/workflows/backend/pkg => ./pkg

replace github.com/DaDevFox/task-systems/inventory-core/backend => ../../inventory-core/backend

replace github.com/DaDevFox/task-systems/shared => ../../shared

replace github.com/DaDevFox/task-systems/shared/events => ../../shared/events

replace github.com/DaDevFox/task-systems/tasker-core => ../../tasker-core
