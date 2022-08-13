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

# Add todo item
deno run -A src/sava.ts add <message>

# Finish todo item
deno run -A src/sava.ts end <hash>

# Add memo item
deno run -A src/sava.ts add -m <message>

# Delete item
deno run -A src/sava.ts del <hash>

# List log items
deno run -A src/sava.ts list
```

### Formatter

```
./scripts/format.sh
```