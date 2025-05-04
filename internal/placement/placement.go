package placement

import (
	"fmt"
	"math/rand"
	"time"

	// Use the full module path for internal packages
	"github.com/adtyap26/kafka-partition-visualizer/internal/config"
)

// Package placement contains the logic for simulating Kafka partition placement
// based on the provided configuration.

// CalculatePlacement simulates partition placement based on the input config.
// It returns a map representing the DCs and brokers with their assigned replicas,
// and a string containing MRC placement recommendations (if applicable).
// This is a simplified simulation focusing on distribution.
func CalculatePlacement(cfg config.PlacementConfig) (map[int]*config.DCInfo, string) {
	// Seed random locally if not already done globally (good practice per package)
	// Note: If main already seeds, this might be redundant but harmless.
	// Consider a central seeding strategy if randomness needs strict control.
	rand.Seed(time.Now().UnixNano())

	dcs := make(map[int]*config.DCInfo)
	brokerIDCounter := 0
	totalBrokers := 0
	mrcRecommendation := ""

	// Initialize DCs and Brokers
	numDCs := cfg.NumDCs
	brokersPerDC := cfg.NumBrokers
	if cfg.ClusterType == config.SingleCluster {
		numDCs = 1 // Force 1 DC for single cluster type
		// brokersPerDC remains cfg.NumBrokers (total brokers)
	}

	for dcIdx := 0; dcIdx < numDCs; dcIdx++ {
		dcID := dcIdx + 1 // 1-based DC IDs
		dcs[dcID] = &config.DCInfo{
			ID:      dcID,
			Brokers: make(map[int]*config.BrokerInfo),
		}
		// For single cluster, brokersPerDC is the total number of brokers
		numBrokersInThisDC := brokersPerDC
		if cfg.ClusterType == config.SingleCluster {
			numBrokersInThisDC = cfg.NumBrokers // Use the total broker count directly
		}

		for brokerIdx := 0; brokerIdx < numBrokersInThisDC; brokerIdx++ {
			// Ensure we don't exceed total brokers if it's a single cluster loop
			if cfg.ClusterType == config.SingleCluster && brokerIDCounter >= cfg.NumBrokers {
				break
			}
			brokerID := brokerIDCounter
			dcs[dcID].Brokers[brokerID] = &config.BrokerInfo{
				ID:       brokerID,
				Replicas: []config.ReplicaInfo{},
			}
			brokerIDCounter++
		}
	}
	totalBrokers = brokerIDCounter

	// --- MRC Recommendation ---
	if cfg.ClusterType == config.MRC {
		mrcRecommendation = fmt.Sprintf("Distribute %d replicas across %d DCs for fault tolerance.", cfg.ReplicationFactor, cfg.NumDCs)
		if cfg.ReplicationFactor <= cfg.NumDCs {
			mrcRecommendation += " Aim for at most one replica per DC per partition."
		} else {
			minPerDC := cfg.ReplicationFactor / cfg.NumDCs
			extra := cfg.ReplicationFactor % cfg.NumDCs
			mrcRecommendation += fmt.Sprintf(" Aim for ~%d replicas per DC, with %d DCs having an extra replica.", minPerDC, extra)
		}
	}

	// --- Placement Logic ---
	allBrokerIDs := make([]int, 0, totalBrokers)
	for dcID := 1; dcID <= numDCs; dcID++ {
		// Check if DC exists (important for single cluster case where numDCs=1)
		if dcInfo, ok := dcs[dcID]; ok {
			for brokerID := range dcInfo.Brokers {
				allBrokerIDs = append(allBrokerIDs, brokerID)
			}
		}
	}
	// Ensure allBrokerIDs isn't empty if totalBrokers > 0
	if totalBrokers > 0 && len(allBrokerIDs) == 0 {
		// This indicates an issue with DC/Broker initialization logic
		// For now, return empty results to avoid panic, but log potentially
		fmt.Println("Warning: No broker IDs collected for placement.")
		return dcs, mrcRecommendation
	}
	if totalBrokers == 0 {
		// No brokers to place on
		return dcs, mrcRecommendation
	}

	for p := 0; p < cfg.NumPartitions; p++ {
		partitionID := p + 1 // 1-based partition IDs

		// Shuffle brokers for each partition for better distribution simulation
		shuffledBrokerIDs := make([]int, len(allBrokerIDs))
		copy(shuffledBrokerIDs, allBrokerIDs)
		rand.Shuffle(len(shuffledBrokerIDs), func(i, j int) {
			shuffledBrokerIDs[i], shuffledBrokerIDs[j] = shuffledBrokerIDs[j], shuffledBrokerIDs[i]
		})

		// Determine leader broker (simple modulo for initial placement)
		leaderBrokerID := allBrokerIDs[p%totalBrokers] // Start leader assignment round-robin

		// Find the DC and Broker object for the leader
		leaderDC, leaderBroker := findBroker(leaderBrokerID, dcs)
		if leaderBroker == nil {
			fmt.Printf("Warning: Could not find leader broker %d for partition %d\n", leaderBrokerID, partitionID)
			continue // Skip this partition if leader assignment fails
		}

		// Assign Leader
		leaderBroker.Replicas = append(leaderBroker.Replicas, config.ReplicaInfo{PartitionID: partitionID, Role: config.Leader})
		assignedBrokerIDs := map[int]bool{leaderBrokerID: true}
		assignedDCs := map[int]bool{leaderDC.ID: true}
		replicasPlaced := 1

		brokersToTry := shuffledBrokerIDs // Use shuffled list

		// Variables only needed for MRC role differentiation
		var numFollowers, numObservers, targetFollowers, targetObservers int
		if cfg.ClusterType == config.MRC {
			targetFollowers = cfg.MinInSyncReplicas - 1 // Followers needed for ISR quorum
			if targetFollowers < 0 {
				targetFollowers = 0
			}
			targetObservers = cfg.ReplicationFactor - 1 - targetFollowers // Remaining replicas
			if targetObservers < 0 {
				targetObservers = 0
			}
		}

		// First pass (try spreading across DCs for MRC)
		for _, brokerID := range brokersToTry {
			if replicasPlaced >= cfg.ReplicationFactor {
				break
			} // Stop if RF met
			if assignedBrokerIDs[brokerID] {
				continue
			} // Skip if broker already has a replica for this partition

			dc, broker := findBroker(brokerID, dcs)
			if broker == nil {
				continue // Should not happen if brokerID is from allBrokerIDs
			}

			// MRC Placement Strategy: Try to place in different DCs first
			placeInThisDC := true
			if cfg.ClusterType == config.MRC && len(assignedDCs) < cfg.NumDCs {
				if assignedDCs[dc.ID] {
					// Check if we can place elsewhere before placing in an already used DC
					canPlaceElsewhere := false
					for _, otherBrokerID := range brokersToTry {
						if !assignedBrokerIDs[otherBrokerID] {
							otherDC, _ := findBroker(otherBrokerID, dcs)
							if otherDC != nil && !assignedDCs[otherDC.ID] { // Check otherDC is not nil
								canPlaceElsewhere = true
								break
							}
						}
					}
					if canPlaceElsewhere {
						placeInThisDC = false
					}
				}
			}

			if placeInThisDC {
				var role config.ReplicaRole
				if cfg.ClusterType == config.SingleCluster {
					// In Single Cluster, all non-leaders are just Followers
					role = config.Follower
				} else { // MRC logic
					// Assign role based on ISR needs first, then observers
					if numFollowers < targetFollowers {
						role = config.Follower
						numFollowers++
					} else if numObservers < targetObservers {
						role = config.Observer
						numObservers++
					} else {
						// Fallback if RF > minISR + observers needed
						role = config.Observer
						numObservers++
					}
				}

				broker.Replicas = append(broker.Replicas, config.ReplicaInfo{PartitionID: partitionID, Role: role})
				assignedBrokerIDs[brokerID] = true
				assignedDCs[dc.ID] = true // Track used DCs for MRC strategy
				replicasPlaced++
			}
		}

		// Second pass for MRC if needed (allow placing in same DC)
		if cfg.ClusterType == config.MRC && replicasPlaced < cfg.ReplicationFactor {
			for _, brokerID := range brokersToTry {
				if replicasPlaced >= cfg.ReplicationFactor {
					break
				}
				if assignedBrokerIDs[brokerID] {
					continue
				}

				_, broker := findBroker(brokerID, dcs)
				if broker == nil {
					continue
				}

				// Assign role based on remaining needs for MRC
				var role config.ReplicaRole
				if numFollowers < targetFollowers {
					role = config.Follower
					numFollowers++
				} else if numObservers < targetObservers {
					role = config.Observer
					numObservers++
				} else {
					role = config.Observer // Assign remaining as Observers
					numObservers++
				}

				broker.Replicas = append(broker.Replicas, config.ReplicaInfo{PartitionID: partitionID, Role: role})
				assignedBrokerIDs[brokerID] = true
				// assignedDCs doesn't need update here
				replicasPlaced++
			}
		}
	}

	return dcs, mrcRecommendation
}

// findBroker searches all DCs to find the broker with the given ID.
// Kept unexported as it's internal to the placement logic.
func findBroker(brokerID int, dcs map[int]*config.DCInfo) (*config.DCInfo, *config.BrokerInfo) {
	for _, dc := range dcs {
		if broker, ok := dc.Brokers[brokerID]; ok {
			return dc, broker
		}
	}
	return nil, nil // Not found
}
