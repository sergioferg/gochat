import { useState, useEffect } from "react";
import { Routes, Route, NavLink, Link, useLocation } from "react-router-dom";
import Home from "./pages/Home";
import ChatPage from "./pages/ChatPage";
import Auth from "./components/Auth";
import UserProfile from "./components/UserProfile";
import { fetchCurrentUser, logoutUser, getToken } from "./api";
import "./App.css";

function App() {
    const [currentUser, setCurrentUser] = useState(null);
    const [initializing, setInitializing] = useState(true);
    const [isAuthModalOpen, setIsAuthModalOpen] = useState(false);
    const location = useLocation();

    // Open modal if user navigated directly to /login
    useEffect(() => {
        if (location.pathname === "/login") {
            setIsAuthModalOpen(true);
        }
    }, [location.pathname]);

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
        // Automatically close modal after login
        setIsAuthModalOpen(false);
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
                <div className="skeleton-container">
                    <div className="skeleton-box" style={{ height: "32px", width: "40%", marginBottom: "16px" }}></div>
                    <div className="skeleton-box" style={{ height: "18px", width: "80%", marginBottom: "24px" }}></div>
                    <div className="skeleton-box" style={{ height: "180px", width: "100%" }}></div>
                </div>
            </div>
        );
    }

    return (
        <div className="chat-container">
            {/* Top Navigation Bar */}
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
                        Home
                    </NavLink>
                    {currentUser && (
                        <NavLink
                            to="/chat"
                            className={({ isActive }) => `nav-btn ${isActive ? "active" : ""}`}
                            style={{ textDecoration: "none" }}
                        >
                            Chat
                        </NavLink>
                    )}
                    <button
                        type="button"
                        className={`nav-btn ${isAuthModalOpen ? "active" : ""}`}
                        onClick={() => setIsAuthModalOpen(true)}
                    >
                        {currentUser
                            ? `Account (${currentUser.nickname || currentUser.email})`
                            : "Login / Register"}
                    </button>
                </div>
            </header>

            {/* Page Routes */}
            <Routes>
                <Route
                    path="/"
                    element={
                        <Home
                            currentUser={currentUser}
                            onOpenAuthModal={() => setIsAuthModalOpen(true)}
                        />
                    }
                />
                <Route
                    path="/login"
                    element={
                        <Home
                            currentUser={currentUser}
                            onOpenAuthModal={() => setIsAuthModalOpen(true)}
                        />
                    }
                />
                <Route
                    path="/chat"
                    element={
                        currentUser ? (
                            <ChatPage currentUser={currentUser} />
                        ) : (
                            <Home
                                currentUser={currentUser}
                                onOpenAuthModal={() => setIsAuthModalOpen(true)}
                            />
                        )
                    }
                />
                <Route
                    path="*"
                    element={
                        <Home
                            currentUser={currentUser}
                            onOpenAuthModal={() => setIsAuthModalOpen(true)}
                        />
                    }
                />
            </Routes>

            {/* Centered Login / Register Modal with Dimmed Background */}
            {isAuthModalOpen && (
                <div
                    className="modal-overlay"
                    onClick={() => setIsAuthModalOpen(false)}
                >
                    <div
                        className="modal-content"
                        onClick={(e) => e.stopPropagation()}
                    >
                        <button
                            type="button"
                            className="modal-close-btn"
                            onClick={() => setIsAuthModalOpen(false)}
                            aria-label="Close modal"
                        >
                            ✕
                        </button>
                        {currentUser ? (
                            <UserProfile
                                user={currentUser}
                                onLogout={() => {
                                    handleLogout();
                                    setIsAuthModalOpen(false);
                                }}
                            />
                        ) : (
                            <Auth onLoginSuccess={handleLoginSuccess} />
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}

export default App;
