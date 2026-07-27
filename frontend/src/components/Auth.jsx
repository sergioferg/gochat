import { useState, useEffect } from "react";
import { loginUser, registerUser } from "../api";

export default function Auth({ onLoginSuccess, initialMode = "login" }) {
  const [mode, setMode] = useState(initialMode); // 'login' | 'register'

  useEffect(() => {
    setMode(initialMode);
  }, [initialMode]);
  
  // Login state
  const [loginEmail, setLoginEmail] = useState("");
  const [loginPassword, setLoginPassword] = useState("");

  // Register state
  const [regNickname, setRegNickname] = useState("");
  const [regEmail, setRegEmail] = useState("");
  const [regPassword, setRegPassword] = useState("");
  const [regConfirmPassword, setRegConfirmPassword] = useState("");

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
        `Account created successfully for ${res?.nickname || regNickname}. Please check your email to complete verification.`
      );
    } catch (err) {
      setError(err.message || "Failed to register account");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="auth-card">
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
            {loading && <span className="spinner" />}
            <span>{loading ? "Logging in..." : "Log In"}</span>
          </button>

          <div className="auth-divider">or</div>

          <a
            href="https://api.trygochat.tech/oauth/github/login"
            className="btn-github"
          >
            <svg width="18" height="18" viewBox="0 0 19 19" fill="currentColor" aria-hidden="true">
              <path fillRule="evenodd" clipRule="evenodd" d="M9.356 1.85C5.05 1.85 1.57 5.356 1.57 9.694a7.84 7.84 0 0 0 5.324 7.44c.387.079.528-.168.528-.376 0-.182-.013-.805-.013-1.454-2.165.467-2.616-.935-2.616-.935-.349-.91-.864-1.143-.864-1.143-.71-.48.051-.48.051-.48.787.051 1.2.805 1.2.805.695 1.194 1.817.857 2.268.649.064-.507.27-.857.49-1.052-1.728-.182-3.545-.857-3.545-3.87 0-.857.31-1.558.8-2.104-.078-.195-.349-1 .077-2.078 0 0 .657-.208 2.14.805a7.5 7.5 0 0 1 1.946-.26c.657 0 1.328.092 1.946.26 1.483-1.013 2.14-.805 2.14-.805.426 1.078.155 1.883.078 2.078.502.546.799 1.247.799 2.104 0 3.013-1.818 3.675-3.558 3.87.284.247.528.714.528 1.454 0 1.052-.012 1.896-.012 2.156 0 .208.142.455.528.377a7.84 7.84 0 0 0 5.324-7.441c.013-4.338-3.48-7.844-7.773-7.844" />
            </svg>
            <span>Login with: Github</span>
          </a>

          <p className="auth-switch-text">
            Don't have an account?
            <button
              type="button"
              className="auth-switch-btn"
              onClick={() => handleModeSwitch("register")}
            >
              Register
            </button>
          </p>
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
            {loading && <span className="spinner" />}
            <span>{loading ? "Registering..." : "Register"}</span>
          </button>

          <div className="auth-divider">or</div>

          <a
            href="https://api.trygochat.tech/oauth/github/login"
            className="btn-github"
          >
            <svg width="18" height="18" viewBox="0 0 19 19" fill="currentColor" aria-hidden="true">
              <path fillRule="evenodd" clipRule="evenodd" d="M9.356 1.85C5.05 1.85 1.57 5.356 1.57 9.694a7.84 7.84 0 0 0 5.324 7.44c.387.079.528-.168.528-.376 0-.182-.013-.805-.013-1.454-2.165.467-2.616-.935-2.616-.935-.349-.91-.864-1.143-.864-1.143-.71-.48.051-.48.051-.48.787.051 1.2.805 1.2.805.695 1.194 1.817.857 2.268.649.064-.507.27-.857.49-1.052-1.728-.182-3.545-.857-3.545-3.87 0-.857.31-1.558.8-2.104-.078-.195-.349-1 .077-2.078 0 0 .657-.208 2.14.805a7.5 7.5 0 0 1 1.946-.26c.657 0 1.328.092 1.946.26 1.483-1.013 2.14-.805 2.14-.805.426 1.078.155 1.883.078 2.078.502.546.799 1.247.799 2.104 0 3.013-1.818 3.675-3.558 3.87.284.247.528.714.528 1.454 0 1.052-.012 1.896-.012 2.156 0 .208.142.455.528.377a7.84 7.84 0 0 0 5.324-7.441c.013-4.338-3.48-7.844-7.773-7.844" />
            </svg>
            <span>Login with: Github</span>
          </a>

          <p className="auth-switch-text">
            Already have an account?
            <button
              type="button"
              className="auth-switch-btn"
              onClick={() => handleModeSwitch("login")}
            >
              Sign in
            </button>
          </p>
        </form>
      )}
    </div>
  );
}
