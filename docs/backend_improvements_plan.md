# Backend Architecture & Improvements Plan

## Current Codebase Structure

The `gochat` project is a full-stack application composed of a Go backend and a React/Vite frontend.

### Backend (Go)
The backend currently follows idiomatic Go folder structures:
- **`main.go`**: The entry point that wires all dependencies together (Database, RabbitMQ, WebSockets), sets up HTTP multiplexers (using the standard library `ServeMux`), and starts the server.
- **`internal/`**: Contains private application logic.
  - **`handlers/`**: Contains HTTP handler functions grouped by domain (e.g., `handlers_chats.go`, `handlers_users.go`). All handlers are methods on a central `API` struct (`config.go`) which holds all dependencies (DB, RabbitMQ, etc.).
  - **`middleware/`**: Contains custom HTTP middlewares (e.g., Auth, CORS) and uses `justinas/alice` for middleware chaining.
  - **`database/`**: Generated `sqlc` code for database interactions.
  - **`pubsub/` & `routing/`**: RabbitMQ logic for pub/sub messaging.
  - **`ws/`**: WebSocket manager to handle real-time connections.
- **`sql/`**: Contains raw SQL files (`schema/` and `queries/`) used by `sqlc` to generate the Go database code.
- **`Dockerfile`**: A highly optimized multi-stage build using `distroless` for the final image.

---

## Architectural Improvements

To make the backend more scalable, testable, and maintainable as the project grows, we propose the following three architectural improvements:

### 1. Routing Organization
Currently, `main.go` registers every single route (over 15 routes) on one `ServeMux` and applies middleware chains manually. 

**What is it?**
Extracting route definitions into dedicated functions or packages grouped by domain (e.g., `users`, `chats`, `auth`).

**Why improve it?**
It prevents `main.go` from becoming a massive file and makes it clear which routes belong to which domain.

**How it looks:**

*Current `main.go`:*
```go
mux.Handle("GET /chats", fullAccountChain.ThenFunc(api.HandlerGetUserChats))
mux.Handle("POST /chats", fullAccountChain.ThenFunc(api.HandlerCreateChat))
// ... 15 more lines
```

*Refactored:*
We create helper methods on the `API` struct to group these.
```go
// internal/handlers/routes.go
func (api *API) SetupRoutes(mux *http.ServeMux, authChain, fullAccountChain alice.Chain) {
    api.setupChatRoutes(mux, fullAccountChain)
    api.setupUserRoutes(mux, authChain, fullAccountChain)
    // ...
}

func (api *API) setupChatRoutes(mux *http.ServeMux, chain alice.Chain) {
    mux.Handle("GET /chats", chain.ThenFunc(api.HandlerGetUserChats))
    mux.Handle("POST /chats", chain.ThenFunc(api.HandlerCreateChat))
    mux.Handle("GET /chats/{id}/messages", chain.ThenFunc(api.HandlerGetMessages))
}
```
*New `main.go`:*
```go
api.SetupRoutes(mux, authChain, fullAccountChain)
```

---

### 2. Centralized Error Handling
Currently, handlers use `respond.WithError(w, http.StatusInternalServerError, "Failed to create chat", err)`. If the `err` is a database connection issue, returning 500 is fine, but if it's a "User Not Found" error from `sqlc`, we should return a 404.

**What is it?**
Defining custom application error types (e.g., `AppError`) that map to HTTP status codes. The `respond.WithError` function will inspect the error type and automatically determine the correct HTTP status code and user-facing message.

**Why improve it?**
It standardizes API error responses and ensures sensitive database errors aren't accidentally leaked to the client.

**How it looks:**

*internal/errors/errors.go*
```go
var (
    ErrNotFound     = errors.New("resource not found")
    ErrUnauthorized = errors.New("unauthorized access")
    ErrConflict     = errors.New("resource already exists")
)
```

*internal/respond/respond.go*
```go
// The new unified error responder
func WithError(w http.ResponseWriter, err error) {
    status := http.StatusInternalServerError
    message := "Internal Server Error"

    // Map domain errors to HTTP codes automatically
    switch {
    case errors.Is(err, apperrors.ErrNotFound):
        status = http.StatusNotFound
        message = err.Error()
    case errors.Is(err, apperrors.ErrUnauthorized):
        status = http.StatusUnauthorized
        message = err.Error()
    case errors.Is(err, apperrors.ErrConflict):
        status = http.StatusConflict
        message = err.Error()
    }

    if status == http.StatusInternalServerError {
        logrus.Errorf("internal error: %v", err)
    }

    WithJSON(w, status, map[string]string{"error": message})
}
```

---

### 3. The Service Layer
Currently, HTTP handlers in `internal/handlers/` do everything: parsing JSON, executing business logic (like checking if a user is already in a chat), and talking to the database. 

**What is it?**
A Service Layer sits between your HTTP handlers and your database. 
- **Handlers** only care about HTTP (parsing requests, returning JSON, reading headers).
- **Services** only care about business logic (validation, coordinating multiple database calls).

**Why improve it?**
It makes testing incredibly easy because you can test business logic without spinning up an HTTP server. It also allows different handlers to reuse the same logic.

**How it looks:**

*Current Handler (Bloated):*
```go
func (api *API) HandlerCreateChat(w http.ResponseWriter, r *http.Request) {
    // 1. Parse HTTP Request
    var params parameters
    json.NewDecoder(r.Body).Decode(&params)

    // 2. Business Logic
    if !params.IsGroup && len(params.ParticipantIDs) != 2 {
        respond.WithError(w, http.StatusBadRequest, "1-on-1 chats must have exactly 2 participants", nil)
        return
    }

    // 3. Database Calls
    chat, err := api.DB.CreateChat(r.Context(), ...)
    // ...
}
```

*Refactored (Clean):*
```go
// internal/services/chat_service.go
type ChatService struct {
    DB *database.Queries
}

func (s *ChatService) CreateChat(ctx context.Context, userID uuid.UUID, params CreateChatDTO) (Chat, error) {
    if !params.IsGroup && len(params.ParticipantIDs) != 2 {
        return Chat{}, ErrInvalidParticipants
    }
    // ... DB logic
    return chat, nil
}

// internal/handlers/handlers_chats.go
func (api *API) HandlerCreateChat(w http.ResponseWriter, r *http.Request) {
    var params CreateChatDTO
    json.NewDecoder(r.Body).Decode(&params)

    chat, err := api.ChatService.CreateChat(r.Context(), userID, params)
    if err != nil {
        respond.WithError(w, err) 
        return
    }

    respond.WithJSON(w, http.StatusCreated, chat)
}
```
