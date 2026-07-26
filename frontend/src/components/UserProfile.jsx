export default function UserProfile({ user, onLogout }) {
  if (!user) return null;

  return (
    <div className="user-profile-card">
      <div className="profile-header">
        <h2>Logged in as {user.nickname || user.email}</h2>
        <button type="button" onClick={onLogout} className="btn-secondary">
          Log Out
        </button>
      </div>

      <div className="profile-details">
        <div className="profile-item">
          <strong>Nickname:</strong> {user.nickname || "N/A"}
        </div>
        <div className="profile-item">
          <strong>Email:</strong> {user.email || "N/A"}
        </div>
        <div className="profile-item">
          <strong>Status:</strong>{" "}
          <span className={`status-badge status-${user.status}`}>
            {user.status || "active"}
          </span>
        </div>
        {user.id && (
          <div className="profile-item">
            <strong>ID:</strong> <code className="user-id">{user.id}</code>
          </div>
        )}
      </div>
    </div>
  );
}
