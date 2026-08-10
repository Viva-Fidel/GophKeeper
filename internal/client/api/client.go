// Package api — HTTP-клиент к серверу GophKeeper.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gophkeeper/internal/secret"
	"gophkeeper/internal/user"
)

// Client — клиент REST API.
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// New создаёт API-клиент.
func New(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Register регистрирует пользователя и сохраняет токен.
func (c *Client) Register(login, password, saltB64 string) (*user.AuthResponse, error) {
	var resp user.AuthResponse
	if err := c.doJSON(http.MethodPost, "/api/user/register", nil, user.RegisterRequest{
		Login: login, Password: password, Salt: saltB64,
	}, &resp); err != nil {
		return nil, err
	}
	c.Token = resp.Token
	return &resp, nil
}

// Login аутентифицирует пользователя и сохраняет токен.
func (c *Client) Login(login, password string) (*user.AuthResponse, error) {
	var resp user.AuthResponse
	if err := c.doJSON(http.MethodPost, "/api/user/login", nil, user.LoginRequest{
		Login: login, Password: password,
	}, &resp); err != nil {
		return nil, err
	}
	c.Token = resp.Token
	return &resp, nil
}

// Upsert создаёт или обновляет секрет на сервере.
func (c *Client) Upsert(req secret.UpsertRequest) (*secret.SecretDTO, error) {
	var resp secret.SecretDTO
	if err := c.doJSON(http.MethodPost, "/api/secrets", c.authHeader(), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// List возвращает список секретов.
func (c *Client) List() ([]secret.SecretDTO, error) {
	var resp []secret.SecretDTO
	if err := c.doJSON(http.MethodGet, "/api/secrets", c.authHeader(), nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Get возвращает секрет по id.
func (c *Client) Get(id string) (*secret.SecretDTO, error) {
	var resp secret.SecretDTO
	if err := c.doJSON(http.MethodGet, "/api/secrets/"+id, c.authHeader(), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Delete удаляет секрет по id.
func (c *Client) Delete(id string) (*secret.SecretDTO, error) {
	var resp secret.SecretDTO
	if err := c.doJSON(http.MethodDelete, "/api/secrets/"+id, c.authHeader(), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Sync выполняет двустороннюю синхронизацию.
func (c *Client) Sync(req secret.SyncRequest) (*secret.SyncResponse, error) {
	var resp secret.SyncResponse
	if err := c.doJSON(http.MethodPost, "/api/secrets/sync", c.authHeader(), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// authHeader возвращает заголовок Authorization с текущим JWT.
func (c *Client) authHeader() map[string]string {
	return map[string]string{"Authorization": "Bearer " + c.Token}
}

// doJSON выполняет JSON HTTP-запрос и декодирует ответ в out.
func (c *Client) doJSON(method, path string, headers map[string]string, body any, out any) error {
	var rdr io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("api error: %s: %s", resp.Status, string(data))
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}
