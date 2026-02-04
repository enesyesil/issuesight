# IssueSight – Manual Test Guide

All services are running. Use this to test the app in your browser.

---

## What’s running

| Service        | URL                  | Purpose                    |
|----------------|----------------------|----------------------------|
| **Frontend**   | http://localhost:3000 | Next.js app (UI)           |
| **Gateway**    | http://localhost:8080 | API, auth, tutorials       |
| **Collector**  | http://localhost:8081 | GitHub issue collection    |
| **AI Processor** | http://localhost:8082 | Tutorial generation        |

---

## 1. Open the app

1. In your browser go to: **http://localhost:3000**
2. You should see the IssueSight landing page.

---

## 2. Health checks (optional)

- Frontend: http://localhost:3000  
- Gateway: http://localhost:8080/health  
- Collector: http://localhost:8081/health  
- AI Processor: http://localhost:8082/health  

Each should return OK or a healthy JSON response.

---

## 3. Login (required for protected features)

1. Click **Login** (or go to http://localhost:3000/login).
2. Choose **GitHub** or **Google** (only if configured in `.env`).
3. Complete OAuth; you should be redirected back to the app and see the dashboard.

**Note:** If OAuth is not set up (missing `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` in `.env`), login will fail. You can still test public pages and health endpoints.

---

## 4. Dashboard

1. After login, you should land on the **Dashboard** (e.g. http://localhost:3000/dashboard).
2. Check:
   - **Tutorials** link → list of your tutorials (may be empty).
   - **Issues** or “Submit issue” → form to submit a GitHub issue URL.

---

## 5. Submit a GitHub issue (full flow)

1. Go to **Dashboard → Issues** (or the issue submission page).
2. Enter a **public** GitHub issue URL, e.g.:  
   `https://github.com/golang/go/issues/12345`
3. Submit the form.
4. You should get:
   - An “accepted” or “processing” style message (e.g. 202 Accepted).
5. Wait 30–90 seconds (LLM + collector).
6. Go to **Dashboard → Tutorials**.
7. You should see a new tutorial for that issue; open it and check content.

**If you get errors:**

- **401** → Not logged in; complete step 3.
- **403 / quota** → Daily quota exceeded; check gateway logs or quota API.
- **409** → This issue URL is already being processed; try another issue or wait.
- **5xx** → Check gateway/collector/ai-processor logs:  
  `docker compose -f deployments/docker-compose.yml logs -f gateway collector ai-processor`

---

## 6. Quota (optional)

- If the UI shows quota, use it to see remaining submissions.
- Or call: `GET http://localhost:8080/api/quota` with your auth cookie (browser dev tools → copy as cURL and use the cookie, or use a REST client with cookies).

---

## 7. Single theme and tutorial page design (verification)

After running `npm run dev`, use this checklist to verify the theme, tutorial page, and concepts flow.

### Theme
- [ ] Primary buttons (e.g. Login, Create Tutorial) are **green** in dark theme.
- [ ] On a tutorial detail page, **completed** step checkboxes use green (primary color).

### Tutorial detail page
- [ ] **Back link:** “Back to Dashboard” with arrow is visible; click it → lands on `/dashboard/tutorials`.
- [ ] **Metadata row:** “Tutorial” tag (muted green), “X min read”, and date (with clock icon) are visible.
- [ ] **Copy Link:** Click → paste elsewhere → same URL; “Copied” appears briefly.
- [ ] **Bookmark:** Click to bookmark → refresh page → bookmark state persists; click again to remove → refresh → state cleared.
- [ ] **Share:** Click → share sheet (if supported) or same as Copy Link; no crash.

### Concepts (interface-only; backend not required)
- [ ] **Sidebar:** “Concepts” link is visible; click → `/dashboard/concepts`.
- [ ] **Concepts list:** Page shows “No concepts yet” / “Concepts will appear here once the backend is set up.”
- [ ] **Concept detail:** Open `/dashboard/concepts/any-slug` → “Concept not found” or empty state; no crash.

---

## 8. Stop everything

```bash
# Stop frontend: Ctrl+C in the terminal where `npm run dev` is running.

# Stop backend + DB + Redis:
cd /Users/enesyesil/Documents/GitHub/issuesight
docker compose -f deployments/docker-compose.yml down
```

---

## Quick checklist

- [ ] Open http://localhost:3000
- [ ] Open http://localhost:8080/health (and others) and see OK
- [ ] Login (if OAuth is configured)
- [ ] Open Dashboard and Tutorials list
- [ ] Submit one GitHub issue URL
- [ ] Wait and then see the new tutorial in the list and open it
- [ ] (Optional) Run through **Section 7** for theme, tutorial page metadata/actions, and Concepts pages

If any step fails, note the URL, action, and error message (and status code from Network tab) to debug.
