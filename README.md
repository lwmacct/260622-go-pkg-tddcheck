# go-pkg-tddcheck

`tddcheck` 是一个面向 Go 项目的架构规则检查器，用来约束分层目录、文件命名、声明内容和跨层 import。

默认规则针对 `handler` / `service` / `repository` 三层，也可以通过 `Config` 改成项目自己的层名和依赖约束。

## 使用方式

命令行运行：

```bash
go run ./cmd/tddcheck -root internal
```

打印版本：

```bash
go run ./cmd/tddcheck -version
```

在项目测试中引用：

```go
package tddcheck_test

import (
	"testing"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck"
)

func TestRules(t *testing.T) {
	tddcheck.ProjectRules{Root: "internal"}.Assert(t)
}
```

运行本地架构检查时建议使用：

```bash
go test -count=1 ./internal/testutil/tddcheck
```

`ProjectRules.Check()` 会返回 `Result`，其中包含 `Passed`、`Err`、`Violations` 和 `Duration`。`Result.Text()` 会输出和 CLI 类似的文本。

## 执行模型

检查流程会先解析项目根目录、读取 `go.mod` module path，然后扫描一次目标目录下的 Go 文件，缓存文件路径、所属层、AST 和 imports。默认执行两组规则：

```text
filelayout  文件命名、文件类型、声明内容、部分跨文件约束
layerdeps   分层 import 依赖约束
```

以下文件和目录会被特殊处理：

```text
*_test.go   不参与规则检查
x_free.go   跳过 filelayout 和 layerdeps 检查
.git/.hg/.svn/vendor/node_modules/dist/build 默认跳过扫描
```

## 默认分层

默认被 filelayout 检查的层目录：

```text
handler
service
repository
```

默认也使用这些层做依赖检查。可以用 `DependencyLayerDirs` 添加只参与依赖检查、不参与文件布局检查的层，例如 `runtime` 或 `appcmd`。

## 文件命名

默认命名模式是 `scope_kind`：

```text
{subject}.{type}.go
x_{scope}.{type}.go
```

`subject` 表示业务主题，不限定为 HTTP/REST resource。`x_` 文件表示架构/共享 scope；`scope` 必须属于所在层的架构 scope 白名单，`type` 仍按所在层允许的文件类型检查。

示例：

```text
internal/handler/device.handler.go
internal/handler/device.dto.go
internal/handler/x_shared.dto.go
internal/handler/x_http.endpoint.go
internal/handler/x_http.context.go

internal/service/device.service.go
internal/service/device.commands.go
internal/service/device.provider.go
internal/service/x_shared.support.go

internal/repository/device.support.go
internal/repository/device.store.go
internal/repository/x_shared.support.go
internal/repository/x_store.repository.go
```

拒绝示例：

```text
device_handler.go
shared.model.go
device_update.utils.go
device.models.go
device.writes.go
device.database.go
helper.utils.go
```

也可以把某一层改成 `package_kind` 命名模式，此时文件名使用：

```text
{type}.go
```

这适合 `internal/adapter/httpauth/service.go` 这类目录即业务 scope 的布局。

## 默认文件类型

各层允许的文件类型：

```text
handler:    support, mapper, context, dto, endpoint, handler, middleware, utils
service:    support, mapper, commands, provider, service
repository: support, repository, schema, store
```

列表顺序约定为：跨层共有项放前面，层专属项放后面。下面的架构 scope 和示例也遵循这个顺序。

各层允许的架构 scope：

```text
handler:    x_shared, x_http
service:    x_shared
repository: x_shared, x_store
```

常见架构文件示例：

```text
handler:
  x_shared.support.go, x_shared.mapper.go, x_shared.dto.go, x_shared.handler.go, x_shared.utils.go
  x_http.context.go, x_http.endpoint.go, x_http.middleware.go, x_http.support.go

service:
  x_shared.support.go, x_shared.mapper.go

repository:
  x_shared.support.go
  x_store.repository.go
```

## 内容规则

```text
*.support.go      声明类型、const、Err* var、util*/validate*/normalize*/Wrap*/Is*/As* 函数
*.mapper.go       只能声明包级 To* 函数；禁止 context/database/http/huma/ORM 相关 import
*.service.go      service 层声明一个 {Subject}Service、New{Subject}Service 和 service receiver 方法
*.repository.go   repository 层只能用于 x_store.repository.go；必须声明 Store struct 和 NewStore
*.store.go        repository 层声明 Store receiver 方法；方法需接受 context.Context 且最后返回 error
*.handler.go      handler 层声明 {subject}Handler、Register* 函数和 handler receiver 方法
*.dto.go          只能声明 DTO/DTOs 类型；不能声明函数
*.context.go      仅 handler/x_http 使用；声明私有 *Key 类型和 Context* / *FromContext helper
*.endpoint.go     仅 handler/x_http 使用；必须声明 Endpoint struct 和 NewEndpoint
*.middleware.go   仅 handler/x_http 使用；声明 Middleware、Endpoint/private receiver 方法和 util* helper
*.utils.go        只能声明包级 util* 函数
*.commands.go     只能声明类型；类型名必须以 Request、Response、Result 或 Item 结尾
*.provider.go     service 层声明 {Subject}Provider、New* 构造和 provider receiver 方法
*.schema.go       repository 层声明 {Subject}*Model struct、schema 生命周期函数和 *Model receiver hook
```

额外约束：

```text
service 文件不得直接依赖 database/sql、gorm、bun、pgx、mongo、firestore、dynamodb 等持久化 API
service 文件不得引用 repository.*Model
service/provider/support 类型不得使用 DTO、Request、Response、Result、Item 等传输/命令后缀
service/provider/support 类型不得声明 json/query/path/bun 等传输或持久化 tag
repository support 不得声明 *Model 或 ORM tag；schema model 必须放在 .schema.go
appcmd 作为依赖层启用时，不得 import huma、注册 huma route 或声明 DTO/TDO 类型
```

## 命名规则

```text
资源 scope 使用 snake_case
架构 scope 使用 x_ 前缀
禁止使用 common、default、helper、helpers、misc、util、utils 等弱 scope
资源 scope 不能把文件类型词编码进 scope，例如 device_update、device_mapper
mapper 函数必须以 To 开头
utils 函数必须以 util 开头
support 函数必须以 util、validate、normalize、Wrap、Is 或 As 开头
```

业务 scope 会从声明名中推断 snake_case。例如 `DeviceGroupService` 对应 `device_group.service.go`。

service 层同一业务 subject 如果声明了 `commands`、`provider` 或 `support` 等文件，默认也必须声明对应的 `{subject}.service.go` 和 `New{Subject}Service`。`x_` 架构 scope 不受这个要求影响。

## 分层依赖

默认禁止的 import：

```text
handler    -> repository
service    -> handler
repository -> handler
repository -> service
```

`layerdeps` 只检查当前 module 的 `internal/` 下 import。例如 module 为 `example.com/app` 时，会识别：

```text
example.com/app/internal/service
example.com/app/internal/repository/device
```

`x_free.go` 不参与分层依赖检查。

## 配置

默认配置：

```go
config := tddcheck.DefaultConfig()
```

自定义配置示例：

```go
func TestRules(t *testing.T) {
	tddcheck.ProjectRules{
		Root: "internal",
		Config: tddcheck.Config{
			LayerDirs: []string{"adapter"},
			DependencyLayerDirs: []string{"adapter", "runtime", "service"},
			LayerFileNameModes: map[string]string{
				"adapter": tddcheck.FileNameModePackageKind,
			},
			LayerFileKinds: map[string][]string{
				"adapter": {"endpoint", "service"},
			},
			ArchitectureScopes: map[string][]string{},
			LayerRules: []tddcheck.LayerDependencyRule{
				{
					SourceLayer: "runtime",
					TargetLayer: "adapter",
					Message:     "runtime must not import adapter",
				},
			},
		},
	}.Assert(t)
}
```

配置字段：

```text
LayerDirs             参与 filelayout 检查的层目录名
DependencyLayerDirs   参与 layerdeps 检查的层目录名；为空时等于 LayerDirs
SkipDirs              扫描时跳过的目录名
LayerRules            禁止的 import 依赖规则
LayerFileNameModes    每层文件命名模式：scope_kind 或 package_kind
LayerFileKinds        每层允许的文件类型
ArchitectureScopes    每层允许的 x_ 架构 scope
EscapedScopeSuffixes  禁止编码进业务 scope 的文件类型/动作词
ForbiddenWeakScopes   禁止使用的弱业务 scope
```

`LayerDependencyRule` 支持按相对路径前缀做约束和例外：

```go
type LayerDependencyRule struct {
	SourceLayer             string
	SourceRelPrefix         string
	ExceptSourceRelPrefixes []string
	TargetLayer             string
	TargetRelPrefix         string
	ExceptTargetRelPrefixes []string
	Message                 string
}
```

示例：只允许部分 adapter import `adapter/sshauth`：

```go
LayerRules: []tddcheck.LayerDependencyRule{
	{
		SourceLayer: "adapter",
		TargetLayer: "adapter",
		TargetRelPrefix: "adapter/sshauth",
		ExceptSourceRelPrefixes: []string{
			"adapter/sshcmd",
			"adapter/sshproxyjump",
		},
		Message: "only ssh command adapters may import sshauth",
	},
}
```

## 输出

通过时：

```text
tddcheck: passed
```

失败时：

```text
tddcheck: failed
internal/handler/device.go:1 [filelayout] handler file must use {scope}.{type}.go naming
internal/handler/device.handler.go:3 [layerdeps] handler must not import repository: example.com/app/internal/repository
```
