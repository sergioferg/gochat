import { useEffect, useState } from "react";
import {
  fetchUserProfile,
  unfriendUser,
  blockUser,
  unblockUser,
  sendRequest,
} from "../api";

export default function UserProfileModal({ userId, onClose, onRelationshipChange }) {
  const [profile, setProfile] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [actionLoading, setActionLoading] = useState(false);

  useEffect(() => {
    if (!userId) return;
    async function loadProfile() {
      setLoading(true);
      setError("");
      try {
        const data = await fetchUserProfile(userId);
        setProfile(data);
      } catch (err) {
        setError(err.message || "Failed to load user profile");
      } finally {
        setLoading(false);
      }
    }
    loadProfile();
  }, [userId]);

  const isDeleted = profile?.status === "deleted" || profile?.nickname?.startsWith("deleted_");

  const displayName = isDeleted
    ? "Deleted Account"
    : profile?.nickname || profile?.real_name || "User";

  const handleUnfriend = async () => {
    setActionLoading(true);
    try {
      await unfriendUser(userId);
      setProfile((prev) => ({ ...prev, relationship_status: "none" }));
      if (onRelationshipChange) onRelationshipChange();
    } catch (err) {
      setError(err.message || "Failed to unfriend user");
    } finally {
      setActionLoading(false);
    }
  };

  const handleBlock = async () => {
    setActionLoading(true);
    try {
      await blockUser(userId);
      setProfile((prev) => ({ ...prev, relationship_status: "blocked_by_me" }));
      if (onRelationshipChange) onRelationshipChange();
    } catch (err) {
      setError(err.message || "Failed to block user");
    } finally {
      setActionLoading(false);
    }
  };

  const handleUnblock = async () => {
    setActionLoading(true);
    try {
      await unblockUser(userId);
      setProfile((prev) => ({ ...prev, relationship_status: "none" }));
      if (onRelationshipChange) onRelationshipChange();
    } catch (err) {
      setError(err.message || "Failed to unblock user");
    } finally {
      setActionLoading(false);
    }
  };

  const handleSendRequest = async () => {
    setActionLoading(true);
    try {
      await sendRequest(userId);
      setProfile((prev) => ({ ...prev, relationship_status: "pending_sent" }));
      if (onRelationshipChange) onRelationshipChange();
    } catch (err) {
      setError(err.message || "Failed to send friend request");
    } finally {
      setActionLoading(false);
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()} style={{ maxWidth: "420px" }}>
        <div className="modal-header">
          <span className="modal-title">User Profile</span>
          <button type="button" className="modal-close-btn" onClick={onClose} aria-label="Close">
            ✕
          </button>
        </div>

        {error && <div className="alert alert-error" style={{ marginBottom: "12px" }}>{error}</div>}

        {loading ? (
          <div className="search-loading" style={{ padding: "24px 0", justifyContent: "center" }}>
            <span className="spinner-lg" />
          </div>
        ) : !profile ? (
          <p className="placeholder-text" style={{ padding: "16px 0", textAlign: "center" }}>
            User not found.
          </p>
        ) : (
          <div className="user-profile-card" style={{ padding: "8px 0" }}>
            <div style={{ display: "flex", alignItems: "center", gap: "16px", marginBottom: "16px" }}>
              <div className="chat-item-avatar" style={{ width: "52px", height: "52px", fontSize: "1.3rem" }}>
                {displayName.charAt(0).toUpperCase()}
              </div>
              <div>
                <h3 style={{ margin: "0 0 4px 0", fontSize: "var(--font-size-lg)" }}>{displayName}</h3>
                {!isDeleted && profile.real_name && (
                  <p style={{ margin: 0, fontSize: "var(--font-size-xs)", color: "var(--color-text-muted)" }}>
                    {profile.real_name}
                  </p>
                )}
              </div>
            </div>

            <div className="profile-details" style={{ marginBottom: "20px" }}>
              <div className="profile-item">
                <strong>Status:</strong>{" "}
                <span className={`status-badge status-${profile.status || "active"}`}>
                  {isDeleted ? "deleted" : profile.status || "active"}
                </span>
              </div>
              <div className="profile-item">
                <strong>Relationship:</strong>{" "}
                <span style={{ textTransform: "capitalize", fontWeight: "600" }}>
                  {profile.relationship_status === "friend"
                    ? "Friends"
                    : profile.relationship_status === "blocked_by_me"
                    ? "Blocked by You"
                    : profile.relationship_status === "blocked_by_them"
                    ? "Blocked"
                    : profile.relationship_status === "pending_sent"
                    ? "Request Sent"
                    : profile.relationship_status === "pending_received"
                    ? "Pending Request Received"
                    : profile.relationship_status === "self"
                    ? "You"
                    : "Not Friends"}
                </span>
              </div>
            </div>

            {!isDeleted && profile.relationship_status !== "self" && (
              <div style={{ display: "flex", gap: "10px", flexWrap: "wrap" }}>
                {profile.relationship_status === "friend" && (
                  <>
                    <button
                      type="button"
                      className="btn-danger-outline"
                      style={{ flex: 1 }}
                      disabled={actionLoading}
                      onClick={handleUnfriend}
                    >
                      {actionLoading ? <span className="spinner" /> : "Unfriend"}
                    </button>
                    <button
                      type="button"
                      className="btn-danger"
                      style={{ flex: 1 }}
                      disabled={actionLoading}
                      onClick={handleBlock}
                    >
                      {actionLoading ? <span className="spinner" /> : "Block User"}
                    </button>
                  </>
                )}

                {profile.relationship_status === "blocked_by_me" && (
                  <button
                    type="button"
                    className="btn-primary"
                    style={{ flex: 1 }}
                    disabled={actionLoading}
                    onClick={handleUnblock}
                  >
                    {actionLoading ? <span className="spinner" /> : "Unblock User"}
                  </button>
                )}

                {profile.relationship_status === "none" && (
                  <>
                    <button
                      type="button"
                      className="btn-primary"
                      style={{ flex: 1 }}
                      disabled={actionLoading}
                      onClick={handleSendRequest}
                    >
                      {actionLoading ? <span className="spinner" /> : "Send Friend Request"}
                    </button>
                    <button
                      type="button"
                      className="btn-danger-outline"
                      disabled={actionLoading}
                      onClick={handleBlock}
                    >
                      Block
                    </button>
                  </>
                )}

                {profile.relationship_status === "pending_sent" && (
                  <button type="button" className="btn-secondary" style={{ flex: 1 }} disabled>
                    Request Sent
                  </button>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
