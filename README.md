# issuesight

# IssueSight 🔭

> **Bridging the gap between "Good First Issues" and "Great First Contributions" via AI-driven mentorship.**

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Architecture](https://img.shields.io/badge/Architecture-Event--Driven-orange?logo=redis&logoColor=white)](https://redis.io/)
[![Database](https://img.shields.io/badge/Database-PostgreSQL-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Status](https://img.shields.io/badge/Status-Active_Development-green)]()

---

## 🎯 The Engineering Goal
**IssueSight** is a distributed, event-driven platform designed to solve a specific problem in the Open Source ecosystem: **Context Switching**.

Junior engineers often struggle to contribute not because they can't code, but because they lack the domain context of massive repositories. IssueSight ingests GitHub issues and uses LLMs to generate **"Context Bridges"**—breaking down complex tickets into freshman-level prerequisites, architectural summaries, and implementation guides.

**For Hiring Managers:** This project serves as a practical demonstration of my skills in:
* **Distributed Systems:** Decoupling ingestion (Writes) from serving (Reads).
* **Concurrency patterns in Go:** Using Goroutines and Channels for efficient data processing.
* **Database Design:** Implementing hybrid Relational/JSONB schemas in PostgreSQL.
* **Caching Strategies:** Utilizing the "Cache-Aside" pattern for high-read throughput.

---

## 🏗️ System Architecture
The system follows a vertical **Microservices Layering** pattern. Traffic flows from the Clients (Top) through the Go Middleware (Center) down to the Persistence Layer (Bottom).

```mermaid
flowchart TB
    %% --- GLOBAL STYLES ---
    classDef client fill:#fff3e0,stroke:#f57c00,stroke-width:2px,rx:10,ry:10;
    classDef external fill:#e3f2fd,stroke:#1565c0,stroke-width:2px,rx:10,ry:10;
    classDef go fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px,rx:5,ry:5;
    classDef data fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px,shape:cyl;
    
    %% --- 1. TOP LAYER: CLIENTS & EXTERNAL ---
    subgraph TopLayer [Sources & Clients]
        direction LR
        UserApp(User App):::client
        GitHub(GitHub API):::external
        OpenAI(OpenAI API):::external
    end

    %% --- 2. MIDDLE LAYER: GO SYSTEM ---
    subgraph GoSystem [Go Microservices System]
        direction TB
        Gateway[API Gateway]:::go
        Collector[Collector Service]:::go
        AIWorker[AI Worker]:::go
    end

    %% --- 3. BOTTOM LAYER: DATA ---
    subgraph DataLayer [Persistence Layer]
        direction TB
        RedisCache[(Redis Cache)]:::data
        Postgres[(PostgreSQL)]:::data
        RedisStream[(Redis Stream)]:::data
    end

    %% ====== CONNECTIONS ======

    %% FLOW A: User Traffic (Left Side)
    UserApp -->|1. GET /issue| Gateway
    Gateway -->|2. Check Cache| RedisCache
    RedisCache -.->|3. Miss| Gateway
    Gateway -->|4. Read DB| Postgres

    %% FLOW B: Ingestion Pipeline (Right Side)
    GitHub -->|A. Poll| Collector
    Collector -->|B. Pub Event| RedisStream
    RedisStream -.->|C. Consume| AIWorker
    AIWorker -->|D. Generate| OpenAI
    AIWorker -->|E. Write Data| Postgres

    %% --- LINK STYLES ---
    linkStyle 0,1,2,3 stroke:#f57c00,stroke-width:2px;   %% User Flow (Orange)
    linkStyle 4,5,6,7,8 stroke:#1565c0,stroke-width:2px; %% Ingestion Flow (Blue)
    ```

### 🔁 Data Flow Breakdown
1.  **Ingestion (The Write Path - Blue Lines):** A background `Collector` service polls GitHub and pushes raw events to a **Redis Stream**. This ensures that if the GitHub API is slow or rate-limited, it does not block the rest of the application.
2.  **Processing (The Worker):** The `AI Worker` consumes the stream, utilizing `OpenAI` to analyze the code complexity. It determines if an issue is truly "Junior Friendly" or if it requires advanced knowledge.
3.  **Serving (The Read Path - Orange Lines):** The `API Gateway` serves the frontend. It implements a **Cache-Aside** strategy: popular issues are served from Redis KV memory (<5ms), while the database is only hit on cache misses.

---

## 🛠️ Key Technical Decisions

### Why Redis Streams?
I chose Redis Streams over a simple cron job to **decouple** the fetching logic from the processing logic. This allows the system to scale independently—if issue volume spikes, I can simply spin up more `AI Worker` replicas without changing the Collector code.

### Why PostgreSQL + JSONB?
GitHub's API response is large and volatile. Instead of strictly normalizing every field, I utilize a **Hybrid Schema**:
* **Structured Columns:** `id`, `status`, `difficulty` (Indexed for fast lookups/filtering).
* **JSONB:** `raw_github_payload` (Stored as-is for future flexibility without schema migrations).

### Why Go?
Go was selected for its native concurrency primitives (`goroutines`), which are essential for handling multiple HTTP requests and background stream processing with minimal memory footprint compared to Node.js or Python.

---

## 💻 Tech Stack

| Component | Technology | Reasoning |
| :--- | :--- | :--- |
| **Backend** | Golang (Gin/Standard Lib) | Strong typing, high performance, native concurrency. |
| **Database** | PostgreSQL 16 | ACID compliance with JSONB support. |
| **Message Broker** | Redis Streams | Lightweight, low-latency event buffering. |
| **Caching** | Redis KV | High-speed read access for API endpoints. |
| **AI Layer** | OpenAI GPT-4o | Context analysis and prerequisite generation. |
| **Infrastructure** | Docker Compose | Reproducible local development environment. |



