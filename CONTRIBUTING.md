Contributing to burstgridgo
First off, thank you for considering contributing! This project is driven by the community, and we welcome any contributions, from bug reports to new features.

How to Contribute
Reporting Bugs: If you find a bug, please open an issue and provide as much detail as possible, including the version you're using and steps to reproduce it.

Suggesting Features: We'd love to hear your ideas! The best place for a new feature idea is in GitHub Discussions, where we can brainstorm and refine it.

Answering Questions: You can help other users by participating in discussions and answering questions.

Development Workflow
Prerequisites: You will need Go, Docker, and Make installed.

Setup: Fork the repository and clone it to your local machine.

Development Loop: The easiest way to work on the project is with the live-reloading development environment. This will automatically recompile and restart the application when you change a file.

[CODE_SNIPPET: dev_setup_command]

Running Tests: Before submitting a contribution, please ensure all tests and linters pass.

To run the test suite:
[CODE_SNIPPET: run_tests_command]

To run the linter:
[CODE_SNIPPET: run_lint_command]

Pull Request Process
Create a new branch for your feature or bugfix.

Make your changes and commit them with a clear, descriptive message.

Ensure you have added or updated tests for your changes.

Push your branch to your fork and open a Pull Request against the main branch.

Link the PR to any relevant issues.

Documenting Your Code
We auto-generate documentation for runners from the source code. If you are adding or changing a runner, please document your code.

Add standard Go doc comments to your runner's configuration struct and its fields.

Add or update the example.hcl file in the runner's directory.

After making your changes, regenerate the documentation by running:

[CODE_SNIPPET: run_doc_gen_command]

Code of Conduct
This project adheres to a Code of Conduct. By participating, you are expected to uphold this code.