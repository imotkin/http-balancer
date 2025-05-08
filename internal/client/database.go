package client

import (
	"context"
	"database/sql"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"

	"github.com/google/uuid"
)

// Проверка структуры на соответствие интерфейса Storage
var _ Storage = (*DatabaseStorage)(nil)

// Структура для работы с локальной (SQLite)
// и удалённой базой данных клиентов (PostgreSQL)
type DatabaseStorage struct {
	conn *sql.DB

	defaults DefaultParams
}

func NewStorage(driver, path string, defaultCapacity, defaultRate uint) (*DatabaseStorage, error) {
	conn, err := sql.Open(driver, path)
	if err != nil {
		return nil, err
	}

	return &DatabaseStorage{
		conn: conn,
		defaults: DefaultParams{
			Capacity: defaultCapacity,
			Rate:     defaultRate,
		},
	}, nil
}

func (s *DatabaseStorage) Defaults() DefaultParams {
	return s.defaults
}

func (s *DatabaseStorage) Connection() *sql.DB {
	return s.conn
}

// Получение списка клиентов в базе данных
func (s *DatabaseStorage) List(ctx context.Context) ([]Client, error) {
	rows, err := s.conn.QueryContext(ctx,
		`SELECT api_key, name, capacity, rate 
		   FROM clients 
		  ORDER BY api_key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []Client

	for rows.Next() {
		var c Client

		err = rows.Scan(&c.Key, &c.Name, &c.Capacity, &c.Rate)
		if err != nil {
			return clients, err
		}

		clients = append(clients, c)
	}

	if err = rows.Err(); err != nil {
		return clients, err
	}

	return clients, nil
}

// Добавление нового клиента в базу данных
func (s *DatabaseStorage) Add(ctx context.Context, client Client) (string, error) {
	key := uuid.NewString()

	_, err := s.conn.ExecContext(ctx, `
		INSERT INTO clients
		VALUES ($1, $2, $3, $4)`, key, client.Name, client.Capacity, client.Rate)
	if err != nil {
		return "", err
	}

	return key, nil
}

// Проверка клиента в базе данных и добавление, если его нет
func (s *DatabaseStorage) Has(ctx context.Context, key string) (*Client, error) {
	var c Client

	err := s.conn.QueryRowContext(ctx, `
		INSERT INTO clients
		VALUES ($1, $2, $3, $4)
		    ON CONFLICT (api_key)
			DO UPDATE 
		   SET api_key = EXCLUDED.api_key
	 RETURNING api_key, name, capacity, rate`,
		key, uuid.NewString(), s.Defaults().Capacity, s.Defaults().Rate).
		Scan(&c.Key, &c.Name, &c.Capacity, &c.Rate)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// Получение клиента из базы данных по заданному ключу
func (s *DatabaseStorage) Get(ctx context.Context, key string) (*Client, error) {
	var c Client

	err := s.conn.QueryRowContext(ctx, `
		SELECT api_key, name, capacity, rate  
		  FROM clients 
		 WHERE api_key = $1`, key).
		Scan(&c.Key, &c.Name, &c.Capacity, &c.Rate)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

// Удаление клиента из базы данных по заданному ключу
func (s *DatabaseStorage) Delete(ctx context.Context, key string) error {
	res, err := s.conn.ExecContext(ctx, "DELETE FROM clients WHERE api_key = $1", key)
	if err != nil {
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return sql.ErrNoRows
	}

	return err
}
