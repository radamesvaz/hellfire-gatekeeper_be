
---

## ✅ Continuous Testing with GitHub Actions

This project includes an automated test workflow using **GitHub Actions**. Every time you push or open a pull request against the `master` branch, a CI pipeline runs to validate that all tests pass.

### What It Covers

- ✅ **Unit tests** (run locally using Go’s testing framework)
- ✅ **Integration tests** (using `testcontainers-go` and Docker)
- ✅ **Migration safety** (any change that breaks the database schema or application logic will be detected)

### How It Works

1. A GitHub Action defined in `.github/workflows/run-tests.yml` runs the following script:

```bash
./run.sh tests
```

2. This script:
   - Loads environment variables
   - Runs `go test ./...` on all modules
   - Fails the pipeline if any test fails

3. If the pipeline fails:
   - ✅ The pull request **cannot be merged**
   - ✅ You’ll get feedback in the **Checks** tab

### Why It Matters

- 🧪 Ensures all changes are safe and tested
- 🔁 Helps identify issues early in the development cycle
- 🔐 Gives you peace of mind when modifying **database migrations**, **models**, or **business logic**

---
