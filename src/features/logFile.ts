import { createHash, ensureFile, join, walk } from '../dependencies.ts';
import { DateString } from '../models/Date.ts';
import { LogFileName } from '../models/LogFileName.ts';
import { LogFile } from '../models/LogFile.ts';
import { compareDatesInDescent } from './hash.ts';

const LOG_FILE_INDENT_SPACE = 2;

export const getLogFile = async (
  logDir: string,
  targetDay?: DateString, /* yyyy-MM-dd */
): Promise<LogFile> => {
  const targetFileName = new LogFileName(
    targetDay === undefined ? new Date() : new Date(targetDay),
  );

  // TODO path resolve
  const targetFileFullPath = join(
    Deno.cwd(),
    logDir,
    targetFileName.withExtension,
  );

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
  const targetFileName = new LogFileName(new Date(targetDay));

  // TODO path resolve
  const targetFileFullPath = join(
    Deno.cwd(),
    logDir,
    targetFileName.withExtension,
  );

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
    const logFileDate = new Date(file.name.split('.')[0]);

    listLog.push(
      Number.isNaN(logFileDate.getDate()) ? undefined : {
        path: join(
          Deno.cwd(),
          file.path,
        ),
        fileName: new LogFileName(logFileDate).name,
      },
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
