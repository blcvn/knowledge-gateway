---
id: DOC-S02
service: sm-document
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-document — API Reference

## gRPC Service Definition

```protobuf
service SmDocumentService {
  rpc CreateDocument(CreateDocumentRequest) returns (Document);
  rpc GetDocument(GetDocumentRequest) returns (Document);
  rpc DeleteDocument(DeleteDocumentRequest) returns (Empty);
  rpc ListDocuments(ListDocumentsRequest) returns (ListDocumentsResponse);
  rpc GetChunks(GetChunksRequest) returns (ChunksResponse);
}
```

## RPCs: CreateDocument, GetDocument, DeleteDocument, ListDocuments, GetChunks

## NATS Events

Published: `sm.document.created` → sm-memory (extract facts), sm-search (index).
Published: `sm.document.deleted` → sm-memory (cleanup), sm-search (deindex).
