import { useEffect } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { setToken, fetchCurrentUser } from "../api";

export default function OAuthCallback({ onLoginSuccess }) {
  const location = useLocation();
  const navigate = useNavigate();

  useEffect(() => {
    async function handleOAuthCallback() {
      const hash = location.hash;
      let token = null;

      if (hash) {
        const hashParams = new URLSearchParams(
          hash.startsWith("#") ? hash.substring(1) : hash
        );
        token = hashParams.get("access_token") || hashParams.get("token");
      }

      if (token) {
        // Save access_token into localStorage
        setToken(token);
        localStorage.setItem("token", token);

        if (onLoginSuccess) {
          try {
            const user = await fetchCurrentUser();
            await onLoginSuccess(user);
          } catch (err) {
            console.error("Failed to fetch user profile after OAuth:", err);
          }
        }

        // Immediately redirect to /chat replacing history entry
        navigate("/chat", { replace: true });
      } else {
        // No token found, redirect to /login
        navigate("/login", { replace: true });
      }
    }

    handleOAuthCallback();
  }, [location.hash, navigate, onLoginSuccess]);

  return (
    <main className="page-container" style={{ display: "flex", justifyContent: "center", alignItems: "center", minHeight: "60vh" }}>
      <div className="hero-card" style={{ textAlign: "center", padding: "var(--space-8)" }}>
        <div className="spinner-lg" style={{ marginBottom: "var(--space-4)" }} />
        <p style={{ color: "var(--color-text-muted)" }}>Completing authentication...</p>
      </div>
    </main>
  );
}
