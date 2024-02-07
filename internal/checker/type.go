package checker

import "time"

type CheckType struct {
	Name   string
	Notes  string
	Status string

	// fields copied to CheckDefinition
	// Update CheckDefinition when adding fields here

	ScriptArgs             []string
	HTTP                   string
	H2PING                 string
	H2PingUseTLS           bool
	Header                 map[string][]string
	Method                 string
	Body                   string
	DisableRedirects       bool
	TCP                    string
	TCPUseTLS              bool
	UDP                    string
	Interval               time.Duration
	AliasNode              string
	AliasService           string
	DockerContainerID      string
	Shell                  string
	GRPC                   string
	GRPCUseTLS             bool
	OSService              string
	TLSServerName          string
	TLSSkipVerify          bool
	Timeout                time.Duration
	TTL                    time.Duration
	SuccessBeforePassing   int
	FailuresBeforeWarning  int
	FailuresBeforeCritical int

	// Definition fields used when exposing checks through a proxy
	ProxyHTTP string
	ProxyGRPC string

	// DeregisterCriticalServiceAfter, if >0, will cause the associated
	// service, if any, to be deregistered if this check is critical for
	// longer than this duration.
	DeregisterCriticalServiceAfter time.Duration
	OutputMaxSize                  int
}
