# go-pkg-tddcheck

`tddcheck` 是一个面向 Go 项目的架构规则检查器。它通过静态扫描约束分层目录、文件命名、声明内容和跨层 import，并从同一次扫描生成架构索引。

默认架构包含 `handler`、`service` 和 `repository` 三层，也可以通过 `Config` 定义项目自己的分层和依赖规则。

## 安装

```bash
go install github.com/lwmacct/260622-go-pkg-tddcheck/cmd/tddcheck@latest
```

## 快速开始

在命令行检查默认的 `internal` 目录：

```bash
tddcheck check --root internal
```

也可以把架构检查固定为项目测试：

```go
package tddcheck_test

import (
	"testing"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck"
)

func TestArchitecture(t *testing.T) {
	tddcheck.Project{Root: "internal"}.Assert(t)
}
```

## CLI

```text
tddcheck check    运行架构检查
tddcheck index    输出架构索引
tddcheck doc      生成 Markdown 架构文档
tddcheck version  输出版本
```

常用命令：

```bash
tddcheck index --root internal
tddcheck index --root internal --format json
tddcheck doc --root internal --output docs/tddcheck.index.gen.md
```

CLI 只接受完整长参数，例如 `--root`、`--format` 和 `--output`。直接运行 `tddcheck` 等同于检查默认的 `internal` 目录。

退出码：

```text
0  命令成功，check 未发现违规
1  check 发现架构违规
2  参数、项目解析或输出操作失败
```

在源码仓库中也可以直接运行：

```bash
go run ./cmd/tddcheck check --root internal
```

## 文档

- [Go 包文档](https://pkg.go.dev/github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck)：`Project`、`Analysis`、配置和架构索引 API。
- [默认规则与配置](docs/default-rules.md)：默认分层、文件类型、内容规则、依赖方向和自定义示例。

`Project.Analyze` 适合需要结构化违规和索引数据的集成；测试通常直接使用 `Project.Assert`。生成并提交架构索引文档时，可以使用 CLI 的 `doc` 命令或 `Project.WriteDoc`。

## 开发

```bash
go test ./...
go vet ./...
```
