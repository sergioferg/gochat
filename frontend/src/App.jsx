import { useState, useEffect } from "react";
import { Routes, Route, NavLink, Link, useLocation, Navigate } from "react-router-dom";
import Home from "./pages/Home";
import ChatPage from "./pages/ChatPage";
import OAuthCallback from "./pages/OAuthCallback";
import VerifyEmailPage from "./pages/VerifyEmailPage";
import CompleteProfile from "./pages/CompleteProfile";
import Auth from "./components/Auth";
import UserProfile from "./components/UserProfile";
import { fetchCurrentUser, logoutUser, getToken } from "./api";
import "./App.css";

function App() {
    const [currentUser, setCurrentUser] = useState(null);
    const [initializing, setInitializing] = useState(true);
    const [isAuthModalOpen, setIsAuthModalOpen] = useState(false);
    const [authMode, setAuthMode] = useState("login");
    const location = useLocation();

    const handleOpenAuthModal = (mode = "login") => {
        setAuthMode(mode);
        setIsAuthModalOpen(true);
    };

    // Open modal if user navigated directly to /login or /register
    useEffect(() => {
        if (location.pathname === "/login") {
            setAuthMode("login");
            setIsAuthModalOpen(true);
        } else if (location.pathname === "/register") {
            setAuthMode("register");
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
                <Link to="/" className="nav-brand" style={{ textDecoration: "none", display: "flex", alignItems: "center", gap: "10px" }}>
                    <img src="/logo.png" alt="GoChat Logo" style={{ width: "28px", height: "28px", borderRadius: "6px", objectFit: "contain" }} />
                    <span>GoChat</span>
                </Link>
                <div className="nav-links">
                    {currentUser ? (
                        <>
                            <NavLink
                                to="/chat"
                                className={({ isActive }) => `nav-btn ${isActive ? "active" : ""}`}
                                style={{ textDecoration: "none" }}
                            >
                                Chat
                            </NavLink>
                            <button
                                type="button"
                                className={`nav-btn ${isAuthModalOpen ? "active" : ""}`}
                                onClick={() => handleOpenAuthModal("login")}
                            >
                                Account ({currentUser.nickname || currentUser.email})
                            </button>
                        </>
                    ) : (
                        <>
                            <button
                                type="button"
                                className={`nav-btn ${isAuthModalOpen && authMode === "login" ? "active" : ""}`}
                                onClick={() => handleOpenAuthModal("login")}
                            >
                                Login
                            </button>
                            <button
                                type="button"
                                className={`nav-btn ${isAuthModalOpen && authMode === "register" ? "active" : ""}`}
                                onClick={() => handleOpenAuthModal("register")}
                            >
                                Register
                            </button>
                        </>
                    )}
                </div>
            </header>

            {/* Page Routes */}
            <Routes>
                <Route
                    path="/"
                    element={
                        <Home
                            currentUser={currentUser}
                            onOpenAuthModal={handleOpenAuthModal}
                        />
                    }
                />
                <Route
                    path="/login"
                    element={
                        <Home
                            currentUser={currentUser}
                            onOpenAuthModal={handleOpenAuthModal}
                        />
                    }
                />
                <Route
                    path="/oauth-callback"
                    element={
                        <OAuthCallback
                            onLoginSuccess={handleLoginSuccess}
                        />
                    }
                />
                <Route
                    path="/verify-email"
                    element={
                        <VerifyEmailPage
                            onOpenAuthModal={handleOpenAuthModal}
                        />
                    }
                />
                <Route
                    path="/complete-profile"
                    element={
                        currentUser ? (
                            currentUser.date_of_birth ? (
                                <Navigate to="/chat" replace />
                            ) : (
                                <CompleteProfile 
                                    currentUser={currentUser} 
                                    onComplete={(user) => setCurrentUser(user)} 
                                />
                            )
                        ) : (
                            <Navigate to="/" replace />
                        )
                    }
                />
                <Route
                    path="/chat"
                    element={
                        currentUser ? (
                            currentUser.date_of_birth ? (
                                <ChatPage currentUser={currentUser} />
                            ) : (
                                <Navigate to="/complete-profile" replace />
                            )
                        ) : (
                            <Home
                                currentUser={currentUser}
                                onOpenAuthModal={handleOpenAuthModal}
                            />
                        )
                    }
                />
                <Route
                    path="*"
                    element={
                        <Home
                            currentUser={currentUser}
                            onOpenAuthModal={handleOpenAuthModal}
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
                        <div className="modal-header">
                            <span className="modal-title">GoChat</span>
                            <button
                                type="button"
                                className="modal-close-btn"
                                onClick={() => setIsAuthModalOpen(false)}
                                aria-label="Close modal"
                            >
                                ✕
                            </button>
                        </div>
                        {currentUser ? (
                            <UserProfile
                                user={currentUser}
                                onLogout={() => {
                                    handleLogout();
                                    setIsAuthModalOpen(false);
                                }}
                            />
                        ) : (
                            <Auth onLoginSuccess={handleLoginSuccess} initialMode={authMode} />
                        )}
                    </div>
                </div>
            )}
        </div>
    );
}

export default App;
