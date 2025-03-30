package persistence

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestNewRepositories(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error %v", err)
	}
	gormDb, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("an error %v", err)
	}
	tests := []struct {
		name    string
		want    *gorm.DB
		wantErr bool
	}{
		{
			name:    "Test #1",
			want:    gormDb,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRepositories()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRepositories() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// if !reflect.DeepEqual(got, tt.want) {
			// 	t.Errorf("NewRepositories() = %v, want %v", got, tt.want)
			// }
		})
	}
}
