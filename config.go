package go_ssr

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/yejune/gotossr/internal/cache"
	"github.com/yejune/gotossr/internal/utils"
)

// Config is the config for starting the engine
type Config struct {
	AppEnv              string            // "production" or "development"
	AssetRoute          string            // The route to serve assets from, e.g. "/assets"
	FrontendDir         string            // The path to the frontend folder, where your React app lives
	GeneratedTypesPath  string            // The path to the generated types file
	PropsStructsPath    string            // The path to the Go structs file(s), comma-separated for multiple files
	LayoutFilePath      string            // The path to the layout file, relative to the frontend dir
	LayoutCSSFilePath   string            // The path to the layout css file, relative to the frontend dir
	TailwindConfigPath  string            // The path to the tailwind config file
	HotReloadServerPort int               // The port to run the hot reload server on, 3001 by default
	JSRuntimePoolSize   int               // The number of JS runtimes to keep in the pool, 10 by default
	CacheConfig         cache.CacheConfig // Cache configuration (local or redis)
	ClientAppPath       string            // Path to client SPA app (e.g., "App.tsx") for client-side routing after hydration
	// SPA hydration mode options (only used when ClientAppPath is set):
	// - "router": wraps with StaticRouter/BrowserRouter for true hydration (default, requires react-router-dom)
	// - "replace": uses createRoot to replace SSR HTML (no hydration, compatible with any SPA structure)
	SPAHydrationMode string // "router" or "replace", defaults to "router"
	// External JS file options (for browser caching optimization):
	// When StaticJSDir is set, JS bundles are written to files instead of inlined in HTML.
	// This enables browser caching - the React library bundle rarely changes and can be cached.
	StaticJSDir string // Directory to write JS files (e.g., "frontend/dist/assets"). If empty, JS is inlined.
	IsDev       bool   // Development mode - enables hot reload, disables caching
}

// Validate validates the config
func (c *Config) Validate() error {
	if !checkPathExists(c.FrontendDir) {
		return fmt.Errorf("frontend dir at %s does not exist", c.FrontendDir)
	}
	// Check all props struct paths (comma-separated)
	if os.Getenv("APP_ENV") != "production" && c.PropsStructsPath != "" {
		for _, p := range strings.Split(c.PropsStructsPath, ",") {
			p = strings.TrimSpace(p)
			if p != "" && !checkPathExists(p) {
				return fmt.Errorf("props structs path at %s does not exist", p)
			}
		}
	}
	if c.LayoutFilePath != "" && !checkPathExists(path.Join(c.FrontendDir, c.LayoutFilePath)) {
		return fmt.Errorf("layout file path at %s/%s does not exist", c.FrontendDir, c.LayoutFilePath)
	}
	if c.LayoutCSSFilePath != "" && !checkPathExists(path.Join(c.FrontendDir, c.LayoutCSSFilePath)) {
		return fmt.Errorf("layout css file path at %s/%s does not exist", c.FrontendDir, c.LayoutCSSFilePath)
	}
	if c.TailwindConfigPath != "" && c.LayoutCSSFilePath == "" {
		return fmt.Errorf("layout css file path must be provided when using tailwind")
	}
	if c.ClientAppPath != "" && !checkPathExists(path.Join(c.FrontendDir, c.ClientAppPath)) {
		return fmt.Errorf("client app path at %s/%s does not exist", c.FrontendDir, c.ClientAppPath)
	}
	if c.HotReloadServerPort == 0 {
		c.HotReloadServerPort = 3001
	}
	if c.JSRuntimePoolSize == 0 {
		c.JSRuntimePoolSize = 10
	}
	// Default SPA hydration mode to "router" for true hydration with React Router
	if c.ClientAppPath != "" && c.SPAHydrationMode == "" {
		c.SPAHydrationMode = "router"
	}
	// Create StaticJSDir if specified
	if c.StaticJSDir != "" {
		c.StaticJSDir = utils.GetFullFilePath(c.StaticJSDir)
		if err := os.MkdirAll(c.StaticJSDir, 0755); err != nil {
			return fmt.Errorf("failed to create static js dir at %s: %w", c.StaticJSDir, err)
		}
	}
	c.setFilePaths()
	return nil
}

// setFilePaths sets any paths in the config to their absolute paths
func (c *Config) setFilePaths() {
	c.FrontendDir = utils.GetFullFilePath(c.FrontendDir)
	c.GeneratedTypesPath = utils.GetFullFilePath(c.GeneratedTypesPath)
	// Handle multiple props struct paths
	if c.PropsStructsPath != "" {
		var fullPaths []string
		for _, p := range strings.Split(c.PropsStructsPath, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				fullPaths = append(fullPaths, utils.GetFullFilePath(p))
			}
		}
		c.PropsStructsPath = strings.Join(fullPaths, ",")
	}
	if c.LayoutFilePath != "" {
		c.LayoutFilePath = path.Join(c.FrontendDir, c.LayoutFilePath)
	}
	if c.LayoutCSSFilePath != "" {
		c.LayoutCSSFilePath = path.Join(c.FrontendDir, c.LayoutCSSFilePath)
	}
	if c.TailwindConfigPath != "" {
		c.TailwindConfigPath = utils.GetFullFilePath(c.TailwindConfigPath)
	}
	if c.ClientAppPath != "" {
		c.ClientAppPath = path.Join(c.FrontendDir, c.ClientAppPath)
	}
}

func checkPathExists(path string) bool {
	_, err := os.Stat(utils.GetFullFilePath(path))
	return !os.IsNotExist(err)
}
