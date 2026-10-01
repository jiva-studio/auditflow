module assessment/tests/e2e

go 1.24

replace (
	assessment/libs/domain => ../../libs/domain
	assessment/libs/protocol => ../../libs/protocol
	assessment/libs/rules => ../../libs/rules
	assessment/modules/agent => ../../modules/agent
	assessment/modules/streamer => ../../modules/streamer
)

require (
	assessment/libs/protocol v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.12
)
