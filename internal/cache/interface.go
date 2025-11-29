package cache

import (
	"github.com/yejune/go-react-ssr/internal/reactbuilder"
)

// Cache defines the interface for build caching
type Cache interface {
	// GetServerBuild retrieves a server build from cache
	GetServerBuild(filePath string) (reactbuilder.BuildResult, bool, error)
	// SetServerBuild stores a server build in cache
	SetServerBuild(filePath string, build reactbuilder.BuildResult) error
	// RemoveServerBuild removes a server build from cache
	RemoveServerBuild(filePath string) error

	// GetClientBuild retrieves a client build from cache
	GetClientBuild(filePath string) (reactbuilder.BuildResult, bool, error)
	// SetClientBuild stores a client build in cache
	SetClientBuild(filePath string, build reactbuilder.BuildResult) error
	// RemoveClientBuild removes a client build from cache
	RemoveClientBuild(filePath string) error

	// Route mapping
	SetParentFile(routeID, filePath string) error
	GetRouteIDSForParentFile(filePath string) ([]string, error)
	GetAllRouteIDS() ([]string, error)
	GetRouteIDSWithFile(filePath string) ([]string, error)

	// Dependencies
	SetParentFileDependencies(filePath string, dependencies []string) error
	GetParentFilesFromDependency(dependencyPath string) ([]string, error)

	// Clear removes all cached data
	Clear() error
}

// CacheType represents the type of cache to use
type CacheType string

const (
	CacheTypeLocal CacheType = "local" // In-memory cache (default)
	CacheTypeRedis CacheType = "redis" // Redis distributed cache
)

// CacheConfig configures the cache
type CacheConfig struct {
	Type CacheType // "local" or "redis"

	// Redis options (only used if Type is "redis")
	RedisAddr     string // Redis address (e.g., "localhost:6379")
	RedisPassword string // Redis password
	RedisDB       int    // Redis database number
	RedisTLS      bool   // Enable TLS for Redis connection
}
