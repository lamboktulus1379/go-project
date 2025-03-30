package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"my-project/domain/model"
	"my-project/domain/repository"
)

type Suite struct {
	suite.Suite
	mock       sqlmock.Sqlmock
	repository repository.IUser
}

func (s *Suite) SetupSuite() {
	var (
		db  *sql.DB
		err error
	)
	db, s.mock, err = sqlmock.New()
	require.NoError(s.T(), err)

	s.repository = NewUserRepository(db)
}

func TestUserRepository_GetByIdSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}

func (s *Suite) TestUserRepository_GetById() {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	createdAtTime, _ := time.Parse(
		"2006-01-02 15:04:05.999999999+07 MST",
		"2023-09-04 01:02:10.911651+07 WIB",
	)
	updatedAtTime, _ := time.Parse(
		"2006-01-02 15:04:05.999999999+07 MST",
		"2023-09-04 01:02:10.911651+07 WIB",
	)
	var (
		ID        = 1
		Name      = "Lambok Tulus Simamora"
		UserName  = "lamboktulus1379"
		Password  = "a252f77af72638ea5a0f9e5fbe5f2b2e"
		CreatedAt = createdAtTime.In(loc)
		UpdatedAt = updatedAtTime.In(loc)
	)

	prep := s.mock.ExpectPrepare(
		regexp.QuoteMeta(`SELECT u.id, u.name, u.user_name, u.password, u.created_at, u.updated_at 
	FROM public.user AS u 
	WHERE u.id = $1`),
	)
	prep.ExpectQuery().WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "user_name", "password", "created_at", "updated_at"}).
			AddRow(ID, Name, UserName, Password, CreatedAt, UpdatedAt))

	res, err := s.repository.GetById(context.Background(), 1)
	exp := model.User{
		ID:        1,
		Name:      "Lambok Tulus Simamora",
		UserName:  "lamboktulus1379",
		Password:  "a252f77af72638ea5a0f9e5fbe5f2b2e",
		CreatedAt: CreatedAt,
		UpdatedAt: UpdatedAt,
	}

	require.Nil(s.T(), err)
	require.Equal(s.T(), exp, res)
}

func (s *Suite) TestUserRepository_GetByIdErrPrepare() {
	s.mock.ExpectPrepare(`SELECT u.id, u.name, u.user_name, u.password, u.created_at, u.updated_at 
	FROM public.user AS u 
	WHERE u.id = $1`).
		WillReturnError(fmt.Errorf("error statement"))

	res, err := s.repository.GetById(context.Background(), 1)
	exp := model.User{}

	require.Error(s.T(), err)
	require.Equal(s.T(), exp, res)
}

func (s *Suite) TestUserRepository_GetByIdErrScan() {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	createdAtTime, _ := time.Parse(
		"2006-01-02 15:04:05.999999999+07 MST",
		"2023-09-04 01:02:10.911651+07 WIB",
	)
	updatedAtTime, _ := time.Parse(
		"2006-01-02 15:04:05.999999999+07 MST",
		"2023-09-04 01:02:10.911651+07 WIB",
	)
	var (
		ID        = 1
		Name      = "Lambok Tulus Simamora"
		UserName  = "lamboktulus1379"
		Password  = "a252f77af72638ea5a0f9e5fbe5f2b2e"
		CreatedAt = createdAtTime.In(loc)
		UpdatedAt = updatedAtTime.In(loc)
	)

	prep := s.mock.ExpectPrepare(
		regexp.QuoteMeta(`SELECT u.id, u.name, u.user_name, u.password, u.created_at, u.updated_at 
	FROM public.user AS u 
	WHERE u.id = $1`),
	)
	prep.ExpectQuery().WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "user_name", "password", "created_at", "updated_at"}).
			AddRow(ID, Name, UserName, Password, CreatedAt, UpdatedAt),
		).
		WillReturnError(fmt.Errorf("error scan"))

	_, err := s.repository.GetById(context.Background(), 1)
	// exp := errors.New("sql: expected 5 destination arguments in Scan, not 6")

	require.NotNil(s.T(), err)
	// require.Equal(s.T(), exp, err)
}

func (s *Suite) TestUserRepository_GetByUserName() {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	createdAtTime, _ := time.Parse(
		"2006-01-02 15:04:05.999999999+07 MST",
		"2023-09-04 01:02:10.911651+07 WIB",
	)
	updatedAtTime, _ := time.Parse(
		"2006-01-02 15:04:05.999999999+07 MST",
		"2023-09-04 01:02:10.911651+07 WIB",
	)
	var (
		ID        = 1
		Name      = "Lambok Tulus Simamora"
		UserName  = "lamboktulus1379"
		Password  = "a252f77af72638ea5a0f9e5fbe5f2b2e"
		CreatedAt = createdAtTime.In(loc)
		UpdatedAt = updatedAtTime.In(loc)
	)

	prep := s.mock.ExpectPrepare(
		regexp.QuoteMeta(`SELECT u.id, u.name, u.user_name, u.password, u.created_at, u.updated_at 
	FROM public.user AS u 
	WHERE u.user_name = $1`),
	)
	prep.ExpectQuery().WithArgs("lamboktulus1379").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "user_name", "password", "created_at", "updated_at"}).
			AddRow(ID, Name, UserName, Password, CreatedAt, UpdatedAt))

	res, err := s.repository.GetByUserName(context.Background(), "lamboktulus1379")
	exp := model.User{
		ID:        1,
		Name:      "Lambok Tulus Simamora",
		UserName:  "lamboktulus1379",
		Password:  "a252f77af72638ea5a0f9e5fbe5f2b2e",
		CreatedAt: CreatedAt,
		UpdatedAt: UpdatedAt,
	}

	require.Nil(s.T(), err)
	require.Equal(s.T(), exp, res)
}

func (s *Suite) TestUserRepository_GetByUserNameErrPrepare() {
	prep := s.mock.ExpectPrepare(regexp.QuoteMeta(`SELECT u.id, u.name, u.user_name, u.password, u.created_at, u.updated_at 
	FROM public.user AS u 
	WHERE u.user_name = $1`)).
		WillReturnError(fmt.Errorf("error statement"))
	prep.ExpectQuery().WithArgs("lamboktulus1379").WillReturnError(errors.New("error expect query"))

	res, err := s.repository.GetByUserName(context.Background(), "lamboktulus1379")
	exp := model.User{}

	require.Error(s.T(), err)
	require.Equal(s.T(), exp, res)
}

func (s *Suite) TestUserRepository_CreateUser() {
	var (
		Name     = "Lambok Tulus Simamora"
		UserName = "lamboktulus1379"
		Password = "a252f77af72638ea5a0f9e5fbe5f2b2e"
	)

	prep := s.mock.ExpectPrepare(
		regexp.QuoteMeta(`INSERT INTO public.user (name, user_name, password) VALUES ($1, $2, $3)`),
	)
	prep.ExpectExec().WithArgs(Name, UserName, Password).
		WillReturnResult(sqlmock.NewResult(1, 1)).WillReturnError(nil)

	user := model.User{
		Name:     "Lambok Tulus Simamora",
		UserName: "lamboktulus1379",
		Password: "a252f77af72638ea5a0f9e5fbe5f2b2e",
	}

	err := s.repository.CreateUser(context.Background(), user)
	require.Nil(s.T(), err)
}

func (s *Suite) TestUserRepository_CreateUserErrPrepare() {
	var (
		Name     = "Lambok Tulus Simamora"
		UserName = "lamboktulus1379"
		Password = "a252f77af72638ea5a0f9e5fbe5f2b2e"
	)

	prep := s.mock.ExpectPrepare(regexp.QuoteMeta(`INSERT INTO public.user (name, user_name, password) VALUES ($1, $2, $3)`)).
		WillReturnError(fmt.Errorf("error statement"))
	prep.ExpectExec().WithArgs(Name, UserName, Password).
		WillReturnResult(sqlmock.NewResult(1, 1)).WillReturnError(nil)

	user := model.User{
		Name:     "Lambok Tulus Simamora",
		UserName: "lamboktulus1379",
		Password: "a252f77af72638ea5a0f9e5fbe5f2b2e",
	}

	err := s.repository.CreateUser(context.Background(), user)
	require.Error(s.T(), err)
}

func (s *Suite) TestUserRepository_CreateUserErrExec() {
	var (
		Name     = "Lambok Tulus Simamora"
		UserName = "lamboktulus1379"
		Password = "a252f77af72638ea5a0f9e5fbe5f2b2e"
	)

	prep := s.mock.ExpectPrepare(regexp.QuoteMeta(`INSERT INTO public.user (name, user_name, password) VALUES ($1, $2, $3)`)).
		WillReturnError(fmt.Errorf("error statement"))
	prep.ExpectExec().WithArgs(Name, UserName, Password).WillReturnError(fmt.Errorf("error exec"))

	user := model.User{
		Name:     "Lambok Tulus Simamora",
		UserName: "lamboktulus1379",
		Password: "a252f77af72638ea5a0f9e5fbe5f2b2e",
	}

	err := s.repository.CreateUser(context.Background(), user)
	require.NotNil(s.T(), err)
}
