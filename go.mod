module go.einride.tech/lcm

go 1.16

require (
	github.com/pierrec/lz4/v4 v4.1.17
	golang.org/x/net v0.7.0
	golang.org/x/sync v0.1.0
	google.golang.org/protobuf v1.28.1
	gotest.tools/v3 v3.4.0
)

// Version has been removed from GitHub
retract (
	v1.20.0
	v1.19.0
	v1.0.0
)
