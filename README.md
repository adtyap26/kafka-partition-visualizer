# kafka-partition-visualizer (kafka-viz)

Interactive terminal UI for Kafka topic partition mapping

[![Go Report Card](https://goreportcard.com/badge/github.com/adtyap26/kafka-partition-visualizer)](https://goreportcard.com/report/github.com/adtyap26/kafka-partition-visualizer)



A terminal-based tool built with Go and Bubble Tea to visualize simulated Kafka partition placement for single-cluster and Multi-Region Cluster (MRC) setups.


## Features

- Visualize simulated partition replica placement across brokers.
- Supports both single Kafka cluster and MRC (Multi-Region Cluster) scenarios.
- Configure key parameters:
  - Number of Brokers (total or per DC)
  - Number of Partitions
  - Replication Factor (RF)
  - Minimum In-Sync Replicas (min.isr)
  - Number of Data Centers (for MRC)
- Simple color-coded TUI:
  - <span style="color:green;">**Leader**</span> (Green)
  - <span style="color:yellow;">**Follower**</span> (Yellow)
  - <span style="color:red;">**Observer**</span> (Red - MRC only)
- Provides basic replica placement recommendations for MRC setups.

## Prerequisites

- Go 1.18 or later (due to Bubble Tea dependencies, though generics aren't heavily used yet).
- `make` (optional, for using the Makefile build commands).

## Installation & Building

1.  **Clone the repository:**

    ```bash
    git clone https://github.com/adtyap26/kafka-partition-visualizer.git
    cd kafka-partition-visualizer
    ```

2.  **Build using Make (Recommended):**

    - Build for your current OS/Architecture:
      ```bash
      make build
      ```
    - Build for other platforms (examples):
      ```bash
      make build-linux
      make build-macos-arm64
      make build-windows
      ```
    - The executable (`kafka-viz` or `kafka-viz.exe`) will be placed in the `bin/` directory (e.g., `bin/linux_amd64/kafka-viz`).

3.  **Build directly using Go:**
    ```bash
    go build -o kafka-viz .
    ```

## Usage

Run the compiled binary:

```bash
# If built with make for current OS
./bin/kafka-viz

# Or if built directly with go build
./kafka-viz

# Or if cross-compiled
./bin/<os_arch>/kafka-viz[.exe]

```
