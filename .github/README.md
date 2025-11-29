# gotossr

React SSR made simple with Go. **"go to SSR"** — just go.

---

# 💭 Why gotossr?

## The Problem

You're a Go developer. You want to use React for your frontend. But React SSR requires Node.js.

**Traditional approach:**
```
Go Server (API) ──── Node.js Server (SSR) ──── Browser
   :8080                 :3000
```

Now you're managing two servers, two runtimes, two deployment pipelines. Your simple Go app became a distributed system.

## The Solution

gotossr embeds a JavaScript runtime (V8 or QuickJS) directly into your Go binary.

**gotossr approach:**
```
Go Server (API + SSR) ──── Browser
        :8080
```

One server. One binary. One deployment.

## Philosophy

1. **Server-first**: Like htmx, we believe the server should render HTML. Unlike htmx, we use React's ecosystem.

2. **No Node.js tax**: React is great. Node.js runtime on your server is not required for SSR.

3. **Plugin, not framework**: Drop into any existing Go web server. Keep your architecture.

4. **Honest trade-offs**: We're not Next.js. We don't have ISR, Streaming SSR, or Image Optimization. We have one thing: Go + React SSR without Node.js.

## When to Use gotossr

✅ **Good fit:**
- Go backend that needs a React frontend
- Internal tools, admin dashboards
- Teams that know Go but want React UI
- "I just want React SSR without the Node.js overhead"

❌ **Not a good fit:**
- High-traffic consumer apps (use Next.js)
- Complex client-side React apps with heavy interactivity
- Teams already comfortable with Node.js infrastructure

## Limitations (Honest Assessment)

| What we don't have | Why |
|--------------------|-----|
| Static Site Generation (SSG) | We're SSR-only |
| Incremental Static Regeneration (ISR) | No build-time generation |
| Streaming SSR | V8/QuickJS don't support React 18 streaming |
| Image Optimization | Use a CDN |
| Edge Runtime | We run on your server |
| App Router / Server Components | React 18 features require Node.js internals |

**If you need these features, use Next.js.** We provide a [migration guide](#-migration-guide-gotossr--nextjs) when you outgrow us.

---

<p>
    <a href="https://goreportcard.com/report/github.com/yejune/gotossr"><img src="https://goreportcard.com/badge/github.com/yejune/gotossr" alt="Go Report"></a>
    <a href="https://pkg.go.dev/github.com/yejune/gotossr?tab=doc"><img src="http://img.shields.io/badge/GoDoc-Reference-blue.svg" alt="GoDoc"></a>
    <a href="https://github.com/yejune/gotossr/blob/master/LICENSE"><img src="https://img.shields.io/badge/License-MIT%202.0-blue.svg" alt="MIT License"></a>
</p>

gotossr is a drop in plugin to **any** existing Go web framework to allow **server rendering** [React](https://react.dev/). It's powered by [esbuild](https://esbuild.github.io/) and allows for passing props from Go to React with **type safety**.

<!--
# 💡 Overview -->

gotossr was developed due to a lack of an existing product in the Go ecosystem that made it easy to build full-stack React apps. At the time, most Go web app projects were either built with a static React frontend with lots of client-side logic or html templates. I envisioned creating a new solution that would allow you to create full-stack Go apps with React but with logic being moved to the server and being able to pass that logic down with type-safe props. This project was inspired by [Remix](https://remix.run/) and [Next.JS](https://nextjs.org/), but aims to be a plugin and not a framework.

# 📜 Features

- Lightning fast compiling with [esbuild](https://esbuild.github.io/)
- **V8 JavaScript engine support** for high performance SSR
- **Runtime pooling** for optimal resource usage
- Auto generated Typescript structs for props
- Hot reloading in development
- Simple error reporting
- Production optimized with build tags
- Drop in to any existing Go web server
- Minimal dependencies (2 in production mode)

<!-- _View more examples [here](github.com/yejune/go-react_old-ssr/examples)_ -->

# 🛠️ Getting Started

gotossr was designed with the idea of being dead simple to install. Below are 2 easy ways of setting it up:

## ⚡️ Using the CLI tool

<img src="https://i.imgur.com/mygp5BT.png" height="400" />

The easiest way to get a project up and running is by using the command line tool. Install it with the following command

```console
$ go install github.com/yejune/gotossr/gossr-cli@latest
```

Then you can call the following command to create a project

```console
$ gossr-cli create
```

You'll be prompted the path to place the project, what web framework you want to use, and whether or not you want to use Tailwind

## 📝 Add to existing web server

To add gotossr to an existing Go web server, take a look at the [examples](/examples) folder to get an idea of what a project looks like. In general, you'll want to follow these commands:

```console
$ go get -u github.com/yejune/gotossr
```

Then, add imports into your main file

```go
import (
	...
	gossr "github.com/yejune/gotossr"
)
```

In your main function, initialize the plugin. Create a folder for your structs that hold your props to go, which is called `models` in the below example. You'll also want to create a folder for your React code (called `frontend` in this example) inside your project and specifiy the paths in the config. You may want to clone the [example folder](/examples/frontend/) and use that.

```go
engine, err := gossr.New(gossr.Config{
    AppEnv:             "development", // or "production"
    AssetRoute:         "/assets",
    FrontendDir:        "./frontend/src",
    GeneratedTypesPath: "./frontend/src/generated.d.ts",
    PropsStructsPath:   "./models/props.go",
})
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `AppEnv` | string | `"development"` | `"development"` or `"production"` |
| `AssetRoute` | string | - | Route to serve assets (e.g., `"/assets"`) |
| `FrontendDir` | string | - | Path to React source directory |
| `GeneratedTypesPath` | string | - | Path for generated TypeScript types |
| `PropsStructsPath` | string | - | Path to Go props structs file |
| `LayoutFilePath` | string | - | Optional layout file path |
| `LayoutCSSFilePath` | string | - | Optional global CSS file path |
| `TailwindConfigPath` | string | - | Optional Tailwind config path |
| `HotReloadServerPort` | int | `3001` | Hot reload WebSocket port |
| `JSRuntimePoolSize` | int | `10` | Number of JS runtimes in pool |

Once the plugin has been initialized, you can call the `engine.RenderRoute` function to compile your React file to a string

```go
g.GET("/", func(c *gin.Context) {
	renderedResponse := engine.RenderRoute(gossr.RenderConfig{
		File:  "Home.tsx", 
		Title: "Example app", 
		MetaTags: map[string]string{
			"og:title":    "Example app", 
			"description": "Hello world!",
		}, 
		Props: &models.IndexRouteProps{
			InitialCount: rand.Intn(100),
		},
	})
	c.Writer.Write(renderedResponse)
})
```

# ⚡ Performance

gotossr supports two JavaScript runtimes:

| Runtime | Build Tag | Performance | Use Case |
|---------|-----------|-------------|----------|
| QuickJS | (default) | Good | Development, low memory |
| V8 | `-tags=use_v8` | **70-85% faster** | Production, high traffic |

### Benchmarks (Apple M4)

```
                          QuickJS      V8         Improvement
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Simple render             225μs       199μs      +12%
Complex render            234μs       137μs      +70%
Parallel (10 cores)       49μs        26μs       +85%
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

# 🏗️ Build Tags

gotossr uses build tags to minimize dependencies in production:

| Build Command | Runtime | Dev Features | Dependencies |
|---------------|---------|--------------|--------------|
| `go build` | QuickJS | ✅ Hot reload, TypeGen | 5 |
| `go build -tags=use_v8` | V8 | ✅ Hot reload, TypeGen | 5 |
| `go build -tags=prod` | QuickJS | ❌ | **2** |
| `go build -tags="prod,use_v8"` | V8 | ❌ | **2** |

### Recommended Production Build

```bash
# Fastest production build with V8
go build -tags="prod,use_v8" -ldflags "-w -s" -o main .
```

# 🚀 Deploying to production

All of the examples come with a Dockerfile that you can use to deploy to production. You can also use the [gossr-cli](#-using-the-cli-tool) to create a project with a Dockerfile.
Below is an example Dockerfile

```Dockerfile
# Build backend with V8 runtime for best performance
FROM golang:1.24-alpine as build-backend
RUN apk add --no-cache git build-base
ADD . /build
WORKDIR /build

RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -tags="prod,use_v8" -ldflags "-w -s" -o main .

# Build frontend
FROM node:20-alpine as build-frontend
ADD ./frontend /frontend
WORKDIR /frontend
RUN npm install

# Final image
FROM alpine:latest
RUN apk add --no-cache libstdc++ libgcc
COPY --from=build-backend /build/main ./app/main
COPY --from=build-frontend /frontend ./app/frontend

WORKDIR /app
RUN chmod +x ./main
EXPOSE 8080
CMD ["./main"]
```

### Lightweight Build (QuickJS)

If you prefer a smaller image without V8:

```Dockerfile
FROM golang:1.24-alpine as build-backend
ADD . /build
WORKDIR /build
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -tags=prod -ldflags "-w -s" -o main .

FROM node:20-alpine as build-frontend
ADD ./frontend /frontend
WORKDIR /frontend
RUN npm install

FROM alpine:latest
COPY --from=build-backend /build/main ./app/main
COPY --from=build-frontend /frontend ./app/frontend
WORKDIR /app
EXPOSE 8080
CMD ["./main"]
```

Go SSR has been tested and deployed on the following platforms:

- [Fly.io](https://fly.io/) - [example app](https://sparkling-smoke-7627.fly.dev/)
- [Render](https://render.com/) - [example app](https://my-gossr-test.onrender.com/)
- [Hop.io](https://hop.io/) - [example app](https://my-gossr-test.hop.sh/)

# 🏛️ Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        HTTP Request                              │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Go Web Framework                             │
│              (Fiber / Gin / Echo / net/http)                     │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                      gotossr Engine                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐  │
│  │   esbuild   │  │  JS Runtime │  │       Cache Manager     │  │
│  │  (bundler)  │  │  Pool (V8/  │  │  (in-memory, per-route) │  │
│  │             │  │   QuickJS)  │  │                         │  │
│  └─────────────┘  └─────────────┘  └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Rendered HTML Response                       │
└─────────────────────────────────────────────────────────────────┘
```

### Request Flow

1. **First Request (Cache Miss)**
   - esbuild bundles the React component (~100-500ms)
   - JS runtime pool executes `renderToString()` (~10-50ms)
   - Result is cached in memory

2. **Subsequent Requests (Cache Hit)**
   - Props are injected into cached JS bundle
   - JS runtime executes immediately (~10-25ms)
   - No bundling overhead

# 📊 Comparison: gotossr vs Next.js + Go

### Architecture Comparison

**Next.js + Go (Traditional):**
```
Browser ──▶ Next.js (SSR) ──▶ Go API ──▶ Database
           Port 3000         Port 8080
           ~500MB RAM        ~50MB RAM
```
- 2 servers to manage
- 4 network hops per request
- Higher infrastructure cost

**gotossr (Single Server):**
```
Browser ──▶ Go Server (SSR + API) ──▶ Database
           Port 8080
           ~100-200MB RAM
```
- 1 server to manage
- 2 network hops per request
- Lower infrastructure cost

### Performance Comparison

| Metric | Next.js + Go | gotossr (V8) |
|--------|-------------|-------------------|
| SSR Latency | 5-20ms | 10-30ms |
| Internal API Call | 5-10ms | 0ms (not needed) |
| **Total Latency** | **35-55ms** | **35-55ms** |
| Memory (idle) | 550MB | 100MB |
| Memory (1000 conn) | 900MB | 200MB |
| Throughput | 500-1000 req/s | 200-500 req/s |

### Monthly Infrastructure Cost (AWS)

| Item | Next.js + Go | gotossr |
|------|-------------|--------------|
| EC2 Instances | $50-80 (2x) | $20-30 (1x) |
| Load Balancers | $40 (2x) | $20 (1x) |
| **Monthly Total** | **$90-120** | **$40-50** |

**Annual Savings: $600-840**

### Feature Comparison

| Feature | gotossr | Next.js |
|---------|--------------|---------|
| SSR | ✅ | ✅ |
| SSG/ISR | ❌ | ✅ |
| Streaming SSR | ❌ | ✅ |
| Image Optimization | ❌ | ✅ |
| Type Safety | ✅ Auto-gen | Manual |
| Single Deployment | ✅ | ❌ |
| Memory Efficiency | ✅ | ❌ |

### When to Use gotossr

✅ **Recommended:**
- Existing Go backend that needs SSR
- Internal tools / admin dashboards
- B2B applications
- Cost optimization priority
- Traffic < 100k req/day
- Teams with Go expertise

❌ **Use Next.js + Go instead:**
- High-traffic consumer apps (>100k req/day)
- Complex React applications with many interactions
- Teams with strong Node.js expertise

### Production Readiness

| Capability | Status |
|------------|--------|
| SSR Rendering | ✅ Stable |
| Type Safety | ✅ Auto-generated |
| Runtime Pooling | ✅ V8/QuickJS |
| Graceful Shutdown | ✅ Built-in |
| Caching | ✅ Local (default) / Redis (optional) |

**Verdict: Production-ready for low-to-medium traffic applications**

### Caching Options

gotossr caches esbuild bundle results to avoid re-bundling on every request.

| Cache Type | Use Case | Configuration |
|------------|----------|---------------|
| Local (default) | Single server, simple setup | No config needed |
| Redis | Multiple servers, shared cache | `RedisAddr` in config |

**Local Cache (default):**
- Each server caches independently
- First request per server: ~200ms (bundling)
- Subsequent requests: ~0ms (cache hit)
- Sufficient for most use cases

**Redis Cache (optional):**
- All servers share one cache
- Only first request globally: ~200ms
- All other requests: ~1ms (Redis fetch)
- Useful for faster cold starts across many servers

```go
// Redis cache configuration (optional)
engine, _ := gossr.New(gossr.Config{
    // ... other config
    CacheType:     "redis",              // "local" (default) or "redis"
    RedisAddr:     "localhost:6379",
    RedisPassword: "",                   // optional
    RedisDB:       0,                    // optional
    RedisTLS:      true,                 // optional, for TLS connection
})
```

### Graceful Shutdown Example

```go
package main

import (
    "context"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    gossr "github.com/yejune/gotossr"
)

func main() {
    engine, _ := gossr.New(gossr.Config{
        AppEnv:      "production",
        FrontendDir: "./frontend/src",
        // ...
    })

    srv := &http.Server{Addr: ":8080"}

    // Handle routes
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        response := engine.RenderRoute(gossr.RenderConfig{
            File: "Home.tsx",
        })
        w.Write(response)
    })

    // Start server in goroutine
    go func() {
        if err := srv.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    <-quit

    // Graceful shutdown with 10s timeout
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Shutdown HTTP server
    srv.Shutdown(ctx)

    // Shutdown gotossr engine (releases runtime pool, clears cache)
    engine.Shutdown(ctx)
}
```

# 🔄 Migration Guide: gotossr → Next.js

When your traffic exceeds 100k req/day or you need advanced features (SSG, ISR, Streaming SSR), migrate to Next.js with this guide.

### Step 1: Create Next.js Project

```bash
# Create Next.js app alongside your Go project
npx create-next-app@latest frontend-nextjs --typescript --app

# Your new structure:
your-project/
├── go-server/           # Existing Go backend (keep as API)
│   ├── main.go
│   └── handlers/
├── frontend-nextjs/     # New Next.js frontend
│   ├── app/
│   └── package.json
└── frontend/            # Old gotossr frontend (to migrate)
```

### Step 2: Copy React Components

```bash
# Copy your React components (they work as-is!)
cp -r frontend/src/components frontend-nextjs/components/
cp -r frontend/src/*.tsx frontend-nextjs/app/

# Components are compatible - no changes needed
```

### Step 3: Convert Props to Data Fetching

**Before (gotossr):**
```go
// Go handler
func HomeHandler(c *gin.Context) {
    data := getDataFromDB()
    response := engine.RenderRoute(gossr.RenderConfig{
        File:  "Home.tsx",
        Props: &HomeProps{Data: data},
    })
    c.Writer.Write(response)
}
```

**After (Next.js + Go API):**
```typescript
// app/page.tsx
async function HomePage() {
  // Fetch from your Go API
  const res = await fetch('http://go-server:8080/api/home', {
    cache: 'no-store' // SSR
  });
  const data = await res.json();

  return <Home data={data} />;
}

// Or use ISR for better performance
async function HomePage() {
  const res = await fetch('http://go-server:8080/api/home', {
    next: { revalidate: 60 } // ISR: regenerate every 60s
  });
  const data = await res.json();
  return <Home data={data} />;
}
```

### Step 4: Add Go API Endpoints

```go
// Add API endpoints to your Go server
func main() {
    r := gin.Default()

    // New: API endpoints for Next.js
    api := r.Group("/api")
    {
        api.GET("/home", func(c *gin.Context) {
            data := getDataFromDB()
            c.JSON(200, data)
        })
        api.GET("/products", getProducts)
        api.GET("/users/:id", getUser)
    }

    // Old: Keep gotossr routes during migration
    r.GET("/", homeHandler)

    r.Run(":8080")
}
```

### Step 5: Gradual Migration with Reverse Proxy

```nginx
# nginx.conf - Route by path during migration
upstream nextjs {
    server localhost:3000;
}
upstream go {
    server localhost:8080;
}

server {
    listen 80;

    # New pages → Next.js
    location /new/ {
        proxy_pass http://nextjs;
    }

    # Migrated pages → Next.js
    location /products {
        proxy_pass http://nextjs;
    }

    # Old pages → gotossr (until migrated)
    location / {
        proxy_pass http://go;
    }

    # API → Go
    location /api/ {
        proxy_pass http://go;
    }
}
```

### Step 6: Update Docker Compose

```yaml
# docker-compose.yml
services:
  go-api:
    build: ./go-server
    ports:
      - "8080:8080"
    environment:
      - APP_ENV=production

  nextjs:
    build: ./frontend-nextjs
    ports:
      - "3000:3000"
    environment:
      - API_URL=http://go-api:8080
    depends_on:
      - go-api

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - go-api
      - nextjs
```

### Step 7: Type Sharing (Optional)

```bash
# Install OpenAPI generator for type sync
npm install @openapitools/openapi-generator-cli -D

# Generate TypeScript types from Go API
# In Go, use swaggo/swag to generate OpenAPI spec
go install github.com/swaggo/swag/cmd/swag@latest
swag init

# Generate TypeScript client
npx openapi-generator-cli generate \
  -i http://localhost:8080/swagger/doc.json \
  -g typescript-fetch \
  -o frontend-nextjs/lib/api
```

### Migration Checklist

```markdown
## Page Migration Tracker

| Page | gotossr | Next.js | Verified |
|------|-------------|---------|----------|
| / (Home) | ✅ | ⬜ | ⬜ |
| /products | ✅ | ⬜ | ⬜ |
| /dashboard | ✅ | ⬜ | ⬜ |
| /settings | ✅ | ⬜ | ⬜ |

## API Endpoints

| Endpoint | Created | Tested |
|----------|---------|--------|
| GET /api/home | ⬜ | ⬜ |
| GET /api/products | ⬜ | ⬜ |
| GET /api/user/:id | ⬜ | ⬜ |
```

### Timeline Estimate

| Phase | Duration | Tasks |
|-------|----------|-------|
| Setup | 1-2 days | Next.js project, Docker, Nginx |
| Simple Pages | 1 week | Static pages, basic data fetching |
| Complex Pages | 2-3 weeks | Forms, auth, real-time features |
| Testing | 1 week | E2E tests, performance comparison |
| Cutover | 1 day | DNS switch, monitoring |

**Total: 4-6 weeks for typical project**

### Rollback Plan

If issues arise, rollback is simple:

```nginx
# nginx.conf - Rollback to gotossr
location / {
    proxy_pass http://go;  # All traffic back to Go
}
```

# 🎨 CSS Framework Support

gotossr supports multiple CSS frameworks out of the box:

### Tailwind CSS v4

```bash
# Install Tailwind v4
npm install tailwindcss @tailwindcss/cli
```

```css
/* src/Main.css */
@import "tailwindcss";

@theme {
  --color-primary: #3b82f6;
}
```

```go
engine, err := gossr.New(gossr.Config{
    // ...
    LayoutCSSFilePath:  "Main.css",
    TailwindConfigPath: "./frontend/tailwind.config.js",
})
```

### Bootstrap 5

```bash
# Install Bootstrap 5
npm install bootstrap react-bootstrap
```

```tsx
// In your React component
import { Button, Container } from "react-bootstrap";
import "bootstrap/dist/css/bootstrap.min.css";

function Home() {
  return (
    <Container>
      <Button variant="primary">Click me</Button>
    </Container>
  );
}
```

### Plain CSS / CSS Modules

No additional configuration needed. Just import your CSS files in your React components.

### Example Projects

| Framework | Directory | Description |
|-----------|-----------|-------------|
| Plain CSS | `examples/frontend/` | Basic CSS styling |
| Tailwind v4 | `examples/frontend-tailwind/` | Tailwind CSS with @theme |
| Bootstrap 5 | `examples/frontend-bootstrap/` | react-bootstrap components |
| MUI | `examples/frontend-mui/` | Material UI components |

# 📁 Project Structure

```
your-project/
├── main.go                 # Go entry point
├── models/
│   └── props.go            # Props structs (auto-converted to TS)
├── frontend/
│   ├── src/
│   │   ├── Home.tsx        # React components
│   │   ├── Layout.tsx      # Optional layout
│   │   └── generated.d.ts  # Auto-generated types
│   └── package.json
└── go.mod
```

# 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

# 📄 License

MIT License - see [LICENSE](../LICENSE) for details.
