import { useEffect, useRef } from "react";
import { Link } from "react-router-dom";

export default function Home({ currentUser, onOpenAuthModal }) {
  const featuresRef = useRef(null);

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            entry.target.classList.add("is-visible");
          }
        });
      },
      { threshold: 0.2 }
    );

    const boxes = featuresRef.current?.querySelectorAll(".feature-box");
    boxes?.forEach((box) => observer.observe(box));

    return () => {
      boxes?.forEach((box) => observer.unobserve(box));
    };
  }, []);

  return (
    <div className="app-home-page">
      <section className="hero-section">
        <div className="hero-content">
          <h1 className="hero-title">
            <div className="hero-brush-line brush-1"></div>
            <div className="hero-brush-line brush-2"></div>
            Go<span>Chat</span>
          </h1>
          <p className="hero-description">
            A real-time messaging application connected to a Go server on Azure App Service and deployed via Azure Static Web Apps.
          </p>
          
          <div className="hero-actions">
            {currentUser ? (
              <>
                <Link to="/chat" className="btn-primary" style={{ textDecoration: "none" }}>
                  Open Chat Interface
                </Link>
                <button
                  type="button"
                  className="btn-secondary"
                  onClick={onOpenAuthModal}
                >
                  Account Details
                </button>
              </>
            ) : (
              <>
                <button
                  type="button"
                  className="btn-primary"
                  onClick={() => onOpenAuthModal("login")}
                >
                  Log In
                </button>
                <button
                  type="button"
                  className="btn-secondary"
                  onClick={() => onOpenAuthModal("register")}
                >
                  Register
                </button>
              </>
            )}
          </div>
        </div>
      </section>

      <section className="features-section" ref={featuresRef}>
        <div className="features-grid">
          <div className="feature-box">
            <h3>Go Backend</h3>
            <p>
              Hosted on Azure App Service at api.trygochat.tech.
            </p>
          </div>
          <div className="feature-box">
            <h3>Azure Static Web Apps</h3>
            <p>
              Single Page Application with navigation fallback routing.
            </p>
          </div>
          <div className="feature-box">
            <h3>JWT Authentication</h3>
            <p>
              Token-based user session handling and email verification.
            </p>
          </div>
        </div>
      </section>

      <footer className="app-footer">
        <p>&copy; 2026 GoChat&trade; - Powered by Go</p>
        <p>Sergio Gomez</p>
      </footer>
    </div>
  );
}
