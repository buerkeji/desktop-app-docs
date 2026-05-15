declare global {
  interface Window {
    go?: {
      main?: {
        App?: Record<string, (...args: unknown[]) => Promise<unknown>>;
      };
    };
  }
}

export function hasAppMethod(method: string): boolean {
  return typeof window.go?.main?.App?.[method] === 'function';
}

export async function invokeApp<T>(method: string, ...args: unknown[]): Promise<T> {
  const handler = window.go?.main?.App?.[method];

  if (typeof handler !== 'function') {
    throw new Error(`Wails bridge method not available: ${method}`);
  }

  return (await handler(...args)) as T;
}

export {};
