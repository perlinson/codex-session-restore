# codex-session-restore

`codex-session-restore` 是一个 Go 编写的 Codex Desktop 会话 provider 修复工具。

它会读取当前用户的 `.codex/config.toml`，获取当前 `model_provider`，然后把 `state_5.sqlite` 中不属于当前 provider 的 sessions 批量切换到当前 provider，并同步修复对应 rollout JSONL 和 `session_index.jsonl`。

## 默认行为

双击可执行文件，或无参数运行时，默认执行：

```bash
codex-session-restore switch-current-provider
```

执行结果会写入可执行文件旁边的 `codex-session-restore.log`。如果该目录不可写，则写入用户主目录。

## 命令

```bash
codex-session-restore
codex-session-restore switch-current-provider
codex-session-restore switch-current-provider --dry-run
codex-session-restore switch-current-provider --include-archived
codex-session-restore inspect --limit 20
codex-session-restore search --keyword project
```

## 安全策略

- 修改前会备份 `state_5.sqlite`、`session_index.jsonl` 和目标 rollout JSONL。
- `--dry-run` 只预览，不写入文件。
- 默认只处理未归档 sessions。
- `--include-archived` 会同时处理已归档 sessions，并将它们恢复为未归档。

## 构建

```bash
go test ./...
go build ./cmd/codex-session-restore
```

多平台构建：

```bash
mkdir -p dist
GOOS=darwin GOARCH=amd64 go build -o dist/codex-session-restore-darwin-amd64 ./cmd/codex-session-restore
GOOS=darwin GOARCH=arm64 go build -o dist/codex-session-restore-darwin-arm64 ./cmd/codex-session-restore
GOOS=windows GOARCH=amd64 go build -o dist/codex-session-restore-windows-amd64.exe ./cmd/codex-session-restore
GOOS=windows GOARCH=arm64 go build -o dist/codex-session-restore-windows-arm64.exe ./cmd/codex-session-restore
GOOS=linux GOARCH=amd64 go build -o dist/codex-session-restore-linux-amd64 ./cmd/codex-session-restore
GOOS=linux GOARCH=arm64 go build -o dist/codex-session-restore-linux-arm64 ./cmd/codex-session-restore
```
