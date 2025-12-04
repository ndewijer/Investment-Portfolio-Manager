/**
 * Sample test to verify Jest configuration is working
 */

describe('Jest Configuration', () => {
  test('Jest is configured and working', () => {
    expect(true).toBe(true);
  });

  test('Jest DOM matchers are available', () => {
    const element = document.createElement('div');
    element.textContent = 'Hello World';
    document.body.appendChild(element);

    expect(element).toBeInTheDocument();
    expect(element).toHaveTextContent('Hello World');
  });

  test('window.matchMedia mock is working', () => {
    const mediaQuery = window.matchMedia('(min-width: 768px)');
    expect(mediaQuery).toBeDefined();
    expect(mediaQuery.matches).toBe(false);
  });
});
