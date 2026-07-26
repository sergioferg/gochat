import { useState } from "react";
import { sendMessage } from "../api";

export default function ChatPage({ currentUser }) {
  const [message, setMessage] = useState("");
  const [sending, setSending] = useState(false);
  const [chatLog, setChatLog] = useState([
    { sender: "system", text: "Welcome to GoChat." },
  ]);

  const handleSend = async (e) => {
    e.preventDefault();
    if (!message.trim() || sending) return;

    const userMessage = message.trim();
    setChatLog((prevLog) => [
      ...prevLog,
      { sender: "me", text: userMessage },
    ]);
    setMessage("");
    setSending(true);

    try {
      const data = await sendMessage(userMessage);
      setChatLog((prevLog) => [
        ...prevLog,
        { sender: "system", text: data?.reply || "Message received." },
      ]);
    } catch (error) {
      console.error("Failed to send message:", error);
      setChatLog((prevLog) => [
        ...prevLog,
        {
          sender: "system",
          text: "Could not connect to the server.",
        },
      ]);
    } finally {
      setSending(false);
    }
  };

  return (
    <main className="page-container">
      {currentUser && (
        <div
          className="user-status-summary"
          style={{ textAlign: "right" }}
        >
          Logged in as{" "}
          <strong>{currentUser.nickname || currentUser.email}</strong>
        </div>
      )}
      <div className="chat-window">
        {chatLog.map((msg, index) => (
          <div key={index} className={`message ${msg.sender}`}>
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
        <button type="submit" disabled={sending || !message.trim()} className="btn-primary" style={{ width: "auto" }}>
          {sending && <span className="spinner" />}
          <span>{sending ? "Sending..." : "Send"}</span>
        </button>
      </form>
    </main>
  );
}
