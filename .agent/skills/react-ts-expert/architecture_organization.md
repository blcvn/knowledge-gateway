# Code Organization & Architecture

## Project Structure & Scalability
- **Feature-Sliced Design / Domain-Driven Structure:** Organizing code by features or domains (e.g., `features/auth`, `features/dashboard`) rather than by technical types (e.g., `components`, `hooks`, `services`) to improve discoverability and scalability.
- **Separation of Concerns:** Decoupling UI presentation components (dumb/pure) from business logic and data fetching components (smart/container).
- **Custom Hooks:** Abstracting complex state logic, side effects, and reusable behaviors into highly focused custom hooks.

## State Management Strategy
- **Local vs. Global State:** Knowing exactly when to use `useState`/`useReducer` for local UI state versus when to elevate to global state.
- **Server State Management:** Utilizing tools like React Query (TanStack Query) or SWR for caching, deduplicating, and managing asynchronous server data independently from UI state.
- **Global UI State:** Using lightweight libraries like Zustand or Jotai, or leveraging React Context API for low-frequency global updates (e.g., themes, auth state) without the boilerplate of Redux.

## TypeScript Excellence
- **Strict Typing:** Enabling `strict` mode in `tsconfig.json`. No `any` types; using `unknown` when necessary.
- **Advanced Types:** Utilizing utility types (`Partial`, `Pick`, `Omit`, `Record`), generic components, and conditional types for highly flexible APIs.
- **API Contracts:** Sharing TypeScript interfaces or generating types from Swagger/GraphQL to ensure exact data syncing between backend and frontend.
