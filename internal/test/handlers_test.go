package test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/sergioferg/gochat/internal/auth"
	"github.com/sergioferg/gochat/internal/database"
	"github.com/sergioferg/gochat/internal/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mockSecret = "test-jwt-secret-32-bytes-minimum!!"

// MockDBTX implements database.DBTX to intercept sqlc database queries in unit tests without PostgreSQL.
type MockDBTX struct {
	ExecFunc     func(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error)
	QueryFunc    func(ctx context.Context, query string, args ...any) (pgx.Rows, error)
	QueryRowFunc func(ctx context.Context, query string, args ...any) pgx.Row
}

func (m *MockDBTX) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if m.ExecFunc != nil {
		return m.ExecFunc(ctx, query, args...)
	}
	return pgconn.CommandTag{}, nil
}

func (m *MockDBTX) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, query, args...)
	}
	return &mockRows{}, nil
}

func (m *MockDBTX) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	if m.QueryRowFunc != nil {
		return m.QueryRowFunc(ctx, query, args...)
	}
	return &mockRow{}
}

type mockRow struct {
	scanFunc func(dest ...any) error
}

func (r *mockRow) Scan(dest ...any) error {
	if r.scanFunc != nil {
		return r.scanFunc(dest...)
	}
	return pgx.ErrNoRows
}

func newMockRow(vals ...any) pgx.Row {
	return &mockRow{
		scanFunc: func(dest ...any) error {
			for i, v := range vals {
				if i >= len(dest) {
					break
				}
				assignMockVal(dest[i], v)
			}
			return nil
		},
	}
}

type mockRows struct {
	rows [][]any
	idx  int
	err  error
}

func (r *mockRows) Close() {}
func (r *mockRows) Err() error { return r.err }
func (r *mockRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (r *mockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (r *mockRows) Next() bool {
	if r.idx < len(r.rows) {
		r.idx++
		return true
	}
	return false
}

func (r *mockRows) Scan(dest ...any) error {
	if r.idx <= 0 || r.idx > len(r.rows) {
		return pgx.ErrNoRows
	}
	currentRow := r.rows[r.idx-1]
	for i, val := range currentRow {
		if i >= len(dest) {
			break
		}
		assignMockVal(dest[i], val)
	}
	return nil
}

func assignMockVal(dest any, val any) {
	if dest == nil {
		return
	}
	switch d := dest.(type) {
	case *uuid.UUID:
		if val != nil {
			if u, ok := val.(uuid.UUID); ok {
				*d = u
			}
		}
	case **uuid.UUID:
		if val != nil {
			if u, ok := val.(*uuid.UUID); ok {
				*d = u
			} else if u, ok := val.(uuid.UUID); ok {
				*d = &u
			}
		} else {
			*d = nil
		}
	case *string:
		if val != nil {
			if s, ok := val.(string); ok {
				*d = s
			}
		}
	case **string:
		if val != nil {
			if s, ok := val.(*string); ok {
				*d = s
			} else if s, ok := val.(string); ok {
				*d = &s
			}
		} else {
			*d = nil
		}
	case *time.Time:
		if val != nil {
			if t, ok := val.(time.Time); ok {
				*d = t
			}
		}
	case **time.Time:
		if val != nil {
			if t, ok := val.(*time.Time); ok {
				*d = t
			} else if t, ok := val.(time.Time); ok {
				*d = &t
			}
		} else {
			*d = nil
		}
	case *pgtype.Timestamptz:
		if val != nil {
			if ts, ok := val.(pgtype.Timestamptz); ok {
				*d = ts
			} else if t, ok := val.(time.Time); ok {
				*d = pgtype.Timestamptz{Time: t, Valid: true}
			}
		} else {
			*d = pgtype.Timestamptz{Valid: false}
		}
	case *bool:
		if val != nil {
			if b, ok := val.(bool); ok {
				*d = b
			}
		}
	case *int64:
		if val != nil {
			if i, ok := val.(int64); ok {
				*d = i
			}
		}
	case *int:
		if val != nil {
			if i, ok := val.(int); ok {
				*d = i
			}
		}
	}
}

func (r *mockRows) Values() ([]any, error) { return nil, nil }
func (r *mockRows) RawValues() [][]byte    { return nil }
func (r *mockRows) Conn() *pgx.Conn        { return nil }

// TestHandlerEndpoint tests GET /api/healthz
func TestHandlerEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	rec := httptest.NewRecorder()

	handlers.HandlerEndpoint(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Equal(t, "OK", rec.Body.String())
}

// TestHandlerGetMessages tests GET /api/chats/{id}/messages
func TestHandlerGetMessages(t *testing.T) {
	userID := uuid.New()
	chatID := uuid.New()
	beforeID := uuid.New()
	now := time.Now().UTC()

	tests := []struct {
		name           string
		chatIDPath     string
		queryParams    string
		includeAuthCtx bool
		authUserID     uuid.UUID
		mockSetup      func(m *MockDBTX)
		wantStatus     int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:           "200 OK - fetch recent messages successfully",
			chatIDPath:     chatID.String(),
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryFunc = func(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
					if strings.Contains(query, "participants") {
						return &mockRows{rows: [][]any{{userID}}}, nil
					}
					if strings.Contains(query, "FROM messages") {
						msgID := uuid.New()
						return &mockRows{rows: [][]any{{msgID, "Hello world", userID, chatID, now, now}}}, nil
					}
					return &mockRows{}, nil
				}
			},
			wantStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var res struct {
					Messages []handlers.Message `json:"messages"`
				}
				err := json.Unmarshal([]byte(body), &res)
				require.NoError(t, err)
				assert.Len(t, res.Messages, 1)
				assert.Equal(t, "Hello world", res.Messages[0].Content)
				assert.Equal(t, userID, res.Messages[0].SenderID)
				assert.Equal(t, chatID, res.Messages[0].ChatID)
			},
		},
		{
			name:           "200 OK - fetch messages before specified before_id cursor",
			chatIDPath:     chatID.String(),
			queryParams:    "?before_id=" + beforeID.String(),
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryFunc = func(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
					if strings.Contains(query, "participants") {
						return &mockRows{rows: [][]any{{userID}}}, nil
					}
					if strings.Contains(query, "AND id < $2") {
						msgID := uuid.New()
						return &mockRows{rows: [][]any{{msgID, "Older message", userID, chatID, now, now}}}, nil
					}
					return &mockRows{}, nil
				}
			},
			wantStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var res struct {
					Messages []handlers.Message `json:"messages"`
				}
				err := json.Unmarshal([]byte(body), &res)
				require.NoError(t, err)
				assert.Len(t, res.Messages, 1)
				assert.Equal(t, "Older message", res.Messages[0].Content)
			},
		},
		{
			name:           "403 Forbidden - missing user authentication context",
			chatIDPath:     chatID.String(),
			includeAuthCtx: false,
			mockSetup:      func(m *MockDBTX) {},
			wantStatus:     http.StatusForbidden,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Not authorized")
			},
		},
		{
			name:           "400 Bad Request - invalid chat ID UUID format",
			chatIDPath:     "invalid-uuid-string",
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup:      func(m *MockDBTX) {},
			wantStatus:     http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Invalid chat ID format")
			},
		},
		{
			name:           "400 Bad Request - invalid before_id query parameter UUID format",
			chatIDPath:     chatID.String(),
			queryParams:    "?before_id=not-a-valid-uuid",
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryFunc = func(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
					if strings.Contains(query, "participants") {
						return &mockRows{rows: [][]any{{userID}}}, nil
					}
					return &mockRows{}, nil
				}
			},
			wantStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Invalid before_id format")
			},
		},
		{
			name:           "403 Forbidden - user is not a participant in the requested chat",
			chatIDPath:     chatID.String(),
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryFunc = func(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
					if strings.Contains(query, "participants") {
						otherUser := uuid.New()
						return &mockRows{rows: [][]any{{otherUser}}}, nil
					}
					return &mockRows{}, nil
				}
			},
			wantStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "You do not have access to this chat")
			},
		},
		{
			name:           "500 Internal Server Error - failure fetching participants from database",
			chatIDPath:     chatID.String(),
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryFunc = func(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
					return nil, errors.New("db connection failure")
				}
			},
			wantStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Failed to verify permissions")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDBTX{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockDB)
			}

			dbQueries := database.New(mockDB)
			api := &handlers.API{DB: dbQueries}

			mux := http.NewServeMux()
			mux.HandleFunc("GET /api/chats/{id}/messages", api.HandlerGetMessages)

			req := httptest.NewRequest(http.MethodGet, "/api/chats/"+tt.chatIDPath+"/messages"+tt.queryParams, nil)
			if tt.includeAuthCtx {
				ctx := context.WithValue(req.Context(), handlers.UserIDContextKey, tt.authUserID)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.String())
			}
		})
	}
}

// TestHandlerUpdateMessage tests PATCH /api/messages/{id}
func TestHandlerUpdateMessage(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	messageID := uuid.New()
	chatID := uuid.New()
	now := time.Now().UTC()

	tests := []struct {
		name           string
		messageIDPath  string
		body           string
		includeAuthCtx bool
		authUserID     uuid.UUID
		mockSetup      func(m *MockDBTX)
		wantStatus     int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:           "200 OK - message updated successfully by sender",
			messageIDPath:  messageID.String(),
			body:           `{"content":"Updated message text"}`,
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					if strings.Contains(query, "SELECT sender_id, chat_id") {
						return newMockRow(userID, chatID)
					}
					if strings.Contains(query, "UPDATE messages") {
						return newMockRow(messageID, "Updated message text", userID, chatID, now, now)
					}
					return &mockRow{}
				}
				m.QueryFunc = func(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
					return nil, errors.New("skip RMQ publish")
				}
			},
			wantStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var res struct {
					handlers.Message
				}
				err := json.Unmarshal([]byte(body), &res)
				require.NoError(t, err)
				assert.Equal(t, messageID, res.ID)
				assert.Equal(t, "Updated message text", res.Content)
				assert.Equal(t, userID, res.SenderID)
				assert.Equal(t, chatID, res.ChatID)
			},
		},
		{
			name:           "403 Forbidden - missing user authentication context",
			messageIDPath:  messageID.String(),
			body:           `{"content":"Updated text"}`,
			includeAuthCtx: false,
			mockSetup:      func(m *MockDBTX) {},
			wantStatus:     http.StatusForbidden,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Not authorized")
			},
		},
		{
			name:           "400 Bad Request - invalid JSON payload body",
			messageIDPath:  messageID.String(),
			body:           `{invalid json`,
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup:      func(m *MockDBTX) {},
			wantStatus:     http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Invalid JSON payload")
			},
		},
		{
			name:           "400 Bad Request - invalid message ID UUID format",
			messageIDPath:  "not-a-valid-uuid",
			body:           `{"content":"Updated text"}`,
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup:      func(m *MockDBTX) {},
			wantStatus:     http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Invalid message ID format")
			},
		},
		{
			name:           "404 Not Found - message does not exist in database",
			messageIDPath:  messageID.String(),
			body:           `{"content":"Updated text"}`,
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					return &mockRow{
						scanFunc: func(dest ...any) error {
							return pgx.ErrNoRows
						},
					}
				}
			},
			wantStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Message not found")
			},
		},
		{
			name:           "401 Unauthorized - user attempting to edit another user's message",
			messageIDPath:  messageID.String(),
			body:           `{"content":"Hijacked message"}`,
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					if strings.Contains(query, "SELECT sender_id, chat_id") {
						return newMockRow(otherUserID, chatID)
					}
					return &mockRow{}
				}
			},
			wantStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "You did not send this message")
			},
		},
		{
			name:           "500 Internal Server Error - database update failure",
			messageIDPath:  messageID.String(),
			body:           `{"content":"Updated content"}`,
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					if strings.Contains(query, "SELECT sender_id, chat_id") {
						return newMockRow(userID, chatID)
					}
					if strings.Contains(query, "UPDATE messages") {
						return &mockRow{
							scanFunc: func(dest ...any) error {
								return errors.New("db error updating message")
							},
						}
					}
					return &mockRow{}
				}
			},
			wantStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Couldn't edit message")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDBTX{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockDB)
			}

			dbQueries := database.New(mockDB)
			api := &handlers.API{DB: dbQueries}

			mux := http.NewServeMux()
			mux.HandleFunc("PATCH /api/messages/{id}", api.HandlerUpdateMessage)

			req := httptest.NewRequest(http.MethodPatch, "/api/messages/"+tt.messageIDPath, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.includeAuthCtx {
				ctx := context.WithValue(req.Context(), handlers.UserIDContextKey, tt.authUserID)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.String())
			}
		})
	}
}

// TestHandlerGetUserChats tests GET /api/chats
func TestHandlerGetUserChats(t *testing.T) {
	userID := uuid.New()
	chatID := uuid.New()
	msgID := uuid.New()
	now := time.Now().UTC()
	chatName := "General Chat"
	msgContent := "Hello team"

	tests := []struct {
		name           string
		includeAuthCtx bool
		authUserID     uuid.UUID
		mockSetup      func(m *MockDBTX)
		wantStatus     int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:           "200 OK - user chats retrieved successfully",
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryFunc = func(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
					return &mockRows{
						rows: [][]any{
							{chatID, &chatName, true, now, msgContent, msgID},
						},
					}, nil
				}
			},
			wantStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var res []struct {
					handlers.Chat
					LastReadAt         time.Time  `json:"last_read_at"`
					LastMessageContent *string    `json:"last_message_content"`
					LastMessageID      *uuid.UUID `json:"last_message_id"`
				}
				err := json.Unmarshal([]byte(body), &res)
				require.NoError(t, err)
				assert.Len(t, res, 1)
				assert.Equal(t, chatID, res[0].ID)
				assert.Equal(t, &chatName, res[0].Name)
				assert.True(t, res[0].IsGroup)
				assert.Equal(t, &msgContent, res[0].LastMessageContent)
				assert.Equal(t, &msgID, res[0].LastMessageID)
			},
		},
		{
			name:           "403 Forbidden - missing user context",
			includeAuthCtx: false,
			mockSetup:      func(m *MockDBTX) {},
			wantStatus:     http.StatusForbidden,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Not authorized")
			},
		},
		{
			name:           "500 Internal Server Error - database query failure",
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryFunc = func(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
					return nil, errors.New("db error")
				}
			},
			wantStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Failed to fetch chats")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDBTX{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockDB)
			}

			dbQueries := database.New(mockDB)
			api := &handlers.API{DB: dbQueries}

			mux := http.NewServeMux()
			mux.HandleFunc("GET /api/chats", api.HandlerGetUserChats)

			req := httptest.NewRequest(http.MethodGet, "/api/chats", nil)
			if tt.includeAuthCtx {
				ctx := context.WithValue(req.Context(), handlers.UserIDContextKey, tt.authUserID)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.String())
			}
		})
	}
}

// TestHandlerCreateChat tests POST /api/chats
func TestHandlerCreateChat(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	existingChatID := uuid.New()
	now := time.Now().UTC()
	groupName := "Dev Team"
	groupNamePtr := &groupName

	tests := []struct {
		name           string
		body           string
		includeAuthCtx bool
		authUserID     uuid.UUID
		mockSetup      func(m *MockDBTX)
		wantStatus     int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:           "201 Created - new group chat created successfully",
			body:           `{"name":"Dev Team","is_group":true,"participant_ids":["` + otherUserID.String() + `"]}`,
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					newChatID := uuid.New()
					return newMockRow(newChatID, groupNamePtr, true, now)
				}
				m.ExecFunc = func(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
					return pgconn.CommandTag{}, nil
				}
			},
			wantStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body string) {
				var res handlers.Chat
				err := json.Unmarshal([]byte(body), &res)
				require.NoError(t, err)
				assert.True(t, res.IsGroup)
				assert.Equal(t, &groupName, res.Name)
			},
		},
		{
			name:           "200 OK - 1-on-1 chat already exists",
			body:           `{"is_group":false,"participant_ids":["` + otherUserID.String() + `"]}`,
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					return newMockRow(existingChatID)
				}
			},
			wantStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var res map[string]any
				err := json.Unmarshal([]byte(body), &res)
				require.NoError(t, err)
				assert.Equal(t, existingChatID.String(), res["id"])
				assert.Equal(t, "Chat already exists", res["message"])
			},
		},
		{
			name:           "400 Bad Request - 1-on-1 chat without exactly 2 participants",
			body:           `{"is_group":false,"participant_ids":[]}`,
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup:      func(m *MockDBTX) {},
			wantStatus:     http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "1-on-1 chats must have exactly 2 participants")
			},
		},
		{
			name:           "400 Bad Request - invalid JSON payload body",
			body:           `{invalid json`,
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup:      func(m *MockDBTX) {},
			wantStatus:     http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Invalid JSON payload")
			},
		},
		{
			name:           "403 Forbidden - missing user context",
			body:           `{"is_group":true,"participant_ids":[]}`,
			includeAuthCtx: false,
			mockSetup:      func(m *MockDBTX) {},
			wantStatus:     http.StatusForbidden,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Not authorized")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDBTX{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockDB)
			}

			dbQueries := database.New(mockDB)
			api := &handlers.API{DB: dbQueries}

			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/chats", api.HandlerCreateChat)

			req := httptest.NewRequest(http.MethodPost, "/api/chats", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.includeAuthCtx {
				ctx := context.WithValue(req.Context(), handlers.UserIDContextKey, tt.authUserID)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.String())
			}
		})
	}
}

// TestHandlerSendMessage tests POST /api/messages
func TestHandlerSendMessage(t *testing.T) {
	userID := uuid.New()
	chatID := uuid.New()

	tests := []struct {
		name           string
		body           string
		includeAuthCtx bool
		authUserID     uuid.UUID
		mockSetup      func(m *MockDBTX)
		wantStatus     int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:           "403 Forbidden - missing user context",
			body:           `{"chat_id":"` + chatID.String() + `","message":"Test"}`,
			includeAuthCtx: false,
			mockSetup:      func(m *MockDBTX) {},
			wantStatus:     http.StatusForbidden,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Not authorized")
			},
		},
		{
			name:           "400 Bad Request - invalid JSON body",
			body:           `{invalid json`,
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup:      func(m *MockDBTX) {},
			wantStatus:     http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Couldn't decode parameters")
			},
		},
		{
			name:           "500 Internal Server Error - database error creating message",
			body:           `{"chat_id":"` + chatID.String() + `","message":"Error test"}`,
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					return &mockRow{
						scanFunc: func(dest ...any) error {
							return errors.New("db insert failure")
						},
					}
				}
			},
			wantStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Something went wrong")
			},
		},
		{
			name:           "500 Internal Server Error - database error fetching chat participants",
			body:           `{"chat_id":"` + chatID.String() + `","message":"Error test"}`,
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					msgID := uuid.New()
					now := time.Now().UTC()
					return newMockRow(msgID, "Error test", userID, chatID, now, now)
				}
				m.QueryFunc = func(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
					return nil, errors.New("failed fetching participants")
				}
			},
			wantStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Something went wrong")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDBTX{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockDB)
			}

			dbQueries := database.New(mockDB)
			api := &handlers.API{DB: dbQueries}

			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/messages", api.HandlerSendMessage)

			req := httptest.NewRequest(http.MethodPost, "/api/messages", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.includeAuthCtx {
				ctx := context.WithValue(req.Context(), handlers.UserIDContextKey, tt.authUserID)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.String())
			}
		})
	}
}

// TestHandlerGetSessions tests GET /api/sessions
func TestHandlerGetSessions(t *testing.T) {
	userID := uuid.New()
	sessionID := uuid.New()
	now := time.Now().UTC()

	tests := []struct {
		name           string
		includeAuthCtx bool
		authUserID     uuid.UUID
		mockSetup      func(m *MockDBTX)
		wantStatus     int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:           "200 OK - sessions fetched successfully",
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryFunc = func(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
					return &mockRows{
						rows: [][]any{
							{sessionID, now, "Mozilla/5.0", "127.0.0.1"},
						},
					}, nil
				}
			},
			wantStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var res struct {
					Sessions []struct {
						ID        uuid.UUID `json:"id"`
						CreatedAt time.Time `json:"created_at"`
						UserAgent string    `json:"user_agent"`
						IpAddress string    `json:"ip_address"`
					} `json:"sessions"`
				}
				err := json.Unmarshal([]byte(body), &res)
				require.NoError(t, err)
				assert.Len(t, res.Sessions, 1)
				assert.Equal(t, sessionID, res.Sessions[0].ID)
				assert.Equal(t, "Mozilla/5.0", res.Sessions[0].UserAgent)
				assert.Equal(t, "127.0.0.1", res.Sessions[0].IpAddress)
			},
		},
		{
			name:           "403 Forbidden - missing user context",
			includeAuthCtx: false,
			mockSetup:      func(m *MockDBTX) {},
			wantStatus:     http.StatusForbidden,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Not authorized")
			},
		},
		{
			name:           "500 Internal Server Error - database failure",
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryFunc = func(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
					return nil, errors.New("db error")
				}
			},
			wantStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Could not fetch sessions")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDBTX{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockDB)
			}

			dbQueries := database.New(mockDB)
			api := &handlers.API{DB: dbQueries}

			mux := http.NewServeMux()
			mux.HandleFunc("GET /api/sessions", api.HandlerGetSessions)

			req := httptest.NewRequest(http.MethodGet, "/api/sessions", nil)
			if tt.includeAuthCtx {
				ctx := context.WithValue(req.Context(), handlers.UserIDContextKey, tt.authUserID)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.String())
			}
		})
	}
}

// TestHandlerRevokeSession tests DELETE /api/sessions/{id}
func TestHandlerRevokeSession(t *testing.T) {
	userID := uuid.New()
	sessionID := uuid.New()

	tests := []struct {
		name           string
		sessionIDPath  string
		includeAuthCtx bool
		authUserID     uuid.UUID
		mockSetup      func(m *MockDBTX)
		wantStatus     int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:           "204 No Content - session revoked successfully",
			sessionIDPath:  sessionID.String(),
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.ExecFunc = func(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
					return pgconn.CommandTag{}, nil
				}
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:           "400 Bad Request - invalid session ID UUID format",
			sessionIDPath:  "invalid-session-uuid",
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup:      func(m *MockDBTX) {},
			wantStatus:     http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Invalid session ID format")
			},
		},
		{
			name:           "403 Forbidden - missing user context",
			sessionIDPath:  sessionID.String(),
			includeAuthCtx: false,
			mockSetup:      func(m *MockDBTX) {},
			wantStatus:     http.StatusForbidden,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Not authorized")
			},
		},
		{
			name:           "500 Internal Server Error - database error revoking session",
			sessionIDPath:  sessionID.String(),
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.ExecFunc = func(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
					return pgconn.CommandTag{}, errors.New("db error")
				}
			},
			wantStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Could not revoke session")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDBTX{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockDB)
			}

			dbQueries := database.New(mockDB)
			api := &handlers.API{DB: dbQueries}

			mux := http.NewServeMux()
			mux.HandleFunc("DELETE /api/sessions/{id}", api.HandlerRevokeSession)

			req := httptest.NewRequest(http.MethodDelete, "/api/sessions/"+tt.sessionIDPath, nil)
			if tt.includeAuthCtx {
				ctx := context.WithValue(req.Context(), handlers.UserIDContextKey, tt.authUserID)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.String())
			}
		})
	}
}

// TestHandlerUserCreate tests POST /api/users
func TestHandlerUserCreate(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()
	hashedPw, _ := auth.HashPassword("Password123!")
	hashedPwPtr := &hashedPw

	tests := []struct {
		name          string
		body          string
		mockSetup     func(m *MockDBTX)
		wantStatus    int
		checkResponse func(t *testing.T, body string)
	}{
		{
			name: "201 Created - user created successfully",
			body: `{"nickname":"alice","email":"alice@example.com","password":"Password123!"}`,
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					if strings.Contains(query, "INSERT INTO users") {
						return newMockRow(userID, "alice", "alice@example.com", hashedPwPtr, "unverified", now, now, pgtype.Timestamptz{Valid: false})
					}
					if strings.Contains(query, "INSERT INTO email_verification_tokens") {
						return newMockRow("hashedtoken", userID, now, now.Add(24*time.Hour))
					}
					return &mockRow{}
				}
			},
			wantStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body string) {
				var res handlers.User
				err := json.Unmarshal([]byte(body), &res)
				require.NoError(t, err)
				assert.Equal(t, userID, res.ID)
				assert.Equal(t, "alice", res.Nickname)
				assert.Equal(t, "alice@example.com", res.Email)
				assert.Equal(t, "unverified", res.Status)
			},
		},
		{
			name:       "400 Bad Request - invalid JSON body",
			body:       `{invalid json`,
			mockSetup:  func(m *MockDBTX) {},
			wantStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Couldn't decode parameters")
			},
		},
		{
			name: "409 Conflict - user email already exists",
			body: `{"nickname":"alice","email":"alice@example.com","password":"Password123!"}`,
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					return &mockRow{
						scanFunc: func(dest ...any) error {
							return &pgconn.PgError{Code: "23505"}
						},
					}
				}
			},
			wantStatus: http.StatusConflict,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "A user with this email already exists")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDBTX{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockDB)
			}

			dbQueries := database.New(mockDB)
			api := &handlers.API{
				DB:           dbQueries,
				BaseURL:      "http://localhost:8080",
				ResendApiKey: "test_resend_key",
			}

			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/users", api.HandlerUserCreate)

			req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.String())
			}
		})
	}
}

// TestHandlerUserLogin tests POST /api/login
func TestHandlerUserLogin(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()
	hashedPw, _ := auth.HashPassword("Password123!")
	hashedPwPtr := &hashedPw

	tests := []struct {
		name          string
		body          string
		mockSetup     func(m *MockDBTX)
		wantStatus    int
		checkResponse func(t *testing.T, body string)
	}{
		{
			name: "200 OK - user login successful",
			body: `{"email":"alice@example.com","password":"Password123!"}`,
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					if strings.Contains(query, "SELECT id, nickname, email") {
						return newMockRow(userID, "alice", "alice@example.com", hashedPwPtr, "active", now, now, pgtype.Timestamptz{Valid: false})
					}
					if strings.Contains(query, "INSERT INTO sessions") {
						sessID := uuid.New()
						return newMockRow(sessID, "hashedtoken", now, now, userID, "test-agent", "127.0.0.1", now.Add(24*time.Hour), pgtype.Timestamptz{Valid: false})
					}
					return &mockRow{}
				}
			},
			wantStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var res struct {
					handlers.User
					Token string `json:"token"`
				}
				err := json.Unmarshal([]byte(body), &res)
				require.NoError(t, err)
				assert.Equal(t, userID, res.ID)
				assert.NotEmpty(t, res.Token)
			},
		},
		{
			name:       "400 Bad Request - invalid JSON body",
			body:       `{invalid json`,
			mockSetup:  func(m *MockDBTX) {},
			wantStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Couldn't decode parameters")
			},
		},
		{
			name: "401 Unauthorized - user not found in database",
			body: `{"email":"nonexistent@example.com","password":"Password123!"}`,
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					return &mockRow{
						scanFunc: func(dest ...any) error {
							return pgx.ErrNoRows
						},
					}
				}
			},
			wantStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Incorrect email or password")
			},
		},
		{
			name: "401 Unauthorized - account unverified",
			body: `{"email":"unverified@example.com","password":"Password123!"}`,
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					return newMockRow(userID, "unverified_user", "unverified@example.com", hashedPwPtr, "unverified", now, now, pgtype.Timestamptz{Valid: false})
				}
			},
			wantStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Account not verified")
			},
		},
		{
			name: "401 Unauthorized - incorrect password",
			body: `{"email":"alice@example.com","password":"WrongPassword!"}`,
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					return newMockRow(userID, "alice", "alice@example.com", hashedPwPtr, "active", now, now, pgtype.Timestamptz{Valid: false})
				}
			},
			wantStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Incorrect email or password")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDBTX{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockDB)
			}

			dbQueries := database.New(mockDB)
			api := &handlers.API{
				DB:     dbQueries,
				Secret: mockSecret,
			}

			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/login", api.HandlerUserLogin)

			req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.String())
			}
		})
	}
}

// TestHandlerUserVerify tests POST /api/verify
func TestHandlerUserVerify(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()
	hashedPw, _ := auth.HashPassword("Password123!")
	hashedPwPtr := &hashedPw

	tests := []struct {
		name          string
		body          string
		mockSetup     func(m *MockDBTX)
		wantStatus    int
		checkResponse func(t *testing.T, body string)
	}{
		{
			name: "200 OK - user email verified successfully",
			body: `{"token":"valid-token-string"}`,
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					return newMockRow(userID, "alice", "alice@example.com", hashedPwPtr, "unverified", now, now, pgtype.Timestamptz{Valid: false})
				}
				m.ExecFunc = func(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
					return pgconn.CommandTag{}, nil
				}
			},
			wantStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Account successfully verified")
			},
		},
		{
			name:       "400 Bad Request - missing token in request body",
			body:       `{"token":""}`,
			mockSetup:  func(m *MockDBTX) {},
			wantStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Missing verification token")
			},
		},
		{
			name: "409 Conflict - invalid or expired verification token",
			body: `{"token":"expired-token"}`,
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					return &mockRow{
						scanFunc: func(dest ...any) error {
							return sql.ErrNoRows
						},
					}
				}
			},
			wantStatus: http.StatusConflict,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Token is invalid or expired")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDBTX{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockDB)
			}

			dbQueries := database.New(mockDB)
			api := &handlers.API{DB: dbQueries}

			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/verify", api.HandlerUserVerify)

			req := httptest.NewRequest(http.MethodPost, "/api/verify", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.String())
			}
		})
	}
}

// TestHandlerUserLogout tests POST /api/logout
func TestHandlerUserLogout(t *testing.T) {
	tests := []struct {
		name          string
		cookie        *http.Cookie
		mockSetup     func(m *MockDBTX)
		wantStatus    int
		checkResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:   "204 No Content - logout with valid refresh token cookie",
			cookie: &http.Cookie{Name: "refresh_token", Value: "valid_refresh_token_value"},
			mockSetup: func(m *MockDBTX) {
				m.ExecFunc = func(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
					return pgconn.CommandTag{}, nil
				}
			},
			wantStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				cookies := rec.Result().Cookies()
				var refreshCookie *http.Cookie
				for _, c := range cookies {
					if c.Name == "refresh_token" {
						refreshCookie = c
						break
					}
				}
				require.NotNil(t, refreshCookie)
				assert.Equal(t, "", refreshCookie.Value)
				assert.Equal(t, -1, refreshCookie.MaxAge)
			},
		},
		{
			name:       "204 No Content - logout without refresh token cookie",
			cookie:     nil,
			mockSetup:  func(m *MockDBTX) {},
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDBTX{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockDB)
			}

			dbQueries := database.New(mockDB)
			api := &handlers.API{DB: dbQueries}

			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/logout", api.HandlerUserLogout)

			req := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec)
			}
		})
	}
}

// TestHandlerRefreshAccessToken tests POST /api/refresh
func TestHandlerRefreshAccessToken(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()
	hashedPw, _ := auth.HashPassword("Password123!")
	hashedPwPtr := &hashedPw

	tests := []struct {
		name          string
		cookie        *http.Cookie
		mockSetup     func(m *MockDBTX)
		wantStatus    int
		checkResponse func(t *testing.T, body string)
	}{
		{
			name:   "200 OK - access token refreshed successfully using cookie",
			cookie: &http.Cookie{Name: "refresh_token", Value: "valid_refresh_token_cookie"},
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					return newMockRow(userID, "alice", "alice@example.com", hashedPwPtr, "active", now, now, pgtype.Timestamptz{Valid: false})
				}
			},
			wantStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var res struct {
					Token string `json:"token"`
				}
				err := json.Unmarshal([]byte(body), &res)
				require.NoError(t, err)
				assert.NotEmpty(t, res.Token)
			},
		},
		{
			name:       "401 Unauthorized - missing refresh token cookie",
			cookie:     nil,
			mockSetup:  func(m *MockDBTX) {},
			wantStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Missing refresh token cookie")
			},
		},
		{
			name:   "401 Unauthorized - expired or invalid refresh token",
			cookie: &http.Cookie{Name: "refresh_token", Value: "expired_token"},
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					return &mockRow{
						scanFunc: func(dest ...any) error {
							return pgx.ErrNoRows
						},
					}
				}
			},
			wantStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Invalid/Expired token")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDBTX{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockDB)
			}

			dbQueries := database.New(mockDB)
			api := &handlers.API{
				DB:     dbQueries,
				Secret: mockSecret,
			}

			mux := http.NewServeMux()
			mux.HandleFunc("POST /api/refresh", api.HandlerRefreshAccessToken)

			req := httptest.NewRequest(http.MethodPost, "/api/refresh", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.String())
			}
		})
	}
}

// TestHandlerUserUpdate tests PATCH /api/users
func TestHandlerUserUpdate(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()
	hashedPw, _ := auth.HashPassword("Password123!")
	hashedPwPtr := &hashedPw

	tests := []struct {
		name           string
		body           string
		includeAuthCtx bool
		authUserID     uuid.UUID
		mockSetup      func(m *MockDBTX)
		wantStatus     int
		checkResponse  func(t *testing.T, body string)
	}{
		{
			name:           "200 OK - user nickname updated successfully",
			body:           `{"nickname":"alice_new"}`,
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					return newMockRow(userID, "alice_new", "alice@example.com", hashedPwPtr, "active", now, now, pgtype.Timestamptz{Valid: false})
				}
			},
			wantStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body string) {
				var res handlers.User
				err := json.Unmarshal([]byte(body), &res)
				require.NoError(t, err)
				assert.Equal(t, "alice_new", res.Nickname)
			},
		},
		{
			name:           "400 Bad Request - no fields provided in body to update",
			body:           `{}`,
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup:      func(m *MockDBTX) {},
			wantStatus:     http.StatusBadRequest,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "No fields provided to update")
			},
		},
		{
			name:           "403 Forbidden - missing user context",
			body:           `{"nickname":"alice_new"}`,
			includeAuthCtx: false,
			mockSetup:      func(m *MockDBTX) {},
			wantStatus:     http.StatusForbidden,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "Not authorized")
			},
		},
		{
			name:           "409 Conflict - updated email already exists",
			body:           `{"email":"existing@example.com"}`,
			includeAuthCtx: true,
			authUserID:     userID,
			mockSetup: func(m *MockDBTX) {
				m.QueryRowFunc = func(ctx context.Context, query string, args ...any) pgx.Row {
					return &mockRow{
						scanFunc: func(dest ...any) error {
							return &pgconn.PgError{Code: "23505"}
						},
					}
				}
			},
			wantStatus: http.StatusConflict,
			checkResponse: func(t *testing.T, body string) {
				assert.Contains(t, body, "A user with this email already exists")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := &MockDBTX{}
			if tt.mockSetup != nil {
				tt.mockSetup(mockDB)
			}

			dbQueries := database.New(mockDB)
			api := &handlers.API{DB: dbQueries}

			mux := http.NewServeMux()
			mux.HandleFunc("PATCH /api/users", api.HandlerUserUpdate)

			req := httptest.NewRequest(http.MethodPatch, "/api/users", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			if tt.includeAuthCtx {
				ctx := context.WithValue(req.Context(), handlers.UserIDContextKey, tt.authUserID)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, rec.Body.String())
			}
		})
	}
}
