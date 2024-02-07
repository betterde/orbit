package checker

import "time"

const (
	// HealthAny is special, and is used as a wild card,
	// not as a specific state.
	HealthAny      = "any"
	HealthPassing  = "passing"
	HealthWarning  = "warning"
	HealthCritical = "critical"
	HealthMaint    = "maintenance"
)

const (
	serviceHealth = "service"
	connectHealth = "connect"
	ingressHealth = "ingress"
)

// HealthCheck is used to represent a single check
type HealthCheck struct {
	Node        string
	CheckID     string
	Name        string
	Notes       string
	Status      string
	Output      string
	ServiceID   string
	ServiceName string
	ServiceTags []string
	Type        string
	Namespace   string `json:",omitempty"`
	Partition   string `json:",omitempty"`
	ExposedPort int
	PeerName    string `json:",omitempty"`

	Definition HealthCheckDefinition

	CreateIndex uint64
	ModifyIndex uint64
}

// HealthCheckDefinition is used to store the details about a health check's execution.
type HealthCheckDefinition struct {
	HTTP                                   string
	Header                                 map[string][]string
	Method                                 string
	Body                                   string
	TLSServerName                          string
	TLSSkipVerify                          bool
	TCP                                    string
	TCPUseTLS                              bool
	UDP                                    string
	GRPC                                   string
	OSService                              string
	GRPCUseTLS                             bool
	TimeoutDuration                        time.Duration `json:"-"`
	IntervalDuration                       time.Duration `json:"-"`
	DeregisterCriticalServiceAfterDuration time.Duration `json:"-"`
}
