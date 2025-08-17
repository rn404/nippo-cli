import { homeDir, join } from '../dependencies.ts';
import { LOG_DIR } from '../const.ts';

export const pathResolve = (path: string[]): string => {
  // TODO(@rn404) Consider if this wrapper function is necessary
  if (path.length === 0) return '';
  return join(...(path as [string, ...string[]]));
};

export const rootDir = (): string => {
  const root = homeDir();
  return root ?? Deno.cwd();
};

export const logDir = (): string => {
  return join(
    rootDir(),
    LOG_DIR,
  );
};
