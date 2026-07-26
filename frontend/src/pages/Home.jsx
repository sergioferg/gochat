import { Link } from "react-router-dom";

export default function Home({ currentUser }) {
  return (
    <main className="page-container">
      <div className="hero-card" style={{ textAlign: "center", padding: "40px 20px" }}>
        <h1 style={{ fontSize: "2.2rem", marginBottom: "12px" }}>Welcome to GoChat 🚀</h1>
        <p style={{ fontSize: "1.1rem", color: "var(--text)", marginBottom: "24px", maxWidth: "600px", marginInline: "auto" }}>
          A fast, real-time messaging application powered by a High-Performance Go backend hosted on Azure App Service and a React frontend on Azure Static Web Apps.
        </p>

        {currentUser ? (
          <div style={{ marginBottom: "28px" }}>
            <p style={{ fontSize: "1rem", marginBottom: "16px" }}>
              Logged in as <strong>{currentUser.nickname || currentUser.email}</strong>
            </p>
            <Link to="/chat" className="btn-primary" style={{ display: "inline-block", width: "auto", padding: "12px 24px", textDecoration: "none" }}>
              💬 Open Chat Interface
            </Link>
          </div>
        ) : (
          <div style={{ display: "flex", gap: "16px", justifyContent: "center", marginBottom: "28px", flexWrap: "wrap" }}>
            <Link to="/login" className="btn-primary" style={{ display: "inline-block", width: "auto", padding: "12px 24px", textDecoration: "none" }}>
              🔑 Sign In / Register
            </Link>
            <Link to="/chat" className="btn-secondary" style={{ display: "inline-block", width: "auto", padding: "12px 24px", color: "var(--text-h)", borderColor: "var(--border)", textDecoration: "none" }}>
              💬 Launch Guest Chat
            </Link>
          </div>
        )}

        <div style={{ marginTop: "32px", display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(220px, 1fr))", gap: "16px", textAlign: "left" }}>
          <div style={{ padding: "16px", border: "1px solid var(--border)", borderRadius: "8px", background: "var(--bg)" }}>
            <h3 style={{ marginTop: 0 }}>⚡ Go Backend</h3>
            <p style={{ fontSize: "0.875rem", color: "var(--text)", margin: 0 }}>Powered by Go on Azure App Service at api.trygochat.tech.</p>
          </div>
          <div style={{ padding: "16px", border: "1px solid var(--border)", borderRadius: "8px", background: "var(--bg)" }}>
            <h3 style={{ marginTop: 0 }}>☁️ Azure Static Web Apps</h3>
            <p style={{ fontSize: "0.875rem", color: "var(--text)", margin: 0 }}>Fast SPA hosting with SPA fallback routing enabled.</p>
          </div>
          <div style={{ padding: "16px", border: "1px solid var(--border)", borderRadius: "8px", background: "var(--bg)" }}>
            <h3 style={{ marginTop: 0 }}>🔐 JWT Authentication</h3>
            <p style={{ fontSize: "0.875rem", color: "var(--text)", margin: 0 }}>Secure user registration, email verification, and tokens.</p>
          </div>
        </div>
      </div>
    </main>
  );
}
