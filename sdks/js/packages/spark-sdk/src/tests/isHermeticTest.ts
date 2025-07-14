/**
 * Node.js only - A hermetic test environment is identified by either:
 * 1. The existence of a marker file at /tmp/spark_hermetic (Node.js only)
 * 2. The HERMETIC_TEST environment variable being set to "true"
 * See index.node.ts for implementation.
 */

let isHermeticTest = false;

export function getIsHermeticTest() {
  return isHermeticTest;
}

export function setIsHermeticTest(value: boolean) {
  isHermeticTest = value;
}
