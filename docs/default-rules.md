# tddcheck 默认规则

本文档描述 `tddcheck.DefaultConfig()` 启用的架构规则、扫描边界和可配置项。

## 执行模型

检查流程通过 `go/packages` 加载目标 module subtree，共享同一个 `token.FileSet`，并缓存 package graph、AST、`types.Info`、imports 和所属层。默认执行两组规则：

```text
filelayout  文件命名、文件 kind、声明内容和部分跨文件约束
layerdeps   分层 import 依赖约束
```

以下文件和目录会被特殊处理：

```text
*_test.go                                      默认不加载；IncludeTests=true 时可供依赖和自定义规则检查
{subject}.free.go / x.{namespace}.free.go      不限制声明内容；仍参与依赖检查，并进入 free 文件审计清单
隐藏目录                                       不参与扫描
vendor/node_modules/dist/build  默认不参与扫描
```

Go build constraints、`Config.BuildFlags` 和当前 GOOS/GOARCH 会决定实际文件集合。默认允许 package/type error 并尽力分析；`Config.StrictPackages=true` 会将其视为操作错误。

CLI 可通过 `--config tddcheck.json` 读取配置。JSON 使用 lowerCamel 字段名并拒绝重复或未知字段。

## 默认分层

默认被 `filelayout` 检查的层目录：

```text
handler
service
repository
```

默认也使用这些层做依赖检查。可以用 `DependencyLayerDirs` 添加只参与依赖检查、不参与文件布局检查的层，例如 `runtime` 或 `appcmd`。

## 文件命名

默认命名模式是 `qualified_kind`：

```text
{subject}.{kind}.go
x.{namespace}.{kind}.go
```

`subject` 表示业务主题，不限定为 HTTP/REST resource。`x` 是架构文件标识，`namespace` 必须属于所在层的架构 namespace 白名单，`kind` 仍按所在层允许的文件 kind 检查。

`subject` 和 `namespace` 都是独立的 lowercase snake_case 组件，`kind` 是单个 lowercase 字母数字原子。多段 kind（例如 `service.handler.go`）、大写、连字符、连续下划线和 kind 中的下划线均不合法。业务 subject 可以与某个 namespace 同名；是否为架构文件只由 `x.` 标识决定。

示例：

```text
internal/handler/device.handler.go
internal/handler/device.dto.go
internal/handler/device.free.go
internal/handler/x.shared.dto.go
internal/handler/x.http.endpoint.go
internal/handler/x.http.context.go
internal/handler/x.http.free.go

internal/service/device.service.go
internal/service/device.commands.go
internal/service/device.provider.go
internal/service/x.shared.support.go
internal/service/x.shared.free.go

internal/repository/device.support.go
internal/repository/device.store.go
internal/repository/x.shared.support.go
internal/repository/x.store.repository.go
internal/repository/x.store.free.go
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
x.unknown.free.go
```

也可以把某一层改成 `package_kind` 命名模式，此时文件名使用：

```text
{kind}.go
```

这适合 `internal/adapter/httpauth/service.go` 这类目录即业务 subject 的布局。

## 默认文件 kind

各层允许的文件 kind：

```text
handler:    free, support, mapper, context, dto, endpoint, handler, middleware, utils
service:    free, support, mapper, commands, provider, service
repository: free, support, repository, schema, store
```

每个允许的 kind 都显式绑定一个声明策略。默认配置中策略 ID 通常与 kind 同名；`free` 策略不限制声明。自定义 kind 如果只需要命名与依赖约束，必须显式映射到 `free`，不存在隐式的未知 kind 放行。

各层允许的架构 namespace：

```text
handler:    shared, http
service:    shared
repository: shared, store
```

常见架构文件：

```text
handler:
  x.shared.support.go, x.shared.mapper.go, x.shared.dto.go, x.shared.handler.go, x.shared.utils.go
  x.http.context.go, x.http.endpoint.go, x.http.middleware.go, x.http.support.go

service:
  x.shared.support.go, x.shared.mapper.go

repository:
  x.shared.support.go
  x.store.repository.go
```

## 内容规则

```text
*.support.go      声明类型、const、Err* var、util*/validate*/normalize*/Wrap*/Is*/As* 函数
*.mapper.go       只能声明包级 To* 函数；禁止 context/database/http/huma/ORM 相关 import
*.service.go      service 层声明一个 {Subject}Service、New{Subject}Service 和 service receiver 方法
*.repository.go   repository 层只能用于 x.store.repository.go；必须声明 Store struct 和 NewStore
*.store.go        repository 层声明 Store receiver 方法；方法需接受 context.Context 且最后返回 error
*.handler.go      handler 层声明 {Subject}Handler、Register* 函数和 handler receiver 方法
*.dto.go          只能声明 DTO/DTOs 类型；不能声明函数
*.context.go      仅 handler/x.http 使用；声明私有 *Key 类型和 Context* / *FromContext helper
*.endpoint.go     仅 handler/x.http 使用；必须声明 Endpoint struct 和 NewEndpoint
*.middleware.go   仅 handler/x.http 使用；声明 Middleware、Endpoint/private receiver 方法和 util* helper
*.utils.go        只能声明包级 util* 函数
*.commands.go     只能声明类型；类型名必须以 Request、Response、Result 或 Item 结尾
*.provider.go     service 层声明 {Subject}Provider、New* 构造和 provider receiver 方法
*.schema.go       repository 层声明 {Subject}*Model struct、schema 生命周期函数和 *Model receiver hook
*.free.go         可以声明任意内容；可用于任意合法 subject 或当前层允许的 namespace；会进入分析审计清单
```

额外约束：

```text
service 文件不得直接依赖 database/sql、gorm、bun、pgx、mongo、firestore、dynamodb 等持久化 API
service 文件不得引用 repository.*Model
service/provider/support 类型不得使用 DTO、Request、Response、Result、Item 等传输或命令后缀
service/provider/support 类型不得声明 json/query/path/bun 等传输或持久化 tag
repository support 不得声明 *Model 或 ORM tag；schema model 必须放在 .schema.go
appcmd 作为依赖层启用时，不得 import huma、注册 huma route 或声明 DTO/TDO 类型
```

## 命名规则

```text
业务 subject 使用 snake_case
架构文件使用 x.{namespace}.{kind}.go；x 是架构标识
禁止使用 common、default、helper、helpers、misc、util、utils 等弱 subject
业务 subject 不能把文件 kind 编码进名称，例如 device_update、device_mapper
mapper 函数必须以 To 开头
utils 函数必须以 util 开头
support 函数必须以 util、validate、normalize、Wrap、Is 或 As 开头
```

业务 subject 会从声明名中推断 snake_case。例如 `DeviceGroupService` 对应 `device_group.service.go`。

默认将 service 层的 `service` kind 配置为 subject 锚点：同一业务 subject 如果声明了 `commands`、`provider` 或 `support` 等文件，也必须声明对应的 `{subject}.service.go`。该文件的声明策略会继续要求 `New{Subject}Service`。架构 namespace 和 `free` kind 不受锚点约束。自定义 qualified-kind 层可配置自己的锚点；package-kind 层没有文件级 subject，不能配置锚点。

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

所有 `free` 文件只免除 filelayout 的声明内容约束，仍参与分层依赖检查，并通过 `Analysis.FreeFiles`、JSON 和生成的 Markdown 文档公开审计。

## 架构索引

架构索引是只读分析结果，不参与架构规则是否通过的判定。它复用同一次静态扫描，不连接数据库，也不执行业务代码。

当前索引按已解析的 `FileIdentity.Kind` 识别，因此 qualified-kind 和 package-kind 两种布局使用同一条提取路径：

```text
handler      *.handler.go 中的 *Handler、Register* 和 receiver 方法
service      *.service.go 中的 *Service、New*Service 和 receiver 方法
store        *.store.go 中的 Store receiver 方法
repository   *.schema.go 中的 *Model、bun table tag、字段 tag 和 ForeignKey 字符串
```

架构索引不识别或表达 API endpoint。API 契约应由项目使用的 API 框架或 OpenAPI 工具生成。

JSON 分析 schema 当前为 v2。handler、service、store、table 和 projection 不再输出含义混杂的 `scope` 字符串，而是统一输出：

```json
{
  "identity": {
    "layer": "service",
    "subject": "device",
    "kind": "service"
  }
}
```

架构文件使用独立的 `namespace` 字段，例如 `x.shared.handler.go` 输出 `"namespace": "shared"`。free 文件不进入架构实体索引，但会进入独立审计清单。

## 诊断契约

每条结构化诊断包含稳定的 `ruleId` 和细分 `code`，例如 `filelayout/invalid-filename`、`filelayout/missing-subject-anchor`、`filelayout/namespace-not-allowed` 和 `layerdeps/forbidden-import`。可机械迁移的问题可包含 `suggestedFix`；旧式 `x_http.*.go` 命名会提供结构化 rename fix。

推荐通过单元测试同时检查规则和文档漂移；显式传入测试定义的 `-update` flag 时才更新文档：

```bash
go test -count=1 ./internal/testutil/tddcheck
go test -count=1 ./internal/testutil/tddcheck -update
```

## 自定义配置

配置字段：

```text
LayerDirs               参与 filelayout 检查的层目录名
DependencyLayerDirs     参与 layerdeps 检查的层目录名；nil 时等于 LayerDirs
SkipDirs                扫描时跳过的目录名
LayerRules              禁止的 import 依赖规则
LayerFileNameModes      每层文件命名模式：qualified_kind 或 package_kind
LayerKindPolicies       每层允许的文件 kind 及其显式声明策略 ID
LayerSubjectAnchorKinds qualified-kind 层要求每个业务 subject 必须存在的锚点 kind
ArchitectureNamespaces  每层允许的架构 namespace；配置值不包含 x. 标识
EscapedSubjectSuffixes  禁止编码进业务 subject 的文件 kind 或动作词
ForbiddenWeakSubjects   禁止使用的弱业务 subject
```

配置的 slice 或 map 字段为 `nil` 时继承默认值；显式设置为非 `nil` 空集合时关闭对应默认项。

自定义分层示例：

```go
config := tddcheck.Config{
	LayerDirs:           []string{"adapter"},
	DependencyLayerDirs: []string{"adapter", "runtime", "service"},
	LayerFileNameModes: map[string]string{
		"adapter": tddcheck.FileNameModePackageKind,
	},
	LayerKindPolicies: map[string]map[string]string{
		"adapter": {
			"doc":     "free",
			"service": "free",
		},
	},
	LayerSubjectAnchorKinds: map[string]string{},
	ArchitectureNamespaces: map[string][]string{},
	LayerRules: []tddcheck.LayerDependencyRule{
		{
			SourceLayer: "runtime",
			TargetLayer: "adapter",
			Message:     "runtime must not import adapter",
		},
	},
}
```

`LayerDependencyRule` 支持用相对路径前缀收窄 source 和 target，并为两者配置例外。source 前缀相对分析根目录，target 前缀相对 module 的 `internal/` 目录。例如只允许部分 adapter import `adapter/sshauth`：

```go
LayerRules: []tddcheck.LayerDependencyRule{
	{
		SourceLayer:     "adapter",
		TargetLayer:     "adapter",
		TargetRelPrefix: "adapter/sshauth",
		ExceptSourceRelPrefixes: []string{
			"adapter/sshcmd",
			"adapter/sshproxyjump",
		},
		Message: "only ssh command adapters may import sshauth",
	},
}
```
