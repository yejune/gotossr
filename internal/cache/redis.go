package cache

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"time"

	"github.com/yejune/go-react-ssr/internal/reactbuilder"
	"github.com/redis/go-redis/v9"
)

// RedisCache provides distributed caching via Redis
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

// RedisConfig configures the Redis cache
type RedisConfig struct {
	Addr     string        // Redis address (e.g., "localhost:6379")
	Password string        // Redis password (empty for no auth)
	DB       int           // Redis database number
	TTL      time.Duration // Cache TTL (0 = no expiration)
	Prefix   string        // Key prefix (default: "gossr:")
	UseTLS   bool          // Enable TLS connection
}

// NewRedisCache creates a new Redis cache
func NewRedisCache(config RedisConfig) (*RedisCache, error) {
	opts := &redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	}

	// Enable TLS if configured
	if config.UseTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	prefix := config.Prefix
	if prefix == "" {
		prefix = "gossr:"
	}

	return &RedisCache{
		client: client,
		ttl:    config.TTL,
		prefix: prefix,
	}, nil
}

// GetServerBuild retrieves a server build from Redis
func (rc *RedisCache) GetServerBuild(filePath string) (reactbuilder.BuildResult, bool, error) {
	ctx := context.Background()
	key := rc.prefix + "server:" + filePath
	data, err := rc.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return reactbuilder.BuildResult{}, false, nil
	}
	if err != nil {
		return reactbuilder.BuildResult{}, false, err
	}

	var result reactbuilder.BuildResult
	if err := json.Unmarshal(data, &result); err != nil {
		return reactbuilder.BuildResult{}, false, err
	}

	return result, true, nil
}

// SetServerBuild stores a server build in Redis
func (rc *RedisCache) SetServerBuild(filePath string, build reactbuilder.BuildResult) error {
	ctx := context.Background()
	key := rc.prefix + "server:" + filePath
	data, err := json.Marshal(build)
	if err != nil {
		return err
	}

	return rc.client.Set(ctx, key, data, rc.ttl).Err()
}

// RemoveServerBuild removes a server build from Redis
func (rc *RedisCache) RemoveServerBuild(filePath string) error {
	ctx := context.Background()
	key := rc.prefix + "server:" + filePath
	return rc.client.Del(ctx, key).Err()
}

// GetClientBuild retrieves a client build from Redis
func (rc *RedisCache) GetClientBuild(filePath string) (reactbuilder.BuildResult, bool, error) {
	ctx := context.Background()
	key := rc.prefix + "client:" + filePath
	data, err := rc.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return reactbuilder.BuildResult{}, false, nil
	}
	if err != nil {
		return reactbuilder.BuildResult{}, false, err
	}

	var result reactbuilder.BuildResult
	if err := json.Unmarshal(data, &result); err != nil {
		return reactbuilder.BuildResult{}, false, err
	}

	return result, true, nil
}

// SetClientBuild stores a client build in Redis
func (rc *RedisCache) SetClientBuild(filePath string, build reactbuilder.BuildResult) error {
	ctx := context.Background()
	key := rc.prefix + "client:" + filePath
	data, err := json.Marshal(build)
	if err != nil {
		return err
	}

	return rc.client.Set(ctx, key, data, rc.ttl).Err()
}

// RemoveClientBuild removes a client build from Redis
func (rc *RedisCache) RemoveClientBuild(filePath string) error {
	ctx := context.Background()
	key := rc.prefix + "client:" + filePath
	return rc.client.Del(ctx, key).Err()
}

// SetParentFile maps a routeID to a parent file path
func (rc *RedisCache) SetParentFile(routeID, filePath string) error {
	ctx := context.Background()
	key := rc.prefix + "routes"
	return rc.client.HSet(ctx, key, routeID, filePath).Err()
}

// GetRouteIDSForParentFile returns all route IDs for a given file path
func (rc *RedisCache) GetRouteIDSForParentFile(filePath string) ([]string, error) {
	ctx := context.Background()
	key := rc.prefix + "routes"
	result, err := rc.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var routes []string
	for route, file := range result {
		if file == filePath {
			routes = append(routes, route)
		}
	}
	return routes, nil
}

// GetAllRouteIDS returns all route IDs
func (rc *RedisCache) GetAllRouteIDS() ([]string, error) {
	ctx := context.Background()
	key := rc.prefix + "routes"
	result, err := rc.client.HKeys(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetRouteIDSWithFile returns route IDs associated with a file
func (rc *RedisCache) GetRouteIDSWithFile(filePath string) ([]string, error) {
	reactFilesWithDependency, err := rc.GetParentFilesFromDependency(filePath)
	if err != nil {
		return nil, err
	}
	if len(reactFilesWithDependency) == 0 {
		reactFilesWithDependency = []string{filePath}
	}
	var routeIDS []string
	for _, reactFile := range reactFilesWithDependency {
		routes, err := rc.GetRouteIDSForParentFile(reactFile)
		if err != nil {
			return nil, err
		}
		routeIDS = append(routeIDS, routes...)
	}
	return routeIDS, nil
}

// SetParentFileDependencies sets dependencies for a parent file with reverse index
func (rc *RedisCache) SetParentFileDependencies(filePath string, dependencies []string) error {
	ctx := context.Background()

	// Get old dependencies to remove from reverse index
	oldDepsKey := rc.prefix + "deps:" + filePath
	oldDepsData, err := rc.client.Get(ctx, oldDepsKey).Bytes()
	var oldDeps []string
	if err == nil {
		json.Unmarshal(oldDepsData, &oldDeps)
	}

	// Remove from reverse index for old dependencies
	reverseKey := rc.prefix + "revdeps"
	for _, dep := range oldDeps {
		rc.client.SRem(ctx, reverseKey+":"+dep, filePath)
	}

	// Set forward index
	data, err := json.Marshal(dependencies)
	if err != nil {
		return err
	}
	if err := rc.client.Set(ctx, oldDepsKey, data, rc.ttl).Err(); err != nil {
		return err
	}

	// Add to reverse index for new dependencies
	for _, dep := range dependencies {
		if err := rc.client.SAdd(ctx, reverseKey+":"+dep, filePath).Err(); err != nil {
			return err
		}
		// Set TTL on reverse index key if TTL is configured
		if rc.ttl > 0 {
			rc.client.Expire(ctx, reverseKey+":"+dep, rc.ttl)
		}
	}

	return nil
}

// GetParentFilesFromDependency returns parent files that depend on a given file using reverse index
func (rc *RedisCache) GetParentFilesFromDependency(dependencyPath string) ([]string, error) {
	ctx := context.Background()
	reverseKey := rc.prefix + "revdeps:" + dependencyPath

	result, err := rc.client.SMembers(ctx, reverseKey).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Clear removes all gossr keys from cache
func (rc *RedisCache) Clear() error {
	ctx := context.Background()
	pattern := rc.prefix + "*"
	var cursor uint64
	for {
		keys, nextCursor, err := rc.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			if err := rc.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

// Invalidate removes a specific key from cache
func (rc *RedisCache) Invalidate(filePath string) error {
	ctx := context.Background()
	keys := []string{
		rc.prefix + "server:" + filePath,
		rc.prefix + "client:" + filePath,
	}
	return rc.client.Del(ctx, keys...).Err()
}

// Close closes the Redis connection
func (rc *RedisCache) Close() error {
	return rc.client.Close()
}

// Stats returns cache statistics
func (rc *RedisCache) Stats(ctx context.Context) (map[string]interface{}, error) {
	info, err := rc.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}

	// Count gossr keys
	pattern := rc.prefix + "*"
	var count int64
	var cursor uint64
	for {
		keys, nextCursor, err := rc.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}
		count += int64(len(keys))
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return map[string]interface{}{
		"type":       "redis",
		"key_count":  count,
		"prefix":     rc.prefix,
		"redis_info": info,
	}, nil
}
