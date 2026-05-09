# EKMS Quick Start Guide
## Getting Started with Enterprise Knowledge Management System Development

**Target Audience:** Development Team  
**Time to Complete:** 2-4 hours for local setup  
**Prerequisites:** Docker, Kubernetes (minikube/kind), Node.js 18+, Python 3.11+, Go 1.21+

---

## 1. Repository Structure

```
ekms/
├── backend/
│   ├── services/
│   │   ├── api-gateway/          # Kong/custom API gateway
│   │   ├── user-service/         # User management (Go)
│   │   ├── document-service/     # Document CRUD (Go)
│   │   ├── search-service/       # Search API (Go)
│   │   ├── ai-service/           # AI/ML APIs (Python)
│   │   └── agent-orchestrator/   # Agent framework (Python)
│   ├── shared/
│   │   ├── proto/                # gRPC definitions
│   │   ├── auth/                 # Auth middleware
│   │   └── database/             # DB utilities
│   └── scripts/
│       ├── migrate.sh
│       └── seed.sh
├── frontend/
│   ├── web/                      # Next.js application
│   ├── mobile/                   # React Native (future)
│   └── components/               # Shared UI components
├── ai/
│   ├── agents/
│   │   ├── curator/              # Curation agent
│   │   ├── retrieval/            # Retrieval agent
│   │   ├── analyst/              # Analysis agent
│   │   └── compliance/           # Compliance agent
│   ├── rag/
│   │   ├── embeddings/           # Embedding generation
│   │   ├── retrieval/            # Retrieval pipeline
│   │   └── generation/           # LLM generation
│   └── models/
│       └── fine-tuned/           # Custom model weights
├── infrastructure/
│   ├── terraform/                # IaC for cloud resources
│   ├── kubernetes/
│   │   ├── base/                 # Base K8s manifests
│   │   ├── overlays/             # Environment-specific
│   │   └── helm/                 # Helm charts
│   └── docker/
│       └── docker-compose.yml    # Local development
├── docs/
│   ├── architecture/
│   ├── api/
│   └── user-guides/
└── tests/
    ├── unit/
    ├── integration/
    └── e2e/
```

---

## 2. Local Development Setup

### Step 1: Clone Repository

```bash
git clone <repository-url>
cd ekms
```

### Step 2: Environment Setup

```bash
# Copy environment templates
cp .env.example .env.local

# Edit environment variables
# Required: DATABASE_URL, REDIS_URL, OPENAI_API_KEY, etc.
nano .env.local
```

**Required Environment Variables:**

```bash
# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/ekms
MONGO_URL=mongodb://localhost:27017/ekms
NEO4J_URL=bolt://localhost:7687
REDIS_URL=redis://localhost:6379

# Vector Database
QDRANT_URL=http://localhost:6333
QDRANT_API_KEY=your-key

# AI Services
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...

# Search
ELASTICSEARCH_URL=http://localhost:9200

# Auth
JWT_SECRET=your-secret-key
OAUTH_CLIENT_ID=your-client-id
OAUTH_CLIENT_SECRET=your-client-secret

# Object Storage
S3_BUCKET=ekms-documents
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...

# Application
NODE_ENV=development
API_PORT=8080
FRONTEND_PORT=3000
```

### Step 3: Start Infrastructure with Docker Compose

```bash
# Start all infrastructure services
docker-compose up -d

# Verify services are running
docker-compose ps

# Expected services:
# - postgres (port 5432)
# - mongodb (port 27017)
# - neo4j (port 7687, 7474)
# - redis (port 6379)
# - elasticsearch (port 9200)
# - qdrant (port 6333)
# - rabbitmq (port 5672, 15672)
```

**docker-compose.yml:**

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: ekms
      POSTGRES_USER: ekms_user
      POSTGRES_PASSWORD: ekms_pass
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
  
  mongodb:
    image: mongo:7
    ports:
      - "27017:27017"
    volumes:
      - mongo_data:/data/db
  
  neo4j:
    image: neo4j:5
    environment:
      NEO4J_AUTH: neo4j/password123
    ports:
      - "7474:7474"  # HTTP
      - "7687:7687"  # Bolt
    volumes:
      - neo4j_data:/data
  
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
  
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.11.0
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
      - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
    ports:
      - "9200:9200"
    volumes:
      - es_data:/usr/share/elasticsearch/data
  
  qdrant:
    image: qdrant/qdrant:latest
    ports:
      - "6333:6333"
      - "6334:6334"
    volumes:
      - qdrant_data:/qdrant/storage
  
  rabbitmq:
    image: rabbitmq:3-management
    ports:
      - "5672:5672"
      - "15672:15672"  # Management UI
    volumes:
      - rabbitmq_data:/var/lib/rabbitmq

volumes:
  postgres_data:
  mongo_data:
  neo4j_data:
  redis_data:
  es_data:
  qdrant_data:
  rabbitmq_data:
```

### Step 4: Database Migrations

```bash
# Run PostgreSQL migrations
cd backend/services/user-service
go run migrations/migrate.go up

# Seed initial data
go run scripts/seed.go

# Verify
psql -h localhost -U ekms_user -d ekms -c "\dt"
```

### Step 5: Start Backend Services

**Terminal 1 - User Service:**
```bash
cd backend/services/user-service
go mod download
go run cmd/server/main.go
# Listening on :8081
```

**Terminal 2 - Document Service:**
```bash
cd backend/services/document-service
go mod download
go run cmd/server/main.go
# Listening on :8082
```

**Terminal 3 - Search Service:**
```bash
cd backend/services/search-service
go mod download
go run cmd/server/main.go
# Listening on :8083
```

**Terminal 4 - AI Service:**
```bash
cd backend/services/ai-service
python -m venv venv
source venv/bin/activate
pip install -r requirements.txt
uvicorn app.main:app --reload --port 8084
# Listening on :8084
```

**Terminal 5 - Agent Orchestrator:**
```bash
cd backend/services/agent-orchestrator
source venv/bin/activate
pip install -r requirements.txt
uvicorn app.main:app --reload --port 8085
# Listening on :8085
```

### Step 6: Start Frontend

```bash
cd frontend/web
npm install
npm run dev
# Running on http://localhost:3000
```

### Step 7: Verify Setup

```bash
# Health check script
./scripts/health-check.sh

# Expected output:
# ✓ PostgreSQL: Connected
# ✓ MongoDB: Connected
# ✓ Neo4j: Connected
# ✓ Redis: Connected
# ✓ Elasticsearch: Connected
# ✓ Qdrant: Connected
# ✓ RabbitMQ: Connected
# ✓ User Service: Healthy (http://localhost:8081/health)
# ✓ Document Service: Healthy (http://localhost:8082/health)
# ✓ Search Service: Healthy (http://localhost:8083/health)
# ✓ AI Service: Healthy (http://localhost:8084/health)
# ✓ Agent Orchestrator: Healthy (http://localhost:8085/health)
# ✓ Frontend: Running (http://localhost:3000)
```

---

## 3. Development Workflow

### Creating a New Feature

```bash
# 1. Create feature branch
git checkout -b feature/EKMS-123-new-feature

# 2. Make changes
# Edit code, add tests

# 3. Run tests
make test

# 4. Run linters
make lint

# 5. Commit with conventional commits
git commit -m "feat(search): add semantic search filter"

# 6. Push and create PR
git push origin feature/EKMS-123-new-feature
```

### Running Tests

```bash
# Unit tests (all services)
make test-unit

# Integration tests
make test-integration

# E2E tests
make test-e2e

# Specific service
cd backend/services/user-service
go test ./... -v

# Frontend tests
cd frontend/web
npm test
```

### Code Quality

```bash
# Backend (Go)
cd backend/services/user-service
golangci-lint run
go fmt ./...
go vet ./...

# Backend (Python)
cd backend/services/ai-service
black .
flake8 .
mypy .

# Frontend
cd frontend/web
npm run lint
npm run type-check
npm run format
```

---

## 4. Common Development Tasks

### Adding a New Endpoint

**1. Define API Contract (OpenAPI):**

```yaml
# docs/api/openapi.yml
paths:
  /api/v1/documents:
    post:
      summary: Upload document
      requestBody:
        content:
          multipart/form-data:
            schema:
              type: object
              properties:
                file:
                  type: string
                  format: binary
      responses:
        '201':
          description: Document created
```

**2. Implement Service (Go):**

```go
// backend/services/document-service/internal/handler/document.go
package handler

import (
    "github.com/gofiber/fiber/v2"
)

type DocumentHandler struct {
    service DocumentService
}

func (h *DocumentHandler) Upload(c *fiber.Ctx) error {
    file, err := c.FormFile("file")
    if err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "No file provided"})
    }
    
    doc, err := h.service.CreateDocument(c.Context(), file)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }
    
    return c.Status(201).JSON(doc)
}
```

**3. Add Tests:**

```go
// backend/services/document-service/internal/handler/document_test.go
func TestDocumentHandler_Upload(t *testing.T) {
    // Setup
    mockService := new(MockDocumentService)
    handler := &DocumentHandler{service: mockService}
    
    // Test
    // ...
}
```

### Adding a New Agent

**1. Create Agent Class:**

```python
# ai/agents/custom/my_agent.py
from agents.base import BaseAgent, AgentCapability

class MyCustomAgent(BaseAgent):
    def __init__(self):
        super().__init__(
            name="MyCustomAgent",
            capabilities=[AgentCapability.ANALYZE, AgentCapability.NOTIFY]
        )
    
    async def perceive(self, context: Context) -> List[Observation]:
        # Gather information
        observations = []
        # ... implementation
        return observations
    
    async def reason(self, observations: List[Observation]) -> Plan:
        # Decide what to do
        plan = await self.llm.generate_plan(observations)
        return plan
    
    async def act(self, plan: Plan) -> List[Action]:
        # Execute actions
        actions = []
        # ... implementation
        return actions
```

**2. Register Agent:**

```python
# ai/agents/registry.py
from agents.custom.my_agent import MyCustomAgent

AGENTS = {
    'curator': CuratorAgent,
    'retrieval': RetrievalAgent,
    'my_custom': MyCustomAgent,  # Add here
}
```

**3. Test Agent:**

```python
# tests/agents/test_my_agent.py
import pytest
from agents.custom.my_agent import MyCustomAgent

@pytest.mark.asyncio
async def test_my_agent_perceive():
    agent = MyCustomAgent()
    context = create_test_context()
    
    observations = await agent.perceive(context)
    
    assert len(observations) > 0
    assert observations[0].type == "expected_type"
```

### Working with Vector Database

```python
# Example: Add document embeddings
from qdrant_client import QdrantClient
from openai import OpenAI

# Initialize clients
qdrant = QdrantClient(url="http://localhost:6333")
openai_client = OpenAI(api_key="your-key")

# Create collection (first time)
qdrant.create_collection(
    collection_name="documents",
    vectors_config={
        "size": 3072,  # text-embedding-3-large
        "distance": "Cosine"
    }
)

# Generate embedding
def embed_text(text: str) -> list[float]:
    response = openai_client.embeddings.create(
        model="text-embedding-3-large",
        input=text
    )
    return response.data[0].embedding

# Insert document
qdrant.upsert(
    collection_name="documents",
    points=[
        {
            "id": "doc-123",
            "vector": embed_text("Document content here"),
            "payload": {
                "title": "Document Title",
                "author": "Author Name",
                "created_at": "2026-02-17"
            }
        }
    ]
)

# Search
query_vector = embed_text("search query")
results = qdrant.search(
    collection_name="documents",
    query_vector=query_vector,
    limit=10,
    query_filter={
        "must": [
            {"key": "author", "match": {"value": "Author Name"}}
        ]
    }
)
```

### Working with Knowledge Graph

```python
# Example: Add entities and relationships to Neo4j
from neo4j import GraphDatabase

driver = GraphDatabase.driver(
    "bolt://localhost:7687",
    auth=("neo4j", "password123")
)

def create_document_node(tx, doc_id, title, author):
    query = """
    CREATE (d:Document {id: $doc_id, title: $title})
    CREATE (a:Person {name: $author})
    CREATE (a)-[:AUTHORED]->(d)
    """
    tx.run(query, doc_id=doc_id, title=title, author=author)

def find_related_documents(tx, doc_id):
    query = """
    MATCH (d:Document {id: $doc_id})-[:REFERENCES*1..2]-(related:Document)
    RETURN related.title AS title, related.id AS id
    LIMIT 10
    """
    result = tx.run(query, doc_id=doc_id)
    return [{"title": record["title"], "id": record["id"]} for record in result]

# Usage
with driver.session() as session:
    session.execute_write(create_document_node, "doc-123", "AI Guide", "John Doe")
    related = session.execute_read(find_related_documents, "doc-123")
    print(related)
```

---

## 5. Debugging Tips

### Enable Debug Logging

```bash
# Backend (Go)
export LOG_LEVEL=debug
go run cmd/server/main.go

# Backend (Python)
export LOG_LEVEL=DEBUG
uvicorn app.main:app --reload --log-level debug

# Frontend
export NEXT_PUBLIC_DEBUG=true
npm run dev
```

### Using Debugger

**Go (VS Code):**

```json
// .vscode/launch.json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug User Service",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/backend/services/user-service/cmd/server",
            "env": {
                "LOG_LEVEL": "debug"
            }
        }
    ]
}
```

**Python:**

```python
# Add breakpoint
import pdb; pdb.set_trace()

# Or use debugpy for VS Code
import debugpy
debugpy.listen(5678)
debugpy.wait_for_client()
```

### Common Issues

**Issue: Can't connect to database**
```bash
# Check if services are running
docker-compose ps

# Check logs
docker-compose logs postgres

# Restart service
docker-compose restart postgres
```

**Issue: Port already in use**
```bash
# Find process using port
lsof -i :8080

# Kill process
kill -9 <PID>
```

**Issue: OpenAI API rate limit**
```python
# Implement exponential backoff
import time
from openai import RateLimitError

def call_openai_with_retry(func, max_retries=3):
    for attempt in range(max_retries):
        try:
            return func()
        except RateLimitError:
            wait_time = 2 ** attempt
            time.sleep(wait_time)
    raise Exception("Max retries exceeded")
```

---

## 6. Useful Commands

### Makefile

```makefile
# Makefile
.PHONY: help setup start stop test clean

help:
	@echo "Available commands:"
	@echo "  make setup     - Install dependencies"
	@echo "  make start     - Start all services"
	@echo "  make stop      - Stop all services"
	@echo "  make test      - Run all tests"
	@echo "  make clean     - Clean up resources"

setup:
	docker-compose up -d
	cd backend/services/user-service && go mod download
	cd backend/services/ai-service && pip install -r requirements.txt
	cd frontend/web && npm install

start:
	docker-compose up -d
	./scripts/start-services.sh

stop:
	docker-compose down
	./scripts/stop-services.sh

test:
	./scripts/run-tests.sh

clean:
	docker-compose down -v
	rm -rf backend/services/*/tmp
	rm -rf frontend/web/.next
```

### Useful Scripts

**Health Check:**

```bash
#!/bin/bash
# scripts/health-check.sh

check_service() {
    local name=$1
    local url=$2
    
    if curl -f -s -o /dev/null "$url"; then
        echo "✓ $name: Healthy"
    else
        echo "✗ $name: Unhealthy"
        return 1
    fi
}

check_service "User Service" "http://localhost:8081/health"
check_service "Document Service" "http://localhost:8082/health"
check_service "Search Service" "http://localhost:8083/health"
check_service "AI Service" "http://localhost:8084/health"
check_service "Frontend" "http://localhost:3000"
```

**Reset Development Environment:**

```bash
#!/bin/bash
# scripts/reset-dev.sh

echo "Resetting development environment..."

# Stop all services
docker-compose down -v

# Remove data
rm -rf tmp/ logs/

# Restart infrastructure
docker-compose up -d

# Wait for services
sleep 10

# Run migrations
cd backend/services/user-service
go run migrations/migrate.go up

# Seed data
go run scripts/seed.go

echo "✓ Development environment reset complete"
```

---

## 7. Testing Guidelines

### Unit Tests

```go
// backend/services/user-service/internal/service/user_test.go
package service

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

func TestUserService_Create(t *testing.T) {
    // Arrange
    mockRepo := new(MockUserRepository)
    service := NewUserService(mockRepo)
    
    user := &User{Email: "test@example.com"}
    mockRepo.On("Create", mock.Anything, user).Return(nil)
    
    // Act
    err := service.Create(context.Background(), user)
    
    // Assert
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}
```

### Integration Tests

```python
# tests/integration/test_search_flow.py
import pytest
from httpx import AsyncClient

@pytest.mark.integration
async def test_document_upload_and_search():
    async with AsyncClient(base_url="http://localhost:8080") as client:
        # Upload document
        files = {"file": ("test.txt", "Test content", "text/plain")}
        upload_response = await client.post("/api/v1/documents", files=files)
        assert upload_response.status_code == 201
        doc_id = upload_response.json()["id"]
        
        # Wait for indexing
        await asyncio.sleep(2)
        
        # Search for document
        search_response = await client.get("/api/v1/search?q=test")
        assert search_response.status_code == 200
        results = search_response.json()["results"]
        assert any(r["id"] == doc_id for r in results)
```

### E2E Tests

```typescript
// tests/e2e/document-flow.spec.ts
import { test, expect } from '@playwright/test';

test('user can upload and search document', async ({ page }) => {
  // Login
  await page.goto('http://localhost:3000/login');
  await page.fill('[name="email"]', 'test@example.com');
  await page.fill('[name="password"]', 'password123');
  await page.click('button[type="submit"]');
  
  // Upload document
  await page.goto('http://localhost:3000/upload');
  await page.setInputFiles('input[type="file"]', './fixtures/test-doc.pdf');
  await page.click('button:has-text("Upload")');
  await expect(page.locator('.success-message')).toBeVisible();
  
  // Search for document
  await page.fill('[name="search"]', 'test document');
  await page.press('[name="search"]', 'Enter');
  await expect(page.locator('.search-results')).toContainText('test-doc.pdf');
});
```

---

## 8. Contributing Guidelines

### Code Style

**Go:**
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Run `golangci-lint` before committing

**Python:**
- Follow [PEP 8](https://www.python.org/dev/peps/pep-0008/)
- Use `black` for formatting
- Use type hints (enforced by `mypy`)

**TypeScript/React:**
- Follow [Airbnb Style Guide](https://github.com/airbnb/javascript)
- Use ESLint + Prettier
- Functional components with hooks

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(search): add semantic search capability
fix(auth): resolve token expiration issue
docs(api): update OpenAPI specification
test(agents): add unit tests for curator agent
refactor(db): optimize query performance
```

### Pull Request Process

1. Create feature branch from `main`
2. Implement feature with tests
3. Update documentation
4. Run all tests and linters
5. Create PR with description
6. Address review feedback
7. Squash and merge

---

## 9. Resources

### Documentation
- [Architecture Docs](../docs/architecture/)
- [API Reference](../docs/api/)
- [User Guides](../docs/user-guides/)

### Tools
- [Postman Collection](../docs/postman/)
- [Database Diagrams](../docs/diagrams/)
- [Component Library](http://localhost:6006) (Storybook)

### External Resources
- [LangChain Docs](https://python.langchain.com/)
- [OpenAI API Reference](https://platform.openai.com/docs)
- [Neo4j Cypher Manual](https://neo4j.com/docs/cypher-manual/)
- [Qdrant Documentation](https://qdrant.tech/documentation/)

---

## 10. Getting Help

### Team Communication
- **Slack:** #ekms-dev channel
- **Daily Standup:** 9:30 AM (15 min)
- **Tech Lead Office Hours:** Tuesday/Thursday 2-3 PM

### Support
- **Bug Reports:** [GitHub Issues](...)
- **Feature Requests:** [Product Board](...)
- **Security Issues:** security@example.com (private)

---

**Happy Coding! 🚀**

---

*This guide is maintained by the EKMS development team. Last updated: 2026-02-17*
