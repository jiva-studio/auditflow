module assessment/modules/apps/streamer

go 1.24

require (
	assessment/modules/libs/domain v0.0.0
	assessment/modules/libs/protocol v0.0.0
	assessment/modules/libs/telemetry v0.0.0
	google.golang.org/protobuf v1.36.5
)

replace (
	assessment/modules/libs/domain => ../../libs/domain
	assessment/modules/libs/protocol => ../../libs/protocol
	assessment/modules/libs/telemetry => ../../libs/telemetry
)
