import { useEffect, useState } from "react";
import { fetchActiveSessions, revokeSession } from "../api";

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
  const [sessions, setSessions] = useState([]);
  const [loadingSessions, setLoadingSessions] = useState(true);
  const [sessionError, setSessionError] = useState("");
  const [revokingId, setRevokingId] = useState(null);
  const [revokingAll, setRevokingAll] = useState(false);

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
    try {
      await revokeSession(sessionId);
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
    } catch (err) {
      setSessionError(err.message || "Failed to terminate all sessions");
    } finally {
      setRevokingAll(false);
    }
  };

  if (!user) return null;

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
              <li key={sess.id} className="session-card">
                <div className="session-info">
                  <div className="session-device">
                    <span className="session-icon">💻</span>
                    <strong>{formatUserAgent(sess.user_agent)}</strong>
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
    </div>
  );
}
