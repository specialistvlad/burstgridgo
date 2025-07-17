# burstgridgo

A declarative, modular, developer-first load testing tool built in Go.  
Simulate real-world user workflows across custom protocols — with concurrency at scale.

---

## 🚀 Motivation

This project started from a real moment of frustration.

I just needed to simulate a realistic user workflow — one that uses a mock JWT token, communicates with a backend over Socket.IO, uploads files to S3, tracks progress events, and waits for a result — and suddenly every existing tool made it harder than it should be.

On top of all that, I was also looking for a meaningful way to finally learn Go — and this project felt like the perfect blend of practical need and curiosity.

I tried **k6** — great branding, solid Go engine, and scripting in JavaScript. But it couldn’t handle Socket.IO out of the box. I had to either reimplement everything manually in JavaScript or compile in a community extension. For a fast start, it was just too much.

Then I tried **Artillery** — also very capable. But it pushed me toward a cloud-based workflow I didn’t want, and its config format mixed declarative and imperative code in a way that felt awkward and hard to reason about.

I want to be clear: **these are well-crafted tools, professionally built and maintained**. But they didn’t give me the flexibility I needed for a quick, focused, extensible solution.

**burstgridgo** is my answer to that gap. I wanted a tool that:
- Embraces declarative structure without boxing you in
- Runs anywhere — locally or in Docker — with zero friction
- Makes it dead simple to add new protocols or custom logic, without fighting the framework

---

## 🧑‍💻 Getting Involved

This project started as a personal learning journey — but I’m building it to be useful for others too.

Have ideas, feedback, or protocol needs I haven’t thought of? I’d love to hear from you.

Even if the code is still green, the vision is clear — and collaboration makes it better.

---

## ✨ Core Ideas

- **Declarative by design** — describe complex workflows as simple config
- **Composable modules** — reuse and share parts across test plans
- **Concurrency-first** — powered by Go’s goroutines and channels
- **Developer-first** — plug in any protocol or logic with minimal friction
- **Extensible** — custom runners, reusable workflows, internal module APIs

---

## ⚙️ What burstgridgo Is (and Isn’t)

✅ A framework for simulating **real user flows**  
✅ A developer-oriented platform to load test **anything with a protocol**
✅ A tool to define **parallel, branching, dependent workflows**  

❌ Not just an HTTP benchmarker  
❌ Not a GUI-driven enterprise suite  
❌ Not finished — this is on concept stage and Go learning journey

---

## 📦 Example Configuration (on concept stage)

Add me

---

## 🧱 Architecture (on concept stage)

Add me

---

## 🧭 Roadmap (on concept stage)

Add me

---

## 📄 License

MIT

---

## 🙏 Acknowledgements

Shout out to the creators of tools like [k6](https://k6.io/), [Vegeta](https://github.com/tsenart/vegeta), and [Artillery](https://www.artillery.io/) — they each pushed the conversation around load testing forward in their own way, and are outstanding in their domains.

Also, a nod to the Go community for designing a language that makes concurrency feel natural and fun to work with.

burstgridgo stands on a lot of great shoulders.
