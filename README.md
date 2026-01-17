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

Junior engineers often struggle to contribute not because they can't code, but because they lack the domain context of massive repositories. IssueSight ingests GitHub issues and uses LLMs to generate **"Context Bridges"** from breaking down complex tickets into junior-level prerequisites, architectural summaries, and implementation guides.

---

## 🏗️ System Architecture
The system follows a vertical **Microservices Layering** pattern. Traffic flows from the Clients (Top) through the Go Middleware (Center) down to the Persistence Layer (Bottom).

![IssueSight Architecture Diagram](./issuesight-design.png)


``` mermaid
---
config:
  theme: neo-dark
---
flowchart TB
 subgraph ClientLayer["1. Client Layer"]
    direction TB
        UserApp("User")
  end
 subgraph GatewayLayer["2. Gateway Layer"]
    direction TB
        APIGateway["API Gateway"]
        AuthMgr["Auth & Quota Manager"]
        LockMgr["Lock Manager"]
  end
 subgraph ExternalLayer["5. External Ecosystem"]
        GitHub("GitHub API")
        LLM("LLM Provider")
  end
 subgraph LogicLayer["3. Logic & Processing Layer"]
    direction TB
        Collector["Collector Worker"]
        AIWorker["AI Generator Worker"]
  end
 subgraph DataLayer["4. Data & State Layer"]
    direction TB
        MongoDB[("MongoDB\nAuth & Quotas")]
        Redis[("Redis Speed Layer\nCache/Locks/Stream")]
        Postgres[("PostgreSQL\nTutorial Archive")]
  end
    UserApp -- "1. Submit Issue / Auth" --> APIGateway
    APIGateway -.-> AuthMgr & LockMgr
    AuthMgr -- "2. Check Limit" --> MongoDB
    LockMgr -- "3. Distributed Lock" --> Redis
    APIGateway -- "4. Enqueue Task" --> Redis
    Collector -- "5. Poll Metadata" --> GitHub
    Collector -- "6. Push Context" --> Redis
    Redis -- "7. Stream Consume" --> AIWorker
    AIWorker -- "8. Generate Content" --> LLM
    AIWorker -- "9. Persist Tutorial" --> Postgres

     UserApp:::client
     APIGateway:::gateway
     AuthMgr:::gateway
     LockMgr:::gateway
     GitHub:::external
     LLM:::external
     Collector:::worker
     AIWorker:::worker
     MongoDB:::data
     Redis:::data
     Postgres:::data
    classDef client fill:#fff3e0,stroke:#f57c00,stroke-width:2px,rx:10,ry:10
    classDef gateway fill:#e3f2fd,stroke:#1565c0,stroke-width:2px,rx:5,ry:5
    classDef worker fill:#e8f5e9,stroke:#2e7d32,stroke-width:2px,rx:5,ry:5
    classDef data fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px,shape:cyl
    classDef external fill:#eeeeee,stroke:#999999,stroke-width:2px,stroke-dasharray: 5 5,rx:5,ry:5
    style UserApp fill:#00C853,color:#000000
    style APIGateway fill:#2962FF
    style AuthMgr fill:#2962FF
    style LockMgr fill:#2962FF
    style GitHub fill:#2962FF
    style LLM fill:#00C853
    style Collector fill:#2962FF
    style AIWorker fill:#FFD600,color:#000000
    style MongoDB fill:#FF6D00
    style Redis fill:#2962FF
    style Postgres fill:#00C853
    style GatewayLayer stroke:#00C853,fill:#00C853,color:#000000
    style DataLayer fill:#FF6D00,color:#000000
    style LogicLayer fill:#00C853,color:#000000
    style ExternalLayer fill:#FFD600,color:#000000
    style ClientLayer fill:#BBDEFB,color:#000000
    linkStyle 0 stroke:#f57c00,stroke-width:2px,fill:none
    linkStyle 1 stroke:#2962FF,fill:none
    linkStyle 2 stroke:#2962FF,fill:none
    linkStyle 3 stroke:#2962FF,fill:none
    linkStyle 4 stroke:#2962FF,fill:none
    linkStyle 5 stroke:#2962FF,fill:none
    linkStyle 6 stroke:#000000,fill:none
    linkStyle 7 stroke:#000000,fill:none
    linkStyle 8 stroke:#2e7d32,stroke-width:2px,fill:none
    linkStyle 9 stroke:#2e7d32,stroke-width:2px,fill:none
    linkStyle 10 stroke:#2962FF,fill:none
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



