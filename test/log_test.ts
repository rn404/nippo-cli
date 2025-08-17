import { assertEquals } from './dependencies.ts';
import { join } from '../src/dependencies.ts';

import { getCurrentFile } from '../src/log.ts';

Deno.test('getCurrentFile', async () => {
  const logDir = 'test/dummy_log_dir';
  const currentTime = new Date();

  const { path, fileName, createdAt, data } = await getCurrentFile(logDir);

  const expectedPath = join(Deno.cwd(), logDir);
  // Using native date formatting instead of removed format function
  const expectedFileName = `${currentTime.getFullYear()}${
    (currentTime.getMonth() + 1).toString().padStart(2, '0')
  }${currentTime.getDate().toString().padStart(2, '0')}.json`;
  const expectedCreatedAt = currentTime.toISOString();

  assertEquals(path, expectedPath);
  assertEquals(fileName, expectedFileName);
  assertEquals(createdAt, expectedCreatedAt);
  assertEquals(data, {});
});
