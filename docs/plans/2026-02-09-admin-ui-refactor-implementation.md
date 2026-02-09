# Admin UI Refactor + Task Auto-Refresh Fix Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor the admin UI to a clean, data-dense dashboard style and fix task auto-refresh so only the task list updates without clearing the input.

**Architecture:** Keep the static HTML/CSS/JS approach but split view rendering into “shell + refresh” functions. Auto-refresh updates only the task list container. Centralize new design tokens in `styles.css` and align login and dashboard visuals.

**Tech Stack:** Static HTML, CSS, vanilla JavaScript (admin assets under `cmd/server/static/admin/`).

---

### Task 1: Establish Design Tokens + Global Base Styles

**Files:**
- Modify: `cmd/server/static/admin/styles.css`

**Step 1: Update design tokens**
- Replace palette in `:root` with new blue/amber system.
- Add font imports for `Fira Sans` and `Fira Code`.

```css
@import url('https://fonts.googleapis.com/css2?family=Fira+Code:wght@400;500;600&family=Fira+Sans:wght@300;400;500;600;700&display=swap');
```

**Step 2: Update base typography**
- Body uses `Fira Sans` and body text color.
- Add `.mono` utility for code-like fields (`Fira Code`).

**Step 3: Run manual check**
- Open `http://localhost:10999/admin/dashboard.html` and confirm fonts load and text contrast is readable.

**Step 4: Commit**
```bash
git add cmd/server/static/admin/styles.css
git commit -m "style: add admin design tokens and typography"
```

---

### Task 2: Refactor Dashboard Layout Structure (HTML)

**Files:**
- Modify: `cmd/server/static/admin/dashboard.html`

**Step 1: Add layout containers**
- Wrap header/sidebar/main with new class names for the updated layout.
- Add stable container elements for each view, especially tasks: `data-slot="tasks-table"`.

**Step 2: Ensure accessibility hooks**
- Add `aria-live` for message areas.
- Ensure buttons have visible focus in CSS (already supported).

**Step 3: Manual check**
- Open dashboard and verify layout is stable and content loads.

**Step 4: Commit**
```bash
git add cmd/server/static/admin/dashboard.html
git commit -m "refactor: restructure admin dashboard layout"
```

---

### Task 3: Split Dashboard Rendering into Shell + Refresh

**Files:**
- Modify: `cmd/server/static/admin/dashboard.js`

**Step 1: Create render shell functions**
- `renderUsersShell()`, `renderTasksShell()`, `renderNotificationsShell()`, `renderUserDetailsShell(fid)`.
- Each shell sets static DOM and placeholders only once.

**Step 2: Create refresh functions**
- `refreshUsersTable()`, `refreshTasksTable()`, `refreshNotificationsTable()`, `refreshUserDetailsTable(fid)`.
- These only update table body or list container.

**Step 3: Fix auto-refresh**
- Auto-refresh interval calls only `refreshTasksTable()`.
- Avoid resetting the form or control bar on refresh.

**Step 4: Manual check**
- In Tasks view, type in the “兑换码” input and confirm it remains while the task list refreshes.

**Step 5: Commit**
```bash
git add cmd/server/static/admin/dashboard.js
git commit -m "fix: refresh task list without losing input"
```

---

### Task 4: Update Login Page to Match New System

**Files:**
- Modify: `cmd/server/static/admin/login.html`
- Modify: `cmd/server/static/admin/styles.css`

**Step 1: Move inline styles to CSS**
- Extract login styles into `styles.css` under a “Login” section.

**Step 2: Update visuals**
- Use the new palette and typography.
- Ensure button states and focus rings remain visible.

**Step 3: Manual check**
- Open `http://localhost:10999/admin/login.html` and confirm visual alignment with dashboard.

**Step 4: Commit**
```bash
git add cmd/server/static/admin/login.html cmd/server/static/admin/styles.css
git commit -m "style: align login page with dashboard design"
```

---

### Task 5: Visual Polish + Responsive Pass

**Files:**
- Modify: `cmd/server/static/admin/styles.css`
- Modify: `cmd/server/static/admin/dashboard.js`

**Step 1: Add UI polish**
- Hover/active states for nav and table rows.
- Badge styling improvements and consistent spacing.

**Step 2: Responsive checks**
- Check at 375/768/1024/1440 widths for layout and table scroll.

**Step 3: Manual check**
- Verify no horizontal scroll on mobile for dashboard and tasks list.

**Step 4: Commit**
```bash
git add cmd/server/static/admin/styles.css cmd/server/static/admin/dashboard.js
git commit -m "style: polish admin dashboard responsive behavior"
```

---

### Task 6: Final Verification

**Files:**
- Modify: none

**Step 1: Run Go tests (if needed)**
```bash
go test ./...
```
Expected: all tests pass

**Step 2: Manual acceptance checks**
- Tasks auto-refresh does not clear input.
- Only task list area updates during refresh.
- Login and dashboard match visual system.

**Step 3: Commit (if any final tweaks)**
```bash
git add -A
git commit -m "chore: finalize admin ui refactor"
```
