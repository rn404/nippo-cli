## motivation
- cli ツール作りたい
- 独り言メモツールしたい(日報補助ツール)

## command
```
sava add 'message'
```

構想は gist に
https://gist.github.com/rn404/decf010fc48d7d8688116af0f4427b44

### Objects
* LogFile > Log > Item (MemoItem, TaskItem)

### Architecture
* Command -- Feature -- LogFile
* Feature -- LogFile
  * LogFileInterface
* Command -- Feature
  * Models(Class instance)


## Usage (developer)
```
# command help
deno run src/sava.ts

# Add memo
deno run --allow-read --allow-write src/sava.ts add '次何するか決める'

# List todos
deno run --allow-read --allow-write src/sava.ts list
```

### Formatter

```
deno fmt -c .config/deno.jsonc 
```