package routes

import (
	"net/http"

	"github.com/justinas/alice"
	"github.com/sergioferg/gochat/internal/handlers"
)

func SetupAuthRoutes(mux *http.ServeMux, api *handlers.API, authChain, fullAccountChain alice.Chain) {
	mux.HandleFunc("POST /refresh", api.HandlerRefreshAccessToken)
	mux.HandleFunc("POST /login", api.HandlerUserLogin)
	mux.HandleFunc("POST /logout", api.HandlerUserLogout)
	mux.HandleFunc("GET /oauth/github/login", api.HandlerGitHubLogin)
	mux.HandleFunc("GET /oauth/github/callback", api.HandlerGitHubCallback)
	mux.Handle("GET /sessions", authChain.ThenFunc(api.HandlerGetSessions))
	mux.Handle("DELETE /sessions/{id}", authChain.ThenFunc(api.HandlerRevokeSession))
}

func SetupUserRoutes(mux *http.ServeMux, api *handlers.API, authChain, fullAccountChain alice.Chain) {
	mux.HandleFunc("POST /users", api.HandlerUserCreate)
	mux.HandleFunc("POST /verify", api.HandlerUserVerify)
	mux.Handle("DELETE /users", authChain.ThenFunc(api.HandlerUserDelete))
	mux.Handle("PATCH /users", authChain.ThenFunc(api.HandlerUserUpdate))
	mux.Handle("GET /users/search", fullAccountChain.ThenFunc(api.HandlerUsersSearch))
	mux.Handle("GET /users/{id}", fullAccountChain.ThenFunc(api.HandlerGetUserProfile))
	mux.Handle("GET /me", authChain.ThenFunc(api.HandlerGetMe))
}

func SetupChatRoutes(mux *http.ServeMux, api *handlers.API, authChain, fullAccountChain alice.Chain) {
	mux.Handle("GET /chats", fullAccountChain.ThenFunc(api.HandlerGetUserChats))
	mux.Handle("POST /chats", fullAccountChain.ThenFunc(api.HandlerCreateChat))
	mux.Handle("GET /chats/{id}/messages", fullAccountChain.ThenFunc(api.HandlerGetMessages))
	mux.Handle("POST /messages", fullAccountChain.ThenFunc(api.HandlerSendMessage))
	mux.Handle("PATCH /messages/{id}", fullAccountChain.ThenFunc(api.HandlerUpdateMessage))
}

func SetupRequestRoutes(mux *http.ServeMux, api *handlers.API, authChain, fullAccountChain alice.Chain) {
	mux.Handle("GET /requests", fullAccountChain.ThenFunc(api.HandlerGetRequests))
	mux.Handle("POST /requests", fullAccountChain.ThenFunc(api.HandlerCreateRequest))
	mux.Handle("PATCH /requests/{id}", fullAccountChain.ThenFunc(api.HandlerUpdateRequest))

	// Direct Relationship & Block Routes
	mux.Handle("DELETE /friends/{id}", fullAccountChain.ThenFunc(api.HandlerUnfriendUser))
	mux.Handle("GET /blocks", fullAccountChain.ThenFunc(api.HandlerGetBlockedUsers))
	mux.Handle("POST /blocks", fullAccountChain.ThenFunc(api.HandlerBlockUser))
	mux.Handle("DELETE /blocks/{id}", fullAccountChain.ThenFunc(api.HandlerUnblockUser))
}

func SetupWebSocketRoutes(mux *http.ServeMux, api *handlers.API, authChain, fullAccountChain alice.Chain) {
	mux.Handle("GET /ws", fullAccountChain.ThenFunc(api.HandlerWebSocket))
}
