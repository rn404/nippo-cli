import { homeDir, join } from '../dependencies.ts';
import { LOG_DIR } from '../const.ts';

export const pathResolve = (path: Array<string>): string => {
  return join(
    ...path,
  );
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
