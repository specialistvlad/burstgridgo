# Contributing to BurstGridGo

Thanks for checking out **BurstGridGo**!  
I’m currently the **sole maintainer** of this project, but I welcome any form of contribution — from ideas to code.  
This project is a great entry point if you’re new to open source. The barrier to entry is low, but good engineering skills are important.  

If you’re just getting started and want to learn, I’m happy to help you refine your skills. Just reach out and express your interest.

---

## How to Contribute

I’m open to all kinds of contributions — discussions, documentation, bug reports, or code. Here’s how you can get involved:

### 💬 Start a Discussion
The easiest way to contribute is to start a conversation.  
You can email me at **specialistvlad@gmail.com**, but I prefer keeping discussions public on [GitHub Discussions](https://github.com/specialistvlad/burstgridgo/discussions) so others can join in.

### ❓ Ask or Answer Questions
If you have questions, ask them in Discussions.  
If you see someone else asking for help — jump in and share your thoughts.

### 🐞 Report Bugs
If you find a bug, open an issue with as much detail as possible:
- The version you’re using  
- Steps to reproduce the problem  
- Expected vs. actual behavior

### 🌱 Suggest or Build Features
I’d love help implementing new features.  
Check out my [project board](https://github.com/users/specialistvlad/projects/1/views/2) to see ongoing ideas, or open a discussion to brainstorm your own.  
You can also create new tasks directly on the board if you have suggestions.

### 🧩 Help Manage the Repository
I’d appreciate help with maintenance tasks — cleaning up tags, managing issues, or organizing tasks.  
If you want to take part in improving the structure or workflows, that’s extremely valuable.

### 📚 Write Documentation
Documentation is currently minimal — which makes it a perfect opportunity to help.  
You can start from scratch or help expand existing sections.

---

## Development Workflow

### 🔧 Prerequisites
You’ll need **Go**, **Make**, and optionally **Docker**.

### 🚀 Setup
Fork the repository, then clone it locally:
```sh
git clone https://github.com/specialistvlad/burstgridgo.git
cd burstgridgo
```

### 🧠 Getting Started
The **Makefile** is the main entry point for development tasks.  
Run the following to see available commands:  
```sh
make
```

### ⚙️ Live Development
You can run the app in live-reload mode while editing:  
```sh
make dev-watch ./examples/http_count_static_fan_in.hcl
```

### 🧪 Running Tests
Run tests continuously while developing:  
```sh
make test-watch
```

Before committing, make sure all tests and checks pass:  
```sh
make check
```

For more commands:  
```sh
make help
```

---

## 🧭 Code of Conduct
By contributing, you agree to follow the project’s [Code of Conduct](CODE_OF_CONDUCT.md).  
I expect everyone to maintain a respectful and supportive environment.