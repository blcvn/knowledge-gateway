# Root Cause Analysis (RCA) & Debugging

## Investigation Methodology: The 5-Layer Diagnosis
When a UI bug is found, always investigate from the outside in:

1. **Layer 1 — UI / Visual Layer:** Is it a CSS rendering issue? A wrong component state? Use DevTools Elements panel to inspect computed styles, layout shifts, and DOM structure.
2. **Layer 2 — JavaScript / Logic Layer:** Is there a runtime error? Use the Console panel to catch unhandled exceptions, type errors, and logical failures. Use Source panel to step through execution.
3. **Layer 3 — State Layer:** Is the application state (React state, Zustand store, Context) incorrect? Use React DevTools to inspect component props and state at the moment of failure.
4. **Layer 4 — Network / API Layer:** Is the data from the server wrong or missing? Use the Network panel to inspect XHR/Fetch requests — check status codes, request payloads, and response bodies.
5. **Layer 5 — Race Condition / Timing Layer:** Is the issue only reproducible sometimes? Look for asynchronous operations completing in the wrong order, missing loading guards, or state updates applied after component unmount.

## Key Debugging Tools
- **Chrome DevTools:** Elements, Console, Network, Sources (breakpoints), Performance, Application (localStorage, cookies, IndexedDB).
- **React DevTools:** Component tree inspection, props/state inspection, and Profiler for rendering performance.
- **Playwright Trace Viewer:** Replaying a full test execution timeline, including screenshots and network calls at every step, to pinpoint exactly when and where a test fails.
- **Lighthouse:** Automated audits for performance, a11y, SEO, and best practices to surface systemic issues.

## Bug Report Standard (High Quality)
A high-quality bug report must include:
- **Title:** `[Module] Concise description of symptom` (e.g., `[Login] Error message not cleared on second submit attempt`)
- **Severity:** Critical / High / Medium / Low
- **Reproducibility:** Always / Intermittent / Rare
- **Environment:** Browser, OS, Screen resolution, App version/commit hash
- **Preconditions:** Exact state of the system before reproduction
- **Steps to Reproduce:** Numbered, minimal, exact steps
- **Expected Result:** What should happen
- **Actual Result:** What actually happens
- **Evidence:** Screenshots, screen recording, console error log (copied as text), Network request/response body
- **Root Cause Hypothesis:** Your best technical diagnosis of *why* this is happening

## Common UI Bug Root Cause Categories
| Category | Symptoms | Common Cause |
|---|---|---|
| **Race Condition** | Intermittent, order-dependent failures | `async/await` missing, state updated after unmount |
| **State Management Bug** | Stale data displayed, UI out of sync | Missing invalidation, wrong selector, shared mutable state |
| **Rendering Bug** | Visual glitch, layout shift, z-index issue | CSS specificity conflict, missing `key` prop in lists |
| **Network / API Error** | Empty screens, error toasts | Wrong API URL, CORS, authentication token expired |
| **Input Validation Gap** | App crashes on edge-case input | Missing null checks, unhandled empty string |
| **Browser Compatibility** | Only broken in one browser | Unsupported CSS property, missing polyfill |
