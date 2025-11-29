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
renderToString(<App {...props} />);  // 작동
renderToString(<StaticRouter><App /></StaticRouter>);  // 빈 문자열, 에러 없음
```

**이게 제일 미친 케이스:** 에러가 없다. 그냥 빈 문자열.

---

## 2. 핵심 발견

### React가 에러를 삼킨다

QuickJS 환경에서 `console.error`가 void 처리됨
→ React 내부 에러가 아무데도 안 찍힘
→ 그냥 빈 문자열만 반환

### 해결: console.error 캡처

```javascript
globalThis.__ssr_errors = [];
var console = {
  log: function(){},
  error: function(){
    globalThis.__ssr_errors.push(arguments);
  }
};
```

---

## 3. 진짜 원인

React Router의 `encodeLocation` 함수:

```javascript
function encodeLocation(to) {
  let encoded = new URL(href, "http://localhost");  // ← QuickJS에서 터짐!
}
```

**QuickJS에는 URL API가 없다.**

브라우저/Node.js에서는 당연히 있는 API가 QuickJS에는 없음.

---

## 4. 수정 내용

### build.go - URL polyfill

```go
var urlPolyfill = `if(typeof URL==="undefined"){
  function URL(u,b){
    // RFC 3986 regex로 URL 파싱
    var m=u.match(/^(([^:/?#]+):)?(\/\/([^/?#]*))?([^?#]*)(\?([^#]*))?(#(.*))?/);
    this.href=u;
    this.protocol=(m[2]||"")+":";
    this.host=m[4]||"";
    this.pathname=m[5]||"/";
    this.search=m[6]||"";
    this.hash=m[8]||"";
  }
}`
```

### build.go - 에러 캡처 & 출력

```go
// Footer에서 에러를 HTML 주석으로 출력
Footer: map[string]string{
    "js": `globalThis.__ssr_result +
           (globalThis.__ssr_errors.length ?
            '<!-- SSR_ERRORS: ' + __ssr_errors.join(' | ') + ' -->' :
            '')`,
},
```

### contents.go - try-catch

```go
var serverSPARouterRenderFunction = `try {
  globalThis.__ssr_result = renderToString(...);
} catch(e) {
  globalThis.__ssr_errors.push('RENDER_ERROR: ' + e.stack);
  globalThis.__ssr_result = '';
}`
```

### engine.go - CSS 캐싱 (부가 버그)

SPA 모드에서 CSS가 빈 문자열로 반환되던 버그도 같이 수정:

```go
type Engine struct {
    CachedServerSPACSS string  // 추가
}

// buildServerSPAApp()에서 저장
engine.CachedServerSPACSS = result.CSS
```

### rendertask.go - 캐시된 CSS 사용

```go
// 이전: css: ""
// 수정: css: rt.engine.CachedServerSPACSS
```

---

## 5. 디버깅 방법

SSR이 빈 문자열 나올 때:

```bash
# 1. SSR_ERRORS 확인
curl http://localhost:8080/ | grep SSR_ERRORS

# 2. minification 끄고 재빌드
# build.go에서:
MinifyWhitespace:  false
MinifyIdentifiers: false
MinifySyntax:      false

# 3. 에러 위치 확인
<!-- SSR_ERRORS: at encodeLocation (<input>:52766:73) -->
```

---

## 6. QuickJS에 없는 API들

| API | 상태 |
|-----|------|
| `URL` | polyfill 추가됨 |
| `URLSearchParams` | 필요시 추가 |
| `TextEncoder/Decoder` | polyfill 있음 |
| `fetch` | 없음 (SSR에서 안 씀) |
| `AbortController` | 없음 |
| `Blob`, `FormData` | 없음 |

---

## 7. 수정된 파일 목록

```
internal/reactbuilder/build.go     # URL polyfill, console.error 캡처
internal/reactbuilder/contents.go  # try-catch 에러 핸들링
engine.go                          # CachedServerSPACSS 필드 추가
rendertask.go                      # SPA에서 CSS 반환
```

---

## 8. 교훈

1. **에러가 안 나온다 ≠ 에러가 없다**
2. **QuickJS는 브라우저 아님** - Web API 없음
3. **console.error 캡처는 SSR 디버깅 필수**
4. **minification 끄면 에러 위치 보임**
5. **컴포넌트 하나씩 테스트로 문제 격리**

---

## 관련 커밋

```
ba01063 - Add URL polyfill and SPA CSS caching for QuickJS SSR
30e176c - Rename module to github.com/yejune/gotossr
```
