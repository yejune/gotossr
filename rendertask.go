package go_ssr

import (
	"fmt"
	"log/slog"

	"github.com/yejune/go-react-ssr/internal/jsruntime"
	"github.com/yejune/go-react-ssr/internal/reactbuilder"
)

type renderTask struct {
	engine             *Engine
	logger             *slog.Logger
	routeID            string
	filePath           string
	props              string
	config             RenderConfig
	serverRenderResult chan serverRenderResult
	clientRenderResult chan clientRenderResult
}

type serverRenderResult struct {
	html string
	css  string
	err  error
}

type clientRenderResult struct {
	js           string
	dependencies []string
	err          error
}

// Start starts the render task, returns the rendered html, css, and js for hydration
func (rt *renderTask) Start() (string, string, string, error) {
	rt.serverRenderResult = make(chan serverRenderResult)
	rt.clientRenderResult = make(chan clientRenderResult)
	// Assigns the parent file to the routeID so that the cache can be invalidated when the parent file changes
	if err := rt.engine.Cache.SetParentFile(rt.routeID, rt.filePath); err != nil {
		rt.logger.Error("Failed to set parent file", "error", err)
	}

	// Render for server and client concurrently
	go rt.doRender("server")
	go rt.doRender("client")

	// Wait for both to finish
	srResult := <-rt.serverRenderResult
	if srResult.err != nil {
		rt.logger.Error("Failed to build for server", "error", srResult.err)
		return "", "", "", srResult.err
	}
	crResult := <-rt.clientRenderResult
	if crResult.err != nil {
		rt.logger.Error("Failed to build for client", "error", crResult.err)
		return "", "", "", crResult.err
	}

	// Set the parent file dependencies so that the cache can be invalidated a dependency changes
	go func() {
		if err := rt.engine.Cache.SetParentFileDependencies(rt.filePath, crResult.dependencies); err != nil {
			rt.logger.Error("Failed to set parent file dependencies", "error", err)
		}
	}()
	return srResult.html, srResult.css, crResult.js, nil
}

func (rt *renderTask) doRender(buildType string) {
	// For client builds with ClientAppPath, use cached SPA bundle
	if buildType == "client" && rt.engine.CachedClientSPAJS != "" {
		rt.clientRenderResult <- clientRenderResult{js: rt.engine.CachedClientSPAJS, dependencies: nil}
		return
	}

	// Check if the build is in the cache
	build, buildFound, err := rt.getBuildFromCache(buildType)
	if err != nil {
		rt.logger.Error("Failed to get build from cache", "error", err, "buildType", buildType)
	}
	if !buildFound {
		// Build the file if it's not in the cache
		newBuild, err := rt.buildFile(buildType)
		if err != nil {
			rt.handleBuildError(err, buildType)
			return
		}
		rt.updateBuildCache(newBuild, buildType)
		build = newBuild
	}
	// JS is built without props so that the props can be injected into cached JS builds
	js := injectProps(build.JS, rt.props)
	if buildType == "server" {
		// Execute the JS using the pooled runtime
		renderedHTML, err := rt.renderReactToHTML(js)
		rt.serverRenderResult <- serverRenderResult{html: renderedHTML, css: build.CSS, err: err}
	} else {
		rt.clientRenderResult <- clientRenderResult{js: js, dependencies: build.Dependencies}
	}
}

// getBuild returns the build from the cache if it exists
func (rt *renderTask) getBuildFromCache(buildType string) (reactbuilder.BuildResult, bool, error) {
	if buildType == "server" {
		return rt.engine.Cache.GetServerBuild(rt.filePath)
	} else {
		return rt.engine.Cache.GetClientBuild(rt.filePath)
	}
}

// buildFile gets the contents of the file to be built and builds it with reactbuilder
func (rt *renderTask) buildFile(buildType string) (reactbuilder.BuildResult, error) {
	buildContents, err := rt.getBuildContents(buildType)
	if err != nil {
		return reactbuilder.BuildResult{}, err
	}
	if buildType == "server" {
		return reactbuilder.BuildServer(buildContents, rt.engine.Config.FrontendDir, rt.engine.Config.AssetRoute)
	} else {
		return reactbuilder.BuildClient(buildContents, rt.engine.Config.FrontendDir, rt.engine.Config.AssetRoute, rt.engine.IsProduction())
	}
}

// getBuildContents gets the required imports based on the config and returns the contents to be built with reactbuilder
func (rt *renderTask) getBuildContents(buildType string) (string, error) {
	var imports []string
	if rt.engine.CachedLayoutCSSFilePath != "" {
		imports = append(imports, fmt.Sprintf(`import "%s";`, rt.engine.CachedLayoutCSSFilePath))
	}
	if rt.engine.Config.LayoutFilePath != "" {
		imports = append(imports, fmt.Sprintf(`import Layout from "%s";`, rt.engine.Config.LayoutFilePath))
	}
	if buildType == "server" {
		return reactbuilder.GenerateServerBuildContents(imports, rt.filePath, rt.engine.Config.LayoutFilePath != "")
	} else {
		return reactbuilder.GenerateClientBuildContents(imports, rt.filePath, rt.engine.Config.LayoutFilePath != "")
	}
}

// handleBuildError handles the error from building the file and sends it to the appropriate channel
func (rt *renderTask) handleBuildError(err error, buildType string) {
	if buildType == "server" {
		rt.serverRenderResult <- serverRenderResult{err: err}
	} else {
		rt.clientRenderResult <- clientRenderResult{err: err}
	}
}

// updateBuildCache updates the cache with the new build
func (rt *renderTask) updateBuildCache(build reactbuilder.BuildResult, buildType string) {
	var err error
	if buildType == "server" {
		err = rt.engine.Cache.SetServerBuild(rt.filePath, build)
	} else {
		err = rt.engine.Cache.SetClientBuild(rt.filePath, build)
	}
	if err != nil {
		rt.logger.Error("Failed to update build cache", "error", err, "buildType", buildType)
	}
}

// injectProps injects the props into the already compiled JS
func injectProps(compiledJS, props string) string {
	return fmt.Sprintf(`var props = %s; %s`, props, compiledJS)
}

// renderReactToHTML executes the server JS using the pooled runtime
func (rt *renderTask) renderReactToHTML(js string) (string, error) {
	return rt.engine.RuntimePool.Execute(js)
}

// renderReactToHTMLWithPool is a package-level function for backward compatibility
func renderReactToHTMLWithPool(pool *jsruntime.Pool, js string) (string, error) {
	return pool.Execute(js)
}
