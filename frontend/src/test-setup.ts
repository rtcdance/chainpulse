import '@testing-library/jest-dom'

// jsdom does not implement URL.createObjectURL / revokeObjectURL. src/lib/export.ts
// uses both to trigger a browser download, which is correct in a browser, so the
// stubs belong here rather than in the component under test.
//
// They were missing entirely, which meant `npm run test` could not run at all:
// vitest reported "Cannot find dependency 'jsdom'" and collected zero tests,
// because jsdom, @testing-library/react and @testing-library/jest-dom were
// declared in neither package.json nor the lockfile. Nothing ran the frontend
// tests, so nothing said so.
//
// Guarded with typeof so a real implementation is never overwritten.
if (typeof URL.createObjectURL !== 'function') {
  URL.createObjectURL = () => 'blob:jsdom-stub'
}
if (typeof URL.revokeObjectURL !== 'function') {
  URL.revokeObjectURL = () => {}
}
