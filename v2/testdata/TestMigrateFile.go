package testdata

import (
	"github.com/zhan3333/gdb/v2"
)

type TestFile struct {
}

func (TestFile) Key() string {
	return "TestFile"
}
func (TestFile) Up(db *gdb.Entry) error {
	db.Exec("create table test (id int)")
	return nil
}
func (TestFile) Down(db *gdb.Entry) error {
	db.Exec("drop table test")
	return nil
}
