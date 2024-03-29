import type { DateString } from '../models/Date.ts';
import type { LogFile } from '../models/LogFile.ts';
import { createHash, ensureFile, walk } from '../dependencies.ts';
import { isDateString } from '../models/Date.ts';
import { LogFileNameFactory } from '../models/factory/LogFileName.ts';
import { compareDatesInDescent } from './hash.ts';
import { pathResolve } from './path.ts';

const LOG_FILE_INDENT_SPACE = 2;

const logFileNameFactory = new LogFileNameFactory(Date);

export const statLogFile = async (
  logDir: string,
  targetDay?: DateString, /* yyyy-MM-dd */
): Promise<LogFile | undefined> => {
  const targetFileName = logFileNameFactory.create(targetDay);

  const targetFileFullPath = pathResolve([
    logDir,
    targetFileName.withExtension,
  ]);

  try {
    const stat: Deno.FileInfo = await Deno.lstat(targetFileFullPath);

    if (stat.isFile === false) {
      throw new Error('Target log file is already exists and not file.');
    }

    const targetLog: LogFile['body'] = JSON.parse(
      await Deno.readTextFile(targetFileFullPath),
    );

    return {
      path: targetFileFullPath,
      fileName: targetFileName.name,
      body: targetLog,
    };
  } catch (error: unknown) {
    if (error instanceof Deno.errors.NotFound) {
      return undefined
    }
    throw error
  }
};

export const getLogFile = async (
  logDir: string,
  targetDay?: DateString, /* yyyy-MM-dd */
): Promise<LogFile> => {
  const targetFileName = logFileNameFactory.create(targetDay);

  const targetFileFullPath = pathResolve([
    logDir,
    targetFileName.withExtension,
  ]);

  // Check if a log file has been created.
  try {
    const stat = await Deno.lstat(targetFileFullPath);
    if (stat.isFile === false) {
      throw new Error('Target log file is already exists and not file.');
    }
  } catch (error) {
    if (error instanceof Deno.errors.NotFound) {
      // create new file
      await ensureFile(targetFileFullPath);
      const newLog: LogFile['body'] = {
        hash: createHash('md5').update(new Date().toString()).toString(),
        freezed: false,
        items: [],
      };
      updateLogFile(logDir, targetFileName.name, newLog);

      // return information on file that new created.
      return {
        path: targetFileFullPath,
        fileName: targetFileName.name,
        body: newLog,
      };
    } else {
      throw error;
    }
  }

  const targetLog: LogFile['body'] = JSON.parse(
    await Deno.readTextFile(targetFileFullPath),
  );

  // return information on file that already exists.
  return {
    path: targetFileFullPath,
    fileName: targetFileName.name,
    body: targetLog,
  };
};

export const updateLogFile = async (
  logDir: string,
  targetDay: DateString, /* yyyy-MM-dd */
  body: LogFile['body'],
): Promise<LogFile> => {
  if (body.freezed === true) {
    throw new Error('This log file is freezed. No updates.');
  }
  const targetFileName = logFileNameFactory.create(targetDay);

  const targetFileFullPath = pathResolve([
    logDir,
    targetFileName.withExtension,
  ]);

  const newLog = new TextEncoder().encode(
    JSON.stringify(body, null, LOG_FILE_INDENT_SPACE),
  );

  await Deno.writeFile(targetFileFullPath, newLog);

  return {
    path: targetFileFullPath,
    fileName: targetDay,
    body,
  };
};

export const listLogFile = async (
  logDir: string,
): Promise<
  Array<Pick<LogFile, 'path' | 'fileName'>>
> => {
  const listLog: Array<Pick<LogFile, 'path' | 'fileName'> | undefined> = [];
  const files = walk(logDir, {
    maxDepth: 1,
    includeDirs: false,
    exts: ['json'],
  });

  for await (const file of files) {
    const logFileDate = file.name.split('.')[0];
    const logFileName = logFileNameFactory.create(logFileDate as DateString);

    listLog.push(
      isDateString(logFileDate)
        ? {
          path: pathResolve([
            logDir,
            file.path,
          ]),
          fileName: logFileName.name,
        }
        : undefined,
    );
  }

  return listLog
    .filter((item): item is Pick<LogFile, 'path' | 'fileName'> =>
      item !== undefined
    )
    .sort((a, b) => compareDatesInDescent(a.fileName, b.fileName));
};

// export const deleteLogFile = (
//   logDir: string,
//   targetDay: DateString, /* yyyy-MM-dd */
// ): Pick<LogFile, 'path' | 'fileName'> => {};
