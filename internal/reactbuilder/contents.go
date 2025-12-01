package reactbuilder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

var baseTemplate = `
import React from "react";
{{range $import := .Imports}}{{$import}} {{end}}
import App from "{{ .FilePath }}";
{{ if .SuppressConsoleLog }}console.log = () => {};{{ end }}
{{ .RenderFunction }}`
var serverRenderFunction = `renderToString(<App {...props} />);`
var serverRenderFunctionWithLayout = `renderToString(<Layout><App {...props} /></Layout>);`
var clientRenderFunction = `hydrateRoot(document.getElementById("root"), <App {...props} />);`
var clientRenderFunctionWithLayout = `hydrateRoot(document.getElementById("root"), <Layout><App {...props} /></Layout>);`

// SPA render functions - "router" mode: uses Router wrapping for true hydration
// globalThis is never minified, so the result survives esbuild optimization
var serverSPARouterRenderFunction = `try { globalThis.__ssr_result = renderToString(<StaticRouter location={props.__requestPath}><App {...props} /></StaticRouter>); } catch(e) { globalThis.__ssr_errors.push('RENDER_ERROR: ' + (e.stack || e.message || String(e))); globalThis.__ssr_result = ''; }`
var clientSPARouterRenderFunction = `
const ssrPropsEl = document.getElementById("__SSR_PROPS__");
const ssrProps = ssrPropsEl ? JSON.parse(ssrPropsEl.textContent || "{}") : {};
hydrateRoot(document.getElementById("root"), <BrowserRouter><App {...ssrProps} /></BrowserRouter>);`

// SPA render functions - "replace" mode: uses createRoot to replace SSR HTML (backward compatible)
var clientSPAReplaceRenderFunction = `
const root = document.getElementById("root");
root.innerHTML = "";
createRoot(root).render(<App />);`

func buildWithTemplate(buildTemplate string, params map[string]interface{}) (string, error) {
	templ, err := template.New("buildTemplate").Parse(buildTemplate)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	err = templ.Execute(&out, params)
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

func GenerateServerBuildContents(imports []string, filePath string, useLayout bool) (string, error) {
	imports = append(imports, `import { renderToString } from "react-dom/server.browser";`)
	params := map[string]interface{}{
		"Imports":            imports,
		"FilePath":           filePath,
		"RenderFunction":     serverRenderFunction,
		"SuppressConsoleLog": true,
	}
	if useLayout {
		params["RenderFunction"] = serverRenderFunctionWithLayout
	}
	return buildWithTemplate(baseTemplate, params)
}

func GenerateClientBuildContents(imports []string, filePath string, useLayout bool) (string, error) {
	imports = append(imports, `import { hydrateRoot } from "react-dom/client";`)
	params := map[string]interface{}{
		"Imports":        imports,
		"FilePath":       filePath,
		"RenderFunction": clientRenderFunction,
	}
	if useLayout {
		params["RenderFunction"] = clientRenderFunctionWithLayout
	}
	return buildWithTemplate(baseTemplate, params)
}

// getReactRouterMajorVersion reads package.json and returns react-router-dom major version
func getReactRouterMajorVersion(frontendDir string) int {
	// Try current dir first, then parent dir (common structure: frontend/src with package.json in frontend/)
	pkgPath := filepath.Join(frontendDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		// Try parent directory
		pkgPath = filepath.Join(filepath.Dir(frontendDir), "package.json")
		data, err = os.ReadFile(pkgPath)
		if err != nil {
			return 6 // default to v6
		}
	}
	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return 6
	}
	version := pkg.Dependencies["react-router-dom"]
	if version == "" {
		return 6
	}
	// Remove ^ or ~ prefix and get major version
	version = strings.TrimLeft(version, "^~")
	parts := strings.Split(version, ".")
	if len(parts) > 0 {
		major, err := strconv.Atoi(parts[0])
		if err == nil {
			return major
		}
	}
	return 6
}

// GenerateServerSPABuildContents generates server build for SPA apps
// mode: "router" uses StaticRouter for true hydration, "replace" uses page component rendering
func GenerateServerSPABuildContents(imports []string, appPath string, mode string, frontendDir string) (string, error) {
	if mode == "router" {
		imports = append(imports, `import { renderToString } from "react-dom/server.browser";`)
		// react-router-dom v7+ uses "react-router" for StaticRouter, v6 uses "react-router-dom/server"
		if getReactRouterMajorVersion(frontendDir) >= 7 {
			imports = append(imports, `import { StaticRouter } from "react-router";`)
		} else {
			imports = append(imports, `import { StaticRouter } from "react-router-dom/server";`)
		}
		params := map[string]interface{}{
			"Imports":            imports,
			"FilePath":           appPath,
			"RenderFunction":     serverSPARouterRenderFunction,
			"SuppressConsoleLog": true,
		}
		return buildWithTemplate(baseTemplate, params)
	}
	// "replace" mode: no server SPA build needed, use individual page rendering
	return "", nil
}

// GenerateClientSPABuildContents generates client SPA app build
// mode: "router" uses hydrateRoot with BrowserRouter, "replace" uses createRoot (backward compatible)
func GenerateClientSPABuildContents(imports []string, appPath string, mode string) (string, error) {
	if mode == "router" {
		imports = append(imports, `import { hydrateRoot } from "react-dom/client";`)
		imports = append(imports, `import { BrowserRouter } from "react-router-dom";`)
		params := map[string]interface{}{
			"Imports":        imports,
			"FilePath":       appPath,
			"RenderFunction": clientSPARouterRenderFunction,
		}
		return buildWithTemplate(baseTemplate, params)
	}
	// "replace" mode: uses createRoot to replace SSR HTML
	imports = append(imports, `import { createRoot } from "react-dom/client";`)
	params := map[string]interface{}{
		"Imports":        imports,
		"FilePath":       appPath,
		"RenderFunction": clientSPAReplaceRenderFunction,
	}
	return buildWithTemplate(baseTemplate, params)
}
