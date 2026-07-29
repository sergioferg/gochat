import { Link } from "react-router-dom";

export default function Home({ currentUser, onOpenAuthModal }) {
  return (
    <main className="page-container">
      <div className="hero-card" style={{ textAlign: "center", padding: "var(--space-8) var(--space-4)" }}>
        <img
          src="/logo.png"
          alt="GoChat Logo"
          style={{ width: "80px", height: "80px", marginBottom: "var(--space-4)", objectFit: "contain" }}
        />
        <h1>Welcome to GoChat</h1>
        <p style={{ maxWidth: "600px", margin: "0 auto var(--space-6)" }}>
          A real-time messaging application connected to a Go server on Azure App Service and deployed via Azure Static Web Apps.
        </p>

        {currentUser ? (
          <div style={{ marginBottom: "var(--space-8)" }}>
            <p style={{ marginBottom: "var(--space-4)" }}>
              Authenticated as <strong>{currentUser.nickname || currentUser.email}</strong>
            </p>
            <div style={{ display: "flex", gap: "var(--space-4)", justifyContent: "center", flexWrap: "wrap" }}>
              <Link to="/chat" className="btn-primary" style={{ width: "auto", textDecoration: "none" }}>
                Open Chat Interface
              </Link>
              <button
                type="button"
                className="btn-secondary"
                onClick={onOpenAuthModal}
              >
                Account Details
              </button>
            </div>
          </div>
        ) : (
          <div style={{ display: "flex", gap: "var(--space-4)", justifyContent: "center", marginBottom: "var(--space-8)", flexWrap: "wrap" }}>
            <button
              type="button"
              className="btn-primary"
              onClick={() => onOpenAuthModal("login")}
              style={{ width: "auto" }}
            >
              Log In
            </button>
            <button
              type="button"
              className="btn-secondary"
              onClick={() => onOpenAuthModal("register")}
              style={{ width: "auto" }}
            >
              Register
            </button>
          </div>
        )}

        <div style={{ marginTop: "var(--space-8)", display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))", gap: "var(--space-4)", textAlign: "left" }}>
          <div style={{ padding: "var(--space-4)", border: "1px solid var(--color-border)", borderRadius: "var(--radius-md)", background: "var(--color-bg)" }}>
            <h3 style={{ marginTop: 0, marginBottom: "var(--space-2)" }}>Go Backend</h3>
            <p style={{ fontSize: "var(--font-size-sm)", color: "var(--color-text-muted)", margin: 0 }}>
              Hosted on Azure App Service at api.trygochat.tech.
            </p>
          </div>
          <div style={{ padding: "var(--space-4)", border: "1px solid var(--color-border)", borderRadius: "var(--radius-md)", background: "var(--color-bg)" }}>
            <h3 style={{ marginTop: 0, marginBottom: "var(--space-2)" }}>Azure Static Web Apps</h3>
            <p style={{ fontSize: "var(--font-size-sm)", color: "var(--color-text-muted)", margin: 0 }}>
              Single Page Application with navigation fallback routing.
            </p>
          </div>
          <div style={{ padding: "var(--space-4)", border: "1px solid var(--color-border)", borderRadius: "var(--radius-md)", background: "var(--color-bg)" }}>
            <h3 style={{ marginTop: 0, marginBottom: "var(--space-2)" }}>JWT Authentication</h3>
            <p style={{ fontSize: "var(--font-size-sm)", color: "var(--color-text-muted)", margin: 0 }}>
              Token-based user session handling and email verification.
            </p>
          </div>
        </div>
      </div>
    </main>
  );
}
