// Package tests 提供基础集成测试示例。
package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// suite 示例：展示测试套件结构。
type testsuite struct {
	suite.Suite
	db *gorm.DB
}

func (s *testsuite) SetupTest()    {}
func (s *testsuite) TearDownTest() {}

func (s *testsuite) TestPing() {
	s.NotNil(s.db)
	// 示例断言：DB 连接可用
	err := s.db.WithContext(context.Background()).Exec("SELECT 1").Error
	assert.NoError(s.T(), err)
}

// Test 入口：若 MYSQL_TEST_DSN 存在则运行套件，否则跳过。
func TestSuiteExample(t *testing.T) {
	db := SetupTestDB(t)
	if db == nil {
		t.Skip("integration tests skipped")
	}
	defer RollbackTestDB(t, db)

	suite.Run(t, &testsuite{db: db})
}
