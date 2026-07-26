import { useState, useEffect } from "react";
import { Routes, Route, NavLink, Link, useNavigate } from "react-router-dom";
import Home from "./pages/Home";
import LoginPage from "./pages/LoginPage";
import ChatPage from "./pages/ChatPage";
import { fetchCurrentUser, logoutUser, getToken } from "./api";
import "./App.css";

function App() {
    const [currentUser, setCurrentUser] = useState(null);
    const [initializing, setInitializing] = useState(true);
    const navigate = useNavigate();

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
        navigate("/chat");
    };

    const handleLogout = async () => {
        await logoutUser();
        setCurrentUser(null);
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
            {/* Top Navigation Bar with react-router-dom Link/NavLink */}
            <header className="top-nav">
                <Link to="/" className="nav-brand" style={{ textDecoration: "none" }}>
                    GoChat
                </Link>
                <div className="nav-links">
                    <NavLink
                        to="/"
                        end
                        className={({ isActive }) => `nav-btn ${isActive ? "active" : ""}`}
                        style={{ textDecoration: "none" }}
                    >
                        🏠 Home
                    </NavLink>
                    <NavLink
                        to="/chat"
                        className={({ isActive }) => `nav-btn ${isActive ? "active" : ""}`}
                        style={{ textDecoration: "none" }}
                    >
                        💬 Chat Page
                    </NavLink>
                    <NavLink
                        to="/login"
                        className={({ isActive }) => `nav-btn ${isActive ? "active" : ""}`}
                        style={{ textDecoration: "none" }}
                    >
                        {currentUser
                            ? `👤 Account (${currentUser.nickname || currentUser.email})`
                            : "🔑 Login / Register"}
                    </NavLink>
                </div>
            </header>

            {/* Page Routes */}
            <Routes>
                <Route path="/" element={<Home currentUser={currentUser} />} />
                <Route
                    path="/login"
                    element={
                        <LoginPage
                            currentUser={currentUser}
                            onLoginSuccess={handleLoginSuccess}
                            onLogout={handleLogout}
                        />
                    }
                />
                <Route
                    path="/chat"
                    element={<ChatPage currentUser={currentUser} />}
                />
                <Route path="*" element={<Home currentUser={currentUser} />} />
            </Routes>
        </div>
    );
}

export default App;
