module assessment/tests/e2e

go 1.24

replace (
	assessment/libs/domain => ../../libs/domain
	assessment/libs/protocol => ../../libs/protocol
	assessment/modules/streamer => ../../modules/streamer
)
