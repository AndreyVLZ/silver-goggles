package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"database/sql"

	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/order"
	"github.com/AndreyVLZ/silver-goggles/internal/gophermart/app/internal/model/user"
	"github.com/lib/pq"
)

type orderDB struct {
	orderID string
	userID  string
	number  string
	status  string
	acc     *float64
	date    time.Time
}

func (o orderDB) order() (order.Order, error) {
	var (
		info    order.Info
		orderID order.ID
		userID  user.ID
		err     error
	)

	info, err = order.ParseInfo(o.number, o.status, o.acc, o.date)
	if err != nil {
		return order.Order{}, err
	}

	orderID, err = order.ParseID(o.orderID)
	if err != nil {
		return order.Order{}, err
	}

	userID, err = user.ParseID(o.userID)
	if err != nil {
		return order.Order{}, err
	}

	return order.New(orderID, userID, info), nil
}

type errFieldConflict struct {
	data any
	name string
}

func (e errFieldConflict) Data() any { return e.data }

func (e errFieldConflict) Error() string {
	return fmt.Sprintf("нарушение уникальности [%s]", e.name)
}

type Config struct {
	ConnDB string
}

type Postgres struct {
	cfg *Config
	db  *sql.DB
}

func New(cfg Config) *Postgres       { return &Postgres{cfg: &cfg} }
func (store *Postgres) Stop() error  { return store.db.Close() }
func (store *Postgres) Ping() error  { return store.db.Ping() }
func (store *Postgres) Name() string { return "postgres" }

func (store *Postgres) Start() error {
	if store.cfg.ConnDB == "" {
		return errors.New("connDB no set")
	}

	db, err := sql.Open("postgres", store.cfg.ConnDB)
	if err != nil {
		return err
	}

	store.db = db

	if err := store.Ping(); err != nil {
		return err
	}

	return store.createTable()
}

func (store *Postgres) createTable() error {
	createTablesSQL := `
CREATE TABLE IF NOT EXISTS auser (
	user_id varchar(100) NOT NULL ,
	login varchar(50) NOT NULL,
	hash varchar(100) NOT NULL,
	PRIMARY KEY (user_id),
	UNIQUE(login)
);

CREATE TABLE IF NOT EXISTS ordr (
	o_id varchar(100) NOT NULL,
	user_id varchar(100) NOT NULL REFERENCES auser(user_id),
	num bigint NOT NULL,
	status varchar(20) NOT NULL,
	upload TIMESTAMP default NULL,
	PRIMARY KEY (o_id),
	UNIQUE(num)
);

CREATE TABLE IF NOT EXISTS o_sum (
	o_id varchar(100) NOT NULL REFERENCES ordr(o_id),
	acc double precision NOT NULL
);`

	ctx := context.Background()
	if _, err := store.db.ExecContext(ctx, createTablesSQL); err != nil {
		return err
	}

	return nil
}

// Сохранение юзера
func (store *Postgres) SaveUser(ctx context.Context, userNew user.User) (user.User, error) {
	registStmt := `
INSERT INTO auser (user_id, login, hash) 
VALUES ($1, $2, $3)
ON CONFLICT(login) 
DO UPDATE SET 
login=EXCLUDED.login
RETURNING user_id, login;`

	var userIDStr string
	var loginInDB string

	if err := store.db.QueryRowContext(ctx,
		registStmt,
		userNew.ID().String(),
		userNew.Login(),
		userNew.Hash(),
	).Scan(
		&userIDStr,
		&loginInDB,
	); err != nil {
		return user.User{}, err
	}

	if userIDStr != userNew.ID().String() {
		return user.User{}, errFieldConflict{
			data: userNew.ID(),
			name: "-userID-",
		}
	}

	return user.Parse(userIDStr, string(userNew.Login()), userNew.Hash())
}

// Возвращает юзера по логину.
func (store *Postgres) UserByLogin(ctx context.Context, userLogin user.Login) (user.User, error) {
	loginStmt := `SELECT user_id, login, hash FROM auser WHERE login = $1;`

	var (
		userIDStr    string
		userLoginStr string
		userHash     string
	)

	args := []any{userLogin}

	if err := store.db.QueryRowContext(ctx, loginStmt, args...).Scan(&userIDStr, &userLoginStr, &userHash); err != nil {
		if errors.Is(sql.ErrNoRows, err) {
			return user.User{}, nil
		}

		return user.User{}, fmt.Errorf("exec stmt [%s] args [%v]: %w", loginStmt, args, err)
	}

	usr, err := user.Parse(userIDStr, userLoginStr, userHash)
	if err != nil {
		return user.User{}, fmt.Errorf("postgres Login: user: %w", err)
	}

	return usr, nil
}

// Сохрание заказа. Возвращает ошибку errFieldConflict,
// если заказ с таким номером уже существует
func (store *Postgres) SaveOrder(ctx context.Context, ordr order.Order) error {
	saveOrderStmt := `
INSERT INTO ordr(o_id, user_id, num, status, upload)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT(num) 
DO UPDATE SET 
num=EXCLUDED.num
RETURNING o_id, user_id, num;`

	stmtSum := `
INSERT INTO o_sum(o_id, acc)
VALUES ($1, $2);`

	var (
		orderIDStr string
		userIDStr  string
		num        string
	)

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	oi := ordr.Info()
	if err := tx.QueryRowContext(
		ctx,
		saveOrderStmt,
		ordr.ID().String(),
		ordr.UserID().String(),
		oi.Number().String(),
		oi.Status().String(),
		oi.Date(),
	).Scan(
		&orderIDStr,
		&userIDStr,
		&num,
	); err != nil {
		tx.Rollback()
		return err
	}

	if userIDStr != ordr.UserID().String() {
		tx.Rollback()
		return errFieldConflict{data: ordr.UserID(), name: "-userID-"}
	}

	// проверяем что база вернула только что добавленный заказ
	if orderIDStr != ordr.ID().String() {
		tx.Rollback()
		return errFieldConflict{data: ordr.ID(), name: "-orderID-"}
	}

	if oi.Accrual() == nil {
		return tx.Commit()
	}

	if _, err := tx.ExecContext(
		ctx,
		stmtSum,
		ordr.ID().String(),
		oi.Accrual().ToFloat(),
	); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

func ordersRowsScan(rows *sql.Rows) ([]order.Order, error) {
	var orderDB orderDB

	var orders []order.Order

	for rows.Next() {
		if err := rows.Scan(
			&orderDB.orderID,
			&orderDB.userID,
			&orderDB.number,
			&orderDB.status,
			&orderDB.acc,
			&orderDB.date,
		); err != nil {
			return nil, err
		}

		ordr, err := orderDB.order()
		if err != nil {
			return nil, err
		}

		orders = append(orders, ordr)
	}

	return orders, nil
}

func prepareAsRow(ctx context.Context, query string, fnStmt func(context.Context, string) (*sql.Stmt, error), args ...any) (*sql.Row, error) {
	stmt, err := fnStmt(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("prepare stmt: %w", err)
	}

	return stmt.QueryRowContext(ctx, args...), nil
}

func (store *Postgres) execStmt(ctx context.Context, stmt string, args ...any) (*sql.Rows, error) {
	preStmt, err := store.db.Prepare(stmt)
	if err != nil {
		return nil, fmt.Errorf("prepare stmt[%s]: %w", stmt, err)
	}

	rows, err := preStmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("execute stmt [%s] args [%v]: %w", stmt, args, err)
	}

	return rows, nil
}

// Получение заказов пользователя с конкретными статусами
func (store *Postgres) OrdersByStatuses(ctx context.Context, userID user.ID, statuses []order.Status) ([]order.Order, error) {
	stmt := `
SELECT o_id, user_id, num, status, s.acc, upload
FROM ordr
LEFT OUTER JOIN o_sum s USING (o_id)
WHERE user_id=$1
AND status = ANY($2)
ORDER BY upload
;`

	pqArr := make([]string, len(statuses))
	for i := range statuses {
		pqArr[i] = statuses[i].String()
	}

	rows, err := store.execStmt(ctx, stmt,
		userID.String(),
		pq.Array(pqArr),
	)
	if err != nil {
		return nil, fmt.Errorf("ordersByStatuses: %w", err)
	}

	return ordersRowsScan(rows)
}

// Обновление нескольких заказов
func (store *Postgres) OrdersUpdate(ctx context.Context, orders []order.Order) error {
	stmt1 := `
UPDATE ordr
SET status=$2, upload=$3
WHERE o_id = $1;`

	stmt2 := `
UPDATE o_sum
SET acc=$2
WHERE o_id = $1;`

	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	ps1, err := tx.PrepareContext(ctx, stmt1)
	if err != nil {
		tx.Rollback()
		return err
	}

	ps2, err := tx.PrepareContext(ctx, stmt2)
	if err != nil {
		tx.Rollback()
		return err
	}

	for i := range orders {
		oi := orders[i].Info()
		ordrID := orders[i].ID().String()
		if _, err := ps1.ExecContext(
			ctx,
			ordrID,
			oi.Status().String(),
			oi.Date(),
		); err != nil {
			tx.Rollback()
			return err
		}

		if oi.Accrual() == nil {
			continue
		}

		if _, err := ps2.ExecContext(
			ctx,
			ordrID,
			oi.Accrual().ToFloat(),
		); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// Получение заказов по статусу
func (store *Postgres) OrdersBatch(ctx context.Context, statuses []order.Status) ([]order.Order, error) {
	stmt2 := `
SELECT o.o_id, o.user_id, o.num, o.status, s.acc, o.upload
FROM ordr o
LEFT OUTER JOIN o_sum s USING (o_id)
WHERE status = ANY($1);`

	var orderDB orderDB

	preStmt, err := store.db.Prepare(stmt2)
	if err != nil {
		return nil, fmt.Errorf("prepare stmt: %w", err)
	}

	pqArr := make([]string, len(statuses))
	for i := range statuses {
		pqArr[i] = statuses[i].String()
	}

	rows, err := preStmt.QueryContext(ctx, pq.Array(pqArr))
	if err != nil || rows.Err() != nil {
		return nil, fmt.Errorf("execute stmt: %w", err)
	}

	var orders []order.Order

	for rows.Next() {
		if err := rows.Scan(
			&orderDB.orderID,
			&orderDB.userID,
			&orderDB.number,
			&orderDB.status,
			&orderDB.acc,
			&orderDB.date,
		); err != nil {
			return nil, fmt.Errorf("row scan: %w", err)
		}

		ordr, err := orderDB.order()
		if err != nil {
			return nil, fmt.Errorf("build order: %w", err)
		}

		orders = append(orders, ordr)
	}

	return orders, nil
}
