export const API_BASE =
  import.meta.env.VITE_API_BASE ||
  (import.meta.env.PROD
    ? "https://api.trygochat.tech"
    : "http://localhost:8080");

export const WS_BASE =
  import.meta.env.VITE_WS_BASE ||
  (import.meta.env.PROD
    ? "wss://api.trygochat.tech"
    : "ws://localhost:8080");

export function getWebSocketUrl(path = "/ws") {
  const cleanPath = path.startsWith("/") ? path : `/${path}`;
  return `${WS_BASE}${cleanPath}`;
}

let accessToken = localStorage.getItem("token") || "";

export const setToken = (token) => {
  accessToken = token || "";
  if (token) {
    localStorage.setItem("token", token);
  } else {
    localStorage.removeItem("token");
  }
};

export const getToken = () => localStorage.getItem("token") || accessToken || "";

async function request(endpoint, options = {}) {
  const token = getToken();
  const headers = {
    "Content-Type": "application/json",
    ...options.headers,
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
    credentials: "include",
  });

  if (response.status === 204) {
    return null;
  }

  const data = await response.json().catch(() => ({}));

  if (!response.ok) {
    const errorMsg = data.error || `Request failed with status ${response.status}`;
    throw new Error(errorMsg);
  }

  return data;
}

export async function loginUser(email, password) {
  const data = await request("/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
  if (data.token) {
    setToken(data.token);
  }
  return data;
}

export async function registerUser(nickname, email, password) {
  return await request("/users", {
    method: "POST",
    body: JSON.stringify({ nickname, email, password }),
  });
}

export async function fetchCurrentUser() {
  return await request("/me", {
    method: "GET",
  });
}

export async function verifyUser(token) {
  return await request("/verify", {
    method: "POST",
    body: JSON.stringify({ token }),
  });
}

export async function logoutUser() {
  try {
    await request("/logout", {
      method: "POST",
    });
  } finally {
    setToken("");
  }
}

export async function refreshToken() {
  const data = await request("/refresh", {
    method: "POST",
  });
  if (data && data.token) {
    setToken(data.token);
  }
  return data;
}

export async function sendMessage(text) {
  return await request("/messages", {
    method: "POST",
    body: JSON.stringify({ text }),
  });
}

export async function fetchActiveSessions() {
  const data = await request("/sessions", {
    method: "GET",
  });
  return data?.sessions || [];
}

export async function revokeSession(sessionId) {
  return await request(`/sessions/${sessionId}`, {
    method: "DELETE",
  });
}

