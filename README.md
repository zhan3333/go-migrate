# go-migrate

go gorm 数据库版本迁移, 支持 MySQL 数据库

## 使用

定义迁移文件
```go
package testdata

import "github.com/jinzhu/gorm"

type FooMigrateFile struct {
}

// 迁移文件标识, 唯一性
func (FooMigrateFile) Key() string {
	return "FooMigrateFile"
}

// 迁移时执行
func (FooMigrateFile) Up(tx *gorm.DB) error {
	tx.Exec("create table test (id int)")
	return nil
}

// 回滚时执行
func (FooMigrateFile) Down(tx *gorm.DB) error {
	tx.Exec("drop table test")
	return nil
}

```

```go
package main

import "github.com/zhan3333/go-migrate"
import "github.com/jinzhu/gorm"

func main() {
    var err error
    // 初始化数据库连接
    migrate.DB, err = gorm.Open("mysql", "")
    // 初始化 Migrations table
    _ = migrate.InitMigrationTable()
    // 注册迁移文件
    migrate.Register(&FooMigrateFile{})
    // 迁移
    migrate.Migrate(1)
    // 回滚
    migrate.Rollback(1)
}
```