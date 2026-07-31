import { useEffect, useState } from "react";
import { fetchActiveSessions, revokeSession, deleteUserAccount } from "../api";

function formatUserAgent(ua) {
  if (!ua) return "Unknown Device";
  let browser = "Web Browser";
  let os = "";

  if (ua.includes("Firefox")) browser = "Firefox";
  else if (ua.includes("Chrome") || ua.includes("CriOS")) browser = "Chrome";
  else if (ua.includes("Safari") && !ua.includes("Chrome")) browser = "Safari";
  else if (ua.includes("Edg")) browser = "Edge";

  if (ua.includes("Macintosh") || ua.includes("Mac OS")) os = "macOS";
  else if (ua.includes("Windows")) os = "Windows";
  else if (ua.includes("Linux")) os = "Linux";
  else if (ua.includes("iPhone") || ua.includes("iPad")) os = "iOS";
  else if (ua.includes("Android")) os = "Android";

  return os ? `${browser} on ${os}` : browser;
}

export default function UserProfile({ user, onLogout }) {
  const [view, setView] = useState("profile"); // 'profile' | 'delete_confirm'
  const [sessions, setSessions] = useState([]);
  const [loadingSessions, setLoadingSessions] = useState(true);
  const [sessionError, setSessionError] = useState("");
  const [revokingId, setRevokingId] = useState(null);
  const [revokingAll, setRevokingAll] = useState(false);

  // Delete account state
  const [deleteInput, setDeleteInput] = useState("");
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState("");

  useEffect(() => {
    async function loadSessions() {
      setLoadingSessions(true);
      setSessionError("");
      try {
        const data = await fetchActiveSessions();
        setSessions(data || []);
      } catch (err) {
        setSessionError(err.message || "Failed to load active sessions");
      } finally {
        setLoadingSessions(false);
      }
    }

    loadSessions();
  }, []);

  const handleRevokeSingle = async (sessionId) => {
    setRevokingId(sessionId);
    setSessionError("");
    const targetSession = sessions.find((s) => s.id === sessionId);
    try {
      await revokeSession(sessionId);
      if (targetSession?.is_current) {
        onLogout();
        return;
      }
      setSessions((prev) => prev.filter((s) => s.id !== sessionId));
    } catch (err) {
      setSessionError(err.message || "Failed to revoke session");
    } finally {
      setRevokingId(null);
    }
  };

  const handleRevokeAll = async () => {
    if (!sessions.length) return;
    setRevokingAll(true);
    setSessionError("");
    try {
      await Promise.all(sessions.map((s) => revokeSession(s.id)));
      setSessions([]);
      onLogout();
    } catch (err) {
      setSessionError(err.message || "Failed to terminate all sessions");
    } finally {
      setRevokingAll(false);
    }
  };

  const handleDeleteAccount = async () => {
    if (deleteInput !== "DELETE") return;
    setDeleting(true);
    setDeleteError("");
    try {
      await deleteUserAccount();
      onLogout();
    } catch (err) {
      setDeleteError(err.message || "Failed to delete account");
      setDeleting(false);
    }
  };

  if (!user) return null;

  if (view === "delete_confirm") {
    return (
      <div className="user-profile-card">
        <div className="profile-header" style={{ marginBottom: "var(--space-3)" }}>
          <h2 style={{ color: "var(--color-error-text, #ef4444)" }}>⚠️ Delete Account Warning</h2>
        </div>

        {deleteError && (
          <div className="alert alert-error" style={{ marginBottom: "var(--space-3)" }}>
            {deleteError}
          </div>
        )}

        <div className="delete-warning-card">
          <p style={{ fontWeight: "600", marginBottom: "8px" }}>
            Are you sure you want to permanently delete your account?
          </p>
          <ul style={{ margin: "0", paddingLeft: "20px", display: "flex", flexDirection: "column", gap: "6px" }}>
            <li><strong>Permanent Data Loss:</strong> Your profile, nickname, email, and credentials will be erased.</li>
            <li><strong>Session Revocation:</strong> All active sessions across all devices will be terminated immediately.</li>
            <li><strong>Chat & Request Clean-Up:</strong> You will be removed from all active chats and pending user requests.</li>
            <li><strong>Irreversible:</strong> This action <em>cannot</em> be undone.</li>
          </ul>
        </div>

        <div style={{ marginTop: "16px" }}>
          <label htmlFor="delete-confirm-input" style={{ display: "block", fontSize: "var(--font-size-xs)", fontWeight: "600", marginBottom: "4px" }}>
            Type <code style={{ color: "#ef4444", background: "rgba(239,68,68,0.1)", padding: "2px 6px", borderRadius: "4px" }}>DELETE</code> to confirm:
          </label>
          <input
            id="delete-confirm-input"
            type="text"
            className="delete-confirm-input"
            value={deleteInput}
            onChange={(e) => setDeleteInput(e.target.value)}
            placeholder='Type "DELETE" here'
            autoFocus
          />
        </div>

        <div style={{ display: "flex", gap: "12px", marginTop: "16px" }}>
          <button
            type="button"
            className="btn-danger"
            style={{ flex: 1 }}
            disabled={deleteInput !== "DELETE" || deleting}
            onClick={handleDeleteAccount}
          >
            {deleting ? "Deleting Account..." : "Permanently Delete Account"}
          </button>
          <button
            type="button"
            className="btn-secondary"
            onClick={() => {
              setView("profile");
              setDeleteInput("");
              setDeleteError("");
            }}
            disabled={deleting}
          >
            Cancel
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="user-profile-card">
      <div className="profile-header">
        <h2>{user.nickname || user.email}</h2>
        <button type="button" onClick={onLogout} className="btn-danger">
          Log Out
        </button>
      </div>

      <div className="profile-details">
        <div className="profile-item">
          <strong>Nickname:</strong> {user.nickname || "N/A"}
        </div>
        <div className="profile-item">
          <strong>Real Name:</strong> {user.real_name || "N/A"}
        </div>
        <div className="profile-item">
          <strong>Date of Birth:</strong> {user.date_of_birth || "N/A"}
        </div>
        <div className="profile-item">
          <strong>Email:</strong> {user.email || "N/A"}
        </div>
        <div className="profile-item">
          <strong>Status:</strong>{" "}
          <span className={`status-badge status-${user.status || "active"}`}>
            {user.status || "active"}
          </span>
        </div>
      </div>

      {/* Active Sessions Section */}
      <div className="sessions-section">
        <div className="sessions-header">
          <h3>Active Sessions ({sessions.length})</h3>
          {sessions.length > 0 && (
            <button
              type="button"
              className="btn-danger-outline"
              onClick={handleRevokeAll}
              disabled={revokingAll || loadingSessions}
            >
              {revokingAll ? "Terminating All..." : "Terminate All Sessions"}
            </button>
          )}
        </div>

        {sessionError && (
          <div className="alert alert-error" style={{ marginTop: "var(--space-3)", marginBottom: "var(--space-3)" }}>
            {sessionError}
          </div>
        )}

        {loadingSessions ? (
          <div className="sessions-loading">
            <span className="spinner" /> <span>Loading active sessions...</span>
          </div>
        ) : sessions.length === 0 ? (
          <p className="sessions-empty">No active sessions found.</p>
        ) : (
          <ul className="sessions-list">
            {sessions.map((sess) => (
              <li key={sess.id} className={`session-card ${sess.is_current ? "session-card-current" : ""}`}>
                <div className="session-info">
                  <div className="session-device">
                    <span className="session-icon">💻</span>
                    <strong>{formatUserAgent(sess.user_agent)}</strong>
                    {sess.is_current && (
                      <span className="badge-current-session" style={{ marginLeft: "8px", fontSize: "12px", color: "var(--color-primary, #6366f1)", fontWeight: "600" }}>
                        (Current Session)
                      </span>
                    )}
                  </div>
                  <div className="session-meta">
                    <span>IP: {sess.ip_address || "Unknown"}</span>
                    <span> • </span>
                    <span>
                      Created: {sess.created_at ? new Date(sess.created_at).toLocaleString() : "Unknown"}
                    </span>
                  </div>
                </div>

                <button
                  type="button"
                  className="btn-terminate-sm"
                  onClick={() => handleRevokeSingle(sess.id)}
                  disabled={revokingId === sess.id || revokingAll}
                >
                  {revokingId === sess.id ? "Terminating..." : "Terminate"}
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      {/* Danger Zone */}
      <div className="danger-zone" style={{ marginTop: "24px", paddingTop: "16px", borderTop: "1px solid var(--color-border)" }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <div>
            <h4 style={{ margin: "0 0 4px 0", color: "var(--color-error-text, #ef4444)", fontSize: "var(--font-size-sm)" }}>
              Danger Zone
            </h4>
            <p style={{ margin: 0, fontSize: "var(--font-size-xs)", color: "var(--color-text-muted)" }}>
              Permanently delete your account and all associated data.
            </p>
          </div>
          <button
            type="button"
            className="btn-danger-outline"
            onClick={() => setView("delete_confirm")}
          >
            Delete Account
          </button>
        </div>
      </div>
    </div>
  );
}
