package repository_test

import (
	"github.com/stretchr/testify/suite"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type BaseRepoSuite struct {
	suite.Suite
	DB     *gorm.DB
	models []interface{}
}

func NewBaseRepoSuite(models ...interface{}) BaseRepoSuite {
	return BaseRepoSuite{
		models: models,
	}
}

func (s *BaseRepoSuite) SetupSuite() {
	dsn := "file::memory:?cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		s.T().Fatalf("Failed to initiate database RAM: %v", err)
	}
	s.DB = db
}

func (s *BaseRepoSuite) SetupTest() {
	if len(s.models) > 0 {
		s.DB.AutoMigrate(s.models...)
	}
}

func (s *BaseRepoSuite) TearDownTest() {
	for _, model := range s.models {
		s.DB.Migrator().DropTable(model)
	}
}