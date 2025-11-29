# QuickJS + React Router SSR 트러블슈팅

> 2025-11-30 | gotossr 라이브러리 SSR 버그 해결 기록

---

## TL;DR

**문제:** `StaticRouter`로 감싸면 SSR 결과가 빈 문자열
**원인:** QuickJS에 `URL` Web API 없음 → React Router 내부에서 터짐
**해결:** URL polyfill 추가

---

## 1. 요청 사항

```
환경:
- go-react-ssr (Go + esbuild + QuickJS)
- React 서버 렌더링

작동하는 기존 코드:
renderToString(<App {...props} />);

안되는 새 코드:
renderToString(<StaticRouter location={url}><App /></StaticRouter>);

증상:
- esbuild 번들링 성공
- QuickJS 실행 시 에러 없음
- 하지만 결과가 빈 문자열 ""

질문:
1. 같은 renderToString() 호출인데 왜 StaticRouter 감싸면 빈 문자열?
2. React Router의 StaticRouter가 QuickJS 환경에서 특별히 필요한 게 있나?
```

**이게 제일 미친 케이스:** 에러가 없다. 그냥 빈 문자열.

---

## 2. 디버깅 과정

### 2.1 문제 격리 - 테스트 컴포넌트 생성

StaticRouter 자체가 문제인지, Routes가 문제인지 확인하기 위해 단계별 테스트 컴포넌트 생성:

```tsx
// TestApp.tsx - 간단한 Routes 테스트
import { Routes, Route } from 'react-router-dom';

function SimplePage() {
  return <div>Hello from SimplePage!</div>;
}

export default function TestApp() {
  return (
    <Routes>
      <Route path="/" element={<SimplePage />} />
      <Route path="*" element={<div>404 Not Found</div>} />
    </Routes>
  );
}
```

```tsx
// MinimalApp.tsx - Router 없이 순수 React만
export default function MinimalApp() {
  return (
    <div>
      <h1>Hello World!</h1>
      <p>This is a minimal test app.</p>
    </div>
  );
}
```

```tsx
// HookTestApp.tsx - useLocation hook 테스트
import { useLocation } from 'react-router-dom';

export default function HookTestApp() {
  const location = useLocation();
  return (
    <div>
      <h1>Hook Test</h1>
      <p>Current path: {location.pathname}</p>
    </div>
  );
}
```

### 2.2 테스트 결과

| 테스트 | 결과 |
|--------|------|
| `MinimalApp` (Router 없음) | ✅ 작동 |
| `HookTestApp` (useLocation) | ✅ 작동 |
| `TestApp` (Routes 사용) | ❌ 빈 문자열 |
| `RoutesTestApp` (Routes + useLocation) | ❌ 빈 문자열 |

**결론:** `useLocation()`은 되는데 `Routes`/`useRoutes()`만 안 됨

### 2.3 에러 캡처 시스템 추가

React가 에러를 삼키고 있었음. console.error를 캡처하도록 수정:

**build.go 수정:**
```go
// 기존
var consolePolyfill = `var console = {log: function(){}};`

// 수정
var consolePolyfill = `globalThis.__ssr_errors=[];var console = {
  log: function(){},
  warn: function(){},
  error: function(){
    var a=Array.prototype.slice.call(arguments);
    globalThis.__ssr_errors.push(a.map(function(x){
      return x&&x.stack?x.stack:String(x)
    }).join(' '));
  }
};`
```

**Footer에서 에러 출력:**
```go
// 기존
Footer: map[string]string{
    "js": "globalThis.__ssr_result",
},

// 수정
Footer: map[string]string{
    "js": "globalThis.__ssr_result+(globalThis.__ssr_errors&&globalThis.__ssr_errors.length?'<!-- SSR_ERRORS: '+globalThis.__ssr_errors.join(' | ')+' -->':'')",
},
```

### 2.4 minification 비활성화

에러 위치를 정확히 파악하기 위해 minification 임시 비활성화:

```go
// build.go
MinifyWhitespace:  false,  // true → false
MinifyIdentifiers: false,  // true → false
MinifySyntax:      false,  // true → false
```

### 2.5 에러 발견!

```bash
curl -s http://localhost:8080/ | grep SSR_ERRORS
```

결과:
```
<!-- SSR_ERRORS: RENDER_ERROR: at encodeLocation (<input>:52766:73) -->
```

### 2.6 원인 파악

`react-router-dom/server.js` 분석:

```javascript
function encodeLocation(to) {
  let href = typeof to === "string" ? to : createPath(to);
  // 여기서 URL constructor 사용 - QuickJS에서 실패!
  let encoded = new URL(href, "http://localhost");
  return {
    pathname: encoded.pathname,
    search: encoded.search,
    hash: encoded.hash
  };
}
```

**QuickJS에는 URL Web API가 없다!**

브라우저/Node.js에서는 당연히 있는 `URL` 클래스가 QuickJS에는 없음.

---

## 3. 해결책

### 3.1 URL Polyfill 추가

RFC 3986 기반 URL 파싱 polyfill:

```go
// internal/reactbuilder/build.go
var urlPolyfill = `if(typeof URL==="undefined"){
  function URL(u,b){
    if(b&&u.indexOf("://")===-1){
      u=b.replace(/\/$/,"")+"/"+u.replace(/^\//,"")
    }
    var m=u.match(/^(([^:/?#]+):)?(\/\/([^/?#]*))?([^?#]*)(\?([^#]*))?(#(.*))?/);
    this.href=u;
    this.protocol=(m[2]||"")+":";
    this.host=m[4]||"";
    this.hostname=this.host.split(":")[0];
    this.port=this.host.split(":")[1]||"";
    this.pathname=m[5]||"/";
    this.search=m[6]||"";
    this.hash=m[8]||"";
    this.origin=this.protocol+"//"+this.host
  }
  URL.prototype.toString=function(){return this.href}
}`
```

Banner에 polyfill 추가:
```go
Banner: map[string]string{
    "js": globalThisPolyfill + urlPolyfill + textEncoderPolyfill + processPolyfill + consolePolyfill,
},
```

### 3.2 try-catch 에러 핸들링

렌더링 실패해도 앱이 죽지 않도록:

```go
// internal/reactbuilder/contents.go
// 기존
var serverSPARouterRenderFunction = `globalThis.__ssr_result = renderToString(<StaticRouter location={props.__requestPath}><App /></StaticRouter>)`

// 수정
var serverSPARouterRenderFunction = `try {
  globalThis.__ssr_result = renderToString(<StaticRouter location={props.__requestPath}><App /></StaticRouter>);
} catch(e) {
  globalThis.__ssr_errors.push('RENDER_ERROR: ' + (e.stack || e.message || String(e)));
  globalThis.__ssr_result = '';
}`
```

### 3.3 CSS 캐싱 버그 수정 (부가 발견)

SPA 모드에서 CSS가 빈 문자열로 반환되던 버그도 발견:

**engine.go:**
```go
type Engine struct {
    // 기존 필드들...
    CachedServerSPACSS string  // 추가
}

// buildServerSPAApp()에서
engine.CachedServerSPAJS = result.JS
engine.CachedServerSPACSS = result.CSS  // 추가
```

**rendertask.go:**
```go
// 기존
rt.serverRenderResult <- serverRenderResult{html: renderedHTML, css: "", err: err}

// 수정
rt.serverRenderResult <- serverRenderResult{html: renderedHTML, css: rt.engine.CachedServerSPACSS, err: err}
```

---

## 4. 수정된 파일 상세

### internal/reactbuilder/build.go

```go
// 1. URL polyfill 변수 추가 (line 29)
var urlPolyfill = `if(typeof URL==="undefined"){function URL(u,b){...}}`

// 2. console polyfill 수정 - error 캡처 추가 (line 28)
var consolePolyfill = `globalThis.__ssr_errors=[];var console = {...}`

// 3. Banner에 urlPolyfill 추가 (line 58)
Banner: map[string]string{
    "js": globalThisPolyfill + urlPolyfill + textEncoderPolyfill + processPolyfill + consolePolyfill,
},

// 4. Footer에 에러 출력 추가 (line 62)
Footer: map[string]string{
    "js": "globalThis.__ssr_result+(globalThis.__ssr_errors&&globalThis.__ssr_errors.length?'<!-- SSR_ERRORS: '+globalThis.__ssr_errors.join(' | ')+' -->':'')",
},
```

### internal/reactbuilder/contents.go

```go
// serverSPARouterRenderFunction에 try-catch 추가 (line 21)
var serverSPARouterRenderFunction = `try {
  globalThis.__ssr_result = renderToString(<StaticRouter location={props.__requestPath}><App /></StaticRouter>);
} catch(e) {
  globalThis.__ssr_errors.push('RENDER_ERROR: ' + (e.stack || e.message || String(e)));
  globalThis.__ssr_result = '';
}`
```

### engine.go

```go
// Engine struct에 필드 추가 (line 23)
type Engine struct {
    // ...
    CachedServerSPACSS string
}

// buildServerSPAApp()에서 CSS 저장 (line 148)
engine.CachedServerSPACSS = result.CSS

// 로그에 cssLen 추가 (line 155)
engine.Logger.Debug("Built server SPA app", ..., "cssLen", len(result.CSS), ...)
```

### rendertask.go

```go
// SPA 렌더링 결과에 캐시된 CSS 사용 (line 82)
rt.serverRenderResult <- serverRenderResult{
    html: renderedHTML,
    css: rt.engine.CachedServerSPACSS,  // 기존: ""
    err: err,
}
```

---

## 5. 디버깅 체크리스트

SSR이 빈 문자열 반환할 때:

```bash
# 1. SSR_ERRORS 확인
curl http://localhost:8080/ | grep SSR_ERRORS

# 2. 서버 로그 확인
# htmlLen=0 이면 렌더링 실패
time=... level=DEBUG msg="SPA server render result" htmlLen=0 requestPath=/

# 3. minification 끄고 재빌드
# build.go에서:
MinifyWhitespace:  false
MinifyIdentifiers: false
MinifySyntax:      false

# 4. 에러 위치 확인 (minify 끄면 읽을 수 있음)
<!-- SSR_ERRORS: at encodeLocation (<input>:52766:73) -->

# 5. 단계별 테스트
# - 순수 React 컴포넌트 → 작동하면 React 자체는 OK
# - useLocation() → 작동하면 StaticRouter context는 OK
# - Routes 사용 → 실패하면 route matching 문제
```

---

## 6. QuickJS에 없는 API들

| API | 상태 | 비고 |
|-----|------|------|
| `URL` | ✅ polyfill 추가됨 | React Router 필수 |
| `URLSearchParams` | ❌ 없음 | 필요시 추가 |
| `TextEncoder/Decoder` | ✅ polyfill 있음 | 기존에 추가됨 |
| `fetch` | ❌ 없음 | SSR에서 안 씀 |
| `AbortController` | ❌ 없음 | |
| `Blob`, `FormData` | ❌ 없음 | |
| `Headers`, `Request`, `Response` | ❌ 없음 | |

---

## 7. 교훈

1. **에러가 안 나온다 ≠ 에러가 없다**
   - React가 내부적으로 에러를 삼킴
   - console.error 캡처 필수

2. **QuickJS는 브라우저가 아님**
   - Web API 대부분 없음
   - Node.js도 아님
   - 필요한 API는 직접 polyfill

3. **minification은 디버깅의 적**
   - 에러 위치 파악 불가능
   - 개발 중에는 끄고 테스트

4. **문제 격리가 핵심**
   - 컴포넌트 하나씩 테스트
   - 어디서 실패하는지 범위 좁히기

5. **서버 로그를 믿지 마라**
   - htmlLen=0 + 에러 없음 = 뭔가 잘못됨
   - HTML 응답을 직접 확인

---

## 8. 관련 커밋

```
ba01063 - Add URL polyfill and SPA CSS caching for QuickJS SSR
30e176c - Rename module to github.com/yejune/gotossr
356f32a - Update troubleshooting doc with detailed debugging process
```

---

## 9. 테스트 검증

```bash
# 빌드 및 실행
go build -tags='!wails' -o /tmp/unified_web .
/tmp/unified_web &

# SSR 확인 - HTML이 있어야 함
curl -s http://localhost:8080/ | grep -o '<div id="root">.*</div>' | head -c 200

# CSS 확인 - 스타일이 있어야 함
curl -s http://localhost:8080/ | grep -A5 '<style>'

# 에러 확인 - 없어야 함
curl -s http://localhost:8080/ | grep SSR_ERRORS
# (출력 없으면 정상)
```
