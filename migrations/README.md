
---

### 📦 Database Migrations with `golang-migrate`

This project uses [`golang-migrate`](https://github.com/golang-migrate/migrate) to manage database schema changes in a safe and structured way.

#### ✅ Requirements

Make sure `golang-migrate` is installed on your machine.

```bash
# With Homebrew
brew install golang-migrate
```

> For other installation methods, refer to the [official docs](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate).

---

### 🛠 Creating a New Migration

Each migration consists of **two files**:  
- One for applying the changes (`up`)
- One for rolling them back (`down`)

To create a migration (e.g., to add a `roles` table and reference it from `users`):

```bash
migrate create -ext sql -dir migrations -seq add_roles_table_and_user_relation
```

This will generate:

```
migrations/
  ├── 001_add_roles_table_and_user_relation.up.sql
  └── 001_add_roles_table_and_user_relation.down.sql
```

#### Example: `001_add_roles_table_and_user_relation.up.sql`

```sql
CREATE TABLE roles (
  id_role INT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(50) NOT NULL UNIQUE
);

INSERT INTO roles (name) VALUES ('admin'), ('client');

ALTER TABLE users
ADD COLUMN id_role INT NOT NULL DEFAULT 2,
ADD CONSTRAINT fk_role FOREIGN KEY (id_role) REFERENCES roles(id_role);
```

#### Example: `001_add_roles_table_and_user_relation.down.sql`

```sql
ALTER TABLE users DROP FOREIGN KEY fk_role;
ALTER TABLE users DROP COLUMN id_role;

DROP TABLE roles;
```

---

### 🚀 Applying Migrations

Run the following command to apply all pending migrations:

```bash
migrate -path migrations -database "mysql://USER:PASSWORD@tcp(localhost:3306)/DATABASE_NAME" up
```

Replace `USER`, `PASSWORD`, and `DATABASE_NAME` with your actual database credentials.

---

### 🔁 Rolling Back Migrations (optional)

To undo the last applied migration:

```bash
migrate -path migrations -database "mysql://..." down
```

---

**Handling Migration Failures with GitHub Actions**

When working with database migrations, it's crucial to detect errors early and ensure consistency across environments. Thanks to the automated testing pipeline we implemented in GitHub Actions, any issues introduced by faulty migrations will be caught automatically before merging changes to the `master` branch.

### How It Works

Our GitHub Actions workflow is configured to:
1. Set up the test environment (including creating `.env` with test DB credentials).
2. Apply migrations using `golang-migrate`.
3. Run the complete test suite.

If any migration causes errors (e.g., syntax errors, constraint issues, missing fields), the test suite will fail, and the workflow will report a failed check on the Pull Request. This prevents the merge from happening unless the issue is resolved.

### Why This Helps
- **Prevents breaking `master`**: Ensures that only verified migrations reach the main branch.
- **Saves debugging time**: Issues are caught automatically before deployment.
- **Keeps development safe**: Team members won’t pull a broken schema.

### How to Recover
If a migration causes the workflow to fail:
1. Fix the migration file or create a new one that reverts the change.
2. Push the fix to your feature branch.
3. Let GitHub Actions rerun the workflow.

Once the check passes, you'll be able to merge safely.

### Best Practice
- Always run migrations locally before committing.
- Keep migration files atomic and minimal.
- Include tests that confirm the presence of required tables, fields, or constraints.

By leveraging the GitHub Actions pipeline, we build confidence in our database layer and avoid accidental downtime caused by unchecked schema changes.
