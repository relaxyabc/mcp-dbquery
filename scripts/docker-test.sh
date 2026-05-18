#!/bin/bash

# Docker Test Environment Script
# 使用 Docker 启动测试数据库容器进行集成测试

set -e

echo "=========================================="
echo "Docker Test Environment Setup"
echo "=========================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# MySQL 容器配置
MYSQL_CONTAINER="db-tools-mysql-test"
MYSQL_PORT=3306
MYSQL_USER="test_user"
MYSQL_PASS="test_password"
MYSQL_DB="test_db"

# MongoDB 容器配置
MONGO_CONTAINER="db-tools-mongo-test"
MONGO_PORT=27017
MONGO_USER="test_user"
MONGO_PASS="test_password"
MONGO_DB="test_db"

# 启动 MySQL 容器
start_mysql() {
    echo ""
    echo "启动 MySQL 测试容器..."

    # 检查容器是否已存在
    if docker ps -a --format '{{.Names}}' | grep -q "^${MYSQL_CONTAINER}$"; then
        echo "${YELLOW}MySQL容器已存在，正在重启...${NC}"
        docker start ${MYSQL_CONTAINER}
    else
        docker run -d \
            --name ${MYSQL_CONTAINER} \
            -e MYSQL_ROOT_PASSWORD=root_password \
            -e MYSQL_USER=${MYSQL_USER} \
            -e MYSQL_PASSWORD=${MYSQL_PASS} \
            -e MYSQL_DATABASE=${MYSQL_DB} \
            -p ${MYSQL_PORT}:3306 \
            mysql:8.0

        echo "${GREEN}✓ MySQL容器已启动${NC}"
    fi

    # 等待MySQL就绪
    echo "等待MySQL就绪..."
    sleep 10

    # 创建测试数据
    echo "创建测试数据..."
    docker exec -i ${MYSQL_CONTAINER} mysql -u${MYSQL_USER} -p${MYSQL_PASS} ${MYSQL_DB} <<EOF
CREATE TABLE IF NOT EXISTS users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS orders (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT,
    amount DECIMAL(10, 2),
    status VARCHAR(20),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- 创建索引
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_orders_user_id ON orders(user_id);

-- 插入测试数据
INSERT INTO users (name, email) VALUES ('Alice', 'alice@test.com');
INSERT INTO users (name, email) VALUES ('Bob', 'bob@test.com');
INSERT INTO users (name, email) VALUES ('Charlie', 'charlie@test.com');

INSERT INTO orders (user_id, amount, status) VALUES (1, 100.00, 'completed');
INSERT INTO orders (user_id, amount, status) VALUES (2, 200.00, 'pending');
INSERT INTO orders (user_id, amount, status) VALUES (1, 150.00, 'completed');
EOF

    echo "${GREEN}✓ MySQL测试数据已创建${NC}"
}

# 启动 MongoDB 容器
start_mongodb() {
    echo ""
    echo "启动 MongoDB 测试容器..."

    # 检查容器是否已存在
    if docker ps -a --format '{{.Names}}' | grep -q "^${MONGO_CONTAINER}$"; then
        echo "${YELLOW}MongoDB容器已存在，正在重启...${NC}"
        docker start ${MONGO_CONTAINER}
    else
        docker run -d \
            --name ${MONGO_CONTAINER} \
            -e MONGO_INITDB_ROOT_USERNAME=${MONGO_USER} \
            -e MONGO_INITDB_ROOT_PASSWORD=${MONGO_PASS} \
            -p ${MONGO_PORT}:27017 \
            mongo:7.0

        echo "${GREEN}✓ MongoDB容器已启动${NC}"
    fi

    # 等待MongoDB就绪
    echo "等待MongoDB就绪..."
    sleep 5

    # 创建测试数据
    echo "创建测试数据..."
    docker exec -i ${MONGO_CONTAINER} mongosh -u${MONGO_USER} -p${MONGO_PASS} --authenticationDatabase admin <<EOF
use ${MONGO_DB}

// 创建集合
db.createCollection('products')
db.createCollection('reviews')

// 插入测试数据
db.products.insertMany([
    { name: 'Product A', price: 99.99, category: 'electronics', stock: 100 },
    { name: 'Product B', price: 49.99, category: 'books', stock: 50 },
    { name: 'Product C', price: 149.99, category: 'electronics', stock: 25 }
])

db.reviews.insertMany([
    { productId: ObjectId(), userId: 'user1', rating: 5, comment: 'Great product' },
    { productId: ObjectId(), userId: 'user2', rating: 4, comment: 'Good value' },
    { productId: ObjectId(), userId: 'user3', rating: 3, comment: 'Average' }
])

// 创建索引
db.products.createIndex({ name: 1 })
db.products.createIndex({ category: 1, price: -1 })
db.reviews.createIndex({ productId: 1 })
EOF

    echo "${GREEN}✓ MongoDB测试数据已创建${NC}"
}

# 运行集成测试
run_tests() {
    echo ""
    echo "运行集成测试..."
    echo "----------------------------------------"

    # 设置测试环境变量
    export TEST_MYSQL_HOST="localhost"
    export TEST_MYSQL_PORT=${MYSQL_PORT}
    export TEST_MYSQL_USER=${MYSQL_USER}
    export TEST_MYSQL_PASS=${MYSQL_PASS}
    export TEST_MYSQL_DB=${MYSQL_DB}

    export TEST_MONGO_HOST="localhost"
    export TEST_MONGO_PORT=${MONGO_PORT}
    export TEST_MONGO_USER=${MONGO_USER}
    export TEST_MONGO_PASS=${MONGO_PASS}
    export TEST_MONGO_DB=${MONGO_DB}

    # 运行测试
    go test ./tests/integration/... -v -timeout 60s

    echo "${GREEN}✓ 集成测试完成${NC}"
}

# 停止容器
stop_containers() {
    echo ""
    echo "停止测试容器..."

    docker stop ${MYSQL_CONTAINER} 2>/dev/null || true
    docker stop ${MONGO_CONTAINER} 2>/dev/null || true

    echo "${GREEN}✓ 容器已停止${NC}"
}

# 清理容器
cleanup() {
    echo ""
    echo "清理测试容器..."

    docker rm -f ${MYSQL_CONTAINER} 2>/dev/null || true
    docker rm -f ${MONGO_CONTAINER} 2>/dev/null || true

    echo "${GREEN}✓ 容器已清理${NC}"
}

# 主函数
main() {
    case "$1" in
        start)
            start_mysql
            start_mongodb
            ;;
        test)
            start_mysql
            start_mongodb
            run_tests
            stop_containers
            ;;
        stop)
            stop_containers
            ;;
        cleanup)
            cleanup
            ;;
        all)
            start_mysql
            start_mongodb
            run_tests
            stop_containers
            ;;
        *)
            echo "用法: $0 {start|test|stop|cleanup|all}"
            echo ""
            echo "  start   - 启动测试数据库容器"
            echo "  test    - 启动容器并运行测试"
            echo "  stop    - 停止容器"
            echo "  cleanup - 删除容器"
            echo "  all     - 完整流程：启动->测试->停止"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"