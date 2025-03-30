package repository_test

import (
	"testing"

	"my-project/infrastructure/persistence"
)

func Test_GetById(t *testing.T) {
	db, _ := persistence.NewNativeDb()
	err := db.Close()
	if err != nil {
		return
	}
}
