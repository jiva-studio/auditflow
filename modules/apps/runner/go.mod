module assessment/modules/apps/runner

go 1.24

require (
	assessment/modules/libs/telemetry v0.0.0
)

replace (
	assessment/modules/libs/telemetry => ../../libs/telemetry
)
