import { useEffect, useState } from "react";
import { useLocation, useNavigate, Link } from "react-router-dom";
import { verifyUser } from "../api";

export default function OAuthCallbackPage({ onOpenAuthModal }) {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState(false);
  const location = useLocation();
  const navigate = useNavigate();

  useEffect(() => {
    async function doVerification() {
      let token = null;

      // Extract access_token from URL hash (e.g. #access_token=xyz)
      if (location.hash) {
        const hashParams = new URLSearchParams(
          location.hash.startsWith("#") ? location.hash.substring(1) : location.hash
        );
        token = hashParams.get("access_token") || hashParams.get("token");
      }

      // Fallback: extract from URL search query (e.g. ?access_token=xyz)
      if (!token && location.search) {
        const searchParams = new URLSearchParams(location.search);
        token = searchParams.get("access_token") || searchParams.get("token");
      }

      if (!token) {
        setError("No verification token found in URL.");
        setLoading(false);
        return;
      }

      try {
        await verifyUser(token);
        setSuccess(true);
      } catch (err) {
        setError(err.message || "Verification failed.");
      } finally {
        setLoading(false);
      }
    }

    doVerification();
  }, [location.hash, location.search]);

  return (
    <main className="page-container" style={{ display: "flex", justifyContent: "center", alignItems: "center", minHeight: "60vh" }}>
      <div className="hero-card" style={{ textAlign: "center", width: "100%", maxWidth: "480px", padding: "var(--space-8) var(--space-6)" }}>
        {loading && (
          <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: "var(--space-4)" }}>
            <div className="spinner-lg" />
            <p style={{ fontSize: "var(--font-size-base)", fontWeight: "var(--font-weight-medium)", color: "var(--color-text-main)", margin: 0 }}>
              verificating...
            </p>
          </div>
        )}

        {!loading && success && (
          <div>
            <div style={{ fontSize: "2.5rem", marginBottom: "var(--space-3)" }}>✅</div>
            <h2 style={{ marginBottom: "var(--space-3)", color: "var(--color-success-text, #10b981)" }}>
              Verification Successful!
            </h2>
            <p style={{ marginBottom: "var(--space-6)", color: "var(--color-text-muted)" }}>
              Your account has been verified successfully. You may now log in.
            </p>
            <div style={{ display: "flex", gap: "var(--space-3)", justifyContent: "center" }}>
              <button
                type="button"
                className="btn-primary"
                onClick={() => {
                  if (onOpenAuthModal) onOpenAuthModal("login");
                  navigate("/");
                }}
                style={{ width: "auto" }}
              >
                Log In Now
              </button>
            </div>
          </div>
        )}

        {!loading && error && (
          <div>
            <div style={{ fontSize: "2.5rem", marginBottom: "var(--space-3)" }}>❌</div>
            <h2 style={{ marginBottom: "var(--space-3)", color: "var(--color-error-text, #ef4444)" }}>
              Verification Failed
            </h2>
            <p style={{ marginBottom: "var(--space-6)", color: "var(--color-text-muted)" }}>
              {error}
            </p>
            <div style={{ display: "flex", gap: "var(--space-3)", justifyContent: "center" }}>
              <Link to="/" className="btn-secondary" style={{ textDecoration: "none" }}>
                Back to Home
              </Link>
            </div>
          </div>
        )}
      </div>
    </main>
  );
}
