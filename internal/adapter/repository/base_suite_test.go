package repository_test

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// BaseRepoSuite gives every test a fresh in-memory SQLite database whose
// schema is created from the given models.
//
// SQLite keeps the tests fast and dependency-free. For PostgreSQL-specific
// behaviour (partial indexes, JSONB, ...) add integration tests against a
// real database (e.g. testcontainers-go) behind a build tag.
type BaseRepoSuite struct {
	suite.Suite
	DB     *gorm.DB
	models []any
}

func NewBaseRepoSuite(models ...any) BaseRepoSuite {
	return BaseRepoSuite{models: models}
}

func (s *BaseRepoSuite) SetupTest() {
	// A unique DSN per test gives full isolation between tests.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", s.T().Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		TranslateError: true, // same as production
		Logger:         logger.Discard,
	})
	s.Require().NoError(err)
	s.Require().NoError(db.AutoMigrate(s.models...))
	s.DB = db
}

func (s *BaseRepoSuite) TearDownTest() {
	if sqlDB, err := s.DB.DB(); err == nil {
		_ = sqlDB.Close()
	}
}
