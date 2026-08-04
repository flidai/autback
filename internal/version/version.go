// Package version owns the Autback release version shared by every binary.
package version

const (
	Current                       = "0.1.11"
	ClientVersionHeader           = "Autback-Client-Version"
	ClientCapabilitiesHeader      = "Autback-Client-Capabilities"
	CapabilityBuildLeaseHeartbeat = "build-lease-heartbeat"
)
