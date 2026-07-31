import { useState, useEffect, useRef, Fragment } from "react";
import {
  sendMessage,
  getWebSocketUrl,
  searchUsers,
  sendRequest,
  fetchPendingRequests,
  updateRequestAction,
  fetchUserChats,
  fetchChatMessages,
  refreshToken,
} from "../api";
import UserProfileModal from "../components/UserProfileModal";

export default function ChatPage({ currentUser }) {
  const [message, setMessage] = useState("");
  const [sending, setSending] = useState(false);
  const [wsStatus, setWsStatus] = useState("disconnected");
  const [chatLog, setChatLog] = useState([
    { id: "sys-1", sender: "system", text: "Welcome to GoChat." },
  ]);
  const [loadingHistory, setLoadingHistory] = useState(false);
  const [loadingOlder, setLoadingOlder] = useState(false);
  const [hasMoreMessages, setHasMoreMessages] = useState(true);
  const [isTyping, setIsTyping] = useState(false);
  const [selectedUserId, setSelectedUserId] = useState(null);

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

  const activeChatIdRef = useRef(activeChatId);
  useEffect(() => {
    activeChatIdRef.current = activeChatId;
  }, [activeChatId]);

  const wsRef = useRef(null);
  const chatWindowRef = useRef(null);
  const inputRef = useRef(null);
  const skipAutoScrollRef = useRef(false);
  const typingTimeoutRef = useRef(null);
  const lastTypingSentRef = useRef(0);

  const formatTime = (dateString) => {
    if (!dateString) return "";
    const date = new Date(dateString);
    if (isNaN(date.getTime())) return "";
    return date.toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
  };

  const formatDateLabel = (dateString) => {
    if (!dateString) return "";
    const date = new Date(dateString);
    if (isNaN(date.getTime())) return "";
    const today = new Date();
    const yesterday = new Date(today);
    yesterday.setDate(yesterday.getDate() - 1);

    if (date.toDateString() === today.toDateString()) return "Today";
    if (date.toDateString() === yesterday.toDateString()) return "Yesterday";

    return date.toLocaleDateString([], { month: "short", day: "numeric", year: "numeric" });
  };

  // Auto scroll chat window to bottom when messages update
  useEffect(() => {
    if (skipAutoScrollRef.current) {
      skipAutoScrollRef.current = false;
      return;
    }
    if (chatWindowRef.current) {
      chatWindowRef.current.scrollTop = chatWindowRef.current.scrollHeight;
    }
  }, [chatLog, loadingHistory]);

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

  // Fetch Message History when activeChatId changes
  useEffect(() => {
    setIsTyping(false);
    if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current);
    setHasMoreMessages(true);
    if (!activeChatId || activeChatId === "general") {
      setChatLog([{ id: "sys-1", sender: "system", text: "Welcome to GoChat." }]);
      return;
    }

    async function loadHistory() {
      setLoadingHistory(true);
      try {
        const msgs = await fetchChatMessages(activeChatId);
        if (!msgs || msgs.length < 50) {
          setHasMoreMessages(false);
        } else {
          setHasMoreMessages(true);
        }
        const formatted = msgs.map((m) => ({
          id: m.id,
          sender: m.sender_id === currentUser?.id ? "me" : "them",
          text: m.content,
          createdAt: m.created_at,
          chatId: m.chat_id,
        })).reverse();
        setChatLog(formatted);
      } catch (err) {
        console.error("Failed to load chat history:", err);
        setChatLog([
          { id: "sys-err", sender: "system", text: "Failed to load chat history." },
        ]);
      } finally {
        setLoadingHistory(false);
      }
    }

    loadHistory();
  }, [activeChatId, currentUser?.id]);

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
      setPendingRequests((prev) => prev.filter((r) => r.id !== requestId));

      if (action === "accept") {
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
    let pingTimer = null;
    let reconnectAttempts = 0;
    let isMounted = true;

    function connect() {
      if (!isMounted) return;
      setWsStatus("connecting");

      const wsUrl = getWebSocketUrl("/ws");
      socket = new WebSocket(wsUrl);
      wsRef.current = socket;

      socket.onopen = () => {
        if (!isMounted) return;
        setWsStatus("connected");
        reconnectAttempts = 0;

        if (pingTimer) clearInterval(pingTimer);
        pingTimer = setInterval(() => {
          if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify({ type: "ping" }));
          }
        }, 30000);
      };

      socket.onmessage = (event) => {
        if (!isMounted) return;
        try {
          const payload = JSON.parse(event.data);
          if (payload.type === "new_message") {
            const isMe = payload.sender_id === currentUser?.id;
            const currentActiveChatId = activeChatIdRef.current;
            const matchesActiveChat =
              payload.chat_id === currentActiveChatId ||
              (currentActiveChatId === "general" && !payload.chat_id);

            if (matchesActiveChat) {
              const ZERO_UUID = "00000000-0000-0000-0000-000000000000";
              const validMsgId =
                payload.message_id && payload.message_id !== ZERO_UUID
                  ? payload.message_id
                  : null;

              setChatLog((prevLog) => {
                const isDuplicate = prevLog.some(
                  (msg) =>
                    (validMsgId && msg.id === validMsgId) ||
                    (msg.sender === "me" && isMe && msg.text === payload.content && msg.pending)
                );
                if (isDuplicate) {
                  return prevLog.map((msg) =>
                    msg.sender === "me" && isMe && msg.text === payload.content && msg.pending
                      ? {
                          ...msg,
                          id: validMsgId || msg.id,
                          pending: false,
                        }
                      : msg
                  );
                }
                return [
                  ...prevLog,
                  {
                    id: validMsgId || `msg-${Date.now()}-${Math.random()}`,
                    sender: isMe ? "me" : "them",
                    text: payload.content,
                    chatId: payload.chat_id,
                    createdAt: payload.created_at || new Date().toISOString(),
                  },
                ];
              });
            }

            // Always update sidebar chat preview when a new message arrives
            loadChats();
          } else if (payload.type === "new_request") {
            loadRequests();
          } else if (payload.type === "typing") {
            const currentActiveChatId = activeChatIdRef.current;
            if (
              payload.chat_id === currentActiveChatId &&
              payload.sender_id !== currentUser?.id
            ) {
              setIsTyping(true);
              if (typingTimeoutRef.current) clearTimeout(typingTimeoutRef.current);
              typingTimeoutRef.current = setTimeout(() => {
                setIsTyping(false);
              }, 3000);
            }
          }
        } catch (err) {
          console.error("Error parsing WebSocket message:", err);
        }
      };

      socket.onclose = async () => {
        if (pingTimer) clearInterval(pingTimer);
        if (!isMounted) return;
        setWsStatus("disconnected");

        try {
          await refreshToken();
        } catch (err) {
          console.warn("Token refresh failed during WS disconnect recovery:", err);
        }

        if (!isMounted) return;

        const delay = Math.min(30000, Math.pow(2, reconnectAttempts) * 1000);
        reconnectAttempts++;

        reconnectTimer = setTimeout(() => {
          if (isMounted) {
            setWsStatus("reconnecting");
            connect();
          }
        }, delay);
      };

      socket.onerror = (err) => {
        console.warn("WebSocket error encountered:", err);
      };
    }

    connect();

    return () => {
      isMounted = false;
      if (pingTimer) clearInterval(pingTimer);
      if (reconnectTimer) clearTimeout(reconnectTimer);
      if (socket) {
        socket.close();
      }
    };
  }, [currentUser?.id]);

  const handleScroll = async (e) => {
    const container = e.target;
    if (
      container.scrollTop === 0 &&
      !loadingHistory &&
      !loadingOlder &&
      hasMoreMessages &&
      activeChatId &&
      activeChatId !== "general"
    ) {
      const oldestMsg = chatLog.find(
        (m) => m.id && !String(m.id).startsWith("sys-") && !String(m.id).startsWith("temp-")
      );
      if (!oldestMsg) return;

      setLoadingOlder(true);
      const oldScrollHeight = container.scrollHeight;

      try {
        const olderMsgs = await fetchChatMessages(activeChatId, oldestMsg.id);
        if (!olderMsgs || olderMsgs.length === 0) {
          setHasMoreMessages(false);
        } else {
          if (olderMsgs.length < 50) {
            setHasMoreMessages(false);
          }
          const formatted = olderMsgs
            .map((m) => ({
              id: m.id,
              sender: m.sender_id === currentUser?.id ? "me" : "them",
              text: m.content,
              createdAt: m.created_at,
              chatId: m.chat_id,
            }))
            .reverse();

          skipAutoScrollRef.current = true;
          setChatLog((prev) => [...formatted, ...prev]);

          requestAnimationFrame(() => {
            const newScrollHeight = container.scrollHeight;
            container.scrollTop = newScrollHeight - oldScrollHeight;
          });
        }
      } catch (err) {
        console.error("Failed to load older messages:", err);
      } finally {
        setLoadingOlder(false);
      }
    }
  };

  const handleInputChange = (e) => {
    const val = e.target.value;
    setMessage(val);

    if (val.trim() && wsStatus === "connected" && wsRef.current) {
      const now = Date.now();
      if (now - lastTypingSentRef.current > 2000) {
        console.log("Sending typing event...", { type: "typing", chat_id: activeChatId });
        try {
          if (wsRef.current.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify({ type: "typing", chat_id: activeChatId }));
            lastTypingSentRef.current = now;
          } else {
            console.log("WebSocket not open. ReadyState:", wsRef.current.readyState);
          }
        } catch (err) {
          console.error("Failed to send typing event", err);
        }
      }
    }
  };

  const handleSend = async (e) => {
    e.preventDefault();
    if (!message.trim() || sending) return;

    const userMessage = message.trim();
    const tempId = `temp-${Date.now()}`;

    setChatLog((prevLog) => [
      ...prevLog,
      { id: tempId, sender: "me", text: userMessage, pending: true, createdAt: new Date().toISOString() },
    ]);
    setMessage("");
    setSending(true);

    try {
      await sendMessage(activeChatId, userMessage);
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
      setTimeout(() => {
        inputRef.current?.focus();
      }, 0);
    }
  };

  const activeChat = activeChatId && activeChatId !== "general"
    ? myChats.find((c) => c.id === activeChatId)
    : null;

  const isDeletedAccount = (user) => {
    if (!user) return false;
    return user.status === "deleted";
  };

  const formatUserDisplayName = (user, fallback = "User") => {
    if (!user) return fallback;
    if (isDeletedAccount(user)) return "Deleted Account";
    if (typeof user === "string") return user;
    return user.nickname || user.real_name || fallback;
  };

  const getChatDisplayName = (chat) => {
    if (!chat) return "";
    if (chat.name) return chat.name;
    if (chat.is_group) return "Group Chat";
    if (chat.participants && Array.isArray(chat.participants)) {
      const other = chat.participants.find((p) => p.id !== currentUser?.id);
      if (other) {
        return formatUserDisplayName(other, "Direct Chat");
      }
    }
    return "Direct Chat";
  };

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
                    <div
                      className="search-user-info"
                      style={{ cursor: "pointer" }}
                      onClick={() => setSelectedUserId(user.id)}
                    >
                      <span className="search-user-name">{formatUserDisplayName(user)}</span>
                      {!isDeletedAccount(user) && user.real_name && (
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
                        {formatUserDisplayName({
                          nickname: req.sender_nickname,
                          real_name: req.sender_real_name,
                          status: req.status,
                        })}
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
              const chatName = getChatDisplayName(chat);
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
              Active Chat:{" "}
              <strong
                style={{
                  cursor: activeChat?.participants ? "pointer" : "default",
                  textDecoration: activeChat?.participants ? "underline" : "none",
                }}
                onClick={() => {
                  if (activeChat?.participants) {
                    const other = activeChat.participants.find((p) => p.id !== currentUser?.id);
                    if (other) setSelectedUserId(other.id);
                  }
                }}
              >
                {!activeChatId || activeChatId === "general"
                  ? "General Chat"
                  : getChatDisplayName(activeChat)}
              </strong>
            </div>

            {currentUser && (
              <div>
                Logged in as <strong>{currentUser.nickname || currentUser.email}</strong>
              </div>
            )}
          </div>
        </div>

        <div className="chat-window-container">
          <div className="chat-window" ref={chatWindowRef} onScroll={handleScroll}>
            {loadingOlder && (
              <div className="search-loading" style={{ padding: "8px 0", justifyContent: "center" }}>
                <span className="spinner" /> <span>Loading older messages...</span>
              </div>
            )}
            {loadingHistory ? (
              <div className="search-loading" style={{ margin: "auto" }}>
                <span className="spinner-lg" />
              </div>
            ) : chatLog.length === 0 ? (
              <p className="placeholder-text" style={{ margin: "auto" }}>
                No messages yet. Send a message to start chatting!
              </p>
            ) : (
              chatLog.map((msg, index) => {
                const currentLabel = msg.createdAt ? formatDateLabel(msg.createdAt) : null;
                const previousLabel =
                  index > 0 && chatLog[index - 1].createdAt
                    ? formatDateLabel(chatLog[index - 1].createdAt)
                    : null;
                const showSeparator = currentLabel && currentLabel !== previousLabel;

                return (
                  <Fragment key={msg.id}>
                    {showSeparator && <div className="date-separator">{currentLabel}</div>}
                    <div className={`message-wrapper ${msg.sender}`}>
                      <div className={`message ${msg.sender}`}>
                        <div className="message-content">{msg.text}</div>
                      </div>
                      {msg.createdAt && (
                        <div className="message-time">{formatTime(msg.createdAt)}</div>
                      )}
                    </div>
                  </Fragment>
                );
              })
            )}
          </div>

          {isTyping && (
            <div className="typing-indicator">
              <span className="spinner" /> <span>Someone is typing...</span>
            </div>
          )}
        </div>

        {activeChat && activeChat.can_send_messages === false ? (
          <div className="chat-disabled-message" style={{ textAlign: "center", padding: "16px", color: "var(--text-secondary)", fontStyle: "italic", borderTop: "1px solid var(--border-light)", background: "var(--surface-50)" }}>
            You cannot send messages to this user.
          </div>
        ) : (
          <form onSubmit={handleSend} className="chat-input-area">
            <input
              ref={inputRef}
              type="text"
              value={message}
              onChange={handleInputChange}
              placeholder="Type a message..."
              autoFocus
            />
            <button
              type="submit"
              disabled={sending || !message.trim()}
              className="btn-primary"
              style={{ width: "auto" }}
            >
              {sending ? <span className="spinner" /> : "Send"}
            </button>
          </form>
        )}
      </main>

      {selectedUserId && (
        <UserProfileModal
          userId={selectedUserId}
          onClose={() => setSelectedUserId(null)}
          onRelationshipChange={() => {
            loadChats();
            loadRequests();
          }}
        />
      )}
    </div>
  );
}
