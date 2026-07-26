import { useState } from "react";
import { sendMessage } from "../api";

export default function ChatPage({ currentUser }) {
  const [message, setMessage] = useState("");
  const [chatLog, setChatLog] = useState([
    { sender: "system", text: "Welcome to GoChat!" },
  ]);

  const handleSend = async (e) => {
    e.preventDefault();
    if (!message.trim()) return;

    const userMessage = message;

    setChatLog((prevLog) => [
      ...prevLog,
      { sender: "me", text: userMessage },
    ]);
    setMessage("");

    try {
      const data = await sendMessage(userMessage);
      setChatLog((prevLog) => [
        ...prevLog,
        { sender: "system", text: data?.reply || "Message received!" },
      ]);
    } catch (error) {
      console.error("Failed to send message:", error);
      setChatLog((prevLog) => [
        ...prevLog,
        {
          sender: "system",
          text: "⚠️ Could not connect to the server.",
        },
      ]);
    }
  };

  return (
    <main className="page-container">
      {currentUser && (
        <div
          className="user-status-summary"
          style={{ marginBottom: "12px", textAlign: "right" }}
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
          placeholder="Type your message..."
          autoFocus
        />
        <button type="submit">Send</button>
      </form>
    </main>
  );
}
