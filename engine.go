package go_ssr

import (
	"context"
	"log/slog"
	"os"

	"github.com/yejune/go-react-ssr/internal/cache"
	"github.com/yejune/go-react-ssr/internal/jsruntime"
	"github.com/yejune/go-react-ssr/internal/utils"
)

type Engine struct {
	Logger                  *slog.Logger
	Config                  *Config
	HotReload               *HotReload
	Cache                   cache.Cache
	RuntimePool             *jsruntime.Pool
	CachedLayoutCSSFilePath string
}

// New creates a new gossr Engine instance
func New(config Config) (*Engine, error) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	if err := os.Setenv("APP_ENV", config.AppEnv); err != nil {
		logger.Error("Failed to set APP_ENV environment variable", "error", err)
	}

	// Validate config first to set defaults
	err := config.Validate()
	if err != nil {
		logger.Error("Failed to validate config", "error", err)
		return nil, err
	}

	// Initialize cache based on config
	cacheInstance, err := cache.NewCache(config.CacheConfig)
	if err != nil {
		logger.Error("Failed to initialize cache", "error", err)
		return nil, err
	}

	engine := &Engine{
		Logger: logger,
		Config: &config,
		Cache:  cacheInstance,
	}

	// Initialize the JS runtime pool after validation (defaults are now set)
	engine.RuntimePool = jsruntime.NewPool(jsruntime.PoolConfig{
		PoolSize: config.JSRuntimePoolSize,
	})
	engine.Logger.Debug("Initialized JS runtime pool",
		"runtime", jsruntime.DefaultRuntimeType(),
		"pool_size", config.JSRuntimePoolSize)
	utils.CleanCacheDirectories()
	// If using a layout css file, build it and cache it
	if config.LayoutCSSFilePath != "" {
		if err = engine.BuildLayoutCSSFile(); err != nil {
			engine.Logger.Error("Failed to build layout css file", "error", err)
			return nil, err
		}
	}

	// Initialize dev tools (hot reload, type converter) - no-op in prod builds
	if err := engine.initDevTools(); err != nil {
		return nil, err
	}

	return engine, nil
}

// Shutdown gracefully shuts down the engine and releases all resources.
// It should be called when the server is shutting down.
// The context can be used to set a timeout for the shutdown.
func (engine *Engine) Shutdown(ctx context.Context) error {
	engine.Logger.Info("Shutting down go-react-ssr engine")

	// Close the runtime pool
	if engine.RuntimePool != nil {
		engine.RuntimePool.Close()
		engine.Logger.Debug("Runtime pool closed")
	}

	// Clear the cache
	if engine.Cache != nil {
		if err := engine.Cache.Clear(); err != nil {
			engine.Logger.Error("Failed to clear cache", "error", err)
		}
		engine.Logger.Debug("Cache cleared")
	}

	// Stop hot reload server (dev only)
	engine.stopHotReload()

	engine.Logger.Info("go-react-ssr engine shutdown complete")
	return nil
}
