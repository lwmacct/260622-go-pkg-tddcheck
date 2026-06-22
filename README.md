# go-pkg-tddcheck

`tddcheck` provides mechanical architecture checks for Go projects using a compact handler/service/repository layout.

## Usage

Run as a CLI:

```bash
go run ./cmd/tddcheck -root internal
```

Run from a project test:

```go
package tddcheck_test

import (
	"testing"

	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rules/filelayout"
	"github.com/lwmacct/260622-go-pkg-tddcheck/pkg/tddcheck/rules/layerdeps"
)

func TestRules(t *testing.T) {
	t.Run("file-layout", func(t *testing.T) {
		filelayout.New("internal").Assert(t)
	})
	t.Run("dependency-layerdeps", func(t *testing.T) {
		layerdeps.New("internal").Assert(t)
	})
}
```

Use `-count=1` when running architecture checks from Go tests:

```bash
go test -count=1 ./internal/testutil/tddcheck
```

## File Layout

Business resource files use:

```text
{resource}.{type}.go
```

Architecture and shared files use:

```text
x_{scope}.{type}.go
```

The `x_` prefix makes non-resource files visible and keeps generic scopes from colliding with business resource names.

Allowed file types:

```text
handler:    dto, handler, mapper, utils
service:    commands, constants, errors, mapper, models, service, utils, validation, writes
repository: constants, database, errors, model, models, repository, schema, store, utils
```

Allowed architecture scopes:

```text
handler:    x_api, x_frontend, x_router, x_shared
service:    x_batch, x_id, x_shared
repository: x_database, x_schema, x_store, x_shared
```

Examples:

```text
internal/handler/device.dto.go
internal/handler/device.handler.go
internal/handler/x_router.handler.go

internal/service/device.commands.go
internal/service/device.service.go
internal/service/x_batch.service.go
internal/service/x_id.validation.go
internal/service/x_shared.models.go

internal/repository/device.model.go
internal/repository/device.store.go
internal/repository/x_database.repository.go
internal/repository/x_schema.utils.go
internal/repository/x_store.repository.go
```

Rejected examples:

```text
shared.models.go
batch.service.go
device_update.utils.go
x_database.service.go
helper.utils.go
```

## Mapper Functions

Mapper files are for pure package-level conversion functions.

Rules:

```text
*.mapper.go functions must start with To
*.mapper.go functions must not use receivers
*.mapper.go files must not declare types, vars, or consts
*.mapper.go files must not import context, HTTP, database, or ORM packages
```

Recommended names:

```text
ToDeviceDTO             service model -> handler DTO
ToServiceCreateDevice   handler DTO -> service command
ToRepositoryDevicePatch service command -> repository command
```

Avoid `Map*`, `From*`, `Build*`, and `Convert*`; they hide the target type and make mapper direction harder to read at call sites.

## Layer Dependencies

Default forbidden imports:

```text
handler    -> repository
service    -> handler
repository -> handler
repository -> service
```
