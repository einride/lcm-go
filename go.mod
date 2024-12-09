module go.einride.tech/lcm

go 1.21

toolchain go1.23.2

require (
	github.com/pierrec/lz4/v4 v4.1.21
	golang.org/x/net v0.32.0
	golang.org/x/sync v0.10.0
	google.golang.org/protobuf v1.35.2
	gotest.tools/v3 v3.5.1
)

require (
	github.com/google/go-cmp v0.6.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
)

// Version has been removed from GitHub
retract (
	v1.20.0
	v1.19.0
	v1.18.0
	v1.17.0
	v1.16.0
	v1.15.0
	v1.14.0
	v1.13.0
	v1.12.0
	v1.11.0
	v1.10.0
	v1.9.1
	v1.9.0
	v1.8.0
	v1.7.0
	v1.6.0
	v1.5.0
	v1.4.0
	v1.3.0
	v1.2.0
	v1.1.0
	v1.0.1
	v1.0.0
)
