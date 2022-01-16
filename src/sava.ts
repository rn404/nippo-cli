import { Command, HelpCommand } from './dependencies.ts';
import {
  APP_NAME,
  VERSION
} from './const.ts';

await new Command()
  .name(APP_NAME)
  .version(VERSION)
  .default('help')
  .command(
    'add <contents:string>',
    new Command()
      .option(
        '-m, --memo',
        'Add contents like memo item.'
      )
      .description('Add contents to nippo log.')
      .action((options, contents) => {
        console.log('addCommand', { options, contents })
      })
  )
  .command(
    'end <hash:string>',
    new Command()
      .description('end to task.')
      .action((_options, hash) => {
        console.log('endCommand', { hash })
      })
  )
  .command(
    'delete <hash:string>',
    new Command()
      .description('delete task.')
      .action((_options, hash) => {
        console.log('deleteItemCommand', { hash })
      })
  )
  .command(
    'list',
    new Command()
      .description('list all logs.')
      .option(
        '-a, --all',
        'show all logs'
      )
      .option(
        '-s, --stat',
        'show summary of list'
      )
      .action((options) => {
        console.log('listCommand', { options })
      })
  )
  .command(
    'clean',
    new Command()
      .description('delete log')
      .option(
        '-a, --all',
        'clean all logs'
      )
      .action((options) => {
        console.log('cleanCommand', { options })
      })
  )
  .command(
    'help',
    new HelpCommand()
  )
  .parse(Deno.args);
