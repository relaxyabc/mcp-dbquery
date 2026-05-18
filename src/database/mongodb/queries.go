package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// ExecuteFind 执行MongoDB find查询
func (d *MongoDBDriver) ExecuteFind(ctx context.Context, collectionName string, filter bson.M, limit int) (*database.QueryResult, error) {
	start := time.Now()

	// 验证操作是否为只读
	if err := d.ValidateQuery("find"); err != nil {
		return nil, err
	}

	utils.GlobalLogger.Info("执行MongoDB find查询 [连接=%s] [集合=%s]", d.ID, collectionName)

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(ctx, time.Duration(d.Config.Timeout)*time.Second)
	defer cancel()

	// 获取集合
	collection := d.GetCollection(collectionName)

	// 设置查询选项
	findOpts := options.Find().
		SetLimit(int64(limit)).
		SetBatchSize(int32(limit))

	// 执行查询
	cursor, err := collection.Find(ctx, filter, findOpts)
	if err != nil {
		return database.NewErrorResult(d.ID, "QUERY_ERROR", fmt.Sprintf("查询执行失败: %s", err)), err
	}
	defer cursor.Close(ctx)

	// 读取结果
	data := []map[string]interface{}{}
	actualCount := 0

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			utils.GlobalLogger.Warn("文档解码失败: %s", err)
			continue
		}

		// 添加到结果（不超过限制）
		if len(data) < limit {
			data = append(data, convertBSONToMap(doc))
		}
		actualCount++

		// 达到限制后停止读取
		if actualCount >= limit {
			break
		}
	}

	// 检查游标错误
	if err := cursor.Err(); err != nil {
		return database.NewErrorResult(d.ID, "QUERY_ERROR", fmt.Sprintf("游标错误: %s", err)), err
	}

	// 构建结果
	executionTime := database.MeasureExecutionTime(start)
	resultType := database.QueryTypeData

	if actualCount > limit {
		return database.NewTruncatedResult(d.ID, data, resultType, executionTime, actualCount), nil
	}

	utils.GlobalLogger.Info("MongoDB查询完成 [集合=%s] [返回文档数=%d] [耗时=%dms]", collectionName, len(data), executionTime)
	return database.NewQueryResult(d.ID, data, resultType, executionTime), nil
}

// ExecuteAggregate 执行MongoDB聚合查询
func (d *MongoDBDriver) ExecuteAggregate(ctx context.Context, collectionName string, pipeline bson.A, limit int) (*database.QueryResult, error) {
	start := time.Now()

	// 验证操作是否为只读
	if err := d.ValidateQuery("aggregate"); err != nil {
		return nil, err
	}

	// 验证聚合管道
	if err := ValidateAggregatePipeline(pipeline); err != nil {
		return nil, err
	}

	utils.GlobalLogger.Info("执行MongoDB聚合查询 [连接=%s] [集合=%s]", d.ID, collectionName)

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(ctx, time.Duration(d.Config.Timeout)*time.Second)
	defer cancel()

	// 获取集合
	collection := d.GetCollection(collectionName)

	// 设置聚合选项
	aggOpts := options.Aggregate()

	// 执行聚合
	cursor, err := collection.Aggregate(ctx, pipeline, aggOpts)
	if err != nil {
		return database.NewErrorResult(d.ID, "QUERY_ERROR", fmt.Sprintf("聚合执行失败: %s", err)), err
	}
	defer cursor.Close(ctx)

	// 读取结果
	data := []map[string]interface{}{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			utils.GlobalLogger.Warn("文档解码失败: %s", err)
			continue
		}

		if len(data) < limit {
			data = append(data, convertBSONToMap(doc))
		}
	}

	// 构建结果
	executionTime := database.MeasureExecutionTime(start)

	utils.GlobalLogger.Info("MongoDB聚合完成 [集合=%s] [返回文档数=%d] [耗时=%dms]", collectionName, len(data), executionTime)
	return database.NewQueryResult(d.ID, data, database.QueryTypeData, executionTime), nil
}

// ExecuteCount 执行计数查询
func (d *MongoDBDriver) ExecuteCount(ctx context.Context, collectionName string, filter bson.M) (int64, error) {
	// 验证操作是否为只读
	if err := d.ValidateQuery("count"); err != nil {
		return 0, err
	}

	utils.GlobalLogger.Info("执行MongoDB计数查询 [连接=%s] [集合=%s]", d.ID, collectionName)

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(ctx, time.Duration(d.Config.Timeout)*time.Second)
	defer cancel()

	// 获取集合
	collection := d.GetCollection(collectionName)

	// 执行计数
	countOpts := options.Count()
	count, err := collection.CountDocuments(ctx, filter, countOpts)
	if err != nil {
		return 0, err
	}

	utils.GlobalLogger.Info("MongoDB计数完成 [集合=%s] [文档数=%d]", collectionName, count)
	return count, nil
}

// ExecuteDistinct 执行distinct查询
func (d *MongoDBDriver) ExecuteDistinct(ctx context.Context, collectionName string, fieldName string, filter bson.M) ([]interface{}, error) {
	// 验证操作是否为只读
	if err := d.ValidateQuery("distinct"); err != nil {
		return nil, err
	}

	utils.GlobalLogger.Info("执行MongoDB distinct查询 [连接=%s] [集合=%s] [字段=%s]", d.ID, collectionName, fieldName)

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(ctx, time.Duration(d.Config.Timeout)*time.Second)
	defer cancel()

	// 获取集合
	collection := d.GetCollection(collectionName)

	// 执行distinct
	distinctOpts := options.Distinct()
	values, err := collection.Distinct(ctx, fieldName, filter, distinctOpts)
	if err != nil {
		return nil, err
	}

	utils.GlobalLogger.Info("MongoDB distinct完成 [集合=%s] [字段=%s] [唯一值数=%d]", collectionName, fieldName, len(values))
	return values, nil
}

// FindOne 查询单个文档
func (d *MongoDBDriver) FindOne(ctx context.Context, collectionName string, filter bson.M) (map[string]interface{}, error) {
	utils.GlobalLogger.Info("执行MongoDB findOne查询 [连接=%s] [集合=%s]", d.ID, collectionName)

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(ctx, time.Duration(d.Config.Timeout)*time.Second)
	defer cancel()

	// 获取集合
	collection := d.GetCollection(collectionName)

	// 执行查询
	findOneOpts := options.FindOne()
	result := collection.FindOne(ctx, filter, findOneOpts)

	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("文档不存在")
		}
		return nil, result.Err()
	}

	var doc bson.M
	if err := result.Decode(&doc); err != nil {
		return nil, err
	}

	return convertBSONToMap(doc), nil
}

// convertBSONToMap 将BSON文档转换为普通map
func convertBSONToMap(doc bson.M) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range doc {
		// 处理特殊类型
		switch v := value.(type) {
		case bson.RawValue:
			// 解析原始BSON值
			switch v.Type {
			case bson.TypeString:
				result[key] = v.StringValue()
			case bson.TypeInt32:
				result[key] = v.Int32()
			case bson.TypeInt64:
				result[key] = v.Int64()
			case bson.TypeDouble:
				result[key] = v.Double()
			case bson.TypeBoolean:
				result[key] = v.Boolean()
			case bson.TypeDateTime:
				result[key] = v.Time()
			case bson.TypeObjectID:
				result[key] = v.ObjectID().Hex()
			default:
				result[key] = value
			}
		case bson.M:
			result[key] = convertBSONToMap(v)
		case bson.A:
			array := []interface{}{}
			for _, item := range v {
				if m, ok := item.(bson.M); ok {
					array = append(array, convertBSONToMap(m))
				} else {
					array = append(array, item)
				}
			}
			result[key] = array
		default:
			result[key] = value
		}
	}

	return result
}
