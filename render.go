package go_ssr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path"
	"path/filepath"

	"github.com/yejune/gotossr/internal/html"
	"github.com/yejune/gotossr/internal/utils"
)

// RenderConfig is the config for rendering a route
type RenderConfig struct {
	File        string
	Title       string
	MetaTags    map[string]string
	Props       interface{}
	RequestPath string // Current request URL path for SPA routing (e.g., "/board/1")
}

// RenderRoute renders a route to html
func (engine *Engine) RenderRoute(renderConfig RenderConfig) []byte {
	filePath := filepath.ToSlash(utils.GetFullFilePath(engine.Config.FrontendDir + "/" + renderConfig.File))

	// Generate stable routeID from file path (survives binary rebuilds)
	routeID := generateRouteID(filePath)

	props, err := propsToString(renderConfig.Props)
	if err != nil {
		return html.RenderError(err, routeID)
	}
	task := renderTask{
		engine:   engine,
		logger:   engine.Logger,
		routeID:  routeID,
		props:    props,
		filePath: filePath,
		config:   renderConfig,
	}
	renderedHTML, css, js, err := task.Start()
	if err != nil {
		return html.RenderError(err, task.routeID)
	}

	params := html.Params{
		Title:      renderConfig.Title,
		MetaTags:   renderConfig.MetaTags,
		RouteID:    task.routeID,
		ServerHTML: template.HTML(renderedHTML),
		PropsJSON:  template.JS(props), // SSR props for client hydration
	}

	// External JS file mode: write JS to file and use <script src>
	if engine.Config.StaticJSDir != "" {
		jsPath, err := engine.writeStaticJS(js, routeID)
		if err != nil {
			engine.Logger.Error("Failed to write static JS", "error", err)
			// Fallback to inline
			params.JS = template.JS(js)
		} else {
			params.JSPath = jsPath
		}
		// CSS도 외부 파일로 (TODO: 별도 구현 가능)
		params.CSS = template.CSS(css)
	} else {
		// Inline mode (default)
		params.JS = template.JS(js)
		params.CSS = template.CSS(css)
	}

	return html.RenderHTMLString(params)
}

// writeStaticJS writes JS to a file and returns the URL path
func (engine *Engine) writeStaticJS(js string, routeID string) (string, error) {
	// Generate hash from JS content for cache busting
	hash := sha256.Sum256([]byte(js))
	hashStr := hex.EncodeToString(hash[:8])

	// Filename: app-{routeID}.{hash}.js
	filename := fmt.Sprintf("app-%s.%s.js", routeID[:8], hashStr)
	filePath := path.Join(engine.Config.StaticJSDir, filename)

	// Check if file already exists (same content)
	if _, err := os.Stat(filePath); err == nil {
		// File exists, return URL path
		return engine.Config.AssetRoute + "/" + filename, nil
	}

	// Write JS to file
	if err := os.WriteFile(filePath, []byte(js), 0644); err != nil {
		return "", fmt.Errorf("failed to write JS file: %w", err)
	}

	engine.Logger.Info("Written static JS file", "path", filePath, "size", len(js))
	return engine.Config.AssetRoute + "/" + filename, nil
}

// generateRouteID creates a stable route ID from file path
func generateRouteID(filePath string) string {
	hash := sha256.Sum256([]byte(filePath))
	return hex.EncodeToString(hash[:8]) // 16 char hex string
}

// Convert props to JSON string, or set to null if no props are passed
func propsToString(props interface{}) (string, error) {
	if props != nil {
		propsJSON, err := json.Marshal(props)
		return string(propsJSON), err
	}
	return "null", nil
}
