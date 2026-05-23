# Fetitele Powerpuff: Secure Medical Streaming & Analytics

![Project Logo](web/client/src/assets/logo.svg)

## 🛡️ Project Overview

**Fetitele Powerpuff** (Secure Systems Project) is a sophisticated, end-to-end secure system designed for the capture, transmission, and intelligent analysis of medical documentation. Built with a "Security-First" philosophy, it leverages **mTLS (Mutual TLS)** and **JWT** to ensure the integrity and confidentiality of sensitive medical data as it flows from edge devices to a centralized analytics dashboard.

The system is uniquely equipped with an **AI Governance engine** that audits architectural compliance and a dedicated **gRPC OCR service** optimized for medical documents in both English and Romanian.

---

## ✨ Key Features

### 🔐 Multi-Layered Security
- **mTLS Communication:** All device-to-server (MQTT) and service-to-service (gRPC) communication is encrypted and authenticated via Mutual TLS.
- **JWT Authentication:** Secure user and device registration with JSON Web Token-based access control.
- **RBAC Zones:** Granular Role-Based Access Control enforced at both the application and AI governance layers.

### 📸 Intelligent Image Pipeline
- **Distributed Capture:** Supports medical image ingestion from Android mobile applications.
- **Live Streaming:** Real-time "Live Mode" for continuous monitoring or single-shot "Normal Mode" capture.
- **Advanced OCR Engine:** A dedicated Go-based service using Tesseract to extract medical insights with multi-language support (English/Romanian).

### 📊 Data Analytics & Dashboard
- **Medical Insights:** Automatic classification of medical opinions (APT, Inapt, etc.) and control types (Periodic, Employment, etc.).
- **Full-Text Search:** Search through vast medical records using the text extracted via OCR.
- **Interactive Visualizations:** Real-time charts and statistics for medical data distribution.

### 🤖 AI Governance & Auditing
- **CrewAI Orchestration:** Autonomous agents manage architectural guidelines and project workflows.
- **Immutable Audit Logs:** Every action taken by the AI system is recorded in a secure, local audit trail for transparency and accountability.
- **Architectural Guardrails:** Automated enforcement of project-wide development standards.

---

## 🏗️ System Architecture

The project follows a microservices-inspired architecture containerized with Docker:

```mermaid
graph TD
    A[Mobile App] -- mTLS / MQTT --> B(Mosquitto Broker)
    B -- mTLS --> C[Go Backend Server]
    C -- gRPC / mTLS --> D[OCR Service]
    C -- SQL --> E[(PostgreSQL)]
    F[React Web Dashboard] -- HTTP / JWT --> C
    G[AI Governance Engine] -- Audit / RBAC --> H[Project Workspace]
```

---

## 💻 Technology Stack

| Layer | Technologies |
|---|---|
| **Frontend** | React, TypeScript, Vite, TailwindCSS, Chart.js |
| **Backend** | Go (Golang), GORM, Paho MQTT |
| **OCR Service** | Go, gRPC, Tesseract (Gosseract) |
| **Mobile** | Android (Kotlin/Java) |
| **AI/Orchestration** | Python, CrewAI, GitPython |
| **Infrastructure** | Docker Compose, Mosquitto MQTT, PostgreSQL |

---

## 🚀 Getting Started

### Prerequisites
- **Docker & Docker Compose**
- **Node.js** (v24+ recommended)
- **Python** (3.10+ for AI Governance)
- **Android Studio** (for mobile development)

### Quick Start (Development Mode)
1. **Clone the repository:**
   ```bash
   git clone https://github.com/your-org/ss-project.git
   cd ss-project/web
   ```

2. **Generate Security Certificates:**
   ```bash
   ./scripts/gen-ca.sh
   ./scripts/gen-android-stores.sh
   ```

3. **Launch the Stack:**
   ```bash
   ./start.sh
   ```
   *This script installs dependencies, starts Docker containers (API, DB, MQTT, OCR), and launches the Vite dev server.*

4. **Access the Dashboard:**
   Navigate to `http://localhost:5173`. Use the default credentials or register a new account.

---

## 🛡️ AI Governance & Ethics

The `ai-governance/` module is a core component of the project. It ensures that any AI-driven modifications or analyses adhere to strict safety and architectural standards:
- **Zone Control:** Restricts file system access based on predefined "Red Zones" (secrets, logs) and "Developer Zones" (src, web).
- **Accountability:** Every agent thought process and tool invocation is logged to `ai-governance/audit.log`.
- **Integrity:** Prevents unauthorized writes to sensitive configuration files.

---

## 📄 License

This project is developed for the **Security of Systems** course. All rights reserved.

---

*Built with ❤️ by the **Fetitele Powerpuff** Team.*
