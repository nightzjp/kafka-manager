export function confirmationMatches(value: string, expected?: string): boolean {
  return expected === undefined || value === expected;
}
