Step 0: Install Go and check it works
go version
If that fails, install Go from go.dev/dl or with brew install go.

Step 1: Create a module
cd /Users/samuel/Desktop/rate-limiter
go mod init github.com/devallisamuel/Rate-limiter
This creates go.mod, which names your project and tracks its dependencies. Read the file afterward.

Step 2: Start with the algorithm, not HTTP. Build a plain Go package that answers "is this request allowed?" before you add a server. Try them in this order, since each one adds a concept:

┌────────────────┬─────────────────────────────────────────────────────────────┬─────────────────────────────────────────┐
│   Algorithm    │                        How it works                         │        Go concepts you'll learn         │
├────────────────┼─────────────────────────────────────────────────────────────┼─────────────────────────────────────────┤
│ Fixed window   │ Count requests per time window, then reset                  │ structs, methods, time package, maps    │
├────────────────┼─────────────────────────────────────────────────────────────┼─────────────────────────────────────────┤
│ Token bucket   │ Tokens refill at a steady rate, and each request spends one │ time.Duration, float math, constructors │
├────────────────┼─────────────────────────────────────────────────────────────┼─────────────────────────────────────────┤
│ Sliding window │ Track timestamps of recent requests                         │ slices, trimming old entries            │
└────────────────┴─────────────────────────────────────────────────────────────┴─────────────────────────────────────────┘

Step 3: Make it safe for concurrent use. A web server handles many requests at once, so your limiter will be hit by multiple goroutines. This is where Go gets interesting:
- sync.Mutex: why you need it and what a data race is
- Run go test -race ./... to catch races

Step 4: Support many clients. Keep one limiter per client key, such as an IP address. This brings in maps of structs, and the question of when to clean up old entries.

Step 5: Wrap it in HTTP middleware. Use the standard library's net/http and write a function that takes a http.Handler and returns one. Reject over-limit requests with 429 Too Many Requests and a Retry-After header.

Step 6: Tests throughout. Write _test.go files with table-driven tests, which is the idiomatic Go style. Try to make time injectable so your tests don't need time.Sleep.

Suggested project layout

rate-limiter/
  go.mod
  limiter/          # the algorithms (package limiter)
    tokenbucket.go
    tokenbucket_test.go
  cmd/server/       # a main package that wires up HTTP
    main.go

Things to read alongside

- A Tour of Go (go.dev/tour): do the basics, methods, and concurrency sections first.
- Effective Go (go.dev/doc/effective_go): for idioms.
- golang.org/x/time/rate: the official token bucket. Don't use it, but read its source after you build your own and compare.

Where to begin

Do steps 0 and 1, then try to write a fixed-window limiter with this shape:

- A struct holding the limit, the window size, a counter, and the window start time
- A constructor, New(...)
- A method, Allow() bool
