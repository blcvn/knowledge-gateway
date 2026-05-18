import { describe, it, expect } from 'vitest';

describe('App Smoke Test', () => {
  it('should verify the test framework is functional', () => {
    expect(1 + 1).toBe(2);
  });

  it('should confirm environment variables are defined', () => {
    expect(typeof import.meta.env).toBe('object');
  });
});
