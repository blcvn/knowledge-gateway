# Test Automation & Scripting

## Automation Framework Expertise
- **Playwright (Primary):** Deep expertise in Playwright for end-to-end UI automation — browser contexts, page objects, network interception, visual snapshot testing, and parallel test execution.
- **Vitest / Jest + Testing Library:** Writing fast, reliable unit and component-level tests for React components, focusing on user interactions rather than implementation details.
- **Cypress:** Proficient in Cypress for component testing and E2E flows, especially for interactive real-time UI behaviors.

## Automation Best Practices
### Page Object Model (POM)
Structuring automation code so that selectors and interaction logic for a UI page are encapsulated in a Page Object class. This prevents fragile, duplicated selector strings scattered across tests.
```typescript
// Example: LoginPage.ts
export class LoginPage {
  constructor(private page: Page) {}

  async navigateTo() {
    await this.page.goto('/login');
  }

  async login(email: string, password: string) {
    await this.page.getByLabel('Email').fill(email);
    await this.page.getByLabel('Password').fill(password);
    await this.page.getByRole('button', { name: 'Sign In' }).click();
  }
}
```

### Selector Strategy
- **Priority Order (Most to Least Stable):**
  1. `getByRole()` — ARIA roles (most semantic and resilient)
  2. `getByLabel()` — Form labels
  3. `getByText()` — Visible user-facing text
  4. `data-testid` attribute — Explicit test hooks added by developers
  5. CSS selectors / XPath — Last resort; brittle, avoid in automation

### Test Data Management
- Using factory functions or fixtures to generate deterministic, isolated test data.
- Avoiding hardcoded shared test accounts; each test should own its data lifecycle.
- Using API calls in `beforeAll`/`beforeEach` to set up test state efficiently (bypassing UI for non-subject-under-test setup).

### Flakiness Prevention
- Using `expect(locator).toBeVisible()` instead of arbitrary `page.waitForTimeout()`.
- Leveraging network interception (`page.route`) to mock unstable third-party APIs.
- Running tests in parallel with isolated browser contexts to eliminate shared state.
