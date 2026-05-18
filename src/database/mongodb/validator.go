package mongodb

import (
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

// AllowedOperations MongoDB允许的操作（只读操作）
// 宪章原则 I 要求：仅允许find、aggregate、listCollections、listIndexes
var AllowedOperations = []string{
	"find",            // 查询文档
	"aggregate",       // 聚合查询
	"listCollections", // 列出集合
	"listIndexes",     // 列出索引
	"count",           // 计数（只读）
	"distinct",        // 获取唯一值（只读）
	"mapReduce",       // MapReduce（只读模式）
}

// ForbiddenOperations MongoDB禁止的操作（修改操作）
// 宪章原则 I 要求：严格禁止所有修改操作
var ForbiddenOperations = []string{
	"insert",            // 插入文档
	"insertOne",         // 插入单个文档
	"insertMany",        // 插入多个文档
	"update",            // 更新文档
	"updateOne",         // 更新单个文档
	"updateMany",        // 更新多个文档
	"delete",            // 删除文档
	"deleteOne",         // 删除单个文档
	"deleteMany",        // 删除多个文档
	"drop",              // 删除集合/数据库
	"dropCollection",    // 删除集合
	"dropDatabase",      // 删除数据库
	"createCollection",  // 创建集合
	"createIndex",       // 创建索引（修改）
	"dropIndex",         // 删除索引（修改）
	"renameCollection",  // 重命名集合
	"bulkWrite",         // 批量写入操作
	"replaceOne",        // 替换文档
	"findOneAndDelete",  // 查找并删除
	"findOneAndUpdate",  // 查找并更新
	"findOneAndReplace", // 查找并替换
}

// ForbiddenPipelineStages MongoDB聚合管道中禁止的阶段
// 宪章要求：聚合管道不能包含修改数据的阶段
var ForbiddenPipelineStages = []string{
	"$out",     // 输出到集合（修改）
	"$merge",   // 合并到集合（修改）
	"$insert",  // 插入阶段
	"$update",  // 更新阶段
	"$delete",  // 删除阶段
	"$replace", // 替换阶段
}

// ValidateMongoOperation 验证MongoDB操作是否为只读
func ValidateMongoOperation(operation string) error {
	// 去除前后空白
	operation = strings.TrimSpace(operation)

	// 检查空操作
	if operation == "" {
		return fmt.Errorf("操作不能为空")
	}

	// 获取操作类型（第一个关键字）
	firstWord := strings.ToLower(strings.Fields(operation)[0])

	// 检查是否为禁止的操作
	for _, forbidden := range ForbiddenOperations {
		if firstWord == forbidden {
			return fmt.Errorf("操作 %s 不被允许：只允许只读操作 (find, aggregate, listCollections, listIndexes)", forbidden)
		}
	}

	// 检查是否为允许的操作
	allowed := false
	for _, permit := range AllowedOperations {
		if firstWord == permit {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("操作 %s 不在允许列表中：只允许只读操作 (find, aggregate, listCollections, listIndexes)", firstWord)
	}

	return nil
}

// ValidateMongoDBOperation 验证MongoDB操作类型
func ValidateMongoDBOperation(operationType string) error {
	opType := strings.ToLower(operationType)

	// 检查禁止操作
	for _, forbidden := range ForbiddenOperations {
		if opType == forbidden {
			return fmt.Errorf("操作类型 %s 不被允许：只允许只读操作", forbidden)
		}
	}

	// 检查允许操作
	allowed := false
	for _, permit := range AllowedOperations {
		if opType == permit {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("操作类型 %s 不被允许", opType)
	}

	return nil
}

// ValidateAggregatePipeline 验证聚合管道是否为只读
func ValidateAggregatePipeline(pipeline interface{}) error {
	// pipeline通常是bson.A或[]bson.M
	// 检查是否包含禁止的阶段

	pipelineDocs, ok := pipeline.(bson.A)
	if !ok {
		// 尝试其他格式
		pipelineSlice, ok := pipeline.([]bson.M)
		if !ok {
			return nil // 无法解析，跳过检查
		}
		pipelineDocs = bson.A{}
		for _, doc := range pipelineSlice {
			pipelineDocs = append(pipelineDocs, doc)
		}
	}

	// 遍历管道阶段
	for _, stage := range pipelineDocs {
		stageDoc, ok := stage.(bson.M)
		if !ok {
			continue
		}

		// 检查每个阶段是否包含禁止操作
		for key := range stageDoc {
			keyLower := strings.ToLower(key)
			for _, forbidden := range ForbiddenPipelineStages {
				if keyLower == forbidden {
					return fmt.Errorf("聚合管道阶段 %s 不被允许：只能进行只读聚合", key)
				}
			}
		}
	}

	return nil
}

// IsReadOnlyOperation 快速判断操作是否为只读（简化版）
func IsReadOnlyOperation(operationType string) bool {
	return ValidateMongoDBOperation(operationType) == nil
}

// GetOperationType 获取操作类型
func GetOperationType(operation string) string {
	operation = strings.TrimSpace(operation)
	if operation == "" {
		return "UNKNOWN"
	}

	firstWord := strings.ToLower(strings.Fields(operation)[0])

	switch firstWord {
	case "find":
		return "find"
	case "aggregate":
		return "aggregate"
	case "listcollections":
		return "listCollections"
	case "listindexes":
		return "listIndexes"
	case "count":
		return "count"
	case "distinct":
		return "distinct"
	default:
		return "UNKNOWN"
	}
}
