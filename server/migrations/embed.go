package migrations

import "embed"

// FS 把 migrations/*.sql 编进二进制，部署时无需额外拷贝 SQL 文件。
//
//go:embed *.sql
var FS embed.FS
