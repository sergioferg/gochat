# GoChat API Documentation

Welcome to the **GoChat API** reference guide. This document outlines the available RESTful HTTP endpoints, authentication mechanism, and WebSocket protocol used for real-time messaging.

For the full machine-readable OpenAPI specification, see [openapi.yaml](file:///home/sergiog/Projects/go/src/gochat/docs/openapi.yaml).

---

## Overview & Base URLs

- **Local Development Base URL:** `http://localhost:8080`
- **Production Base URL:** `https://<your-azure-domain>.azurewebsites.net` (Azure App Service)

---

## Authentication & Security

GoChat uses **JWT (JSON Web Tokens)** for authenticating API requests.

1. **Access Token:** Short-lived JWT passed via the `Authorization` header on protected endpoints:
   ```http
   Authorization: Bearer <your_access_token>
   ```
2. **Refresh Token:** Sent as an HTTP-only cookie or in body payload for `/refresh` to obtain new access tokens.
3. **GitHub OAuth:** Flow starts at `/oauth/github/login` and redirects back to `/oauth/github/callback`.

---

## REST API Endpoints Summary

### System & Health

| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :---: |
| `GET` | `/healthz` | Health check endpoint returning `OK` status | No |

---

### Authentication & User Management

| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :---: |
| `POST` | `/users` | Register a new user account | No |
| `POST` | `/verify` | Verify user email address using verification token | No |
| `POST` | `/login` | Authenticate user with Email and Password | No |
| `POST` | `/logout` | Terminate user session and revoke refresh token | Yes |
| `POST` | `/refresh` | Obtain a new access token using a valid refresh token | No |
| `GET` | `/me` | Retrieve authenticated user profile information | Yes |
| `GET` | `/users/search` | Search for users by nickname query | Yes |
| `PATCH` | `/users` | Update user profile details (nickname, email, password) | Yes |
| `DELETE` | `/users` | Delete authenticated user account | Yes |

---

### OAuth (GitHub)

| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :---: |
| `GET` | `/oauth/github/login` | Redirect to GitHub for OAuth authentication | No |
| `GET` | `/oauth/github/callback` | Callback URL handled after GitHub OAuth approval | No |

---

### Active Sessions Management

| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :---: |
| `GET` | `/sessions` | List all active login sessions for the authenticated user | Yes |
| `DELETE` | `/sessions/{id}` | Revoke a specific active session by ID | Yes |

---

### User Relationships & Requests

| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :---: |
| `GET` | `/requests` | Retrieve pending incoming chat requests for authenticated user | Yes |
| `POST` | `/requests` | Send a new chat request to another user | Yes |
| `PATCH` | `/requests/{id}` | Respond to a chat request (`accept`, `reject`, or `block`) | Yes |

---

### Chat Rooms & Messaging

| Method | Endpoint | Description | Auth Required |
| :--- | :--- | :--- | :---: |
| `GET` | `/chats` | Get all chat rooms the authenticated user is part of | Yes |
| `POST` | `/chats` | Create a new chat room | Yes |
| `GET` | `/chats/{id}/messages` | Retrieve message history for a specific chat room | Yes |
| `POST` | `/messages` | Send a new text message to a chat room | Yes |
| `PATCH` | `/messages/{id}` | Edit an existing message by message ID | Yes |

---

## WebSocket API (Real-Time Communication)

- **Endpoint:** `GET /ws`
- **Protocol:** `ws://` (Local) or `wss://` (Production)
- **Header:** Requires `Authorization: Bearer <access_token>` during WebSocket handshake.

### Real-Time Event Payload Example

When a new message is published via RabbitMQ, it is broadcast to connected WebSocket clients as a JSON event payload:

```json
{
  "type": "new_message",
  "chat_id": "123e4567-e89b-12d3-a456-426614174000",
  "message": {
    "id": "987f6543-e21b-12d3-a456-426614174000",
    "sender_id": "usr_112233",
    "content": "Hello world!",
    "created_at": "2026-07-26T16:00:00Z"
  },
  "target_user_ids": ["usr_112233", "usr_445566"]
}
```

---

## OpenAPI Specification

Detailed schema definitions, response codes, and examples can be found in [openapi.yaml](file:///home/sergiog/Projects/go/src/gochat/docs/openapi.yaml).
