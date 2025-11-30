package go_ssr

import (
	"context"
	"log/slog"
	"os"

	"github.com/yejune/gotossr/internal/cache"
	"github.com/yejune/gotossr/internal/jsruntime"
	"github.com/yejune/gotossr/internal/reactbuilder"
	"github.com/yejune/gotossr/internal/utils"
)

type Engine struct {
	Logger                  *slog.Logger
	Config                  *Config
	HotReload               *HotReload
	Cache                   cache.Cache
	RuntimePool             *jsruntime.Pool
	CachedLayoutCSSFilePath string
	CachedClientSPAJS       string // Cached client SPA bundle JS
	CachedServerSPAJS       string // Cached server SPA bundle JS (for StaticRouter rendering)
	CachedServerSPACSS      string // Cached server SPA bundle CSS
}

// IsProduction returns true if running in production mode
func (engine *Engine) IsProduction() bool {
	return engine.Config.AppEnv == "production"
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

	// If using client SPA app, build bundles based on SPAHydrationMode
	if config.ClientAppPath != "" {
		if config.SPAHydrationMode == "router" {
			// "router" mode: build server SPA bundle for StaticRouter rendering
			if err = engine.buildServerSPAApp(); err != nil {
				engine.Logger.Error("Failed to build server SPA app", "error", err)
				return nil, err
			}
		}
		// Both modes need client SPA bundle
		if err = engine.buildClientSPAApp(); err != nil {
			engine.Logger.Error("Failed to build client SPA app", "error", err)
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

// buildServerSPAApp builds the server SPA app bundle (with StaticRouter for "router" mode)
func (engine *Engine) buildServerSPAApp() error {
	imports := []string{}
	if engine.CachedLayoutCSSFilePath != "" {
		imports = append(imports, `import "`+engine.CachedLayoutCSSFilePath+`";`)
	}

	buildContents, err := reactbuilder.GenerateServerSPABuildContents(imports, engine.Config.ClientAppPath, engine.Config.SPAHydrationMode)
	if err != nil {
		return err
	}
	if buildContents == "" {
		// "replace" mode doesn't need server SPA build
		return nil
	}

	result, err := reactbuilder.BuildServer(buildContents, engine.Config.FrontendDir, engine.Config.AssetRoute)
	if err != nil {
		return err
	}

	engine.CachedServerSPAJS = result.JS
	engine.CachedServerSPACSS = result.CSS
	// Debug: show last 500 chars of generated JS
	jsLen := len(result.JS)
	lastPart := result.JS
	if jsLen > 500 {
		lastPart = result.JS[jsLen-500:]
	}
	engine.Logger.Debug("Built server SPA app", "path", engine.Config.ClientAppPath, "mode", engine.Config.SPAHydrationMode, "jsLen", jsLen, "cssLen", len(result.CSS), "lastPart", lastPart)
	return nil
}

// buildClientSPAApp builds the client SPA app bundle
// "router" mode: uses hydrateRoot with BrowserRouter
// "replace" mode: uses createRoot (backward compatible)
func (engine *Engine) buildClientSPAApp() error {
	imports := []string{}
	if engine.CachedLayoutCSSFilePath != "" {
		imports = append(imports, `import "`+engine.CachedLayoutCSSFilePath+`";`)
	}

	buildContents, err := reactbuilder.GenerateClientSPABuildContents(imports, engine.Config.ClientAppPath, engine.Config.SPAHydrationMode)
	if err != nil {
		return err
	}

	result, err := reactbuilder.BuildClient(buildContents, engine.Config.FrontendDir, engine.Config.AssetRoute, engine.IsProduction())
	if err != nil {
		return err
	}

	engine.CachedClientSPAJS = result.JS
	engine.Logger.Debug("Built client SPA app", "path", engine.Config.ClientAppPath, "mode", engine.Config.SPAHydrationMode)
	return nil
}
