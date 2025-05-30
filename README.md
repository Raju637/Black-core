# ⚫ Black-Core

> 💻 Rajat didn’t just build a chatbot — he built a digital weapon.

**Black-Core** is an AI-powered command-line assistant built in **Go (Golang)** for **ethical hackers, red teamers, and cybersecurity pros**. It converts **natural language requests into terminal-ready hacking commands**, automating tool selection and syntax like a digital cyber-brain.

---

## 🧠 What is Black-Core?

You give it instructions like:

> “Scan a website for open ports.”  
> “Run a brute-force attack on SSH.”  
> “Generate a reverse shell payload.”

Black-Core uses AI to respond with perfectly structured terminal commands using tools like **Nmap**, **Metasploit**, **SQLMap**, **Hydra**, and more.

---

## ✨ Features

- 🧠 Converts natural language → hacking commands  
- ⚙️ Built with Go for performance and portability  
- 🔍 Supports popular tools (Nmap, Hydra, SQLMap, etc.)  
- 📚 Can use **Phind API** or **Ollama** (offline AI backend)  
- 🎨 CLI effects with colorful output and formatting  
- 💬 Multiple AI modes (default: Phind, optional: local LLMs)  
- 📜 Multi-line command outputs with detailed descriptions  
- 🧑‍💻 Works fully from the terminal — no GUI required  

---

## 🧰 Built With

- ⚙️ [Go (Golang)](https://golang.org/)
- 🧠 [Phind AI](https://www.phind.com/) or [Ollama](https://ollama.com/) for language understanding
- 🎨 [Fatih Color](https://github.com/fatih/color) (optional) for CLI formatting

---

## 🧪 Installation

### Prerequisites
- Go 1.18 or higher
- Internet connection (for Phind API)
- `.env` file for config (see below)

### 1. Clone the Repository

```bash
git clone https://github.com/yourusername/black-core
cd black-core
