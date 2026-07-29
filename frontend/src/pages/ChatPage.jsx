import { useState, useEffect, useRef } from "react";
import {
  sendMessage,
  getWebSocketUrl,
  searchUsers,
  sendRequest,
  fetchPendingRequests,
  updateRequestAction,
  fetchUserChats,
} from "../api";

export default function ChatPage({ currentUser }) {
  const [message, setMessage] = useState("");
  const [sending, setSending] = useState(false);
  const [wsStatus, setWsStatus] = useState("disconnected");
  const [chatLog, setChatLog] = useState([
    { id: "sys-1", sender: "system", text: "Welcome to GoChat." },
  ]);

  // Search & Request states
  const [searchQuery, setSearchQuery] = useState("");
  const [searchResults, setSearchResults] = useState([]);
  const [searching, setSearching] = useState(false);
  const [searchError, setSearchError] = useState("");
  const [requestStatus, setRequestStatus] = useState({});
  const [requestNotice, setRequestNotice] = useState(null);

  // Pending Requests state
  const [pendingRequests, setPendingRequests] = useState([]);
  const [loadingRequests, setLoadingRequests] = useState(false);
  const [actionInProgress, setActionInProgress] = useState({});

  // Active Chats state
  const [myChats, setMyChats] = useState([]);
  const [loadingChats, setLoadingChats] = useState(false);
  const [activeChatId, setActiveChatId] = useState("general");

  const wsRef = useRef(null);

  // Load Pending Requests
  const loadRequests = async () => {
    setLoadingRequests(true);
    try {
      const reqs = await fetchPendingRequests();
      setPendingRequests(reqs);
    } catch (err) {
      console.error("Failed to load requests:", err);
    } finally {
      setLoadingRequests(false);
    }
  };

  // Load User Chats
  const loadChats = async (selectChatId = null) => {
    setLoadingChats(true);
    try {
      const chats = await fetchUserChats();
      setMyChats(chats);
      if (selectChatId) {
        setActiveChatId(selectChatId);
      }
    } catch (err) {
      console.error("Failed to load chats:", err);
    } finally {
      setLoadingChats(false);
    }
  };

  useEffect(() => {
    loadRequests();
    loadChats();
  }, []);

  // Debounced user search
  useEffect(() => {
    const trimmed = searchQuery.trim();
    if (!trimmed) {
      setSearchResults([]);
      setSearching(false);
      setSearchError("");
      return;
    }

    setSearching(true);
    setSearchError("");

    const timer = setTimeout(async () => {
      try {
        const users = await searchUsers(trimmed);
        setSearchResults(users);
      } catch (err) {
        console.error("Error searching users:", err);
        setSearchError(err.message || "Failed to search users");
        setSearchResults([]);
      } finally {
        setSearching(false);
      }
    }, 350);

    return () => clearTimeout(timer);
  }, [searchQuery]);

  const handleSendRequest = async (user) => {
    if (requestStatus[user.id] === "sending" || requestStatus[user.id] === "sent") return;

    setRequestStatus((prev) => ({ ...prev, [user.id]: "sending" }));
    setRequestNotice(null);

    try {
      await sendRequest(user.id);
      setRequestStatus((prev) => ({ ...prev, [user.id]: "sent" }));
      setRequestNotice({
        type: "success",
        text: `Request sent to ${user.nickname || user.real_name || "user"}`,
      });
    } catch (err) {
      const msg = err.message || "";
      if (
        msg.includes("already exists") ||
        msg.includes("Conflict") ||
        msg.includes("409")
      ) {
        setRequestNotice({
          type: "error",
          text: "Request already sent or relationship exists",
        });
        setRequestStatus((prev) => ({ ...prev, [user.id]: "sent" }));
      } else {
        setRequestNotice({
          type: "error",
          text: msg || "Failed to send request",
        });
        setRequestStatus((prev) => ({ ...prev, [user.id]: "idle" }));
      }
    }
  };

  const handleRequestAction = async (requestId, action) => {
    setActionInProgress((prev) => ({ ...prev, [requestId]: action }));
    try {
      const res = await updateRequestAction(requestId, action);
      // Remove request from UI list
      setPendingRequests((prev) => prev.filter((r) => r.id !== requestId));

      if (action === "accept") {
        // Refetch chats and select newly created chat
        const newChatId = res?.chat_id;
        await loadChats(newChatId);
      }
    } catch (err) {
      console.error(`Failed to ${action} request:`, err);
      alert(err.message || `Failed to ${action} request`);
    } finally {
      setActionInProgress((prev) => ({ ...prev, [requestId]: null }));
    }
  };

  useEffect(() => {
    let socket = null;
    let reconnectTimer = null;
    let isMounted = true;

    function connect() {
      if (!isMounted) return;
      setWsStatus("connecting");

      const wsUrl = getWebSocketUrl("/ws");
      socket = new WebSocket(wsUrl);
      wsRef.current = socket;

      socket.onopen = () => {
        if (isMounted) {
          setWsStatus("connected");
        }
      };

      socket.onmessage = (event) => {
        if (!isMounted) return;
        try {
          const payload = JSON.parse(event.data);
          if (payload.type === "new_message") {
            const isMe = payload.sender_id === currentUser?.id;
            setChatLog((prevLog) => {
              const isDuplicate = prevLog.some(
                (msg) =>
                  (msg.id && msg.id === payload.message_id) ||
                  (msg.sender === "me" && isMe && msg.text === payload.content && msg.pending)
              );
              if (isDuplicate) {
                return prevLog.map((msg) =>
                  msg.sender === "me" && isMe && msg.text === payload.content && msg.pending
                    ? { id: payload.message_id, sender: "me", text: payload.content }
                    : msg
                );
              }
              return [
                ...prevLog,
                {
                  id: payload.message_id || `msg-${Date.now()}-${Math.random()}`,
                  sender: isMe ? "me" : "them",
                  text: payload.content,
                },
              ];
            });
          }
        } catch (err) {
          console.error("Error parsing WebSocket message:", err);
        }
      };

      socket.onclose = () => {
        if (isMounted) {
          setWsStatus("disconnected");
          reconnectTimer = setTimeout(() => {
            if (isMounted) {
              setWsStatus("reconnecting");
              connect();
            }
          }, 3000);
        }
      };

      socket.onerror = (err) => {
        console.warn("WebSocket error encountered:", err);
      };
    }

    connect();

    return () => {
      isMounted = false;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      if (socket) {
        socket.close();
      }
    };
  }, [currentUser?.id]);

  const handleSend = async (e) => {
    e.preventDefault();
    if (!message.trim() || sending) return;

    const userMessage = message.trim();
    const tempId = `temp-${Date.now()}`;

    setChatLog((prevLog) => [
      ...prevLog,
      { id: tempId, sender: "me", text: userMessage, pending: true },
    ]);
    setMessage("");
    setSending(true);

    try {
      await sendMessage(userMessage);
    } catch (error) {
      console.error("Failed to send message:", error);
      setChatLog((prevLog) => [
        ...prevLog.filter((m) => m.id !== tempId),
        {
          id: `err-${Date.now()}`,
          sender: "system",
          text: "Could not send message to server.",
        },
      ]);
    } finally {
      setSending(false);
    }
  };

  const activeChat = activeChatId && activeChatId !== "general"
    ? myChats.find((c) => c.id === activeChatId)
    : null;

  return (
    <div className="chat-layout-container">
      {/* Left Sidebar */}
      <aside className="chat-sidebar">
        {/* Search Users Section */}
        <section className="sidebar-section">
          <h3 className="sidebar-heading">Search Users</h3>
          <div className="sidebar-placeholder">
            <input
              type="text"
              placeholder="Search by nickname..."
              className="sidebar-search-input"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
            />
          </div>

          {requestNotice && (
            <div className={`search-notice ${requestNotice.type}`}>
              {requestNotice.text}
            </div>
          )}

          {searching && (
            <div className="search-loading">
              <span className="spinner" /> <span>Searching...</span>
            </div>
          )}

          {searchError && <div className="search-error">{searchError}</div>}

          {!searching && searchQuery.trim() !== "" && searchResults.length === 0 && (
            <p className="placeholder-text">No users found</p>
          )}

          {searchResults.length > 0 && (
            <ul className="search-results-list">
              {searchResults.map((user) => {
                const status = requestStatus[user.id] || "idle";
                return (
                  <li key={user.id} className="search-user-item">
                    <div className="search-user-info">
                      <span className="search-user-name">{user.nickname}</span>
                      {user.real_name && (
                        <span className="search-user-sub">{user.real_name}</span>
                      )}
                    </div>
                    <button
                      type="button"
                      className={`btn-send-request ${status === "sent" ? "sent" : ""}`}
                      onClick={() => handleSendRequest(user)}
                      disabled={status === "sending" || status === "sent"}
                    >
                      {status === "sending" ? (
                        <>
                          <span className="spinner" /> <span>Sending</span>
                        </>
                      ) : status === "sent" ? (
                        "Sent"
                      ) : (
                        "Send Request"
                      )}
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
        </section>

        {/* Pending Requests Section */}
        <section className="sidebar-section">
          <div className="sidebar-heading-row">
            <h3 className="sidebar-heading">Pending Requests ({pendingRequests.length})</h3>
            <button
              type="button"
              className="btn-icon-sm"
              onClick={loadRequests}
              title="Refresh requests"
              disabled={loadingRequests}
            >
              {loadingRequests ? <span className="spinner" /> : "↻"}
            </button>
          </div>

          {loadingRequests && pendingRequests.length === 0 ? (
            <div className="search-loading">
              <span className="spinner" /> <span>Loading requests...</span>
            </div>
          ) : pendingRequests.length === 0 ? (
            <div className="sidebar-placeholder">
              <p className="placeholder-text">No pending requests</p>
            </div>
          ) : (
            <ul className="requests-list">
              {pendingRequests.map((req) => {
                const currentAction = actionInProgress[req.id];
                return (
                  <li key={req.id} className="request-card">
                    <div className="request-info">
                      <span className="request-sender">
                        {req.sender_nickname || req.sender_real_name || "User"}
                      </span>
                      {req.initial_message && (
                        <p className="request-msg">"{req.initial_message}"</p>
                      )}
                    </div>
                    <div className="request-actions">
                      <button
                        type="button"
                        className="btn-req-action accept"
                        disabled={!!currentAction}
                        onClick={() => handleRequestAction(req.id, "accept")}
                      >
                        {currentAction === "accept" ? <span className="spinner" /> : "Accept"}
                      </button>
                      <button
                        type="button"
                        className="btn-req-action reject"
                        disabled={!!currentAction}
                        onClick={() => handleRequestAction(req.id, "reject")}
                      >
                        {currentAction === "reject" ? <span className="spinner" /> : "Reject"}
                      </button>
                      <button
                        type="button"
                        className="btn-req-action block"
                        disabled={!!currentAction}
                        onClick={() => handleRequestAction(req.id, "block")}
                      >
                        {currentAction === "block" ? <span className="spinner" /> : "Block"}
                      </button>
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </section>

        {/* My Chats Section */}
        <section className="sidebar-section">
          <div className="sidebar-heading-row">
            <h3 className="sidebar-heading">My Chats</h3>
            <button
              type="button"
              className="btn-icon-sm"
              onClick={() => loadChats()}
              title="Refresh chats"
              disabled={loadingChats}
            >
              {loadingChats ? <span className="spinner" /> : "↻"}
            </button>
          </div>

          <div className="sidebar-chats-list">
            <div
              className={`chat-item ${!activeChatId || activeChatId === "general" ? "active" : ""}`}
              onClick={() => setActiveChatId("general")}
            >
              <div className="chat-item-avatar">G</div>
              <div className="chat-item-info">
                <span className="chat-item-name">General Chat</span>
                <span className="chat-item-preview">Global room</span>
              </div>
            </div>

            {myChats.map((chat) => {
              const isSelected = activeChatId === chat.id;
              const chatName = chat.name || "Direct Chat";
              const preview = chat.last_message_content || "No messages yet";
              const initial = chatName.charAt(0).toUpperCase();

              return (
                <div
                  key={chat.id}
                  className={`chat-item ${isSelected ? "active" : ""}`}
                  onClick={() => setActiveChatId(chat.id)}
                >
                  <div className="chat-item-avatar">{initial}</div>
                  <div className="chat-item-info">
                    <span className="chat-item-name">{chatName}</span>
                    <span className="chat-item-preview">{preview}</span>
                  </div>
                </div>
              );
            })}
          </div>
        </section>
      </aside>

      {/* Right Main Panel */}
      <main className="chat-main-panel">
        <div className="user-status-summary">
          <div className={`ws-status-badge ${wsStatus}`}>
            <span className="ws-dot" />
            <span>
              {wsStatus === "connected"
                ? "WebSocket Real-Time Connected"
                : wsStatus === "connecting"
                ? "Connecting WS..."
                : wsStatus === "reconnecting"
                ? "Reconnecting WS..."
                : "WebSocket Disconnected"}
            </span>
          </div>

          <div style={{ display: "flex", gap: "16px", alignItems: "center" }}>
            <div>
              Active Chat: <strong>{activeChat?.name || (activeChatId === "general" || !activeChatId ? "General Chat" : "Direct Chat")}</strong>
            </div>

            {currentUser && (
              <div>
                Logged in as <strong>{currentUser.nickname || currentUser.email}</strong>
              </div>
            )}
          </div>
        </div>

        <div className="chat-window">
          {chatLog.map((msg) => (
            <div key={msg.id} className={`message ${msg.sender}`}>
              {msg.text}
            </div>
          ))}
        </div>

        <form onSubmit={handleSend} className="chat-input-area">
          <input
            type="text"
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            placeholder="Type a message..."
            disabled={sending}
            autoFocus
          />
          <button
            type="submit"
            disabled={sending || !message.trim()}
            className="btn-primary"
            style={{ width: "auto" }}
          >
            {sending && <span className="spinner" />}
            <span>{sending ? "Sending..." : "Send"}</span>
          </button>
        </form>
      </main>
    </div>
  );
}
