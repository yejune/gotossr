package go_ssr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"path/filepath"

	"github.com/yejune/go-react-ssr/internal/html"
	"github.com/yejune/go-react-ssr/internal/utils"
)

// RenderConfig is the config for rendering a route
type RenderConfig struct {
	File     string
	Title    string
	MetaTags map[string]string
	Props    interface{}
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
	return html.RenderHTMLString(html.Params{
		Title:      renderConfig.Title,
		MetaTags:   renderConfig.MetaTags,
		JS:         template.JS(js),
		CSS:        template.CSS(css),
		RouteID:    task.routeID,
		ServerHTML: template.HTML(renderedHTML),
	})
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
