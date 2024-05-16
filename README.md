## Motivation / Background
エンジニアはほとんどの時間を console をみて過ごしている.
作業時間やちょっとした思考をメモしておくのに他ツールとのスイッチはコストが高いと考えます.
作業と並行して気軽にメモを残していけるツールはメリットが大きいと思います.
(作業時間計測も工数管理の観点では非常に関心のあることだと思います)

以上のことを簡単に示すと
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
