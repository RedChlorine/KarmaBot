# KarmaBot (v2 SQL Edition)

KarmaBot is a robust Telegram bot built in **Go** that manages user reputation, tracks keywords, and handles message pinning across groups.

This version ("v2") has been upgraded from a file-based system to a fully containerized **PostgreSQL** database architecture, ensuring data persistence, scalability, and performance.

## 🚀 Features

* **Reputation System:**
    * Users gain (`+1`) or lose (`-1`) reputation based on specific keywords (e.g., "thanks", "bad bot").
    * Prevents users from voting for themselves (anti-farming).
    * "Passive Registration" system automatically learns users when they speak.
* **Keyword Management:**
    * Admins can add/delete regex-based keywords dynamically.
    * Supports both positive (reputation increase) and negative (reputation decrease) keywords.
* **Pin Manager:**
    * Pin messages in the local chat.
    * **Broadcast & Pin:** Send a message to *all* registered groups and pin it instantly (`/pinall`).
    * Global Unpin: Clear pins from all groups with one command.
* **Database & Caching:**
    * Powered by **PostgreSQL** running in a Podman/Docker container.
    * Includes **pgAdmin** for easy database management.
    * **In-Memory User Cache:** Drastically reduces database load by caching known users in RAM.
* **Logging:**
    * Critical errors and startup events are logged to a dedicated Telegram channel.

## 🛠️ Tech Stack

* **Language:** Go (Golang) 1.25+
* **Database:** PostgreSQL 15 (Alpine)
* **Containerization:** Podman (or Docker)
* **Libraries:**
    * `go-telegram-bot-api/telegram-bot-api/v5`
    * `lib/pq` (Postgres Driver)
    * `joho/godotenv`

## 📋 Prerequisites

1.  **Go:** Installed on your machine ([Download](https://go.dev/dl/)).
2.  **Podman** (or Docker Desktop): Required to run the database.
3.  **Podman Compose** (or Docker Compose): To handle the multi-container setup.

## ⚙️ Installation & Setup

### 1. Clone the Repository
```bash
git clone [https://github.com/RedChlorine/karmabotv02.git](https://github.com/RedChlorine/karmabotv02.git)
cd karmabotv02
