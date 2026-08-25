# go-pkg-tddcheck

`tddcheck` 是一个面向 Go 项目的架构规则检查器。它通过静态扫描约束分层目录、文件命名、声明内容和跨层 import，并从同一次扫描生成架构索引。

默认架构包含 `handler`、`service` 和 `repository` 三层，也可以通过 `Config` 定义项目自己的分层和依赖规则。

## 测试依赖

```bash
go get github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck@latest
```

## 快速开始

以单元测试作为架构检查和文档维护的主入口：

```go
package tddcheck_test

import (
	"flag"
	"testing"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck"
)

var update = flag.Bool("update", false, "update tddcheck documentation")

func TestArchitecture(t *testing.T) {
	analyzer, err := tddcheck.New(tddcheck.Options{Root: "internal"})
	if err != nil {
		t.Fatal(err)
	}
	tddcheck.Assert(t, analyzer, tddcheck.TestOptions{
		Markdown: &tddcheck.MarkdownTestOptions{Update: *update},
	})
}
```

日常检查和显式更新文档分别使用：

```bash
go test -count=1 ./internal/testutil/tddcheck
go test -count=1 ./internal/testutil/tddcheck -update
```

普通测试只检查 `docs/tddcheck.index.gen.md` 是否漂移，不修改工作区；`-update` 才会在规则全部通过后原子更新文档。

## Go Tool

CLI 是辅助入口，应通过 Go 1.27 tool directive 固定到项目版本，无需全局安装：

```bash
go get -tool github.com/lwmacct/260622-go-pkg-tddcheck/cmd/tddcheck@latest
```

```text
go tool tddcheck check    运行架构检查
go tool tddcheck index    输出架构索引
go tool tddcheck doc      生成 Markdown 架构文档
go tool tddcheck version  输出固定的模块版本
```

辅助命令：

```bash
go tool tddcheck index --root internal
go tool tddcheck index --root internal --format json
go tool tddcheck check --root internal --config tddcheck.json
```

配置文件使用 `Config` 的 lowerCamel JSON 字段名。解析采用 `encoding/json/v2` 严格语义：重复字段、未知字段和无效 UTF-8 会直接失败。

warning 默认不会使分析失败；需要把架构债务纳入 CI 失败条件时，在配置中设置 `"failOnWarnings": true`。
subject ownership 可按 layer/kind 设置为 `error`、`warning` 或 `off`；常规全大写缩写放入 `initialisms` 列表，特殊大小写缩写放入 `initialismOverrides`。

工具只接受完整长参数，例如 `--root`、`--format` 和 `--output`。直接运行 `go tool tddcheck` 等同于检查默认的 `internal` 目录。

退出码：

```text
0  命令成功，check 未发现违规
1  check 发现架构违规，或 doc --check 发现文档缺失/漂移
2  参数、项目解析或输出操作失败
```

## 文档

- [Go 包文档](https://pkg.go.dev/github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck)：`Analyzer`、`Analysis`、配置和架构索引 API。
- [默认规则与配置](docs/default-rules.md)：默认分层、文件类型、内容规则、依赖方向和自定义示例。

`Analyzer.Analyze(ctx)` 返回带稳定 code、源码范围和可选修复建议的结构化诊断，以及使用 `FileIdentity` 的 schema v2 架构索引和 free 文件审计清单。`tddcheck.Assert` 会在测试日志中保留 warning 诊断，并在 `FailOnWarnings` 开启时使测试失败；它同时完成规则断言与文档检查/更新。

源码通过 Go package loader 加载，因此 build tags、平台文件、import alias 和 package graph 都按真实构建视图处理。`Config.IncludeTests` 可包含测试变体，`Config.StrictPackages` 可要求所有 package 完整通过类型检查。

## 开发

```bash
go test ./...
go vet ./...
```
