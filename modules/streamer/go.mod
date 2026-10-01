module assessment/modules/streamer

go 1.24

require (
	assessment/libs/domain v0.0.0
	assessment/libs/protocol v0.0.0
	google.golang.org/protobuf v1.36.5
)

replace (
	assessment/libs/domain => ../../libs/domain
	assessment/libs/protocol => ../../libs/protocol
)
