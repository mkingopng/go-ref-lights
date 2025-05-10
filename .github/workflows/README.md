## 🚨 Emergency Rollback Guide

If something goes wrong with the production deployment, manually roll back to a previously tagged known-good version (such as `v0.3.1`).

### How to Roll Back in Production

1. Go to the GitHub repository:
   [https://github.com/mkingopng/go-ref-lights](https://github.com/mkingopng/go-ref-lights)

2. Click the **Actions** tab.

3. Select the workflow titled **“CI/CD Pipeline”**.

4. Click the **“Run workflow”** button (top right).

5. In the **Deploy tag** input field, enter the tag or branch you want to deploy, e.g.:

6. Click **Run workflow**.

7. The deployment will start immediately. You’ll receive a Slack message when deployment is complete.

### When to Use This

Use this rollback procedure if:
- Production has broken due to a recent code change.
- You need to restore a known-good state immediately (e.g. during a competition).
- A patch isn't immediately ready.

---

### Known-Good Tags

| Tag       | Description                              |
|-----------|------------------------------------------|
| `v0.3.1`  | Used for Mountain Top Rumble – known stable |
