# Zep Core API

## Overview
The Zep Core Service consolidates User, Thread, and Memory management into a single high-performance binary, achieving sub-200ms latency for synchronous context operations.

## gRPC Services (Port 9061)

### ZepUserService
Manages user profiles and metadata.
```protobuf
rpc CreateUser(CreateUserRequest) returns (User);
rpc GetUser(GetUserRequest) returns (User);
rpc UpdateUser(UpdateUserRequest) returns (User);
rpc DeleteUser(DeleteUserRequest) returns (Empty);
rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
```

### ZepThreadService
Manages conversation sessions.
```protobuf
rpc CreateThread(CreateThreadRequest) returns (Thread);
rpc GetThread(GetThreadRequest) returns (Thread);
rpc UpdateThread(UpdateThreadRequest) returns (Thread);
rpc EndThread(EndThreadRequest) returns (Thread);
rpc ListThreads(ListThreadsRequest) returns (ListThreadsResponse);
```

### ZepMemoryService
Ingests messages and retrieves augmented context.
```protobuf
rpc PutMemory(PutMemoryRequest) returns (PutMemoryResponse);
rpc GetMemory(GetMemoryRequest) returns (GetMemoryResponse);
rpc DeleteMemory(DeleteMemoryRequest) returns (Empty);
```

## Workflow

### PutMemory (Ingestion)
1. Internally routes to `ThreadService.UpsertSession`.
2. Inserts message into PostgreSQL.
3. Publishes `zep.memory.messages.ingested` to NATS (Async to `zep-graph`).
4. Returns instantly (≤200ms target).

### GetMemory (Retrieval)
1. Retrieves raw session messages from DB.
2. Calls `zep-search` via gRPC for relevant facts/graph nodes.
3. Overlays external facts onto the raw message context.
4. Returns the enriched memory context payload.
