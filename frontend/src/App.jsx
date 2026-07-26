import { useState, useEffect } from "react";
import Auth from "./components/Auth";
import UserProfile from "./components/UserProfile";
import { fetchCurrentUser, logoutUser, getToken } from "./api";
import "./App.css";

function App() {
    const [currentPage, setCurrentPage] = useState("chat"); // 'chat' | 'auth'
    const [currentUser, setCurrentUser] = useState(null);
    const [initializing, setInitializing] = useState(true);

    // Original Chat State
    const [message, setMessage] = useState("");
    const [chatLog, setChatLog] = useState([
        { sender: "system", text: "Welcome to GoChat!" },
    ]);

    // Restore user session on mount
    useEffect(() => {
        async function loadUser() {
            if (getToken()) {
                try {
                    const user = await fetchCurrentUser();
                    setCurrentUser(user);
                } catch (err) {
                    console.warn("Session restore note:", err.message);
                }
            }
            setInitializing(false);
        }
        loadUser();
    }, []);

    const handleLoginSuccess = async (user) => {
        if (user && user.email) {
            setCurrentUser(user);
        } else {
            try {
                const fullUser = await fetchCurrentUser();
                setCurrentUser(fullUser);
            } catch (err) {
                console.error("Could not fetch profile:", err);
            }
        }
        // Automatically navigate to chat page upon successful login
        setCurrentPage("chat");
    };

    const handleLogout = async () => {
        await logoutUser();
        setCurrentUser(null);
    };

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
            const response = await fetch("http://localhost:8080/messages", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    ...(getToken()
                        ? { Authorization: `Bearer ${getToken()}` }
                        : {}),
                },
                // Adjust this payload object keys (e.g., text, content, or chat_id)
                // to match whatever your Go HandlerSendMessage struct expects!
                body: JSON.stringify({ text: userMessage }),
            });

            if (!response.ok) {
                throw new Error(`Server error: ${response.status}`);
            }

            const data = await response.json();

            setChatLog((prevLog) => [
                ...prevLog,
                { sender: "system", text: data.reply || "Message received!" },
            ]);
        } catch (error) {
            console.error("Failed to fetch:", error);
            setChatLog((prevLog) => [
                ...prevLog,
                {
                    sender: "system",
                    text: "⚠️ Could not connect to the server.",
                },
            ]);
        }
    };

    if (initializing) {
        return (
            <div className="chat-container">
                <header className="top-nav">
                    <span className="nav-brand">GoChat</span>
                </header>
                <div className="page-container">
                    <p>Loading application...</p>
                </div>
            </div>
        );
    }

    return (
        <div className="chat-container">
            {/* Top Navigation Bar for switching between pages */}
            <header className="top-nav">
                <span className="nav-brand">GoChat</span>
                <div className="nav-links">
                    <button
                        type="button"
                        className={`nav-btn ${currentPage === "chat" ? "active" : ""}`}
                        onClick={() => setCurrentPage("chat")}
                    >
                        💬 Chat Page
                    </button>
                    <button
                        type="button"
                        className={`nav-btn ${currentPage === "auth" ? "active" : ""}`}
                        onClick={() => setCurrentPage("auth")}
                    >
                        {currentUser
                            ? `👤 Account (${currentUser.nickname || currentUser.email})`
                            : "🔑 Login / Register"}
                    </button>
                </div>
            </header>

            {/* Page 1: Chat Page */}
            {currentPage === "chat" && (
                <main className="page-container">
                    {currentUser && (
                        <div
                            className="user-status-summary"
                            style={{ marginBottom: "12px", textAlign: "right" }}
                        >
                            Logged in as{" "}
                            <strong>
                                {currentUser.nickname || currentUser.email}
                            </strong>
                        </div>
                    )}
                    <div className="chat-window">
                        {chatLog.map((msg, index) => (
                            <div
                                key={index}
                                className={`message ${msg.sender}`}
                            >
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
            )}

            {/* Page 2: Auth Page */}
            {currentPage === "auth" && (
                <main className="page-container">
                    {currentUser ? (
                        <UserProfile
                            user={currentUser}
                            onLogout={handleLogout}
                        />
                    ) : (
                        <div className="auth-wrapper">
                            <Auth onLoginSuccess={handleLoginSuccess} />
                        </div>
                    )}
                </main>
            )}
        </div>
    );
}

export default App;
