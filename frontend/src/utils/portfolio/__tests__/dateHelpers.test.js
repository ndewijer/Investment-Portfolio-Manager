/**
 * Date Helpers Tests
 *
 * Tests for date utility functions used in portfolio management.
 * These are pure functions that should achieve 100% coverage.
 */

import { isDateInFuture, formatDisplayDate, getTodayString, toDateString } from '../dateHelpers';

describe('dateHelpers', () => {
  describe('isDateInFuture', () => {
    it('returns true for future dates', () => {
      const futureDate = new Date();
      futureDate.setDate(futureDate.getDate() + 10);
      const futureDateString = futureDate.toISOString().split('T')[0];

      expect(isDateInFuture(futureDateString)).toBe(true);
    });

    it('returns false for past dates', () => {
      const pastDate = new Date();
      pastDate.setDate(pastDate.getDate() - 10);
      const pastDateString = pastDate.toISOString().split('T')[0];

      expect(isDateInFuture(pastDateString)).toBe(false);
    });

    it('returns false for today', () => {
      const today = new Date().toISOString().split('T')[0];

      expect(isDateInFuture(today)).toBe(false);
    });

    it('returns true for null date', () => {
      expect(isDateInFuture(null)).toBe(true);
    });

    it('returns true for undefined date', () => {
      expect(isDateInFuture(undefined)).toBe(true);
    });

    it('returns true for empty string', () => {
      expect(isDateInFuture('')).toBe(true);
    });

    it('handles far future dates correctly', () => {
      expect(isDateInFuture('2099-12-31')).toBe(true);
    });

    it('handles far past dates correctly', () => {
      expect(isDateInFuture('2000-01-01')).toBe(false);
    });

    it('compares dates at midnight (ignores time)', () => {
      // Both dates should be compared at 00:00:00
      const today = new Date();
      today.setHours(23, 59, 59, 999); // End of today
      const todayString = today.toISOString().split('T')[0];

      // Should still be false since we compare dates at midnight
      expect(isDateInFuture(todayString)).toBe(false);
    });

    it('handles dates in YYYY-MM-DD format', () => {
      expect(isDateInFuture('2099-06-15')).toBe(true);
      expect(isDateInFuture('2020-06-15')).toBe(false);
    });

    it('handles end of month dates', () => {
      const nextMonth = new Date();
      nextMonth.setMonth(nextMonth.getMonth() + 1);
      nextMonth.setDate(1); // First day of next month
      const nextMonthString = nextMonth.toISOString().split('T')[0];

      expect(isDateInFuture(nextMonthString)).toBe(true);
    });

    it('handles year boundaries', () => {
      const nextYear = new Date();
      nextYear.setFullYear(nextYear.getFullYear() + 1);
      nextYear.setMonth(0); // January
      nextYear.setDate(1); // First day
      const nextYearString = nextYear.toISOString().split('T')[0];

      expect(isDateInFuture(nextYearString)).toBe(true);
    });
  });

  describe('formatDisplayDate', () => {
    it('formats a valid date string', () => {
      const result = formatDisplayDate('2024-01-15');
      // Result depends on browser locale, so just verify it's a string
      expect(typeof result).toBe('string');
      expect(result.length).toBeGreaterThan(0);
    });

    it('formats dates with different months', () => {
      const dates = ['2024-01-01', '2024-06-15', '2024-12-31'];

      dates.forEach((date) => {
        const result = formatDisplayDate(date);
        expect(typeof result).toBe('string');
        expect(result.length).toBeGreaterThan(0);
      });
    });

    it('formats dates from different years', () => {
      const dates = ['2020-01-01', '2023-06-15', '2024-12-31'];

      dates.forEach((date) => {
        const result = formatDisplayDate(date);
        expect(typeof result).toBe('string');
        expect(result.length).toBeGreaterThan(0);
      });
    });

    it('returns a locale-specific format', () => {
      // The format will depend on the browser's locale
      // In US locale: M/D/YYYY
      // In UK locale: DD/MM/YYYY
      const result = formatDisplayDate('2024-06-15');

      // Verify it contains the key components (year, month, day)
      expect(result).toMatch(/\d/); // Contains digits
      expect(result).toMatch(/\//); // Contains slashes (most locales)
    });

    it('handles edge case dates correctly', () => {
      expect(typeof formatDisplayDate('2000-01-01')).toBe('string');
      expect(typeof formatDisplayDate('2099-12-31')).toBe('string');
    });
  });

  describe('getTodayString', () => {
    it('returns a string in YYYY-MM-DD format', () => {
      const result = getTodayString();

      expect(typeof result).toBe('string');
      expect(result).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    });

    it("returns today's date", () => {
      const result = getTodayString();
      const expected = new Date().toISOString().split('T')[0];

      expect(result).toBe(expected);
    });

    it('returns a date that is not in the future', () => {
      const result = getTodayString();
      const resultDate = new Date(result);
      const now = new Date();

      expect(resultDate.getTime()).toBeLessThanOrEqual(now.getTime());
    });

    it('returns consistent format across calls', () => {
      const result1 = getTodayString();
      const result2 = getTodayString();

      // If called within the same day, should be same
      expect(result1).toBe(result2);
    });

    it('returns valid date components', () => {
      const result = getTodayString();
      const [year, month, day] = result.split('-').map(Number);

      expect(year).toBeGreaterThanOrEqual(2000);
      expect(year).toBeLessThanOrEqual(2100);
      expect(month).toBeGreaterThanOrEqual(1);
      expect(month).toBeLessThanOrEqual(12);
      expect(day).toBeGreaterThanOrEqual(1);
      expect(day).toBeLessThanOrEqual(31);
    });

    it('has 4-digit year, 2-digit month, 2-digit day', () => {
      const result = getTodayString();
      const parts = result.split('-');

      expect(parts).toHaveLength(3);
      expect(parts[0]).toHaveLength(4); // Year
      expect(parts[1]).toHaveLength(2); // Month
      expect(parts[2]).toHaveLength(2); // Day
    });
  });

  describe('toDateString', () => {
    it('converts Date object to YYYY-MM-DD string', () => {
      const date = new Date('2024-01-15T10:30:00Z');
      const result = toDateString(date);

      expect(result).toBe('2024-01-15');
    });

    it('extracts date from ISO string with timestamp', () => {
      const dateString = '2024-01-15T10:30:00Z';
      const result = toDateString(dateString);

      expect(result).toBe('2024-01-15');
    });

    it('handles date string without time component', () => {
      const dateString = '2024-01-15';
      const result = toDateString(dateString);

      expect(result).toBe('2024-01-15');
    });

    it('works with Date objects from different months', () => {
      const dates = [
        new Date('2024-01-01T00:00:00Z'),
        new Date('2024-06-15T12:00:00Z'),
        new Date('2024-12-31T23:59:59Z'),
      ];

      const expected = ['2024-01-01', '2024-06-15', '2024-12-31'];

      dates.forEach((date, index) => {
        expect(toDateString(date)).toBe(expected[index]);
      });
    });

    it('works with date strings from different formats', () => {
      const testCases = [
        { input: '2024-01-15T00:00:00Z', expected: '2024-01-15' },
        { input: '2024-06-15T10:30:45.123Z', expected: '2024-06-15' },
        { input: '2024-12-31T23:59:59.999Z', expected: '2024-12-31' },
        { input: '2024-12-31', expected: '2024-12-31' },
      ];

      testCases.forEach(({ input, expected }) => {
        expect(toDateString(input)).toBe(expected);
      });
    });

    it('handles dates with different timezones consistently', () => {
      const date = new Date('2024-06-15T12:00:00Z');
      const result = toDateString(date);

      // Should always extract the date portion
      expect(result).toMatch(/^\d{4}-\d{2}-\d{2}$/);
    });

    it('preserves date when converting from Date object', () => {
      const original = '2024-06-15';
      const date = new Date(original + 'T00:00:00Z');
      const result = toDateString(date);

      expect(result).toBe(original);
    });

    it('preserves date when extracting from ISO string', () => {
      const original = '2024-06-15';
      const withTime = original + 'T10:30:45.123Z';
      const result = toDateString(withTime);

      expect(result).toBe(original);
    });

    it('handles edge case dates', () => {
      expect(toDateString('2000-01-01T00:00:00Z')).toBe('2000-01-01');
      expect(toDateString('2099-12-31T23:59:59Z')).toBe('2099-12-31');
      expect(toDateString(new Date('2024-02-29T12:00:00Z'))).toBe('2024-02-29'); // Leap year
    });

    it('returns consistent format for both input types', () => {
      const dateStr = '2024-06-15';
      const dateObj = new Date(dateStr + 'T00:00:00Z');

      expect(toDateString(dateStr)).toBe('2024-06-15');
      expect(toDateString(dateObj)).toBe('2024-06-15');
    });
  });
});
