
### 🛠 Prerequisites
Before you start, make sure you have these installed on your machine:
1. **Git**
2. **Docker Engine / Docker Desktop** (Make sure the Docker daemon is actively running)
3. **Make** (Usually pre-installed on Linux/Mac. Windows users can use WSL2 or Git Bash).

### Setup Steps

**1. Clone the Repository**
Pull down the code to your local machine:
```bash
git clone https://github.com/Kprateek283/Rapid-Response-System.git
cd Rapid-Response-System
```

**2. Start the Infrastructure**
We have a `Makefile` that automates the Docker setup and environment variables. Simply run:
```bash
make up
```
*What this does under the hood:* * Automatically copies `.env.example` to a local `.env` file so you have all the necessary connection strings.
* Downloads and starts PostgreSQL (with PostGIS), Redis, RabbitMQ, and MinIO (our local Google Cloud Storage emulator) in the background.

**3. Verify Everything is Running**
You can check if the containers are running smoothly by accessing the local management UIs in your browser:
* **RabbitMQ UI:** `http://localhost:15672` (User: `rapid_user` / Pass: `rapid_password`)
* **MinIO Storage UI:** `http://localhost:9001` (User: `rapid_admin` / Pass: `rapid_password_123`)

### Important: Branching Strategy
We are using a Trunk-Based Development workflow. The repository is locked down via GitHub rules.
* **DO NOT** push directly to `main` or `dev`.
* `main` is strictly for production-ready code.
* `dev` is our active integration branch.
* When you pick up a task, branch off `dev` (e.g., `git checkout -b feat/your-feature-name`).
* When you are done, push your branch and open a **Pull Request to the `dev` branch**. It requires at least one teammate approval before it can be merged.

### Useful Commands
Whenever you are done working for the day, or if you need to wipe your local databases clean, use these:

* `make down` -> Stops all infrastructure containers but **keeps your data**.
* `make clean` -> Stops containers AND **wipes all databases/queues clean** (useful if your local database gets corrupted).
* `make logs` -> Shows you the live output of the infrastructure containers if you need to debug a connection issue.
