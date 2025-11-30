# gotossr

Build Go powered React web apps with server-side rendering.

---

<p>
    <a href="https://goreportcard.com/report/github.com/yejune/gotossr"><img src="https://goreportcard.com/badge/github.com/yejune/gotossr" alt="Go Report"></a>
    <a href="https://pkg.go.dev/github.com/yejune/gotossr?tab=doc"><img src="http://img.shields.io/badge/GoDoc-Reference-blue.svg" alt="GoDoc"></a>
    <a href="https://github.com/yejune/gotossr/blob/main/LICENSE"><img src="https://img.shields.io/badge/License-MIT%202.0-blue.svg" alt="MIT License"></a>
</p>

gotossr is a drop in plugin to **any** existing Go web framework to allow **server rendering** [React](https://react.dev/). It's powered by [esbuild](https://esbuild.github.io/) and allows for passing props from Go to React with **type safety**.

> Forked from [natewong1313/go-react-ssr](https://github.com/natewong1313/go-react-ssr)

## What's New

- **V8 Runtime Support** - 70-85% faster than QuickJS
- **Runtime Pooling** - Reuse JS runtimes for better performance
- **Redis Cache** - Optional distributed cache for multi-server deployments
- **Graceful Shutdown** - Clean resource cleanup
- **Build Tags** - Minimize dependencies in production (2 deps only)
- **Reduced Dependencies** - Replaced zerolog/jsonparser with stdlib

# 📜 Features

- Lightning fast compiling with [esbuild](https://esbuild.github.io/)
- V8 or QuickJS runtime (selectable)
- Auto generated Typescript structs for props
- Hot reloading in development
- Simple error reporting
- Production optimized with build tags
- Drop in to any existing Go web server

# 🛠️ Getting Started

## ⚡️ Using the CLI tool

```console
$ go install github.com/yejune/gotossr/gossr-cli@latest
$ gossr-cli create
```

## 📝 Add to existing web server

```console
$ go get -u github.com/yejune/gotossr
```

```go
import gossr "github.com/yejune/gotossr"

engine, err := gossr.New(gossr.Config{
    AppEnv:             "development", // or "production"
    AssetRoute:         "/assets",
    FrontendDir:        "./frontend/src",
    GeneratedTypesPath: "./frontend/src/generated.d.ts",
    PropsStructsPath:   "./models/props.go",
})
```

```go
g.GET("/", func(c *gin.Context) {
    response := engine.RenderRoute(gossr.RenderConfig{
        File:  "Home.tsx",
        Title: "Example app",
        Props: &models.IndexRouteProps{
            InitialCount: rand.Intn(100),
        },
    })
    c.Writer.Write(response)
})
```

# ⚡ Performance

| Runtime | Build Tag | Performance |
|---------|-----------|-------------|
| QuickJS | (default) | Good |
| V8 | `-tags=use_v8` | **70-85% faster** |

# 🏗️ Build Tags

| Build Command | Runtime | Dependencies |
|---------------|---------|--------------|
| `go build` | QuickJS | 5 |
| `go build -tags=use_v8` | V8 | 5 |
| `go build -tags=prod` | QuickJS | **2** |
| `go build -tags="prod,use_v8"` | V8 | **2** |

# 🚀 Deploying to production

```bash
go build -tags="prod,use_v8" -ldflags "-w -s" -o main .
```

Example Dockerfile:

```Dockerfile
FROM golang:1.24-alpine as build-backend
RUN apk add --no-cache git build-base
ADD . /build
WORKDIR /build
RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -tags="prod,use_v8" -ldflags "-w -s" -o main .

FROM node:20-alpine as build-frontend
ADD ./frontend /frontend
WORKDIR /frontend
RUN npm install

FROM alpine:latest
RUN apk add --no-cache libstdc++ libgcc
COPY --from=build-backend /build/main ./app/main
COPY --from=build-frontend /frontend ./app/frontend
WORKDIR /app
EXPOSE 8080
CMD ["./main"]
```

# 📄 License

MIT License - see [LICENSE](../LICENSE) for details.
