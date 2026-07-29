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
  const token = getToken();
  const cleanPath = path.startsWith("/") ? path : `/${path}`;
  const url = `${WS_BASE}${cleanPath}`;
  if (token && !url.includes("token=")) {
    const separator = url.includes("?") ? "&" : "?";
    return `${url}${separator}token=${encodeURIComponent(token)}`;
  }
  return url;
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

let isRefreshing = false;
let refreshSubscribers = [];

function subscribeTokenRefresh(cb) {
  refreshSubscribers.push(cb);
}

function onRefreshed(newToken) {
  refreshSubscribers.forEach((cb) => cb(newToken));
  refreshSubscribers = [];
}

async function request(endpoint, options = {}, isRetry = false) {
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

  if (
    response.status === 401 &&
    token &&
    !isRetry &&
    !["/login", "/refresh", "/users", "/verify"].includes(endpoint)
  ) {
    if (!isRefreshing) {
      isRefreshing = true;
      try {
        const refreshData = await refreshToken();
        isRefreshing = false;
        if (refreshData && refreshData.token) {
          onRefreshed(refreshData.token);
          return await request(endpoint, options, true);
        }
      } catch (refreshErr) {
        isRefreshing = false;
        refreshSubscribers = [];
        setToken("");
        throw refreshErr;
      }
    } else {
      return new Promise((resolve, reject) => {
        subscribeTokenRefresh(() => {
          request(endpoint, options, true).then(resolve).catch(reject);
        });
      });
    }
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
  if (["unverified", "deactivated", "deleted"].includes(data.status)) {
    throw new Error(`Account is ${data.status}`);
  }
  if (data.token) {
    setToken(data.token);
  }
  return data;
}

export async function registerUser(nickname, email, password, realName, dateOfBirth) {
  return await request("/users", {
    method: "POST",
    body: JSON.stringify({
      nickname,
      email,
      password,
      real_name: realName,
      date_of_birth: dateOfBirth,
    }),
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

export async function updateUser(updates) {
  return await request("/users", {
    method: "PATCH",
    body: JSON.stringify(updates),
  });
}

export async function searchUsers(query) {
  if (!query || !query.trim()) return [];
  const data = await request(`/users/search?q=${encodeURIComponent(query.trim())}`, {
    method: "GET",
  });
  return data?.users || [];
}

export async function sendRequest(targetUserId, initialMessage = "") {
  return await request("/requests", {
    method: "POST",
    body: JSON.stringify({
      target_user_id: targetUserId,
      initial_message: initialMessage ? initialMessage.trim() : undefined,
    }),
  });
}

export async function fetchPendingRequests() {
  const data = await request("/requests", {
    method: "GET",
  });
  return data?.requests || [];
}

export async function updateRequestAction(requestId, action) {
  return await request(`/requests/${requestId}`, {
    method: "PATCH",
    body: JSON.stringify({ action }),
  });
}

export async function fetchUserChats() {
  const data = await request("/chats", {
    method: "GET",
  });
  return Array.isArray(data) ? data : [];
}

