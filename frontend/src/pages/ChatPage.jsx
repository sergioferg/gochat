import { useState, useEffect, useRef } from "react";
import { sendMessage, getWebSocketUrl } from "../api";

export default function ChatPage({ currentUser }) {
  const [message, setMessage] = useState("");
  const [sending, setSending] = useState(false);
  const [wsStatus, setWsStatus] = useState("disconnected");
  const [chatLog, setChatLog] = useState([
    { id: "sys-1", sender: "system", text: "Welcome to GoChat." },
  ]);

  const wsRef = useRef(null);

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
              // Deduplicate if message was added optimistically or already received
              const isDuplicate = prevLog.some(
                (msg) =>
                  (msg.id && msg.id === payload.message_id) ||
                  (msg.sender === "me" && isMe && msg.text === payload.content && msg.pending)
              );
              if (isDuplicate) {
                // Replace pending message with official message
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
          // Attempt automatic reconnection after 3 seconds
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

    // Optimistically add message
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

  return (
    <div className="chat-layout-container">
      {/* Left Sidebar */}
      <aside className="chat-sidebar">
        <section className="sidebar-section">
          <h3 className="sidebar-heading">Search Users</h3>
          <div className="sidebar-placeholder">
            <input
              type="text"
              placeholder="Search users..."
              className="sidebar-search-input"
              readOnly
            />
          </div>
        </section>

        <section className="sidebar-section">
          <h3 className="sidebar-heading">Pending Requests</h3>
          <div className="sidebar-placeholder">
            <p className="placeholder-text">No pending requests</p>
          </div>
        </section>

        <section className="sidebar-section">
          <h3 className="sidebar-heading">My Chats</h3>
          <div className="sidebar-placeholder">
            <div className="chat-item active">
              <div className="chat-item-avatar">G</div>
              <div className="chat-item-info">
                <span className="chat-item-name">General Chat</span>
                <span className="chat-item-preview">Welcome to GoChat.</span>
              </div>
            </div>
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

          {currentUser && (
            <div>
              Logged in as <strong>{currentUser.nickname || currentUser.email}</strong>
            </div>
          )}
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
