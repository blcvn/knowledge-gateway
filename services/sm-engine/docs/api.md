# sm-engine — API Reference

> **Service**: `sm-engine`  
> **Status**: Ready

---

## gRPC Service Definitions

`sm-engine` consolidates three distinct gRPC services exposed on a single port (9071), handling document ingestion, memory extraction and management, and profile traits.

### 1. SmDocumentService

Handles document CRUD operations and chunking pipelines.

```protobuf
service SmDocumentService {
  rpc CreateDocument(CreateDocumentRequest) returns (Document);
  rpc GetDocument(GetDocumentRequest) returns (Document);
  rpc DeleteDocument(DeleteDocumentRequest) returns (Empty);
  rpc ListDocuments(ListDocumentsRequest) returns (ListDocumentsResponse);
  rpc GetChunks(GetChunksRequest) returns (ChunksResponse);
}
```

### 2. SmMemoryService

Handles memory extraction, relationship tracking, and the forgetting curve.

```protobuf
service SmMemoryService {
  rpc CreateMemory(CreateMemoryRequest) returns (Memory);
  rpc GetMemory(GetMemoryRequest) returns (Memory);
  rpc ForgetMemory(ForgetMemoryRequest) returns (Empty);
  rpc ListMemories(ListMemoriesRequest) returns (ListMemoriesResponse);
  rpc CreateRelation(CreateRelationRequest) returns (Relation);
}
```

### 3. SmProfileService

Handles user profile management, both static and dynamic traits.

```protobuf
service SmProfileService {
  rpc GetProfile(GetProfileRequest) returns (Profile);
  rpc UpdateProfile(UpdateProfileRequest) returns (Profile);
  rpc GetDynamicTraits(GetTraitsRequest) returns (TraitsResponse);
}
```

## Authentication

All endpoints require valid JWT or API key via `vnp-gateway`.
Tenant isolation enforced via `x-tenant-id` gRPC metadata.
