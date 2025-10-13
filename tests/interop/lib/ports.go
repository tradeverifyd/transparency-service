package lib

import (
	"fmt"
	"net"
	"sync"
	"testing"
)

// PortAllocator manages unique port allocation for parallel tests
type PortAllocator struct {
	minPort      int
	maxPort      int
	allocatedMu  sync.Mutex
	allocated    map[int]bool
	nextPort     int
}

// GlobalPortAllocator is the default port allocator for tests
var GlobalPortAllocator = NewPortAllocator(20000, 30000)

// NewPortAllocator creates a new port allocator for the given range
func NewPortAllocator(minPort, maxPort int) *PortAllocator {
	return &PortAllocator{
		minPort:   minPort,
		maxPort:   maxPort,
		allocated: make(map[int]bool),
		nextPort:  minPort,
	}
}

// AllocatePort allocates a unique port for a test
// The port is marked as in-use and will be released when the test completes
func (pa *PortAllocator) AllocatePort(t *testing.T) int {
	t.Helper()

	pa.allocatedMu.Lock()
	defer pa.allocatedMu.Unlock()

	// Find next available port
	for i := 0; i < (pa.maxPort - pa.minPort); i++ {
		candidatePort := pa.nextPort
		pa.nextPort++
		if pa.nextPort > pa.maxPort {
			pa.nextPort = pa.minPort
		}

		// Skip if already allocated
		if pa.allocated[candidatePort] {
			continue
		}

		// Verify port is actually available by attempting to listen
		if !isPortAvailable(candidatePort) {
			continue
		}

		// Mark as allocated
		pa.allocated[candidatePort] = true

		// Register cleanup to release port when test completes
		t.Cleanup(func() {
			pa.allocatedMu.Lock()
			defer pa.allocatedMu.Unlock()
			delete(pa.allocated, candidatePort)
		})

		return candidatePort
	}

	t.Fatalf("No available ports in range %d-%d", pa.minPort, pa.maxPort)
	return 0
}

// isPortAvailable checks if a port is available by attempting to listen on it
func isPortAvailable(port int) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// GetPortRange returns the configured port range from environment variables
// or defaults to 20000-30000
func GetPortRange() (minPort, maxPort int) {
	// TODO: Read from SCITT_PORT_MIN and SCITT_PORT_MAX environment variables
	return 20000, 30000
}
