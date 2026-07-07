# Contributing to Synchroma

First off, thank you for considering contributing to Synchroma! It's people like you that make open-source tools great.

## Development Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/zlfzx/synchroma.git
   cd synchroma
   ```

2. **Install dependencies:**
   Ensure you have Go installed (version 1.20+ recommended).
   ```bash
   go mod download
   ```

3. **Running the code:**
   You can run the tool locally during development:
   ```bash
   go run main.go --help
   ```

4. **Running tests:**
   Before submitting any changes, make sure all tests pass:
   ```bash
   go test ./...
   ```

## Pull Request Process

1. Fork the repository and create a new branch from `main`.
2. Ensure your code follows the standard Go formatting (`go fmt ./...`).
3. If you've added new features, please write tests for them.
4. Update the `README.md` with details of changes to the interface, new flags, or features.
5. Create a Pull Request with a clear title and description of your changes.

## Reporting Issues

If you find a bug or have a feature request, please create an issue on GitHub. Include as much detail as possible, such as:
- Your operating system and Go version
- The source and target database types and versions
- Steps to reproduce the issue
- Expected vs actual behavior

Thank you for your contributions!
