module assessment/modules/apps/server

go 1.24

replace (
	assessment/modules/libs/domain => ../../libs/domain
	assessment/modules/libs/protocol => ../../libs/protocol
)

require (
	assessment/modules/libs/domain v0.0.0
	assessment/modules/libs/protocol v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.12
)
