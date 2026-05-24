# Performance Optimization

## React Rendering Lifecycle
- **Memoization Mastery:** Properly using `React.memo`, `useMemo`, and `useCallback` to prevent unnecessary re-renders. Understanding the cost of memoization and avoiding its overuse.
- **State Colocation:** Moving state down the component tree as close as possible to where it is needed to localize re-renders.
- **Concurrent Features:** Leveraging React 18+ features like `useTransition` and `useDeferredValue` to keep the UI responsive during heavy state updates.

## Asset & Bundle Optimization
- **Code Splitting & Lazy Loading:** Using `React.lazy()` and dynamic `import()` to split the application into smaller chunks and load routes or heavy components on demand.
- **Tree Shaking:** Ensuring imports are structured so that unused code from libraries is stripped out by the bundler (e.g., Webpack, Vite, Rollup).
- **Image Optimization:** Serving properly sized images, using modern formats (WebP/AVIF), and implementing lazy loading (`loading="lazy"`) for images outside the initial viewport.

## Core Web Vitals
- **LCP (Largest Contentful Paint):** Optimizing the critical rendering path, preloading key assets, and utilizing Server-Side Rendering (SSR) or Static Site Generation (SSG) with frameworks like Next.js where appropriate.
- **CLS (Cumulative Layout Shift):** Defining explicit dimensions for media (images, videos) and avoiding dynamic injection of content above existing content.
- **INP (Interaction to Next Paint):** Minimizing long tasks on the main thread and deferring non-critical JavaScript execution.
