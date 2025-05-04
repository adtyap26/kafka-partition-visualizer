package config

// Package config holds the core data structures and type definitions
// used across the application, particularly for representing Kafka
// cluster configuration and placement results.

// ClusterType defines whether the simulation is for a single cluster or MRC.
type ClusterType int

const (
	SingleCluster ClusterType = iota
	MRC
)

// ReplicaRole defines the role of a partition replica on a broker.
type ReplicaRole string

const (
	Leader   ReplicaRole = "Leader"
	Follower ReplicaRole = "Follower"
	Observer ReplicaRole = "Observer" // Used for MRC visualization
)

// ReplicaInfo stores information about a single partition replica.
type ReplicaInfo struct {
	PartitionID int
	Role        ReplicaRole
}

// BrokerInfo stores information about a single broker and its replicas.
type BrokerInfo struct {
	ID       int
	Replicas []ReplicaInfo
}

// DCInfo stores information about a Data Center and the brokers within it.
type DCInfo struct {
	ID      int
	Brokers map[int]*BrokerInfo // Map BrokerID -> BrokerInfo
}

// PlacementConfig holds all the user-defined parameters needed for calculation.
// This can be passed from the TUI to the placement logic.
type PlacementConfig struct {
	ClusterType       ClusterType
	NumPartitions     int
	ReplicationFactor int
	MinInSyncReplicas int
	NumBrokers        int // Total for single, per DC for MRC
	NumDCs            int
}
