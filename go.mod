module go.einride.tech/lcm

go 1.16

require (
	github.com/pierrec/lz4/v4 v4.1.15
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect
	google.golang.org/protobuf v1.28.1
	gotest.tools/v3 v3.3.0
)

// Version has been removed from GitHub
retract (
	v1.20.0
	v1.19.0
	v1.0.0
)
