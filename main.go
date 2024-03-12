package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"github.com/chzyer/readline"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func executeSQLFromStdin(db *sql.DB) {
	scanner := bufio.NewScanner(os.Stdin)
	var sqlStatements string

	for scanner.Scan() {
		sqlStatements += scanner.Text()
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("读取标准输入错误:", err)
		os.Exit(1)
	}

	// 你可能需要根据实际的SQL分隔符调整这里的逻辑
	for _, statement := range strings.Split(sqlStatements, ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		_, err := db.Exec(statement)
		if err != nil {
			fmt.Println("执行SQL失败:", err)
			return
		}
	}

	fmt.Println("SQL文件执行完成")
}

// containsFilter 检查一行数据中是否至少有一个值包含了过滤字符串
// exclude 参数为 true 时，实现 grep -v 的功能
func containsFilter(values []sql.RawBytes, filterString string, exclude bool) bool {
	filterStringLower := strings.ToLower(filterString)
	if exclude {
		// 排除模式: 所有字段都必须不包含过滤字符串才返回true
		for _, value := range values {
			valueLower := strings.ToLower(string(value))
			if strings.Contains(valueLower, filterStringLower) {
				return false // 只要有一个字段包含过滤字符串，就返回false
			}
		}
		return true // 所有字段都不包含过滤字符串，返回true
	} else {
		// 包含模式: 只要有一个字段包含过滤字符串就返回true
		for _, value := range values {
			valueLower := strings.ToLower(string(value))
			if strings.Contains(valueLower, filterStringLower) {
				return true // 找到匹配的字段，返回true
			}
		}
		return false // 没有字段包含过滤字符串，返回false
	}
}

// 执行SQL语句并打印结果
func executeAndPrintSQL(db *sql.DB, sqlStatement string) {
	start := time.Now()
	rows, err := db.Query(sqlStatement)
	if err != nil {
		fmt.Printf("执行查询失败: %v\n", err)
		return
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		fmt.Printf("获取列失败: %v\n", err)
		return
	}

	fmt.Println(strings.Join(cols, "\t"))

	var rowCount int
	for rows.Next() {
		rowCount++
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			fmt.Printf("读取行失败: %v\n", err)
			return
		}

		for _, col := range columns {
			switch v := col.(type) {
			case nil:
				fmt.Print("NULL\t")
			case []byte:
				fmt.Print(string(v) + "\t")
			default:
				// 对于非[]byte类型，使用 fmt.Sprint 将其转换为字符串
				fmt.Print(fmt.Sprint(v) + "\t")
			}
		}
		fmt.Println()
	}

	fmt.Printf("%d rows in set (%.2f sec)\n", rowCount, time.Since(start).Seconds())
}

// 此函数用于格式化并打印查询结果
func printResults(rows *sql.Rows, verticalFormat bool, start time.Time, filterString string, exclude bool) error {
	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	filterString = strings.TrimSuffix(filterString, ";")

	// 对于垂直格式的输出
	if verticalFormat {
		var values = make([]interface{}, len(cols))
		for i := range values {
			values[i] = new(sql.RawBytes)
		}

		rowCount := 0
		for rows.Next() {
			rowCount++
			err = rows.Scan(values...)
			if err != nil {
				return err
			}
			fmt.Printf("*************************** %d. row ***************************\n", rowCount)
			for i, col := range cols {
				val := string(*(values[i].(*sql.RawBytes)))
				fmt.Printf("%30s: %s\n", col, val)
			}
		}
		fmt.Printf("%d row in set (%.2f sec)\n", rowCount, time.Since(start).Seconds())
	} else {
		fmt.Println(strings.Join(cols, "\t"))
		var rowCount int
		for rows.Next() {
			var values = make([]sql.RawBytes, len(cols))
			var scanArgs = make([]interface{}, len(values))
			for i := range values {
				scanArgs[i] = &values[i]
			}

			err = rows.Scan(scanArgs...)
			if err != nil {
				return err
			}

			// 检查整行数据是否包含过滤字符串
			if filterString == "" || containsFilter(values, filterString, exclude) {
				rowCount++
				var valueStrings []string
				for _, value := range values {
					if value == nil {
						valueStrings = append(valueStrings, "NULL")
					} else {
						valueStrings = append(valueStrings, string(value))
					}
				}
				fmt.Println(strings.Join(valueStrings, "\t"))
			}
		}
		fmt.Printf("%d rows in set (%.2f sec)\n", rowCount, time.Since(start).Seconds())
	}

	return rows.Err()
}

func parseArgs(args []string) (user, password, host string, port int, dbname string, executeSQL string, sqlFilePath string, timeout string) {
	host = "localhost" // 默认值
	port = 3306        // 默认值
	timeout = "10"

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			var value string
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				value = args[i+1]
				i++ // 跳过下一个参数，因为它已经被作为当前参数的值处理了
			} else {
				value = strings.TrimPrefix(arg, arg[:2])
			}

			switch {
			case strings.HasPrefix(arg, "-u"):
				user = value
			case strings.HasPrefix(arg, "-p"):
				password = value
			case strings.HasPrefix(arg, "-h"):
				host = value
			case strings.HasPrefix(arg, "-P"):
				var err error
				port, err = strconv.Atoi(value)
				if err != nil {
					fmt.Printf("无效的端口号: %s\n", value)
					os.Exit(1)
				}
			case strings.HasPrefix(arg, "-D"):
				dbname = value
			case strings.HasPrefix(arg, "-e"):
				executeSQL = value
			case strings.HasPrefix(arg, "-f"):
				sqlFilePath = value
			case strings.HasPrefix(arg, "-t"):
				timeout = value
			}
		}
	}
	return
}

func getServerVersion(db *sql.DB) string {
	var version string
	err := db.QueryRow("SELECT VERSION();").Scan(&version)
	if err != nil {
		fmt.Println("获取服务器版本失败:", err)
		return "Unknown version"
	}
	return version
}

func executeSQLFile(db *sql.DB, filePath string) error {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %v", err)
	}

	// 可能需要根据你的需求调整逻辑，例如处理多个SQL语句等
	requests := strings.Split(string(fileContent), ";")

	for _, request := range requests {
		request = strings.TrimSpace(request)
		if request == "" {
			continue // 忽略空语句
		}
		_, err := db.Exec(request)
		if err != nil {
			return fmt.Errorf("执行SQL失败: %v", err)
		}
	}

	return nil
}

// 分割SQL语句并检查是否包含SELECT或SHOW
func containsSelectOrShow(sql string) bool {
	// 以分号分割SQL语句，可能需要考虑去除注释和字符串常量中的分号
	statements := strings.Split(sql, ";")
	for _, statement := range statements {
		trimmedStatement := strings.TrimSpace(statement)
		if strings.HasPrefix(strings.ToUpper(trimmedStatement), "SELECT") || strings.HasPrefix(strings.ToUpper(trimmedStatement), "SHOW") {
			return true
		}
	}
	return false
}

func main() {
	// 通过自定义解析来获取数据库连接信息
	user, password, host, port, dbname, executeSQL, sqlFilePath, timeout := parseArgs(os.Args[1:])
	// 构建数据源名称（DSN）
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?multiStatements=true&timeout=%ss", user, password, host, port, dbname, timeout)
	// 连接MySQL数据库
	db, err := sql.Open("mysql", dsn)
	executableName := filepath.Base(os.Args[0])
	if err != nil {
		fmt.Println("连接数据库失败:", err)
		fmt.Println("使用例子:")
		fmt.Printf("%s -uroot -p123456 -h127.0.0.1 -P3306\n", executableName)
		fmt.Printf("%s -u root -p 123456 -h 127.0.0.1 -P 3306\n", executableName)
		fmt.Printf("%s -uroot -p123456 -h127.0.0.1 -P3306 -Ddbname\n", executableName)
		fmt.Printf("%s -uroot -p123456 -h127.0.0.1 -P3306 -f xxx.sql\n", executableName)
		fmt.Printf("%s -uroot -p123456 -h127.0.0.1 -P3306 -Ddbname < xxx.sql\n", executableName)
		fmt.Printf("%s -uroot -p123456 -h127.0.0.1 -P3306 -Ddbname -e 'select * from users limit 10;'\n", executableName)
		os.Exit(1)
	}
	defer db.Close()
	// 确保连接正常
	if err := db.Ping(); err != nil {
		fmt.Println("数据库连接异常:", err)
		fmt.Println("使用例子:")
		fmt.Printf("%s -uroot -p123456 -h127.0.0.1 -P3306\n", executableName)
		fmt.Printf("%s -u root -p 123456 -h 127.0.0.1 -P 3306\n", executableName)
		fmt.Printf("%s -uroot -p123456 -h127.0.0.1 -P3306 -Ddbname\n", executableName)
		fmt.Printf("%s -uroot -p123456 -h127.0.0.1 -P3306 -f xxx.sql\n", executableName)
		fmt.Printf("%s -uroot -p123456 -h127.0.0.1 -P3306 -Ddbname < xxx.sql\n", executableName)
		fmt.Printf("%s -uroot -p123456 -h127.0.0.1 -P3306 -Ddbname -e 'select * from users limit 10;'\n", executableName)
		os.Exit(1)
	}

	// 执行导入文件
	if sqlFilePath != "" {
		// 执行SQL文件
		if err := executeSQLFile(db, sqlFilePath); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("SQL文件执行完成")
		os.Exit(0)
	}

	// 检查是否有从标准输入传入的数据
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		fmt.Println("检查标准输入失败:", err)
		os.Exit(1)
	}
	if fileInfo.Mode()&os.ModeCharDevice == 0 {
		// 有数据通过标准输入传入，例如重定向或管道
		executeSQLFromStdin(db)
		os.Exit(0)
	}

	if executeSQL != "" {
		// 如果有直接执行的SQL语句，执行它并退出
		// 检测是否为SELECT或SHOW命令
		if strings.HasPrefix(strings.ToUpper(executeSQL), "SELECT") || strings.HasPrefix(strings.ToUpper(executeSQL), "SHOW") || containsSelectOrShow(executeSQL) {
			// 执行查询并打印结果
			executeAndPrintSQL(db, executeSQL)
		} else {
			// 对于非查询操作，使用Exec并输出受影响的行数
			result, err := db.Exec(executeSQL)
			if err != nil {
				fmt.Printf("执行命令失败: %v\n", err)
			} else {
				affectedRows, err := result.RowsAffected()
				if err != nil {
					fmt.Printf("获取受影响行数失败: %v\n", err)
				} else {
					fmt.Printf("Query OK, %d row(s) affected\n", affectedRows)
				}
			}
		}
		os.Exit(0)
	}
	// 获取服务器版本
	version := getServerVersion(db)
	fmt.Println("欢迎使用MySQL-go客户端. by:yy")
	fmt.Printf("Server version: %s\n", version)
	fmt.Println("退出命令:exit exit; quit quit;")
	fmt.Println("导入文件:mysql -uroot -p123 -h192.168.0.100 -P4000 -fxxxx.sql\n")
	fmt.Println("过滤命令:select * from xxx; | grep xxxxxxx")
	fmt.Println("排除过滤命令:select * from xxx; | grepv xxxxxxx #类似linux中的grep -v\n")

	rl, err := readline.New("mysql> ")
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	var multiLineQuery string
	for {
		line, err := rl.Readline()
		if err != nil { // EOF 或 Ctrl+C 会产生错误
			break
		}
		// 分离命令和可能的grep部分
		var filterString string
		var excludeFilter bool // 用于指示是否应排除包含特定关键字的行

		// 检测 grep 或 grepv 命令
		if parts := strings.Split(line, "| grep "); len(parts) > 1 {
			line = parts[0]         // 命令部分
			filterString = parts[1] // grep过滤器
			excludeFilter = false   // 不排除
		} else if parts := strings.Split(line, "| grepv "); len(parts) > 1 {
			line = parts[0]         // 命令部分
			filterString = parts[1] // grepv过滤器
			excludeFilter = true    // 排除匹配行
		}
		if parts := strings.Split(line, "|grep "); len(parts) > 1 {
			line = parts[0]         // 命令部分
			filterString = parts[1] // grep过滤器
			excludeFilter = false   // 不排除
		} else if parts := strings.Split(line, "|grepv "); len(parts) > 1 {
			line = parts[0]         // 命令部分
			filterString = parts[1] // grepv过滤器
			excludeFilter = true    // 排除匹配行
		}

		trimmedLine := strings.TrimSpace(line)
		// 先检查是否是退出命令
		if trimmedLine == "exit" || trimmedLine == "exit;" || trimmedLine == "quit" || trimmedLine == "quit;" {
			fmt.Println("退出程序")
			break
		}

		multiLineQuery += " " + line // 继续累积命令行

		// 检查是否需要执行查询：查询以分号结束或包含\G（可能跟随分号）
		executeQuery := false
		verticalFormat := false

		// 检查垂直格式标记\G
		if strings.Contains(strings.ToUpper(multiLineQuery), "\\G") {
			verticalFormat = true
			// 移除\G和\G;，准备执行
			multiLineQuery = strings.TrimSuffix(multiLineQuery, "\\G")        // 对于没有分号的情况
			multiLineQuery = strings.Replace(multiLineQuery, "\\G;", ";", -1) // 对于\G;的情况
			executeQuery = true
		}

		if strings.HasSuffix(strings.TrimSpace(multiLineQuery), ";") {
			executeQuery = true
			// 确保移除结尾的分号，如果之前没有做
			multiLineQuery = strings.TrimSuffix(multiLineQuery, ";")
		}

		if executeQuery {
			// 执行SQL查询
			multiLineQuery = strings.TrimSpace(multiLineQuery) // 清理前后空格
			start := time.Now()                                // 开始计时

			var result sql.Result
			var affectedRows int64
			var err error
			if strings.HasPrefix(strings.ToUpper(multiLineQuery), "SELECT") || strings.HasPrefix(strings.ToUpper(multiLineQuery), "SHOW") || containsSelectOrShow(multiLineQuery) {
				// 对于SELECT和SHOW语句使用Query
				rows, err := db.Query(multiLineQuery)
				if err != nil {
					fmt.Println("执行查询失败:", err)
				} else {
					defer rows.Close()
					if err := printResults(rows, verticalFormat, start, filterString, excludeFilter); err != nil {
						fmt.Println("打印结果失败:", err)
					}
				}
			} else {
				result, err = db.Exec(multiLineQuery) // 对于非SELECT语句使用Exec
				if err != nil {
					fmt.Println("执行命令失败:", err)
				} else {
					affectedRows, err = result.RowsAffected() // 获取受影响的行数
					if err != nil {
						fmt.Println("获取受影响行数失败:", err)
					} else {
						fmt.Printf("Query OK, %d rows affected (%.2f sec)\n", affectedRows, time.Since(start).Seconds())
					}
				}
			}

			// 重置multiLineQuery以便于下一个查询
			multiLineQuery = ""
		}
	}

}
