import { assertEquals } from './dependencies.ts'
import { format, join } from '../src/dependencies.ts'

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
