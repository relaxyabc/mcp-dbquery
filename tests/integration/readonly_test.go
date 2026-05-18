package integration

import (
	"fmt"
	"testing"

	"github.com/relaxyabc/mcp-dbquery/src/database/mongodb"
	"github.com/relaxyabc/mcp-dbquery/src/database/mysql"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// TestReadOnlyEnforcementMySQL 测试MySQL只读操作强制执行
// 宪章原则 I 要求：100%拒绝修改操作
func TestReadOnlyEnforcementMySQL(t *testing.T) {
	utils.GlobalLogger.Info("开始MySQL只读强制测试")

	// 所有禁止操作的测试
	forbiddenOperations := []struct {
		name  string
		query string
	}{
		{"INSERT", "INSERT INTO users (id, name) VALUES (1, 'test')"},
		{"UPDATE", "UPDATE users SET name = 'modified' WHERE id = 1"},
		{"DELETE", "DELETE FROM users WHERE id = 1"},
		{"DROP_TABLE", "DROP TABLE users"},
		{"DROP_DATABASE", "DROP DATABASE test_db"},
		{"ALTER_TABLE_ADD", "ALTER TABLE users ADD COLUMN age INT"},
		{"ALTER_TABLE_DROP", "ALTER TABLE users DROP COLUMN name"},
		{"ALTER_TABLE_MODIFY", "ALTER TABLE users MODIFY COLUMN name VARCHAR(100)"},
		{"CREATE_TABLE", "CREATE TABLE forbidden_test (id INT)"},
		{"CREATE_DATABASE", "CREATE DATABASE forbidden_db"},
		{"TRUNCATE", "TRUNCATE TABLE users"},
		{"REPLACE", "REPLACE INTO users (id, name) VALUES (1, 'test')"},
		{"GRANT", "GRANT ALL PRIVILEGES ON *.* TO 'user'@'host'"},
		{"REVOKE", "REVOKE ALL PRIVILEGES ON *.* FROM 'user'@'host'"},
		{"LOAD_DATA", "LOAD DATA INFILE 'data.csv' INTO TABLE users"},
		{"LOCK_TABLES", "LOCK TABLES users WRITE"},
		{"UNLOCK_TABLES", "UNLOCK TABLES"},
	}

	allRejected := true
	for _, op := range forbiddenOperations {
		err := mysql.ValidateMySQLQuery(op.query)
		if err == nil {
			t.Errorf("禁止操作未被拒绝: %s - %s", op.name, op.query)
			allRejected = false
		} else {
			utils.GlobalLogger.Info("MySQL禁止操作正确拒绝 [操作=%s] [错误=%s]", op.name, err)
		}
	}

	if allRejected {
		utils.GlobalLogger.Info("MySQL只读强制测试完全通过: 100%%拒绝修改操作")
	}
}

// TestReadOnlyEnforcementMongoDB 测试MongoDB只读操作强制执行
// 宪章原则 I 要求：100%拒绝修改操作
func TestReadOnlyEnforcementMongoDB(t *testing.T) {
	utils.GlobalLogger.Info("开始MongoDB只读强制测试")

	// 所有禁止操作的测试
	forbiddenOperations := []struct {
		name      string
		operation string
	}{
		{"insert", "insert"},
		{"insertOne", "insertOne"},
		{"insertMany", "insertMany"},
		{"update", "update"},
		{"updateOne", "updateOne"},
		{"updateMany", "updateMany"},
		{"delete", "delete"},
		{"deleteOne", "deleteOne"},
		{"deleteMany", "deleteMany"},
		{"drop", "drop"},
		{"dropCollection", "dropCollection"},
		{"dropDatabase", "dropDatabase"},
		{"createCollection", "createCollection"},
		{"createIndex", "createIndex"},
		{"dropIndex", "dropIndex"},
		{"renameCollection", "renameCollection"},
		{"bulkWrite", "bulkWrite"},
		{"replaceOne", "replaceOne"},
		{"findOneAndDelete", "findOneAndDelete"},
		{"findOneAndUpdate", "findOneAndUpdate"},
		{"findOneAndReplace", "findOneAndReplace"},
	}

	allRejected := true
	for _, op := range forbiddenOperations {
		err := mongodb.ValidateMongoDBOperation(op.operation)
		if err == nil {
			t.Errorf("禁止操作未被拒绝: %s - %s", op.name, op.operation)
			allRejected = false
		} else {
			utils.GlobalLogger.Info("MongoDB禁止操作正确拒绝 [操作=%s] [错误=%s]", op.name, err)
		}
	}

	if allRejected {
		utils.GlobalLogger.Info("MongoDB只读强制测试完全通过: 100%%拒绝修改操作")
	}
}

// TestReadOnlyEnforcementEdgeCases 测试边界情况
func TestReadOnlyEnforcementEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		dbType  string
		wantErr bool
	}{
		// MySQL边界情况
		{"MySQL-空查询", "", "mysql", true},
		{"MySQL-带注释的SELECT", "/* comment */ SELECT * FROM users", "mysql", false},
		{"MySQL-带分号的SELECT", "SELECT * FROM users;", "mysql", false},
		{"MySQL-多语句注入尝试", "SELECT * FROM users; DROP TABLE users;", "mysql", true},
		{"MySQL-子查询中的INSERT", "SELECT * FROM users WHERE id IN (INSERT INTO temp VALUES (1))", "mysql", true},
		{"MySQL-大小写混合", "select * from users", "mysql", false},
		{"MySQL-DESC简写", "DESC users", "mysql", false},

		// MongoDB边界情况
		{"MongoDB-空操作", "", "mongodb", true},
		{"MongoDB-大小写混合", "Find", "mongodb", false},
		{"MongoDB-未知操作", "unknownOperation", "mongodb", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error

			if tt.dbType == "mysql" {
				err = mysql.ValidateMySQLQuery(tt.query)
			} else {
				err = mongodb.ValidateMongoDBOperation(tt.query)
			}

			if tt.wantErr && err == nil {
				t.Errorf("期望错误但未返回错误: %s", tt.query)
			}

			if !tt.wantErr && err != nil {
				t.Errorf("期望成功但返回错误: %v", err)
			}
		})
	}
}

// TestConstitutionCompliance 测试宪章合规性
// 宪章原则 I (NON-NEGOTIABLE): 严格只读操作
func TestConstitutionCompliance(t *testing.T) {
	utils.GlobalLogger.Info("开始宪章合规性测试")

	// SC-002要求：100%拒绝修改操作
	// 测试MySQL
	mysqlRejectedCount := 0
	mysqlTotalCount := 0

	for _, forbidden := range mysql.ForbiddenKeywords {
		mysqlTotalCount++
		query := fmt.Sprintf("%s test_table", forbidden)
		if mysql.ValidateMySQLQuery(query) != nil {
			mysqlRejectedCount++
		}
	}

	mysqlRejectRate := float64(mysqlRejectedCount) / float64(mysqlTotalCount) * 100
	if mysqlRejectRate != 100.0 {
		t.Errorf("MySQL拒绝率未达到100%%: %.2f%%", mysqlRejectRate)
	}

	// 测试MongoDB
	mongoRejectedCount := 0
	mongoTotalCount := 0

	for _, forbidden := range mongodb.ForbiddenOperations {
		mongoTotalCount++
		if mongodb.ValidateMongoDBOperation(forbidden) != nil {
			mongoRejectedCount++
		}
	}

	mongoRejectRate := float64(mongoRejectedCount) / float64(mongoTotalCount) * 100
	if mongoRejectRate != 100.0 {
		t.Errorf("MongoDB拒绝率未达到100%%: %.2f%%", mongoRejectRate)
	}

	utils.GlobalLogger.Info("宪章合规性测试通过 [MySQL拒绝率=%.2f%%] [MongoDB拒绝率=%.2f%%]",
		mysqlRejectRate, mongoRejectRate)
}
