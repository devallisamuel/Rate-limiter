
Rate limiter: roadmap

I wrote the snippets as skeletons with // TODO markers, so you fill in the logic. Each phase ends with a "done when" check.

Phase 0: commit what you have

gofmt -w . && go vet ./... && go test -race ./...
git add . && git commit -m "Add middleware, server and concurrency safety" && git push
Done when: git status is clean.

---

Phase 1: per-client limits

Problem: all visitors share one limiter.
Learn: maps, net.SplitHostPort, a second mutex, "get or create".

// limiter/store.go
type Store struct {
        mu       sync.Mutex
        clients  map[string]*Limiter
        limit    int
        window   time.Duration
}

func NewStore(limit int, window time.Duration) *Store { /* TODO: init the map */ }

func (s *Store) Get(key string) *Limiter {
        s.mu.Lock()
        defer s.mu.Unlock()
        // TODO: return the existing limiter, or create, save and return a new one
}

// limiter/middleware.go
func clientIP(r *http.Request) string {
        host, _, err := net.SplitHostPort(r.RemoteAddr)
        // TODO: if err != nil, fall back to r.RemoteAddr
        return host
}

func (s *Store) Middleware(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                l := s.Get(clientIP(r))
                // TODO: same allow / 429 / next logic as before, using l
        })
}

Test idea: build requests with req.RemoteAddr = "1.1.1.1:1234" and "2.2.2.2:1234". Exhaust the first, then confirm the second still gets 200.
Done when: the two-IP test passes under -race.

---

Phase 2: clean up idle clients

Problem: the map grows forever.
Learn: goroutines, time.Ticker, context.

// add to Limiter: lastSeen time.Time, updated inside Allow()

func (s *Store) Cleanup(ctx context.Context, every, idle time.Duration) {
        ticker := time.NewTicker(every)
        defer ticker.Stop()
        for {
                select {
                case <-ticker.C:
                        // TODO: lock, delete entries whose lastSeen is older than idle
                case <-ctx.Done():
                        return
                }
        }
}

// in main: go store.Cleanup(ctx, time.Minute, 10*time.Minute)
Gotcha: Allow() writes lastSeen under the limiter's lock, but cleanup reads it from another goroutine, so it needs a safe read method.
Done when: a test with a tiny idle time shows the map shrinks (add a Len() method), and there's no race.

---

Phase 3: informative responses

Learn: HTTP headers, a method that exposes state.

func (l *Limiter) Remaining() int { /* TODO: lock, return limit - counter */ }
func (l *Limiter) ResetIn() time.Duration { /* TODO: window end minus now */ }

// in the middleware, before writing the response:
w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(l.Remaining()))
// on 429 only:
w.Header().Set("Retry-After", strconv.Itoa(int(l.ResetIn().Seconds())))
Done when: curl -i shows the headers and a test asserts Retry-After on a 429.

---

Phase 4: injectable clock

Problem: tests use time.Sleep.
Learn: functions as values.

type Limiter struct {
        // ...
        now func() time.Time // defaults to time.Now in New
}

// in tests:
fake := time.Now()
l.now = func() time.Time { return fake }
fake = fake.Add(time.Minute) // "time passes" instantly
Replace every time.Now() and time.Since in the limiter with l.now().
Done when: no test calls time.Sleep.

---

Phase 5: interface and a second algorithm

Learn: interfaces, the biggest Go idea.

type Allower interface {
        Allow() bool
}

type TokenBucket struct {
        mu       sync.Mutex
        capacity float64
        tokens   float64
        rate     float64 // tokens per second
        last     time.Time
}

func (b *TokenBucket) Allow() bool {
        // TODO: refill = elapsed * rate, cap at capacity;
        //       if tokens >= 1 { tokens--; return true }; return false
}
Make Store hold Allower values (or take a factory function) so either algorithm plugs in.
Test idea: a fixed window lets 2x the limit through across a boundary, and a token bucket doesn't. Write a test that shows it.
Done when: both types satisfy Allower, share tests, and the middleware doesn't care which it gets.

---

Phase 6: production readiness

Learn: flag/os, http.Server, signals.

// config from env
port := os.Getenv("PORT")
if port == "" { port = "8080" }
limit, _ := strconv.Atoi(os.Getenv("LIMIT"))

// health endpoint, registered on the mux but NOT behind the limiter:
root := http.NewServeMux()
root.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
root.Handle("/", store.Middleware(appMux))

// graceful shutdown
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
srv := &http.Server{Addr: ":" + port, Handler: root}
go func() { _ = srv.ListenAndServe() }()
<-ctx.Done()
shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
srv.Shutdown(shutdownCtx)
Client IP behind a proxy: only trust X-Forwarded-For if the request comes from a proxy you control. Otherwise clients can spoof it.
Done when: Ctrl+C exits cleanly, /healthz returns 200 even when you're rate limited, and config comes from env vars.

---

Phase 7: Docker

FROM golang:1.23 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /rate-limiter ./server

FROM gcr.io/distroless/static
COPY --from=build /rate-limiter /rate-limiter
EXPOSE 8080
ENTRYPOINT ["/rate-limiter"]
Use your own Go version from go.mod.
docker build -t rate-limiter .
docker run -p 8080:8080 -e LIMIT=5 rate-limiter
Done when: the curl loop from before gives 200s then 429s against the container.

---

Phase 8: Kubernetes

# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata: { name: rate-limiter }
spec:
  replicas: 1            # keep at 1 until you add Redis (see Phase 9)
  selector: { matchLabels: { app: rate-limiter } }
  template:
    metadata: { labels: { app: rate-limiter } }
    spec:
      containers:
        - name: app
          image: rate-limiter:latest
          ports: [{ containerPort: 8080 }]
          env:
            - { name: LIMIT, value: "5" }
          readinessProbe: { httpGet: { path: /healthz, port: 8080 } }
          livenessProbe:  { httpGet: { path: /healthz, port: 8080 } }
---
apiVersion: v1
kind: Service
metadata: { name: rate-limiter }
spec:
  selector: { app: rate-limiter }
  ports: [{ port: 80, targetPort: 8080 }]
kind create cluster
kind load docker-image rate-limiter:latest
kubectl apply -f k8s/
kubectl port-forward svc/rate-limiter 8080:80
Done when: the pod is Ready and curl works through the port-forward.

---

Phase 9: scale out with Redis

Problem: with 3 replicas, counters aren't shared, so the effective limit is 3x.
Learn: external dependencies, an interface boundary, atomic operations.

type Counter interface {
        // returns the new count for key within the window
        Incr(ctx context.Context, key string, window time.Duration) (int, error)
}
// Implementations: memoryCounter (what you have) and redisCounter.
In Redis the usual recipe is INCR key, plus EXPIRE key window on the first increment. Do that in a Lua script or a transaction so the two steps are atomic. Look up the go-redis client.
Decide: if Redis is down, fail open (let requests through) or fail closed (block them)?
Done when: 3 replicas share one limit, verified with a script that fires requests at the Service.

---

Extras, anytime

- Benchmarks: func BenchmarkAllow(b *testing.B) and go test -bench .
- A real README.md (what it is, how to run, config table)
- GitHub Actions running go vet and go test -race
- Metrics (allowed and denied counters)

