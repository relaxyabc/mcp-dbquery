package mongodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/relaxyabc/mcp-dbquery/src/database"
	"github.com/relaxyabc/mcp-dbquery/src/utils"
)

// init() 自动注册 MongoDB 驱动到全局注册表
func init() {
	database.RegisterDriver(database.DatabaseTypeMongoDB, func(config database.DatabaseConfig) database.Database {
		return NewMongoDBDriver(config)
	})
}

// MongoDBDriver MongoDB数据库驱动实现
type MongoDBDriver struct {
	ID     string
	Config database.DatabaseConfig
	Client *mongo.Client
	DB     *mongo.Database
	State  database.ConnectionState
}

// NewMongoDBDriver 创建MongoDB驱动实例
func NewMongoDBDriver(config database.DatabaseConfig) *MongoDBDriver {
	return &MongoDBDriver{
		ID:     config.ID,
		Config: config,
		State:  database.StateDisconnected,
	}
}

// Connect 建立MongoDB连接
func (d *MongoDBDriver) Connect(ctx context.Context) error {
	// 如果已有连接，先关闭
	if d.Client != nil {
		d.State = database.StateClosed
		oldClient := d.Client
		d.Client = nil
		d.DB = nil
		// 尝试关闭旧连接（不阻塞）
		go func() {
			_ = oldClient.Disconnect(context.Background())
		}()
	}

	d.State = database.StateConnecting
	utils.GlobalLogger.Info("MongoDB开始连接 [ID=%s] [用户=%s] [主机=%s:%d] [数据库=%s]",
		d.ID, d.Config.Username, d.Config.Host, d.Config.Port, d.Config.Database)

	// 强制使用带超时的上下文，防止连接卡住
	// 即使传入的 ctx 没有超时，也要确保最多 30 秒
	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 构建连接选项
	clientOpts := options.Client().
		ApplyURI(d.buildConnectionString()).
		SetMaxPoolSize(uint64(d.Config.PoolSize)).
		SetMinPoolSize(1).
		SetMaxConnIdleTime(10 * time.Minute).
		SetConnectTimeout(10 * time.Second).
		SetServerSelectionTimeout(10 * time.Second)

	// 创建客户端
	utils.GlobalLogger.Info("MongoDB正在建立连接...")
	client, err := mongo.Connect(connectCtx, clientOpts)
	if err != nil {
		d.State = database.StateError
		utils.GlobalLogger.LogError("CONNECTION_ERROR", "MongoDB连接失败", err.Error())
		return fmt.Errorf("MongoDB连接失败: %s", err)
	}

	// 测试连接
	utils.GlobalLogger.Info("MongoDB正在Ping测试连接...")
	if err := client.Ping(connectCtx, nil); err != nil {
		d.State = database.StateError
		_ = client.Disconnect(context.Background()) // 清理失败的连接
		utils.GlobalLogger.LogError("CONNECTION_ERROR", "MongoDB Ping测试失败", err.Error())
		return fmt.Errorf("MongoDB连接测试失败: %s", err)
	}

	d.Client = client
	d.DB = client.Database(d.Config.Database)
	d.State = database.StateConnected
	utils.GlobalLogger.Info("MongoDB连接成功 [ID=%s] [用户=%s]", d.ID, d.Config.Username)

	return nil
}

// Close 关闭MongoDB连接
func (d *MongoDBDriver) Close(ctx context.Context) error {
	if d.Client == nil {
		return nil
	}

	d.State = database.StateClosed
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "closing")

	if err := d.Client.Disconnect(ctx); err != nil {
		utils.GlobalLogger.LogError("CONNECTION_ERROR", "MongoDB连接关闭失败", err.Error())
		return err
	}

	d.Client = nil
	d.DB = nil
	utils.GlobalLogger.LogConnection(d.ID, d.GetMaskedConnectionString(), "closed")
	return nil
}

// IsConnected 检查连接状态
// 信任连接池内部状态，避免频繁 Ping 导致的性能问题
func (d *MongoDBDriver) IsConnected() bool {
	// 仅检查状态，不执行 Ping
	// MongoDB 连接池会自动管理连接健康状态
	return d.Client != nil && d.State == database.StateConnected
}

// GetType 返回数据库类型
func (d *MongoDBDriver) GetType() database.DatabaseType {
	return database.DatabaseTypeMongoDB
}

// GetID 返回连接标识符
func (d *MongoDBDriver) GetID() string {
	return d.ID
}

// ExecuteQuery 执行只读查询
func (d *MongoDBDriver) ExecuteQuery(ctx context.Context, query string, limit int, timeout time.Duration) (*database.QueryResult, error) {
	// MongoDB使用find操作，query参数应为JSON格式的filter
	// 实际实现在queries.go中
	return nil, fmt.Errorf("请使用ExecuteFind或ExecuteAggregate方法")
}

// ExecuteSelectQuery 执行SQL SELECT查询（MongoDB不支持，返回错误）
func (d *MongoDBDriver) ExecuteSelectQuery(ctx context.Context, query string, limit int) (*database.QueryResult, error) {
	return nil, fmt.Errorf("MongoDB 不支持 SQL SELECT 查询，请使用 ExecuteFind 执行 MongoDB find 查询")
}

// ValidateQuery 验证查询是否为只读（宪章要求：严格只读）
func (d *MongoDBDriver) ValidateQuery(query string) error {
	// MongoDB验证逻辑在validator.go中实现
	return ValidateMongoOperation(query)
}

// buildConnectionString 构建MongoDB连接字符串（内部使用）
// 支持副本集连接和认证参数
func (d *MongoDBDriver) buildConnectionString() string {
	// 构建主机列表（处理副本集）
	hosts := d.buildHostsString()

	// 构建基础URI
	var uri string
	if d.Config.Username != "" && d.Config.Password != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s/%s",
			d.Config.Username, d.Config.Password,
			hosts, d.Config.Database)
	} else {
		uri = fmt.Sprintf("mongodb://%s/%s", hosts, d.Config.Database)
	}

	// 构建查询参数
	params := d.buildConnectionParams()
	if params != "" {
		uri += "?" + params
	}

	return uri
}

// buildHostsString 构建主机字符串（处理副本集格式）
func (d *MongoDBDriver) buildHostsString() string {
	// 检查是否为副本集（逗号分隔的主机）
	if strings.Contains(d.Config.Host, ",") {
		// 副本集：为每个主机添加端口
		hostList := strings.Split(d.Config.Host, ",")
		hosts := ""
		for i, h := range hostList {
			if i > 0 {
				hosts += ","
			}
			hosts += fmt.Sprintf("%s:%d", strings.TrimSpace(h), d.Config.Port)
		}
		return hosts
	}
	// 单主机
	return fmt.Sprintf("%s:%d", d.Config.Host, d.Config.Port)
}

// buildConnectionParams 构建连接参数字符串
func (d *MongoDBDriver) buildConnectionParams() string {
	params := []string{}

	// 认证源（默认使用admin数据库）
	authSource := d.Config.AuthSource
	if authSource == "" {
		authSource = "admin" // MongoDB通常将用户存储在admin数据库
	}
	params = append(params, fmt.Sprintf("authSource=%s", authSource))

	// 认证机制（如果指定）
	if d.Config.AuthMechanism != "" {
		params = append(params, fmt.Sprintf("authMechanism=%s", d.Config.AuthMechanism))
	}

	// 副本集名称（如果指定）
	if d.Config.ReplicaSet != "" {
		params = append(params, fmt.Sprintf("replicaSet=%s", d.Config.ReplicaSet))
	}

	// 单主机时使用直连模式
	if !strings.Contains(d.Config.Host, ",") {
		params = append(params, "directConnection=true")
	}

	return strings.Join(params, "&")
}

// GetMaskedConnectionString 获取遮蔽密码的连接字符串（日志使用）
func (d *MongoDBDriver) GetMaskedConnectionString() string {
	hosts := d.buildHostsString()
	if d.Config.Username != "" {
		return fmt.Sprintf("mongodb://%s:[REDACTED]@%s/%s",
			d.Config.Username, hosts, d.Config.Database)
	}
	return fmt.Sprintf("mongodb://%s/%s", hosts, d.Config.Database)
}

// GetCollection 获取集合引用
func (d *MongoDBDriver) GetCollection(collectionName string) *mongo.Collection {
	return d.DB.Collection(collectionName)
}

// GetDatabase 获取数据库引用
func (d *MongoDBDriver) GetDatabase() *mongo.Database {
	return d.DB
}

// GetSchema 获取集合结构元数据（实现Database接口）
func (d *MongoDBDriver) GetSchema(ctx context.Context, collectionName string) (*database.SchemaMetadata, error) {
	return d.InferSchema(ctx, collectionName)
}

// GetIndexes 获取索引元数据（实现Database接口）
func (d *MongoDBDriver) GetIndexes(ctx context.Context, collectionName string) (*database.IndexListMetadata, error) {
	return d.ListIndexes(ctx, collectionName)
}

// ListTables 列出所有集合（实现Database接口）
func (d *MongoDBDriver) ListTables(ctx context.Context) ([]string, error) {
	return d.ListCollections(ctx)
}

// InferSchema 从集合文档推断结构
func (d *MongoDBDriver) InferSchema(ctx context.Context, collectionName string) (*database.SchemaMetadata, error) {
	utils.GlobalLogger.Info("推断MongoDB集合结构 [连接=%s] [集合=%s]", d.ID, collectionName)

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(ctx, time.Duration(d.Config.Timeout)*time.Second)
	defer cancel()

	// 获取集合
	collection := d.GetCollection(collectionName)

	// 创建Schema元数据
	schema := database.NewSchemaMetadata(collectionName, d.Config.Database, "collection")

	// 获取样本文档来推断结构
	findOpts := options.Find().SetLimit(100)
	cursor, err := collection.Find(ctx, bson.M{}, findOpts)
	if err != nil {
		return nil, fmt.Errorf("获取样本文档失败: %s", err)
	}
	defer cursor.Close(ctx)

	// 收集所有字段
	fieldTypes := make(map[string]string)
	fieldCounts := make(map[string]int)
	totalDocs := 0

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		totalDocs++

		// 分析文档结构
		for key, value := range doc {
			fieldCounts[key]++
			if _, exists := fieldTypes[key]; !exists {
				fieldTypes[key] = inferBSONType(value)
			}
		}
	}

	// 构建字段元数据
	for fieldName, fieldType := range fieldTypes {
		nullable := fieldCounts[fieldName] < totalDocs
		field := database.FieldMetadata{
			Name:     fieldName,
			Type:     fieldType,
			Nullable: nullable,
		}

		// 特殊处理_id字段
		if fieldName == "_id" {
			field.PrimaryKey = true
			field.Nullable = false
		}

		schema.AddField(field)
	}

	utils.GlobalLogger.Info("MongoDB结构推断完成 [集合=%s] [字段数=%d] [样本文档数=%d]",
		collectionName, len(schema.Fields), totalDocs)

	return schema, nil
}

// ListIndexes 列出集合的所有索引
func (d *MongoDBDriver) ListIndexes(ctx context.Context, collectionName string) (*database.IndexListMetadata, error) {
	utils.GlobalLogger.Info("列出MongoDB索引 [连接=%s] [集合=%s]", d.ID, collectionName)

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(ctx, time.Duration(d.Config.Timeout)*time.Second)
	defer cancel()

	// 获取集合
	collection := d.GetCollection(collectionName)

	// 创建索引列表
	indexList := database.NewIndexListMetadata(collectionName)

	// 获取索引信息
	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取索引列表失败: %s", err)
	}
	defer cursor.Close(ctx)

	// 解析索引信息
	for cursor.Next(ctx) {
		var indexDoc bson.M
		if err := cursor.Decode(&indexDoc); err != nil {
			continue
		}

		// 提取索引信息
		indexName, _ := indexDoc["name"].(string)
		unique, _ := indexDoc["unique"].(bool)
		sparse, _ := indexDoc["sparse"].(bool)
		indexType := "normal"
		if unique {
			indexType = "unique"
		}

		// 创建索引元数据
		index := database.NewIndexMetadata(indexName, collectionName, d.Config.Database)
		index.Type = indexType
		index.SetUnique(unique)
		index.SetSparse(sparse)

		// 提取索引键
		if keys, ok := indexDoc["key"].(bson.M); ok {
			for key, order := range keys {
				orderStr := "ASC"
				if order == -1 {
					orderStr = "DESC"
				}
				index.AddField(key, orderStr)
			}
		}

		indexList.AddIndex(*index)
	}

	utils.GlobalLogger.Info("MongoDB索引列表完成 [集合=%s] [索引数=%d]",
		collectionName, len(indexList.Indexes))

	return indexList, nil
}

// ListCollections 列出数据库中所有集合
func (d *MongoDBDriver) ListCollections(ctx context.Context) ([]string, error) {
	utils.GlobalLogger.Info("列出MongoDB集合 [连接=%s] [数据库=%s]", d.ID, d.Config.Database)

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(ctx, time.Duration(d.Config.Timeout)*time.Second)
	defer cancel()

	// 获取集合列表
	cursor, err := d.DB.ListCollections(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("获取集合列表失败: %s", err)
	}
	defer cursor.Close(ctx)

	// 收集集合名称
	collections := []string{}
	for cursor.Next(ctx) {
		var collDoc bson.M
		if err := cursor.Decode(&collDoc); err != nil {
			continue
		}
		name, _ := collDoc["name"].(string)
		if name != "" && !strings.HasPrefix(name, "system.") { // 排除系统集合
			collections = append(collections, name)
		}
	}

	utils.GlobalLogger.Info("MongoDB集合列表完成 [数据库=%s] [集合数=%d]",
		d.Config.Database, len(collections))

	return collections, nil
}

// inferBSONType 推断BSON值的类型
func inferBSONType(value interface{}) string {
	switch value.(type) {
	case string:
		return "string"
	case int, int32, int64:
		return "int"
	case float64:
		return "double"
	case bool:
		return "boolean"
	case bson.M:
		return "object"
	case bson.A, []interface{}:
		return "array"
	case primitive.ObjectID:
		return "objectId"
	case primitive.DateTime, time.Time:
		return "date"
	case primitive.Binary:
		return "binData"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}
