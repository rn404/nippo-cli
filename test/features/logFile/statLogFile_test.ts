import { assertEquals, assertExists } from '../../dependencies.ts'
import { statLogFile } from '../../../src/features/logFile.ts'

const DUMMY_LOG_DIR = './test/dummy_log_dir'

Deno.test('should return undefined when the file is not present', async () => {
  const actual = await statLogFile(
    DUMMY_LOG_DIR,
    '2022-10-01'
  )
  assertEquals(actual, undefined)
})

Deno.test('should return undefined if the file exists but has an incorrect extension', async () => {
  const actual = await statLogFile(
    DUMMY_LOG_DIR,
    '2022-01-16'
  )
  assertEquals(actual, undefined)
})

Deno.test('should return undefined in the case where the file exists but is a folder', async () => {
  const actual = await statLogFile(
    DUMMY_LOG_DIR,
    '2022-01-17'
  )
  assertEquals(actual, undefined)
})

Deno.test('should return undefined when the file is present and meets the criteria', async () => {
  const actual = await statLogFile(
    DUMMY_LOG_DIR,
    '2022-01-18'
  )
  assertEquals(actual, undefined)
})

Deno.test('should return LogFile when the file exists', async () => {
  const actual = await statLogFile(
    DUMMY_LOG_DIR,
    '2022-01-15'
  )
  assertExists(actual?.fileName)
})
