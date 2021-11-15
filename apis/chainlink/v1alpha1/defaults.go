package v1alpha1

const (
	// DefaultTLSPort is the default tls port
	DefaultTLSPort uint = 6689
)

var (
	// DefaultCorsDomains is the default cors domains from which to accept requests
	DefaultCorsDomains = []string{"*"}
)

// Resources
const (
	// DefaultNodeCPURequest is the cpu requested by chainlink node
	DefaultNodeCPURequest = "2"
	// DefaultNodeCPULimit is the cpu limit for chainlink node
	DefaultNodeCPULimit = "4"

	// DefaultNodeMemoryRequest is the memory requested by chainlink node
	DefaultNodeMemoryRequest = "2Gi"
	// DefaultNodeMemoryLimit is the memory limit for chainlink node
	DefaultNodeMemoryLimit = "4Gi"

	// DefaultNodeStorageRequest is the Storage requested by chainlink node
	DefaultNodeStorageRequest = "20Gi"
)