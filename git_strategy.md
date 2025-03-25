Here's a general approach to reorganizing your Git repo so that you have a well-defined production branch (often called “main” or “master”) and a stable development branch:

1. **Rename or keep your current “master” as “main” (optional):**
	- Many teams are moving toward “main” as the primary branch name. If you decide to rename:

	  ```bash
	  git branch -m master main
	  git push origin main
	  git push origin --delete master
	  ```
	- Then, update your repo settings on GitHub so that “main” is marked as the default branch. If you prefer to keep it as “master,” that's okay—just keep in mind that references below will say “main,” but the idea is the same if you keep “master.”

2. **Create a dedicated “dev” (or “develop”) branch from your production branch:**
	- If you renamed your production branch to “main,” then you do:

	  ```bash
	  git checkout main
	  git pull origin main
	  git checkout -b dev
	  git push -u origin dev
	  ```
	- In GitHub, you can optionally set “dev” as the default branch if you want new pull requests to open against it, or you can keep “main” as default but just communicate to your team that feature work should happen off “dev.”

3. **Adopt a branching strategy:**
	- **Git Flow** (traditional approach for many teams):
		- “main” (or “master”) is always production-ready code.
		- “dev” (or “develop”) is the main integration branch for new features.
		- Each new feature/bugfix starts on a feature branch off “dev,” e.g. `feature/qr-code-updates`.
		- When a feature is done, you open a pull request into “dev” and merge after code review.
		- Periodically, you create a release branch off “dev” (e.g., `release/1.2.0`) to finalize the release. When it’s stable, you merge it into “main” and tag it.
	- **GitHub Flow** (simpler approach for smaller teams or continuous deployment):
		- “main” is the only long-lived branch. Everything is done in short-lived branches off “main.” Merges into “main” trigger deploys.
	- **Trunk-Based Development** (used in very fast-paced environments with continuous integration):
		- “main” is the trunk where everyone merges frequently (possibly multiple times a day), with feature toggles controlling partially done features.

   For most teams, **Git Flow** or a simplified variation works well to keep production stable and allow collaborative feature development on a “dev” branch.

4. **Protect your branches on GitHub:**
	- In the repository “Settings” → “Branches” section, add rules so that direct pushes to “main” (and maybe also “dev”) are not allowed. Require pull requests (PRs) and code reviews.

5. **Clean up old ad hoc branches:**
	- If you have a bunch of unmerged or stale branches, merge or close them. Tag any important historical commits if you need them for reference. Then align your new feature or bugfix work under the newly established branching model.

6. **Process going forward** (example Git Flow style):
	- Pick an issue or feature → create a branch from “dev” → do the work → push and open a pull request to “dev” → merge after review → once stable, merge “dev” into “main” (production). Tag the release.

---

### Typical Day-To-Day Example

1. **Create a feature branch**:

   ```bash
   git checkout dev
   git pull origin dev
   git checkout -b feature/fix-404-when-logout
   ```
2. **Implement your fixes**, commit and push:

   ```bash
   git add .
   git commit -m "Fix 404 logout bug"
   git push -u origin feature/fix-404-when-logout
   ```
3. **Open a PR** from `feature/fix-404-when-logout` into `dev` on GitHub.
4. **Merge** once it’s reviewed and tested.
5. **Periodically** (when you’re ready for production), open a PR from `dev` → `main` (or merge it via the command line). This code is now in production.

---

### Summary of Benefits

- **Isolate production** from incomplete or buggy features.
- **Reduce conflicts** and keep a clear record of what was merged, and when.
- **Enable reviews and testing** before code goes to production.
- **Clean release history** using tags and merges into “main” so you can easily roll back if needed.

Following a structured branching model makes your version-control process smoother and more predictable—particularly crucial as your project grows and more contributors become involved.

-------------------------
# Git Versioning

Versioning is the practice of assigning meaningful labels (versions) to distinct states of your software so that you can keep track of what changed, when it changed, and why it changed. This makes it easier to communicate changes to collaborators and users, and to roll back to previous stable points if something goes wrong.

Below is an overview of versioning concepts and how you might integrate them into your GitHub flow.

---

## Why Versioning Matters

1. **Traceability:** Every release can be quickly identified and mapped to the exact code changes, new features, or bug fixes it contains.
2. **Rollback & Support:** If you discover a bug in version 1.2.3, you can roll back to version 1.2.2, or you can release a hotfix as version 1.2.4 without interfering with new development.
3. **Communication:** Users, teammates, or stakeholders immediately know the scope of changes in a release.

---

## Semantic Versioning

A widely accepted standard for version numbering is [Semantic Versioning](https://semver.org/), which typically looks like:
```
vMAJOR.MINOR.PATCH
```
Where:
- **MAJOR**: Increased when you make incompatible API (or big) changes.
- **MINOR**: Increased when you add functionality in a backwards-compatible manner.
- **PATCH**: Increased when you make backwards-compatible bug fixes only.

Examples:
- **v1.0.0** → your first stable release.
- **v1.1.0** → new features added (backwards-compatible).
- **v1.1.1** → small bugfix release.

---

## Versioning with Git Tags

A simple way to apply versioning in GitHub is to use **Git tags** to mark commits (especially merges into your production branch) with a version number:
1. Once you have merged your code from “dev” into “main” (or “master”) and tested that it is stable, decide on the new version number—e.g., `v1.2.0`.
2. Tag that commit in Git:
   ```bash
   git checkout main
   git pull origin main
   git tag -a v1.2.0 -m "Release v1.2.0"
   git push origin v1.2.0
   ```
3. On GitHub, you’ll see a “Release” or “Tag” entry for `v1.2.0`. That becomes your official release artifact.

Using this approach, you can see each release under the “Tags” or “Releases” tab in GitHub.

---

## Integrating Versioning into Your GitHub Strategy

1. **Decide on a Branching Model** (e.g. Git Flow):
	- You have “main” as production, “dev” as your integration branch.
	- When “dev” is stable and tested, you merge into “main” and create a release.

2. **Automate the Tagging/Release Process**:
	- You can manually run the commands to create a tag after each “main” merge.
	- Alternatively, set up a GitHub Action that tags automatically. Some teams use tools like [semantic-release](https://github.com/semantic-release/semantic-release) to parse commit messages and decide whether to bump MAJOR, MINOR, or PATCH automatically.

3. **Create a Release on GitHub**:
	- After pushing the tag, you can go to the “Releases” section in GitHub and create a new release entry.
	- This often includes release notes about what changed, any migration steps, etc.

4. **Hotfix Releases**:
	- If you discover a bug in production (`main` branch) while new development is happening in “dev,” you can:
		- Branch off `main` → fix bug → merge back into `main` → tag with a bump in the PATCH number (e.g. `v1.2.1`).
		- Also merge the fix into “dev” so it’s included in future releases.

5. **Document Changes**:
	- Maintaining a `CHANGELOG.md` can help you keep a curated history of each version’s changes. Or you can rely on GitHub’s releases page to track changes if you include detailed release notes.

---

### Example Flow

1. **Coding a new feature**
	- You branch off “dev” into `feature/add-qr-redirect`, do your work, then merge into “dev” when ready.

2. **Prepare a Release**
	- Everything on “dev” is tested, so you create a pull request from “dev” into “main.”
	- Once merged, `main` is now at a stable release point.

3. **Assign Version Number**
	- Suppose this is your second minor release since `v1.1.0`, so you decide on `v1.2.0`.
	- Tag it:
	  ```bash
	  git checkout main
	  git pull origin main
	  git tag -a v1.2.0 -m "Release v1.2.0"
	  git push origin v1.2.0
	  ```
4. **Create a GitHub Release**
	- Go to GitHub → “Releases” → “Draft new release.”
	- Title: “v1.2.0”
	- Add notes: “Added QR code redirect for referees, improved health checks,” etc.

5. **Users / Admins Download or Deploy**
	- Now that version is pinned. The tagged commit and the built artifact (if you have a CI/CD pipeline) can be deployed or tested further.

---

## Handling Larger Projects

- **Release Branches**: In more complex teams (especially those using Git Flow), you might have separate “release/x.y” branches where you finalize a release. You merge from “dev” → “release/x.y” → do final bugfixes → then merge “release/x.y” → “main” and tag.
- **Continuous Deployment**: If you release very frequently, your tags might reflect smaller, more frequent increments (e.g., daily or weekly builds).

---

### Tips
- **Keep versions consistent** across code, docs, and environment.
- **Use your CI/CD** to automate building and deploying whenever you push a tag.
- **Use a `CHANGELOG.md` or GitHub’s release notes** to communicate key changes.

---
# Versioning

## Summary

- **Semantic Versioning** is a clear system for numbering releases.
- **Git Tags** mark commits with these versions.
- **GitHub Releases** help you host, describe, and track each version.
- **Adopt a branching model** (Git Flow, GitHub Flow, or Trunk-Based Development) and integrate your version tags/release steps at the point you merge stable code into production (“main” or “master”).

By consistently tagging your production commits and incrementing your version number according to the significance of the changes (Semantic Versioning), you’ll have a clean, traceable release history and an easy way to manage and communicate changes.
