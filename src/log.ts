import { join } from 'https://deno.land/std@0.100.0/path/mod.ts'
import { ensureDir } from 'https://deno.land/std@0.120.0/fs/ensure_dir.ts'
import { ensureFile } from 'https://deno.land/std@0.120.0/fs/ensure_file.ts'
import format from 'https://deno.land/x/date_fns@v2.22.1/format/index.js'
import { createHash } from 'https://deno.land/std@0.77.0/hash/mod.ts'
import { Log, LogFileInfo, LogItem } from './models/Log.ts'
import { MemoItem, createMemoItem } from './models/MemoItem.ts'
import { TaskItem, createTaskItem } from './models/TaskItem.ts'

const LOG_DIR = '.log'
const LOG_FILE_EXT = 'json'


export const getCurrentFile = async (): Promise<LogFileInfo> => {
  const currentTime = new Date()
  const createdAt = currentTime.toISOString()
  const todayLogFileName = format(currentTime, 'yyyyMMdd', {}) + `.${LOG_FILE_EXT}`
  const path = join(Deno.cwd(), LOG_DIR)

  try {
    const stat = await Deno.lstat(join(path, todayLogFileName))
    if (stat.isFile === false) {
      throw new Error('Log is not file.')
    }
  } catch(error) {
    if (error instanceof Deno.errors.NotFound) {
      await create(path, todayLogFileName)
      return {
        path,
        fileName: todayLogFileName,
        createdAt,
        data: {}
      }
    }
  }

  const resource: { [hash: string]: LogItem } =
    JSON.parse(await Deno.readTextFile(join(path, todayLogFileName)))

  return {
    path,
    fileName: todayLogFileName,
    createdAt,
    data: resource
  }
}

export const create = async (path: string, fileName: string): Promise<void> => {
  await ensureDir(path)
  await ensureFile(join(path, fileName))

  const encoder = new TextEncoder()
  const data = encoder.encode(JSON.stringify({}))
  await Deno.writeFile(join(path, fileName), data)
}

export const update = async (content = {}, path: string, fileName: string): Promise<void> => {
  const targetFile = join(path, fileName)
  const encoder = new TextEncoder()
  const data = encoder.encode(JSON.stringify(content))
  await Deno.writeFile(targetFile, data)
}

export const addItem = async (newItem: LogItem, oldData: Log): Promise<Log> => {
  const data = Object.assign({}, oldData)
  const hash = createHash('md5')
  hash.update(newItem.createdAt.toString())
  const hashKey = hash.toString()
  data[hashKey] = newItem
  return data
}

export const parse = (log: Log): {
  tasks: Array<TaskItem>,
  memos: Array<MemoItem>
} => {
  const tasks: Array<TaskItem> = []
  const memos: Array<MemoItem> = []

  Object.keys(log).forEach((hashKey) => {
    const item = log[hashKey]
    if (item.closed !== undefined) {
      tasks.push(createTaskItem(
        item.content,
        item.createdAt,
        item.updatedAt,
        item.closed === 'true'
      ))
    } else {
      memos.push(createMemoItem(
        item.content,
        item.createdAt,
        item.updatedAt,
      ))
    }
  })

  // TODO sort する

  return {
    tasks, memos
  }
}
