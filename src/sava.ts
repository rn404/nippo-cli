import { Command, HelpCommand } from './dependencies.ts';
import { APP_NAME, VERSION } from './const.ts';
import { addCommand } from './commands/add.ts';
import { endCommand } from './commands/end.ts';
import { deleteCommand } from './commands/delete.ts';
import { listCommand } from './commands/list.ts';
import { clearCommand } from './commands/clear.ts';

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
      // deno-lint-ignore no-explicit-any
      .action(async (options: any, contents: any) => {
        await addCommand(options, contents);
      }),
  )
  .command(
    'end <hash:string>',
    new Command()
      .description('end to task.')
      // deno-lint-ignore no-explicit-any
      .action(async (_options: any, hash: any) => {
        await endCommand(hash);
      }),
  )
  .command(
    'del <hash:string>',
    new Command()
      .description('delete task.')
      // deno-lint-ignore no-explicit-any
      .action(async (_options: any, hash: any) => {
        await deleteCommand(hash);
      }),
  )
  .command(
    'list [hash:string]',
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
      // deno-lint-ignore no-explicit-any
      .action(async (options: any, hash: any) => {
        await listCommand(options, hash);
      }),
  )
  .command(
    'clear',
    new Command()
      .description('delete log')
      .option(
        '-a, --all',
        'clear all logs',
      )
      // deno-lint-ignore no-explicit-any
      .action(async (options: any) => {
        await clearCommand(options);
      }),
  )
  .command(
    'help',
    new HelpCommand(),
  )
  .parse(Deno.args);
