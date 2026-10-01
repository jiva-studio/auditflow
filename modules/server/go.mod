module assessment/modules/server

go 1.24

replace (
	assessment/libs/domain => ../../libs/domain
	assessment/libs/protocol => ../../libs/protocol
)

require (
	assessment/libs/domain v0.0.0
	assessment/libs/protocol v0.0.0-00010101000000-000000000000
	google.golang.org/protobuf v1.36.12
)
