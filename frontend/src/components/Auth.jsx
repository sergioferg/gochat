import { useState } from "react";
import { loginUser, registerUser, verifyUser } from "../api";

export default function Auth({ onLoginSuccess }) {
  const [mode, setMode] = useState("login"); // 'login' | 'register' | 'verify'
  
  // Login state
  const [loginEmail, setLoginEmail] = useState("");
  const [loginPassword, setLoginPassword] = useState("");

  // Register state
  const [regNickname, setRegNickname] = useState("");
  const [regEmail, setRegEmail] = useState("");
  const [regPassword, setRegPassword] = useState("");
  const [regConfirmPassword, setRegConfirmPassword] = useState("");

  // Verify state
  const [verifyToken, setVerifyToken] = useState("");

  // Status state
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [successMsg, setSuccessMsg] = useState("");

  const clearMessages = () => {
    setError("");
    setSuccessMsg("");
  };

  const handleModeSwitch = (newMode) => {
    clearMessages();
    setMode(newMode);
  };

  const handleLogin = async (e) => {
    e.preventDefault();
    clearMessages();

    if (!loginEmail || !loginPassword) {
      setError("Please fill in all fields.");
      return;
    }

    setLoading(true);
    try {
      const user = await loginUser(loginEmail, loginPassword);
      setSuccessMsg("Logged in successfully!");
      if (onLoginSuccess) {
        onLoginSuccess(user);
      }
    } catch (err) {
      setError(err.message || "Failed to log in");
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async (e) => {
    e.preventDefault();
    clearMessages();

    if (!regNickname || !regEmail || !regPassword) {
      setError("Please fill in all required fields.");
      return;
    }

    if (regPassword !== regConfirmPassword) {
      setError("Passwords do not match.");
      return;
    }

    setLoading(true);
    try {
      const res = await registerUser(regNickname, regEmail, regPassword);
      setSuccessMsg(
        `Account created successfully for ${res.nickname || regNickname}! Please check your email for the verification token.`
      );
      // Automatically switch to verify mode or pre-fill email
      setMode("verify");
    } catch (err) {
      setError(err.message || "Failed to register account");
    } finally {
      setLoading(false);
    }
  };

  const handleVerify = async (e) => {
    e.preventDefault();
    clearMessages();

    if (!verifyToken.trim()) {
      setError("Please enter a verification token.");
      return;
    }

    setLoading(true);
    try {
      const res = await verifyUser(verifyToken.trim());
      setSuccessMsg(res?.message || "Account verified successfully! You can now log in.");
      setMode("login");
    } catch (err) {
      setError(err.message || "Failed to verify email");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-card">
      <div className="auth-tabs">
        <button
          type="button"
          className={mode === "login" ? "active" : ""}
          onClick={() => handleModeSwitch("login")}
        >
          Login
        </button>
        <button
          type="button"
          className={mode === "register" ? "active" : ""}
          onClick={() => handleModeSwitch("register")}
        >
          Register
        </button>
        <button
          type="button"
          className={mode === "verify" ? "active" : ""}
          onClick={() => handleModeSwitch("verify")}
        >
          Verify Email
        </button>
      </div>

      {error && <div className="alert alert-error">{error}</div>}
      {successMsg && <div className="alert alert-success">{successMsg}</div>}

      {mode === "login" && (
        <form onSubmit={handleLogin} className="auth-form">
          <h2>Login</h2>
          <div className="form-group">
            <label htmlFor="login-email">Email</label>
            <input
              id="login-email"
              type="email"
              placeholder="user@example.com"
              value={loginEmail}
              onChange={(e) => setLoginEmail(e.target.value)}
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="login-password">Password</label>
            <input
              id="login-password"
              type="password"
              placeholder="Enter your password"
              value={loginPassword}
              onChange={(e) => setLoginPassword(e.target.value)}
              required
            />
          </div>

          <button type="submit" disabled={loading} className="btn-primary">
            {loading ? "Logging in..." : "Log In"}
          </button>
        </form>
      )}

      {mode === "register" && (
        <form onSubmit={handleRegister} className="auth-form">
          <h2>Create Account</h2>
          <div className="form-group">
            <label htmlFor="reg-nickname">Nickname</label>
            <input
              id="reg-nickname"
              type="text"
              placeholder="e.g. alice"
              value={regNickname}
              onChange={(e) => setRegNickname(e.target.value)}
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="reg-email">Email</label>
            <input
              id="reg-email"
              type="email"
              placeholder="user@example.com"
              value={regEmail}
              onChange={(e) => setRegEmail(e.target.value)}
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="reg-password">Password</label>
            <input
              id="reg-password"
              type="password"
              placeholder="Password"
              value={regPassword}
              onChange={(e) => setRegPassword(e.target.value)}
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="reg-confirm-password">Confirm Password</label>
            <input
              id="reg-confirm-password"
              type="password"
              placeholder="Confirm password"
              value={regConfirmPassword}
              onChange={(e) => setRegConfirmPassword(e.target.value)}
              required
            />
          </div>

          <button type="submit" disabled={loading} className="btn-primary">
            {loading ? "Registering..." : "Register"}
          </button>
        </form>
      )}

      {mode === "verify" && (
        <form onSubmit={handleVerify} className="auth-form">
          <h2>Verify Account</h2>
          <p className="form-help">
            Enter the verification token sent to your email to activate your account.
          </p>

          <div className="form-group">
            <label htmlFor="verify-token">Verification Token</label>
            <input
              id="verify-token"
              type="text"
              placeholder="e.g. verify_token_12345"
              value={verifyToken}
              onChange={(e) => setVerifyToken(e.target.value)}
              required
            />
          </div>

          <button type="submit" disabled={loading} className="btn-primary">
            {loading ? "Verifying..." : "Verify Token"}
          </button>
        </form>
      )}
    </div>
  );
}
