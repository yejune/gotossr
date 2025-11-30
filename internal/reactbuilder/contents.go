package reactbuilder

import (
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
var serverSPARouterRenderFunction = `try { globalThis.__ssr_result = renderToString(<StaticRouter location={props.__requestPath}><App /></StaticRouter>); } catch(e) { globalThis.__ssr_errors.push('RENDER_ERROR: ' + (e.stack || e.message || String(e))); globalThis.__ssr_result = ''; }`
var clientSPARouterRenderFunction = `hydrateRoot(document.getElementById("root"), <BrowserRouter><App /></BrowserRouter>);`

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

// GenerateServerSPABuildContents generates server build for SPA apps
// mode: "router" uses StaticRouter for true hydration, "replace" uses page component rendering
func GenerateServerSPABuildContents(imports []string, appPath string, mode string) (string, error) {
	if mode == "router" {
		imports = append(imports, `import { renderToString } from "react-dom/server.browser";`)
		imports = append(imports, `import { StaticRouter } from "react-router-dom/server";`)
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
