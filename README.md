## motivation
- cli ツール作りたい
- 独り言メモツールしたい(日報補助ツール)

## command 暫定
アプリの名前なににするかまだ未定

構想は gist に移動
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
deno run --allow-read --allow-write src/todo.ts todo '次何するか決める' 
deno run --allow-read --allow-write src/todo.ts todo list
```

### Formatter

```
deno fmt -c .config/deno.jsonc 
```