package testdata

import "gorm.io/gorm"

type TestFile struct {
}

func (TestFile) Key() string {
	return "TestFile"
}
func (TestFile) Up(tx *gorm.DB) error {
	tx.Exec("create table test (id int)")
	return nil
}
func (TestFile) Down(tx *gorm.DB) error {
	tx.Exec("drop table test")
	return nil
}
