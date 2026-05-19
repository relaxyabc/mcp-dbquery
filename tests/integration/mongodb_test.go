package integration

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/database/mongodb"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// TestMongoDBConnection 测试MongoDB连接
func TestMongoDBConnection(t *testing.T) {
	config := database.DatabaseConfig{
		ID:       "test-mongodb",
		Type:     database.DatabaseTypeMongoDB,
		Host:     "localhost",
		Port:     27017,
		Username: "test_user",
		Password: "test_password",
		Database: "test_db",
		PoolSize: 5,
		Timeout:  30,
	}

	driver := mongodb.NewMongoDBDriver(config)

	// 测试驱动初始化逻辑
	if driver.GetID() != "test-mongodb" {
		t.Error("驱动ID设置错误")
	}

	if driver.GetType() != database.DatabaseTypeMongoDB {
		t.Error("驱动类型设置错误")
	}

	utils.GlobalLogger.Info("MongoDB驱动初始化测试通过")
}

// TestMongoDBReadOnlyValidation 测试MongoDB只读验证
func TestMongoDBReadOnlyValidation(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		wantErr   bool
	}{
		{"允许find操作", "find", false},
		{"允许aggregate操作", "aggregate", false},
		{"允许listCollections操作", "listCollections", false},
		{"允许listIndexes操作", "listIndexes", false},
		{"允许count操作", "count", false},
		{"允许distinct操作", "distinct", false},
		{"禁止insert操作", "insert", true},
		{"禁止insertOne操作", "insertOne", true},
		{"禁止insertMany操作", "insertMany", true},
		{"禁止update操作", "update", true},
		{"禁止updateOne操作", "updateOne", true},
		{"禁止updateMany操作", "updateMany", true},
		{"禁止delete操作", "delete", true},
		{"禁止deleteOne操作", "deleteOne", true},
		{"禁止deleteMany操作", "deleteMany", true},
		{"禁止drop操作", "drop", true},
		{"禁止createCollection操作", "createCollection", true},
		{"禁止createIndex操作", "createIndex", true},
		{"禁止bulkWrite操作", "bulkWrite", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mongodb.ValidateMongoDBOperation(tt.operation)

			if tt.wantErr && err == nil {
				t.Errorf("期望错误但未返回错误: %s", tt.operation)
			}

			if !tt.wantErr && err != nil {
				t.Errorf("期望成功但返回错误: %s, 错误: %v", tt.operation, err)
			}

			utils.GlobalLogger.Info("MongoDB验证测试通过 [操作=%s] [期望错误=%v]",
				tt.operation, tt.wantErr)
		})
	}
}

// TestMongoDBAggregatePipelineValidation 测试MongoDB聚合管道验证
func TestMongoDBAggregatePipelineValidation(t *testing.T) {
	tests := []struct {
		name     string
		pipeline bson.A
		wantErr  bool
	}{
		{
			name: "允许只读聚合管道",
			pipeline: bson.A{
				bson.M{"$match": bson.M{"status": "active"}},
				bson.M{"$group": bson.M{"_id": "$category", "count": bson.M{"$sum": 1}}},
			},
			wantErr: false,
		},
		{
			name: "禁止$out阶段",
			pipeline: bson.A{
				bson.M{"$match": bson.M{"status": "active"}},
				bson.M{"$out": "output_collection"},
			},
			wantErr: true,
		},
		{
			name: "禁止$merge阶段",
			pipeline: bson.A{
				bson.M{"$match": bson.M{"status": "active"}},
				bson.M{"$merge": bson.M{"into": "target_collection"}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mongodb.ValidateAggregatePipeline(tt.pipeline)

			if tt.wantErr && err == nil {
				t.Errorf("期望错误但未返回错误: %s", tt.name)
			}

			if !tt.wantErr && err != nil {
				t.Errorf("期望成功但返回错误: %s, 错误: %v", tt.name, err)
			}
		})
	}
}