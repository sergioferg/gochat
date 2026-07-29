import { useState } from "react";
import { updateUser, fetchCurrentUser } from "../api";

function calculateAge(dobString) {
  if (!dobString) return 0;
  const birthDate = new Date(dobString);
  if (isNaN(birthDate.getTime())) return 0;
  const today = new Date();
  let age = today.getFullYear() - birthDate.getFullYear();
  const monthDiff = today.getMonth() - birthDate.getMonth();
  if (monthDiff < 0 || (monthDiff === 0 && today.getDate() < birthDate.getDate())) {
    age--;
  }
  return age;
}

export default function CompleteProfile({ currentUser, onComplete }) {
  const [realName, setRealName] = useState(currentUser?.real_name || "");
  const [dateOfBirth, setDateOfBirth] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const age = calculateAge(dateOfBirth);
  const isUnderage = dateOfBirth && age < 18;

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError("");

    if (!realName || !dateOfBirth) {
      setError("Please fill out all fields.");
      return;
    }

    if (isUnderage) {
      setError("You must be at least 18 years old to use GoChat.");
      return;
    }

    setLoading(true);
    try {
      await updateUser({
        real_name: realName,
        date_of_birth: dateOfBirth,
      });

      const updatedUser = await fetchCurrentUser();
      onComplete(updatedUser);
    } catch (err) {
      setError(err.message || "Failed to update profile.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <main className="page-container">
      <div className="auth-wrapper" style={{ margin: "auto", maxWidth: "450px", marginTop: "var(--space-8)" }}>
        <div className="auth-card">
          <form onSubmit={handleSubmit} className="auth-form">
            <h2>Complete Your Profile</h2>
            <p style={{ marginBottom: "var(--space-4)", color: "var(--color-text-muted)" }}>
              Please provide the following details to finish setting up your account before you can start chatting.
            </p>
            
            {error && <div className="alert alert-error">{error}</div>}

            <div className="form-group">
              <label htmlFor="cp-real-name">Real Name</label>
              <input
                id="cp-real-name"
                type="text"
                placeholder="e.g. John Doe"
                value={realName}
                onChange={(e) => setRealName(e.target.value)}
                required
              />
            </div>

            <div className="form-group">
              <label htmlFor="cp-dob">Date of Birth</label>
              <input
                id="cp-dob"
                type="date"
                value={dateOfBirth}
                onChange={(e) => setDateOfBirth(e.target.value)}
                required
              />
              {isUnderage && (
                <span className="field-error-text" style={{ display: "block", marginTop: "var(--space-1)" }}>
                  You must be at least 18 years old.
                </span>
              )}
            </div>

            <button type="submit" disabled={loading || isUnderage} className="btn-primary">
              {loading && <span className="spinner" />}
              <span>{loading ? "Saving..." : "Save & Continue"}</span>
            </button>
          </form>
        </div>
      </div>
    </main>
  );
}
