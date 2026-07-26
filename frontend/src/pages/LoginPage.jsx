import Auth from "../components/Auth";
import UserProfile from "../components/UserProfile";

export default function LoginPage({ currentUser, onLoginSuccess, onLogout }) {
  return (
    <main className="page-container">
      {currentUser ? (
        <UserProfile user={currentUser} onLogout={onLogout} />
      ) : (
        <div className="auth-wrapper">
          <Auth onLoginSuccess={onLoginSuccess} />
        </div>
      )}
    </main>
  );
}
