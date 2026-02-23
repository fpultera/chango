**chango** is an open-source, cloud-native platform for building real-time chat and voice communities.

Designed with Go and Kubernetes at its core, it provides a scalable foundation to create distributed communication systems similar to modern collaboration platforms. The project focuses on simplicity, performance, and operational transparency, making it suitable for self-hosted environments, experimentation, and production-grade deployments.

### ✨ Features

* **Real-time text messaging** with low-latency delivery
* **Voice rooms** designed for scalable group communication
* **Kubernetes-native architecture** for horizontal scalability
* **Stateless services + event-driven backbone** for resilience
* **Self-hosted and fully open** — no vendor lock-in
* **Modular design** so you can extend presence, moderation, or federation
* **Observability-first** with metrics, logs, and tracing in mind
* **Built with Go** for performance, concurrency, and simplicity

### 🏗️ Architecture Philosophy

chango embraces modern distributed-system principles:

* API Gateway handling WebSocket / streaming connections
* Independent services for messaging, presence, and voice coordination
* Event-driven communication (Pub/Sub friendly: NATS, Kafka, Redis Streams, etc.)
* Kubernetes orchestration for scaling rooms and workloads dynamically
* Cloud-agnostic by design — run locally, on-prem, or in any cloud

### 🎯 Goals

This project is not just a chat server — it is a **reference architecture** for building real-time platforms using open technologies.
It aims to help developers learn, experiment, and deploy their own communication infrastructure without relying on proprietary ecosystems.

### 🤝 Use Cases

* Self-hosted community platforms
* Internal team collaboration tools
* Gaming communities with voice channels
* Real-time system design experimentation
* Kubernetes + Go learning projects

---

**Build your own communication platform. Own your infrastructure. Learn by running it.**
