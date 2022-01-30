import { Command, HelpCommand } from './dependencies.ts';
import { APP_NAME, VERSION } from './const.ts';
import { addCommand } from './commands/add.ts';
import { endCommand } from './commands/end.ts';
import { deleteCommand } from './commands/delete.ts';
import { listCommand } from './commands/list.ts';
import { clearCommand } from './commands/clean.ts';

await new Command()
  .name(APP_NAME)
  .version(VERSION)
  .default('help')
  .command(
    'add <contents:string>',
    new Command()
      .option(
        '-m, --memo',
        'Add contents like memo item.',
      )
      .description('Add contents to nippo log.')
      .action(async (options, contents) => {
        await addCommand(options, contents);
      }),
  )
  .command(
    'end <hash:string>',
    new Command()
      .description('end to task.')
      .action(async (_options, hash) => {
        await endCommand(hash);
      }),
  )
  .command(
    'del <hash:string>',
    new Command()
      .description('delete task.')
      .action(async (_options, hash) => {
        await deleteCommand(hash);
      }),
  )
  .command(
    'list',
    new Command()
      .description('list all logs.')
      .option(
        '-a, --all',
        'show all logs',
      )
      .option(
        '-s, --stat',
        'show summary of list',
      )
      .action(async (options) => {
        await listCommand(options);
      }),
  )
  .command(
    'clear',
    new Command()
      .description('delete log')
      // .option(
      //   '-a, --all',
      //   'clear all logs',
      // )
      .action(async (options) => {
        await clearCommand(options);
      }),
  )
  .command(
    'help',
    new HelpCommand(),
  )
  .parse(Deno.args);
