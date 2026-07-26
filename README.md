# GoChat

A high-performance, real-time messaging application featuring a Go backend and a React/Vite frontend, structured as a monorepo.

> ⚠️ **Important Note:** The backend is fully functional and deployed, but the frontend is currently a **work in progress (WIP)** and not yet ready for production use.

---

## Description

**GoChat** is a real-time chat application designed to provide reliable, low-latency communication across web clients. Built as a monorepo, it pairs an asynchronous Go backend with a modern React and Vite frontend interface.

### Tech Stack & Features

- **Go Backend:** Built using Go (Golang) with standard `net/http` routing for optimal performance, simplicity, and concurrency.
- **Real-Time Communication:** WebSockets enable instant, bi-directional message transmission between clients and server.
- **Message Delivery & Queuing:** Integrated **RabbitMQ** for reliable message delivery and asynchronous event handling.
- **Robust Authentication:** Complete security layer supporting **JWT (JSON Web Tokens)**, standard **Email/Password** authentication, and **GitHub OAuth** integration.
- **Frontend Engine:** Modern **React** web app built with **Vite** for fast module replacement and efficient builds.
- **Containerization:** The backend is containerized using **Docker** for consistent runtime environments and simplified deployment.

---

## Motivation

The primary motivation behind **GoChat** is to explore and construct a highly scalable, real-time messaging system leveraging Go's native concurrency models (goroutines and channels), while establishing end-to-end modern CI/CD deployment pipelines and cloud infrastructure practices on Microsoft Azure.

---

## Quick Start

Follow these steps to set up and run GoChat in your local development environment.

### Prerequisites

- [Go](https://go.dev/) (v1.20+ recommended) or [Docker](https://www.docker.com/)
- [Node.js](https://nodejs.org/) & `npm`

### 1. Clone the Repository

```bash
git clone https://github.com/sergioferg/gochat.git
cd gochat
```

### 2. Start the Go Backend

You can run the backend directly using Go, or build and run it using Docker:

#### Option A: Running with Go

```bash
go run main.go
```

#### Option B: Running with Docker

```bash
docker build -t gochat-backend .
docker run -p 8080:8080 --env-file .env gochat-backend
```

The Go backend server will start listening on port `8080` for REST and WebSocket connections.

### 3. Start the Vite Frontend

In a separate terminal, navigate to the frontend directory, install dependencies, and launch the Vite development server:

```bash
cd frontend
npm install
npm run dev
```

---

## Usage

### Architecture & Deployment Strategy

GoChat is architected as a decoupled monorepo, with separate cloud deployment targets for backend and frontend services:

- **Backend Deployment:** The containerized backend is deployed on **Azure App Service** behind a custom domain with automated SSL encryption via **App Service Managed Certificates**.
- **Frontend Deployment:** Deployed using **Azure Static Web Apps** with automated **GitHub Actions** CI/CD pipelines and attached to its own custom root domain.
- **Communication Protocol:** The frontend interacts with the Go backend via **CORS-enabled RESTful endpoints** for user management and authentication, and establishes long-lived **WebSocket** connections for instant message routing.

### API Documentation

For full details on authentication, REST endpoints, request/response bodies, and real-time WebSocket event formats, refer to the API documentation:

- 📖 **[API Reference Guide](docs/API.md)**
- 📄 **[OpenAPI 3.0 Specification](docs/openapi.yaml)**

---

## Contributing

Contributions, issues, and feature requests are welcome! Since the frontend is currently under active development, feedback and pull requests are especially helpful.

1. Fork the project repository.
2. Create a feature branch (`git checkout -b feature/my-new-feature`).
3. Commit your changes (`git commit -m 'Add feature/my-new-feature'`).
4. Push to your branch (`git push origin feature/my-new-feature`).
5. Open a Pull Request or report an Issue.
