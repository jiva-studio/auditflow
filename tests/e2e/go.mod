module assessment/tests/e2e

go 1.24

replace (
	assessment/modules/libs/domain => ../../modules/libs/domain
	assessment/modules/libs/protocol => ../../modules/libs/protocol
	assessment/modules/libs/rules => ../../modules/libs/rules
	assessment/modules/apps/agent => ../../modules/apps/agent
	assessment/modules/apps/streamer => ../../modules/apps/streamer
)

require (
	assessment/modules/libs/protocol v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.12
)
