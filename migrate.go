package migrate

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"math"
)

type Migration struct {
	Id        uint   `json:"id" gorm:"primary_key"`
	Migration string `json:"migration" gorm:"type:varchar(255)"`
	Batch     uint   `json:"batch"`
}

func (Migration) TableName() string {
	return DefaultTableName
}

type File interface {
	Key() string
	Up(tx *gorm.DB) error
	Down(tx *gorm.DB) error
}

// 定义的迁移文件需要在这里注册
var Files []File

// 使用的数据库连接
var DB *gorm.DB
var DefaultTableName = "migrations"

func InitMigrationTable() error {
	return DB.AutoMigrate(&Migration{}).Error
}

// 注册迁移文件
func Register(file File) {
	for _, f := range Files {
		if f.Key() == file.Key() {
			return
		}
	}
	Files = append(Files, file)
}

// 获取需要迁移的 migrateFiles
// files 有, migrations 里没有的数据
func getNeedMigrateFiles(migrateFiles []File, step int) []File {
	var ans []File
	var ms = getAllMigrations()
	diff := map[string]string{}
	for _, migrateFile := range migrateFiles {
		diff[migrateFile.Key()] = ""
	}
	for _, migration := range ms {
		delete(diff, migration.Migration)
	}
	for _, migrateFile := range migrateFiles {
		if step == 0 {
			break
		}
		if _, ok := diff[migrateFile.Key()]; ok {
			ans = append(ans, migrateFile)
			step--
		}
	}
	return ans
}

// 获取需要回滚的 migrateFiles
func getNeedRollbackKeys(step int) []File {
	var ans []File
	var ms = getAllMigrations()
	var keyMigrateFile = map[string]File{}
	if step < 1 {
		return ans
	}
	for _, migrateFile := range Files {
		keyMigrateFile[migrateFile.Key()] = migrateFile
	}
	cur := 0
	for _, migrate := range ms {
		if step < 1 {
			break
		}
		if m, ok := keyMigrateFile[migrate.Migration]; ok {
			ans = append(ans, m)
		}
		if int(migrate.Batch) != cur {
			step--
		}
	}
	return ans
}

// 获取所有迁移记录
func getAllMigrations() []Migration {
	_ = InitMigrationTable()
	var ms []Migration
	DB.Order("id desc").Find(&ms)
	return ms
}

// 获取下一个迁移版本号
func getNextBatchNo() uint {
	_ = InitMigrationTable()
	m := Migration{}
	batch := uint(0)
	DB.Order("batch desc").Select("batch").First(&m)
	batch = m.Batch + 1
	return batch
}

func createMigrate(migration string, batch uint) (err error) {
	m := Migration{
		Migration: migration,
		Batch:     batch,
	}
	err = DB.Create(&m).Error
	return
}

func deleteMigrate(migration string) (err error) {
	m := Migration{
		Migration: migration,
	}
	err = DB.Where(&m).Delete(Migration{}).Error
	return
}

// 执行迁移
func Migrate(step int) error {
	var err error
	err = InitMigrationTable()
	if err != nil {
		return errors.Wrap(err, "create migrate table failed")
	}
	mfs := getNeedMigrateFiles(Files, step)
	nextBatch := getNextBatchNo()
	if len(mfs) == 0 {
		return nil
	}
	for _, mf := range mfs {
		// transaction
		err = DB.Transaction(func(tx *gorm.DB) error {
			err = mf.Up(tx)
			if err != nil {
				return errors.Wrapf(err, "[migrate failed] %s: %s", mf.Key(), err.Error())
			}
			err = createMigrate(mf.Key(), nextBatch)
			if err != nil {
				return errors.Wrapf(err, "[migrate failed] %s: %s", mf.Key(), err.Error())
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// 执行回滚
func Rollback(step int) error {
	var err error
	mfs := getNeedRollbackKeys(step)
	if len(mfs) == 0 {
		return nil
	}
	for _, mf := range mfs {
		err = DB.Transaction(func(tx *gorm.DB) error {
			err = mf.Down(tx)
			if err != nil {
				return errors.Wrapf(err, "[Rollback failed] %s: %s", mf.Key(), err.Error())
			}
			err = deleteMigrate(mf.Key())
			if err != nil {
				return errors.Wrapf(err, "[Rollback failed] %s: %s", mf.Key(), err.Error())
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// 获取连接中的所有表
func Tables() ([]string, error) {
	var tables []string
	err := DB.Raw("show tables").Pluck("Tables_in_mysql", &tables).Error
	if err != nil {
		return tables, err
	}
	return tables, nil
}

// 删除所有表, 并重新执行迁移
func Fresh() error {
	if err := DelAll(); err != nil {
		return err
	}
	if err := Migrate(math.MaxInt64); err != nil {
		return err
	}
	return nil
}

// 删除所有表
func DelAll() error {
	fmt.Printf("Delete all \n")
	tables, err := Tables()
	if err != nil {
		return errors.Wrap(err, "get all table failed")
	}
	for _, table := range tables {
		if err = DB.Exec("drop table " + table).Error; err != nil {
			return errors.Wrapf(err, "drop table %s failed", table)
		}
	}
	return nil
}

// 删除指定表
func Del(tableName string) error {
	if err := DB.Exec("drop table " + tableName).Error; err != nil {
		return errors.Wrapf(err, "drop table %s failed", tableName)
	}
	return nil
}

// 清空表
func Truncate(tableName string) error {
	return DB.Exec(fmt.Sprintf("truncate %s", tableName)).Error
}

// 表格是否存在
func TableExist(tableName string) (bool, error) {
	tables, err := Tables()
	if err != nil {
		return false, err
	}
	for _, table := range tables {
		if table == tableName {
			return true, nil
		}
	}
	return false, nil
}
