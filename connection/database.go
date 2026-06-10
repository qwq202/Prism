package connection

import (
	"chat/globals"
	"chat/utils"
	"crypto/tls"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

var DB *sql.DB

const defaultRootUsername = "root"

func getInitialRootPassword() (string, bool) {
	password := strings.TrimSpace(viper.GetString("root.initial_password"))
	if password == "" {
		return utils.GenerateChar(24), false
	}

	if len(password) < 6 || len(password) > 36 {
		globals.Warn("[service] root.initial_password must be 6-36 characters; generated a random root password instead")
		return utils.GenerateChar(24), false
	}

	return password, true
}

func InitMySQLSafe() *sql.DB {
	ConnectDatabase()

	// using DB as a global variable to point to the latest db connection
	MysqlWorker(DB)
	return DB
}

func getConn() *sql.DB {
	if viper.GetString("mysql.host") == "" {
		globals.SqliteEngine = true
		globals.Warn("[connection] mysql host is not set, using sqlite (~/db/chatnio.db)")
		db, err := sql.Open("sqlite3", utils.FileSafe("./db/chatnio.db"))
		if err != nil {
			panic(err)
		}

		return db
	}

	mysqlUrl := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s",
		viper.GetString("mysql.user"),
		viper.GetString("mysql.password"),
		viper.GetString("mysql.host"),
		viper.GetInt("mysql.port"),
		utils.GetStringConfs("mysql.database", "mysql.db"),
	)
	if viper.GetBool("mysql.tls") {
		mysql.RegisterTLSConfig("tls", &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: viper.GetString("mysql.host"),
		})

		mysqlUrl += "?tls=tls"
	}

	for {
		db, err := sql.Open("mysql", mysqlUrl)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			globals.Debug(fmt.Sprintf("[connection] connected to mysql server (host: %s)", viper.GetString("mysql.host")))
			return db
		}

		globals.Warn(
			fmt.Sprintf("[connection] failed to connect to mysql server: %s (message: %s), will retry in 5 seconds",
				viper.GetString("mysql.host"), utils.GetError(err),
			),
		)

		if db != nil {
			_ = db.Close()
		}
		utils.Sleep(5000)
	}
}

func ConnectDatabase() *sql.DB {
	db := getConn()

	db.SetMaxOpenConns(64)
	db.SetMaxIdleConns(16)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	CreateUserTable(db)
	CreateConversationTable(db)
	CreateMaskTable(db)
	CreateSharingTable(db)
	CreatePackageTable(db)
	CreateQuotaTable(db)
	CreateSubscriptionTable(db)
	CreatePasskeyCredentialTable(db)
	CreateInvitationTable(db)
	CreateRedeemTable(db)
	CreateBroadcastTable(db)
	CreateBillingTable(db)
	CreateModelUsageMetricsTable(db)
	CreatePaymentOrdersTable(db)
	CreateMemoryTable(db)

	migrateDatabaseOrPanic(db)

	DB = db

	return db
}

func panicDatabaseSetup(step string, err error) {
	if err == nil {
		return
	}

	message := fmt.Sprintf("[connection] %s failed: %s", step, err.Error())
	globals.Error(message)
	panic(message)
}

func migrateDatabaseOrPanic(db *sql.DB) {
	if err := doMigration(db); err != nil {
		_ = db.Close()
		panicDatabaseSetup("database migration", err)
	}
}

func mustCreateTable(db *sql.DB, name string, query string) {
	_, err := globals.ExecDb(db, query)
	panicDatabaseSetup(fmt.Sprintf("create %s table", name), err)
}

func InitRootUser(db *sql.DB) {
	// create root user if totally empty
	var count int
	err := globals.QueryRowDb(db, "SELECT COUNT(*) FROM auth").Scan(&count)
	if err != nil {
		panicDatabaseSetup("query root user count", err)
	}

	if count == 0 {
		password, configured := getInitialRootPassword()
		if configured {
			globals.Warn("[service] no user found, creating root user with configured root.initial_password; change it after first login")
		} else {
			globals.Warn(fmt.Sprintf("[service] no user found, creating root user with generated password (username: %s, password: %s); save it now or reset it with `prism root <new-password>`", defaultRootUsername, password))
		}

		hash, err := utils.HashPassword(password)
		if err != nil {
			panicDatabaseSetup("hash initial root password", err)
		}
		_, err = globals.ExecDb(db, `
			INSERT INTO auth (username, password, email, is_admin, bind_id, token)
			VALUES (?, ?, ?, ?, ?, ?)
		`, defaultRootUsername, hash, "root@example.com", true, 0, defaultRootUsername)
		if err != nil {
			panicDatabaseSetup("create initial root user", err)
		}
	} else {
		globals.Debug(fmt.Sprintf("[service] %d user(s) found, skip creating root user", count))
	}
}

func CreateUserTable(db *sql.DB) {
	mustCreateTable(db, "auth", `
		CREATE TABLE IF NOT EXISTS auth (
		  id INT PRIMARY KEY AUTO_INCREMENT,
		  bind_id INT UNIQUE,
		  username VARCHAR(24) UNIQUE,
		  token VARCHAR(255) NOT NULL,
		  email VARCHAR(255) UNIQUE,
		  password VARCHAR(64) NOT NULL,
		  is_admin BOOLEAN DEFAULT FALSE,
		  is_banned BOOLEAN DEFAULT FALSE,
		  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)

	InitRootUser(db)
}

func CreatePackageTable(db *sql.DB) {
	mustCreateTable(db, "package", `
		CREATE TABLE IF NOT EXISTS package (
		  id INT PRIMARY KEY AUTO_INCREMENT,
		  user_id INT,
		  type VARCHAR(255),
		  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  FOREIGN KEY (user_id) REFERENCES auth(id),
		  UNIQUE KEY (user_id, type)
		);
	`)
}

func CreateQuotaTable(db *sql.DB) {
	mustCreateTable(db, "quota", `
		CREATE TABLE IF NOT EXISTS quota (
		  id INT PRIMARY KEY AUTO_INCREMENT,
		  user_id INT UNIQUE,
		  quota DECIMAL(24, 6),
		  used DECIMAL(24, 6),
		  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  FOREIGN KEY (user_id) REFERENCES auth(id)
		);
	`)
}

func CreateConversationTable(db *sql.DB) {
	mustCreateTable(db, "conversation", `
		CREATE TABLE IF NOT EXISTS conversation (
		  id INT PRIMARY KEY AUTO_INCREMENT,
		  user_id INT,
		  conversation_id INT,
		  conversation_name VARCHAR(255),
		  data MEDIUMTEXT,
		  model VARCHAR(255) NOT NULL DEFAULT 'gpt-3.5-turbo-0613',
		  task_id VARCHAR(255) NULL,
		  favorite BOOLEAN DEFAULT FALSE,
		  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  UNIQUE KEY (user_id, conversation_id)
		);
	`)
}

func CreateMemoryTable(db *sql.DB) {
	mustCreateTable(db, "memories", `
		CREATE TABLE IF NOT EXISTS memories (
		  id INT PRIMARY KEY AUTO_INCREMENT,
		  user_id INT NOT NULL,
		  scope_type VARCHAR(32) NOT NULL DEFAULT 'user',
		  scope_id VARCHAR(128) NOT NULL,
		  content TEXT NOT NULL,
		  source VARCHAR(32) NULL,
		  confidence FLOAT NULL,
		  pinned BOOLEAN NOT NULL DEFAULT FALSE,
		  category VARCHAR(64) NULL,
		  last_used_at DATETIME NULL,
		  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
		  FOREIGN KEY (user_id) REFERENCES auth(id)
		);
	`)
}

func CreateMaskTable(db *sql.DB) {
	mustCreateTable(db, "mask", `
		CREATE TABLE IF NOT EXISTS mask (
		  id INT PRIMARY KEY AUTO_INCREMENT,
		  user_id INT,
		  avatar VARCHAR(255),
		  name VARCHAR(255),
		  description TEXT,
		  context TEXT,
		  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  FOREIGN KEY (user_id) REFERENCES auth(id)
		);
	`)
}

func CreateSharingTable(db *sql.DB) {
	// refs is an array of message id, separated by comma (-1 means all messages)
	mustCreateTable(db, "sharing", `
		CREATE TABLE IF NOT EXISTS sharing (
		  id INT PRIMARY KEY AUTO_INCREMENT,
		  hash CHAR(32) UNIQUE,
		  user_id INT,
		  conversation_id INT,
		  refs TEXT,
		  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  FOREIGN KEY (user_id) REFERENCES auth(id)
		);
	`)
}

func CreateSubscriptionTable(db *sql.DB) {
	mustCreateTable(db, "subscription", `
		CREATE TABLE IF NOT EXISTS subscription (
		  id INT PRIMARY KEY AUTO_INCREMENT,
		  level INT DEFAULT 1,
		  user_id INT UNIQUE,
		  expired_at DATETIME,
		  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  total_month INT DEFAULT 0,
		  enterprise BOOLEAN DEFAULT FALSE,
		  FOREIGN KEY (user_id) REFERENCES auth(id)
		);
	`)
}

func CreatePasskeyCredentialTable(db *sql.DB) {
	mustCreateTable(db, "passkey_credential", `
		CREATE TABLE IF NOT EXISTS passkey_credential (
		  id INT PRIMARY KEY AUTO_INCREMENT,
		  user_id INT NOT NULL,
		  credential_id VARCHAR(512) NOT NULL UNIQUE,
		  name VARCHAR(255),
		  transports VARCHAR(255),
		  attestation_object TEXT,
		  client_data_json TEXT,
		  public_key TEXT,
		  sign_count INT DEFAULT 0,
		  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  FOREIGN KEY (user_id) REFERENCES auth(id)
		);
	`)
}

func CreateInvitationTable(db *sql.DB) {
	mustCreateTable(db, "invitation", `
		CREATE TABLE IF NOT EXISTS invitation (
		  id INT PRIMARY KEY AUTO_INCREMENT,
		  code VARCHAR(255) UNIQUE,
		  quota DECIMAL(16, 4),
		  type VARCHAR(255),
		  used BOOLEAN DEFAULT FALSE,
		  used_id INT,
		  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  UNIQUE KEY (used_id, type),
		  FOREIGN KEY (used_id) REFERENCES auth(id)
		);
	`)
}

func CreateRedeemTable(db *sql.DB) {
	mustCreateTable(db, "redeem", `
		CREATE TABLE IF NOT EXISTS redeem (
		  id INT PRIMARY KEY AUTO_INCREMENT,
		  code VARCHAR(255) UNIQUE,
		  quota DECIMAL(16, 4),
		  used BOOLEAN DEFAULT FALSE,
		  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
}

func CreateBroadcastTable(db *sql.DB) {
	mustCreateTable(db, "broadcast", `
		CREATE TABLE IF NOT EXISTS broadcast (
		  id INT PRIMARY KEY AUTO_INCREMENT,
		  poster_id INT,
		  content TEXT,
		  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  FOREIGN KEY (poster_id) REFERENCES auth(id)
		);
	`)
}

func CreateBillingTable(db *sql.DB) {
	mustCreateTable(db, "billing", `
		CREATE TABLE IF NOT EXISTS billing (
		  id INT PRIMARY KEY AUTO_INCREMENT,
		  user_id INT,
		  username VARCHAR(255),
		  type VARCHAR(50) DEFAULT 'consume',
		  token_name VARCHAR(255),
		  model VARCHAR(255),
		  input_tokens INT DEFAULT 0,
		  output_tokens INT DEFAULT 0,
		  quota DECIMAL(16, 6) DEFAULT 0,
		  duration FLOAT DEFAULT 0,
		  detail TEXT,
		  prompts MEDIUMTEXT,
		  response_prompts MEDIUMTEXT,
		  channel INT DEFAULT 0,
		  channel_name VARCHAR(255),
		  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  FOREIGN KEY (user_id) REFERENCES auth(id)
		);
	`)
}

func CreateModelUsageMetricsTable(db *sql.DB) {
	mustCreateTable(db, "model_usage_metrics", `
		CREATE TABLE IF NOT EXISTS model_usage_metrics (
		  id INT PRIMARY KEY AUTO_INCREMENT,
		  model VARCHAR(255) NOT NULL,
		  success BOOLEAN DEFAULT FALSE,
		  error_type VARCHAR(50),
		  input_tokens INT DEFAULT 0,
		  output_tokens INT DEFAULT 0,
		  duration FLOAT DEFAULT 0,
		  error TEXT,
		  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
}

func CreatePaymentOrdersTable(db *sql.DB) {
	mustCreateTable(db, "payment_orders", `
		CREATE TABLE IF NOT EXISTS payment_orders (
		  id INT PRIMARY KEY AUTO_INCREMENT,
		  user_id INT,
		  username VARCHAR(255),
		  type VARCHAR(100),
		  service VARCHAR(255),
		  amount DECIMAL(10, 2) DEFAULT 0,
		  order_id VARCHAR(255) UNIQUE,
		  name VARCHAR(255),
		  device VARCHAR(50),
		  state BOOLEAN DEFAULT FALSE,
		  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		  FOREIGN KEY (user_id) REFERENCES auth(id)
		);
	`)
}
