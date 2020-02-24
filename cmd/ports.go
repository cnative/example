package main

const (
	reportsServerGRPCPort       = 5601 // GRPC Port
	reportsServerGRPGatewayPort = 5602 // HTTP Gateway Port

	metricsPort = 9101  // prometheus /metrics endpoint
	healthPort  = 4400  // /ready and /live endpoints are exposed on this
	debugPort   = 6060  // debug port on which net/http/pprof data is exposed
	ocAgentPort = 55678 // opencensus agent port
)
