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
* **Caching Strategies:** utilizing the "Cache-Aside" pattern for high-read throughput.

---

## 🏗️ System Architecture
The system is architected as a **Go Monorepo** composed of loosely coupled microservices. It utilizes **Redis Streams** as an event bus to handle the asynchronous nature of AI processing and GitHub polling.

```mermaid
flowchart LR
    %% --- STYLING (ByteByteGo Aesthetic) ---
    classDef container fill:#ffffff,stroke:#333333,stroke-width:2px,rx:5,ry:5;
    classDef goService fill:#e6f7ff,stroke:#1890ff,stroke-width:2px,rx:5,ry:5;
    classDef database fill:#ffffff,stroke:#333333,stroke-width:2px,shape:cyl;
    classDef cache fill:#fff0f6,stroke:#eb2f96,stroke-width:2px,shape:cyl;

    %% --- 1. CLIENT LAYER ---
    subgraph ClientLayer ["1. Client Layer"]
        direction TB
        User((User App)):::container
    end

    %% --- 2. GO SERVICES LAYER ---
    subgraph ServiceLayer ["2. Go Services Layer"]
        direction TB
        Gateway[API Gateway]:::goService
        Worker[AI Worker]:::goService
    end

    %% --- 3. DATA LAYER ---
    subgraph DataLayer ["3. Data Layer"]
        direction TB
        Redis[(Redis Cache)]:::cache
        DB[(PostgreSQL)]:::database
    end

    %% ====== EXECUTION FLOW ======
    User -- "1. GET /issue" --> Gateway
    Gateway -- "2. Check Cache" --> Redis
    Redis -.-> |"3. Miss"| Gateway
    Gateway -- "4. Read Data" --> DB
    DB -- "5. Return Data" --> Gateway
    Gateway -- "6. JSON Response" --> User
    
    %% Background Write
    Worker -.-> |"Background Ingestion"| DB

    %% --- LINK STYLING ---
    linkStyle 0,5 stroke:#1890ff,stroke-width:2px;
    linkStyle 1,2,3,4 stroke:#52c41a,stroke-width:2px;
