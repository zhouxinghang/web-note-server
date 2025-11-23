package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

type DataRecord struct {
	Value string `json:"value"`
}

type RecordResponse struct {
	Id         int64  `json:"id"`
	Value      string `json:"value"`
	CreateTime string `json:"create_time"`
}

func main() {
	// 连接 SQLite 数据库
	db, err := sql.Open("sqlite3", "./data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 修改数据表结构
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			value TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	// 处理写入请求的 handler
	http.HandleFunc("/write", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
			return
		}

		var data DataRecord
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, "无效的请求数据", http.StatusBadRequest)
			return
		}

		// 写入数据库并返回ID和创建时间
		var id int64
		var createTime string
		err := db.QueryRow(`
			INSERT INTO records (value) 
			VALUES (?) 
			RETURNING id, created_at`,
			data.Value).Scan(&id, &createTime)

		if err != nil {
			log.Println("数据库写入失败", err.Error())
			http.Error(w, "数据库写入失败", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "success",
			"message":     "数据已成功写入",
			"id":          id,
			"create_time": createTime,
		})
	})

	// 查询所有数据的接口
	http.HandleFunc("/query/all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "只支持 GET 请求", http.StatusMethodNotAllowed)
			return
		}

		rows, err := db.Query("SELECT id, value, created_at FROM records ORDER BY created_at DESC")
		if err != nil {
			http.Error(w, "查询失败", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var records []RecordResponse
		for rows.Next() {
			var record RecordResponse
			err := rows.Scan(&record.Id, &record.Value, &record.CreateTime)
			if err != nil {
				http.Error(w, "数据解析失败", http.StatusInternalServerError)
				return
			}
			records = append(records, record)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"total":   len(records),
			"records": records,
		})
	})

	// 启动服务器
	log.Println("服务器启动在 :8080 端口")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
