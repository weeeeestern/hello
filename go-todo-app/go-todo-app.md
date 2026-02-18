
# Go + MySQL + Docker Compose Todo 앱 구현 정리
# go-todo-app

A simple Todo REST API built with Go (Gin + GORM) and MySQL, containerized with Docker Compose.

## Structure

- `docker-compose.yml`  
  Defines `db` (MySQL) and `app` (Go API) services, env vars, ports, and volume.

- `mysql/initdb.d/init.sql`  
  Initializes the DB schema (creates `todos` table) on first MySQL startup.

- `Dockerfile`  
  Builds the Go backend image and prepares `wait-for.sh`.

- `wait-for.sh`  
  Waits until MySQL is reachable before starting the API container.

- `db.go`  
  Connects to MySQL using env vars, retries DB connection, and runs `AutoMigrate`.

- `todo.go`  
  Todo model struct.

- `todo_handler.go`  
  CRUD handlers (GetTodos, GetTodo, CreateTodo, UpdateTodo, DeleteTodo).

- `main.go`  
  Gin router + endpoint registration.

## Run

```bash
docker compose up --build
```
API: http://localhost:8080

DB (host access): localhost:3307

Endpoints

- GET /todo 
- GET /todo/:id 
- POST /todo 
- PUT /todo/:id 
- DELETE /todo/:id
---

# 1. 데이터베이스 스키마 설계

```sql
CREATE TABLE todos (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    is_completed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
````

---

# 2. MySQL을 Docker Compose 파일로 작성 후 설치

Docker Compose : 여러 개의 도커 컨테이너를 정의하고 실행할 수 있는 도구

`docker-compose.yml` 파일을 사용하여 여러 컨테이너(서비스)의 설정과 그 관계들을 설정

> Dockerfile은 하나의 이미지(컨테이너) 정의

---

## 구성 요소

### 서비스(Service)

컨테이너의 정의

* 어떤 이미지를 사용할지
* 어떤 환경변수를 설정할지
* 어떤 포트를 노출할지

### 네트워크(Network)

서비스 간의 통신을 정의

### 볼륨(Volume)

데이터의 영구 저장을 정의

```yaml
volumes:
  - db-data:/var/lib/mysql
```

→ 로컬의 특정 디렉토리에 `db-data` 라는 볼륨을 바인드 마운트

---

## docker-compose.yml 예시

```yaml
services:
  db:
    image: mysql:latest
    container_name: todo-db
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: todoapp
      MYSQL_USER: user
      MYSQL_PASSWORD: password
      TZ: Asia/Seoul
    ports:
      - "3307:3306"
    volumes:
      - db-data:/var/lib/mysql
      - /etc/timezone:/etc/timezone:ro
      - ./mysql/initdb.d:/docker-entrypoint-initdb.d

  app:
     build: .
     container_name: todo-app
     ports:
       - "8080:8080"
     depends_on:
       - db

volumes:
  db-data:
```

---

## init.sql 기능

* `init.sql`을 통해 어플리케이션(app)이 실행되기 전에 DB 구조 설정
* volumes 사용 → 데이터 영구 저장
* DB 파일 로컬에 저장

app service에서는

* 클라이언트 요청 처리
* DB와 상호작용

---

# 3. Go 애플리케이션으로 백엔드 코드 작성 (Todo CRUD)

---

## 사용 패키지

```go
import (
    "github.com/gin-gonic/gin"
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    "time"
)
```

* gin : 라우팅, JSON 직렬화
* gorm mysql 드라이버 : MySQL과 상호작용
* gorm : Go ORM (객체와 DB 매핑)

※ docker-compose의 init.sql에서 이미 create table 했기 때문에 AutoMigrate 호출 X (초기에는)

---

## Todo 구조체

```go
type Todo struct {
    ID          uint      `gorm:"primaryKey"`
    Title       string    `json:"title"`
    IsCompleted bool      `json:"is_completed"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

---

## DB 연결 설정

```go
var db *gorm.DB
var err error
```

* `var db *gorm.DB` : DB 연결 객체 포인터
* 초기값은 nil

```go
func init() {
    dsn := "user:password@tcp(localhost:3306)/todoapp?charset=utf8mb4&parseTime=True&loc=Local"
    db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
    if err != nil {
        panic("failed to connect database")
    }
}
```

* 패키지 초기화 시 자동 호출
* dsn : DB 연결 문자열
* mysql.Open(dsn) : mysql 드라이버 객체 반환
* gorm.Config{} : gorm 설정 객체

포인터를 전달하여 메모리 효율 증가, 원본 수정 가능

---

## main 함수

```go
func main() {
    r := gin.Default()

    r.GET("/todo", GetTodos)
    r.GET("/todo/:id", GetTodo)
    r.POST("/todo", CreateTodo)
    r.PUT("/todo/:id", UpdateTodo)
    r.DELETE("/todo/:id", DeleteTodo)

    r.Run()
}
```

* URL과 핸들러 함수 매핑
* API 서버 구현

---

## CRUD 구현

### GetTodos

```go
func GetTodos(c *gin.Context) {
    var todos []Todo
    if err := db.Find(&todos).Error; err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, todos)
}
```

* `db.Find(&todos)` : 전체 조회
* 슬라이스 : 동적 배열
* 500 : 내부 서버 오류
* 200 : 성공

---

### GetTodo

```go
func GetTodo(c *gin.Context) {
    var todo Todo
    id := c.Param("id")
    if err := db.First(&todo, id).Error; err != nil {
        c.JSON(404, gin.H{"error": "Todo not found"})
        return
    }
    c.JSON(200, todo)
}
```

* 특정 ID 조회
* 없으면 404

---

### CreateTodo

```go
func CreateTodo(c *gin.Context) {
    var todo Todo
    if err := c.ShouldBindJSON(&todo); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    if err := db.Create(&todo).Error; err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, todo)
}
```

* 400 : 클라이언트 요청 오류
* 500 : 서버 오류

---

### UpdateTodo

```go
func UpdateTodo(c *gin.Context) {
    var todo Todo
    id := c.Param("id")
    if err := db.First(&todo, id).Error; err != nil {
        c.JSON(404, gin.H{"error": "Todo not found"})
        return
    }
    if err := c.ShouldBindJSON(&todo); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    if err := db.Save(&todo).Error; err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, todo)
}
```

* `db.Save()` : 존재하면 업데이트, 없으면 생성

---

### DeleteTodo

```go
func DeleteTodo(c *gin.Context) {
    var todo Todo
    id := c.Param("id")
    if err := db.First(&todo, id).Error; err != nil {
        c.JSON(404, gin.H{"error": "Todo not found"})
        return
    }
    if err := db.Delete(&todo).Error; err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, gin.H{"message": "Todo deleted"})
}
```

---

# 4. 백엔드와 DB 연결 확인

* Postman으로 API 요청 확인
* OS 환경에서 app 컨테이너 실행 문제 발생
* init.sql 대신 gorm AutoMigrate 사용
* Docker Desktop에서 실행 후 정상 동작

---

# 5. 백엔드 Dockerfile 작성 및 Compose에 추가

멀티스테이징 시도했으나 문제 발생 → 단일 스테이지 사용

---

## docker-compose depends_on 이슈

`depends_on`은 컨테이너 시작 순서만 보장
DB ready 상태는 보장하지 않음

→ DB는 실행되었지만 MySQL 준비 전 백엔드 연결 시도
→ 실패 후 종료

---

## 해결 방법 1: wait-for 스크립트 사용

### wait-for.sh

```sh
#!/bin/sh

host="$1"
shift
cmd="$@"

until nc -z "$host" 3306; do
echo "Waiting for MySQL at $host:3306..."
sleep 2
done

exec $cmd
```

※ `nc` 사용을 위해 netcat 설치 필요

Dockerfile 추가

```dockerfile
RUN apt-get update && apt-get install -y netcat-openbsd
```

docker-compose.yml 추가

```yaml
backend:
  entrypoint: ["./wait-for.sh", "db", "/go-todo"]
```

→ docker compose up 속도 느려짐

---

## 해결 방법 2: Go 코드에서 DB 재시도 로직 구현

```go
func init() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"))

	maxRetries := 10
	for i := 1; i <= maxRetries; i++ {
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		fmt.Printf("[%d/%d] Failed to connect to DB: %v\n", i, maxRetries, err)
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		panic(fmt.Sprintf("Could not connect to DB after %d attempts: %v", maxRetries, err))
	}

	err = db.AutoMigrate(&Todo{})
	if err != nil {
		panic("AutoMigrate failed: " + err.Error())
	}
}
```

* 최대 10번 재시도
* 실패 시 3초 대기
* 10번 실패 시 panic 종료
* AutoMigrate 수행


