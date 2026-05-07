---
skill_id: SKILL-011
version: 1.0.0
status: active
priority: existing
group: Quality Assurance
created_at: 2026-04-24
---

# SKILL-011 · UI Testing & Automation

## Mô tả

Viết test cases, test scripts Playwright/Cypress, tìm root cause lỗi UI qua 5-layer diagnosis.

## Agents sử dụng

- `qa-pipeline-agent`

---

## Năng lực cốt lõi

### 1. BDD Test Cases

```gherkin
# Feature file chuẩn
Feature: Document Processing Pipeline

  Background:
    Given I am logged in as "engineer@company.com"
    And I have an active project "EKMS Demo"

  Scenario: Successfully upload and process a requirement document
    When I navigate to "Documents" page
    And I click "Upload Document"
    And I upload file "sample-prd.md"
    And I click "Start Processing"
    Then I should see status "Processing" within 2 seconds
    And I should see status "Completed" within 60 seconds
    And the Knowledge Graph should contain at least 5 nodes

  Scenario: Handle large document gracefully
    When I try to upload a file larger than 10MB
    Then I should see error "File size exceeds 10MB limit"
    And no job should be created
```

### 2. Playwright Page Object Model

```typescript
// page-objects/DocumentPage.ts
export class DocumentPage {
  private readonly uploadButton = this.page.getByRole('button', { name: 'Upload Document' })
  private readonly fileInput = this.page.locator('input[type="file"]')
  private readonly statusBadge = (jobId: string) => 
    this.page.getByTestId(`job-status-${jobId}`)

  constructor(private page: Page) {}

  async uploadDocument(filePath: string): Promise<string> {
    await this.uploadButton.click()
    await this.fileInput.setInputFiles(filePath)
    await this.page.getByRole('button', { name: 'Start Processing' }).click()
    
    // Wait for job ID in URL
    await this.page.waitForURL(/\/jobs\/(.+)/)
    const url = this.page.url()
    return url.split('/jobs/')[1]
  }

  async waitForStatus(jobId: string, status: string, timeout = 60000) {
    await expect(this.statusBadge(jobId))
      .toHaveText(status, { timeout })
  }
}

// test.spec.ts
test('document processing pipeline', async ({ page }) => {
  const docPage = new DocumentPage(page)
  await page.goto('/documents')
  
  const jobId = await docPage.uploadDocument('fixtures/sample-prd.md')
  await docPage.waitForStatus(jobId, 'Processing')
  await docPage.waitForStatus(jobId, 'Completed', 60000)
})
```

### 3. Selector Strategy

```typescript
// Priority order (most to least resilient):
// 1. Role-based (most user-centric)
page.getByRole('button', { name: 'Submit' })
page.getByRole('textbox', { name: 'Project Name' })

// 2. Test IDs (stable, intent-clear)
page.getByTestId('document-upload-btn')

// 3. Labels (for form fields)
page.getByLabel('Document Title')

// 4. Text content
page.getByText('Upload Document')

// ❌ Avoid (brittle):
page.locator('.btn-primary.upload-btn')
page.locator('#root > div > div:nth-child(2) > button')
```

### 4. Flakiness Prevention

```typescript
// ✅ Wait for network idle after navigation
await page.goto('/documents', { waitUntil: 'networkidle' })

// ✅ Wait for specific API response
const [response] = await Promise.all([
  page.waitForResponse(r => r.url().includes('/api/v1/jobs') && r.status() === 201),
  page.click('[data-testid="start-processing"]'),
])

// ✅ Retry on flaky assertion
await expect(async () => {
  const count = await page.locator('[data-testid="node-count"]').textContent()
  expect(parseInt(count!)).toBeGreaterThan(5)
}).toPass({ timeout: 10000 })

// ✅ Stable test data (seed DB before tests)
test.beforeAll(async () => {
  await seedTestDatabase()
})
```

### 5. 5-Layer Root Cause Diagnosis

```
When a UI test fails, diagnose in this order:

Layer 1: Network Layer
  → Check: API returned expected status? Correct data shape?
  → Tool: page.route(), response interception

Layer 2: State Layer
  → Check: Component received correct props/state?
  → Tool: React DevTools, Zustand devtools

Layer 3: Render Layer
  → Check: Component rendered expected HTML?
  → Tool: page.content(), element screenshots

Layer 4: Style Layer
  → Check: Element visible? Not hidden by CSS? Correct z-index?
  → Tool: page.locator().isVisible(), getBoundingClientRect()

Layer 5: Interaction Layer
  → Check: Click registered? Input focused? Event fired?
  → Tool: console.log, page.on('console'), event listeners
```

---

## Checklist

- [ ] Page Object Model cho tất cả pages
- [ ] `data-testid` attributes trên interactive elements
- [ ] API mocking cho E2E tests (không call real APIs)
- [ ] Test isolation: seed + teardown data per test
- [ ] Flakiness prevention: wait for state, not time
- [ ] Visual regression tests cho critical components
- [ ] CI: Playwright tests chạy trên chromium + firefox
