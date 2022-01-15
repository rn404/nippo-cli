import { assertEquals } from 'https://deno.land/std@0.121.0/testing/asserts.ts';
import format from 'https://deno.land/x/date_fns@v2.22.1/format/index.js';
import { join } from 'https://deno.land/std@0.100.0/path/mod.ts';
import { getCurrentFile } from '../src/log.ts'

Deno.test('getCurrentFile', async () => {
  const logDir = 'test/dummy_log_dir'
  const currentTime = new Date()

  const { path, fileName, createdAt, data } = await getCurrentFile(logDir)

  const expectedPath = join(Deno.cwd(), logDir)
  const expectedFileName = `${format(currentTime, 'yyyyMMdd', {})}.json`
  const expectedCreatedAt = currentTime.toISOString()

  assertEquals(path, expectedPath)
  assertEquals(fileName, expectedFileName)
  assertEquals(createdAt, expectedCreatedAt)
  assertEquals(data, {})
})
