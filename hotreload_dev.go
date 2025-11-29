//go:build !prod

package go_ssr

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"github.com/yejune/go-react-ssr/internal/utils"
)

type HotReload struct {
	engine           *Engine
	logger           *slog.Logger
	connectedClients map[string][]*websocket.Conn
}

// newHotReload creates a new HotReload instance
func newHotReload(engine *Engine) *HotReload {
	return &HotReload{
		engine:           engine,
		logger:           engine.Logger,
		connectedClients: make(map[string][]*websocket.Conn),
	}
}

// Start starts the hot reload server and watcher
func (hr *HotReload) Start() {
	go hr.startServer()
	go hr.startWatcher()
}

// startServer starts the hot reload websocket server
func (hr *HotReload) startServer() {
	hr.logger.Info("Hot reload websocket running", "port", hr.engine.Config.HotReloadServerPort)
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			hr.logger.Error("Failed to upgrade websocket", "error", err)
			return
		}
		// Client should send routeID as first message
		_, routeID, err := ws.ReadMessage()
		if err != nil {
			hr.logger.Error("Failed to read message from websocket", "error", err)
			return
		}
		err = ws.WriteMessage(1, []byte("Connected"))
		if err != nil {
			hr.logger.Error("Failed to write message to websocket", "error", err)
			return
		}
		// Add client to connectedClients
		hr.connectedClients[string(routeID)] = append(hr.connectedClients[string(routeID)], ws)
	})
	err := http.ListenAndServe(fmt.Sprintf(":%d", hr.engine.Config.HotReloadServerPort), nil)
	if err != nil {
		hr.logger.Error("Hot reload server quit unexpectedly", "error", err)
	}
}

// startWatcher starts the file watcher
func (hr *HotReload) startWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		hr.logger.Error("Failed to start watcher", "error", err)
		return
	}
	defer watcher.Close()
	// Walk through all files in the frontend directory and add them to the watcher
	if err = filepath.Walk(hr.engine.Config.FrontendDir, func(path string, fi os.FileInfo, err error) error {
		if fi.Mode().IsDir() {
			return watcher.Add(path)
		}
		return nil
	}); err != nil {
		hr.logger.Error("Failed to add files in directory to watcher", "error", err)
		return
	}

	for {
		select {
		case event := <-watcher.Events:
			// Watch for file created, deleted, updated, or renamed events
			if event.Op.String() != "CHMOD" && !strings.Contains(event.Name, "gossr-temporary") {
				filePath := utils.GetFullFilePath(event.Name)
				hr.logger.Info("File changed, reloading", "file", filePath)
				// Store the routes that need to be reloaded
				var routeIDS []string
				var cacheErr error
				switch {
				case filePath == hr.engine.Config.LayoutFilePath: // If the layout file has been updated, reload all routes
					routeIDS, cacheErr = hr.engine.Cache.GetAllRouteIDS()
					if cacheErr != nil {
						hr.logger.Error("Failed to get all route IDs", "error", cacheErr)
						continue
					}
				case hr.layoutCSSFileUpdated(filePath): // If the global css file has been updated, rebuild it and reload all routes
					if err := hr.engine.BuildLayoutCSSFile(); err != nil {
						hr.logger.Error("Failed to build global css file", "error", err)
						continue
					}
					routeIDS, cacheErr = hr.engine.Cache.GetAllRouteIDS()
					if cacheErr != nil {
						hr.logger.Error("Failed to get all route IDs", "error", cacheErr)
						continue
					}
				case hr.needsTailwindRecompile(filePath): // If tailwind is enabled and a React file has been updated, rebuild the global css file and reload all routes
					if err := hr.engine.BuildLayoutCSSFile(); err != nil {
						hr.logger.Error("Failed to build global css file", "error", err)
						continue
					}
					fallthrough
				default:
					// Get all route ids that use that file or have it as a dependency
					routeIDS, cacheErr = hr.engine.Cache.GetRouteIDSWithFile(filePath)
					if cacheErr != nil {
						hr.logger.Error("Failed to get route IDs with file", "error", cacheErr)
						continue
					}
				}
				// Find any parent files that import the file that was modified and delete their cached build
				parentFiles, cacheErr := hr.engine.Cache.GetParentFilesFromDependency(filePath)
				if cacheErr != nil {
					hr.logger.Error("Failed to get parent files from dependency", "error", cacheErr)
				}
				for _, parentFile := range parentFiles {
					if err := hr.engine.Cache.RemoveServerBuild(parentFile); err != nil {
						hr.logger.Error("Failed to remove server build", "error", err)
					}
					if err := hr.engine.Cache.RemoveClientBuild(parentFile); err != nil {
						hr.logger.Error("Failed to remove client build", "error", err)
					}
				}
				// Reload any routes that import the modified file
				go hr.broadcastFileUpdateToClients(routeIDS)

			}
		case err := <-watcher.Errors:
			hr.logger.Error("Error watching files", "error", err)
		}
	}
}

// layoutCSSFileUpdated checks if the layout css file has been updated
func (hr *HotReload) layoutCSSFileUpdated(filePath string) bool {
	return utils.GetFullFilePath(filePath) == hr.engine.Config.LayoutCSSFilePath
}

// needsTailwindRecompile checks if the file that was updated is a React file
func (hr *HotReload) needsTailwindRecompile(filePath string) bool {
	if hr.engine.Config.TailwindConfigPath == "" {
		return false
	}
	fileTypes := []string{".tsx", ".ts", ".jsx", ".js"}
	for _, fileType := range fileTypes {
		if strings.HasSuffix(filePath, fileType) {
			return true
		}
	}
	return false
}

// broadcastFileUpdateToClients sends a message to all connected clients to reload the page
func (hr *HotReload) broadcastFileUpdateToClients(routeIDS []string) {
	// Iterate over each route ID
	for _, routeID := range routeIDS {
		// Find all clients listening for that route ID
		for i, ws := range hr.connectedClients[routeID] {
			// Send reload message to client
			err := ws.WriteMessage(1, []byte("reload"))
			if err != nil {
				// remove client if browser is closed or page changed
				hr.connectedClients[routeID] = append(hr.connectedClients[routeID][:i], hr.connectedClients[routeID][i+1:]...)
			}
		}
	}
}
